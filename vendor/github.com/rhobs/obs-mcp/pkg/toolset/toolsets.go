package toolset

import (
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"

	"github.com/rhobs/obs-mcp/pkg/logs"
	metrics "github.com/rhobs/obs-mcp/pkg/metrics/toolset"
	"github.com/rhobs/obs-mcp/pkg/otelcol"
	"github.com/rhobs/obs-mcp/pkg/traces"
)

func init() {
	toolsets.Register(&metrics.Toolset{})
	toolsets.Register(&logs.Toolset{})
	toolsets.Register(&traces.Toolset{})
	toolsets.Register(&otelcol.Toolset{})
}
