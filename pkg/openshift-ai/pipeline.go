package openshiftai

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
)

// PipelineClient handles Pipeline operations
type PipelineClient struct {
	client *Client
}

// NewPipelineClient creates a new Pipeline client
func NewPipelineClient(client *Client) *PipelineClient {
	return &PipelineClient{
		client: client,
	}
}

// ListPipelines retrieves all pipelines with optional filtering
func (c *PipelineClient) List(ctx context.Context, namespace string, filters map[string]string) ([]*api.Pipeline, error) {
	gvr, err := c.client.GetGVR("pipelines")
	if err != nil {
		return nil, err
	}

	// Build list options with label selector from filters
	listOpts := metav1.ListOptions{}
	if len(filters) > 0 {
		labelSelector := ""
		for k, v := range filters {
			if labelSelector != "" {
				labelSelector += ","
			}
			labelSelector += k + "=" + v
		}
		listOpts.LabelSelector = labelSelector
	}

	resourceInterface := c.client.GetDynamicClient().Resource(gvr)
	var list *unstructured.UnstructuredList

	if namespace != "" {
		list, err = resourceInterface.Namespace(namespace).List(ctx, listOpts)
	} else {
		list, err = resourceInterface.List(ctx, listOpts)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list pipelines: %w", err)
	}

	pipelines := make([]*api.Pipeline, 0, len(list.Items))
	for _, item := range list.Items {
		pipeline, err := c.unstructuredToPipeline(&item)
		if err != nil {
			return nil, fmt.Errorf("failed to convert pipeline: %w", err)
		}
		pipelines = append(pipelines, pipeline)
	}

	return pipelines, nil
}

// GetPipeline retrieves a specific pipeline by name
func (c *PipelineClient) Get(ctx context.Context, namespace, name string) (*api.Pipeline, error) {
	gvr, err := c.client.GetGVR("pipelines")
	if err != nil {
		return nil, err
	}

	resourceInterface := c.client.GetDynamicClient().Resource(gvr)
	var item *unstructured.Unstructured

	if namespace != "" {
		item, err = resourceInterface.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	} else {
		item, err = resourceInterface.Get(ctx, name, metav1.GetOptions{})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline %s: %w", name, err)
	}

	return c.unstructuredToPipeline(item)
}

// CreatePipeline creates a new pipeline
func (c *PipelineClient) Create(ctx context.Context, namespace string, pipeline *api.Pipeline) (*api.Pipeline, error) {
	gvr, err := c.client.GetGVR("pipelines")
	if err != nil {
		return nil, err
	}

	resourceInterface := c.client.GetDynamicClient().Resource(gvr)

	// Convert to unstructured
	unstructuredObj, err := c.pipelineToUnstructured(pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to convert pipeline to unstructured: %w", err)
	}

	unstructuredObj.SetNamespace(namespace)

	var created *unstructured.Unstructured
	if namespace != "" {
		created, err = resourceInterface.Namespace(namespace).Create(ctx, unstructuredObj, metav1.CreateOptions{})
	} else {
		created, err = resourceInterface.Create(ctx, unstructuredObj, metav1.CreateOptions{})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create pipeline: %w", err)
	}

	return c.unstructuredToPipeline(created)
}

// DeletePipeline deletes a pipeline by name
func (c *PipelineClient) Delete(ctx context.Context, namespace, name string) error {
	gvr, err := c.client.GetGVR("pipelines")
	if err != nil {
		return err
	}

	resourceInterface := c.client.GetDynamicClient().Resource(gvr)

	if namespace != "" {
		err = resourceInterface.Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	} else {
		err = resourceInterface.Delete(ctx, name, metav1.DeleteOptions{})
	}

	if err != nil {
		return fmt.Errorf("failed to delete pipeline %s: %w", name, err)
	}

	return nil
}

