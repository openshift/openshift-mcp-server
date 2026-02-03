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

type PodVolumeSuite struct {
	suite.Suite
	ctx    context.Context
	client *fake.FakeDynamicClient
}

func (s *PodVolumeSuite) SetupTest() {
	s.ctx = context.Background()
	gvrToListKind := map[schema.GroupVersionResource]string{
		PodVolumeBackupGVR:  "PodVolumeBackupList",
		PodVolumeRestoreGVR: "PodVolumeRestoreList",
	}
	s.client = fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind)
}

func (s *PodVolumeSuite) createTestPodVolumeBackup(name, namespace string) *unstructured.Unstructured {
	pvb := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": VeleroGroup + "/" + VeleroVersion,
			"kind":       "PodVolumeBackup",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"node": "test-node",
				"pod": map[string]any{
					"name":      "test-pod",
					"namespace": "test-ns",
				},
				"volume":                "test-volume",
				"backupStorageLocation": "default",
			},
		},
	}
	created, err := s.client.Resource(PodVolumeBackupGVR).Namespace(namespace).Create(s.ctx, pvb, metav1.CreateOptions{})
	s.Require().NoError(err)
	return created
}

func (s *PodVolumeSuite) createTestPodVolumeRestore(name, namespace string) *unstructured.Unstructured {
	pvr := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": VeleroGroup + "/" + VeleroVersion,
			"kind":       "PodVolumeRestore",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"pod": map[string]any{
					"name":      "test-pod",
					"namespace": "test-ns",
				},
				"volume":                "test-volume",
				"backupStorageLocation": "default",
			},
		},
	}
	created, err := s.client.Resource(PodVolumeRestoreGVR).Namespace(namespace).Create(s.ctx, pvr, metav1.CreateOptions{})
	s.Require().NoError(err)
	return created
}

func (s *PodVolumeSuite) TestListPodVolumeBackups() {
	s.Run("returns empty list when no pod volume backups exist", func() {
		list, err := ListPodVolumeBackups(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Empty(list.Items)
	})

	s.Run("returns pod volume backups in namespace", func() {
		s.createTestPodVolumeBackup("test-pvb", DefaultOADPNamespace)

		list, err := ListPodVolumeBackups(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Len(list.Items, 1)
		s.Equal("test-pvb", list.Items[0].GetName())
	})
}

func (s *PodVolumeSuite) TestGetPodVolumeBackup() {
	s.Run("returns pod volume backup by name", func() {
		s.createTestPodVolumeBackup("get-test", DefaultOADPNamespace)

		pvb, err := GetPodVolumeBackup(s.ctx, s.client, DefaultOADPNamespace, "get-test")
		s.NoError(err)
		s.Equal("get-test", pvb.GetName())
	})

	s.Run("returns error for non-existent pod volume backup", func() {
		_, err := GetPodVolumeBackup(s.ctx, s.client, DefaultOADPNamespace, "non-existent")
		s.Error(err)
	})
}

func (s *PodVolumeSuite) TestListPodVolumeRestores() {
	s.Run("returns empty list when no pod volume restores exist", func() {
		list, err := ListPodVolumeRestores(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Empty(list.Items)
	})

	s.Run("returns pod volume restores in namespace", func() {
		s.createTestPodVolumeRestore("test-pvr", DefaultOADPNamespace)

		list, err := ListPodVolumeRestores(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Len(list.Items, 1)
		s.Equal("test-pvr", list.Items[0].GetName())
	})
}

func (s *PodVolumeSuite) TestGetPodVolumeRestore() {
	s.Run("returns pod volume restore by name", func() {
		s.createTestPodVolumeRestore("get-test", DefaultOADPNamespace)

		pvr, err := GetPodVolumeRestore(s.ctx, s.client, DefaultOADPNamespace, "get-test")
		s.NoError(err)
		s.Equal("get-test", pvr.GetName())
	})

	s.Run("returns error for non-existent pod volume restore", func() {
		_, err := GetPodVolumeRestore(s.ctx, s.client, DefaultOADPNamespace, "non-existent")
		s.Error(err)
	})
}

func TestPodVolumeSuite(t *testing.T) {
	suite.Run(t, new(PodVolumeSuite))
}
