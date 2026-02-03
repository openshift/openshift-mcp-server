package oadp

import (
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/oadp"
	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// DataMoverAction represents the action to perform on data mover resources
type DataMoverAction string

const (
	DataMoverActionList   DataMoverAction = "list"
	DataMoverActionGet    DataMoverAction = "get"
	DataMoverActionCancel DataMoverAction = "cancel"
)

// DataMoverType represents the type of data mover resource
type DataMoverType string

const (
	DataMoverTypeUpload   DataMoverType = "upload"
	DataMoverTypeDownload DataMoverType = "download"
)

func initDataMoverTools() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "oadp_data_mover",
				Description: "Manage Velero data mover resources (DataUpload and DataDownload for CSI snapshots): list, get, or cancel",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"action": {
							Type:        "string",
							Enum:        []any{string(DataMoverActionList), string(DataMoverActionGet), string(DataMoverActionCancel)},
							Description: "Action to perform: 'list', 'get', or 'cancel'",
						},
						"type": {
							Type:        "string",
							Enum:        []any{string(DataMoverTypeUpload), string(DataMoverTypeDownload)},
							Description: "Resource type: 'upload' (DataUpload) or 'download' (DataDownload)",
						},
						"namespace": {
							Type:        "string",
							Description: "Namespace containing resources (default: openshift-adp)",
						},
						"name": {
							Type:        "string",
							Description: "Name of the resource (required for get, cancel)",
						},
						"labelSelector": {
							Type:        "string",
							Description: "Label selector to filter resources (for list action)",
						},
					},
					Required: []string{"action", "type"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "OADP: Data Mover",
					ReadOnlyHint:    ptr.To(false),
					DestructiveHint: ptr.To(false),
				},
			},
			Handler: dataMoverHandler,
		},
	}
}

func dataMoverHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	action, err := api.RequiredString(params, "action")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	moverType, err := api.RequiredString(params, "type")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	switch DataMoverType(moverType) {
	case DataMoverTypeUpload:
		return handleDataUpload(params, namespace, DataMoverAction(action))
	case DataMoverTypeDownload:
		return handleDataDownload(params, namespace, DataMoverAction(action))
	default:
		return api.NewToolCallResult("", fmt.Errorf("invalid type '%s': must be 'upload' or 'download'", moverType)), nil
	}
}

func handleDataUpload(params api.ToolHandlerParams, namespace string, action DataMoverAction) (*api.ToolCallResult, error) {
	switch action {
	case DataMoverActionList:
		labelSelector := ""
		if v, ok := params.GetArguments()["labelSelector"].(string); ok {
			labelSelector = v
		}
		uploads, err := oadp.ListDataUploads(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to list data uploads: %w", err)), nil
		}
		return api.NewToolCallResult(params.ListOutput.PrintObj(uploads)), nil

	case DataMoverActionGet:
		name, ok := params.GetArguments()["name"].(string)
		if !ok || name == "" {
			return api.NewToolCallResult("", fmt.Errorf("name is required for get action")), nil
		}
		upload, err := oadp.GetDataUpload(params.Context, params.DynamicClient(), namespace, name)
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to get data upload: %w", err)), nil
		}
		return api.NewToolCallResult(params.ListOutput.PrintObj(upload)), nil

	case DataMoverActionCancel:
		name, ok := params.GetArguments()["name"].(string)
		if !ok || name == "" {
			return api.NewToolCallResult("", fmt.Errorf("name is required for cancel action")), nil
		}
		updated, err := oadp.CancelDataUpload(params.Context, params.DynamicClient(), namespace, name)
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to cancel data upload: %w", err)), nil
		}
		return api.NewToolCallResult(params.ListOutput.PrintObj(updated)), nil

	default:
		return api.NewToolCallResult("", fmt.Errorf("invalid action '%s'", action)), nil
	}
}

func handleDataDownload(params api.ToolHandlerParams, namespace string, action DataMoverAction) (*api.ToolCallResult, error) {
	switch action {
	case DataMoverActionList:
		labelSelector := ""
		if v, ok := params.GetArguments()["labelSelector"].(string); ok {
			labelSelector = v
		}
		downloads, err := oadp.ListDataDownloads(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to list data downloads: %w", err)), nil
		}
		return api.NewToolCallResult(params.ListOutput.PrintObj(downloads)), nil

	case DataMoverActionGet:
		name, ok := params.GetArguments()["name"].(string)
		if !ok || name == "" {
			return api.NewToolCallResult("", fmt.Errorf("name is required for get action")), nil
		}
		download, err := oadp.GetDataDownload(params.Context, params.DynamicClient(), namespace, name)
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to get data download: %w", err)), nil
		}
		return api.NewToolCallResult(params.ListOutput.PrintObj(download)), nil

	case DataMoverActionCancel:
		name, ok := params.GetArguments()["name"].(string)
		if !ok || name == "" {
			return api.NewToolCallResult("", fmt.Errorf("name is required for cancel action")), nil
		}
		updated, err := oadp.CancelDataDownload(params.Context, params.DynamicClient(), namespace, name)
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to cancel data download: %w", err)), nil
		}
		return api.NewToolCallResult(params.ListOutput.PrintObj(updated)), nil

	default:
		return api.NewToolCallResult("", fmt.Errorf("invalid action '%s'", action)), nil
	}
}
