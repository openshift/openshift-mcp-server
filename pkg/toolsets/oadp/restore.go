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

// RestoreAction represents the action to perform on restores
type RestoreAction string

const (
	RestoreActionList   RestoreAction = "list"
	RestoreActionGet    RestoreAction = "get"
	RestoreActionCreate RestoreAction = "create"
	RestoreActionDelete RestoreAction = "delete"
	RestoreActionLogs   RestoreAction = "logs"
)

func initRestoreTools() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "oadp_restore",
				Description: "Manage Velero/OADP restore operations: list, get, create, delete, or retrieve logs",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"action": {
							Type:        "string",
							Enum:        []any{string(RestoreActionList), string(RestoreActionGet), string(RestoreActionCreate), string(RestoreActionDelete), string(RestoreActionLogs)},
							Description: "Action to perform: 'list' (list all restores), 'get' (get restore details), 'create' (create new restore), 'delete' (delete restore), 'logs' (get restore logs)",
						},
						"namespace": {
							Type:        "string",
							Description: "Namespace containing restores (default: openshift-adp)",
						},
						"name": {
							Type:        "string",
							Description: "Name of the restore (required for get, create, delete, logs)",
						},
						"labelSelector": {
							Type:        "string",
							Description: "Label selector to filter restores (for list action)",
						},
						"backupName": {
							Type:        "string",
							Description: "Name of the backup to restore from (required for create action)",
						},
						"includedNamespaces": {
							Type:        "array",
							Description: "Namespaces to restore (for create action)",
							Items:       &jsonschema.Schema{Type: "string"},
						},
						"excludedNamespaces": {
							Type:        "array",
							Description: "Namespaces to exclude from restore (for create action)",
							Items:       &jsonschema.Schema{Type: "string"},
						},
						"includedResources": {
							Type:        "array",
							Description: "Resource types to restore (for create action)",
							Items:       &jsonschema.Schema{Type: "string"},
						},
						"excludedResources": {
							Type:        "array",
							Description: "Resource types to exclude (for create action)",
							Items:       &jsonschema.Schema{Type: "string"},
						},
						"namespaceMapping": {
							Type:        "object",
							Description: "Map source namespaces to target namespaces (for create action)",
						},
						"restorePVs": {
							Type:        "boolean",
							Description: "Whether to restore persistent volumes (for create action)",
						},
						"preserveNodePorts": {
							Type:        "boolean",
							Description: "Preserve service node ports during restore (for create action)",
						},
					},
					Required: []string{"action"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "OADP: Restore",
					ReadOnlyHint:    ptr.To(false),
					DestructiveHint: ptr.To(false),
				},
			},
			Handler: restoreHandler,
		},
	}
}

func restoreHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	action, err := api.RequiredString(params, "action")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	switch RestoreAction(action) {
	case RestoreActionList:
		return handleRestoreList(params, namespace)
	case RestoreActionGet:
		return handleRestoreGet(params, namespace)
	case RestoreActionCreate:
		return handleRestoreCreate(params, namespace)
	case RestoreActionDelete:
		return handleRestoreDelete(params, namespace)
	case RestoreActionLogs:
		return handleRestoreLogs(params, namespace)
	default:
		return api.NewToolCallResult("", fmt.Errorf("invalid action '%s': must be one of 'list', 'get', 'create', 'delete', 'logs'", action)), nil
	}
}

func handleRestoreList(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	labelSelector := ""
	if v, ok := params.GetArguments()["labelSelector"].(string); ok {
		labelSelector = v
	}

	restores, err := oadp.ListRestores(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list restores: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(restores)), nil
}

func handleRestoreGet(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required for get action")), nil
	}

	restore, err := oadp.GetRestore(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get restore: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(restore)), nil
}

func handleRestoreCreate(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required for create action")), nil
	}

	backupName, ok := params.GetArguments()["backupName"].(string)
	if !ok || backupName == "" {
		return api.NewToolCallResult("", fmt.Errorf("backupName is required for create action")), nil
	}

	spec := map[string]any{
		"backupName": backupName,
	}

	if v, ok := params.GetArguments()["includedNamespaces"].([]any); ok {
		spec["includedNamespaces"] = v
	}
	if v, ok := params.GetArguments()["excludedNamespaces"].([]any); ok {
		spec["excludedNamespaces"] = v
	}
	if v, ok := params.GetArguments()["includedResources"].([]any); ok {
		spec["includedResources"] = v
	}
	if v, ok := params.GetArguments()["excludedResources"].([]any); ok {
		spec["excludedResources"] = v
	}
	if v, ok := params.GetArguments()["namespaceMapping"].(map[string]any); ok {
		spec["namespaceMapping"] = v
	}
	if v, ok := params.GetArguments()["restorePVs"].(bool); ok {
		spec["restorePVs"] = v
	}
	if v, ok := params.GetArguments()["preserveNodePorts"].(bool); ok {
		spec["preserveNodePorts"] = v
	}

	restore := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": oadp.VeleroGroup + "/" + oadp.VeleroVersion,
			"kind":       "Restore",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": spec,
		},
	}

	created, err := oadp.CreateRestore(params.Context, params.DynamicClient(), restore)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create restore: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(created)), nil
}

func handleRestoreDelete(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required for delete action")), nil
	}

	err := oadp.DeleteRestore(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete restore: %w", err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("Restore %s/%s deleted", namespace, name), nil), nil
}

func handleRestoreLogs(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required for logs action")), nil
	}

	logs, err := oadp.GetRestoreLogs(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get restore logs: %w", err)), nil
	}

	return api.NewToolCallResult(logs, nil), nil
}
