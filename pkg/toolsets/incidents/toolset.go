package incidents

import (
	"context"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
	analyzerincidents "github.com/openshift/cluster-health-analyzer/pkg/toolset/incidents"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const ToolsetName = "observability/incidents"

var monitoringUIPluginGVR = schema.GroupVersionResource{
	Group: "observability.openshift.io", Version: "v1alpha1", Resource: "uiplugins",
}

type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

func (t *Toolset) GetName() string {
	return ToolsetName
}

func (t *Toolset) GetDescription() string {
	return strings.Join(strings.Fields((&analyzerincidents.Toolset{}).GetDescription()), " ")
}

func (t *Toolset) GetTools(ctx context.Context, toolsetContext api.ToolsetContext) []api.ServerTool {
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
	return (&analyzerincidents.Toolset{}).GetTools(ctx, toolsetContext)
}

func (t *Toolset) GetPrompts(_ context.Context, _ api.ToolsetContext) []api.ServerPrompt {
	return nil
}

func (t *Toolset) GetResources(_ context.Context, _ api.ToolsetContext) []api.ServerResource {
	return nil
}

func (t *Toolset) GetResourceTemplates(_ context.Context, _ api.ToolsetContext) []api.ServerResourceTemplate {
	return nil
}

func init() {
	toolsets.Register(&Toolset{})
}
