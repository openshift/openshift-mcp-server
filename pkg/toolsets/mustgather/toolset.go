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
	return "Analyze OpenShift must-gather archives offline without a live cluster connection. Call mustgather_list first to discover available archives and their must_gather_archive_id, then pass that ID to the other mustgather_* tools."
}

func (t *Toolset) GetTools(_ api.FilteringProvider) []api.ServerTool {
	return slices.Concat(
		initList(),
		initResources(),
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

func (t *Toolset) GetResources() []api.ServerResource {
	return initMCPResources()
}

func (t *Toolset) GetResourceTemplates() []api.ServerResourceTemplate {
	return initMCPResourceTemplates()
}

func init() {
	toolsets.Register(&Toolset{})
}
