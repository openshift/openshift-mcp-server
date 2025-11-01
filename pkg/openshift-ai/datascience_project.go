package openshiftai

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// DataScienceProject represents an OpenShift AI Data Science Project
type DataScienceProject struct {
	Name        string                   `json:"name"`
	Namespace   string                   `json:"namespace"`
	DisplayName *string                  `json:"displayName,omitempty"`
	Description *string                  `json:"description,omitempty"`
	Labels      map[string]string        `json:"labels,omitempty"`
	Annotations map[string]string        `json:"annotations,omitempty"`
	Status      DataScienceProjectStatus `json:"status"`
}

// DataScienceProjectStatus represents the status of a DataScienceProject
type DataScienceProjectStatus struct {
	Phase   string `json:"phase"`
	Message string `json:"message,omitempty"`
}

// DataScienceProjectClient handles DataScienceProject operations
type DataScienceProjectClient struct {
	client *Client
}

// NewDataScienceProjectClient creates a new DataScienceProject client
func NewDataScienceProjectClient(client *Client) *DataScienceProjectClient {
	return &DataScienceProjectClient{
		client: client,
	}
}

// List lists all DataScienceProjects in the cluster or in a specific namespace
func (c *DataScienceProjectClient) List(ctx context.Context, namespace string) ([]*DataScienceProject, error) {
	gvr, err := c.client.GetGVR("datascienceprojects")
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
		return nil, fmt.Errorf("failed to list DataScienceProjects: %w", err)
	}

	projects := make([]*DataScienceProject, 0, len(list.Items))
	for _, item := range list.Items {
		project, err := c.unstructuredToDataScienceProject(&item)
		if err != nil {
			return nil, fmt.Errorf("failed to convert DataScienceProject: %w", err)
		}
		projects = append(projects, project)
	}

	return projects, nil
}

// Get gets a specific DataScienceProject
func (c *DataScienceProjectClient) Get(ctx context.Context, name, namespace string) (*DataScienceProject, error) {
	gvr, err := c.client.GetGVR("datascienceprojects")
	if err != nil {
		return nil, err
	}

	resourceInterface := c.client.GetDynamicClient().Resource(gvr)
	obj, err := resourceInterface.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get DataScienceProject %s/%s: %w", namespace, name, err)
	}

	return c.unstructuredToDataScienceProject(obj)
}

// Create creates a new DataScienceProject
func (c *DataScienceProjectClient) Create(ctx context.Context, project *DataScienceProject) (*DataScienceProject, error) {
	gvr, err := c.client.GetGVR("datascienceprojects")
	if err != nil {
		return nil, err
	}

	obj := c.dataScienceProjectToUnstructured(project)

	resourceInterface := c.client.GetDynamicClient().Resource(gvr)
	createdObj, err := resourceInterface.Namespace(project.Namespace).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create DataScienceProject %s/%s: %w", project.Namespace, project.Name, err)
	}

	return c.unstructuredToDataScienceProject(createdObj)
}

// Delete deletes a DataScienceProject
func (c *DataScienceProjectClient) Delete(ctx context.Context, name, namespace string) error {
	gvr, err := c.client.GetGVR("datascienceprojects")
	if err != nil {
		return err
	}

	resourceInterface := c.client.GetDynamicClient().Resource(gvr)
	err = resourceInterface.Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete DataScienceProject %s/%s: %w", namespace, name, err)
	}

	return nil
}

// unstructuredToDataScienceProject converts an Unstructured object to DataScienceProject
func (c *DataScienceProjectClient) unstructuredToDataScienceProject(obj *unstructured.Unstructured) (*DataScienceProject, error) {
	project := &DataScienceProject{
		Name:        obj.GetName(),
		Namespace:   obj.GetNamespace(),
		Labels:      obj.GetLabels(),
		Annotations: obj.GetAnnotations(),
	}

	// Get display name from annotations (DataSciencePipelinesApplication doesn't have displayName in spec)
	if displayName, ok := obj.GetAnnotations()["openshift.io/display-name"]; ok {
		project.DisplayName = &displayName
	}

	// Get description from annotations (DataSciencePipelinesApplication doesn't have description in spec)
	if description, ok := obj.GetAnnotations()["openshift.io/description"]; ok {
		project.Description = &description
	}

	// Get status
	if phase, ok, _ := unstructured.NestedString(obj.Object, "status", "phase"); ok {
		project.Status.Phase = phase
	}
	if message, ok, _ := unstructured.NestedString(obj.Object, "status", "message"); ok {
		project.Status.Message = message
	}

	return project, nil
}

// dataScienceProjectToUnstructured converts a DataScienceProject to Unstructured object
func (c *DataScienceProjectClient) dataScienceProjectToUnstructured(project *DataScienceProject) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "datasciencepipelinesapplications.opendatahub.io/v1",
			"kind":       "DataSciencePipelinesApplication",
			"metadata": map[string]interface{}{
				"name":      project.Name,
				"namespace": project.Namespace,
			},
			"spec": map[string]interface{}{
				"dspVersion": "v2",
				"objectStorage": map[string]interface{}{
					"disableHealthCheck":  false,
					"enableExternalRoute": false,
				},
				"apiServer": map[string]interface{}{
					"deploy":      true,
					"enableOauth": true,
				},
				"database": map[string]interface{}{
					"disableHealthCheck": false,
					"mariaDB": map[string]interface{}{
						"deploy":         true,
						"pipelineDBName": "mlpipeline",
						"pvcSize":        "10Gi",
						"username":       "mlpipeline",
					},
				},
			},
		},
	}

	// Store display name and description in annotations since DataSciencePipelinesApplication doesn't have these fields in spec
	if project.Annotations == nil {
		project.Annotations = make(map[string]string)
	}
	if project.DisplayName != nil {
		project.Annotations["openshift.io/display-name"] = *project.DisplayName
	}
	if project.Description != nil {
		project.Annotations["openshift.io/description"] = *project.Description
	}

	if len(project.Labels) > 0 {
		obj.SetLabels(project.Labels)
	}

	if len(project.Annotations) > 0 {
		obj.SetAnnotations(project.Annotations)
	}

	return obj
}
