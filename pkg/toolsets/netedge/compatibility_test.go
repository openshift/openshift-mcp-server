package netedge

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
)

type recordingFilteringProvider struct {
	result bool
	calls  [][]schema.GroupVersionKind
}

func (p *recordingFilteringProvider) IsTargetCompatibilityToolFiltersEnabled() bool { return true }

func (p *recordingFilteringProvider) AnyTargetHasGVKs(_ context.Context, gvks []schema.GroupVersionKind) bool {
	p.calls = append(p.calls, append([]schema.GroupVersionKind(nil), gvks...))
	return p.result
}

func (p *recordingFilteringProvider) reset() {
	p.calls = nil
}

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
			tools := (&Toolset{}).GetTools(provider)
			s.Require().Len(tools, 10)

			byName := indexToolsByName(tools)
			s.Require().Len(byName, 10, "tool names must be unique")
			for name, gvk := range expected {
				tool, found := byName[name]
				s.Require().True(found, "expected tool %s", name)
				s.Require().Len(tool.TargetCompatibilityFilters, 1, "tool %s must have one exact GVK filter", name)

				provider.reset()
				s.Equal(result, tool.TargetCompatibilityFilters[0](), "tool %s must propagate the provider result", name)
				s.Require().Len(provider.calls, 1, "tool %s must make one provider call", name)
				s.Equal([]schema.GroupVersionKind{gvk}, provider.calls[0], "tool %s must request only its exact GVK", name)
			}
		})
	}
}

func (s *CompatibilityTestSuite) TestLocalProbesAreUnfiltered() {
	provider := &recordingFilteringProvider{result: true}
	tools := indexToolsByName((&Toolset{}).GetTools(provider))

	for _, name := range []string{"probe_dns_local", "probe_http"} {
		tool, found := tools[name]
		s.Require().True(found, "expected tool %s", name)
		s.Empty(tool.TargetCompatibilityFilters, "local probe %s must remain unfiltered", name)
	}
	s.Empty(provider.calls, "constructing tools must not evaluate compatibility filters")
}

func (s *CompatibilityTestSuite) TestInitQueryPrometheusUsesSuppliedProvider() {
	provider := &recordingFilteringProvider{result: true}
	tools := InitQueryPrometheus(provider)

	s.Require().Len(tools, 1)
	s.Require().Len(tools[0].TargetCompatibilityFilters, 1)
	s.True(tools[0].TargetCompatibilityFilters[0]())
	s.Equal([][]schema.GroupVersionKind{{{Group: "route.openshift.io", Version: "v1", Kind: "Route"}}}, provider.calls)
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
