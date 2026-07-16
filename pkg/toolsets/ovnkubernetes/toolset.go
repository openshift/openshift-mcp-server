package ovnkubernetes

import (
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

func (t *Toolset) GetTools(_ api.FilteringProvider) []api.ServerTool {
	return slices.Concat(ovn.InitOVNTools(), ovs.Tools())
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
