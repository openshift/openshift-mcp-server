package openshiftai

import (
	"encoding/json"
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	openshiftai "github.com/containers/kubernetes-mcp-server/pkg/openshift-ai"
	"k8s.io/client-go/rest"
)

func initModels() []api.ServerTool {
	return []api.ServerTool{
		{Tool: api.GetModelListTool(), Handler: modelsList},
		{Tool: api.GetModelGetTool(), Handler: modelsGet},
		{Tool: api.GetModelCreateTool(), Handler: modelsCreate},
		{Tool: api.GetModelUpdateTool(), Handler: modelsUpdate},
		{Tool: api.GetModelDeleteTool(), Handler: modelsDelete},
	}
}

func modelsList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()

	// Get namespace parameter (optional)
	namespace, _ := args["namespace"].(string)
	// Get model type filter (optional)
	modelType, _ := args["model_type"].(string)
	// Get status filter (optional)
	status, _ := args["status"].(string)

	// Get OpenShift AI client from Kubernetes manager
	clientInterface, err := params.GetOrCreateOpenShiftAIClient(func(cfg *rest.Config, config interface{}) (interface{}, error) {
		return openshiftai.NewClient(cfg, nil)
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get OpenShift AI client: %w", err)), nil
	}
	openshiftAIClient := clientInterface.(*openshiftai.Client)

	// Create Model client
	modelClient := openshiftai.NewModelClient(openshiftAIClient)

	// List models
	models, err := modelClient.List(params.Context, namespace, modelType, status)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list models: %w", err)), nil
	}

	// Convert to JSON response
	content, err := json.Marshal(models)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal models: %w", err)), nil
	}

	return api.NewToolCallResult(string(content), nil), nil
}

// handleModelGet handles model_get tool
func modelsGet(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
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
	clientInterface, err := params.GetOrCreateOpenShiftAIClient(func(cfg *rest.Config, config interface{}) (interface{}, error) {
		return openshiftai.NewClient(cfg, nil)
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get OpenShift AI client: %w", err)), nil
	}
	openshiftAIClient := clientInterface.(*openshiftai.Client)

	// Create Model client
	modelClient := openshiftai.NewModelClient(openshiftAIClient)

	// Get the model
	model, err := modelClient.Get(params.Context, name, namespace)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get model '%s' in namespace '%s': %w", name, namespace, err)), nil
	}

	// Convert to JSON response
	content, err := json.Marshal(model)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal model: %w", err)), nil
	}

	return api.NewToolCallResult(string(content), nil), nil
}

// handleModelCreate handles model_create tool
func modelsCreate(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	name, ok := args["name"].(string)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("name parameter is required")), nil
	}

	namespace, ok := args["namespace"].(string)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("namespace parameter is required")), nil
	}

	modelType, ok := args["model_type"].(string)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("model_type parameter is required")), nil
	}

	format, ok := args["format"].(string)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("format parameter is required")), nil
	}

	// Get optional parameters
	displayName, _ := args["display_name"].(string)
	description, _ := args["description"].(string)
	frameworkVersion, _ := args["framework_version"].(string)
	version, _ := args["version"].(string)

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
	clientInterface, err := params.GetOrCreateOpenShiftAIClient(func(cfg *rest.Config, config interface{}) (interface{}, error) {
		return openshiftai.NewClient(cfg, nil)
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get OpenShift AI client: %w", err)), nil
	}
	openshiftAIClient := clientInterface.(*openshiftai.Client)

	// Create Model client
	modelClient := openshiftai.NewModelClient(openshiftAIClient)

	// Create model
	model := &api.Model{
		Name:             name,
		Namespace:        namespace,
		DisplayName:      &displayName,
		Description:      &description,
		ModelType:        &modelType,
		FrameworkVersion: &frameworkVersion,
		Format:           &format,
		Version:          &version,
		Labels:           labels,
		Annotations:      annotations,
	}

	createdModel, err := modelClient.Create(params.Context, model)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create model '%s' in namespace '%s': %w", name, namespace, err)), nil
	}

	// Convert to JSON response
	content, err := json.Marshal(createdModel)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal created model: %w", err)), nil
	}

	return api.NewToolCallResult(string(content), nil), nil
}

// handleModelUpdate handles model_update tool
func modelsUpdate(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	name, ok := args["name"].(string)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("name parameter is required")), nil
	}

	namespace, ok := args["namespace"].(string)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("namespace parameter is required")), nil
	}

	// Get optional parameters
	displayName, _ := args["display_name"].(string)
	description, _ := args["description"].(string)

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
	clientInterface, err := params.GetOrCreateOpenShiftAIClient(func(cfg *rest.Config, config interface{}) (interface{}, error) {
		return openshiftai.NewClient(cfg, nil)
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get OpenShift AI client: %w", err)), nil
	}
	openshiftAIClient := clientInterface.(*openshiftai.Client)

	// Create Model client
	modelClient := openshiftai.NewModelClient(openshiftAIClient)

	// Update model
	model := &api.Model{
		Name:        name,
		Namespace:   namespace,
		DisplayName: &displayName,
		Description: &description,
		Labels:      labels,
		Annotations: annotations,
	}

	updatedModel, err := modelClient.Update(params.Context, model)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to update model '%s' in namespace '%s': %w", name, namespace, err)), nil
	}

	// Convert to JSON response
	content, err := json.Marshal(updatedModel)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal updated model: %w", err)), nil
	}

	return api.NewToolCallResult(string(content), nil), nil
}

// handleModelDelete handles model_delete tool
func modelsDelete(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
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
	clientInterface, err := params.GetOrCreateOpenShiftAIClient(func(cfg *rest.Config, config interface{}) (interface{}, error) {
		return openshiftai.NewClient(cfg, nil)
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get OpenShift AI client: %w", err)), nil
	}
	openshiftAIClient := clientInterface.(*openshiftai.Client)

	// Create Model client
	modelClient := openshiftai.NewModelClient(openshiftAIClient)

	// Delete model
	err = modelClient.Delete(params.Context, name, namespace)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete model '%s' in namespace '%s': %w", name, namespace, err)), nil
	}

	content := fmt.Sprintf("Successfully deleted model '%s' in namespace '%s'", name, namespace)
	return api.NewToolCallResult(content, nil), nil
}
