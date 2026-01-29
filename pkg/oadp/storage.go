package oadp

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// GetBackupStorageLocation retrieves a BackupStorageLocation by namespace and name
func GetBackupStorageLocation(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(BackupStorageLocationGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// ListBackupStorageLocations lists all BackupStorageLocations in a namespace
func ListBackupStorageLocations(ctx context.Context, client dynamic.Interface, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return client.Resource(BackupStorageLocationGVR).Namespace(namespace).List(ctx, opts)
}

// GetVolumeSnapshotLocation retrieves a VolumeSnapshotLocation by namespace and name
func GetVolumeSnapshotLocation(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(VolumeSnapshotLocationGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// ListVolumeSnapshotLocations lists all VolumeSnapshotLocations in a namespace
func ListVolumeSnapshotLocations(ctx context.Context, client dynamic.Interface, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return client.Resource(VolumeSnapshotLocationGVR).Namespace(namespace).List(ctx, opts)
}
