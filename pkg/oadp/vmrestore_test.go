package oadp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

type VMRestoreSuite struct {
	suite.Suite
	ctx    context.Context
	client *fake.FakeDynamicClient
}

func (s *VMRestoreSuite) SetupTest() {
	s.ctx = context.Background()
	gvrToListKind := map[schema.GroupVersionResource]string{
		VirtualMachineBackupsDiscoveryGVR: "VirtualMachineBackupsDiscoveryList",
		VirtualMachineFileRestoreGVR:      "VirtualMachineFileRestoreList",
	}
	s.client = fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind)
}

// VirtualMachineBackupsDiscovery tests

func (s *VMRestoreSuite) createTestVMBackupsDiscovery(name, namespace string) *unstructured.Unstructured {
	vmbd := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": OADPGroup + "/" + OADPVersion,
			"kind":       "VirtualMachineBackupsDiscovery",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"virtualMachine": map[string]any{
					"name":      "test-vm",
					"namespace": "vm-ns",
				},
			},
		},
	}
	created, err := s.client.Resource(VirtualMachineBackupsDiscoveryGVR).Namespace(namespace).Create(s.ctx, vmbd, metav1.CreateOptions{})
	s.Require().NoError(err)
	return created
}

func (s *VMRestoreSuite) TestListVMBackupsDiscoveries() {
	s.Run("returns empty list when no VM backup discoveries exist", func() {
		list, err := ListVirtualMachineBackupsDiscoveries(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Empty(list.Items)
	})

	s.Run("returns VM backup discoveries in namespace", func() {
		s.createTestVMBackupsDiscovery("test-vmbd", DefaultOADPNamespace)

		list, err := ListVirtualMachineBackupsDiscoveries(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Len(list.Items, 1)
		s.Equal("test-vmbd", list.Items[0].GetName())
	})
}

func (s *VMRestoreSuite) TestGetVMBackupsDiscovery() {
	s.Run("returns VM backup discovery by name", func() {
		s.createTestVMBackupsDiscovery("get-test", DefaultOADPNamespace)

		vmbd, err := GetVirtualMachineBackupsDiscovery(s.ctx, s.client, DefaultOADPNamespace, "get-test")
		s.NoError(err)
		s.Equal("get-test", vmbd.GetName())
	})

	s.Run("returns error for non-existent VM backup discovery", func() {
		_, err := GetVirtualMachineBackupsDiscovery(s.ctx, s.client, DefaultOADPNamespace, "non-existent")
		s.Error(err)
	})
}

func (s *VMRestoreSuite) TestCreateVMBackupsDiscovery() {
	s.Run("creates VM backup discovery", func() {
		vmbd := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": OADPGroup + "/" + OADPVersion,
				"kind":       "VirtualMachineBackupsDiscovery",
				"metadata": map[string]any{
					"name":      "create-test",
					"namespace": DefaultOADPNamespace,
				},
				"spec": map[string]any{
					"virtualMachine": map[string]any{
						"name":      "test-vm",
						"namespace": "vm-ns",
					},
				},
			},
		}

		created, err := CreateVirtualMachineBackupsDiscovery(s.ctx, s.client, vmbd)
		s.NoError(err)
		s.Equal("create-test", created.GetName())

		// Verify it was created
		fetched, err := GetVirtualMachineBackupsDiscovery(s.ctx, s.client, DefaultOADPNamespace, "create-test")
		s.NoError(err)
		s.Equal("create-test", fetched.GetName())
	})
}

func (s *VMRestoreSuite) TestDeleteVMBackupsDiscovery() {
	s.Run("deletes existing VM backup discovery", func() {
		s.createTestVMBackupsDiscovery("delete-test", DefaultOADPNamespace)

		err := DeleteVirtualMachineBackupsDiscovery(s.ctx, s.client, DefaultOADPNamespace, "delete-test")
		s.NoError(err)

		// Verify it's deleted
		_, err = GetVirtualMachineBackupsDiscovery(s.ctx, s.client, DefaultOADPNamespace, "delete-test")
		s.Error(err)
	})

	s.Run("returns error for non-existent VM backup discovery", func() {
		err := DeleteVirtualMachineBackupsDiscovery(s.ctx, s.client, DefaultOADPNamespace, "non-existent")
		s.Error(err)
	})
}

