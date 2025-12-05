package openshiftai

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
)

// ApplicationClient handles Application operations
type ApplicationClient struct {
	client *Client
}

// NewApplicationClient creates a new Application client
func NewApplicationClient(client *Client) *ApplicationClient {
	return &ApplicationClient{
		client: client,
	}
}

// List lists all Applications in the cluster or in a specific namespace
func (c *ApplicationClient) List(ctx context.Context, namespace, status, appType string) ([]*api.Application, error) {
	gvr, err := c.client.GetGVR("applications")
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
		return nil, fmt.Errorf("failed to list Applications: %w", err)
	}

	applications := make([]*api.Application, 0, len(list.Items))
	for _, item := range list.Items {
		application, err := c.unstructuredToApplication(&item)
		if err != nil {
			return nil, fmt.Errorf("failed to convert Application: %w", err)
		}
		applications = append(applications, application)
	}

	return applications, nil
}

// Get gets a specific Application
func (c *ApplicationClient) Get(ctx context.Context, name, namespace string) (*api.Application, error) {
	gvr, err := c.client.GetGVR("applications")
	if err != nil {
		return nil, err
	}

	resourceInterface := c.client.GetDynamicClient().Resource(gvr)
	obj, err := resourceInterface.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Application %s/%s: %w", namespace, name, err)
	}

	return c.unstructuredToApplication(obj)
}

// Create creates a new Application
func (c *ApplicationClient) Create(ctx context.Context, application *api.Application) (*api.Application, error) {
	gvr, err := c.client.GetGVR("applications")
	if err != nil {
		return nil, err
	}

	obj := c.applicationToUnstructured(application)

	resourceInterface := c.client.GetDynamicClient().Resource(gvr)
	createdObj, err := resourceInterface.Namespace(application.Namespace).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create Application %s/%s: %w", application.Namespace, application.Name, err)
	}

	return c.unstructuredToApplication(createdObj)
}

// Delete deletes an Application
func (c *ApplicationClient) Delete(ctx context.Context, name, namespace string) error {
	gvr, err := c.client.GetGVR("applications")
	if err != nil {
		return err
	}

	resourceInterface := c.client.GetDynamicClient().Resource(gvr)
	err = resourceInterface.Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete Application %s/%s: %w", namespace, name, err)
	}

	return nil
}

// unstructuredToApplication converts an Unstructured object to Application
func (c *ApplicationClient) unstructuredToApplication(obj *unstructured.Unstructured) (*api.Application, error) {
	application := &api.Application{
		Name:        obj.GetName(),
		Namespace:   obj.GetNamespace(),
		Labels:      obj.GetLabels(),
		Annotations: obj.GetAnnotations(),
	}

	// Get display name from annotations
	if displayName, ok := obj.GetAnnotations()["openshift.io/display-name"]; ok {
		application.DisplayName = &displayName
	}

	// Get description from annotations
	if description, ok := obj.GetAnnotations()["openshift.io/description"]; ok {
		application.Description = &description
	}

	// Get status
	if phase, ok, _ := unstructured.NestedString(obj.Object, "status", "phase"); ok {
		application.Status.Phase = phase
	}
	if message, ok, _ := unstructured.NestedString(obj.Object, "status", "message"); ok {
		application.Status.Message = &message
	}
	if ready, ok, _ := unstructured.NestedBool(obj.Object, "status", "ready"); ok {
		application.Status.Ready = ready
	}
	if appType, ok, _ := unstructured.NestedString(obj.Object, "spec", "appType"); ok {
		application.AppType = appType
	}
	if url, ok, _ := unstructured.NestedString(obj.Object, "status", "url"); ok {
		application.Status.URL = &url
	}
	if lastUpdated, ok, _ := unstructured.NestedString(obj.Object, "status", "lastUpdated"); ok {
		application.Status.LastUpdated = &lastUpdated
	}

	return application, nil
}

// applicationToUnstructured converts a Application to Unstructured object
func (c *ApplicationClient) applicationToUnstructured(application *api.Application) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "app.opendatahub.io/v1",
			"kind":       "Application",
			"metadata": map[string]interface{}{
				"name":      application.Name,
				"namespace": application.Namespace,
			},
			"spec": map[string]interface{}{
				"appType": application.AppType,
			},
		},
	}

	// Store display name and description in annotations
	if application.Annotations == nil {
		application.Annotations = make(map[string]string)
	}
	if application.DisplayName != nil {
		application.Annotations["openshift.io/display-name"] = *application.DisplayName
	}
	if application.Description != nil {
		application.Annotations["openshift.io/description"] = *application.Description
	}

	if len(application.Labels) > 0 {
		obj.SetLabels(application.Labels)
	}

	if len(application.Annotations) > 0 {
		obj.SetAnnotations(application.Annotations)
	}

	return obj
}
