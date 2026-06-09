package mustgather

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

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

// isAllowedImageRegistry checks if the image reference starts with one of the allowed Red Hat registries.
func isAllowedImageRegistry(image string) bool {
	for _, registry := range allowedImageRegistries {
		if strings.HasPrefix(image, registry+"/") {
			return true
		}
	}
	return false
}

// isAllowedImageRegistry checks if the gather command specified by user is allowed or not.
func isAllowedGatherCommand(command string) bool {
	for _, knownCmd := range allowedGatherCommands {
		if command == knownCmd {
			return true
		}
	}

	return false
}

// ParseNodeSelector parses a comma-separated key=value selector string into a map.
func ParseNodeSelector(selector string) map[string]string {
	if selector == "" {
		return nil
	}

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
