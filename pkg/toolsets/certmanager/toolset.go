// Package certmanager provides MCP tools for managing cert-manager resources.
package certmanager

import (
	"slices"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	internalk8s "github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
)

// Toolset implements the cert-manager MCP toolset
type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

// GetName returns the name of the toolset
func (t *Toolset) GetName() string {
	return "certmanager"
}

// GetDescription returns a description of the toolset
func (t *Toolset) GetDescription() string {
	return "Tools for managing cert-manager certificates, issuers, and troubleshooting TLS certificate issues"
}

// GetTools returns all tools in this toolset
func (t *Toolset) GetTools(_ internalk8s.Openshift) []api.ServerTool {
	return slices.Concat(
		initCertificates(),
		initIssuers(),
		initTroubleshoot(),
		initOperator(),
		initLogs(),
	)
}

func init() {
	toolsets.Register(&Toolset{})
}

