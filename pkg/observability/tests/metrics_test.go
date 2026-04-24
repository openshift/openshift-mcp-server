//go:build observability_e2e

package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/containers/kubernetes-mcp-server/internal/test"
	"github.com/containers/kubernetes-mcp-server/pkg/config"
	internalk8s "github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	mcpserver "github.com/containers/kubernetes-mcp-server/pkg/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	_ "github.com/rhobs/obs-mcp/pkg/toolset"
)

const (
	openshiftMonitoringNS = "openshift-monitoring"
	thanosQuerierRoute    = "thanos-querier"
	alertmanagerRoute     = "alertmanager-main"
)

// MetricsE2ESuite tests the obs-mcp (metrics) toolset against a real
// OpenShift cluster where Prometheus and Alertmanager are available
// in the openshift-monitoring namespace.
//
// URLs are auto-discovered from OpenShift routes in the openshift-monitoring
// namespace. Override with PROMETHEUS_URL / ALERTMANAGER_URL env vars if needed.
//
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
type MetricsE2ESuite struct {
	suite.Suite
	*test.McpClient
	server          *mcpserver.Server
	prometheusURL   string
	alertmanagerURL string
}

func (s *MetricsE2ESuite) SetupSuite() {
	s.prometheusURL = os.Getenv("PROMETHEUS_URL")
	s.alertmanagerURL = os.Getenv("ALERTMANAGER_URL")

	if s.prometheusURL == "" || s.alertmanagerURL == "" {
		discovered := s.discoverRoutes()
		if s.prometheusURL == "" {
			s.prometheusURL = discovered[thanosQuerierRoute]
		}
		if s.alertmanagerURL == "" {
			s.alertmanagerURL = discovered[alertmanagerRoute]
		}
	}

	if s.prometheusURL == "" {
		s.T().Fatal("Could not determine Prometheus URL: set PROMETHEUS_URL or ensure the thanos-querier route exists in openshift-monitoring")
	}
	s.T().Logf("Prometheus URL: %s", s.prometheusURL)
	s.T().Logf("Alertmanager URL: %s", s.alertmanagerURL)
}

// discoverRoutes reads OpenShift Route objects from the openshift-monitoring
// namespace and returns a map of route-name -> URL.
func (s *MetricsE2ESuite) discoverRoutes() map[string]string {
	result := make(map[string]string)

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, nil)
	restConfig, err := kubeConfig.ClientConfig()
	if err != nil {
		s.T().Logf("Could not load kubeconfig for route discovery: %v", err)
		return result
	}

	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		s.T().Logf("Could not create dynamic client for route discovery: %v", err)
		return result
	}

	routeGVR := schema.GroupVersionResource{
		Group:    "route.openshift.io",
		Version:  "v1",
		Resource: "routes",
	}

	for _, routeName := range []string{thanosQuerierRoute, alertmanagerRoute} {
		route, err := dynClient.Resource(routeGVR).Namespace(openshiftMonitoringNS).
			Get(context.Background(), routeName, metav1.GetOptions{})
		if err != nil {
			s.T().Logf("Could not get route %s/%s: %v", openshiftMonitoringNS, routeName, err)
			continue
		}

		host, found, _ := unstructured.NestedString(route.Object, "spec", "host")
		if !found || host == "" {
			s.T().Logf("Route %s has no spec.host", routeName)
			continue
		}

		tls, _, _ := unstructured.NestedString(route.Object, "spec", "tls", "termination")
		scheme := "http"
		if tls != "" {
			scheme = "https"
		}

		result[routeName] = fmt.Sprintf("%s://%s", scheme, host)
	}

	return result
}

