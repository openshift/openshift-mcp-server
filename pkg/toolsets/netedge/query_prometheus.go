package netedge

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/prometheus"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/netedge/internal/defaults"
)

const (
	// defaultMonitoringNamespace is the default namespace for OpenShift monitoring components
	defaultMonitoringNamespace = "openshift-monitoring"

	// thanosQuerierRoute is the route name for Thanos Querier
	thanosQuerierRoute = "thanos-querier"
)

var routeGVR = schema.GroupVersionResource{
	Group:    "route.openshift.io",
	Version:  "v1",
	Resource: "routes",
}

func InitQueryPrometheus() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        defaults.ToolsetName() + "_query_prometheus",
				Description: "Executes specialized diagnostic queries for specific NetEdge components (ingress, dns).",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"diagnostic_target": {
							Type:        "string",
							Description: "Run specialized diagnostics for a specific component.",
							Enum:        []any{"ingress", "dns", "operators"},
						},
					},
					Required: []string{"diagnostic_target"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "NetEdge Diagnostics",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					IdempotentHint:  ptr.To(true),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: queryPrometheusHandler,
		},
	}
}

type DiagnosticResult struct {
	Name      string      `json:"name"`
	Query     string      `json:"query"`
	Result    interface{} `json:"result,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

func queryPrometheusHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	diagnosticTarget, ok := params.GetArguments()["diagnostic_target"].(string)
	if !ok || diagnosticTarget == "" {
		return api.NewToolCallResult("", fmt.Errorf("diagnostic_target is required")), nil
	}

	return handleDiagnosticTarget(params, diagnosticTarget)
}

func handleDiagnosticTarget(params api.ToolHandlerParams, target string) (*api.ToolCallResult, error) {
	var queries map[string]string

	switch target {
	case "ingress":
		queries = map[string]string{
			"ingress_error_rate":       `sum(rate(haproxy_server_http_responses_total{code!~"2.."}[5m]))`,
			"ingress_active_conns":     `sum(haproxy_backend_connections_active_total)`,
			"ingress_reloads_last_day": `changes(haproxy_server_start_time_seconds[1d])`,
			"ingress_top_error_routes": `topk(5, sum by (route) (rate(haproxy_server_http_responses_total{code!~"2.."}[5m])))`,
		}
	case "dns":
		queries = map[string]string{
			"dns_request_rate":    `sum(rate(coredns_dns_request_count_total[5m]))`,
			"dns_nxdomain_rate":   `sum(rate(coredns_dns_request_count_total{rcode="NXDOMAIN"}[5m]))`,
			"dns_servfail_rate":   `sum(rate(coredns_dns_request_count_total{rcode="SERVFAIL"}[5m]))`,
			"dns_panic_recovery":  `sum(rate(coredns_panic_count_total[5m]))`,
			"dns_error_breakdown": `sum by (rcode) (rate(coredns_dns_request_count_total{rcode!="NOERROR"}[5m]))`,
			"dns_rewrite_count":   `sum(rate(coredns_plugin_rewrite_request_count_total[5m]))`,
		}
	case "operators":
		queries = map[string]string{
			"active_alerts": `ALERTS{alertstate="firing", namespace=~"openshift-ingress-operator|openshift-dns"}`,
			"operator_up":   `up{job=~"cluster-ingress-operator|dns-operator"}`,
		}
	default:
		return api.NewToolCallResult("", fmt.Errorf("unknown diagnostic target: %s", target)), nil
	}

	// Resolve Thanos URL
	baseURL, err := getRouteURL(params.Context, params, thanosQuerierRoute, defaultMonitoringNamespace)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get Thanos Querier route: %v", err)), nil
	}

	opts := []prometheus.ClientOption{
		prometheus.WithBearerTokenFromRESTConfig(params.RESTConfig()),
		prometheus.WithTLSFromRESTConfig(params.RESTConfig()),
	}

	// Explicitly handle insecure config since WithTLSFromRESTConfig might not cover it strictly if CAs fail to load but system pool works
	if params.RESTConfig().Insecure {
		opts = append(opts, prometheus.WithInsecure(true))
	}

	client := prometheus.NewClient(baseURL, opts...)

	results := make([]DiagnosticResult, 0, len(queries))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for name, q := range queries {
		wg.Add(1)
		go func(n, qStr string) {
			defer wg.Done()
			res := DiagnosticResult{
				Name:      n,
				Query:     qStr,
				Timestamp: time.Now(),
			}

			// Using generic Query since these are instant queries
			promResp, err := client.Query(params.Context, qStr, "")
			if err != nil {
				res.Error = err.Error()
			} else {
				res.Result = promResp.Data.Result
			}

			mu.Lock()
			results = append(results, res)
			mu.Unlock()
		}(name, q)
	}

	wg.Wait()

	jsonResult, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal diagnostic results: %v", err)), nil
	}

	return api.NewToolCallResult(string(jsonResult), nil), nil
}

// getRouteURL retrieves the URL for an OpenShift route.
func getRouteURL(ctx context.Context, params api.ToolHandlerParams, routeName, namespace string) (string, error) {
	route, err := params.DynamicClient().Resource(routeGVR).Namespace(namespace).Get(ctx, routeName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get route %s/%s: %w", namespace, routeName, err)
	}

	host, found, err := unstructured.NestedString(route.Object, "spec", "host")
	if err != nil {
		return "", fmt.Errorf("failed to read route host: %w", err)
	}
	if !found || host == "" {
		return "", fmt.Errorf("route %s/%s has no host configured", namespace, routeName)
	}

	return fmt.Sprintf("https://%s", host), nil
}
