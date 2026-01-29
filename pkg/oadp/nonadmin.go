package oadp

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// NonAdminBackup operations

// ListNonAdminBackups lists all NonAdminBackups in a namespace
func ListNonAdminBackups(ctx context.Context, client dynamic.Interface, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return client.Resource(NonAdminBackupGVR).Namespace(namespace).List(ctx, opts)
}

// GetNonAdminBackup retrieves a NonAdminBackup by namespace and name
func GetNonAdminBackup(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(NonAdminBackupGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// CreateNonAdminBackup creates a new NonAdminBackup
func CreateNonAdminBackup(ctx context.Context, client dynamic.Interface, nab *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	namespace := nab.GetNamespace()
	return client.Resource(NonAdminBackupGVR).Namespace(namespace).Create(ctx, nab, metav1.CreateOptions{})
}

// DeleteNonAdminBackup deletes a NonAdminBackup
func DeleteNonAdminBackup(ctx context.Context, client dynamic.Interface, namespace, name string) error {
	return client.Resource(NonAdminBackupGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// NonAdminRestore operations

// ListNonAdminRestores lists all NonAdminRestores in a namespace
func ListNonAdminRestores(ctx context.Context, client dynamic.Interface, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return client.Resource(NonAdminRestoreGVR).Namespace(namespace).List(ctx, opts)
}

// GetNonAdminRestore retrieves a NonAdminRestore by namespace and name
func GetNonAdminRestore(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(NonAdminRestoreGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// CreateNonAdminRestore creates a new NonAdminRestore
func CreateNonAdminRestore(ctx context.Context, client dynamic.Interface, nar *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	namespace := nar.GetNamespace()
	return client.Resource(NonAdminRestoreGVR).Namespace(namespace).Create(ctx, nar, metav1.CreateOptions{})
}

// DeleteNonAdminRestore deletes a NonAdminRestore
func DeleteNonAdminRestore(ctx context.Context, client dynamic.Interface, namespace, name string) error {
	return client.Resource(NonAdminRestoreGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// NonAdminBackupStorageLocation operations

// ListNonAdminBackupStorageLocations lists all NonAdminBackupStorageLocations in a namespace
func ListNonAdminBackupStorageLocations(ctx context.Context, client dynamic.Interface, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return client.Resource(NonAdminBackupStorageLocationGVR).Namespace(namespace).List(ctx, opts)
}

// GetNonAdminBackupStorageLocation retrieves a NonAdminBackupStorageLocation by namespace and name
func GetNonAdminBackupStorageLocation(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(NonAdminBackupStorageLocationGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// CreateNonAdminBackupStorageLocation creates a new NonAdminBackupStorageLocation
func CreateNonAdminBackupStorageLocation(ctx context.Context, client dynamic.Interface, nabsl *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	namespace := nabsl.GetNamespace()
	return client.Resource(NonAdminBackupStorageLocationGVR).Namespace(namespace).Create(ctx, nabsl, metav1.CreateOptions{})
}

// UpdateNonAdminBackupStorageLocation updates an existing NonAdminBackupStorageLocation
func UpdateNonAdminBackupStorageLocation(ctx context.Context, client dynamic.Interface, nabsl *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	namespace := nabsl.GetNamespace()
	return client.Resource(NonAdminBackupStorageLocationGVR).Namespace(namespace).Update(ctx, nabsl, metav1.UpdateOptions{})
}

// DeleteNonAdminBackupStorageLocation deletes a NonAdminBackupStorageLocation
func DeleteNonAdminBackupStorageLocation(ctx context.Context, client dynamic.Interface, namespace, name string) error {
	return client.Resource(NonAdminBackupStorageLocationGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// NonAdminBackupStorageLocationRequest operations

// ListNonAdminBackupStorageLocationRequests lists all NonAdminBackupStorageLocationRequests in a namespace
func ListNonAdminBackupStorageLocationRequests(ctx context.Context, client dynamic.Interface, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return client.Resource(NonAdminBackupStorageLocationRequestGVR).Namespace(namespace).List(ctx, opts)
}

// GetNonAdminBackupStorageLocationRequest retrieves a NonAdminBackupStorageLocationRequest by namespace and name
func GetNonAdminBackupStorageLocationRequest(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(NonAdminBackupStorageLocationRequestGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// ApproveNonAdminBackupStorageLocationRequest sets the approval decision on a request
func ApproveNonAdminBackupStorageLocationRequest(ctx context.Context, client dynamic.Interface, namespace, name, decision string) (*unstructured.Unstructured, error) {
	req, err := GetNonAdminBackupStorageLocationRequest(ctx, client, namespace, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get NonAdminBackupStorageLocationRequest: %w", err)
	}

	if err := unstructured.SetNestedField(req.Object, decision, "spec", "approvalDecision"); err != nil {
		return nil, fmt.Errorf("failed to set approvalDecision field: %w", err)
	}

	return client.Resource(NonAdminBackupStorageLocationRequestGVR).Namespace(namespace).Update(ctx, req, metav1.UpdateOptions{})
}

// NonAdminDownloadRequest operations

// ListNonAdminDownloadRequests lists all NonAdminDownloadRequests in a namespace
func ListNonAdminDownloadRequests(ctx context.Context, client dynamic.Interface, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return client.Resource(NonAdminDownloadRequestGVR).Namespace(namespace).List(ctx, opts)
}

// GetNonAdminDownloadRequest retrieves a NonAdminDownloadRequest by namespace and name
func GetNonAdminDownloadRequest(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(NonAdminDownloadRequestGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// CreateNonAdminDownloadRequest creates a new NonAdminDownloadRequest
func CreateNonAdminDownloadRequest(ctx context.Context, client dynamic.Interface, nadr *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	namespace := nadr.GetNamespace()
	return client.Resource(NonAdminDownloadRequestGVR).Namespace(namespace).Create(ctx, nadr, metav1.CreateOptions{})
}

// DeleteNonAdminDownloadRequest deletes a NonAdminDownloadRequest
func DeleteNonAdminDownloadRequest(ctx context.Context, client dynamic.Interface, namespace, name string) error {
	return client.Resource(NonAdminDownloadRequestGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
