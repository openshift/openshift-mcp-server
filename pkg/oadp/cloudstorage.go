package oadp

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// ListCloudStorages lists all CloudStorages in a namespace
func ListCloudStorages(ctx context.Context, client dynamic.Interface, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return client.Resource(CloudStorageGVR).Namespace(namespace).List(ctx, opts)
}

// GetCloudStorage retrieves a CloudStorage by namespace and name
func GetCloudStorage(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(CloudStorageGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// CreateCloudStorage creates a new CloudStorage
func CreateCloudStorage(ctx context.Context, client dynamic.Interface, cs *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	namespace := cs.GetNamespace()
	if namespace == "" {
		namespace = DefaultOADPNamespace
	}
	return client.Resource(CloudStorageGVR).Namespace(namespace).Create(ctx, cs, metav1.CreateOptions{})
}

// DeleteCloudStorage deletes a CloudStorage
func DeleteCloudStorage(ctx context.Context, client dynamic.Interface, namespace, name string) error {
	return client.Resource(CloudStorageGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
