//go:build observability_e2e

package observability

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/containers/kubernetes-mcp-server/internal/test"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// StacksOpenShiftE2ESuite covers Tempo / Loki / otelcol against a live OpenShift cluster.
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
	if os.Getenv("SKIP_OBS_TEMPO") != "" {
		s.T().Skip("SKIP_OBS_TEMPO set; Tempo setup failed")
	}
	prepareObsE2EKubeconfig(s.T())
	if !crdExists(s.T(), "tempostacks.tempo.grafana.com") {
		s.T().Skip("TempoStack CRD not installed; run make setup-observability-e2e")
	}
	for _, name := range []string{"tempo1", "tempo2", "multitenant"} {
		if !namedResourceExists(s.T(), tempoStackGVR, "tracing", name) {
			s.T().Skipf("TempoStack tracing/%s not found; run make setup-observability-e2e", name)
		}
		if !namedResourceReady(s.T(), tempoStackGVR, "tracing", name) {
			s.T().Skipf("TempoStack tracing/%s exists but is not Ready; skipping (partial install)", name)
		}
	}
}

func (s *StacksOpenShiftE2ESuite) requireLokiStack() {
	s.T().Helper()
	if os.Getenv("SKIP_OBS_LOKI") != "" {
		s.T().Skip("SKIP_OBS_LOKI set; Loki setup failed")
	}
	prepareObsE2EKubeconfig(s.T())
	if !crdExists(s.T(), "lokistacks.loki.grafana.com") {
		s.T().Skip("LokiStack CRD not installed; run make setup-observability-e2e")
	}
	if !namedResourceExists(s.T(), lokiStackGVR, "observability-loki", "observability-loki") {
		s.T().Skip("LokiStack observability-loki/observability-loki not found; run make setup-observability-e2e")
	}
	if !namedResourceReady(s.T(), lokiStackGVR, "observability-loki", "observability-loki") {
		s.T().Skip("LokiStack observability-loki/observability-loki exists but is not Ready; skipping (partial install)")
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

func lokiBaseArgs() map[string]any {
	return map[string]any{
		"lokiNamespace": "observability-loki",
		"lokiName":      "observability-loki",
		"tenant":        "network",
	}
}

func mergeArgs(base, extra map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func (s *StacksOpenShiftE2ESuite) TestLokiLabelNames() {
	s.requireLokiStack()
	_, client := startServerFromTOML(s.T(), s.useRouteTOML("observability/logs"))

	toolResult := callUntilReady(s.T(), client, "loki_label_names", lokiBaseArgs(), func(r *mcpsdk.CallToolResult) bool {
		if r.IsError {
			return false
		}
		structured := structuredMap(s.T(), r.StructuredContent)
		labels, ok := structured["labels"].([]any)
		return ok && len(labels) > 0
	})

	structured := structuredMap(s.T(), toolResult.StructuredContent)
	labels := anyStrings(structured["labels"].([]any))
	s.Contains(labels, "app")
	s.Contains(labels, "level")
	s.T().Logf("loki_label_names returned %d labels", len(labels))
}

func (s *StacksOpenShiftE2ESuite) TestLokiLabelValues() {
	s.requireLokiStack()
	_, client := startServerFromTOML(s.T(), s.useRouteTOML("observability/logs"))

	toolResult := callUntilReady(s.T(), client, "loki_label_values", mergeArgs(lokiBaseArgs(), map[string]any{
		"label": "app",
	}), func(r *mcpsdk.CallToolResult) bool {
		if r.IsError {
			return false
		}
		structured := structuredMap(s.T(), r.StructuredContent)
		values, ok := structured["values"].([]any)
		return ok && len(values) > 0
	})

	structured := structuredMap(s.T(), toolResult.StructuredContent)
	values := anyStrings(structured["values"].([]any))
	s.Contains(values, "observability-log-generator")
}

func (s *StacksOpenShiftE2ESuite) TestLokiQueryRange() {
	s.requireLokiStack()
	_, client := startServerFromTOML(s.T(), s.useRouteTOML("observability/logs"))

	toolResult := callUntilReady(s.T(), client, "loki_query_range", mergeArgs(lokiBaseArgs(), map[string]any{
		"query":    `{app="observability-log-generator"} |= "observability-loki-hack"`,
		"duration": "15m",
		"limit":    5,
	}), func(r *mcpsdk.CallToolResult) bool {
		if r.IsError {
			return false
		}
		structured := structuredMap(s.T(), r.StructuredContent)
		streams, ok := structured["streams"].([]any)
		return ok && len(streams) > 0
	})

	structured := structuredMap(s.T(), toolResult.StructuredContent)
	s.Equal("streams", structured["resultType"])
	streams := structured["streams"].([]any)
	stream := streams[0].(map[string]any)
	labels, ok := stream["labels"].(map[string]any)
	s.Require().True(ok)
	s.Equal("observability-log-generator", labels["app"])

	entries, ok := stream["entries"].([]any)
	s.Require().True(ok && len(entries) > 0)
	line, _ := entries[0].(map[string]any)["line"].(string)
	s.Contains(line, "observability-loki-hack")
}

func (s *StacksOpenShiftE2ESuite) TestTempoSearchTraces_Multitenant() {
	s.requireTempoStacks()
	_, client := startServerFromTOML(s.T(), s.useRouteTOML("observability/traces"))

	toolResult := callUntilReady(s.T(), client, "tempo_search_traces", map[string]any{
		"tempoNamespace": "tracing",
		"tempoName":      "multitenant",
		"tenant":         "project1",
		"query":          "{}",
	}, func(r *mcpsdk.CallToolResult) bool {
		structured := structuredMap(s.T(), r.StructuredContent)
		traces, ok := structured["traces"].([]any)
		return ok && len(traces) > 0
	})

	structured := structuredMap(s.T(), toolResult.StructuredContent)
	traces := structured["traces"].([]any)
	s.T().Logf("tempo_search_traces (multitenant) returned %d traces", len(traces))
}

func (s *StacksOpenShiftE2ESuite) TestTempoSearchTraces_SingleTenant() {
	s.requireTempoStacks()
	_, client := startServerFromTOML(s.T(), s.useRouteTOML("observability/traces"))

	toolResult := callUntilReady(s.T(), client, "tempo_search_traces", map[string]any{
		"tempoNamespace": "tracing",
		"tempoName":      "tempo1",
		"query":          "{}",
	}, func(r *mcpsdk.CallToolResult) bool {
		structured := structuredMap(s.T(), r.StructuredContent)
		traces, ok := structured["traces"].([]any)
		return ok && len(traces) > 0
	})

	structured := structuredMap(s.T(), toolResult.StructuredContent)
	traces := structured["traces"].([]any)
	s.T().Logf("tempo_search_traces (tempo1) returned %d traces", len(traces))
}

func (s *StacksOpenShiftE2ESuite) TestTempoSearchTags() {
	s.requireTempoStacks()
	_, client := startServerFromTOML(s.T(), s.useRouteTOML("observability/traces"))

	toolResult := callUntilReady(s.T(), client, "tempo_search_tags", map[string]any{
		"tempoNamespace": "tracing",
		"tempoName":      "tempo1",
	}, func(r *mcpsdk.CallToolResult) bool {
		structured := structuredMap(s.T(), r.StructuredContent)
		scopes, ok := structured["scopes"].([]any)
		return ok && len(scopes) > 0
	})

	structured := structuredMap(s.T(), toolResult.StructuredContent)
	s.NotEmpty(structured["scopes"])
	s.T().Logf("tempo_search_tags returned scopes")
}

func (s *StacksOpenShiftE2ESuite) TestTempoSearchTagValues() {
	s.requireTempoStacks()
	_, client := startServerFromTOML(s.T(), s.useRouteTOML("observability/traces"))

	toolResult := callUntilReady(s.T(), client, "tempo_search_tag_values", map[string]any{
		"tempoNamespace": "tracing",
		"tempoName":      "tempo1",
		"tag":            "resource.service.name",
	}, func(r *mcpsdk.CallToolResult) bool {
		structured := structuredMap(s.T(), r.StructuredContent)
		return structured["tagValues"] != nil
	})

	structured := structuredMap(s.T(), toolResult.StructuredContent)
	s.NotNil(structured["tagValues"])
	s.T().Logf("tempo_search_tag_values returned values for resource.service.name")
}

func (s *StacksOpenShiftE2ESuite) TestTempoGetTraceByID() {
	s.requireTempoStacks()
	_, client := startServerFromTOML(s.T(), s.useRouteTOML("observability/traces"))

	searchResult := callUntilReady(s.T(), client, "tempo_search_traces", map[string]any{
		"tempoNamespace": "tracing",
		"tempoName":      "tempo1",
		"query":          "{}",
		"limit":          1,
	}, func(r *mcpsdk.CallToolResult) bool {
		structured := structuredMap(s.T(), r.StructuredContent)
		traces, ok := structured["traces"].([]any)
		return ok && len(traces) > 0
	})

	traceID := firstTraceID(s.T(), searchResult.StructuredContent)
	s.Require().NotEmpty(traceID, "expected traceID from search")

	toolResult, err := client.CallTool("tempo_get_trace_by_id", map[string]any{
		"tempoNamespace": "tracing",
		"tempoName":      "tempo1",
		"traceid":        traceID,
	})
	s.Require().NoError(err)
	s.Require().False(toolResult.IsError, "tool error: %s", textContent(toolResult))

	structured := structuredMap(s.T(), toolResult.StructuredContent)
	s.NotNil(structured["trace"])
	s.T().Logf("tempo_get_trace_by_id returned trace %s", traceID)
}

func (s *StacksOpenShiftE2ESuite) TestOtelcolToolset() {
	// Schema/version helpers do not need Route discovery or CRD gating.
	// Avoid target-compatibility filters here: with a single-cluster provider
	// those filters hide otelcol tools when the OpenTelemetryCollector CRD
	// is absent (previously masked by multi-context discovery fail-open).
	_, client := startServerFromTOML(s.T(), `
		toolsets = ["observability/otelcol"]
	`)

	versionsResult, err := client.CallTool("otelcol_get_versions", map[string]any{})
	s.Require().NoError(err)
	s.Require().False(versionsResult.IsError, "tool error: %s", textContent(versionsResult))
	versions := structuredMap(s.T(), versionsResult.StructuredContent)
	s.NotEmpty(versions["versions"])
	s.NotEmpty(versions["latest_version"])

	listResult, err := client.CallTool("otelcol_list_components", map[string]any{})
	s.Require().NoError(err)
	s.Require().False(listResult.IsError, "tool error: %s", textContent(listResult))
	components := structuredMap(s.T(), listResult.StructuredContent)
	receivers := anyStrings(mustSlice(s.T(), components["receivers"]))
	processors := anyStrings(mustSlice(s.T(), components["processors"]))
	s.Contains(receivers, "otlp")
	s.Contains(processors, "batch")

	schemaResult, err := client.CallTool("otelcol_get_component_schema", map[string]any{
		"component_type": "processor",
		"component_name": "batch",
	})
	s.Require().NoError(err)
	s.Require().False(schemaResult.IsError, "tool error: %s", textContent(schemaResult))
	schemaOut := structuredMap(s.T(), schemaResult.StructuredContent)
	s.Equal("batch", schemaOut["name"])
	schema, ok := schemaOut["schema"].(map[string]any)
	s.Require().True(ok && len(schema) > 0, "expected non-empty schema")

	validResult, err := client.CallTool("otelcol_validate_config", map[string]any{
		"component_type": "processor",
		"component_name": "batch",
		"config":         "send_batch_size: 8192\ntimeout: 200ms",
		"format":         "yaml",
	})
	s.Require().NoError(err)
	s.Require().False(validResult.IsError, "tool error: %s", textContent(validResult))
	validOut := structuredMap(s.T(), validResult.StructuredContent)
	s.Equal(true, validOut["valid"])

	invalidResult, err := client.CallTool("otelcol_validate_config", map[string]any{
		"component_type": "processor",
		"component_name": "batch",
		"config":         "does_not_exist: 200ms",
		"format":         "yaml",
	})
	s.Require().NoError(err)
	s.Require().False(invalidResult.IsError, "tool error: %s", textContent(invalidResult))
	invalidOut := structuredMap(s.T(), invalidResult.StructuredContent)
	s.Equal(false, invalidOut["valid"])
}

// callUntilReady retries a tool call until ready or the deadline expires.
func callUntilReady(t *testing.T, client *test.McpClient, tool string, args map[string]any, ready func(*mcpsdk.CallToolResult) bool) *mcpsdk.CallToolResult {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for {
		result, err := client.CallTool(tool, args)
		require.NoError(t, err, "CallTool %s", tool)
		if result.IsError {
			body := textContent(result)
			if isOutOfClusterBackendError(body) {
				t.Skipf("%s unreachable out-of-cluster (need Route): %s", tool, body)
			}
			if !isRetryableToolError(result) {
				t.Fatalf("%s error: %s", tool, body)
			}
		} else if ready(result) {
			return result
		}
		if time.Now().After(deadline) {
			body := textContent(result)
			if result.IsError {
				t.Fatalf("%s still failing after retries: %s", tool, body)
			}
			t.Fatalf("%s did not become ready within 60s; last result: %s", tool, body)
		}
		time.Sleep(5 * time.Second)
	}
}

func isRetryableToolError(result *mcpsdk.CallToolResult) bool {
	body := textContent(result)
	return strings.Contains(body, "connection refused") ||
		strings.Contains(body, "i/o timeout") ||
		strings.Contains(body, "connection reset") ||
		strings.Contains(body, "no such host") ||
		strings.Contains(body, "temporary failure in name resolution") ||
		strings.Contains(body, "TLS handshake timeout") ||
		strings.Contains(body, "EOF")
}

// isOutOfClusterBackendError detects backends that only expose in-cluster .svc DNS.
// Auth failures are not skipped: prepareObsE2EKubeconfig must supply a usable bearer.
func isOutOfClusterBackendError(body string) bool {
	return strings.Contains(body, ".svc") && (strings.Contains(body, "no such host") ||
		strings.Contains(body, "lookup ") ||
		strings.Contains(body, "i/o timeout") ||
		strings.Contains(body, "connection refused"))
}

func anyStrings(in []any) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func mustSlice(t *testing.T, v any) []any {
	t.Helper()
	s, ok := v.([]any)
	require.True(t, ok, "expected []any, got %T", v)
	return s
}

func firstTraceID(t *testing.T, structuredContent any) string {
	t.Helper()
	structured := structuredMap(t, structuredContent)
	traces, ok := structured["traces"].([]any)
	require.True(t, ok && len(traces) > 0, "expected traces")
	first, ok := traces[0].(map[string]any)
	require.True(t, ok, "expected trace map")
	for _, key := range []string{"traceID", "traceId", "trace_id"} {
		if id, ok := first[key].(string); ok && id != "" {
			return id
		}
	}
	t.Fatalf("no traceID in search hit: %#v", first)
	return ""
}
