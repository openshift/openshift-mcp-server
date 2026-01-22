package observability

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
)

const (
	// defaultMonitoringNamespace is the default namespace for OpenShift monitoring components
	defaultMonitoringNamespace = "openshift-monitoring"

	// thanosQuerierRoute is the route name for Thanos Querier
	thanosQuerierRoute = "thanos-querier"

	// alertmanagerRoute is the route name for Alertmanager
	alertmanagerRoute = "alertmanager-main"

	// maxQueryLength is the maximum allowed query length to prevent DoS
	maxQueryLength = 10000

	// maxResponseSize is the maximum response size (10MB)
	maxResponseSize = 10 * 1024 * 1024

	// defaultTimeout is the default HTTP timeout
	defaultTimeout = 30 * time.Second
)

// routeGVR is the GroupVersionResource for OpenShift Routes
var routeGVR = schema.GroupVersionResource{
	Group:    "route.openshift.io",
	Version:  "v1",
	Resource: "routes",
}

// allowedPrometheusEndpoints is a whitelist of allowed Prometheus API endpoints
var allowedPrometheusEndpoints = map[string]bool{
	"/api/v1/query":       true,
	"/api/v1/query_range": true,
	"/api/v1/series":      true,
	"/api/v1/labels":      true,
}

// allowedPrometheusLabelPattern matches /api/v1/label/<label>/values endpoints
var allowedPrometheusLabelPattern = regexp.MustCompile(`^/api/v1/label/[^/]+/values$`)

// allowedAlertmanagerEndpoints is a whitelist of allowed Alertmanager API endpoints
var allowedAlertmanagerEndpoints = map[string]bool{
	"/api/v2/alerts":   true,
	"/api/v2/silences": true,
	"/api/v1/alerts":   true,
}

// getMonitoringNamespace returns the monitoring namespace from config or the default.
func getMonitoringNamespace(params api.ToolHandlerParams) string {
	if cfg, ok := params.GetToolsetConfig("observability"); ok {
		if obsCfg, ok := cfg.(*Config); ok && obsCfg.MonitoringNamespace != "" {
			return obsCfg.MonitoringNamespace
		}
	}
	return defaultMonitoringNamespace
}

// getRouteURL retrieves the URL for an OpenShift route.
func getRouteURL(ctx context.Context, params api.ToolHandlerParams, routeName, namespace string) (string, error) {
	route, err := params.DynamicClient().Resource(routeGVR).Namespace(namespace).Get(ctx, routeName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get route %s/%s: %w; check RBAC permissions for routes in the monitoring namespace", namespace, routeName, err)
	}

	host, found, err := unstructured.NestedString(route.Object, "spec", "host")
	if err != nil {
		return "", fmt.Errorf("failed to read route host: %w", err)
	}
	if !found || host == "" {
		return "", fmt.Errorf("route %s/%s has no host configured; verify the monitoring stack is properly deployed", namespace, routeName)
	}

	return fmt.Sprintf("https://%s", host), nil
}

// getAuthToken retrieves the bearer token for API authentication.
func getAuthToken(params api.ToolHandlerParams) (string, error) {
	config := params.RESTConfig()

	// Try bearer token directly
	if config.BearerToken != "" {
		return config.BearerToken, nil
	}

	// Try bearer token file
	if config.BearerTokenFile != "" {
		token, err := os.ReadFile(config.BearerTokenFile)
		if err != nil {
			return "", fmt.Errorf("failed to read token file: %w", err)
		}
		return strings.TrimSpace(string(token)), nil
	}

	return "", fmt.Errorf("no bearer token available in REST config; run 'oc login' or ensure KUBECONFIG is set correctly")
}

// createHTTPClient creates an HTTP client configured with TLS settings from the REST config.
// It attempts to load the cluster CA for proper TLS verification, falling back to insecure
// mode only if no CA is available.
func createHTTPClient(params api.ToolHandlerParams) *http.Client {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	config := params.RESTConfig()

	// Try to build a cert pool with the cluster CA
	var certPool *x509.CertPool
	var caLoaded bool

	// First, try to load CA from REST config's CAData
	if len(config.CAData) > 0 {
		// Start with system cert pool if available
		if systemPool, err := x509.SystemCertPool(); err == nil && systemPool != nil {
			certPool = systemPool
		} else {
			certPool = x509.NewCertPool()
		}
		if ok := certPool.AppendCertsFromPEM(config.CAData); ok {
			tlsConfig.RootCAs = certPool
			caLoaded = true
			klog.V(4).Info("Loaded cluster CA from REST config CAData")
		} else {
			klog.V(2).Info("Failed to parse CA certificates from REST config CAData")
		}
	}

	// If CAData wasn't available or didn't work, try CAFile
	if !caLoaded && config.CAFile != "" {
		caPEM, err := os.ReadFile(config.CAFile)
		if err != nil {
			klog.V(2).Infof("Failed to read CA file %s: %v", config.CAFile, err)
		} else {
			// Start with system cert pool if available
			if systemPool, err := x509.SystemCertPool(); err == nil && systemPool != nil {
				certPool = systemPool
			} else {
				certPool = x509.NewCertPool()
			}
			if ok := certPool.AppendCertsFromPEM(caPEM); ok {
				tlsConfig.RootCAs = certPool
				caLoaded = true
				klog.V(4).Infof("Loaded cluster CA from file %s", config.CAFile)
			} else {
				klog.V(2).Infof("Failed to parse CA certificates from file %s", config.CAFile)
			}
		}
	}

	// If no CA was loaded, try system cert pool alone (for routes with public CAs)
	if !caLoaded {
		if systemPool, err := x509.SystemCertPool(); err == nil && systemPool != nil {
			tlsConfig.RootCAs = systemPool
			klog.V(4).Info("Using system certificate pool for TLS verification")
		} else {
			// Last resort: skip verification with a warning
			klog.Warning("No cluster CA available and system cert pool failed; using insecure TLS (skip verification)")
			tlsConfig.InsecureSkipVerify = true
		}
	}

	return &http.Client{
		Timeout: defaultTimeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
}

// executeHTTPRequest executes an HTTP GET request with authentication.
func executeHTTPRequest(ctx context.Context, params api.ToolHandlerParams, requestURL string) ([]byte, error) {
	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication
	token, err := getAuthToken(params)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	// Execute request
	client := createHTTPClient(params)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response with size limit
	limitedReader := io.LimitReader(resp.Body, maxResponseSize+1)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if len(body) > maxResponseSize {
		return nil, fmt.Errorf("response size exceeds maximum of %d bytes", maxResponseSize)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncateString(string(body), 200))
	}

	return body, nil
}

