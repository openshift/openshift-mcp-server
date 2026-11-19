package netedge

import (
	"errors"
	"slices"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/config"
	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/mustgather"
	"github.com/google/jsonschema-go/jsonschema"
)

func offlineEnabled(cfg *config.Config) bool {
	return cfg != nil && slices.Contains(cfg.Toolsets.Get(), "openshift/mustgather")
}

// withOfflineSupport advertises archive selection only when must-gather is enabled.
func withOfflineSupport(tools []api.ServerTool, cfg *config.Config) []api.ServerTool {
	if !offlineEnabled(cfg) {
		return tools
	}
	for i := range tools {
		tool := &tools[i].Tool
		tool.Description += " Set archive_id to read a must-gather archive instead of the live cluster."
		if tool.InputSchema.Properties == nil {
			tool.InputSchema.Properties = make(map[string]*jsonschema.Schema)
		}
		tool.InputSchema.Properties["archive_id"] = &jsonschema.Schema{
			Type:        "string",
			Description: "Must-gather archive ID from mustgather_list. Omit for the live cluster.",
		}
	}
	return tools
}

// selectedArchive returns a provider and true when an archive is resolved.
// A missing ID returns false to request the live cluster. Callers must check
// the error first: an invalid ID must never fall back to the live cluster.
func selectedArchive(params api.ToolHandlerParams) (*mg.Provider, bool, error) {
	p := api.WrapParams(params)
	id := p.OptionalString("archive_id", "")
	if err := p.Err(); err != nil {
		return nil, false, err
	}
	if id == "" {
		return nil, false, nil
	}
	if !offlineEnabled(params.Config) {
		return nil, false, errors.New("archive_id requires the openshift/mustgather toolset to be enabled")
	}
	provider, err := mustgather.ProviderForArchive(params, id)
	return provider, err == nil, err
}
