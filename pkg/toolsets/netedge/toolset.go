package netedge

import (
	"context"
	"slices"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/netedge/internal/defaults"
)

// Toolset implements the netedge toolset for Network Ingress & DNS troubleshooting.
type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

func (t *Toolset) GetName() string {
	return defaults.ToolsetName()
}

func (t *Toolset) GetDescription() string {
	return defaults.ToolsetDescription()
}

func (t *Toolset) GetTools(_ context.Context, _ api.ToolsetContext) []api.ServerTool {
	return slices.Concat(
		InitQueryPrometheus(),
		initCoreDNS(),
		initEndpoints(),
		initProbeDNSLocal(),
		initProbeHTTP(),
		initRoutes(),
		initExecDNSInPod(),
		initRouter(),
	)
}

func (t *Toolset) GetPrompts(_ context.Context, _ api.ToolsetContext) []api.ServerPrompt {
	// NetEdge toolset presently does not provide prompts
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
