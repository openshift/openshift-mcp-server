package oadp

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// ListDataUploads lists all DataUploads in a namespace
func ListDataUploads(ctx context.Context, client dynamic.Interface, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return client.Resource(DataUploadGVR).Namespace(namespace).List(ctx, opts)
}

// GetDataUpload retrieves a DataUpload by namespace and name
func GetDataUpload(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(DataUploadGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// CancelDataUpload sets the cancel field on a DataUpload to request cancellation
func CancelDataUpload(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	du, err := GetDataUpload(ctx, client, namespace, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get DataUpload: %w", err)
	}

	if err := unstructured.SetNestedField(du.Object, true, "spec", "cancel"); err != nil {
		return nil, fmt.Errorf("failed to set cancel field: %w", err)
	}

	return client.Resource(DataUploadGVR).Namespace(namespace).Update(ctx, du, metav1.UpdateOptions{})
}

// ListDataDownloads lists all DataDownloads in a namespace
func ListDataDownloads(ctx context.Context, client dynamic.Interface, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return client.Resource(DataDownloadGVR).Namespace(namespace).List(ctx, opts)
}

// GetDataDownload retrieves a DataDownload by namespace and name
func GetDataDownload(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(DataDownloadGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// CancelDataDownload sets the cancel field on a DataDownload to request cancellation
func CancelDataDownload(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	dd, err := GetDataDownload(ctx, client, namespace, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get DataDownload: %w", err)
	}

	if err := unstructured.SetNestedField(dd.Object, true, "spec", "cancel"); err != nil {
		return nil, fmt.Errorf("failed to set cancel field: %w", err)
	}

	return client.Resource(DataDownloadGVR).Namespace(namespace).Update(ctx, dd, metav1.UpdateOptions{})
}
