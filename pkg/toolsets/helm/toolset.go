package helm

import (
	"slices"

	"github.com/kiali/kiali-mcp-server/pkg/api"
	internalk8s "github.com/kiali/kiali-mcp-server/pkg/kubernetes"
	"github.com/kiali/kiali-mcp-server/pkg/toolsets"
)

type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

func (t *Toolset) GetName() string {
	return "helm"
}

func (t *Toolset) GetDescription() string {
	return "Tools for managing Helm charts and releases"
}

func (t *Toolset) GetTools(_ internalk8s.Openshift) []api.ServerTool {
	return slices.Concat(
		initHelm(),
	)
}

func init() {
	toolsets.Register(&Toolset{})
}