func (s *MetricsE2ESuite) SetupTest() {
	tomlCfg := fmt.Sprintf(`
		toolsets = ["metrics"]
		[toolset_configs.metrics]
		prometheus_url = "%s"
		alertmanager_url = "%s"
		insecure = true
		guardrails = "none"
	`, s.prometheusURL, s.alertmanagerURL)

	cfg, err := config.ReadToml([]byte(tomlCfg))
	s.Require().NoError(err, "Failed to parse test config")

	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		cfg.KubeConfig = kubeconfig
	}

	provider, err := internalk8s.NewProvider(cfg)
	s.Require().NoError(err, "Failed to create k8s provider")

	s.server, err = mcpserver.NewServer(mcpserver.Configuration{StaticConfig: cfg}, provider)
	s.Require().NoError(err, "Failed to create MCP server")

	s.McpClient = test.NewMcpClient(s.T(), s.server.ServeHTTP())
}

func (s *MetricsE2ESuite) TearDownTest() {
	if s.McpClient != nil {
		s.Close()
	}
	if s.server != nil {
		s.server.Close()
	}
}

func (s *MetricsE2ESuite) TestListMetrics() {
	s.Run("lists metrics matching a regex pattern", func() {
		toolResult, err := s.CallTool("list_metrics", map[string]any{
			"name_regex": ".*cpu.*",
		})
		s.Require().NoError(err)
		s.Require().False(toolResult.IsError, "tool returned error: %s", textContent(toolResult))

		var output listMetricsOutput
		s.Require().NoError(json.Unmarshal([]byte(textContent(toolResult)), &output))
		s.NotEmpty(output.Metrics, "expected at least one CPU metric")
	})
}

func (s *MetricsE2ESuite) TestExecuteInstantQuery() {
	s.Run("executes an instant query for up metric", func() {
		toolResult, err := s.CallTool("execute_instant_query", map[string]any{
			"query": "up",
		})
		s.Require().NoError(err)
		s.Require().False(toolResult.IsError, "tool returned error: %s", textContent(toolResult))

		var output instantQueryOutput
		s.Require().NoError(json.Unmarshal([]byte(textContent(toolResult)), &output))
		s.Equal("vector", output.ResultType)
		s.NotEmpty(output.Result, "expected at least one result for 'up' query")
	})
}

func (s *MetricsE2ESuite) TestExecuteRangeQuery() {
	s.Run("executes a range query with step", func() {
		toolResult, err := s.CallTool("execute_range_query", map[string]any{
			"query":    "up",
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
			"query":    "up",
			"step":     "1m",
			"duration": "5m",
			"title":    "Targets Up",
		})
		s.Require().NoError(err)
		s.Require().False(toolResult.IsError, "tool returned error: %s", textContent(toolResult))
	})
}

func (s *MetricsE2ESuite) TestGetLabelNames() {
	s.Run("retrieves label names", func() {
		toolResult, err := s.CallTool("get_label_names", map[string]any{})
		s.Require().NoError(err)
		s.Require().False(toolResult.IsError, "tool returned error: %s", textContent(toolResult))

		var output labelNamesOutput
		s.Require().NoError(json.Unmarshal([]byte(textContent(toolResult)), &output))
		s.NotEmpty(output.Labels, "expected at least one label name")
		s.Contains(output.Labels, "__name__")
	})
}

func (s *MetricsE2ESuite) TestGetLabelValues() {
	s.Run("retrieves label values for job label", func() {
		toolResult, err := s.CallTool("get_label_values", map[string]any{
			"label": "job",
		})
		s.Require().NoError(err)
		s.Require().False(toolResult.IsError, "tool returned error: %s", textContent(toolResult))

		var output labelValuesOutput
		s.Require().NoError(json.Unmarshal([]byte(textContent(toolResult)), &output))
		s.NotEmpty(output.Values, "expected at least one job label value")
	})
}

func (s *MetricsE2ESuite) TestGetSeries() {
	s.Run("retrieves series for up metric", func() {
		toolResult, err := s.CallTool("get_series", map[string]any{
			"matches": "up",
		})
		s.Require().NoError(err)
		s.Require().False(toolResult.IsError, "tool returned error: %s", textContent(toolResult))

		var output seriesOutput
		s.Require().NoError(json.Unmarshal([]byte(textContent(toolResult)), &output))
		s.NotEmpty(output.Series, "expected at least one series for 'up'")
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
