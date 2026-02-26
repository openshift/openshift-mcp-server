package oadp

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// ListBackupRepositories lists all BackupRepositories in a namespace
func ListBackupRepositories(ctx context.Context, client dynamic.Interface, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return client.Resource(BackupRepositoryGVR).Namespace(namespace).List(ctx, opts)
}

// GetBackupRepository retrieves a BackupRepository by namespace and name
func GetBackupRepository(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(BackupRepositoryGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// DeleteBackupRepository deletes a BackupRepository
func DeleteBackupRepository(ctx context.Context, client dynamic.Interface, namespace, name string) error {
	return client.Resource(BackupRepositoryGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
