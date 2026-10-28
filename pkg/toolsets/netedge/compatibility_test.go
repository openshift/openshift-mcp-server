package netedge

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
)

type recordingFilteringProvider struct {
	api.ClusterInspector
	api.TargetProvider
	result   bool
	kind     string
	calls    []string
	contexts []context.Context
}

func (provider *recordingFilteringProvider) Discovery() api.AggregateDiscovery { return provider }
func (provider *recordingFilteringProvider) GetTargets(context.Context) ([]string, error) {
	return []string{"fixture"}, nil
}
func (provider *recordingFilteringProvider) ServerResourcesForGroupVersion(ctx context.Context, groupVersion string) api.Results[*metav1.APIResourceList] {
	provider.calls = append(provider.calls, groupVersion)
	provider.contexts = append(provider.contexts, ctx)
	return api.NewResults(ctx, provider, func(context.Context, string) (*metav1.APIResourceList, error) {
		kinds := []metav1.APIResource{{Kind: "DifferentKind"}}
		if provider.result {
			kinds = []metav1.APIResource{{Kind: provider.kind}}
		}
		return &metav1.APIResourceList{GroupVersion: groupVersion, APIResources: kinds}, nil
	})
}
func (provider *recordingFilteringProvider) reset() { provider.calls = nil; provider.contexts = nil }

type CompatibilityTestSuite struct {
	suite.Suite
}

func (s *CompatibilityTestSuite) TestToolsetRegistersExactCompatibilityRequirements() {
	routeGVK := schema.GroupVersionKind{Group: "route.openshift.io", Version: "v1", Kind: "Route"}
	configMapGVK := schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"}
	endpointSliceGVK := schema.GroupVersionKind{Group: "discovery.k8s.io", Version: "v1", Kind: "EndpointSlice"}
	podGVK := schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}
	expected := map[string]schema.GroupVersionKind{
		(&Toolset{}).GetName() + "_query_prometheus": routeGVK,
		"inspect_route":         routeGVK,
		"get_coredns_config":    configMapGVK,
		"get_service_endpoints": endpointSliceGVK,
		"exec_dns_in_pod":       podGVK,
		"get_router_config":     podGVK,
		"get_router_info":       podGVK,
		"get_router_sessions":   podGVK,
	}

	for _, result := range []bool{true, false} {
		s.Run(map[bool]string{true: "available", false: "unavailable"}[result], func() {
			provider := &recordingFilteringProvider{result: result}
			tools := (&Toolset{}).GetTools(s.T().Context(), api.ToolsetContext{Inspector: provider})
			s.Require().Len(tools, 10)

			byName := indexToolsByName(tools)
			s.Require().Len(byName, 10, "tool names must be unique")
			for name, gvk := range expected {
				tool, found := byName[name]
				s.Require().True(found, "expected tool %s", name)
				s.Require().Len(tool.TargetCompatibilityFilters, 1, "tool %s must have one exact GVK filter", name)

				provider.reset()
				provider.kind = gvk.Kind
				s.Equal(result, tool.TargetCompatibilityFilters[0](), "tool %s must propagate the provider result", name)
				s.Require().Len(provider.calls, 1, "tool %s must make one provider call", name)
				s.Equal(gvk.GroupVersion().String(), provider.calls[0], "tool %s must request only its exact GVK", name)
			}
		})
	}
}

func (s *CompatibilityTestSuite) TestLocalProbesAreUnfiltered() {
	provider := &recordingFilteringProvider{result: true}
	tools := indexToolsByName((&Toolset{}).GetTools(s.T().Context(), api.ToolsetContext{Inspector: provider}))

	for _, name := range []string{"probe_dns_local", "probe_http"} {
		tool, found := tools[name]
		s.Require().True(found, "expected tool %s", name)
		s.Empty(tool.TargetCompatibilityFilters, "local probe %s must remain unfiltered", name)
	}
	s.Empty(provider.calls, "constructing tools must not evaluate compatibility filters")
}

func (s *CompatibilityTestSuite) TestInitQueryPrometheusUsesSuppliedProvider() {
	provider := &recordingFilteringProvider{result: true, kind: "Route"}
	ctx, cancel := context.WithCancel(s.T().Context())
	defer cancel()
	tools := InitQueryPrometheus(ctx, provider)
	s.Empty(provider.calls)

	s.Require().Len(tools, 1)
	s.Require().Len(tools[0].TargetCompatibilityFilters, 1)
	s.True(tools[0].TargetCompatibilityFilters[0]())
	s.Equal([]string{"route.openshift.io/v1"}, provider.calls)
	s.Same(ctx, provider.contexts[0])
}

func (s *CompatibilityTestSuite) TestDirectConstructorsUseExactKindsAndContext() {
	contracts := []struct {
		name         string
		groupVersion string
		kind         string
		construct    func(context.Context, api.ClusterInspector) []api.ServerTool
	}{
		{(&Toolset{}).GetName() + "_query_prometheus", "route.openshift.io/v1", "Route", InitQueryPrometheus},
		{"inspect_route", "route.openshift.io/v1", "Route", initRoutes},
		{"get_coredns_config", "v1", "ConfigMap", initCoreDNS},
		{"get_service_endpoints", "discovery.k8s.io/v1", "EndpointSlice", initEndpoints},
		{"exec_dns_in_pod", "v1", "Pod", initExecDNSInPod},
		{"get_router_config", "v1", "Pod", initRouter},
		{"get_router_info", "v1", "Pod", initRouter},
		{"get_router_sessions", "v1", "Pod", initRouter},
		{"exec_dns_in_pod", "v1", "Pod", func(ctx context.Context, inspector api.ClusterInspector) []api.ServerTool {
			return initExecDNSInPodWith(ctx, inspector, nil)
		}},
	}
	for _, contract := range contracts {
		s.Run(contract.name, func() {
			for _, kind := range []string{contract.kind, "DifferentKind"} {
				inspector := &recordingFilteringProvider{result: true, kind: kind}
				ctx, cancel := context.WithCancel(s.T().Context())
				defer cancel()
				tools := contract.construct(ctx, inspector)
				s.Empty(inspector.calls)
				tool, found := indexToolsByName(tools)[contract.name]
				s.Require().True(found)
				s.Require().Len(tool.TargetCompatibilityFilters, 1)
				s.Equal(kind == contract.kind, tool.TargetCompatibilityFilters[0]())
				s.Equal([]string{contract.groupVersion}, inspector.calls)
				s.Same(ctx, inspector.contexts[0])
			}
		})
	}
}

func indexToolsByName(tools []api.ServerTool) map[string]api.ServerTool {
	byName := make(map[string]api.ServerTool, len(tools))
	for _, tool := range tools {
		byName[tool.Tool.Name] = tool
	}
	return byName
}

func TestCompatibility(t *testing.T) {
	suite.Run(t, new(CompatibilityTestSuite))
}
