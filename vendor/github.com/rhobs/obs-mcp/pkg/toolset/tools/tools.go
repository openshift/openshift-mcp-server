package tools

import (
	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/utils/ptr"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
)

// InitListMetrics creates the list_metrics tool.
func InitListMetrics() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name: "list_metrics",
				Description: `MANDATORY FIRST STEP: List all available metric names in Prometheus.

YOU MUST CALL THIS TOOL BEFORE ANY OTHER QUERY TOOL

This tool MUST be called first for EVERY observability question to:
1. Discover what metrics actually exist in this environment
2. Find the EXACT metric name to use in queries
3. Avoid querying non-existent metrics

NEVER skip this step. NEVER guess metric names. Metric names vary between environments.

After calling this tool:
1. Search the returned list for relevant metrics
2. Use the EXACT metric name found in subsequent queries
3. If no relevant metric exists, inform the user`,
				InputSchema: &jsonschema.Schema{
					Type:       "object",
					Properties: map[string]*jsonschema.Schema{},
				},
				Annotations: api.ToolAnnotations{
					Title:           "List Available Metrics",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					IdempotentHint:  ptr.To(true),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: ListMetricsHandler,
		},
	}
}

// InitExecuteInstantQuery creates the execute_instant_query tool.
func InitExecuteInstantQuery() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name: "execute_instant_query",
				Description: `Execute a PromQL instant query to get current/point-in-time values.

PREREQUISITE: You MUST call list_metrics first to verify the metric exists

WHEN TO USE:
- Current state questions: "What is the current error rate?"
- Point-in-time snapshots: "How many pods are running?"
- Latest values: "Which pods are in Pending state?"

The 'query' parameter MUST use metric names that were returned by list_metrics.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"query": {
							Type:        "string",
							Description: "PromQL query string using metric names verified via list_metrics",
						},
						"time": {
							Type:        "string",
							Description: "Evaluation time as RFC3339 or Unix timestamp. Omit or use 'NOW' for current time.",
						},
					},
					Required: []string{"query"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Execute Instant Query",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					IdempotentHint:  ptr.To(true),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: ExecuteInstantQueryHandler,
		},
	}
}

// InitExecuteRangeQuery creates the execute_range_query tool.
func InitExecuteRangeQuery() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name: "execute_range_query",
				Description: `Execute a PromQL range query to get time-series data over a period.

PREREQUISITE: You MUST call list_metrics first to verify the metric exists

WHEN TO USE:
- Trends over time: "What was CPU usage over the last hour?"
- Rate calculations: "How many requests per second?"
- Historical analysis: "Were there any restarts in the last 5 minutes?"

TIME PARAMETERS:
- 'duration': Look back from now (e.g., "5m", "1h", "24h")
- 'step': Data point resolution (e.g., "1m" for 1-hour duration, "5m" for 24-hour duration)

The 'query' parameter MUST use metric names that were returned by list_metrics.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"query": {
							Type:        "string",
							Description: "PromQL query string using metric names verified via list_metrics",
						},
						"step": {
							Type:        "string",
							Description: "Query resolution step width (e.g., '15s', '1m', '1h'). Choose based on time range: shorter ranges use smaller steps.",
							Pattern:     `^\d+[smhdwy]$`,
						},
						"start": {
							Type:        "string",
							Description: "Start time as RFC3339 or Unix timestamp (optional)",
						},
						"end": {
							Type:        "string",
							Description: "End time as RFC3339 or Unix timestamp (optional). Use `NOW` for current time.",
						},
						"duration": {
							Type:        "string",
							Description: "Duration to look back from now (e.g., '1h', '30m', '1d', '2w') (optional)",
							Pattern:     `^\d+[smhdwy]$`,
						},
					},
					Required: []string{"query", "step"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Execute Range Query",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					IdempotentHint:  ptr.To(true),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: ExecuteRangeQueryHandler,
		},
	}
}

// InitGetLabelNames creates the get_label_names tool.
func InitGetLabelNames() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name: "get_label_names",
				Description: `Get all label names (dimensions) available for filtering a metric.

WHEN TO USE (after calling list_metrics):
- To discover how to filter metrics (by namespace, pod, service, etc.)
- Before constructing label matchers in PromQL queries

The 'metric' parameter should use a metric name from list_metrics output.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"metric": {
							Type:        "string",
							Description: "Metric name (from list_metrics) to get label names for. Leave empty for all metrics.",
						},
						"start": {
							Type:        "string",
							Description: "Start time for label discovery as RFC3339 or Unix timestamp (optional, defaults to 1 hour ago)",
						},
						"end": {
							Type:        "string",
							Description: "End time for label discovery as RFC3339 or Unix timestamp (optional, defaults to now)",
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Get Label Names",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					IdempotentHint:  ptr.To(true),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: GetLabelNamesHandler,
		},
	}
}

// InitGetLabelValues creates the get_label_values tool.
func InitGetLabelValues() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name: "get_label_values",
				Description: `Get all unique values for a specific label.

WHEN TO USE (after calling list_metrics and get_label_names):
- To find exact label values for filtering (namespace names, pod names, etc.)
- To see what values exist before constructing queries

The 'metric' parameter should use a metric name from list_metrics output.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"label": {
							Type:        "string",
							Description: "Label name (from get_label_names) to get values for",
						},
						"metric": {
							Type:        "string",
							Description: "Metric name (from list_metrics) to scope the label values to. Leave empty for all metrics.",
						},
						"start": {
							Type:        "string",
							Description: "Start time for label value discovery as RFC3339 or Unix timestamp (optional, defaults to 1 hour ago)",
						},
						"end": {
							Type:        "string",
							Description: "End time for label value discovery as RFC3339 or Unix timestamp (optional, defaults to now)",
						},
					},
					Required: []string{"label"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Get Label Values",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					IdempotentHint:  ptr.To(true),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: GetLabelValuesHandler,
		},
	}
}

// InitGetSeries creates the get_series tool.
func InitGetSeries() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name: "get_series",
				Description: `Get time series matching selectors and preview cardinality.

WHEN TO USE (optional, after calling list_metrics):
- To verify label filters match expected series before querying
- To check cardinality and avoid slow queries

CARDINALITY GUIDANCE:
- <100 series: Safe
- 100-1000: Usually fine
- >1000: Add more label filters

The selector should use metric names from list_metrics output.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"matches": {
							Type:        "string",
							Description: "PromQL series selector using metric names from list_metrics",
						},
						"start": {
							Type:        "string",
							Description: "Start time for series discovery as RFC3339 or Unix timestamp (optional, defaults to 1 hour ago)",
						},
						"end": {
							Type:        "string",
							Description: "End time for series discovery as RFC3339 or Unix timestamp (optional, defaults to now)",
						},
					},
					Required: []string{"matches"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Get Series",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					IdempotentHint:  ptr.To(true),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: GetSeriesHandler,
		},
	}
}
