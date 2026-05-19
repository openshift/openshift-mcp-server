package oadp

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// ListDataProtectionTests lists all DataProtectionTests in a namespace
func ListDataProtectionTests(ctx context.Context, client dynamic.Interface, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return client.Resource(DataProtectionTestGVR).Namespace(namespace).List(ctx, opts)
}

// GetDataProtectionTest retrieves a DataProtectionTest by namespace and name
func GetDataProtectionTest(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(DataProtectionTestGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// CreateDataProtectionTest creates a new DataProtectionTest
func CreateDataProtectionTest(ctx context.Context, client dynamic.Interface, dpt *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	namespace := dpt.GetNamespace()
	if namespace == "" {
		namespace = DefaultOADPNamespace
	}
	return client.Resource(DataProtectionTestGVR).Namespace(namespace).Create(ctx, dpt, metav1.CreateOptions{})
}

// DeleteDataProtectionTest deletes a DataProtectionTest
func DeleteDataProtectionTest(ctx context.Context, client dynamic.Interface, namespace, name string) error {
	return client.Resource(DataProtectionTestGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
