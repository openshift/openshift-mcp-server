package olm

import (
	"context"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// OLM API identifiers used by OpenShift's two operator lifecycle implementations.
var (
	SubscriptionGVK     = schema.GroupVersionKind{Group: "operators.coreos.com", Version: "v1alpha1", Kind: "Subscription"}
	CSVGVK              = schema.GroupVersionKind{Group: "operators.coreos.com", Version: "v1alpha1", Kind: "ClusterServiceVersion"}
	InstallPlanGVK      = schema.GroupVersionKind{Group: "operators.coreos.com", Version: "v1alpha1", Kind: "InstallPlan"}
	CatalogSourceGVK    = schema.GroupVersionKind{Group: "operators.coreos.com", Version: "v1alpha1", Kind: "CatalogSource"}
	ClusterExtensionGVK = schema.GroupVersionKind{Group: "olm.operatorframework.io", Version: "v1", Kind: "ClusterExtension"}
	ClusterCatalogGVK   = schema.GroupVersionKind{Group: "olm.operatorframework.io", Version: "v1", Kind: "ClusterCatalog"}
)

// Toolset provides read-only semantic OLMv0 and OLMv1 inspection tools.
type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

func (t *Toolset) GetName() string { return "openshift/olm" }

func (t *Toolset) GetDescription() string {
	return "Read-only OpenShift Operator Lifecycle Manager tools for OLMv0 and OLMv1 resources, catalogs, status, and diagnostics"
}

func (t *Toolset) GetTools(p api.FilteringProvider) []api.ServerTool {
	return tools(p)
}

func (t *Toolset) GetPrompts() []api.ServerPrompt                     { return nil }
func (t *Toolset) GetResources() []api.ServerResource                 { return nil }
func (t *Toolset) GetResourceTemplates() []api.ServerResourceTemplate { return nil }

func init() { toolsets.Register(&Toolset{}) }

func hasAnyOLMAPI(p api.FilteringProvider, gvks ...schema.GroupVersionKind) func() bool {
	return func() bool {
		if p == nil || !p.IsTargetCompatibilityToolFiltersEnabled() {
			return true
		}
		for _, gvk := range gvks {
			if p.AnyTargetHasGVKs(context.TODO(), []schema.GroupVersionKind{gvk}) {
				return true
			}
		}
		return false
	}
}
