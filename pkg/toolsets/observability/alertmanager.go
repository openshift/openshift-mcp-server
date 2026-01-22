package observability

import (
	"fmt"
	"net/url"

	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/utils/ptr"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
)

// initAlertmanager returns the Alertmanager tools.
func initAlertmanager() []api.ServerTool {
	return []api.ServerTool{
		initAlertmanagerAlerts(),
	}
}

// initAlertmanagerAlerts creates the alertmanager_alerts tool.
func initAlertmanagerAlerts() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name: "alertmanager_alerts",
			Description: `Query active and pending alerts from the cluster's Alertmanager.
Useful for monitoring cluster health, detecting issues, and incident response.

Returns alerts with their labels, annotations, status, and timing information.
Can filter by active/silenced/inhibited state.

Common use cases:
- Check for critical alerts affecting the cluster
- Monitor for specific alert types (e.g., high CPU, disk pressure)
- Verify alert silences are working correctly`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"active": {
						Type:        "boolean",
						Description: "Filter for active (firing) alerts. Default: true",
						Default:     api.ToRawMessage(true),
					},
					"silenced": {
						Type:        "boolean",
						Description: "Include silenced alerts in the results. Default: false",
						Default:     api.ToRawMessage(false),
					},
					"inhibited": {
						Type:        "boolean",
						Description: "Include inhibited alerts in the results. Default: false",
						Default:     api.ToRawMessage(false),
					},
					"filter": {
						Type:        "string",
						Description: "Optional filter using Alertmanager filter syntax. Examples: 'alertname=Watchdog', 'severity=critical', 'namespace=openshift-monitoring'",
					},
				},
			},
			Annotations: api.ToolAnnotations{
				Title:           "Alertmanager: Get Alerts",
				ReadOnlyHint:    ptr.To(true),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(true),
				OpenWorldHint:   ptr.To(true),
			},
		},
		Handler: alertmanagerAlertsHandler,
	}
}

// alertmanagerAlertsHandler handles Alertmanager alerts queries.
func alertmanagerAlertsHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	// Get Alertmanager URL
	baseURL, err := getRouteURL(params.Context, params, alertmanagerRoute, getMonitoringNamespace(params))
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get Alertmanager route: %w", err)), nil
	}

	// Validate endpoint
	endpoint := "/api/v2/alerts"
	if err := validateAlertmanagerEndpoint(endpoint); err != nil {
		return api.NewToolCallResult("", err), nil
	}

	// Build query parameters
	queryParams := url.Values{}

	// Handle active parameter (default: true)
	active := true
	if v, ok := params.GetArguments()["active"].(bool); ok {
		active = v
	}
	queryParams.Set("active", fmt.Sprintf("%t", active))

	// Handle silenced parameter (default: false)
	silenced := false
	if v, ok := params.GetArguments()["silenced"].(bool); ok {
		silenced = v
	}
	queryParams.Set("silenced", fmt.Sprintf("%t", silenced))

	// Handle inhibited parameter (default: false)
	inhibited := false
	if v, ok := params.GetArguments()["inhibited"].(bool); ok {
		inhibited = v
	}
	queryParams.Set("inhibited", fmt.Sprintf("%t", inhibited))

	// Handle optional filter
	if filter, ok := params.GetArguments()["filter"].(string); ok && filter != "" {
		// Validate filter length
		if len(filter) > maxQueryLength {
			return api.NewToolCallResult("", fmt.Errorf("filter exceeds maximum length of %d characters", maxQueryLength)), nil
		}
		queryParams.Add("filter", filter)
	}

	// Build URL and execute request
	requestURL := buildQueryURL(baseURL, endpoint, queryParams)
	body, err := executeHTTPRequest(params.Context, params, requestURL)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("alertmanager query failed: %w", err)), nil
	}

	// Format response
	result, err := prettyJSON(body)
	if err != nil {
		return api.NewToolCallResult(string(body), nil), nil
	}

	return api.NewToolCallResult(result, nil), nil
}
