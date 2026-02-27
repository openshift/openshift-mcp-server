package olm

import (
	"slices"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
)

type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

func (t *Toolset) GetName() string {
	return "olm"
}

func (t *Toolset) GetDescription() string {
	return "Tools for tracing and diagnosing OLM (Operator Lifecycle Manager) operator installations"
}

func (t *Toolset) GetTools(_ api.Openshift) []api.ServerTool {
	return slices.Concat(
		initTraceOLM(),
	)
}

func (t *Toolset) GetPrompts() []api.ServerPrompt {
	return nil
}

func init() {
	toolsets.Register(&Toolset{})
}
