package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/utils/ptr"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/netedge"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/netedge/internal/defaults"
)

func InitQueryPrometheus() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        defaults.ToolsetName() + "_query_prometheus",
				Description: "Executes a PromQL query against the cluster's Prometheus service, or runs specialized diagnostic queries for specific components.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"query": {
							Type:        "string",
							Description: "The PromQL query to execute. Required unless 'diagnostic_target' is provided.",
						},
						"diagnostic_target": {
							Type:        "string",
							Description: "Run specialized diagnostics for a specific component. If provided, 'query' is ignored.",
							Enum:        []any{"ingress", "dns"},
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Query Prometheus",
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
	query, _ := params.GetArguments()["query"].(string)
	diagnosticTarget, _ := params.GetArguments()["diagnostic_target"].(string)
	query = strings.TrimSpace(query)

	if query == "" && diagnosticTarget == "" {
		return api.NewToolCallResult("", fmt.Errorf("either 'query' or 'diagnostic_target' must be provided")), nil
	}

	client := netedge.NewNetEdgeClient(params, params.RESTConfig())

	if diagnosticTarget != "" {
		return handleDiagnosticTarget(params, client, diagnosticTarget)
	}

	result, err := client.QueryPrometheus(params.Context, query)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to query prometheus: %v", err)), nil
	}

	// Marshaling the full result to JSON string
	jsonResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal result: %v", err)), nil
	}

	return api.NewToolCallResult(string(jsonResult), nil), nil
}

func handleDiagnosticTarget(params api.ToolHandlerParams, client *netedge.NetEdgeClient, target string) (*api.ToolCallResult, error) {
	var queries map[string]string

	switch target {
	case "ingress":
		queries = map[string]string{
			"ingress_error_rate":       `sum(rate(haproxy_server_http_responses_total{code!~"2.."}[5m]))`,
			"ingress_active_conns":     `sum(haproxy_backend_connections_active_total)`,
			"ingress_reloads_last_day": `changes(haproxy_server_start_time_seconds[1d])`, // Improved from just start time
		}
	case "dns":
		queries = map[string]string{
			"dns_request_rate":   `sum(rate(coredns_dns_request_count_total[5m]))`,
			"dns_nxdomain_rate":  `sum(rate(coredns_dns_request_count_total{rcode="NXDOMAIN"}[5m]))`,
			"dns_servfail_rate":  `sum(rate(coredns_dns_request_count_total{rcode="SERVFAIL"}[5m]))`,
			"dns_panic_recovery": `sum(rate(coredns_panic_count_total[5m]))`, // Added useful signal
		}
	default:
		return api.NewToolCallResult("", fmt.Errorf("unknown diagnostic target: %s", target)), nil
	}

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

			promResp, err := client.QueryPrometheus(params.Context, qStr)
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
