//go:build observability_e2e

package observability

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/containers/kubernetes-mcp-server/internal/test"
	"github.com/containers/kubernetes-mcp-server/pkg/config"
	internalk8s "github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	mcpserver "github.com/containers/kubernetes-mcp-server/pkg/mcp"
	"github.com/containers/kubernetes-mcp-server/pkg/obs/openshift"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
)

// MetricsE2ESuite tests the observability/metrics toolset against a real
// OpenShift cluster where Prometheus and Alertmanager are available
// in the openshift-monitoring namespace.
//
// URLs are auto-discovered from OpenShift routes in the openshift-monitoring
// namespace. Override with PROMETHEUS_URL / ALERTMANAGER_URL env vars if needed.
// Run with:
//
//	make test-observability-e2e
//
// Or directly:
//
//	go test -tags observability_e2e ./pkg/observability/tests/ -v
//
// Optional environment variables:
//
//	PROMETHEUS_URL   - Override auto-discovered Prometheus/Thanos Querier URL
//	ALERTMANAGER_URL - Override auto-discovered Alertmanager URL
//	KUBECONFIG       - Path to kubeconfig file (defaults to ~/.kube/config)
//	PROMETHEUS_K8S_TOKEN - Bearer for monitoring Route HTTP checks (CI)
type MetricsE2ESuite struct {
	suite.Suite
	*test.McpClient
	server          *mcpserver.Server
	prometheusURL   string
	alertmanagerURL string
	testMetric      string
}

func (s *MetricsE2ESuite) SetupSuite() {
	ensurePrometheusK8sToken(s.T())
	s.prometheusURL = os.Getenv("PROMETHEUS_URL")
	s.alertmanagerURL = os.Getenv("ALERTMANAGER_URL")

	if s.prometheusURL == "" || s.alertmanagerURL == "" {
		discovered := s.discoverRoutesViaResolver()
		if s.prometheusURL == "" {
			s.prometheusURL = discovered["thanos"]
		}
		if s.alertmanagerURL == "" {
			s.alertmanagerURL = discovered["alertmanager"]
		}
	}

	if s.prometheusURL == "" {
		s.T().Fatal("Could not determine Prometheus URL: set PROMETHEUS_URL or ensure the thanos-querier route exists in openshift-monitoring")
	}
	s.T().Logf("Prometheus URL: %s", s.prometheusURL)
	s.T().Logf("Alertmanager URL: %s", s.alertmanagerURL)

	// Discover a real metric from the cluster for use in query tests.
	// This also warms up the Thanos Querier label cache so that
	// ValidateMetricsExist (which lists all metric names) succeeds
	// reliably in subsequent per-test calls.
	s.testMetric = s.discoverTestMetric()
	s.Require().NotEmpty(s.testMetric, "could not discover any metric from Prometheus")
	s.T().Logf("Test metric: %s", s.testMetric)
}

// discoverTestMetric creates a temporary MCP server to call list_metrics
// and returns the first CPU metric found. This also pre-warms the Thanos
// Querier's metric name cache.
func (s *MetricsE2ESuite) discoverTestMetric() string {
	cfg := s.buildConfig()

	provider, err := internalk8s.NewProvider(s.T().Context(), cfg)
	if err != nil {
		s.T().Logf("Could not create provider for metric discovery: %v", err)
		return ""
	}

	server, err := mcpserver.NewServer(s.T().Context(), mcpserver.Configuration{Config: cfg}, provider)
	if err != nil {
		s.T().Logf("Could not create server for metric discovery: %v", err)
		return ""
	}
	defer server.Close()

	client := test.NewMcpClient(s.T(), server.ServeHTTP(), mcpAuthOptions()...)
	defer client.Close()

	result, err := client.CallTool("list_metrics", map[string]any{"name_regex": ".*cpu.*"})
	if err != nil || result.IsError {
		s.T().Logf("list_metrics failed during discovery: err=%v, isError=%v", err, result != nil && result.IsError)
		return ""
	}

	var output listMetricsOutput
	if err := json.Unmarshal([]byte(textContent(result)), &output); err != nil || len(output.Metrics) == 0 {
		return ""
	}
	return output.Metrics[0]
}

