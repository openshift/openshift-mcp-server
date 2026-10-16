package netedge

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
)

// When a handler's Kubernetes API dependency changes, review its colocated GVK,
// registration contract tests, and catalog negative tests together.
func targetHasGVK(ctx context.Context, inspector api.ClusterInspector, gvk schema.GroupVersionKind) func() bool {
	return func() bool {
		return api.AnyTargetHasGVK(ctx, inspector, gvk)
	}
}
