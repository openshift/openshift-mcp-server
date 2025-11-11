package openshiftai

import (
	"slices"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	internalk8s "github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
)

type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

func (t *Toolset) GetName() string {
	return "openshift-ai"
}

func (t *Toolset) GetDescription() string {
	return "OpenShift AI specific tools for managing Data Science Projects, Jupyter Notebooks, model serving, and pipelines"
}

func (t *Toolset) GetTools(_ internalk8s.Openshift) []api.ServerTool {
	return slices.Concat(
		initDataScienceProjects(),
		initModels(),
		initApplications(),
		initExperiments(),
		initPipelines(),
	)
}

func init() {
	toolsets.Register(&Toolset{})
}
