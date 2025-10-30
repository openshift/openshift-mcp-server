package openshiftai

import (
	"encoding/json"
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	openshiftai "github.com/containers/kubernetes-mcp-server/pkg/openshift-ai"
	"k8s.io/client-go/rest"
)

// PipelinesToolset provides tools for managing OpenShift AI Data Science Pipelines
type PipelinesToolset struct {
	*openshiftai.BaseToolset
}

// NewPipelinesToolset creates a new Pipelines toolset
func NewPipelinesToolset(client *openshiftai.Client) *PipelinesToolset {
	base := openshiftai.NewBaseToolset(
		"pipelines",
		"Tools for managing OpenShift AI data science pipelines",
		client,
	)
	return &PipelinesToolset{
		BaseToolset: base,
	}
}

// GetTools returns all tools for this toolset
func (t *PipelinesToolset) GetTools(o kubernetes.Openshift) []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.GetPipelinesListTool(),
			Handler: func(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
				return t.handlePipelinesList(params)
			},
		},
		{
			Tool: api.GetPipelineGetTool(),
			Handler: func(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
				return t.handlePipelineGet(params)
			},
		},
		{
			Tool: api.GetPipelineCreateTool(),
			Handler: func(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
				return t.handlePipelineCreate(params)
			},
		},
		{
			Tool: api.GetPipelineDeleteTool(),
			Handler: func(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
				return t.handlePipelineDelete(params)
			},
		},
		{
			Tool: api.GetPipelineRunsListTool(),
			Handler: func(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
				return t.handlePipelineRunsList(params)
			},
		},
		{
			Tool: api.GetPipelineRunGetTool(),
			Handler: func(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
				return t.handlePipelineRunGet(params)
			},
		},
	}
}

// handlePipelinesList handles pipelines_list tool
func (t *PipelinesToolset) handlePipelinesList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()

	// Get namespace parameter (optional)
	namespace, _ := args["namespace"].(string)
	// Get status filter (optional)
	status, _ := args["status"].(string)

	// Get OpenShift AI client from Kubernetes manager
	clientInterface, err := params.Kubernetes.GetOrCreateOpenShiftAIClient(func(cfg *rest.Config, config interface{}) (interface{}, error) {
		return openshiftai.NewClient(cfg, nil)
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get OpenShift AI client: %w", err)), nil
	}
	openshiftAIClient := clientInterface.(*openshiftai.Client)

	// Create Pipeline client
	pipelineClient := openshiftai.NewPipelineClient(openshiftAIClient)

	// Build filters
	filters := make(map[string]string)
	if status != "" {
		filters["status"] = status
	}

	// List pipelines
	pipelines, err := pipelineClient.List(params.Context, namespace, filters)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list pipelines: %w", err)), nil
	}

	// Convert to response format
	response := make([]map[string]interface{}, len(pipelines))
	for i, pipeline := range pipelines {
		response[i] = map[string]interface{}{
			"name":         pipeline.Name,
			"namespace":    pipeline.Namespace,
			"display_name": pipeline.DisplayName,
			"description":  pipeline.Description,
			"labels":       pipeline.Labels,
			"annotations":  pipeline.Annotations,
			"status":       pipeline.Status,
		}
	}

	result := map[string]interface{}{
		"pipelines": response,
		"count":     len(response),
	}

	// Convert to JSON response
	content, err := json.Marshal(result)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal result: %w", err)), nil
	}

	return api.NewToolCallResult(string(content), nil), nil
}

