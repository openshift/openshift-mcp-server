package mustgather

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/yaml"
)

const (
	defaultGatherSourceDir = "/must-gather/"
	defaultMustGatherImage = "registry.redhat.io/openshift4/ose-must-gather:latest"
	defaultGatherCmd       = "/usr/bin/gather"
	mgAnnotation           = "operators.openshift.io/must-gather-image"
	maxConcurrentGathers   = 8
)

// PlanMustGatherParams contains the parameters for planning a must-gather collection.
type PlanMustGatherParams struct {
	NodeName      string
	NodeSelector  map[string]string
	HostNetwork   bool
	SourceDir     string // custom gather directory inside pod, default is "/must-gather"
	Namespace     string
	KeepResources bool
	GatherCommand string   // custom gather command, default is "/usr/bin/gather"
	AllImages     bool     // whether to use custom gather images from installed operators on cluster
	Images        []string // custom list of must-gather images
	Timeout       string
	Since         string
}

// PlanMustGather generates a must-gather plan with YAML manifests for creating the required resources.
// It returns the plan as a string containing YAML manifests and instructions.
func PlanMustGather(ctx context.Context, k api.KubernetesClient, params PlanMustGatherParams) (string, error) {
	dynamicClient := k.DynamicClient()
	k8sCore := kubernetes.NewCore(k)

	sourceDir := params.SourceDir
	if sourceDir == "" {
		sourceDir = defaultGatherSourceDir
	} else {
		sourceDir = path.Clean(sourceDir)
	}

	namespace := params.Namespace
	if namespace == "" {
		namespace = fmt.Sprintf("openshift-must-gather-%s", rand.String(6))
	}

	gatherCmd := params.GatherCommand
	if gatherCmd == "" {
		gatherCmd = defaultGatherCmd
	}

	images := params.Images
	if params.AllImages {
		componentImages, err := getComponentImages(ctx, dynamicClient)
		if err != nil {
			return "", fmt.Errorf("failed to get operator images: %v", err)
		}
		images = append(images, componentImages...)
	}

	if len(images) > maxConcurrentGathers {
		return "", fmt.Errorf("more than %d gather images are not supported", maxConcurrentGathers)
	}

	timeout := params.Timeout
	if timeout != "" {
		_, err := time.ParseDuration(timeout)
		if err != nil {
			return "", fmt.Errorf("timeout duration is not valid")
		}
		gatherCmd = fmt.Sprintf("/usr/bin/timeout %s %s", timeout, gatherCmd)
	}

	since := params.Since
	if since != "" {
		_, err := time.ParseDuration(since)
		if err != nil {
			return "", fmt.Errorf("since duration is not valid")
		}
	}

	envVars := []corev1.EnvVar{}
	if since != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "MUST_GATHER_SINCE",
			Value: since,
		})
	}

	// template container for gather,
	// if multiple images are added multiple containers in the same pod will be spin up
	gatherContainerTemplate := corev1.Container{
		Name:            "gather",
		Image:           defaultMustGatherImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{gatherCmd},
		Env:             envVars,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "must-gather-output",
				MountPath: sourceDir,
			},
		},
	}

	var gatherContainers = []corev1.Container{
		*gatherContainerTemplate.DeepCopy(),
	}

	if len(images) > 0 {
		gatherContainers = make([]corev1.Container, len(images))
	}

	for i, image := range images {
		gatherContainers[i] = *gatherContainerTemplate.DeepCopy()

		// if more than one gather container(s) are added,
		// suffix container name with int id
		if len(images) > 1 {
			gatherContainers[i].Name = fmt.Sprintf("gather-%d", i+1)
		}
		gatherContainers[i].Image = image
	}

	serviceAccountName := "must-gather-collector"

	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			// Avoiding generateName as resources_create_or_update fails without explicit name.
			Name:      fmt.Sprintf("must-gather-%s", rand.String(6)),
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: serviceAccountName,
			NodeName:           params.NodeName,
			PriorityClassName:  "system-cluster-critical",
			RestartPolicy:      corev1.RestartPolicyNever,
			Volumes: []corev1.Volume{
				{
					Name: "must-gather-output",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
			Containers: append(gatherContainers, corev1.Container{
				Name:            "wait",
				Image:           "registry.redhat.io/ubi9/ubi-minimal",
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/bin/bash", "-c", "sleep infinity"},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "must-gather-output",
						MountPath: "/must-gather",
					},
				},
			}),
			HostNetwork:  params.HostNetwork,
			NodeSelector: params.NodeSelector,
			Tolerations: []corev1.Toleration{
				{
					Operator: "Exists",
				},
			},
		},
	}

	namespaceExists := false
	_, err := k8sCore.ResourcesGet(ctx, &schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Namespace",
	}, "", namespace)
	if err == nil {
		namespaceExists = true
	}

	var namespaceObj *corev1.Namespace
	if !namespaceExists {
		namespaceObj = &corev1.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
	}

	serviceAccount := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: namespace,
		},
	}

	clusterRoleBindingName := fmt.Sprintf("%s-must-gather-collector", namespace)
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleBindingName,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccountName,
				Namespace: namespace,
			},
		},
	}

	allowChecks := map[string]struct {
		schema.GroupVersionResource
		name string
		verb string
	}{
		"create_namespace": {
			GroupVersionResource: schema.GroupVersionResource{Version: "v1", Resource: "namespaces"},
			verb:                 "create",
		},
		"create_serviceaccount": {
			GroupVersionResource: schema.GroupVersionResource{Version: "v1", Resource: "serviceaccounts"},
			verb:                 "create",
		},
		"create_clusterrolebinding": {
			GroupVersionResource: schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings"},
			verb:                 "create",
		},
		"create_pod": {
			GroupVersionResource: schema.GroupVersionResource{Version: "v1", Resource: "pods"},
			verb:                 "create",
		},
		"use_scc_hostnetwork": {
			GroupVersionResource: schema.GroupVersionResource{Group: "security.openshift.io", Version: "v1", Resource: "securitycontextconstraints"},
			name:                 "hostnetwork-v2",
			verb:                 "use",
		},
	}
	isAllowed := make(map[string]bool)

	for key, check := range allowChecks {
		isAllowed[key] = k8sCore.CanIUse(ctx, &check.GroupVersionResource, "", check.verb)
	}

	var result strings.Builder
	result.WriteString("The generated plan contains YAML manifests for must-gather pods and required resources (namespace, serviceaccount, clusterrolebinding). " +
		"Suggest how the user can apply the manifest and copy results locally (`oc cp` / `kubectl cp`). \n\n",
	)
	result.WriteString("Ask the user if they want to apply the plan \n" +
		"- use the resource_create_or_update tool to apply the manifest \n" +
		"- alternatively, advise the user to execute `oc apply` / `kubectl apply` instead. \n\n",
	)

	if !params.KeepResources {
		result.WriteString("Once the must-gather collection is completed, the user may wish to cleanup the created resources. \n" +
			"- use the resources_delete tool to delete the namespace and the clusterrolebinding \n" +
			"- or, execute cleanup using `kubectl delete`. \n\n")
	}

	if !namespaceExists && isAllowed["create_namespace"] {
		namespaceYaml, err := yaml.Marshal(namespaceObj)
		if err != nil {
			return "", fmt.Errorf("failed to marshal namespace to yaml: %w", err)
		}

		result.WriteString("```yaml\n")
		result.Write(namespaceYaml)
		result.WriteString("```\n\n")
	}

	if !namespaceExists && !isAllowed["create_namespace"] {
		result.WriteString("WARNING: The resources_create_or_update call does not have permission to create namespace(s).\n")
	}

	// yaml(s) are dumped into individual code blocks of ``` ```
	// because resources_create_or_update tool call fails when content has more than one more resource,
	// some models are smart to detect an error and retry with one resource a time though.

	serviceAccountYaml, err := yaml.Marshal(serviceAccount)
	if err != nil {
		return "", fmt.Errorf("failed to marshal service account to yaml: %w", err)
	}
	result.WriteString("```yaml\n")
	result.Write(serviceAccountYaml)
	result.WriteString("```\n\n")

	if !isAllowed["create_serviceaccount"] {
		result.WriteString("WARNING: The resources_create_or_update call does not have permission to create serviceaccount(s).\n")
	}

	clusterRoleBindingYaml, err := yaml.Marshal(clusterRoleBinding)
	if err != nil {
		return "", fmt.Errorf("failed to marshal cluster role binding to yaml: %w", err)
	}

	result.WriteString("```yaml\n")
	result.Write(clusterRoleBindingYaml)
	result.WriteString("```\n\n")

	if !isAllowed["create_clusterrolebinding"] {
		result.WriteString("WARNING: The resources_create_or_update call does not have permission to create clusterrolebinding(s).\n")
	}

	podYaml, err := yaml.Marshal(pod)
	if err != nil {
		return "", fmt.Errorf("failed to marshal pod to yaml: %w", err)
	}

	result.WriteString("```yaml\n")
	result.Write(podYaml)
	result.WriteString("```\n")

	if !isAllowed["create_pod"] {
		result.WriteString("WARNING: The resources_create_or_update call does not have permission to create pod(s).\n")
	}

	if params.HostNetwork && !isAllowed["use_scc_hostnetwork"] {
		result.WriteString("WARNING: The resources_create_or_update call does not have permission to create pod(s) with hostNetwork: true.\n")
	}

	return result.String(), nil
}

