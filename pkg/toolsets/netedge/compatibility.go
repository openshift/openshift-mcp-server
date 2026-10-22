package netedge

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
)

// When a handler's Kubernetes API dependency changes, review its colocated GVK,
// registration contract tests, and catalog negative tests together.
func targetHasGVK(inspector api.ClusterInspector, gvk schema.GroupVersionKind) api.TargetCompatibilityFilter {
	return func(ctx context.Context) bool {
		return api.AnyTargetHasGVK(ctx, inspector, gvk)
	}
}