// VirtualMachineFileRestore tests

func (s *VMRestoreSuite) createTestVMFileRestore(name, namespace string) *unstructured.Unstructured {
	vmfr := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": OADPGroup + "/" + OADPVersion,
			"kind":       "VirtualMachineFileRestore",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"backupName": "test-backup",
				"virtualMachine": map[string]any{
					"name":      "test-vm",
					"namespace": "vm-ns",
				},
			},
		},
	}
	created, err := s.client.Resource(VirtualMachineFileRestoreGVR).Namespace(namespace).Create(s.ctx, vmfr, metav1.CreateOptions{})
	s.Require().NoError(err)
	return created
}

func (s *VMRestoreSuite) TestListVMFileRestores() {
	s.Run("returns empty list when no VM file restores exist", func() {
		list, err := ListVirtualMachineFileRestores(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Empty(list.Items)
	})

	s.Run("returns VM file restores in namespace", func() {
		s.createTestVMFileRestore("test-vmfr", DefaultOADPNamespace)

		list, err := ListVirtualMachineFileRestores(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Len(list.Items, 1)
		s.Equal("test-vmfr", list.Items[0].GetName())
	})
}

func (s *VMRestoreSuite) TestGetVMFileRestore() {
	s.Run("returns VM file restore by name", func() {
		s.createTestVMFileRestore("get-test", DefaultOADPNamespace)

		vmfr, err := GetVirtualMachineFileRestore(s.ctx, s.client, DefaultOADPNamespace, "get-test")
		s.NoError(err)
		s.Equal("get-test", vmfr.GetName())
	})

	s.Run("returns error for non-existent VM file restore", func() {
		_, err := GetVirtualMachineFileRestore(s.ctx, s.client, DefaultOADPNamespace, "non-existent")
		s.Error(err)
	})
}

func (s *VMRestoreSuite) TestCreateVMFileRestore() {
	s.Run("creates VM file restore", func() {
		vmfr := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": OADPGroup + "/" + OADPVersion,
				"kind":       "VirtualMachineFileRestore",
				"metadata": map[string]any{
					"name":      "create-test",
					"namespace": DefaultOADPNamespace,
				},
				"spec": map[string]any{
					"backupName": "test-backup",
					"virtualMachine": map[string]any{
						"name":      "test-vm",
						"namespace": "vm-ns",
					},
				},
			},
		}

		created, err := CreateVirtualMachineFileRestore(s.ctx, s.client, vmfr)
		s.NoError(err)
		s.Equal("create-test", created.GetName())

		// Verify it was created
		fetched, err := GetVirtualMachineFileRestore(s.ctx, s.client, DefaultOADPNamespace, "create-test")
		s.NoError(err)
		s.Equal("create-test", fetched.GetName())
	})
}

func (s *VMRestoreSuite) TestDeleteVMFileRestore() {
	s.Run("deletes existing VM file restore", func() {
		s.createTestVMFileRestore("delete-test", DefaultOADPNamespace)

		err := DeleteVirtualMachineFileRestore(s.ctx, s.client, DefaultOADPNamespace, "delete-test")
		s.NoError(err)

		// Verify it's deleted
		_, err = GetVirtualMachineFileRestore(s.ctx, s.client, DefaultOADPNamespace, "delete-test")
		s.Error(err)
	})

	s.Run("returns error for non-existent VM file restore", func() {
		err := DeleteVirtualMachineFileRestore(s.ctx, s.client, DefaultOADPNamespace, "non-existent")
		s.Error(err)
	})
}

func TestVMRestoreSuite(t *testing.T) {
	suite.Run(t, new(VMRestoreSuite))
}
