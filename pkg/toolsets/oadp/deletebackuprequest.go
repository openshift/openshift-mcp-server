package oadp

import (
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/oadp"
	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func initDeleteBackupRequestTools() []api.ServerTool {
	return []api.ServerTool{
		initDeleteBackupRequestList(),
		initDeleteBackupRequestGet(),
	}
}

func initDeleteBackupRequestList() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_delete_backup_request_list",
			Description: "List all DeleteBackupRequests to monitor backup deletion progress",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace containing DeleteBackupRequests (default: openshift-adp)",
					},
				},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: List Delete Backup Requests",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: deleteBackupRequestListHandler,
	}
}

func deleteBackupRequestListHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	dbrs, err := oadp.ListDeleteBackupRequests(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list delete backup requests: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(dbrs)), nil
}

func initDeleteBackupRequestGet() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_delete_backup_request_get",
			Description: "Get detailed information about a DeleteBackupRequest including deletion status",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the DeleteBackupRequest (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the DeleteBackupRequest",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Get Delete Backup Request",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: deleteBackupRequestGetHandler,
	}
}

func deleteBackupRequestGetHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	dbr, err := oadp.GetDeleteBackupRequest(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get delete backup request: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(dbr)), nil
}
