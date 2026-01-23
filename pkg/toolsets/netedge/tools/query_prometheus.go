package tools

import (
	"encoding/json"
	"fmt"
	"strings"

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
				Description: "Executes a PromQL query against the cluster's Prometheus service",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"query": {
							Type:        "string",
							Description: "The PromQL query to execute",
						},
					},
					Required: []string{"query"},
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

func queryPrometheusHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	query, _ := params.GetArguments()["query"].(string)
	query = strings.TrimSpace(query)

	if query == "" {
		return api.NewToolCallResult("", fmt.Errorf("query is required")), nil
	}

	client := netedge.NewNetEdgeClient(params, params.RESTConfig())
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
