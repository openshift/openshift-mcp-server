package openshift

import (
	"slices"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
)

type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

func (t *Toolset) GetName() string {
	return "openshift"
}

func (t *Toolset) GetDescription() string {
	return "OpenShift-specific tools for cluster management and troubleshooting"
}

func (t *Toolset) GetTools(o api.Openshift) []api.ServerTool {
	return slices.Concat(
		initMustGatherPlan(o),
	)
}

func (t *Toolset) GetPrompts() []api.ServerPrompt {
	// OpenShift toolset does not provide prompts
	return nil
}

func init() {
	toolsets.Register(&Toolset{})
}
