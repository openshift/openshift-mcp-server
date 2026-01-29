package oadp

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// ListServerStatusRequests lists all ServerStatusRequests in a namespace
func ListServerStatusRequests(ctx context.Context, client dynamic.Interface, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return client.Resource(ServerStatusRequestGVR).Namespace(namespace).List(ctx, opts)
}

// GetServerStatusRequest retrieves a ServerStatusRequest by namespace and name
func GetServerStatusRequest(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(ServerStatusRequestGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// CreateServerStatusRequest creates a new ServerStatusRequest
func CreateServerStatusRequest(ctx context.Context, client dynamic.Interface, ssr *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	namespace := ssr.GetNamespace()
	if namespace == "" {
		namespace = DefaultOADPNamespace
	}
	return client.Resource(ServerStatusRequestGVR).Namespace(namespace).Create(ctx, ssr, metav1.CreateOptions{})
}

// DeleteServerStatusRequest deletes a ServerStatusRequest
func DeleteServerStatusRequest(ctx context.Context, client dynamic.Interface, namespace, name string) error {
	return client.Resource(ServerStatusRequestGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
