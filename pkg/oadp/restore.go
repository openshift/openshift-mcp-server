package oadp

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// GetRestore retrieves a Restore by namespace and name
func GetRestore(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(RestoreGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// ListRestores lists all restores in a namespace
func ListRestores(ctx context.Context, client dynamic.Interface, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return client.Resource(RestoreGVR).Namespace(namespace).List(ctx, opts)
}

// CreateRestore creates a new restore
func CreateRestore(ctx context.Context, client dynamic.Interface, restore *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	namespace := restore.GetNamespace()
	if namespace == "" {
		namespace = DefaultOADPNamespace
	}
	return client.Resource(RestoreGVR).Namespace(namespace).Create(ctx, restore, metav1.CreateOptions{})
}

// DeleteRestore deletes a restore record
func DeleteRestore(ctx context.Context, client dynamic.Interface, namespace, name string) error {
	return client.Resource(RestoreGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// GetRestorePhase retrieves the restore phase/status
func GetRestorePhase(restore *unstructured.Unstructured) (string, bool, error) {
	return unstructured.NestedString(restore.Object, "status", "phase")
}

// GetRestoreStatus retrieves detailed restore status information including phase, errors, warnings, and timestamps
func GetRestoreStatus(ctx context.Context, client dynamic.Interface, namespace, name string) (string, error) {
	restore, err := GetRestore(ctx, client, namespace, name)
	if err != nil {
		return "", fmt.Errorf("failed to get restore: %w", err)
	}

	phase, _, _ := GetRestorePhase(restore)
	errors, _, _ := unstructured.NestedInt64(restore.Object, "status", "errors")
	warnings, _, _ := unstructured.NestedInt64(restore.Object, "status", "warnings")
	startTime, _, _ := unstructured.NestedString(restore.Object, "status", "startTimestamp")
	completionTime, _, _ := unstructured.NestedString(restore.Object, "status", "completionTimestamp")
	failureReason, _, _ := unstructured.NestedString(restore.Object, "status", "failureReason")

	result := fmt.Sprintf("Restore: %s/%s\nPhase: %s\nStart Time: %s\nCompletion Time: %s\nErrors: %d\nWarnings: %d",
		namespace, name, phase, startTime, completionTime, errors, warnings)

	if failureReason != "" {
		result += fmt.Sprintf("\nFailure Reason: %s", failureReason)
	}

	return result, nil
}