// handlePipelineGet handles pipeline_get tool
func (t *PipelinesToolset) handlePipelineGet(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()

	name, _ := args["name"].(string)
	namespace, _ := args["namespace"].(string)

	if name == "" {
		return api.NewToolCallResult("", fmt.Errorf("pipeline name is required")), nil
	}
	if namespace == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace is required")), nil
	}

	// Get OpenShift AI client from Kubernetes manager
	clientInterface, err := params.Kubernetes.GetOrCreateOpenShiftAIClient(func(cfg *rest.Config, config interface{}) (interface{}, error) {
		return openshiftai.NewClient(cfg, nil)
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get OpenShift AI client: %w", err)), nil
	}
	openshiftAIClient := clientInterface.(*openshiftai.Client)

	// Create Pipeline client
	pipelineClient := openshiftai.NewPipelineClient(openshiftAIClient)

	pipeline, err := pipelineClient.Get(params.Context, namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get pipeline %s: %w", name, err)), nil
	}

	result := map[string]interface{}{
		"name":         pipeline.Name,
		"namespace":    pipeline.Namespace,
		"display_name": pipeline.DisplayName,
		"description":  pipeline.Description,
		"labels":       pipeline.Labels,
		"annotations":  pipeline.Annotations,
		"status":       pipeline.Status,
	}

	// Convert to JSON response
	content, err := json.Marshal(result)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal result: %w", err)), nil
	}

	return api.NewToolCallResult(string(content), nil), nil
}

// handlePipelineCreate handles pipeline_create tool
func (t *PipelinesToolset) handlePipelineCreate(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()

	name, _ := args["name"].(string)
	namespace, _ := args["namespace"].(string)
	displayName, _ := args["display_name"].(string)
	description, _ := args["description"].(string)
	labels, _ := args["labels"].(map[string]interface{})
	annotations, _ := args["annotations"].(map[string]interface{})

	if name == "" {
		return api.NewToolCallResult("", fmt.Errorf("pipeline name is required")), nil
	}
	if namespace == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace is required")), nil
	}

	// Get OpenShift AI client from Kubernetes manager
	clientInterface, err := params.Kubernetes.GetOrCreateOpenShiftAIClient(func(cfg *rest.Config, config interface{}) (interface{}, error) {
		return openshiftai.NewClient(cfg, nil)
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get OpenShift AI client: %w", err)), nil
	}
	openshiftAIClient := clientInterface.(*openshiftai.Client)

	// Create Pipeline client
	pipelineClient := openshiftai.NewPipelineClient(openshiftAIClient)

	// Convert labels and annotations
	labelsMap := make(map[string]string)
	for k, v := range labels {
		if s, ok := v.(string); ok {
			labelsMap[k] = s
		}
	}

	annotationsMap := make(map[string]string)
	for k, v := range annotations {
		if s, ok := v.(string); ok {
			annotationsMap[k] = s
		}
	}

	pipeline := &api.Pipeline{
		Name:        name,
		Namespace:   namespace,
		Labels:      labelsMap,
		Annotations: annotationsMap,
	}

	if displayName != "" {
		pipeline.DisplayName = &displayName
	}
	if description != "" {
		pipeline.Description = &description
	}

	createdPipeline, err := pipelineClient.Create(params.Context, namespace, pipeline)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create pipeline: %w", err)), nil
	}

	result := map[string]interface{}{
		"name":         createdPipeline.Name,
		"namespace":    createdPipeline.Namespace,
		"display_name": createdPipeline.DisplayName,
		"description":  createdPipeline.Description,
		"labels":       createdPipeline.Labels,
		"annotations":  createdPipeline.Annotations,
		"status":       createdPipeline.Status,
		"message":      "Pipeline created successfully",
	}

	// Convert to JSON response
	content, err := json.Marshal(result)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal result: %w", err)), nil
	}

	return api.NewToolCallResult(string(content), nil), nil
}

// handlePipelineDelete handles pipeline_delete tool
func (t *PipelinesToolset) handlePipelineDelete(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()

	name, _ := args["name"].(string)
	namespace, _ := args["namespace"].(string)

	if name == "" {
		return api.NewToolCallResult("", fmt.Errorf("pipeline name is required")), nil
	}
	if namespace == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace is required")), nil
	}

	// Get OpenShift AI client from Kubernetes manager
	clientInterface, err := params.Kubernetes.GetOrCreateOpenShiftAIClient(func(cfg *rest.Config, config interface{}) (interface{}, error) {
		return openshiftai.NewClient(cfg, nil)
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get OpenShift AI client: %w", err)), nil
	}
	openshiftAIClient := clientInterface.(*openshiftai.Client)

	// Create Pipeline client
	pipelineClient := openshiftai.NewPipelineClient(openshiftAIClient)

	err = pipelineClient.Delete(params.Context, namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete pipeline %s: %w", name, err)), nil
	}

	result := map[string]interface{}{
		"name":    name,
		"message": "Pipeline deleted successfully",
	}

	// Convert to JSON response
	content, err := json.Marshal(result)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal result: %w", err)), nil
	}

	return api.NewToolCallResult(string(content), nil), nil
}

