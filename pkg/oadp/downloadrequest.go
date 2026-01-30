package oadp

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// ListDownloadRequests lists all DownloadRequests in a namespace
func ListDownloadRequests(ctx context.Context, client dynamic.Interface, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return client.Resource(DownloadRequestGVR).Namespace(namespace).List(ctx, opts)
}

// GetDownloadRequest retrieves a DownloadRequest by namespace and name
func GetDownloadRequest(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(DownloadRequestGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// CreateDownloadRequest creates a new DownloadRequest
func CreateDownloadRequest(ctx context.Context, client dynamic.Interface, dr *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	namespace := dr.GetNamespace()
	if namespace == "" {
		namespace = DefaultOADPNamespace
	}
	return client.Resource(DownloadRequestGVR).Namespace(namespace).Create(ctx, dr, metav1.CreateOptions{})
}

// DeleteDownloadRequest deletes a DownloadRequest
func DeleteDownloadRequest(ctx context.Context, client dynamic.Interface, namespace, name string) error {
	return client.Resource(DownloadRequestGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
