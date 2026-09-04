package olm

import (
	"context"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

type filteringProvider struct {
	enabled bool
	has     map[schema.GroupVersionKind]bool
}

func (f filteringProvider) IsTargetCompatibilityToolFiltersEnabled() bool { return f.enabled }
func (f filteringProvider) AnyTargetHasGVKs(_ context.Context, gvks []schema.GroupVersionKind) bool {
	for _, gvk := range gvks {
		if f.has[gvk] {
			return true
		}
	}
	return false
}

func TestToolsetMetadata(t *testing.T) {
	tools := (&Toolset{}).GetTools(filteringProvider{enabled: true, has: map[schema.GroupVersionKind]bool{
		SubscriptionGVK:     true,
		CatalogSourceGVK:    true,
		ClusterExtensionGVK: true,
		ClusterCatalogGVK:   true,
	}})
	require.Len(t, tools, 4)
	for _, tool := range tools {
		require.Equal(t, "object", tool.Tool.InputSchema.Type)
		require.NotNil(t, tool.Tool.OutputSchema)
		require.NotNil(t, tool.Tool.Annotations.ReadOnlyHint)
		require.True(t, *tool.Tool.Annotations.ReadOnlyHint)
		require.NotNil(t, tool.Tool.Annotations.DestructiveHint)
		require.False(t, *tool.Tool.Annotations.DestructiveHint)
		require.Len(t, tool.TargetCompatibilityFilters, 1)
		require.True(t, tool.TargetCompatibilityFilters[0]())
	}
}

func TestToolsetFiltersOutWithoutOLMAPI(t *testing.T) {
	tools := (&Toolset{}).GetTools(filteringProvider{enabled: true, has: map[schema.GroupVersionKind]bool{}})
	for _, tool := range tools {
		require.False(t, tool.TargetCompatibilityFilters[0](), tool.Tool.Name)
	}
}

func TestToolsetFiltersDisabledRemainVisible(t *testing.T) {
	tools := (&Toolset{}).GetTools(filteringProvider{enabled: false, has: map[schema.GroupVersionKind]bool{}})
	for _, tool := range tools {
		require.True(t, tool.TargetCompatibilityFilters[0](), tool.Tool.Name)
	}
}

func TestSummarizePreservesStatusAndKnownFields(t *testing.T) {
	obj := unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": "example", "namespace": "operators"},
		"spec":     map[string]any{"package": "example-operator"},
		"status": map[string]any{
			"phase":      "Succeeded",
			"currentCSV": "example.v1.2.3",
			"conditions": []any{map[string]any{"type": "Healthy", "status": "True"}},
		},
	}}

	got := summarize(obj, "v0", "subscriptions")
	require.Equal(t, "example", got["name"])
	require.Equal(t, "operators", got["namespace"])
	require.Equal(t, "example-operator", got["spec.package"])
	require.Equal(t, "example.v1.2.3", got["status.currentCSV"])
	require.Equal(t, "Succeeded", got["status.phase"])
	require.NotNil(t, got["status"])
	require.NotNil(t, got["conditions"])
}

func TestParseSelectionRejectsInvalidVersion(t *testing.T) {
	params := api.ToolHandlerParams{ToolCallRequest: &fakeRequest{arguments: map[string]any{"version": "v2"}}}
	_, _, _, err := parseSelection(params)
	require.EqualError(t, err, "version must be one of auto, v0, or v1")
}

func TestListHandlerReadsBothAPIGenerations(t *testing.T) {
	scheme := runtime.NewScheme()
	listKinds := map[schema.GroupVersionResource]string{
		subscriptionGVR: "SubscriptionList", csvGVR: "ClusterServiceVersionList", installPlanGVR: "InstallPlanList",
		clusterExtensionGVR: "ClusterExtensionList", catalogSourceGVR: "CatalogSourceList", clusterCatalogGVR: "ClusterCatalogList",
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, listKinds, object("Subscription", "operators.coreos.com/v1alpha1", "legacy", "operators"), object("ClusterExtension", "olm.operatorframework.io/v1", "modern", "modern"))
	params := api.ToolHandlerParams{Context: context.Background(), KubernetesClient: fakeKubernetesClient{dynamic: dynamicClient}, ToolCallRequest: &fakeRequest{arguments: map[string]any{"version": "auto"}}}

	result, err := listHandler(params)
	require.NoError(t, err)
	require.Contains(t, result.Content, "legacy")
	require.Contains(t, result.Content, "modern")
}

func object(kind, apiVersion, name, namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": apiVersion,
		"kind":       kind,
		"metadata": map[string]any{
			"name":      name,
			"namespace": namespace,
		},
	}}
}

type fakeKubernetesClient struct {
	api.KubernetesClient
	dynamic dynamic.Interface
}

func (c fakeKubernetesClient) DynamicClient() dynamic.Interface { return c.dynamic }

type fakeRequest struct{ arguments map[string]any }

func (r *fakeRequest) GetArguments() map[string]any { return r.arguments }
