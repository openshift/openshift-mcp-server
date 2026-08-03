package oadp

import (
	"context"

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
