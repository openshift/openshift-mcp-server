package ovnkubernetes

import (
	"context"
	"slices"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/ovnkubernetes/ovn"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/ovnkubernetes/ovs"
)

type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

func (t *Toolset) GetName() string {
	return "ovn-kubernetes"
}

func (t *Toolset) GetDescription() string {
	return "OVN-Kubernetes CNI network troubleshooting tools"
}

func (t *Toolset) GetTools(_ context.Context, _ api.ToolsetContext) []api.ServerTool {
	return slices.Concat(ovn.InitOVNTools(), ovs.Tools())
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
