//go:build observability_e2e

package observability

import (
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

// MetricsE2ESuite hosts OpenShift-only metrics checks that TestObs* does not cover
// (Route-backed URLs + CMO metric). Shared tool smoke tests live in test/e2e TestObs*.
type MetricsE2ESuite struct {
	suite.Suite
	*test.McpClient
	server          *mcpserver.Server
	prometheusURL   string
	alertmanagerURL string
}

func (s *MetricsE2ESuite) SetupSuite() {
	prepareObsE2EKubeconfig(s.T())
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
}

// discoverRoutesViaResolver uses the production MetricsRouteResolver.
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
