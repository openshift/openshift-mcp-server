package openshiftai

import (
	"encoding/json"
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	openshiftai "github.com/containers/kubernetes-mcp-server/pkg/openshift-ai"
	"k8s.io/client-go/rest"
)

func initApplications() []api.ServerTool {
	return []api.ServerTool{
		{Tool: api.GetApplicationsListTool(), Handler: applicationsList},
		{Tool: api.GetApplicationGetTool(), Handler: applicationsGet},
		{Tool: api.GetApplicationCreateTool(), Handler: applicationsCreate},
		{Tool: api.GetApplicationDeleteTool(), Handler: applicationsDelete},
	}
}
func applicationsList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()

	// Get namespace parameter (optional)
	namespace, _ := args["namespace"].(string)
	// Get status filter (optional)
	status, _ := args["status"].(string)
	// Get app type filter (optional)
	appType, _ := args["app_type"].(string)

	// Get OpenShift AI client from Kubernetes manager
	clientInterface, err := params.GetOrCreateOpenShiftAIClient(func(cfg *rest.Config, config interface{}) (interface{}, error) {
		return openshiftai.NewClient(cfg, nil)
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get OpenShift AI client: %w", err)), nil
	}
	openshiftAIClient := clientInterface.(*openshiftai.Client)

	// Create Application client
	applicationClient := openshiftai.NewApplicationClient(openshiftAIClient)

	// List applications
	applications, err := applicationClient.List(params.Context, namespace, status, appType)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list applications: %w", err)), nil
	}

	// Convert to JSON response
	content, err := json.Marshal(applications)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal applications: %w", err)), nil
	}

	return api.NewToolCallResult(string(content), nil), nil
}
func applicationsGet(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
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

	// Create Application client
	applicationClient := openshiftai.NewApplicationClient(openshiftAIClient)

	// Get application
	application, err := applicationClient.Get(params.Context, name, namespace)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get application '%s' in namespace '%s': %w", name, namespace, err)), nil
	}

	// Convert to JSON response
	content, err := json.Marshal(application)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal application: %w", err)), nil
	}

	return api.NewToolCallResult(string(content), nil), nil
}
func applicationsCreate(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	name, ok := args["name"].(string)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("name parameter is required")), nil
	}

	namespace, ok := args["namespace"].(string)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("namespace parameter is required")), nil
	}

	appType, ok := args["app_type"].(string)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("app_type parameter is required")), nil
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

	// Create Application client
	applicationClient := openshiftai.NewApplicationClient(openshiftAIClient)

	// Create application
	application := &api.Application{
		Name:        name,
		Namespace:   namespace,
		DisplayName: displayName,
		Description: description,
		AppType:     appType,
		Labels:      labels,
		Annotations: annotations,
		Status: api.ApplicationStatus{
			Phase: "Creating",
			Ready: false,
		},
	}

	createdApplication, err := applicationClient.Create(params.Context, application)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create application: %w", err)), nil
	}

	// Convert to JSON response
	content, err := json.Marshal(createdApplication)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal created application: %w", err)), nil
	}

	return api.NewToolCallResult(string(content), nil), nil
}
func applicationsDelete(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
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

	// Create Application client
	applicationClient := openshiftai.NewApplicationClient(openshiftAIClient)

	// Delete application
	err = applicationClient.Delete(params.Context, name, namespace)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete application '%s' in namespace '%s': %w", name, namespace, err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("Application %s/%s deleted successfully", namespace, name), nil), nil
}
