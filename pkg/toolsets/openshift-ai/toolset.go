package openshiftai

import (
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	internalk8s "github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
)

const (
	ToolsetName        = "openshift-ai"
	ToolsetDescription = "OpenShift AI specific tools for managing Data Science Projects, Jupyter Notebooks, model serving, and pipelines"
)

// Toolset represents the OpenShift AI toolset
type Toolset struct{}

// GetName returns the name of the toolset
func (t *Toolset) GetName() string {
	return ToolsetName
}

// GetDescription returns the description of the toolset
func (t *Toolset) GetDescription() string {
	return ToolsetDescription
}

// GetTools returns all available tools for this toolset
func (t *Toolset) GetTools(o internalk8s.Openshift) []api.ServerTool {
	tools := []api.ServerTool{}

	// For initial integration, only Data Science Project tools are registered.
	// Additional OpenShift AI tools (models, experiments, applications, pipelines)
	// can be enabled in future PRs once test data and docs are updated.
	tools = append(tools, t.createDataScienceProjectTools(o)...)

	return tools
}

// createDataScienceProjectTools creates Data Science Project management tools
func (t *Toolset) createDataScienceProjectTools(o internalk8s.Openshift) []api.ServerTool {
	// Create a temporary DataScienceProjectsToolset to get the tools
	// We'll pass nil for the client since it will be created dynamically
	tempToolset := &DataScienceProjectsToolset{}
	return tempToolset.GetTools(o)
}

// createModelTools creates Model management tools
// The following helpers for additional OpenShift AI tools are intentionally
// omitted from registration for now to keep the scope minimal and tests stable.

func init() {
	toolsets.Register(&Toolset{})
}
