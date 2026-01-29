package oadp

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// GetDataProtectionApplication retrieves a DataProtectionApplication by namespace and name
func GetDataProtectionApplication(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(DataProtectionApplicationGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// ListDataProtectionApplications lists all DataProtectionApplications in a namespace
func ListDataProtectionApplications(ctx context.Context, client dynamic.Interface, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return client.Resource(DataProtectionApplicationGVR).Namespace(namespace).List(ctx, opts)
}
