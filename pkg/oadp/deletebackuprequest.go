package oadp

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// ListDeleteBackupRequests lists all DeleteBackupRequests in a namespace
func ListDeleteBackupRequests(ctx context.Context, client dynamic.Interface, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return client.Resource(DeleteBackupRequestGVR).Namespace(namespace).List(ctx, opts)
}

// GetDeleteBackupRequest retrieves a DeleteBackupRequest by namespace and name
func GetDeleteBackupRequest(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(DeleteBackupRequestGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}
