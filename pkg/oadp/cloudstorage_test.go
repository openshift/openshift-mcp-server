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

type CloudStorageSuite struct {
	suite.Suite
	ctx    context.Context
	client *fake.FakeDynamicClient
}

func (s *CloudStorageSuite) SetupTest() {
	s.ctx = context.Background()
	gvrToListKind := map[schema.GroupVersionResource]string{
		CloudStorageGVR: "CloudStorageList",
	}
	s.client = fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind)
}

func (s *CloudStorageSuite) createTestCloudStorage(name, namespace string) *unstructured.Unstructured {
	cs := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": OADPGroup + "/" + OADPVersion,
			"kind":       "CloudStorage",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"name":     "test-bucket",
				"provider": "aws",
				"creationSecret": map[string]any{
					"name": "cloud-credentials",
					"key":  "cloud",
				},
			},
		},
	}
	created, err := s.client.Resource(CloudStorageGVR).Namespace(namespace).Create(s.ctx, cs, metav1.CreateOptions{})
	s.Require().NoError(err)
	return created
}

func (s *CloudStorageSuite) TestListCloudStorages() {
	s.Run("returns empty list when no cloud storages exist", func() {
		list, err := ListCloudStorages(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Empty(list.Items)
	})

	s.Run("returns cloud storages in namespace", func() {
		s.createTestCloudStorage("test-cs", DefaultOADPNamespace)

		list, err := ListCloudStorages(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Len(list.Items, 1)
		s.Equal("test-cs", list.Items[0].GetName())
	})
}

func (s *CloudStorageSuite) TestGetCloudStorage() {
	s.Run("returns cloud storage by name", func() {
		s.createTestCloudStorage("get-test", DefaultOADPNamespace)

		cs, err := GetCloudStorage(s.ctx, s.client, DefaultOADPNamespace, "get-test")
		s.NoError(err)
		s.Equal("get-test", cs.GetName())
	})

	s.Run("returns error for non-existent cloud storage", func() {
		_, err := GetCloudStorage(s.ctx, s.client, DefaultOADPNamespace, "non-existent")
		s.Error(err)
	})
}

func (s *CloudStorageSuite) TestCreateCloudStorage() {
	s.Run("creates cloud storage", func() {
		cs := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": OADPGroup + "/" + OADPVersion,
				"kind":       "CloudStorage",
				"metadata": map[string]any{
					"name":      "create-test",
					"namespace": DefaultOADPNamespace,
				},
				"spec": map[string]any{
					"name":     "new-bucket",
					"provider": "aws",
					"creationSecret": map[string]any{
						"name": "cloud-credentials",
						"key":  "cloud",
					},
				},
			},
		}

		created, err := CreateCloudStorage(s.ctx, s.client, cs)
		s.NoError(err)
		s.Equal("create-test", created.GetName())

		// Verify it was created
		fetched, err := GetCloudStorage(s.ctx, s.client, DefaultOADPNamespace, "create-test")
		s.NoError(err)
		s.Equal("create-test", fetched.GetName())
	})
}

func (s *CloudStorageSuite) TestDeleteCloudStorage() {
	s.Run("deletes existing cloud storage", func() {
		s.createTestCloudStorage("delete-test", DefaultOADPNamespace)

		err := DeleteCloudStorage(s.ctx, s.client, DefaultOADPNamespace, "delete-test")
		s.NoError(err)

		// Verify it's deleted
		_, err = GetCloudStorage(s.ctx, s.client, DefaultOADPNamespace, "delete-test")
		s.Error(err)
	})

	s.Run("returns error for non-existent cloud storage", func() {
		err := DeleteCloudStorage(s.ctx, s.client, DefaultOADPNamespace, "non-existent")
		s.Error(err)
	})
}

func TestCloudStorageSuite(t *testing.T) {
	suite.Run(t, new(CloudStorageSuite))
}
