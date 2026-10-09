package netobserv

import (
	"context"
	"slices"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/config"
	netobservclient "github.com/containers/kubernetes-mcp-server/pkg/netobserv"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/netobserv/internal/defaults"
	netobservTools "github.com/containers/kubernetes-mcp-server/pkg/toolsets/netobserv/tools"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Toolset struct{}

var _ api.ConfiguredToolset = (*Toolset)(nil)

func (t *Toolset) GetName() string {
	return defaults.ToolsetName()
}

func (t *Toolset) GetDescription() string {
	return defaults.ToolsetDescription()
}

func (t *Toolset) GetTools(p api.FilteringProvider) []api.ServerTool {
	return t.GetToolsWithConfig(p, nil)
}

// GetToolsWithConfig preserves custom plugin endpoints even when the target
// cluster does not expose the FlowCollector API.
func (t *Toolset) GetToolsWithConfig(p api.FilteringProvider, cfg *config.Config) []api.ServerTool {
	explicitURL := false
	if cfg != nil {
		if extended, ok := cfg.GetToolsetConfig("netobserv"); ok {
			if netobservConfig, ok := extended.(*netobservclient.Config); ok && netobservConfig != nil {
				explicitURL = strings.TrimSpace(netobservConfig.Url) != ""
			}
		}
	}
	compatible := func() bool {
		// Without discovery support, retain tools rather than assume absence.
		if explicitURL || p == nil {
			return true
		}
		return p.AnyTargetHasGVKs(context.TODO(), []schema.GroupVersionKind{
			{Group: "flows.netobserv.io", Kind: "FlowCollector"},
		})
	}
	tools := slices.Concat(
		netobservTools.InitListFlows(),
		netobservTools.InitGetFlowMetrics(),
		netobservTools.InitExportFlows(),
	)
	for i := range tools {
		tools[i].TargetCompatibilityFilters = append(tools[i].TargetCompatibilityFilters, compatible)
	}
	return tools
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
