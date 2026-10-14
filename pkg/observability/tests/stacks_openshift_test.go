//go:build observability_e2e

package observability

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// StacksOpenShiftE2ESuite covers OpenShift-specific Tempo / Loki / otelcol cases.
// Requires stacks from: make setup-observability-e2e
// Individual tests skip when the corresponding CRD / instances are missing.
type StacksOpenShiftE2ESuite struct {
	suite.Suite
}

func TestStacksOpenShiftE2E(t *testing.T) {
	suite.Run(t, new(StacksOpenShiftE2ESuite))
}

func (s *StacksOpenShiftE2ESuite) useRouteTOML(extraToolsets ...string) string {
	toolsets := `["observability/otelcol"`
	for _, ts := range extraToolsets {
		toolsets += `, "` + ts + `"`
	}
	toolsets += `]`
	// kubeconfig auth for both K8s CR discovery and Route HTTP.
	// Do not use PROMETHEUS_K8S_TOKEN here: that SA can hit monitoring Routes
	// only and cannot list TempoStack/LokiStack. Callers must not pass
	// mcpAuthOptions() into startServerFromTOML for these tests.
	return `
		toolsets = ` + toolsets + `
		experimental_enable_target_compatibility_tool_filters = true

		[toolset_configs."observability/logs"]
		auth_mode = "kubeconfig"
		use_route = true

		[toolset_configs."observability/traces"]
		auth_mode = "kubeconfig"
		use_route = true
	`
}

var (
	tempoStackGVR = schema.GroupVersionResource{
		Group: "tempo.grafana.com", Version: "v1alpha1", Resource: "tempostacks",
	}
	lokiStackGVR = schema.GroupVersionResource{
		Group: "loki.grafana.com", Version: "v1", Resource: "lokistacks",
	}
)

func (s *StacksOpenShiftE2ESuite) requireTempoStacks() {
	s.T().Helper()
	if !crdExists(s.T(), "tempostacks.tempo.grafana.com") {
		s.T().Skip("TempoStack CRD not installed; run make setup-observability-e2e")
	}
	for _, name := range []string{"tempo1", "tempo2", "multitenant"} {
		if !namedResourceExists(s.T(), tempoStackGVR, "tracing", name) {
			s.T().Skipf("TempoStack tracing/%s not found; run make setup-observability-e2e", name)
		}
	}
}

func (s *StacksOpenShiftE2ESuite) requireLokiStack() {
	s.T().Helper()
	if !crdExists(s.T(), "lokistacks.loki.grafana.com") {
		s.T().Skip("LokiStack CRD not installed; run make setup-observability-e2e")
	}
	if !namedResourceExists(s.T(), lokiStackGVR, "observability-loki", "observability-loki") {
		s.T().Skip("LokiStack observability-loki/observability-loki not found; run make setup-observability-e2e")
	}
}

func (s *StacksOpenShiftE2ESuite) TestTempoListInstances() {
	s.requireTempoStacks()
	_, client := startServerFromTOML(s.T(), s.useRouteTOML("observability/traces"))

	toolResult, err := client.CallTool("tempo_list_instances", map[string]any{})
	s.Require().NoError(err)
	s.Require().False(toolResult.IsError, "tool error: %s", textContent(toolResult))

	structured := structuredMap(s.T(), toolResult.StructuredContent)
	instances, ok := structured["instances"].([]any)
	s.Require().True(ok, "expected instances array")
	require.ElementsMatch(s.T(), []any{
		map[string]any{"kind": "TempoStack", "tempoNamespace": "tracing", "tempoName": "tempo1", "multitenancy": false, "status": "Ready"},
		map[string]any{"kind": "TempoStack", "tempoNamespace": "tracing", "tempoName": "tempo2", "multitenancy": false, "status": "Ready"},
		map[string]any{"kind": "TempoStack", "tempoNamespace": "tracing", "tempoName": "multitenant", "multitenancy": true, "status": "Ready", "tenants": []any{"project1", "project2"}},
	}, instances)
}

func (s *StacksOpenShiftE2ESuite) TestLokiListInstances_RouteURLs() {
	s.requireLokiStack()
	_, client := startServerFromTOML(s.T(), s.useRouteTOML("observability/logs"))

	toolResult, err := client.CallTool("loki_list_instances", map[string]any{})
	s.Require().NoError(err)
	s.Require().False(toolResult.IsError, "tool error: %s", textContent(toolResult))

	structured := structuredMap(s.T(), toolResult.StructuredContent)
	instances, ok := structured["instances"].([]any)
	s.Require().True(ok && len(instances) > 0, "expected at least one LokiStack instance")

	for i, raw := range instances {
		inst, ok := raw.(map[string]any)
		s.Require().True(ok, "instance %d: expected map", i)
		urlStr, _ := inst["url"].(string)
		s.Require().NotEmpty(urlStr, "instance %d missing url", i)
		assertValidRouteURL(s.T(), urlStr)
		s.T().Logf("loki instance %v/%v url=%s", inst["lokiNamespace"], inst["lokiName"], urlStr)
	}
}

func (s *StacksOpenShiftE2ESuite) TestTempoSearchTraces_Multitenant() {
	s.requireTempoStacks()
	_, client := startServerFromTOML(s.T(), s.useRouteTOML("observability/traces"))

	deadline := time.Now().Add(60 * time.Second)
	for {
		toolResult, err := client.CallTool("tempo_search_traces", map[string]any{
			"tempoNamespace": "tracing",
			"tempoName":      "multitenant",
			"tenant":         "project1",
			"query":          "{}",
		})
		s.Require().NoError(err)
		if toolResult.IsError {
			body := textContent(toolResult)
			if time.Now().Before(deadline) && (strings.Contains(body, "connection refused") ||
				strings.Contains(body, "i/o timeout") ||
				strings.Contains(body, "connection reset")) {
				time.Sleep(5 * time.Second)
				continue
			}
			s.T().Fatalf("tempo_search_traces error: %s", body)
		}
		structured := structuredMap(s.T(), toolResult.StructuredContent)
		traces, ok := structured["traces"].([]any)
		s.Require().True(ok, "expected traces field")
		if len(traces) == 0 {
			if time.Now().Before(deadline) {
				time.Sleep(5 * time.Second)
				continue
			}
			s.T().Fatal("expected at least one trace within 60s; was data ingested?")
		}
		s.T().Logf("tempo_search_traces (multitenant) returned %d traces", len(traces))
		return
	}
}

func (s *StacksOpenShiftE2ESuite) TestOtelcolToolset() {
	// Schema/version helpers do not need Route discovery or CRD gating.
	// Avoid target-compatibility filters here: with a single-cluster provider
	// those filters hide otelcol tools when the OpenTelemetryCollector CRD
	// is absent (previously masked by multi-context discovery fail-open).
	_, client := startServerFromTOML(s.T(), `
		toolsets = ["observability/otelcol"]
	`)

	toolResult, err := client.CallTool("otelcol_get_versions", map[string]any{})
	s.Require().NoError(err)
	s.Require().False(toolResult.IsError, "tool error: %s", textContent(toolResult))

	structured := structuredMap(s.T(), toolResult.StructuredContent)
	s.NotEmpty(structured["versions"])
	s.NotEmpty(structured["latest_version"])
}
