package openshiftai

import (
	"encoding/json"
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	openshiftai "github.com/containers/kubernetes-mcp-server/pkg/openshift-ai"
	"k8s.io/client-go/rest"
)

func initExperiments() []api.ServerTool {
	return []api.ServerTool{
		{Tool: api.GetExperimentsListTool(), Handler: experimentsList},
		{Tool: api.GetExperimentGetTool(), Handler: experimentsGet},
		{Tool: api.GetExperimentCreateTool(), Handler: experimentsCreate},
		{Tool: api.GetExperimentDeleteTool(), Handler: experimentsDelete},
	}
}
func experimentsList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()

	// Get namespace parameter (optional)
	namespace, _ := args["namespace"].(string)
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

	// Create Experiment client
	experimentClient := openshiftai.NewExperimentClient(openshiftAIClient)

	// List experiments
	experiments, err := experimentClient.List(params.Context, namespace, status)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list experiments: %w", err)), nil
	}

	// Convert to JSON response
	content, err := json.Marshal(experiments)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal experiments: %w", err)), nil
	}

	return api.NewToolCallResult(string(content), nil), nil
}
func experimentsGet(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
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

	// Create Experiment client
	experimentClient := openshiftai.NewExperimentClient(openshiftAIClient)

	// Get experiment
	experiment, err := experimentClient.Get(params.Context, name, namespace)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get experiment '%s' in namespace '%s': %w", name, namespace, err)), nil
	}

	// Convert to JSON response
	content, err := json.Marshal(experiment)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal experiment: %w", err)), nil
	}

	return api.NewToolCallResult(string(content), nil), nil
}
func experimentsCreate(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	name, ok := args["name"].(string)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("name parameter is required")), nil
	}

	namespace, ok := args["namespace"].(string)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("namespace parameter is required")), nil
	}

	// Optional parameters
	var displayName *string
	if val, exists := args["display_name"]; exists {
		if str, ok := val.(string); ok && str != "" {
			displayName = &str
		}
	}

	var description *string
	if val, exists := args["description"]; exists {
		if str, ok := val.(string); ok && str != "" {
			description = &str
		}
	}

	var labels map[string]string
	if val, exists := args["labels"]; exists {
		if m, ok := val.(map[string]interface{}); ok {
			labels = make(map[string]string)
			for k, v := range m {
				if str, ok := v.(string); ok {
					labels[k] = str
				}
			}
		}
	}

	var annotations map[string]string
	if val, exists := args["annotations"]; exists {
		if m, ok := val.(map[string]interface{}); ok {
			annotations = make(map[string]string)
			for k, v := range m {
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

	// Create Experiment client
	experimentClient := openshiftai.NewExperimentClient(openshiftAIClient)

	// Create experiment
	experiment := &api.Experiment{
		Name:        name,
		Namespace:   namespace,
		DisplayName: displayName,
		Description: description,
		Labels:      labels,
		Annotations: annotations,
		Status: api.ExperimentStatus{
			Phase:    "Created",
			Ready:    false,
			RunCount: 0,
		},
	}

	createdExperiment, err := experimentClient.Create(params.Context, experiment)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create experiment: %w", err)), nil
	}

	// Convert to JSON response
	content, err := json.Marshal(createdExperiment)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal created experiment: %w", err)), nil
	}

	return api.NewToolCallResult(string(content), nil), nil
}
func experimentsDelete(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
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

	// Create Experiment client
	experimentClient := openshiftai.NewExperimentClient(openshiftAIClient)

	// Delete experiment
	err = experimentClient.Delete(params.Context, name, namespace)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete experiment '%s' in namespace '%s': %w", name, namespace, err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("Experiment %s/%s deleted successfully", namespace, name), nil), nil
}
