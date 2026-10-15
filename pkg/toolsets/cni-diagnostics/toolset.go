package cnidiagnostics

import (
	"context"
	"slices"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/cni-diagnostics/kernel"
	network_tools "github.com/containers/kubernetes-mcp-server/pkg/toolsets/cni-diagnostics/network-tools"
)

type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

func (t *Toolset) GetName() string {
	return "cni-diagnostics"
}

func (t *Toolset) GetDescription() string {
	return "Tools for Container Network Interface (CNI) diagnostics and troubleshooting"
}

func (t *Toolset) GetTools(_ context.Context, _ api.ToolsetContext) []api.ServerTool {
	return slices.Concat(
		kernel.InitKernelTools(),
		network_tools.InitNetworkTools(),
	)
}

func (t *Toolset) GetPrompts(_ context.Context, _ api.ToolsetContext) []api.ServerPrompt {
	return nil
}

func (t *Toolset) GetResources(_ context.Context, _ api.ToolsetContext) []api.ServerResource {
	return nil
}

func (t *Toolset) GetResourceTemplates(_ context.Context, _ api.ToolsetContext) []api.ServerResourceTemplate {
	return nil
}

func init() {
	toolsets.Register(&Toolset{})
}
