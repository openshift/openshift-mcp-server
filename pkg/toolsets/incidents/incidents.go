package incidents

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/obs/auth"
	"github.com/containers/kubernetes-mcp-server/pkg/obs/toolcfg"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/metrics"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/openshift/cluster-health-analyzer/pkg/alertmanager"
	analyzer "github.com/openshift/cluster-health-analyzer/pkg/mcp"
	"github.com/openshift/cluster-health-analyzer/pkg/prom"
	"k8s.io/utils/ptr"
)

// initGetIncidents adapts the analyzer's incident tool definition to the server API.
func initGetIncidents() []api.ServerTool {
	incidentTool := analyzer.NewIncidentsTool("", "")
	return []api.ServerTool{{
		Tool: api.Tool{
			Name:        incidentTool.Tool.Name,
			Description: incidentTool.Tool.Description,
			InputSchema: incidentTool.Tool.InputSchema.(*jsonschema.Schema),
			Annotations: api.ToolAnnotations{
				Title:         incidentTool.Tool.Annotations.Title,
				ReadOnlyHint:  ptr.To(incidentTool.Tool.Annotations.ReadOnlyHint),
				OpenWorldHint: ptr.To(true),
			},
		},
		Handler: handleGetIncidents,
	}}
}

// handleGetIncidents retrieves incidents from the HTTPS backends configured in
// observability/metrics, falling back to observability/incidents only when the
// metrics configuration is absent. It honors the selected auth_mode and insecure
// settings and uses the shared metrics CA fallback behavior.
// It returns the analyzer's explanatory text alongside structured incident data.
func handleGetIncidents(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	rawConfig, found := toolcfg.From(params, metrics.ToolsetName)
	if !found {
		rawConfig, _ = toolcfg.From(params, ToolsetName)
	}
	cfg, ok := rawConfig.(*metrics.Config)
	if !ok || cfg == nil || cfg.PrometheusURL == "" || cfg.AlertmanagerURL == "" {
		return api.NewToolCallResult("", errors.New(
			"get_incidents requires prometheus_url and alertmanager_url in observability/metrics or observability/incidents toolset configuration",
		)), nil
	}

	args, err := parseGetIncidentsArgs(params)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	for _, endpoint := range []string{cfg.PrometheusURL, cfg.AlertmanagerURL} {
		u, err := url.Parse(endpoint)
		if err != nil {
			return api.NewToolCallResult("", err), nil
		}
		if u.Host == "" || u.Scheme != "https" {
			return api.NewToolCallResult("", fmt.Errorf("backend URL %q must use HTTPS", endpoint)), nil
		}
	}

	transport, err := auth.BuildRoundTripper(
		params.Context, params.RESTConfig(), cfg.GetAuthMode(), true, cfg.Insecure,
	)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	promLoader, err := prom.NewLoaderWithRoundTripper(cfg.PrometheusURL, transport)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	alertLoader, err := alertmanager.NewLoader(alertmanager.LoaderConfig{
		AlertManagerURL: cfg.AlertmanagerURL,
		RoundTripper:    transport,
	})
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	incidentTool := analyzer.NewIncidentsTool("", "")
	result, response, err := incidentTool.GetIncidents(params.Context, args, promLoader, alertLoader)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	if result == nil || len(result.Content) != 1 {
		return api.NewToolCallResult("", errors.New("get_incidents returned no text result")), nil
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		return api.NewToolCallResult("", errors.New("get_incidents returned a non-text result")), nil
	}
	return api.NewToolCallResultFull(text.Text, response, nil), nil
}

// parseGetIncidentsArgs validates arguments and applies the analyzer's defaults.
// It distinguishes omitted arguments from explicit zero or empty values so that
// invalid supplied values are rejected rather than replaced with defaults.
func parseGetIncidentsArgs(params api.ToolHandlerParams) (analyzer.GetIncidentsParams, error) {
	arguments := params.GetArguments()
	var parsed analyzer.GetIncidentsParams
	value, timeRangeSupplied := arguments["time_range"]
	if timeRangeSupplied {
		hours, ok := value.(float64)
		if !ok {
			return analyzer.GetIncidentsParams{}, errors.New("time_range must be a number")
		}
		parsed.TimeRange = hours
	}
	value, minSeveritySupplied := arguments["min_severity"]
	if minSeveritySupplied {
		severity, ok := value.(string)
		if !ok {
			return analyzer.GetIncidentsParams{}, errors.New("min_severity must be a string")
		}
		parsed.MinSeverity = severity
	}
	return analyzer.NormalizeGetIncidentsParams(parsed, timeRangeSupplied, minSeveritySupplied)
}
