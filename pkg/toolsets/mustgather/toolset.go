package mustgather

import (
	"slices"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
)

// Toolset provides tools for analyzing OpenShift must-gather archives offline.
type Toolset struct{}

func (t *Toolset) GetName() string {
	return "openshift/mustgather"
}

func (t *Toolset) GetDescription() string {
	return "Analyze OpenShift must-gather archives offline without a live cluster connection"
}

func (t *Toolset) GetTools(_ api.Openshift) []api.ServerTool {
	return slices.Concat(
		initUse(),
		initResources(),
		initNamespaces(),
		initEvents(),
		initPodLogs(),
		initNodes(),
		initEtcd(),
		initMonitoring(),
	)
}

func (t *Toolset) GetPrompts() []api.ServerPrompt {
	return nil
}

func init() {
	toolsets.Register(&Toolset{})
}
