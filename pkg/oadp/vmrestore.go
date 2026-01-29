package oadp

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// VirtualMachineBackupsDiscovery operations

// ListVirtualMachineBackupsDiscoveries lists all VirtualMachineBackupsDiscoveries in a namespace
func ListVirtualMachineBackupsDiscoveries(ctx context.Context, client dynamic.Interface, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return client.Resource(VirtualMachineBackupsDiscoveryGVR).Namespace(namespace).List(ctx, opts)
}

// GetVirtualMachineBackupsDiscovery retrieves a VirtualMachineBackupsDiscovery by namespace and name
func GetVirtualMachineBackupsDiscovery(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(VirtualMachineBackupsDiscoveryGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// CreateVirtualMachineBackupsDiscovery creates a new VirtualMachineBackupsDiscovery
func CreateVirtualMachineBackupsDiscovery(ctx context.Context, client dynamic.Interface, vmbd *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	namespace := vmbd.GetNamespace()
	return client.Resource(VirtualMachineBackupsDiscoveryGVR).Namespace(namespace).Create(ctx, vmbd, metav1.CreateOptions{})
}

// DeleteVirtualMachineBackupsDiscovery deletes a VirtualMachineBackupsDiscovery
func DeleteVirtualMachineBackupsDiscovery(ctx context.Context, client dynamic.Interface, namespace, name string) error {
	return client.Resource(VirtualMachineBackupsDiscoveryGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// VirtualMachineFileRestore operations

// ListVirtualMachineFileRestores lists all VirtualMachineFileRestores in a namespace
func ListVirtualMachineFileRestores(ctx context.Context, client dynamic.Interface, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return client.Resource(VirtualMachineFileRestoreGVR).Namespace(namespace).List(ctx, opts)
}

// GetVirtualMachineFileRestore retrieves a VirtualMachineFileRestore by namespace and name
func GetVirtualMachineFileRestore(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return client.Resource(VirtualMachineFileRestoreGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// CreateVirtualMachineFileRestore creates a new VirtualMachineFileRestore
func CreateVirtualMachineFileRestore(ctx context.Context, client dynamic.Interface, vmfr *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	namespace := vmfr.GetNamespace()
	return client.Resource(VirtualMachineFileRestoreGVR).Namespace(namespace).Create(ctx, vmfr, metav1.CreateOptions{})
}

// DeleteVirtualMachineFileRestore deletes a VirtualMachineFileRestore
func DeleteVirtualMachineFileRestore(ctx context.Context, client dynamic.Interface, namespace, name string) error {
	return client.Resource(VirtualMachineFileRestoreGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
