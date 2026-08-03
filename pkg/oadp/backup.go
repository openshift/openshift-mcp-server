package oadp

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// GetBackup retrieves a Backup by namespace and name
func GetBackup(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(BackupGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// ListBackups lists all backups in a namespace
func ListBackups(ctx context.Context, client dynamic.Interface, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return client.Resource(BackupGVR).Namespace(namespace).List(ctx, opts)
}
