package openshiftai

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

// DynamicClientManager manages dynamic client operations for OpenShift AI CRDs
type DynamicClientManager struct {
	dynamicClient dynamic.Interface
	restMapper    meta.RESTMapper
}

// NewDynamicClientManager creates a new dynamic client manager
func NewDynamicClientManager(dynamicClient dynamic.Interface, restMapper meta.RESTMapper) *DynamicClientManager {
	return &DynamicClientManager{
		dynamicClient: dynamicClient,
		restMapper:    restMapper,
	}
}

// ResourceClient provides a typed interface for working with specific resources
type ResourceClient struct {
	client    dynamic.ResourceInterface
	gvr       schema.GroupVersionResource
	namespace string
}

// GetResourceClient returns a resource client for the specified resource type
func (d *DynamicClientManager) GetResourceClient(resource, namespace string) (*ResourceClient, error) {
	gvr, err := getGVRForResource(resource)
	if err != nil {
		return nil, err
	}

	var resourceInterface dynamic.ResourceInterface
	if namespace != "" {
		resourceInterface = d.dynamicClient.Resource(gvr).Namespace(namespace)
	} else {
		resourceInterface = d.dynamicClient.Resource(gvr)
	}

	return &ResourceClient{
		client:    resourceInterface,
		gvr:       gvr,
		namespace: namespace,
	}, nil
}

// List lists resources of the specified type
func (r *ResourceClient) List(ctx context.Context, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return r.client.List(ctx, opts)
}

// Get gets a specific resource by name
func (r *ResourceClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*unstructured.Unstructured, error) {
	return r.client.Get(ctx, name, opts)
}

// Create creates a new resource
func (r *ResourceClient) Create(ctx context.Context, obj *unstructured.Unstructured, opts metav1.CreateOptions) (*unstructured.Unstructured, error) {
	return r.client.Create(ctx, obj, opts)
}

// Update updates an existing resource
func (r *ResourceClient) Update(ctx context.Context, obj *unstructured.Unstructured, opts metav1.UpdateOptions) (*unstructured.Unstructured, error) {
	return r.client.Update(ctx, obj, opts)
}

// Delete deletes a resource by name
func (r *ResourceClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return r.client.Delete(ctx, name, opts)
}

// DeleteCollection deletes a collection of resources
func (r *ResourceClient) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	return r.client.DeleteCollection(ctx, opts, listOpts)
}

// Patch patches a resource
func (r *ResourceClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions) (*unstructured.Unstructured, error) {
	return r.client.Patch(ctx, name, pt, data, opts)
}

// GetGVR returns the GroupVersionResource for this client
func (r *ResourceClient) GetGVR() schema.GroupVersionResource {
	return r.gvr
}

// GetNamespace returns the namespace for this client
func (r *ResourceClient) GetNamespace() string {
	return r.namespace
}

// getGVRForResource returns the GroupVersionResource for a given resource type
func getGVRForResource(resource string) (schema.GroupVersionResource, error) {
	switch resource {
	case "datascienceprojects":
		// Align with client.GetGVR mapping: DataSciencePipelinesApplication as the project abstraction
		return schema.GroupVersionResource{
			Group:    "datasciencepipelinesapplications.opendatahub.io",
			Version:  "v1",
			Resource: "datasciencepipelinesapplications",
		}, nil
	case "notebooks":
		return schema.GroupVersionResource{
			Group:    "kubeflow.org",
			Version:  "v1",
			Resource: "notebooks",
		}, nil
	case "inferenceservices":
		return schema.GroupVersionResource{
			Group:    "serving.kserve.io",
			Version:  "v1beta1",
			Resource: "inferenceservices",
		}, nil
	case "pipelineruns":
		return schema.GroupVersionResource{
			Group:    "tekton.dev",
			Version:  "v1beta1",
			Resource: "pipelineruns",
		}, nil
	case "pipelines":
		return schema.GroupVersionResource{
			Group:    "tekton.dev",
			Version:  "v1beta1",
			Resource: "pipelines",
		}, nil
	case "nodes":
		return schema.GroupVersionResource{
			Group:    "",
			Version:  "v1",
			Resource: "nodes",
		}, nil
	case "pods":
		return schema.GroupVersionResource{
			Group:    "",
			Version:  "v1",
			Resource: "pods",
		}, nil
	case "namespaces":
		return schema.GroupVersionResource{
			Group:    "",
			Version:  "v1",
			Resource: "namespaces",
		}, nil
	default:
		return schema.GroupVersionResource{}, fmt.Errorf("unknown resource type: %s", resource)
	}
}

// ValidateResource checks if a resource type is available in the cluster
func (d *DynamicClientManager) ValidateResource(ctx context.Context, resource string) error {
	gvr, err := getGVRForResource(resource)
	if err != nil {
		return err
	}

	// Try to list the resource to verify it exists
	_, err = d.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		return fmt.Errorf("resource %s is not available: %w", resource, err)
	}

	return nil
}

// GetAvailableResources returns a list of available OpenShift AI resource types
func (d *DynamicClientManager) GetAvailableResources(ctx context.Context) ([]string, error) {
	resources := []string{}
	resourceTypes := []string{
		"datascienceprojects",
		"notebooks",
		"inferenceservices",
		"pipelineruns",
		"pipelines",
	}

	for _, resource := range resourceTypes {
		if err := d.ValidateResource(ctx, resource); err == nil {
			resources = append(resources, resource)
		}
	}

	return resources, nil
}