// ListPipelineRuns retrieves all pipeline runs with optional filtering
func (c *PipelineClient) ListRuns(ctx context.Context, namespace string, filters map[string]string) ([]*api.PipelineRun, error) {
	gvr, err := c.client.GetGVR("pipelineruns")
	if err != nil {
		return nil, err
	}

	// Build list options with label selector from filters
	listOpts := metav1.ListOptions{}
	if len(filters) > 0 {
		labelSelector := ""
		for k, v := range filters {
			if labelSelector != "" {
				labelSelector += ","
			}
			labelSelector += k + "=" + v
		}
		listOpts.LabelSelector = labelSelector
	}

	resourceInterface := c.client.GetDynamicClient().Resource(gvr)
	var list *unstructured.UnstructuredList

	if namespace != "" {
		list, err = resourceInterface.Namespace(namespace).List(ctx, listOpts)
	} else {
		list, err = resourceInterface.List(ctx, listOpts)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list pipeline runs: %w", err)
	}

	pipelineRuns := make([]*api.PipelineRun, 0, len(list.Items))
	for _, item := range list.Items {
		pipelineRun, err := c.unstructuredToPipelineRun(&item)
		if err != nil {
			return nil, fmt.Errorf("failed to convert pipeline run: %w", err)
		}
		pipelineRuns = append(pipelineRuns, pipelineRun)
	}

	return pipelineRuns, nil
}

// GetPipelineRun retrieves a specific pipeline run by name
func (c *PipelineClient) GetRun(ctx context.Context, namespace, name string) (*api.PipelineRun, error) {
	gvr, err := c.client.GetGVR("pipelineruns")
	if err != nil {
		return nil, err
	}

	resourceInterface := c.client.GetDynamicClient().Resource(gvr)
	var item *unstructured.Unstructured

	if namespace != "" {
		item, err = resourceInterface.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	} else {
		item, err = resourceInterface.Get(ctx, name, metav1.GetOptions{})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline run %s: %w", name, err)
	}

	return c.unstructuredToPipelineRun(item)
}

// unstructuredToPipeline converts Unstructured object to Pipeline
func (c *PipelineClient) unstructuredToPipeline(obj *unstructured.Unstructured) (*api.Pipeline, error) {
	pipeline := &api.Pipeline{
		Name:        obj.GetName(),
		Namespace:   obj.GetNamespace(),
		Labels:      obj.GetLabels(),
		Annotations: obj.GetAnnotations(),
	}

	// Extract display name from annotations
	if annotations := obj.GetAnnotations(); annotations != nil {
		if displayName, ok := annotations["openshift.io/display-name"]; ok {
			pipeline.DisplayName = &displayName
		}
		if description, ok := annotations["openshift.io/description"]; ok {
			pipeline.Description = &description
		}
	}

	// Extract status
	if status, ok, _ := unstructured.NestedMap(obj.Object, "status"); ok {
		pipeline.Status = c.convertPipelineStatus(status)
	}

	return pipeline, nil
}

// unstructuredToPipelineRun converts Unstructured object to PipelineRun
func (c *PipelineClient) unstructuredToPipelineRun(obj *unstructured.Unstructured) (*api.PipelineRun, error) {
	pipelineRun := &api.PipelineRun{
		Name:        obj.GetName(),
		Namespace:   obj.GetNamespace(),
		Labels:      obj.GetLabels(),
		Annotations: obj.GetAnnotations(),
	}

	// Extract pipeline name from labels or annotations
	if labels := obj.GetLabels(); labels != nil {
		if pipelineName, ok := labels["app.kubernetes.io/part-of"]; ok {
			pipelineRun.PipelineName = pipelineName
		}
	}

	// Extract display name from annotations
	if annotations := obj.GetAnnotations(); annotations != nil {
		if displayName, ok := annotations["openshift.io/display-name"]; ok {
			pipelineRun.DisplayName = &displayName
		}
		if description, ok := annotations["openshift.io/description"]; ok {
			pipelineRun.Description = &description
		}
	}

	// Extract status
	if status, ok, _ := unstructured.NestedMap(obj.Object, "status"); ok {
		pipelineRun.Status = c.convertPipelineRunStatus(status)
	}

	return pipelineRun, nil
}

