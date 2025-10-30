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

	// Register all OpenShift AI tools
	tools = append(tools, t.createDataScienceProjectTools(o)...)
	tools = append(tools, t.createModelTools(o)...)
	tools = append(tools, t.createApplicationTools(o)...)
	tools = append(tools, t.createExperimentTools(o)...)
	tools = append(tools, t.createPipelineTools(o)...)

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
func (t *Toolset) createModelTools(o internalk8s.Openshift) []api.ServerTool {
	tempToolset := &ModelsToolset{}
	return tempToolset.GetTools(o)
}

// createApplicationTools creates Application management tools
func (t *Toolset) createApplicationTools(o internalk8s.Openshift) []api.ServerTool {
	tempToolset := &ApplicationsToolset{}
	return tempToolset.GetTools(o)
}

// createExperimentTools creates Experiment management tools
func (t *Toolset) createExperimentTools(o internalk8s.Openshift) []api.ServerTool {
	tempToolset := &ExperimentsToolset{}
	return tempToolset.GetTools(o)
}

// createPipelineTools creates Pipeline management tools
func (t *Toolset) createPipelineTools(o internalk8s.Openshift) []api.ServerTool {
	tempToolset := &PipelinesToolset{}
	return tempToolset.GetTools(o)
}

func init() {
	toolsets.Register(&Toolset{})
}