// validatePrometheusEndpoint checks if the endpoint is allowed.
func validatePrometheusEndpoint(endpoint string) error {
	if allowedPrometheusEndpoints[endpoint] {
		return nil
	}
	if allowedPrometheusLabelPattern.MatchString(endpoint) {
		return nil
	}
	return fmt.Errorf("endpoint %s is not allowed; allowed endpoints: %v", endpoint, getAllowedPrometheusEndpoints())
}

// getAllowedPrometheusEndpoints returns a list of allowed endpoints for error messages.
func getAllowedPrometheusEndpoints() []string {
	endpoints := make([]string, 0, len(allowedPrometheusEndpoints)+1)
	for ep := range allowedPrometheusEndpoints {
		endpoints = append(endpoints, ep)
	}
	endpoints = append(endpoints, "/api/v1/label/<name>/values")
	return endpoints
}

// validateAlertmanagerEndpoint checks if the endpoint is allowed.
func validateAlertmanagerEndpoint(endpoint string) error {
	if !allowedAlertmanagerEndpoints[endpoint] {
		return fmt.Errorf("endpoint %s is not allowed; allowed endpoints: %v", endpoint, getAllowedAlertmanagerEndpoints())
	}
	return nil
}

// getAllowedAlertmanagerEndpoints returns a list of allowed endpoints for error messages.
func getAllowedAlertmanagerEndpoints() []string {
	endpoints := make([]string, 0, len(allowedAlertmanagerEndpoints))
	for ep := range allowedAlertmanagerEndpoints {
		endpoints = append(endpoints, ep)
	}
	return endpoints
}

// convertRelativeTime converts relative time strings to RFC3339 timestamps.
// Supports: "now", "-10m", "-1h", "-1d", or passthrough for RFC3339/Unix timestamps.
func convertRelativeTime(timeStr string) (string, error) {
	timeStr = strings.TrimSpace(timeStr)

	// If already a timestamp (contains T) or is numeric (Unix timestamp), return as-is
	if strings.Contains(timeStr, "T") || isNumeric(timeStr) {
		return timeStr, nil
	}

	// Handle 'now'
	if timeStr == "now" {
		return time.Now().UTC().Format(time.RFC3339), nil
	}

	// Handle relative times like '-10m', '-1h', '-1d', '-30s'
	if strings.HasPrefix(timeStr, "-") {
		// Parse duration (Go's time.ParseDuration doesn't support 'd' for days)
		durationStr := timeStr[1:] // Remove leading '-'

		// Handle days specially
		if strings.HasSuffix(durationStr, "d") {
			days, err := parseIntFromString(strings.TrimSuffix(durationStr, "d"))
			if err != nil {
				return "", fmt.Errorf("invalid relative time format: %s", timeStr)
			}
			targetTime := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
			return targetTime.Format(time.RFC3339), nil
		}

		// Parse standard durations (s, m, h)
		duration, err := time.ParseDuration(durationStr)
		if err != nil {
			return "", fmt.Errorf("invalid relative time format: %s", timeStr)
		}
		targetTime := time.Now().UTC().Add(-duration)
		return targetTime.Format(time.RFC3339), nil
	}

	return "", fmt.Errorf("invalid time format: %s; expected 'now', relative time like '-10m', '-1h', '-1d', or RFC3339 timestamp", timeStr)
}

// isNumeric checks if a string contains only digits.
func isNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// parseIntFromString parses an integer from a string.
func parseIntFromString(s string) (int, error) {
	var result int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid number: %s", s)
		}
		result = result*10 + int(c-'0')
	}
	return result, nil
}

// prettyJSON formats JSON data with indentation.
func prettyJSON(data []byte) (string, error) {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return string(data), nil // Return raw if not valid JSON
	}

	pretty, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return string(data), nil
	}
	return string(pretty), nil
}

// buildQueryURL constructs a URL with query parameters.
func buildQueryURL(baseURL, endpoint string, params url.Values) string {
	fullURL := baseURL + endpoint
	if len(params) > 0 {
		fullURL += "?" + params.Encode()
	}
	return fullURL
}

// truncateString truncates a string to the specified length.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
