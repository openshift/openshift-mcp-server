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

	apiConfig, err := createAPIConfigFromRESTConfig(params, metricsBackendURL, cfg.Insecure)
	if err != nil {
		return nil, fmt.Errorf("failed to create API config: %w", err)
	}

	promClient, err := prometheus.NewPrometheusClient(apiConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	promClient.WithGuardrails(guardrails)

	return promClient, nil
}

// createAPIConfigFromRESTConfig creates a Prometheus API config from Kubernetes REST config.
// It builds a fresh HTTP transport without the AccessControlRoundTripper to avoid
// Kubernetes API validation on Prometheus endpoints.
func createAPIConfigFromRESTConfig(params api.ToolHandlerParams, prometheusURL string, insecure bool) (promapi.Config, error) {
	restConfig := params.RESTConfig()
	if restConfig == nil {
		return promapi.Config{}, fmt.Errorf("no REST config available")
	}

	token := extractBearerToken(restConfig)

	// Use the same pattern as createAPIConfigWithToken from obs-mcp/pkg/mcp/auth.go
	return createAPIConfigWithToken(restConfig, prometheusURL, token, insecure)
}

// createAPIConfigWithToken creates a Prometheus API config with bearer token authentication.
// This follows the pattern from obs-mcp/pkg/mcp/auth.go to avoid using rest.TransportFor()
// which would inherit the AccessControlRoundTripper.
func createAPIConfigWithToken(restConfig *rest.Config, prometheusURL, token string, insecure bool) (promapi.Config, error) {
	apiConfig := promapi.Config{
		Address: prometheusURL,
	}

	useTLS := strings.HasPrefix(prometheusURL, "https://")
	if useTLS {
		defaultRt, ok := promapi.DefaultRoundTripper.(*http.Transport)
		if !ok {
			return promapi.Config{}, fmt.Errorf("unexpected RoundTripper type: %T, expected *http.Transport", promapi.DefaultRoundTripper)
		}

		if insecure {
			defaultRt.TLSClientConfig = &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: true,
			}
		} else {
			// Build cert pool from REST config
			certs, err := createCertPoolFromRESTConfig(restConfig)
			if err != nil {
				return promapi.Config{}, err
			}
			defaultRt.TLSClientConfig = &tls.Config{
				MinVersion: tls.VersionTLS12,
				RootCAs:    certs,
			}
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

// createCertPoolFromRESTConfig creates a cert pool from Kubernetes REST config.
func createCertPoolFromRESTConfig(restConfig *rest.Config) (*x509.CertPool, error) {
	var certPool *x509.CertPool

	// Start with system cert pool if available
	if systemPool, err := x509.SystemCertPool(); err == nil && systemPool != nil {
		certPool = systemPool
	} else {
		certPool = x509.NewCertPool()
	}

	// Try to append cluster CA from REST config
	var caLoaded bool

	// First, try CAData
	if len(restConfig.CAData) > 0 {
		if ok := certPool.AppendCertsFromPEM(restConfig.CAData); ok {
			caLoaded = true
			slog.Debug("Loaded cluster CA from REST config CAData")
		} else {
			slog.Warn("Failed to parse CA certificates from REST config CAData")
		}
	}

	// If CAData wasn't available, try CAFile
	if !caLoaded && restConfig.CAFile != "" {
		caPEM, err := os.ReadFile(restConfig.CAFile)
		if err != nil {
			slog.Warn("Failed to read CA file", "file", restConfig.CAFile, "error", err)
		} else {
			if ok := certPool.AppendCertsFromPEM(caPEM); ok {
				slog.Debug("Loaded cluster CA from file", "file", restConfig.CAFile)
			} else {
				slog.Warn("Failed to parse CA certificates from file", "file", restConfig.CAFile)
			}
		}
	}

	return certPool, nil
}

// extractBearerToken extracts the bearer token from Kubernetes REST config.
func extractBearerToken(restConfig *rest.Config) string {
	if restConfig == nil {
		return ""
	}

	if restConfig.BearerToken != "" {
		return restConfig.BearerToken
	}

	if restConfig.BearerTokenFile != "" {
		token, err := os.ReadFile(restConfig.BearerTokenFile)
		if err != nil {
			slog.Warn("Failed to read token file", "file", restConfig.BearerTokenFile, "error", err)
			return ""
		}
		return strings.TrimSpace(string(token))
	}

	return ""
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

	// Extract bearer token from REST config
	token := extractBearerToken(restConfig)

	apiConfig, err := createAPIConfigWithToken(restConfig, alertmanagerURL, token, cfg.Insecure)
	if err != nil {
		return nil, fmt.Errorf("failed to create API config: %w", err)
	}

	amClient, err := alertmanager.NewAlertmanagerClient(apiConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Alertmanager client: %w", err)
	}

	return amClient, nil
}
