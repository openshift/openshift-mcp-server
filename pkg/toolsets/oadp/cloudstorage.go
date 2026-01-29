package oadp

import (
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/oadp"
	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"
)

func initCloudStorageTools() []api.ServerTool {
	return []api.ServerTool{
		initCloudStorageList(),
		initCloudStorageGet(),
		initCloudStorageCreate(),
		initCloudStorageDelete(),
	}
}

func initCloudStorageList() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_cloud_storage_list",
			Description: "List all CloudStorage resources for automatic bucket management",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace containing CloudStorages (default: openshift-adp)",
					},
				},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: List Cloud Storages",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: cloudStorageListHandler,
	}
}

func cloudStorageListHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	css, err := oadp.ListCloudStorages(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list cloud storages: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(css)), nil
}

func initCloudStorageGet() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_cloud_storage_get",
			Description: "Get detailed information about a CloudStorage including bucket status",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the CloudStorage (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the CloudStorage",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Get Cloud Storage",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: cloudStorageGetHandler,
	}
}

func cloudStorageGetHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	cs, err := oadp.GetCloudStorage(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get cloud storage: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(cs)), nil
}

func initCloudStorageCreate() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_cloud_storage_create",
			Description: "Create a CloudStorage resource to automatically create and manage a cloud storage bucket",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace for the CloudStorage (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name for the CloudStorage resource",
					},
					"bucketName": {
						Type:        "string",
						Description: "Name of the bucket to create",
					},
					"provider": {
						Type:        "string",
						Description: "Cloud provider: aws, azure, or gcp",
					},
					"credentialSecretName": {
						Type:        "string",
						Description: "Name of the secret containing cloud credentials",
					},
					"credentialSecretKey": {
						Type:        "string",
						Description: "Key in the secret containing credentials (default: cloud)",
					},
					"region": {
						Type:        "string",
						Description: "Region for the bucket (e.g., us-east-1)",
					},
				},
				Required: []string{"name", "bucketName", "provider", "credentialSecretName"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Create Cloud Storage",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(false),
			},
		},
		Handler: cloudStorageCreateHandler,
	}
}

func cloudStorageCreateHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	bucketName, ok := params.GetArguments()["bucketName"].(string)
	if !ok || bucketName == "" {
		return api.NewToolCallResult("", fmt.Errorf("bucketName is required")), nil
	}

	provider, ok := params.GetArguments()["provider"].(string)
	if !ok || provider == "" {
		return api.NewToolCallResult("", fmt.Errorf("provider is required")), nil
	}

	credSecretName, ok := params.GetArguments()["credentialSecretName"].(string)
	if !ok || credSecretName == "" {
		return api.NewToolCallResult("", fmt.Errorf("credentialSecretName is required")), nil
	}

	credSecretKey := "cloud"
	if v, ok := params.GetArguments()["credentialSecretKey"].(string); ok && v != "" {
		credSecretKey = v
	}

	spec := map[string]any{
		"name":     bucketName,
		"provider": provider,
		"creationSecret": map[string]any{
			"name": credSecretName,
			"key":  credSecretKey,
		},
	}

	if v, ok := params.GetArguments()["region"].(string); ok && v != "" {
		spec["region"] = v
	}

	cs := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": oadp.OADPGroup + "/" + oadp.OADPVersion,
			"kind":       "CloudStorage",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": spec,
		},
	}

	created, err := oadp.CreateCloudStorage(params.Context, params.DynamicClient(), cs)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create cloud storage: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(created)), nil
}

func initCloudStorageDelete() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_cloud_storage_delete",
			Description: "Delete a CloudStorage resource. Note: This may also delete the associated bucket.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the CloudStorage (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the CloudStorage to delete",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Delete Cloud Storage",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(true),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: cloudStorageDeleteHandler,
	}
}

func cloudStorageDeleteHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	err := oadp.DeleteCloudStorage(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete cloud storage: %w", err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("CloudStorage %s/%s deleted", namespace, name), nil), nil
}
