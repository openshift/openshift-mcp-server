package gitops

import (
	"slices"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
)

type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

func (t *Toolset) GetName() string {
	return "gitops"
}

func (t *Toolset) GetDescription() string {
	return "ArgoCD and OpenShift GitOps application management"
}

func (t *Toolset) GetTools(_ api.Openshift) []api.ServerTool {
	return slices.Concat(
		initApplications(),
		initProjects(),
	)
}

func (t *Toolset) GetPrompts() []api.ServerPrompt {
	return nil
}

func init() {
	toolsets.Register(&Toolset{})
}
