package openshiftai

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
)

// ExperimentClient handles Experiment operations
type ExperimentClient struct {
	client *Client
}

// NewExperimentClient creates a new Experiment client
func NewExperimentClient(client *Client) *ExperimentClient {
	return &ExperimentClient{
		client: client,
	}
}

// List lists all Experiments in the cluster or in a specific namespace
func (c *ExperimentClient) List(ctx context.Context, namespace, status string) ([]*api.Experiment, error) {
	gvr, err := c.client.GetGVR("experiments")
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
		return nil, fmt.Errorf("failed to list Experiments: %w", err)
	}

	experiments := make([]*api.Experiment, 0, len(list.Items))
	for _, item := range list.Items {
		experiment, err := c.unstructuredToExperiment(&item)
		if err != nil {
			return nil, fmt.Errorf("failed to convert Experiment: %w", err)
		}

		// Filter by status if specified
		if status != "" && experiment.Status.Phase != status {
			continue
		}

		experiments = append(experiments, experiment)
	}

	return experiments, nil
}

// Get gets a specific Experiment
func (c *ExperimentClient) Get(ctx context.Context, name, namespace string) (*api.Experiment, error) {
	gvr, err := c.client.GetGVR("experiments")
	if err != nil {
		return nil, err
	}

	resourceInterface := c.client.GetDynamicClient().Resource(gvr)
	obj, err := resourceInterface.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Experiment %s/%s: %w", namespace, name, err)
	}

	return c.unstructuredToExperiment(obj)
}

// Create creates a new Experiment
func (c *ExperimentClient) Create(ctx context.Context, experiment *api.Experiment) (*api.Experiment, error) {
	gvr, err := c.client.GetGVR("experiments")
	if err != nil {
		return nil, err
	}

	obj := c.experimentToUnstructured(experiment)

	resourceInterface := c.client.GetDynamicClient().Resource(gvr)
	createdObj, err := resourceInterface.Namespace(experiment.Namespace).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create Experiment %s/%s: %w", experiment.Namespace, experiment.Name, err)
	}

	return c.unstructuredToExperiment(createdObj)
}

// Delete deletes an Experiment
func (c *ExperimentClient) Delete(ctx context.Context, name, namespace string) error {
	gvr, err := c.client.GetGVR("experiments")
	if err != nil {
		return err
	}

	resourceInterface := c.client.GetDynamicClient().Resource(gvr)
	err = resourceInterface.Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete Experiment %s/%s: %w", namespace, name, err)
	}

	return nil
}

// unstructuredToExperiment converts an Unstructured object to Experiment
func (c *ExperimentClient) unstructuredToExperiment(obj *unstructured.Unstructured) (*api.Experiment, error) {
	experiment := &api.Experiment{
		Name:        obj.GetName(),
		Namespace:   obj.GetNamespace(),
		Labels:      obj.GetLabels(),
		Annotations: obj.GetAnnotations(),
	}

	// Get display name from annotations
	if displayName, ok := obj.GetAnnotations()["openshift.io/display-name"]; ok {
		experiment.DisplayName = &displayName
	}

	// Get description from annotations
	if description, ok := obj.GetAnnotations()["openshift.io/description"]; ok {
		experiment.Description = &description
	}

	// Get status
	if phase, ok, _ := unstructured.NestedString(obj.Object, "status", "phase"); ok {
		experiment.Status.Phase = phase
	}
	if message, ok, _ := unstructured.NestedString(obj.Object, "status", "message"); ok {
		experiment.Status.Message = &message
	}
	if ready, ok, _ := unstructured.NestedBool(obj.Object, "status", "ready"); ok {
		experiment.Status.Ready = ready
	}
	if runCount, ok, _ := unstructured.NestedInt64(obj.Object, "status", "runCount"); ok {
		experiment.Status.RunCount = int(runCount)
	}
	if lastUpdated, ok, _ := unstructured.NestedString(obj.Object, "status", "lastUpdated"); ok {
		experiment.Status.LastUpdated = &lastUpdated
	}

	return experiment, nil
}

// experimentToUnstructured converts a Experiment to Unstructured object
func (c *ExperimentClient) experimentToUnstructured(experiment *api.Experiment) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "datasciencepipelines.opendatahub.io/v1",
			"kind":       "Experiment",
			"metadata": map[string]interface{}{
				"name":      experiment.Name,
				"namespace": experiment.Namespace,
			},
		},
	}

	// Store display name and description in annotations
	if experiment.Annotations == nil {
		experiment.Annotations = make(map[string]string)
	}
	if experiment.DisplayName != nil {
		experiment.Annotations["openshift.io/display-name"] = *experiment.DisplayName
	}
	if experiment.Description != nil {
		experiment.Annotations["openshift.io/description"] = *experiment.Description
	}

	if len(experiment.Labels) > 0 {
		obj.SetLabels(experiment.Labels)
	}

	if len(experiment.Annotations) > 0 {
		obj.SetAnnotations(experiment.Annotations)
	}

	return obj
}
