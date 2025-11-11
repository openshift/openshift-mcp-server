package openshiftai

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
)

// ModelClient handles Model operations
type ModelClient struct {
	client *Client
}

// NewModelClient creates a new Model client
func NewModelClient(client *Client) *ModelClient {
	return &ModelClient{
		client: client,
	}
}

// List lists all Models in the cluster or in a specific namespace
func (c *ModelClient) List(ctx context.Context, namespace, modelType, status string) ([]*api.Model, error) {
	gvr, err := c.client.GetGVR("models")
	if err != nil {
		return nil, err
	}

	resourceInterface := c.client.GetDynamicClient().Resource(gvr)
	var list *unstructured.UnstructuredList

	if namespace != "" {
		list, err = resourceInterface.Namespace(namespace).List(ctx, metav1.ListOptions{})
	} else {
		list, err = resourceInterface.List(ctx, metav1.ListOptions{})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list Models: %w", err)
	}

	models := make([]*api.Model, 0, len(list.Items))
	for _, item := range list.Items {
		model, err := c.unstructuredToModel(&item)
		if err != nil {
			return nil, fmt.Errorf("failed to convert Model: %w", err)
		}
		models = append(models, model)
	}

	return models, nil
}

// Get gets a specific Model
func (c *ModelClient) Get(ctx context.Context, name, namespace string) (*api.Model, error) {
	gvr, err := c.client.GetGVR("models")
	if err != nil {
		return nil, err
	}

	resourceInterface := c.client.GetDynamicClient().Resource(gvr)
	obj, err := resourceInterface.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Model %s/%s: %w", namespace, name, err)
	}

	return c.unstructuredToModel(obj)
}

// Create creates a new Model
func (c *ModelClient) Create(ctx context.Context, model *api.Model) (*api.Model, error) {
	gvr, err := c.client.GetGVR("models")
	if err != nil {
		return nil, err
	}

	obj := c.modelToUnstructured(model)

	resourceInterface := c.client.GetDynamicClient().Resource(gvr)
	createdObj, err := resourceInterface.Namespace(model.Namespace).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create Model %s/%s: %w", model.Namespace, model.Name, err)
	}

	return c.unstructuredToModel(createdObj)
}

// Update updates an existing Model
func (c *ModelClient) Update(ctx context.Context, model *api.Model) (*api.Model, error) {
	gvr, err := c.client.GetGVR("models")
	if err != nil {
		return nil, err
	}

	obj := c.modelToUnstructured(model)

	resourceInterface := c.client.GetDynamicClient().Resource(gvr)
	updatedObj, err := resourceInterface.Namespace(model.Namespace).Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update Model %s/%s: %w", model.Namespace, model.Name, err)
	}

	return c.unstructuredToModel(updatedObj)
}

// Delete deletes a Model
func (c *ModelClient) Delete(ctx context.Context, name, namespace string) error {
	gvr, err := c.client.GetGVR("models")
	if err != nil {
		return err
	}

	resourceInterface := c.client.GetDynamicClient().Resource(gvr)
	err = resourceInterface.Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete Model %s/%s: %w", namespace, name, err)
	}

	return nil
}

// unstructuredToModel converts an Unstructured object to Model
func (c *ModelClient) unstructuredToModel(obj *unstructured.Unstructured) (*api.Model, error) {
	model := &api.Model{
		Name:        obj.GetName(),
		Namespace:   obj.GetNamespace(),
		Labels:      obj.GetLabels(),
		Annotations: obj.GetAnnotations(),
	}

	// Get display name from annotations
	if displayName, ok := obj.GetAnnotations()["openshift.io/display-name"]; ok {
		model.DisplayName = &displayName
	}

	// Get description from annotations
	if description, ok := obj.GetAnnotations()["openshift.io/description"]; ok {
		model.Description = &description
	}

	// Get model type from labels
	if modelType, ok := obj.GetLabels()["model.opendatahub.io/type"]; ok {
		model.ModelType = &modelType
	}

	// Get framework version from labels
	if frameworkVersion, ok := obj.GetLabels()["model.opendatahub.io/framework-version"]; ok {
		model.FrameworkVersion = &frameworkVersion
	}

	// Get format from labels
	if format, ok := obj.GetLabels()["model.opendatahub.io/format"]; ok {
		model.Format = &format
	}

	// Get version from labels
	if version, ok := obj.GetLabels()["model.opendatahub.io/version"]; ok {
		model.Version = &version
	}

	// Get size from annotations
	if size, ok := obj.GetAnnotations()["model.opendatahub.io/size"]; ok {
		if parsedSize, err := parseSize(size); err == nil {
			model.Size = &parsedSize
		}
	}

	// Get status
	if phase, ok, _ := unstructured.NestedString(obj.Object, "status", "phase"); ok {
		model.Status.Phase = phase
	}
	if message, ok, _ := unstructured.NestedString(obj.Object, "status", "message"); ok {
		model.Status.Message = &message
	}
	if ready, ok, _ := unstructured.NestedBool(obj.Object, "status", "ready"); ok {
		model.Status.Ready = ready
	}
	if deploymentStatus, ok, _ := unstructured.NestedString(obj.Object, "status", "deploymentStatus"); ok {
		model.Status.DeploymentStatus = &deploymentStatus
	}

	return model, nil
}

// modelToUnstructured converts a Model to Unstructured object
func (c *ModelClient) modelToUnstructured(model *api.Model) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "model.opendatahub.io/v1",
			"kind":       "Model",
			"metadata": map[string]interface{}{
				"name":      model.Name,
				"namespace": model.Namespace,
			},
			"spec": map[string]interface{}{
				"modelType":        model.ModelType,
				"frameworkVersion": model.FrameworkVersion,
				"format":           model.Format,
				"version":          model.Version,
			},
		},
	}

	// Store display name and description in annotations
	if model.Annotations == nil {
		model.Annotations = make(map[string]string)
	}
	if model.DisplayName != nil {
		model.Annotations["openshift.io/display-name"] = *model.DisplayName
	}
	if model.Description != nil {
		model.Annotations["openshift.io/description"] = *model.Description
	}

	// Store model metadata in labels
	if model.Labels == nil {
		model.Labels = make(map[string]string)
	}
	if model.ModelType != nil {
		model.Labels["model.opendatahub.io/type"] = *model.ModelType
	}
	if model.FrameworkVersion != nil {
		model.Labels["model.opendatahub.io/framework-version"] = *model.FrameworkVersion
	}
	if model.Format != nil {
		model.Labels["model.opendatahub.io/format"] = *model.Format
	}
	if model.Version != nil {
		model.Labels["model.opendatahub.io/version"] = *model.Version
	}

	if len(model.Labels) > 0 {
		obj.SetLabels(model.Labels)
	}

	if len(model.Annotations) > 0 {
		obj.SetAnnotations(model.Annotations)
	}

	return obj
}

// parseSize parses size string to int64
func parseSize(sizeStr string) (int64, error) {
	// This is a simplified implementation - in a real scenario,
	// you might want to handle different size formats (KB, MB, GB)
	if sizeStr == "" {
		return 0, nil
	}
	// For now, assume it's already in bytes
	return 0, fmt.Errorf("size parsing not implemented")
}
