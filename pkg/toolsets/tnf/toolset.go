package tnf

import (
	"slices"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/tnf/fencing"
)

type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

func (t *Toolset) GetName() string {
	return "tnf"
}

func (t *Toolset) GetDescription() string {
	return "Two-Node Fencing (TNF) cluster diagnostics"
}

func (t *Toolset) GetTools(_ api.Openshift) []api.ServerTool {
	return slices.Concat(
		fencing.InitFencing(),
		fencing.InitSTONITH(),
	)
}

func (t *Toolset) GetPrompts() []api.ServerPrompt {
	return initTNFTroubleshoot()
}

func (t *Toolset) GetResources() []api.ServerResource {
	return initMCPResources()
}

func (t *Toolset) GetResourceTemplates() []api.ServerResourceTemplate {
	return nil
}

func init() {
	toolsets.Register(&Toolset{})
}
