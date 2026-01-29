package tools

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	promapi "github.com/prometheus/client_golang/api"
	promcfg "github.com/prometheus/common/config"
	"k8s.io/client-go/rest"

	"github.com/rhobs/obs-mcp/pkg/alertmanager"
	"github.com/rhobs/obs-mcp/pkg/prometheus"
	toolsetconfig "github.com/rhobs/obs-mcp/pkg/toolset/config"
)

const (
	defaultServiceAccountCAPath = "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
	defaultPrometheusURL        = "http://localhost:9090"
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

// createAPIConfigWithToken creates a Prometheus API config with a bearer token.
func createAPIConfigWithToken(prometheusURL, token string, insecure bool) (promapi.Config, error) {
	apiConfig := promapi.Config{
		Address: prometheusURL,
	}

	useTLS := strings.HasPrefix(prometheusURL, "https://")
	if useTLS {
		defaultRt := promapi.DefaultRoundTripper.(*http.Transport)

		if insecure {
			defaultRt.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		} else {
			certs, err := createCertPool()
			if err != nil {
				return promapi.Config{}, err
			}
			defaultRt.TLSClientConfig = &tls.Config{RootCAs: certs}
		}

		if token != "" {
			apiConfig.RoundTripper = promcfg.NewAuthorizationCredentialsRoundTripper(
				"Bearer", promcfg.NewInlineSecret(token), defaultRt)
		} else {
			apiConfig.RoundTripper = defaultRt
		}
	} else {
		slog.Warn("Connecting to Prometheus without TLS")
	}

	return apiConfig, nil
}

// createCertPool creates a certificate pool from the service account CA.
func createCertPool() (*x509.CertPool, error) {
	certs := x509.NewCertPool()

	pemData, err := os.ReadFile(defaultServiceAccountCAPath)
	if err != nil {
		slog.Error("Failed to read the CA certificate", "err", err)
		return nil, err
	}
	certs.AppendCertsFromPEM(pemData)
	return certs, nil
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
