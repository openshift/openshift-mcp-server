package openshift

import (
	"context"
	"slices"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/openshift/mustgather"
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
	// OpenShift tools are only available when an OpenShift cluster is detected.
	if !o.IsOpenShift(context.Background()) {
		return []api.ServerTool{}
	}

	return slices.Concat(
		mustgather.Tools(),
	)
}

func (t *Toolset) GetPrompts() []api.ServerPrompt {
	// OpenShift toolset does not provide prompts
	return nil
}

func init() {
	toolsets.Register(&Toolset{})
}
