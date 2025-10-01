package config

import (
	"slices"

	"github.com/kiali/kiali-mcp-server/pkg/api"
	internalk8s "github.com/kiali/kiali-mcp-server/pkg/kubernetes"
	"github.com/kiali/kiali-mcp-server/pkg/toolsets"
)

type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

func (t *Toolset) GetName() string {
	return "config"
}

func (t *Toolset) GetDescription() string {
	return "View and manage the current local Kubernetes configuration (kubeconfig)"
}

func (t *Toolset) GetTools(_ internalk8s.Openshift) []api.ServerTool {
	return slices.Concat(
		initConfiguration(),
	)
}

func init() {
	toolsets.Register(&Toolset{})
}
