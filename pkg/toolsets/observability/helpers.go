package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/prometheus"
)

const (
	// defaultMonitoringNamespace is the default namespace for OpenShift monitoring components
	defaultMonitoringNamespace = "openshift-monitoring"

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

// routeURLCache caches resolved route URLs for the lifetime of the server process.
// This avoids repeated Kubernetes API calls since routes rarely change.
// Key format: "apiServerHost/namespace/routeName", value: URL string.
// The API server host is included to support multi-cluster (ACM) environments.
var routeURLCache sync.Map

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
// Results are cached for the lifetime of the server process to avoid repeated API calls.
// The cache key includes the API server host to support multi-cluster (ACM) environments.
func getRouteURL(ctx context.Context, params api.ToolHandlerParams, routeName, namespace string) (string, error) {
	// Include API server host in cache key to support multi-cluster (ACM) environments
	cacheKey := params.RESTConfig().Host + "/" + namespace + "/" + routeName

	// Check cache first
	if cached, ok := routeURLCache.Load(cacheKey); ok {
		klog.V(4).Infof("Using cached route URL for %s", cacheKey)
		return cached.(string), nil
	}

	// Fetch from Kubernetes API
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

	url := fmt.Sprintf("https://%s", host)

	// Cache the result for future calls
	routeURLCache.Store(cacheKey, url)
	klog.V(4).Infof("Cached route URL for %s: %s", cacheKey, url)

	return url, nil
}

// newPrometheusClient creates a new Prometheus client configured with auth and TLS from the REST config.
// This client is used for both Prometheus and Alertmanager API access.
func newPrometheusClient(baseURL string, params api.ToolHandlerParams) *prometheus.Client {
	return prometheus.NewClient(baseURL,
		prometheus.WithBearerTokenFromRESTConfig(params.RESTConfig()),
		prometheus.WithTLSFromRESTConfig(params.RESTConfig()),
	)
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
