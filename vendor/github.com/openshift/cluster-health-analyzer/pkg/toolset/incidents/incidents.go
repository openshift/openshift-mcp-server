// Package incidents exposes cluster health incident tools for Kubernetes MCP servers.
package incidents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/BurntSushi/toml"
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/config"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/openshift/cluster-health-analyzer/pkg/alertmanager"
	analyzer "github.com/openshift/cluster-health-analyzer/pkg/mcp"
	"github.com/openshift/cluster-health-analyzer/pkg/prom"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
)

const (
	metricsToolsetName = "observability/metrics"
	serviceCAFile      = "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
)

// backendConfig holds the Prometheus and Alertmanager URLs from a toolset config.
type backendConfig struct {
	PrometheusURL   string `toml:"prometheus_url"`
	AlertmanagerURL string `toml:"alertmanager_url"`
}

// Validate defers missing URL checks until both configuration sources have been tried.
func (*backendConfig) Validate() error {
	return nil
}

func init() {
	config.RegisterToolsetConfig(ToolsetName, func(_ context.Context, primitive toml.Primitive, md toml.MetaData) (api.ExtendedConfig, error) {
		var cfg backendConfig
		if err := md.PrimitiveDecode(primitive, &cfg); err != nil {
			return nil, err
		}
		return &cfg, nil
	})
}

// initGetIncidents creates the get_incidents tool using the existing MCP tool metadata.
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

// handleGetIncidents runs the tool using the in-cluster service CA.
func handleGetIncidents(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	return handleGetIncidentsWithCAFile(params, serviceCAFile)
}

// handleGetIncidentsWithCAFile queries the configured backends, preferring a
// complete metrics config over a complete incidents config. The supplied CA
// file is used to verify both HTTPS backends.
func handleGetIncidentsWithCAFile(params api.ToolHandlerParams, caFile string) (*api.ToolCallResult, error) {
	var cfg backendConfig
	for _, name := range []string{metricsToolsetName, ToolsetName} {
		rawConfig, ok := params.GetToolsetConfig(name)
		if !ok {
			continue
		}
		data, err := json.Marshal(rawConfig)
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("invalid %s configuration: %w", name, err)), nil
		}
		var candidate backendConfig
		if err := json.Unmarshal(data, &candidate); err != nil {
			return api.NewToolCallResult("", fmt.Errorf("invalid %s configuration: %w", name, err)), nil
		}
		if candidate.PrometheusURL != "" && candidate.AlertmanagerURL != "" {
			cfg = candidate
			break
		}
	}
	if cfg.PrometheusURL == "" || cfg.AlertmanagerURL == "" {
		return api.NewToolCallResult("",
				errors.New("get_incidents requires prometheus_url and alertmanager_url in observability/metrics or observability/incidents toolset configuration")),
			nil
	}

	args, err := parseGetIncidentsArgs(params)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	promTransport, err := backendTransport(params.RESTConfig(), cfg.PrometheusURL, caFile)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("prometheus transport: %w", err)), nil
	}
	alertTransport, err := backendTransport(params.RESTConfig(), cfg.AlertmanagerURL, caFile)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("alertmanager transport: %w", err)), nil
	}
	promLoader, err := prom.NewLoaderWithRoundTripper(cfg.PrometheusURL, promTransport)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("prometheus client: %w", err)), nil
	}
	alertLoader, err := alertmanager.NewLoader(alertmanager.LoaderConfig{
		AlertManagerURL: cfg.AlertmanagerURL,
		RoundTripper:    alertTransport,
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("alertmanager client: %w", err)), nil
	}

	incidentTool := analyzer.NewIncidentsTool("", "")
	result, _, err := incidentTool.GetIncidents(params.Context, args, promLoader, alertLoader)
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
	return api.NewToolCallResult(text.Text, nil), nil
}

// parseGetIncidentsArgs reads tool arguments and applies the shared incident
// defaults and validation.
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

// backendTransport builds an HTTPS transport with the caller's REST credentials
// and the supplied backend CA, rather than the Kubernetes API CA.
func backendTransport(restConfig *rest.Config, endpoint, caFile string) (http.RoundTripper, error) {
	if restConfig == nil {
		return nil, errors.New("no REST config available")
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	if u.Host == "" || u.Scheme != "https" {
		return nil, fmt.Errorf("backend URL %q must use HTTPS", endpoint)
	}
	cfg := rest.CopyConfig(restConfig)
	cfg.Host = endpoint
	cfg.WrapTransport = nil // The Kubernetes API wrapper must not intercept backend requests.
	cfg.Transport = nil
	cfg.Impersonate = rest.ImpersonationConfig{}
	cfg.Insecure = false
	cfg.CAData = nil
	cfg.CAFile = caFile
	return rest.TransportFor(cfg)
}
