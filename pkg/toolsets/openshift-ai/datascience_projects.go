package openshiftai

import (
	"encoding/json"
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	openshiftai "github.com/containers/kubernetes-mcp-server/pkg/openshift-ai"
	"k8s.io/client-go/rest"
)

// DataScienceProjectsToolset provides tools for managing OpenShift AI Data Science Projects
type DataScienceProjectsToolset struct {
	*openshiftai.BaseToolset
}

// NewDataScienceProjectsToolset creates a new DataScience Projects toolset
func NewDataScienceProjectsToolset(client *openshiftai.Client) *DataScienceProjectsToolset {
	base := openshiftai.NewBaseToolset(
		"datascience-projects",
		"Tools for managing OpenShift AI Data Science Projects",
		client,
	)
	return &DataScienceProjectsToolset{
		BaseToolset: base,
	}
}

// GetTools returns all tools for this toolset
func (t *DataScienceProjectsToolset) GetTools(o kubernetes.Openshift) []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.GetDataScienceProjectListTool(),
			Handler: func(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
				return t.handleDataScienceProjectsList(params)
			},
		},
		{
			Tool: api.GetDataScienceProjectGetTool(),
			Handler: func(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
				return t.handleDataScienceProjectGet(params)
			},
		},
		{
			Tool: api.GetDataScienceProjectCreateTool(),
			Handler: func(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
				return t.handleDataScienceProjectCreate(params)
			},
		},
		{
			Tool: api.GetDataScienceProjectDeleteTool(),
			Handler: func(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
				return t.handleDataScienceProjectDelete(params)
			},
		},
	}
}

// handleDataScienceProjectsList handles the datascience_projects_list tool
func (t *DataScienceProjectsToolset) handleDataScienceProjectsList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()

	// Get namespace parameter (optional)
	namespace, _ := args["namespace"].(string)

	// Get OpenShift AI client from Kubernetes manager
	clientInterface, err := params.Kubernetes.GetOrCreateOpenShiftAIClient(func(cfg *rest.Config, config interface{}) (interface{}, error) {
		return openshiftai.NewClient(cfg, nil)
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get OpenShift AI client: %w", err)), nil
	}
	openshiftAIClient := clientInterface.(*openshiftai.Client)

	// Create DataScienceProject client
	dsClient := openshiftai.NewDataScienceProjectClient(openshiftAIClient)

	// List projects
	projects, err := dsClient.List(params.Context, namespace)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list Data Science Projects: %w", err)), nil
	}

	// Convert to JSON response
	content, err := json.Marshal(projects)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal projects: %w", err)), nil
	}

	return api.NewToolCallResult(string(content), nil), nil
}

// handleDataScienceProjectGet handles the datascience_project_get tool
func (t *DataScienceProjectsToolset) handleDataScienceProjectGet(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	name, ok := args["name"].(string)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("name parameter is required")), nil
	}

	namespace, ok := args["namespace"].(string)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("namespace parameter is required")), nil
	}

	// Get OpenShift AI client from Kubernetes manager
	clientInterface, err := params.Kubernetes.GetOrCreateOpenShiftAIClient(func(cfg *rest.Config, config interface{}) (interface{}, error) {
		return openshiftai.NewClient(cfg, nil)
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get OpenShift AI client: %w", err)), nil
	}
	openshiftAIClient := clientInterface.(*openshiftai.Client)

	// Create DataScienceProject client
	dsClient := openshiftai.NewDataScienceProjectClient(openshiftAIClient)

	// Get the project
	project, err := dsClient.Get(params.Context, name, namespace)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get Data Science Project '%s' in namespace '%s': %w", name, namespace, err)), nil
	}

	// Convert to JSON response
	content, err := json.Marshal(project)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal project: %w", err)), nil
	}

	return api.NewToolCallResult(string(content), nil), nil
}

// handleDataScienceProjectCreate handles the datascience_project_create tool
func (t *DataScienceProjectsToolset) handleDataScienceProjectCreate(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	name, ok := args["name"].(string)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("name parameter is required")), nil
	}

	namespace, ok := args["namespace"].(string)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("namespace parameter is required")), nil
	}

	description, _ := args["description"].(string)
	displayName, _ := args["display_name"].(string)

	// Get optional parameters
	var labels map[string]string
	if labelsArg, exists := args["labels"]; exists {
		if labelsMap, ok := labelsArg.(map[string]any); ok {
			labels = make(map[string]string)
			for k, v := range labelsMap {
				if str, ok := v.(string); ok {
					labels[k] = str
				}
			}
		}
	}

	var annotations map[string]string
	if annotationsArg, exists := args["annotations"]; exists {
		if annotationsMap, ok := annotationsArg.(map[string]any); ok {
			annotations = make(map[string]string)
			for k, v := range annotationsMap {
				if str, ok := v.(string); ok {
					annotations[k] = str
				}
			}
		}
	}

	// Get OpenShift AI client from Kubernetes manager
	clientInterface, err := params.Kubernetes.GetOrCreateOpenShiftAIClient(func(cfg *rest.Config, config interface{}) (interface{}, error) {
		return openshiftai.NewClient(cfg, nil)
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get OpenShift AI client: %w", err)), nil
	}
	openshiftAIClient := clientInterface.(*openshiftai.Client)

	// Create DataScienceProject client
	dsClient := openshiftai.NewDataScienceProjectClient(openshiftAIClient)

	// Create project
	project := &openshiftai.DataScienceProject{
		Name:        name,
		Namespace:   namespace,
		DisplayName: &displayName,
		Description: &description,
		Labels:      labels,
		Annotations: annotations,
	}

	createdProject, err := dsClient.Create(params.Context, project)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create Data Science Project '%s' in namespace '%s': %w", name, namespace, err)), nil
	}

	// Convert to JSON response
	content, err := json.Marshal(createdProject)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal created project: %w", err)), nil
	}

	return api.NewToolCallResult(string(content), nil), nil
}

// handleDataScienceProjectDelete handles the datascience_project_delete tool
func (t *DataScienceProjectsToolset) handleDataScienceProjectDelete(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	name, ok := args["name"].(string)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("name parameter is required")), nil
	}

	namespace, ok := args["namespace"].(string)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("namespace parameter is required")), nil
	}

	// Get OpenShift AI client from Kubernetes manager
	clientInterface, err := params.Kubernetes.GetOrCreateOpenShiftAIClient(func(cfg *rest.Config, config interface{}) (interface{}, error) {
		return openshiftai.NewClient(cfg, nil)
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get OpenShift AI client: %w", err)), nil
	}
	openshiftAIClient := clientInterface.(*openshiftai.Client)

	// Create DataScienceProject client
	dsClient := openshiftai.NewDataScienceProjectClient(openshiftAIClient)

	// Delete project
	err = dsClient.Delete(params.Context, name, namespace)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete Data Science Project '%s' in namespace '%s': %w", name, namespace, err)), nil
	}

	content := fmt.Sprintf("Successfully deleted Data Science Project '%s' in namespace '%s'", name, namespace)
	return api.NewToolCallResult(content, nil), nil
}
