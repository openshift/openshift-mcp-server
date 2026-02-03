package openshift

import (
	"slices"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
)

type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

func (t *Toolset) GetName() string {
	return "openshift-core"
}

func (t *Toolset) GetDescription() string {
	return "Core OpenShift-specific tools (Node debugging, etc.)"
}

func (t *Toolset) GetTools(o api.Openshift) []api.ServerTool {
	return slices.Concat(
		initNodes(),
	)
}

func (t *Toolset) GetPrompts() []api.ServerPrompt {
	return nil
}

func init() {
	toolsets.Register(&Toolset{})
}
