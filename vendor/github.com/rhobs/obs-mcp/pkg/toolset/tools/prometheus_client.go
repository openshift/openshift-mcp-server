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
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	promapi "github.com/prometheus/client_golang/api"
	promcfg "github.com/prometheus/common/config"
	"k8s.io/client-go/rest"

	"github.com/rhobs/obs-mcp/pkg/alertmanager"
	"github.com/rhobs/obs-mcp/pkg/prometheus"
	toolsetconfig "github.com/rhobs/obs-mcp/pkg/toolset/config"
)

const (
	defaultPrometheusURL = "http://localhost:9090"
	serviceCAFile        = "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
)

// getConfig retrieves the obs-mcp toolset configuration from params.
func getConfig(params api.ToolHandlerParams) *toolsetconfig.Config {
	if cfg, ok := params.GetToolsetConfig(toolsetconfig.MetricsToolSetName); ok {
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

	apiConfig, err := buildAPIConfig(params, metricsBackendURL, cfg.Insecure, cfg.GetAuthMode())
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

func resolveToken(params api.ToolHandlerParams, restConfig *rest.Config, authMode toolsetconfig.AuthMode) (string, error) {
	slog.Info("Obtaining authentication token", "authMode", authMode)
	switch authMode { //nolint:exhaustive // the default auth_mode value is header
	case toolsetconfig.AuthModeKubeConfig:
		return extractBearerToken(restConfig), nil
	default:
		token := readTokenFromCtx(params)
		if token == "" {
			return "", fmt.Errorf("no bearer token found in request context authorization header")
		}
		return token, nil
	}
}

// buildAPIConfig creates a Prometheus API config using the configured auth mode.
func buildAPIConfig(params api.ToolHandlerParams, prometheusURL string, insecure bool, authMode toolsetconfig.AuthMode) (promapi.Config, error) {
	restConfig := params.RESTConfig()
	if restConfig == nil {
		return promapi.Config{}, fmt.Errorf("no REST config available")
	}

	token, err := resolveToken(params, restConfig, authMode)
	if err != nil {
		return promapi.Config{}, err
	}

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

	// If CAData wasn't available, try serviceCAFile
	if !caLoaded {
		caPEM, err := os.ReadFile(serviceCAFile)
		if err != nil {
			slog.Warn("Failed to read CA file", "file", serviceCAFile, "error", err)
		} else {
			if ok := certPool.AppendCertsFromPEM(caPEM); ok {
				slog.Debug("Loaded cluster CA from file", "file", serviceCAFile)
			} else {
				slog.Warn("Failed to parse CA certificates from file", "file", serviceCAFile)
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

func readTokenFromCtx(params api.ToolHandlerParams) string {
	authHeader, ok := params.Value(kubernetes.OAuthAuthorizationHeader).(string)
	if !ok {
		return ""
	}
	parts := strings.Fields(authHeader)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return strings.TrimSpace(authHeader)
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

	token, err := resolveToken(params, restConfig, cfg.GetAuthMode())
	if err != nil {
		return nil, err
	}

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
