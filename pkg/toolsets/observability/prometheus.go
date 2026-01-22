package observability

import (
	"fmt"
	"net/url"

	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/utils/ptr"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
)

// initPrometheus returns the Prometheus query tools.
func initPrometheus() []api.ServerTool {
	return []api.ServerTool{
		initPrometheusQuery(),
		initPrometheusQueryRange(),
	}
}

// initPrometheusQuery creates the prometheus_query tool.
func initPrometheusQuery() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name: "prometheus_query",
			Description: `Execute an instant PromQL query against the cluster's Thanos Querier.
Returns current metric values at the specified time (or current time if not specified).
Use this for point-in-time metric values.

Common queries:
- up{job="apiserver"} - Check if API server is up
- sum by(namespace) (container_memory_usage_bytes) - Memory usage by namespace
- rate(container_cpu_usage_seconds_total[5m]) - CPU usage rate
- kube_pod_status_phase{phase="Running"} - Running pods count`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"query": {
						Type:        "string",
						Description: "PromQL query string (e.g., 'up{job=\"apiserver\"}', 'sum by(namespace) (container_memory_usage_bytes)')",
					},
					"time": {
						Type:        "string",
						Description: "Optional evaluation timestamp. Accepts RFC3339 format (e.g., '2024-01-01T12:00:00Z') or Unix timestamp. If not provided, uses current time.",
					},
				},
				Required: []string{"query"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "Prometheus: Instant Query",
				ReadOnlyHint:    ptr.To(true),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(true),
				OpenWorldHint:   ptr.To(true),
			},
		},
		Handler: prometheusQueryHandler,
	}
}

// initPrometheusQueryRange creates the prometheus_query_range tool.
func initPrometheusQueryRange() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name: "prometheus_query_range",
			Description: `Execute a range PromQL query against the cluster's Thanos Querier.
Returns metric values over a time range with specified resolution.
Use this for time-series data, trends, and historical analysis.

Supports relative times:
- 'now' for current time
- '-10m', '-1h', '-1d' for relative past times

Example: Get CPU usage over the last hour with 1-minute resolution.`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"query": {
						Type:        "string",
						Description: "PromQL query string (e.g., 'rate(container_cpu_usage_seconds_total[5m])')",
					},
					"start": {
						Type:        "string",
						Description: "Start time. Accepts RFC3339 timestamp (e.g., '2024-01-01T12:00:00Z'), Unix timestamp, or relative time (e.g., '-1h', '-30m', '-1d')",
					},
					"end": {
						Type:        "string",
						Description: "End time. Accepts RFC3339 timestamp, Unix timestamp, 'now', or relative time",
					},
					"step": {
						Type:        "string",
						Description: "Query resolution step width (e.g., '15s', '1m', '5m'). Determines the granularity of returned data points. Default: '1m'",
						Default:     api.ToRawMessage("1m"),
					},
				},
				Required: []string{"query", "start", "end"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "Prometheus: Range Query",
				ReadOnlyHint:    ptr.To(true),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(true),
				OpenWorldHint:   ptr.To(true),
			},
		},
		Handler: prometheusQueryRangeHandler,
	}
}

// prometheusQueryHandler handles instant PromQL queries.
func prometheusQueryHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	// Extract and validate query
	query, ok := params.GetArguments()["query"].(string)
	if !ok || query == "" {
		return api.NewToolCallResult("", fmt.Errorf("query parameter is required")), nil
	}

	if len(query) > maxQueryLength {
		return api.NewToolCallResult("", fmt.Errorf("query exceeds maximum length of %d characters", maxQueryLength)), nil
	}

	// Get Thanos Querier URL
	baseURL, err := getRouteURL(params.Context, params, thanosQuerierRoute, getMonitoringNamespace(params))
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get Thanos Querier route: %w", err)), nil
	}

	// Validate endpoint
	endpoint := "/api/v1/query"
	if err := validatePrometheusEndpoint(endpoint); err != nil {
		return api.NewToolCallResult("", err), nil
	}

	// Build query parameters
	queryParams := url.Values{}
	queryParams.Set("query", query)

	if timeParam, ok := params.GetArguments()["time"].(string); ok && timeParam != "" {
		queryParams.Set("time", timeParam)
	}

	// Build URL and execute request
	requestURL := buildQueryURL(baseURL, endpoint, queryParams)
	body, err := executeHTTPRequest(params.Context, params, requestURL)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("prometheus query failed: %w", err)), nil
	}

	// Format response
	result, err := prettyJSON(body)
	if err != nil {
		return api.NewToolCallResult(string(body), nil), nil
	}

	return api.NewToolCallResult(result, nil), nil
}

// prometheusQueryRangeHandler handles range PromQL queries.
func prometheusQueryRangeHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	// Extract and validate query
	query, ok := params.GetArguments()["query"].(string)
	if !ok || query == "" {
		return api.NewToolCallResult("", fmt.Errorf("query parameter is required")), nil
	}

	if len(query) > maxQueryLength {
		return api.NewToolCallResult("", fmt.Errorf("query exceeds maximum length of %d characters", maxQueryLength)), nil
	}

	// Extract and validate start time
	start, ok := params.GetArguments()["start"].(string)
	if !ok || start == "" {
		return api.NewToolCallResult("", fmt.Errorf("start parameter is required")), nil
	}

	// Extract and validate end time
	end, ok := params.GetArguments()["end"].(string)
	if !ok || end == "" {
		return api.NewToolCallResult("", fmt.Errorf("end parameter is required")), nil
	}

	// Extract step (optional with default)
	step := "1m"
	if s, ok := params.GetArguments()["step"].(string); ok && s != "" {
		step = s
	}

	// Convert relative times
	startTime, err := convertRelativeTime(start)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("invalid start time: %w", err)), nil
	}

	endTime, err := convertRelativeTime(end)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("invalid end time: %w", err)), nil
	}

	// Get Thanos Querier URL
	baseURL, err := getRouteURL(params.Context, params, thanosQuerierRoute, getMonitoringNamespace(params))
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get Thanos Querier route: %w", err)), nil
	}

	// Validate endpoint
	endpoint := "/api/v1/query_range"
	if err := validatePrometheusEndpoint(endpoint); err != nil {
		return api.NewToolCallResult("", err), nil
	}

	// Build query parameters
	queryParams := url.Values{}
	queryParams.Set("query", query)
	queryParams.Set("start", startTime)
	queryParams.Set("end", endTime)
	queryParams.Set("step", step)

	// Build URL and execute request
	requestURL := buildQueryURL(baseURL, endpoint, queryParams)
	body, err := executeHTTPRequest(params.Context, params, requestURL)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("prometheus range query failed: %w", err)), nil
	}

	// Format response
	result, err := prettyJSON(body)
	if err != nil {
		return api.NewToolCallResult(string(body), nil), nil
	}

	return api.NewToolCallResult(result, nil), nil
}
