package oadp

import (
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/oadp"
	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func initDataMoverTools() []api.ServerTool {
	return []api.ServerTool{
		initDataUploadList(),
		initDataUploadGet(),
		initDataUploadCancel(),
		initDataDownloadList(),
		initDataDownloadGet(),
		initDataDownloadCancel(),
	}
}

func initDataUploadList() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_data_upload_list",
			Description: "List all DataUploads which track Kopia data movement for backups",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace containing DataUploads (default: openshift-adp)",
					},
					"labelSelector": {
						Type:        "string",
						Description: "Label selector to filter DataUploads",
					},
				},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: List Data Uploads",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: dataUploadListHandler,
	}
}

func dataUploadListHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	labelSelector := ""
	if v, ok := params.GetArguments()["labelSelector"].(string); ok {
		labelSelector = v
	}

	dus, err := oadp.ListDataUploads(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list data uploads: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(dus)), nil
}

func initDataUploadGet() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_data_upload_get",
			Description: "Get detailed information about a DataUpload including progress and status",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the DataUpload (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the DataUpload",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Get Data Upload",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: dataUploadGetHandler,
	}
}

func dataUploadGetHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	du, err := oadp.GetDataUpload(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get data upload: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(du)), nil
}

func initDataUploadCancel() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_data_upload_cancel",
			Description: "Cancel an in-progress DataUpload operation",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the DataUpload (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the DataUpload to cancel",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Cancel Data Upload",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(true),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: dataUploadCancelHandler,
	}
}

func dataUploadCancelHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	updated, err := oadp.CancelDataUpload(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to cancel data upload: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(updated)), nil
}

func initDataDownloadList() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_data_download_list",
			Description: "List all DataDownloads which track Kopia data movement for restores",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace containing DataDownloads (default: openshift-adp)",
					},
					"labelSelector": {
						Type:        "string",
						Description: "Label selector to filter DataDownloads",
					},
				},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: List Data Downloads",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: dataDownloadListHandler,
	}
}

func dataDownloadListHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	labelSelector := ""
	if v, ok := params.GetArguments()["labelSelector"].(string); ok {
		labelSelector = v
	}

	dds, err := oadp.ListDataDownloads(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list data downloads: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(dds)), nil
}

func initDataDownloadGet() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_data_download_get",
			Description: "Get detailed information about a DataDownload including progress and status",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the DataDownload (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the DataDownload",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Get Data Download",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: dataDownloadGetHandler,
	}
}

func dataDownloadGetHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	dd, err := oadp.GetDataDownload(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get data download: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(dd)), nil
}

func initDataDownloadCancel() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_data_download_cancel",
			Description: "Cancel an in-progress DataDownload operation",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the DataDownload (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the DataDownload to cancel",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Cancel Data Download",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(true),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: dataDownloadCancelHandler,
	}
}

func dataDownloadCancelHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	updated, err := oadp.CancelDataDownload(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to cancel data download: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(updated)), nil
}
