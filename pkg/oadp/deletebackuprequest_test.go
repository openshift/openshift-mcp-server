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

type DeleteBackupRequestSuite struct {
	suite.Suite
	ctx    context.Context
	client *fake.FakeDynamicClient
}

func (s *DeleteBackupRequestSuite) SetupTest() {
	s.ctx = context.Background()
	gvrToListKind := map[schema.GroupVersionResource]string{
		DeleteBackupRequestGVR: "DeleteBackupRequestList",
	}
	s.client = fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind)
}

func (s *DeleteBackupRequestSuite) createTestDeleteBackupRequest(name, namespace string) *unstructured.Unstructured {
	dbr := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": VeleroGroup + "/" + VeleroVersion,
			"kind":       "DeleteBackupRequest",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"backupName": "test-backup",
			},
		},
	}
	created, err := s.client.Resource(DeleteBackupRequestGVR).Namespace(namespace).Create(s.ctx, dbr, metav1.CreateOptions{})
	s.Require().NoError(err)
	return created
}

func (s *DeleteBackupRequestSuite) TestListDeleteBackupRequests() {
	s.Run("returns empty list when no delete backup requests exist", func() {
		list, err := ListDeleteBackupRequests(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Empty(list.Items)
	})

	s.Run("returns delete backup requests in namespace", func() {
		s.createTestDeleteBackupRequest("test-dbr", DefaultOADPNamespace)

		list, err := ListDeleteBackupRequests(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Len(list.Items, 1)
		s.Equal("test-dbr", list.Items[0].GetName())
	})
}

func (s *DeleteBackupRequestSuite) TestGetDeleteBackupRequest() {
	s.Run("returns delete backup request by name", func() {
		s.createTestDeleteBackupRequest("get-test", DefaultOADPNamespace)

		dbr, err := GetDeleteBackupRequest(s.ctx, s.client, DefaultOADPNamespace, "get-test")
		s.NoError(err)
		s.Equal("get-test", dbr.GetName())
	})

	s.Run("returns error for non-existent delete backup request", func() {
		_, err := GetDeleteBackupRequest(s.ctx, s.client, DefaultOADPNamespace, "non-existent")
		s.Error(err)
	})
}

func TestDeleteBackupRequestSuite(t *testing.T) {
	suite.Run(t, new(DeleteBackupRequestSuite))
}
