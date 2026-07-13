package tools

import (
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"

	"github.com/rhobs/obs-mcp/pkg/logs"
	"github.com/rhobs/obs-mcp/pkg/logs/loki"
)

const defaultLokiURL = "http://localhost:3100"

// NewLokiLoader creates a Loki loader using the logs toolset configuration.
func NewLokiLoader(params api.ToolHandlerParams, url, tenant string) (loki.Loader, error) {
	cfg := logs.GetConfig(params)
	lokiURL := url
	if lokiURL == "" {
		lokiURL = cfg.LokiURL
	}
	if lokiURL == "" {
		lokiURL = defaultLokiURL
	}

	apiConfig, err := buildAPIConfig(params, lokiURL, cfg.Insecure, cfg.GetAuthMode())
	if err != nil {
		return nil, fmt.Errorf("failed to create API config: %w", err)
	}

	loader, err := loki.NewLoader(apiConfig, tenant)
	if err != nil {
		return nil, fmt.Errorf("failed to create Loki loader: %w", err)
	}
	return loader, nil
}
