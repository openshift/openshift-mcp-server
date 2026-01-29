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

type DownloadRequestSuite struct {
	suite.Suite
	ctx    context.Context
	client *fake.FakeDynamicClient
}

func (s *DownloadRequestSuite) SetupTest() {
	s.ctx = context.Background()
	gvrToListKind := map[schema.GroupVersionResource]string{
		DownloadRequestGVR: "DownloadRequestList",
	}
	s.client = fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind)
}

func (s *DownloadRequestSuite) createTestDownloadRequest(name, namespace string) *unstructured.Unstructured {
	dr := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": VeleroGroup + "/" + VeleroVersion,
			"kind":       "DownloadRequest",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"target": map[string]any{
					"kind": "BackupLog",
					"name": "test-backup",
				},
			},
		},
	}
	created, err := s.client.Resource(DownloadRequestGVR).Namespace(namespace).Create(s.ctx, dr, metav1.CreateOptions{})
	s.Require().NoError(err)
	return created
}

func (s *DownloadRequestSuite) TestListDownloadRequests() {
	s.Run("returns empty list when no download requests exist", func() {
		list, err := ListDownloadRequests(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Empty(list.Items)
	})

	s.Run("returns download requests in namespace", func() {
		s.createTestDownloadRequest("test-dr", DefaultOADPNamespace)

		list, err := ListDownloadRequests(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Len(list.Items, 1)
		s.Equal("test-dr", list.Items[0].GetName())
	})
}

func (s *DownloadRequestSuite) TestGetDownloadRequest() {
	s.Run("returns download request by name", func() {
		s.createTestDownloadRequest("get-test", DefaultOADPNamespace)

		dr, err := GetDownloadRequest(s.ctx, s.client, DefaultOADPNamespace, "get-test")
		s.NoError(err)
		s.Equal("get-test", dr.GetName())
	})

	s.Run("returns error for non-existent download request", func() {
		_, err := GetDownloadRequest(s.ctx, s.client, DefaultOADPNamespace, "non-existent")
		s.Error(err)
	})
}

func (s *DownloadRequestSuite) TestCreateDownloadRequest() {
	s.Run("creates download request", func() {
		dr := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": VeleroGroup + "/" + VeleroVersion,
				"kind":       "DownloadRequest",
				"metadata": map[string]any{
					"name":      "create-test",
					"namespace": DefaultOADPNamespace,
				},
				"spec": map[string]any{
					"target": map[string]any{
						"kind": "BackupLog",
						"name": "test-backup",
					},
				},
			},
		}

		created, err := CreateDownloadRequest(s.ctx, s.client, dr)
		s.NoError(err)
		s.Equal("create-test", created.GetName())

		// Verify it was created
		fetched, err := GetDownloadRequest(s.ctx, s.client, DefaultOADPNamespace, "create-test")
		s.NoError(err)
		s.Equal("create-test", fetched.GetName())
	})
}

func (s *DownloadRequestSuite) TestDeleteDownloadRequest() {
	s.Run("deletes existing download request", func() {
		s.createTestDownloadRequest("delete-test", DefaultOADPNamespace)

		err := DeleteDownloadRequest(s.ctx, s.client, DefaultOADPNamespace, "delete-test")
		s.NoError(err)

		// Verify it's deleted
		_, err = GetDownloadRequest(s.ctx, s.client, DefaultOADPNamespace, "delete-test")
		s.Error(err)
	})

	s.Run("returns error for non-existent download request", func() {
		err := DeleteDownloadRequest(s.ctx, s.client, DefaultOADPNamespace, "non-existent")
		s.Error(err)
	})
}

func TestDownloadRequestSuite(t *testing.T) {
	suite.Run(t, new(DownloadRequestSuite))
}
