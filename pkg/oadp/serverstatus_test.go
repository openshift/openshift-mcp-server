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

type ServerStatusSuite struct {
	suite.Suite
	ctx    context.Context
	client *fake.FakeDynamicClient
}

func (s *ServerStatusSuite) SetupTest() {
	s.ctx = context.Background()
	gvrToListKind := map[schema.GroupVersionResource]string{
		ServerStatusRequestGVR: "ServerStatusRequestList",
	}
	s.client = fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind)
}

func (s *ServerStatusSuite) createTestServerStatusRequest(name, namespace string) *unstructured.Unstructured {
	ssr := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": VeleroGroup + "/" + VeleroVersion,
			"kind":       "ServerStatusRequest",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{},
		},
	}
	created, err := s.client.Resource(ServerStatusRequestGVR).Namespace(namespace).Create(s.ctx, ssr, metav1.CreateOptions{})
	s.Require().NoError(err)
	return created
}

func (s *ServerStatusSuite) TestListServerStatusRequests() {
	s.Run("returns empty list when no server status requests exist", func() {
		list, err := ListServerStatusRequests(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Empty(list.Items)
	})

	s.Run("returns server status requests in namespace", func() {
		s.createTestServerStatusRequest("test-ssr", DefaultOADPNamespace)

		list, err := ListServerStatusRequests(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Len(list.Items, 1)
		s.Equal("test-ssr", list.Items[0].GetName())
	})
}

func (s *ServerStatusSuite) TestGetServerStatusRequest() {
	s.Run("returns server status request by name", func() {
		s.createTestServerStatusRequest("get-test", DefaultOADPNamespace)

		ssr, err := GetServerStatusRequest(s.ctx, s.client, DefaultOADPNamespace, "get-test")
		s.NoError(err)
		s.Equal("get-test", ssr.GetName())
	})

	s.Run("returns error for non-existent server status request", func() {
		_, err := GetServerStatusRequest(s.ctx, s.client, DefaultOADPNamespace, "non-existent")
		s.Error(err)
	})
}

func (s *ServerStatusSuite) TestCreateServerStatusRequest() {
	s.Run("creates server status request", func() {
		ssr := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": VeleroGroup + "/" + VeleroVersion,
				"kind":       "ServerStatusRequest",
				"metadata": map[string]any{
					"name":      "create-test",
					"namespace": DefaultOADPNamespace,
				},
				"spec": map[string]any{},
			},
		}

		created, err := CreateServerStatusRequest(s.ctx, s.client, ssr)
		s.NoError(err)
		s.Equal("create-test", created.GetName())

		// Verify it was created
		fetched, err := GetServerStatusRequest(s.ctx, s.client, DefaultOADPNamespace, "create-test")
		s.NoError(err)
		s.Equal("create-test", fetched.GetName())
	})
}

func (s *ServerStatusSuite) TestDeleteServerStatusRequest() {
	s.Run("deletes existing server status request", func() {
		s.createTestServerStatusRequest("delete-test", DefaultOADPNamespace)

		err := DeleteServerStatusRequest(s.ctx, s.client, DefaultOADPNamespace, "delete-test")
		s.NoError(err)

		// Verify it's deleted
		_, err = GetServerStatusRequest(s.ctx, s.client, DefaultOADPNamespace, "delete-test")
		s.Error(err)
	})

	s.Run("returns error for non-existent server status request", func() {
		err := DeleteServerStatusRequest(s.ctx, s.client, DefaultOADPNamespace, "non-existent")
		s.Error(err)
	})
}

func TestServerStatusSuite(t *testing.T) {
	suite.Run(t, new(ServerStatusSuite))
}
