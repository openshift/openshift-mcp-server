package mustgather

import (
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
	"sigs.k8s.io/yaml"
)

const (
	defaultGatherSourceDir = "/must-gather/"
	defaultMustGatherImage = "registry.redhat.io/openshift4/ose-must-gather:latest"
	defaultGatherCmd       = "/usr/bin/gather"
	mgAnnotation           = "operators.openshift.io/must-gather-image"
	maxConcurrentGathers   = 8
)

func PlanMustGather(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	k8sCore := kubernetes.NewCore(params)

	var nodeName, sourceDir, namespace, gatherCmd, timeout, since string
	var hostNetwork, keepResources, allImages bool
	var images []string
	var nodeSelector map[string]string

	if args["node_name"] != nil {
		nodeName = args["node_name"].(string)
	}

	if args["node_selector"] != nil {
		nodeSelector = parseNodeSelector(args["node_selector"].(string))
	}

	if args["host_network"] != nil {
		hostNetwork = args["host_network"].(bool)
	}

	sourceDir = defaultGatherSourceDir
	if args["source_dir"] != nil {
		sourceDir = path.Clean(args["source_dir"].(string))
	}

	namespace = fmt.Sprintf("openshift-must-gather-%s", rand.String(6))
	if args["namespace"] != nil {
		namespace = args["namespace"].(string)
	}

	if args["keep_resources"] != nil {
		keepResources = args["keep_resources"].(bool)
	}

	gatherCmd = defaultGatherCmd
	if args["gather_command"] != nil {
		gatherCmd = args["gather_command"].(string)
	}

	if args["all_component_images"] != nil {
		allImages = args["all_component_images"].(bool)
	}

	if args["images"] != nil {
		if imagesArg, ok := args["images"].([]interface{}); ok {
			for _, img := range imagesArg {
				if imgStr, ok := img.(string); ok {
					images = append(images, imgStr)
				}
			}
		}
	}

	if allImages {
		componentImages, err := getComponentImages(params)
		if err != nil {
			return api.NewToolCallResult("",
				fmt.Errorf("failed to get operator images: %v", err),
			), nil
		}

		images = append(images, componentImages...)
	}

	if len(images) > maxConcurrentGathers {
		return api.NewToolCallResult("",
			fmt.Errorf("more than %d gather images are not supported", maxConcurrentGathers),
		), nil
	}

	if args["timeout"] != nil {
		timeout = args["timeout"].(string)

		_, err := time.ParseDuration(timeout)
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("timeout duration is not valid")), nil
		}

		gatherCmd = fmt.Sprintf("/usr/bin/timeout %s %s", timeout, gatherCmd)
	}

	if args["since"] != nil {
		since = args["since"].(string)

		_, err := time.ParseDuration(since)
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("since duration is not valid")), nil
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
			NodeName:           nodeName,
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
			HostNetwork:  hostNetwork,
			NodeSelector: nodeSelector,
			Tolerations: []corev1.Toleration{
				{
					Operator: "Exists",
				},
			},
		},
	}

	namespaceExists := false

	_, err := k8sCore.ResourcesGet(params, &schema.GroupVersionKind{
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

	for k, check := range allowChecks {
		isAllowed[k] = k8sCore.CanIUse(params, &check.GroupVersionResource, "", check.verb)
	}

	var result strings.Builder
	result.WriteString("The generated plan contains YAML manifests for must-gather pods and required resources (namespace, serviceaccount, clusterrolebinding). " +
		"Suggest how the user can apply the manifest and copy results locally (`oc cp` / `kubectl cp`). \n\n",
	)
	result.WriteString("Ask the user if they want to apply the plan \n" +
		"- use the resource_create_or_update tool to apply the manifest \n" +
		"- alternatively, advise the user to execute `oc apply` / `kubectl apply` instead. \n\n",
	)

	if !keepResources {
		result.WriteString("Once the must-gather collection is completed, the user may wish to cleanup the created resources. \n" +
			"- use the resources_delete tool to delete the namespace and the clusterrolebinding \n" +
			"- or, execute cleanup using `kubectl delete`. \n\n")
	}

	if !namespaceExists && isAllowed["create_namespace"] {
		namespaceYaml, err := yaml.Marshal(namespaceObj)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal namespace to yaml: %w", err)
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
		return nil, fmt.Errorf("failed to marshal service account to yaml: %w", err)
	}
	result.WriteString("```yaml\n")
	result.Write(serviceAccountYaml)
	result.WriteString("```\n\n")

	if !isAllowed["create_serviceaccount"] {
		result.WriteString("WARNING: The resources_create_or_update call does not have permission to create serviceaccount(s).\n")
	}

	clusterRoleBindingYaml, err := yaml.Marshal(clusterRoleBinding)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cluster role binding to yaml: %w", err)
	}

	result.WriteString("```yaml\n")
	result.Write(clusterRoleBindingYaml)
	result.WriteString("```\n\n")

	if !isAllowed["create_clusterrolebinding"] {
		result.WriteString("WARNING: The resources_create_or_update call does not have permission to create clusterrolebinding(s).\n")
	}

	podYaml, err := yaml.Marshal(pod)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pod to yaml: %w", err)
	}

	result.WriteString("```yaml\n")
	result.Write(podYaml)
	result.WriteString("```\n")

	if !isAllowed["create_pod"] {
		result.WriteString("WARNING: The resources_create_or_update call does not have permission to create pod(s).\n")
	}

	if hostNetwork && !isAllowed["use_scc_hostnetwork"] {
		result.WriteString("WARNING: The resources_create_or_update call does not have permission to create pod(s) with hostNetwork: true.\n")
	}

	return api.NewToolCallResult(result.String(), nil), nil
}

func getComponentImages(params api.ToolHandlerParams) ([]string, error) {
	var images []string
	k8sCore := kubernetes.NewCore(params)

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

	clusterOperatorsList, err := k8sCore.ResourcesList(params, &schema.GroupVersionKind{
		Group:   "config.openshift.io",
		Version: "v1",
		Kind:    "ClusterOperator",
	}, "", api.ListOptions{})
	if err != nil {
		return nil, err
	}

	if err := clusterOperatorsList.EachListItem(appendImageFromAnnotation); err != nil {
		return images, err
	}

	csvList, err := k8sCore.ResourcesList(params, &schema.GroupVersionKind{
		Group:   "operators.coreos.com",
		Version: "v1alpha1",
		Kind:    "ClusterServiceVersion",
	}, "", api.ListOptions{})
	if err != nil {
		return images, err
	}

	err = csvList.EachListItem(appendImageFromAnnotation)
	return images, err
}

func parseNodeSelector(selector string) map[string]string {
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
