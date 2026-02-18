package netedge

import (
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// Mock implementations
type mockToolCallRequest struct {
	args map[string]interface{}
}

func (m *mockToolCallRequest) GetArguments() map[string]interface{} {
	return m.args
}

func (m *mockToolCallRequest) GetName() string {
	return "mock_tool"
}

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
