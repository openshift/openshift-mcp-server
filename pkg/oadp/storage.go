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

// CreateBackupStorageLocation creates a new BackupStorageLocation
func CreateBackupStorageLocation(ctx context.Context, client dynamic.Interface, bsl *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	namespace := bsl.GetNamespace()
	if namespace == "" {
		namespace = DefaultOADPNamespace
	}
	return client.Resource(BackupStorageLocationGVR).Namespace(namespace).Create(ctx, bsl, metav1.CreateOptions{})
}

// UpdateBackupStorageLocation updates an existing BackupStorageLocation
func UpdateBackupStorageLocation(ctx context.Context, client dynamic.Interface, bsl *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	namespace := bsl.GetNamespace()
	if namespace == "" {
		namespace = DefaultOADPNamespace
	}
	return client.Resource(BackupStorageLocationGVR).Namespace(namespace).Update(ctx, bsl, metav1.UpdateOptions{})
}

// DeleteBackupStorageLocation deletes a BackupStorageLocation
func DeleteBackupStorageLocation(ctx context.Context, client dynamic.Interface, namespace, name string) error {
	return client.Resource(BackupStorageLocationGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// CreateVolumeSnapshotLocation creates a new VolumeSnapshotLocation
func CreateVolumeSnapshotLocation(ctx context.Context, client dynamic.Interface, vsl *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	namespace := vsl.GetNamespace()
	if namespace == "" {
		namespace = DefaultOADPNamespace
	}
	return client.Resource(VolumeSnapshotLocationGVR).Namespace(namespace).Create(ctx, vsl, metav1.CreateOptions{})
}

// UpdateVolumeSnapshotLocation updates an existing VolumeSnapshotLocation
func UpdateVolumeSnapshotLocation(ctx context.Context, client dynamic.Interface, vsl *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	namespace := vsl.GetNamespace()
	if namespace == "" {
		namespace = DefaultOADPNamespace
	}
	return client.Resource(VolumeSnapshotLocationGVR).Namespace(namespace).Update(ctx, vsl, metav1.UpdateOptions{})
}

// DeleteVolumeSnapshotLocation deletes a VolumeSnapshotLocation
func DeleteVolumeSnapshotLocation(ctx context.Context, client dynamic.Interface, namespace, name string) error {
	return client.Resource(VolumeSnapshotLocationGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
