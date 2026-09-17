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
	return "Analyze OpenShift must-gather archives offline without a live cluster connection. Call mustgather_list first to discover available archives and their archive_id, then pass that ID to the other mustgather_* tools."
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
	return Prompts()
}

// GetResources returns no MCP resources. Resource handlers have no access to the
// toolset configuration (GetToolsetConfig is not plumbed into MCP resources), so
// they cannot resolve archives against the per-config registry. Must-gather data
// is exposed through the mustgather_* tools instead; resources can be re-added
// once resource handlers gain toolset-config access upstream.
func (t *Toolset) GetResources() []api.ServerResource {
	return nil
}

func (t *Toolset) GetResourceTemplates() []api.ServerResourceTemplate {
	return nil
}

func init() {
	toolsets.Register(&Toolset{})
}
