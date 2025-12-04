package openshift

import (
	"slices"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	internalk8s "github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/openshift/mustgather"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/openshift/nodes"
)

type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

func (t *Toolset) GetName() string {
	return "openshift-core"
}

func (t *Toolset) GetDescription() string {
	return "Core OpenShift-specific tools (must-gather, Node debugging, etc.)"
}

func (t *Toolset) GetTools(o internalk8s.Openshift) []api.ServerTool {
	return slices.Concat(
		mustgather.MustGatherTools(),
		nodes.NodeTools(),
	)
}

func init() {
	toolsets.Register(&Toolset{})
}
