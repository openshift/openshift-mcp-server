package netedge

import (
	"context"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/stretchr/testify/suite"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// mockKubernetesClient implements api.KubernetesClient for testing
type mockKubernetesClient struct {
	api.KubernetesClient
	restConfig    *rest.Config
	dynamicClient dynamic.Interface
}

func (m *mockKubernetesClient) RESTConfig() *rest.Config {
	return m.restConfig
}

func (m *mockKubernetesClient) DynamicClient() dynamic.Interface {
	return m.dynamicClient
}

type NetEdgeTestSuite struct {
	suite.Suite
	params     api.ToolHandlerParams
	mockReq    *mockToolCallRequest
	mockClient *mockKubernetesClient
}

func (s *NetEdgeTestSuite) SetupTest() {
	s.mockReq = &mockToolCallRequest{args: make(map[string]interface{})}
	s.mockClient = &mockKubernetesClient{
		restConfig: &rest.Config{},
	}
	s.params = api.ToolHandlerParams{
		Context:          context.Background(),
		ToolCallRequest:  s.mockReq,
		KubernetesClient: s.mockClient,
	}
}

func (s *NetEdgeTestSuite) SetArgs(args map[string]interface{}) {
	s.mockReq.args = args
}

func (s *NetEdgeTestSuite) SetDynamicClient(dynClient dynamic.Interface) {
	s.mockClient.dynamicClient = dynClient
	s.params.KubernetesClient = s.mockClient
}

func TestNetEdgeSuite(t *testing.T) {
	suite.Run(t, new(NetEdgeTestSuite))
}

type mockToolCallRequest struct {
	args map[string]any
}

func (m *mockToolCallRequest) GetArguments() map[string]any {
	return m.args
}
