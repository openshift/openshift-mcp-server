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

func initDownloadRequestTools() []api.ServerTool {
	return []api.ServerTool{
		initDownloadRequestList(),
		initDownloadRequestGet(),
		initDownloadRequestCreate(),
		initDownloadRequestDelete(),
	}
}

func initDownloadRequestList() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_download_request_list",
			Description: "List all DownloadRequests for accessing backup/restore logs and data",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace containing DownloadRequests (default: openshift-adp)",
					},
				},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: List Download Requests",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: downloadRequestListHandler,
	}
}

func downloadRequestListHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	drs, err := oadp.ListDownloadRequests(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list download requests: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(drs)), nil
}

func initDownloadRequestGet() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_download_request_get",
			Description: "Get detailed information about a DownloadRequest including the download URL",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the DownloadRequest (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the DownloadRequest",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Get Download Request",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: downloadRequestGetHandler,
	}
}

func downloadRequestGetHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	dr, err := oadp.GetDownloadRequest(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get download request: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(dr)), nil
}

func initDownloadRequestCreate() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_download_request_create",
			Description: "Create a DownloadRequest to get a signed URL for downloading backup/restore logs or data",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace for the DownloadRequest (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name for the DownloadRequest",
					},
					"targetKind": {
						Type:        "string",
						Description: "Kind of data to download: BackupLog, BackupContents, BackupVolumeSnapshots, BackupItemOperations, BackupResourceList, BackupResults, RestoreLog, RestoreResults, RestoreResourceList, RestoreItemOperations",
					},
					"targetName": {
						Type:        "string",
						Description: "Name of the backup or restore to download data from",
					},
				},
				Required: []string{"name", "targetKind", "targetName"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Create Download Request",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(false),
			},
		},
		Handler: downloadRequestCreateHandler,
	}
}

func downloadRequestCreateHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	targetKind, ok := params.GetArguments()["targetKind"].(string)
	if !ok || targetKind == "" {
		return api.NewToolCallResult("", fmt.Errorf("targetKind is required")), nil
	}

	targetName, ok := params.GetArguments()["targetName"].(string)
	if !ok || targetName == "" {
		return api.NewToolCallResult("", fmt.Errorf("targetName is required")), nil
	}

	dr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": oadp.VeleroGroup + "/" + oadp.VeleroVersion,
			"kind":       "DownloadRequest",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"target": map[string]interface{}{
					"kind": targetKind,
					"name": targetName,
				},
			},
		},
	}

	created, err := oadp.CreateDownloadRequest(params.Context, params.DynamicClient(), dr)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create download request: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(created)), nil
}

func initDownloadRequestDelete() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_download_request_delete",
			Description: "Delete a DownloadRequest",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the DownloadRequest (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the DownloadRequest to delete",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Delete Download Request",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(true),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: downloadRequestDeleteHandler,
	}
}

func downloadRequestDeleteHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	err := oadp.DeleteDownloadRequest(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete download request: %w", err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("DownloadRequest %s/%s deleted", namespace, name), nil), nil
}
