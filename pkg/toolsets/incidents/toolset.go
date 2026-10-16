// Package incidents exposes OpenShift cluster health analyzer incidents as MCP tools.
package incidents

import (
	"context"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
	analyzer "github.com/openshift/cluster-health-analyzer/pkg/mcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ToolsetName identifies the incidents toolset in server configuration.
const ToolsetName = "observability/incidents"

var monitoringUIPluginGVR = schema.GroupVersionResource{
	Group: "observability.openshift.io", Version: "v1alpha1", Resource: "uiplugins",
}

// Toolset provides incident tools gated by the monitoring UIPlugin configuration.
type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

// GetName returns the toolset's configuration name.
func (t *Toolset) GetName() string {
	return ToolsetName
}

// GetDescription returns a human-readable description of the toolset.
func (t *Toolset) GetDescription() string {
	return strings.Join(strings.Fields(analyzer.GetIncidentsDescription), " ")
}

// GetTools returns the incident tool when at least one target's monitoring UIPlugin
// has spec.monitoring.clusterHealthAnalyzer.enabled set to true.
// It returns nil without an inspector or when no target confirms enablement,
// including when inspection fails.
func (t *Toolset) GetTools(ctx context.Context, toolsetContext api.ToolsetContext) []api.ServerTool {
	if toolsetContext.Inspector == nil {
		return nil
	}
	enabled, err := toolsetContext.Inspector.Unstructured().
		Resource(monitoringUIPluginGVR).
		Get(ctx, "monitoring", metav1.GetOptions{}).
		Any(func(plugin *unstructured.Unstructured) bool {
			if plugin == nil {
				return false
			}
			value, found, err := unstructured.NestedBool(
				plugin.Object, "spec", "monitoring", "clusterHealthAnalyzer", "enabled",
			)
			return err == nil && found && value
		})
	if err != nil || !enabled {
		return nil
	}
	return initGetIncidents()
}

// GetPrompts returns nil because this toolset provides no prompts.
func (t *Toolset) GetPrompts(_ context.Context, _ api.ToolsetContext) []api.ServerPrompt {
	return nil
}

// GetResources returns nil because this toolset provides no resources.
func (t *Toolset) GetResources(_ context.Context, _ api.ToolsetContext) []api.ServerResource {
	return nil
}

// GetResourceTemplates returns nil because this toolset provides no resource templates.
func (t *Toolset) GetResourceTemplates(_ context.Context, _ api.ToolsetContext) []api.ServerResourceTemplate {
	return nil
}

func init() {
	toolsets.Register(&Toolset{})
}
