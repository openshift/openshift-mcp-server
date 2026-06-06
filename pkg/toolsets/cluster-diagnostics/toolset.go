package clusterdiagnostics

import (
	"slices"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/cluster-diagnostics/nodes"
)

type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

func (t *Toolset) GetName() string {
	return "cluster-diagnostics"
}

func (t *Toolset) GetDescription() string {
	return "Tools for cluster diagnostics and troubleshooting"
}

func (t *Toolset) GetTools(o api.Openshift) []api.ServerTool {
	return slices.Concat(
		nodes.InitNodes(),
	)
}

func (t *Toolset) GetPrompts() []api.ServerPrompt {
	return nil
}

func (t *Toolset) GetResources() []api.ServerResource {
	return nil
}

func (t *Toolset) GetResourceTemplates() []api.ServerResourceTemplate {
	return nil
}

func init() {
	toolsets.Register(&Toolset{})
}
