package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/prometheus"
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

// newPrometheusClient creates a new Prometheus client configured with auth and TLS from the REST config.
func newPrometheusClient(baseURL string, params api.ToolHandlerParams) *prometheus.Client {
	return prometheus.NewClient(baseURL,
		prometheus.WithBearerTokenFromRESTConfig(params.RESTConfig()),
		prometheus.WithTLSFromRESTConfig(params.RESTConfig()),
	)
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

// prettyJSON formats JSON data with indentation.
func prettyJSON(data []byte) (string, error) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return string(data), nil // Return raw if not valid JSON
	}

	pretty, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return string(data), nil
	}
	return string(pretty), nil
}
