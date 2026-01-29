package oadp

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// ListPodVolumeBackups lists all PodVolumeBackups in a namespace
func ListPodVolumeBackups(ctx context.Context, client dynamic.Interface, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return client.Resource(PodVolumeBackupGVR).Namespace(namespace).List(ctx, opts)
}

// GetPodVolumeBackup retrieves a PodVolumeBackup by namespace and name
func GetPodVolumeBackup(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(PodVolumeBackupGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// ListPodVolumeRestores lists all PodVolumeRestores in a namespace
func ListPodVolumeRestores(ctx context.Context, client dynamic.Interface, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return client.Resource(PodVolumeRestoreGVR).Namespace(namespace).List(ctx, opts)
}

// GetPodVolumeRestore retrieves a PodVolumeRestore by namespace and name
func GetPodVolumeRestore(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(PodVolumeRestoreGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}
