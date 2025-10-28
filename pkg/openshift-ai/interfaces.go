package openshiftai

import (
	"context"
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// KubernetesClients provides access to Kubernetes clients for OpenShift AI operations
type KubernetesClients interface {
	GetRESTConfig() *rest.Config
	GetDiscoveryClient() discovery.CachedDiscoveryInterface
	GetDynamicClient() *dynamic.DynamicClient
}

// OpenShiftAIToolset defines the interface for OpenShift AI toolset
type OpenShiftAIToolset interface {
	GetName() string
	GetDescription() string
	IsAvailable(ctx context.Context) bool
	GetTools(clients KubernetesClients) []api.ServerTool
}

// BaseToolset provides common functionality for OpenShift AI toolsets
type BaseToolset struct {
	name        string
	description string
	client      *Client
	detector    *Detector
}

// NewBaseToolset creates a new base toolset
func NewBaseToolset(name, description string, client *Client) *BaseToolset {
	return &BaseToolset{
		name:        name,
		description: description,
		client:      client,
		detector:    NewDetector(client),
	}
}

// GetName returns the name of the toolset
func (b *BaseToolset) GetName() string {
	return b.name
}

// GetDescription returns the description of the toolset
func (b *BaseToolset) GetDescription() string {
	return b.description
}

// IsAvailable checks if OpenShift AI is available
func (b *BaseToolset) IsAvailable(ctx context.Context) bool {
	return b.detector.IsOpenShiftAICluster(ctx)
}

// GetClient returns the OpenShift AI client
func (b *BaseToolset) GetClient() *Client {
	return b.client
}

// GetDetector returns the OpenShift AI detector
func (b *BaseToolset) GetDetector() *Detector {
	return b.detector
}

// ToolHandler defines the common interface for tool handlers
type ToolHandler interface {
	Handle(ctx context.Context, params api.ToolHandlerParams) (*api.ToolCallResult, error)
	Validate(params api.ToolHandlerParams) error
}

// BaseToolHandler provides common functionality for tool handlers
type BaseToolHandler struct {
	name   string
	client *Client
}

// NewBaseToolHandler creates a new base tool handler
func NewBaseToolHandler(name string, client *Client) *BaseToolHandler {
	return &BaseToolHandler{
		name:   name,
		client: client,
	}
}

// GetName returns the name of the tool handler
func (b *BaseToolHandler) GetName() string {
	return b.name
}

// GetClient returns the OpenShift AI client
func (b *BaseToolHandler) GetClient() *Client {
	return b.client
}

// Validate performs basic validation on tool parameters
func (b *BaseToolHandler) Validate(params api.ToolHandlerParams) error {
	if params.GetArguments() == nil {
		return fmt.Errorf("tool arguments are required")
	}
	return nil
}
