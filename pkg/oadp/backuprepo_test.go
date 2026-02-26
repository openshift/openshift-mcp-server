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

type BackupRepositorySuite struct {
	suite.Suite
	ctx    context.Context
	client *fake.FakeDynamicClient
}

func (s *BackupRepositorySuite) SetupTest() {
	s.ctx = context.Background()
	gvrToListKind := map[schema.GroupVersionResource]string{
		BackupRepositoryGVR: "BackupRepositoryList",
	}
	s.client = fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind)
}

func (s *BackupRepositorySuite) createTestBackupRepository(name, namespace string) *unstructured.Unstructured {
	br := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": VeleroGroup + "/" + VeleroVersion,
			"kind":       "BackupRepository",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"volumeNamespace":       "test-ns",
				"backupStorageLocation": "default",
				"repositoryType":        "kopia",
			},
		},
	}
	created, err := s.client.Resource(BackupRepositoryGVR).Namespace(namespace).Create(s.ctx, br, metav1.CreateOptions{})
	s.Require().NoError(err)
	return created
}

func (s *BackupRepositorySuite) TestListBackupRepositories() {
	s.Run("returns empty list when no backup repositories exist", func() {
		list, err := ListBackupRepositories(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Empty(list.Items)
	})

	s.Run("returns backup repositories in namespace", func() {
		s.createTestBackupRepository("test-repo", DefaultOADPNamespace)

		list, err := ListBackupRepositories(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Len(list.Items, 1)
		s.Equal("test-repo", list.Items[0].GetName())
	})

	s.Run("filters by label selector", func() {
		br := s.createTestBackupRepository("labeled-repo", DefaultOADPNamespace)
		br.SetLabels(map[string]string{"app": "test"})
		_, err := s.client.Resource(BackupRepositoryGVR).Namespace(DefaultOADPNamespace).Update(s.ctx, br, metav1.UpdateOptions{})
		s.Require().NoError(err)

		list, err := ListBackupRepositories(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{
			LabelSelector: "app=test",
		})
		s.NoError(err)
		s.Len(list.Items, 1)
	})
}

func (s *BackupRepositorySuite) TestGetBackupRepository() {
	s.Run("returns backup repository by name", func() {
		s.createTestBackupRepository("get-test", DefaultOADPNamespace)

		br, err := GetBackupRepository(s.ctx, s.client, DefaultOADPNamespace, "get-test")
		s.NoError(err)
		s.Equal("get-test", br.GetName())
	})

	s.Run("returns error for non-existent backup repository", func() {
		_, err := GetBackupRepository(s.ctx, s.client, DefaultOADPNamespace, "non-existent")
		s.Error(err)
	})
}

func (s *BackupRepositorySuite) TestDeleteBackupRepository() {
	s.Run("deletes existing backup repository", func() {
		s.createTestBackupRepository("delete-test", DefaultOADPNamespace)

		err := DeleteBackupRepository(s.ctx, s.client, DefaultOADPNamespace, "delete-test")
		s.NoError(err)

		// Verify it's deleted
		_, err = GetBackupRepository(s.ctx, s.client, DefaultOADPNamespace, "delete-test")
		s.Error(err)
	})

	s.Run("returns error for non-existent backup repository", func() {
		err := DeleteBackupRepository(s.ctx, s.client, DefaultOADPNamespace, "non-existent")
		s.Error(err)
	})
}

func TestBackupRepositorySuite(t *testing.T) {
	suite.Run(t, new(BackupRepositorySuite))
}