// discoverRoutesViaResolver uses the production MetricsRouteResolver so suite
// setup and TestRouteDiscovery_* exercise the same code path (no duplicate
// hand-rolled Route parsing).
func (s *MetricsE2ESuite) discoverRoutesViaResolver() map[string]string {
	result := make(map[string]string)
	resolver := &openshift.MetricsRouteResolver{}
	dyn := dynamicClient(s.T())

	if url, err := resolver.ResolveMetricsBackend(s.T().Context(), dyn, "thanos"); err != nil {
		s.T().Logf("Could not resolve thanos/prometheus route: %v", err)
	} else {
		result["thanos"] = url
	}
	if url, err := resolver.ResolveAlertmanager(s.T().Context(), dyn); err != nil {
		s.T().Logf("Could not resolve alertmanager route: %v", err)
	} else {
		result["alertmanager"] = url
	}
	return result
}

func (s *MetricsE2ESuite) buildConfig() *config.Config {
	tomlCfg := fmt.Sprintf(`
		toolsets = ["observability/metrics"]
		[toolset_configs."observability/metrics"]
		auth_mode = "%s"
		prometheus_url = "%s"
		alertmanager_url = "%s"
		insecure = true
		guardrails = "none"
	`, metricsAuthMode(), s.prometheusURL, s.alertmanagerURL)

	cfg, err := config.ReadToml(s.T().Context(), []byte(tomlCfg))
	s.Require().NoError(err, "Failed to parse test config")
	applyE2EDefaults(cfg)
	return cfg
}

func (s *MetricsE2ESuite) SetupTest() {
	cfg := s.buildConfig()

	provider, err := internalk8s.NewProvider(s.T().Context(), cfg)
	s.Require().NoError(err, "Failed to create k8s provider")

	s.server, err = mcpserver.NewServer(s.T().Context(), mcpserver.Configuration{Config: cfg}, provider)
	s.Require().NoError(err, "Failed to create MCP server")

	s.McpClient = test.NewMcpClient(s.T(), s.server.ServeHTTP(), mcpAuthOptions()...)
}

func (s *MetricsE2ESuite) TearDownTest() {
	if s.McpClient != nil {
		s.Close()
	}
	if s.server != nil {
		s.server.Close()
	}
}

func (s *MetricsE2ESuite) TestExecuteInstantQuery() {
	s.Run("executes an instant query", func() {
		toolResult, err := s.CallTool("execute_instant_query", map[string]any{
			"query": s.testMetric,
		})
		s.Require().NoError(err)
		s.Require().False(toolResult.IsError, "tool returned error: %s", textContent(toolResult))

		var output instantQueryOutput
		s.Require().NoError(json.Unmarshal([]byte(textContent(toolResult)), &output))
		s.Equal("vector", output.ResultType)
		s.NotEmpty(output.Result, "expected at least one result")
	})
}

func (s *MetricsE2ESuite) TestExecuteRangeQuery() {
	s.Run("executes a range query with step", func() {
		toolResult, err := s.CallTool("execute_range_query", map[string]any{
			"query":    s.testMetric,
			"step":     "1m",
			"duration": "5m",
		})
		s.Require().NoError(err)
		s.Require().False(toolResult.IsError, "tool returned error: %s", textContent(toolResult))

		var output rangeQueryOutput
		s.Require().NoError(json.Unmarshal([]byte(textContent(toolResult)), &output))
		s.Equal("matrix", output.ResultType)
	})
}

func (s *MetricsE2ESuite) TestShowTimeseries() {
	s.Run("validates a range query for chart rendering", func() {
		toolResult, err := s.CallTool("show_timeseries", map[string]any{
			"query":    s.testMetric,
			"step":     "1m",
			"duration": "5m",
			"title":    "Test Timeseries",
		})
		s.Require().NoError(err)
		s.Require().False(toolResult.IsError, "tool returned error: %s", textContent(toolResult))
	})
}

