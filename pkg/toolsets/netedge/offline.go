package netedge

import (
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/mustgather"
)

// selectedArchive returns the explicitly selected archive. A missing ID means
// the caller requested the live cluster; an invalid ID never falls back to it.
func selectedArchive(params api.ToolHandlerParams) (*mg.Provider, error) {
	p := api.WrapParams(params)
	id := p.OptionalString("archive_id", "")
	if err := p.Err(); err != nil {
		return nil, err
	}
	if id == "" {
		return nil, nil
	}
	return mustgather.ProviderForArchive(params, id)
}
