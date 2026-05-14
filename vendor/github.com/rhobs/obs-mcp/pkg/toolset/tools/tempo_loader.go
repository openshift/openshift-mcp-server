package tools

import (
	"fmt"
	"net/http"

	"github.com/containers/kubernetes-mcp-server/pkg/api"

	"github.com/rhobs/obs-mcp/pkg/traces"
	tempoclient "github.com/rhobs/obs-mcp/pkg/traces/tempo"
)

// NewTempoLoader creates a Tempo loader using the tempo toolset configuration.
func NewTempoLoader(params api.ToolHandlerParams, url string) (tempoclient.Loader, error) {
	cfg := traces.GetConfig(params)

	apiConfig, err := buildAPIConfig(params, url, cfg.Insecure, cfg.GetAuthMode())
	if err != nil {
		return nil, fmt.Errorf("failed to create API config: %w", err)
	}

	httpClient := &http.Client{
		Timeout:   tempoclient.RequestTimeout,
		Transport: apiConfig.RoundTripper,
	}
	return tempoclient.NewTempoLoader(httpClient, url), nil
}