// pipelineToUnstructured converts a Pipeline to Unstructured object
func (c *PipelineClient) pipelineToUnstructured(pipeline *api.Pipeline) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "datasciencepipelines.opendatahub.io/v1alpha1",
			"kind":       "Pipeline",
			"metadata": map[string]interface{}{
				"name":        pipeline.Name,
				"namespace":   pipeline.Namespace,
				"labels":      pipeline.Labels,
				"annotations": pipeline.Annotations,
			},
		},
	}

	// Store display name and description in annotations
	annotations := make(map[string]string)
	if pipeline.Annotations != nil {
		for k, v := range pipeline.Annotations {
			annotations[k] = v
		}
	}
	if pipeline.DisplayName != nil {
		annotations["openshift.io/display-name"] = *pipeline.DisplayName
	}
	if pipeline.Description != nil {
		annotations["openshift.io/description"] = *pipeline.Description
	}
	obj.SetAnnotations(annotations)

	return obj, nil
}

// convertPipelineStatus converts map to PipelineStatus
func (c *PipelineClient) convertPipelineStatus(statusMap map[string]interface{}) api.PipelineStatus {
	status := api.PipelineStatus{
		Phase: "Unknown",
		Ready: false,
	}

	if phase, ok, _ := unstructured.NestedString(statusMap, "phase"); ok {
		status.Phase = phase
	}
	if message, ok, _ := unstructured.NestedString(statusMap, "message"); ok {
		status.Message = &message
	}
	if conditions, ok := statusMap["conditions"].([]interface{}); ok {
		for _, condition := range conditions {
			if conditionMap, ok := condition.(map[string]interface{}); ok {
				if conditionType, ok := conditionMap["type"].(string); ok && conditionType == "Ready" {
					if conditionStatus, ok := conditionMap["status"].(string); ok && conditionStatus == "True" {
						status.Ready = true
					}
				}
			}
		}
	}
	if runCount, ok, _ := unstructured.NestedInt64(statusMap, "runCount"); ok {
		status.RunCount = int(runCount)
	}
	if lastUpdated, ok, _ := unstructured.NestedString(statusMap, "lastUpdated"); ok {
		status.LastUpdated = &lastUpdated
	}

	return status
}

// convertPipelineRunStatus converts map to PipelineRunStatus
func (c *PipelineClient) convertPipelineRunStatus(statusMap map[string]interface{}) api.PipelineRunStatus {
	status := api.PipelineRunStatus{
		Phase: "Unknown",
		Ready: false,
	}

	if phase, ok, _ := unstructured.NestedString(statusMap, "phase"); ok {
		status.Phase = phase
	}
	if message, ok, _ := unstructured.NestedString(statusMap, "message"); ok {
		status.Message = &message
	}
	if conditions, ok := statusMap["conditions"].([]interface{}); ok {
		for _, condition := range conditions {
			if conditionMap, ok := condition.(map[string]interface{}); ok {
				if conditionType, ok := conditionMap["type"].(string); ok && conditionType == "Succeeded" {
					if conditionStatus, ok := conditionMap["status"].(string); ok && conditionStatus == "True" {
						status.Ready = true
					}
				}
			}
		}
	}
	if startedAt, ok, _ := unstructured.NestedString(statusMap, "startTime"); ok {
		status.StartedAt = &startedAt
	}
	if finishedAt, ok, _ := unstructured.NestedString(statusMap, "completionTime"); ok {
		status.FinishedAt = &finishedAt
	}
	if lastUpdated, ok, _ := unstructured.NestedString(statusMap, "lastUpdated"); ok {
		status.LastUpdated = &lastUpdated
	}

	return status
}
