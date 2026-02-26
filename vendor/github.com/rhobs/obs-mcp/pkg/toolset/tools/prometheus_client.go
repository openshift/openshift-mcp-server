package tools

import (
	"fmt"
	"log/slog"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	promapi "github.com/prometheus/client_golang/api"
	"k8s.io/client-go/rest"

	"github.com/rhobs/obs-mcp/pkg/alertmanager"
	"github.com/rhobs/obs-mcp/pkg/prometheus"
	toolsetconfig "github.com/rhobs/obs-mcp/pkg/toolset/config"
)

const (
	defaultPrometheusURL = "http://localhost:9090"
)

// getConfig retrieves the obs-mcp toolset configuration from params.
func getConfig(params api.ToolHandlerParams) *toolsetconfig.Config {
	if cfg, ok := params.GetToolsetConfig("obs-mcp"); ok {
		if obsCfg, ok := cfg.(*toolsetconfig.Config); ok {
			return obsCfg
		}
	}
	// Return default config if not found
	return &toolsetconfig.Config{}
}

// getPromClient creates a Prometheus client using the toolset configuration.
func getPromClient(params api.ToolHandlerParams) (prometheus.Loader, error) {
	cfg := getConfig(params)

	// Get metrics backend URL from config, fallback to default
	metricsBackendURL := cfg.PrometheusURL
	if metricsBackendURL == "" {
		metricsBackendURL = defaultPrometheusURL
		slog.Info("No prometheus_url configured, using default", "url", defaultPrometheusURL)
	}

	// Get guardrails configuration
	guardrails, err := cfg.GetGuardrails()
	if err != nil {
		slog.Warn("Failed to parse guardrails configuration", "err", err)
	}

	// Create API config using the REST config from params
	apiConfig, err := createAPIConfigFromRESTConfig(params, metricsBackendURL, cfg.Insecure)
	if err != nil {
		return nil, fmt.Errorf("failed to create API config: %w", err)
	}

	// Create Prometheus client
	promClient, err := prometheus.NewPrometheusClient(apiConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	promClient.WithGuardrails(guardrails)

	return promClient, nil
}

// createAPIConfigFromRESTConfig creates a Prometheus API config from Kubernetes REST config.
func createAPIConfigFromRESTConfig(params api.ToolHandlerParams, prometheusURL string, insecure bool) (promapi.Config, error) {
	restConfig := params.RESTConfig()
	if restConfig == nil {
		return promapi.Config{}, fmt.Errorf("no REST config available")
	}

	// For routes/ingresses, we need to configure TLS appropriately
	tlsConfig := rest.TLSClientConfig{Insecure: insecure}
	restConfig.TLSClientConfig = tlsConfig

	// Create HTTP client with Kubernetes authentication
	rt, err := rest.TransportFor(restConfig)
	if err != nil {
		return promapi.Config{}, fmt.Errorf("failed to create transport from REST config: %w", err)
	}

	return promapi.Config{
		Address:      prometheusURL,
		RoundTripper: rt,
	}, nil
}

// getAlertmanagerClient creates an Alertmanager client using the toolset configuration.
func getAlertmanagerClient(params api.ToolHandlerParams) (alertmanager.Loader, error) {
	cfg := getConfig(params)

	alertmanagerURL := cfg.AlertmanagerURL
	if alertmanagerURL == "" {
		return nil, fmt.Errorf("alertmanager_url not configured")
	}

	restConfig := params.RESTConfig()
	if restConfig == nil {
		return nil, fmt.Errorf("no REST config available")
	}

	tlsConfig := rest.TLSClientConfig{Insecure: cfg.Insecure}
	restConfig.TLSClientConfig = tlsConfig

	rt, err := rest.TransportFor(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport from REST config: %w", err)
	}

	apiConfig := promapi.Config{
		Address:      alertmanagerURL,
		RoundTripper: rt,
	}

	// Create Alertmanager client
	amClient, err := alertmanager.NewAlertmanagerClient(apiConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Alertmanager client: %w", err)
	}

	return amClient, nil
}