func getComponentImages(ctx context.Context, dynamicClient dynamic.Interface) ([]string, error) {
	var images []string

	appendImageFromAnnotation := func(obj runtime.Object) error {
		unstruct, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return err
		}

		u := unstructured.Unstructured{Object: unstruct}
		annotations := u.GetAnnotations()
		if annotations[mgAnnotation] != "" {
			images = append(images, annotations[mgAnnotation])
		}

		return nil
	}

	// List ClusterOperators
	clusterOperatorGVR := schema.GroupVersionResource{
		Group:    "config.openshift.io",
		Version:  "v1",
		Resource: "clusteroperators",
	}
	clusterOperatorsList, err := dynamicClient.Resource(clusterOperatorGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if err := clusterOperatorsList.EachListItem(appendImageFromAnnotation); err != nil {
		return images, err
	}

	// List ClusterServiceVersions
	csvGVR := schema.GroupVersionResource{
		Group:    "operators.coreos.com",
		Version:  "v1alpha1",
		Resource: "clusterserviceversions",
	}
	csvList, err := dynamicClient.Resource(csvGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return images, err
	}

	err = csvList.EachListItem(appendImageFromAnnotation)
	return images, err
}

// ParseNodeSelector parses a comma-separated key=value selector string into a map.
func ParseNodeSelector(selector string) map[string]string {
	result := make(map[string]string)
	pairs := strings.Split(selector, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(kv) == 2 && strings.TrimSpace(kv[0]) != "" {
			result[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return result
}