// handlePipelineRunsList handles pipeline_runs_list tool
func (t *PipelinesToolset) handlePipelineRunsList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()

	// Get namespace parameter (optional)
	namespace, _ := args["namespace"].(string)
	// Get pipeline name filter (optional)
	pipelineName, _ := args["pipeline_name"].(string)
	// Get status filter (optional)
	status, _ := args["status"].(string)

	// Get OpenShift AI client from Kubernetes manager
	clientInterface, err := params.Kubernetes.GetOrCreateOpenShiftAIClient(func(cfg *rest.Config, config interface{}) (interface{}, error) {
		return openshiftai.NewClient(cfg, nil)
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get OpenShift AI client: %w", err)), nil
	}
	openshiftAIClient := clientInterface.(*openshiftai.Client)

	// Create Pipeline client
	pipelineClient := openshiftai.NewPipelineClient(openshiftAIClient)

	// Build filters
	filters := make(map[string]string)
	if status != "" {
		filters["status"] = status
	}
	if pipelineName != "" {
		filters["pipeline_name"] = pipelineName
	}

	// List pipeline runs
	pipelineRuns, err := pipelineClient.ListRuns(params.Context, namespace, filters)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list pipeline runs: %w", err)), nil
	}

	// Convert to response format
	response := make([]map[string]interface{}, len(pipelineRuns))
	for i, pipelineRun := range pipelineRuns {
		response[i] = map[string]interface{}{
			"name":          pipelineRun.Name,
			"pipeline_name": pipelineRun.PipelineName,
			"namespace":     pipelineRun.Namespace,
			"display_name":  pipelineRun.DisplayName,
			"description":   pipelineRun.Description,
			"labels":        pipelineRun.Labels,
			"annotations":   pipelineRun.Annotations,
			"status":        pipelineRun.Status,
		}
	}

	result := map[string]interface{}{
		"pipeline_runs": response,
		"count":         len(response),
	}

	// Convert to JSON response
	content, err := json.Marshal(result)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal result: %w", err)), nil
	}

	return api.NewToolCallResult(string(content), nil), nil
}

// handlePipelineRunGet handles pipeline_run_get tool
func (t *PipelinesToolset) handlePipelineRunGet(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()

	name, _ := args["name"].(string)
	namespace, _ := args["namespace"].(string)

	if name == "" {
		return api.NewToolCallResult("", fmt.Errorf("pipeline run name is required")), nil
	}
	if namespace == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace is required")), nil
	}

	// Get OpenShift AI client from Kubernetes manager
	clientInterface, err := params.Kubernetes.GetOrCreateOpenShiftAIClient(func(cfg *rest.Config, config interface{}) (interface{}, error) {
		return openshiftai.NewClient(cfg, nil)
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get OpenShift AI client: %w", err)), nil
	}
	openshiftAIClient := clientInterface.(*openshiftai.Client)

	// Create Pipeline client
	pipelineClient := openshiftai.NewPipelineClient(openshiftAIClient)

	pipelineRun, err := pipelineClient.GetRun(params.Context, namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get pipeline run %s: %w", name, err)), nil
	}

	result := map[string]interface{}{
		"name":          pipelineRun.Name,
		"pipeline_name": pipelineRun.PipelineName,
		"namespace":     pipelineRun.Namespace,
		"display_name":  pipelineRun.DisplayName,
		"description":   pipelineRun.Description,
		"labels":        pipelineRun.Labels,
		"annotations":   pipelineRun.Annotations,
		"status":        pipelineRun.Status,
	}

	// Convert to JSON response
	content, err := json.Marshal(result)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal result: %w", err)), nil
	}

	return api.NewToolCallResult(string(content), nil), nil
}
