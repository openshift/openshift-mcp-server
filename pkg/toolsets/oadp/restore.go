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

func initRestoreTools() []api.ServerTool {
	return []api.ServerTool{
		initRestoreList(),
		initRestoreGet(),
		initRestoreCreate(),
		initRestoreDelete(),
		initRestoreLogs(),
	}
}

func initRestoreList() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_restore_list",
			Description: "List all Velero/OADP restore operations in the specified namespace",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace containing restores (default: openshift-adp)",
					},
					"labelSelector": {
						Type:        "string",
						Description: "Label selector to filter restores",
					},
				},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: List Restores",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: restoreListHandler,
	}
}

func restoreListHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

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

func initRestoreGet() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_restore_get",
			Description: "Get detailed information about a specific restore operation including status, warnings, and errors",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the restore (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the restore",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Get Restore",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: restoreGetHandler,
	}
}

func restoreGetHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	restore, err := oadp.GetRestore(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get restore: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(restore)), nil
}

func initRestoreCreate() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_restore_create",
			Description: "Create a restore operation from an existing backup. Supports selective restoration of namespaces and resources.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "OADP namespace where the restore CR will be created (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name for the restore",
					},
					"backupName": {
						Type:        "string",
						Description: "Name of the backup to restore from",
					},
					"includedNamespaces": {
						Type:        "array",
						Description: "Namespaces to restore (default: all from backup)",
						Items:       &jsonschema.Schema{Type: "string"},
					},
					"excludedNamespaces": {
						Type:        "array",
						Description: "Namespaces to exclude from restore",
						Items:       &jsonschema.Schema{Type: "string"},
					},
					"includedResources": {
						Type:        "array",
						Description: "Resource types to restore",
						Items:       &jsonschema.Schema{Type: "string"},
					},
					"excludedResources": {
						Type:        "array",
						Description: "Resource types to exclude",
						Items:       &jsonschema.Schema{Type: "string"},
					},
					"namespaceMapping": {
						Type:        "object",
						Description: "Map source namespaces to target namespaces (e.g., {'old-ns': 'new-ns'})",
					},
					"restorePVs": {
						Type:        "boolean",
						Description: "Whether to restore persistent volumes (default: true)",
					},
					"preserveNodePorts": {
						Type:        "boolean",
						Description: "Preserve service node ports during restore",
					},
				},
				Required: []string{"name", "backupName"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Create Restore",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(true),
				IdempotentHint:  ptr.To(false),
			},
		},
		Handler: restoreCreateHandler,
	}
}

func restoreCreateHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	backupName, ok := params.GetArguments()["backupName"].(string)
	if !ok || backupName == "" {
		return api.NewToolCallResult("", fmt.Errorf("backupName is required")), nil
	}

	// Build the restore spec
	spec := map[string]interface{}{
		"backupName": backupName,
	}

	if v, ok := params.GetArguments()["includedNamespaces"].([]interface{}); ok {
		spec["includedNamespaces"] = v
	}
	if v, ok := params.GetArguments()["excludedNamespaces"].([]interface{}); ok {
		spec["excludedNamespaces"] = v
	}
	if v, ok := params.GetArguments()["includedResources"].([]interface{}); ok {
		spec["includedResources"] = v
	}
	if v, ok := params.GetArguments()["excludedResources"].([]interface{}); ok {
		spec["excludedResources"] = v
	}
	if v, ok := params.GetArguments()["namespaceMapping"].(map[string]interface{}); ok {
		spec["namespaceMapping"] = v
	}
	if v, ok := params.GetArguments()["restorePVs"].(bool); ok {
		spec["restorePVs"] = v
	}
	if v, ok := params.GetArguments()["preserveNodePorts"].(bool); ok {
		spec["preserveNodePorts"] = v
	}

	restore := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": oadp.VeleroGroup + "/" + oadp.VeleroVersion,
			"kind":       "Restore",
			"metadata": map[string]interface{}{
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

func initRestoreDelete() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_restore_delete",
			Description: "Delete a Velero/OADP restore operation record",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the restore (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the restore to delete",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Delete Restore",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(true),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: restoreDeleteHandler,
	}
}

func restoreDeleteHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	err := oadp.DeleteRestore(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete restore: %w", err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("Restore %s/%s deleted", namespace, name), nil), nil
}

func initRestoreLogs() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_restore_logs",
			Description: "Retrieve logs for a specific restore operation to troubleshoot issues",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the restore (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the restore",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Restore Logs",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: restoreLogsHandler,
	}
}

func restoreLogsHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	logs, err := oadp.GetRestoreLogs(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get restore logs: %w", err)), nil
	}

	return api.NewToolCallResult(logs, nil), nil
}
