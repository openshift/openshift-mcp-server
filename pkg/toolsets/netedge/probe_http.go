package netedge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/utils/ptr"
)

// httpDoer interface allows mocking the HTTP client for tests.
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// defaultHTTPClient wraps the standard http.Client.
type defaultHTTPClient struct {
	client *http.Client
}

func (d *defaultHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return d.client.Do(req)
}

// HTTPResult represents the structured JSON response for probe_http.
type HTTPResult struct {
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	LatencyMS  int64               `json:"latency_ms"`
}

func initProbeHTTP() []api.ServerTool {
	return initProbeHTTPWith(&defaultHTTPClient{client: &http.Client{}})
}

// initProbeHTTPWith creates probe_http tools using the provided httpDoer.
// Use initProbeHTTP() for production; pass a mock to this function in tests.
func initProbeHTTPWith(client httpDoer) []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "probe_http",
				Description: "Send an HTTP(S) request from the MCP server host to verify reachability and inspect the response status code and headers.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"url": {
							Type:        "string",
							Description: "The URL to probe (e.g. https://example.com/path).",
						},
						"method": {
							Type:        "string",
							Description: "HTTP method to use. Defaults to GET.",
							Default:     json.RawMessage(`"GET"`),
						},
						"timeout_seconds": {
							Type:        "integer",
							Description: "Request timeout in seconds. Defaults to 5.",
							Default:     json.RawMessage(`5`),
						},
					},
					Required: []string{"url"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Probe HTTP",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			ClusterAware: ptr.To(false),
			Handler:      makeProbeHTTPHandler(client),
		},
	}
}

func makeProbeHTTPHandler(client httpDoer) api.ToolHandlerFunc {
	return func(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
		urlParam, ok := params.GetArguments()["url"].(string)
		if !ok || urlParam == "" {
			return api.NewToolCallResultStructured(nil, fmt.Errorf("url parameter is required")), nil
		}

		methodParam, ok := params.GetArguments()["method"].(string)
		if !ok || methodParam == "" {
			methodParam = "GET"
		}
		methodParam = strings.ToUpper(methodParam)

		timeoutSeconds := 5
		if ts, ok := params.GetArguments()["timeout_seconds"].(float64); ok && ts > 0 {
			timeoutSeconds = int(ts)
		}

		ctx, cancel := context.WithTimeout(params.Context, time.Duration(timeoutSeconds)*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, methodParam, urlParam, nil)
		if err != nil {
			return api.NewToolCallResultStructured(nil, fmt.Errorf("failed to create HTTP request: %w", err)), nil
		}

		start := time.Now()
		resp, err := client.Do(req)
		latency := time.Since(start)

		if err != nil {
			return api.NewToolCallResultStructured(nil, fmt.Errorf("HTTP request failed: %w", err)), nil
		}
		defer resp.Body.Close() //nolint:errcheck

		result := HTTPResult{
			StatusCode: resp.StatusCode,
			Headers:    map[string][]string(resp.Header),
			LatencyMS:  latency.Milliseconds(),
		}

		return api.NewToolCallResultStructured(result, nil), nil
	}
}