func (s *MetricsE2ESuite) TestGetLabelNames() {
	s.Run("retrieves label names for a metric", func() {
		toolResult, err := s.CallTool("get_label_names", map[string]any{
			"metric": s.testMetric,
		})
		s.Require().NoError(err)
		s.Require().False(toolResult.IsError, "tool returned error: %s", textContent(toolResult))

		var output labelNamesOutput
		s.Require().NoError(json.Unmarshal([]byte(textContent(toolResult)), &output))
		s.NotEmpty(output.Labels, "expected at least one label name")
	})
}

func (s *MetricsE2ESuite) TestGetLabelValues() {
	s.Run("retrieves label values for __name__ label", func() {
		toolResult, err := s.CallTool("get_label_values", map[string]any{
			"label":  "__name__",
			"metric": s.testMetric,
		})
		s.Require().NoError(err)
		s.Require().False(toolResult.IsError, "tool returned error: %s", textContent(toolResult))

		var output labelValuesOutput
		s.Require().NoError(json.Unmarshal([]byte(textContent(toolResult)), &output))
		s.NotEmpty(output.Values, "expected at least one label value")
	})
}

func (s *MetricsE2ESuite) TestGetSeries() {
	s.Run("retrieves series for a metric", func() {
		toolResult, err := s.CallTool("get_series", map[string]any{
			"matches": fmt.Sprintf(`{__name__="%s"}`, s.testMetric),
		})
		s.Require().NoError(err)
		s.Require().False(toolResult.IsError, "tool returned error: %s", textContent(toolResult))

		var output seriesOutput
		s.Require().NoError(json.Unmarshal([]byte(textContent(toolResult)), &output))
		s.NotEmpty(output.Series, "expected at least one series")
		s.Greater(output.Cardinality, 0)
	})
}

func (s *MetricsE2ESuite) TestGetAlerts() {
	if s.alertmanagerURL == "" {
		s.T().Skip("Alertmanager URL not available, skipping alerts test")
	}

	s.Run("retrieves alerts from Alertmanager", func() {
		toolResult, err := s.CallTool("get_alerts", map[string]any{})
		s.Require().NoError(err)
		s.Require().False(toolResult.IsError, "tool returned error: %s", textContent(toolResult))

		var output alertsOutput
		s.Require().NoError(json.Unmarshal([]byte(textContent(toolResult)), &output))
		s.NotNil(output.Alerts, "expected alerts array")
	})
}

func (s *MetricsE2ESuite) TestGetSilences() {
	if s.alertmanagerURL == "" {
		s.T().Skip("Alertmanager URL not available, skipping silences test")
	}

	s.Run("retrieves silences from Alertmanager", func() {
		toolResult, err := s.CallTool("get_silences", map[string]any{})
		s.Require().NoError(err)
		s.Require().False(toolResult.IsError, "tool returned error: %s", textContent(toolResult))

		var output silencesOutput
		s.Require().NoError(json.Unmarshal([]byte(textContent(toolResult)), &output))
		s.NotNil(output.Silences, "expected silences array")
	})
}

func TestMetricsE2E(t *testing.T) {
	suite.Run(t, new(MetricsE2ESuite))
}

func textContent(result *mcpsdk.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}
	if tc, ok := result.Content[0].(*mcpsdk.TextContent); ok {
		return tc.Text
	}
	return ""
}

type listMetricsOutput struct {
	Metrics []string `json:"metrics"`
}

type instantQueryOutput struct {
	ResultType string `json:"resultType"`
	Result     []any  `json:"result"`
}

type rangeQueryOutput struct {
	ResultType string `json:"resultType"`
	Summary    []any  `json:"summary,omitempty"`
	Result     []any  `json:"result,omitempty"`
}

type labelNamesOutput struct {
	Labels []string `json:"labels"`
}

type labelValuesOutput struct {
	Values []string `json:"values"`
}

type seriesOutput struct {
	Series      []map[string]string `json:"series"`
	Cardinality int                 `json:"cardinality"`
}

type alertsOutput struct {
	Alerts []any `json:"alerts"`
}

type silencesOutput struct {
	Silences []any `json:"silences"`
}
