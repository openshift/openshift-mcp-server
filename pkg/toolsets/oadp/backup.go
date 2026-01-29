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

func initBackupTools() []api.ServerTool {
	return []api.ServerTool{
		initBackupList(),
		initBackupGet(),
		initBackupCreate(),
		initBackupDelete(),
		initBackupLogs(),
	}
}

func initBackupList() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_backup_list",
			Description: "List all Velero/OADP backups in the specified namespace or across all namespaces",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace containing backups (default: openshift-adp)",
					},
					"labelSelector": {
						Type:        "string",
						Description: "Label selector to filter backups (e.g., 'app=myapp')",
					},
				},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: List Backups",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: backupListHandler,
	}
}

func backupListHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	labelSelector := ""
	if v, ok := params.GetArguments()["labelSelector"].(string); ok {
		labelSelector = v
	}

	backups, err := oadp.ListBackups(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list backups: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(backups)), nil
}

func initBackupGet() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_backup_get",
			Description: "Get detailed information about a specific backup including status, included/excluded resources, and storage location",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the backup (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the backup",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Get Backup",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: backupGetHandler,
	}
}

func backupGetHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	backup, err := oadp.GetBackup(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get backup: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(backup)), nil
}

func initBackupCreate() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_backup_create",
			Description: "Create a new Velero/OADP backup with specified configuration. Supports namespace selection, label selectors, and volume snapshot options.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "OADP namespace where the backup CR will be created (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name for the backup",
					},
					"includedNamespaces": {
						Type:        "array",
						Description: "Namespaces to include in the backup",
						Items:       &jsonschema.Schema{Type: "string"},
					},
					"excludedNamespaces": {
						Type:        "array",
						Description: "Namespaces to exclude from the backup",
						Items:       &jsonschema.Schema{Type: "string"},
					},
					"includedResources": {
						Type:        "array",
						Description: "Resource types to include (e.g., ['pods', 'deployments'])",
						Items:       &jsonschema.Schema{Type: "string"},
					},
					"excludedResources": {
						Type:        "array",
						Description: "Resource types to exclude",
						Items:       &jsonschema.Schema{Type: "string"},
					},
					"labelSelector": {
						Type:        "string",
						Description: "Label selector to filter resources (e.g., 'app=myapp')",
					},
					"storageLocation": {
						Type:        "string",
						Description: "BackupStorageLocation name to use",
					},
					"volumeSnapshotLocations": {
						Type:        "array",
						Description: "VolumeSnapshotLocation names to use",
						Items:       &jsonschema.Schema{Type: "string"},
					},
					"snapshotVolumes": {
						Type:        "boolean",
						Description: "Whether to snapshot persistent volumes (default: true)",
					},
					"defaultVolumesToFsBackup": {
						Type:        "boolean",
						Description: "Use file system backup for volumes instead of snapshots",
					},
					"ttl": {
						Type:        "string",
						Description: "Backup TTL duration (e.g., '720h' for 30 days)",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Create Backup",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(false),
			},
		},
		Handler: backupCreateHandler,
	}
}

func backupCreateHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	// Build the backup spec
	spec := map[string]interface{}{}

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
	if v, ok := params.GetArguments()["labelSelector"].(string); ok && v != "" {
		spec["labelSelector"] = map[string]interface{}{
			"matchLabels": parseLabelSelector(v),
		}
	}
	if v, ok := params.GetArguments()["storageLocation"].(string); ok && v != "" {
		spec["storageLocation"] = v
	}
	if v, ok := params.GetArguments()["volumeSnapshotLocations"].([]interface{}); ok {
		spec["volumeSnapshotLocations"] = v
	}
	if v, ok := params.GetArguments()["snapshotVolumes"].(bool); ok {
		snapshotVolumes := v
		spec["snapshotVolumes"] = &snapshotVolumes
	}
	if v, ok := params.GetArguments()["defaultVolumesToFsBackup"].(bool); ok {
		spec["defaultVolumesToFsBackup"] = v
	}
	if v, ok := params.GetArguments()["ttl"].(string); ok && v != "" {
		spec["ttl"] = v
	}

	backup := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": oadp.VeleroGroup + "/" + oadp.VeleroVersion,
			"kind":       "Backup",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": spec,
		},
	}

	created, err := oadp.CreateBackup(params.Context, params.DynamicClient(), backup)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create backup: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(created)), nil
}

func initBackupDelete() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_backup_delete",
			Description: "Delete a Velero/OADP backup. This creates a DeleteBackupRequest which will also delete backup data from object storage.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the backup (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the backup to delete",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Delete Backup",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(true),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: backupDeleteHandler,
	}
}

func backupDeleteHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	err := oadp.DeleteBackup(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete backup: %w", err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("DeleteBackupRequest created for backup %s/%s", namespace, name), nil), nil
}

func initBackupLogs() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_backup_logs",
			Description: "Retrieve logs for a specific backup operation to troubleshoot issues",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the backup (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the backup",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Backup Logs",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: backupLogsHandler,
	}
}

func backupLogsHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	logs, err := oadp.GetBackupLogs(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get backup logs: %w", err)), nil
	}

	return api.NewToolCallResult(logs, nil), nil
}

// parseLabelSelector parses a label selector string like "app=myapp,env=prod" into a map
func parseLabelSelector(selector string) map[string]string {
	result := make(map[string]string)
	if selector == "" {
		return result
	}

	pairs := splitIgnoreEmpty(selector, ',')
	for _, pair := range pairs {
		kv := splitIgnoreEmpty(pair, '=')
		if len(kv) == 2 {
			result[kv[0]] = kv[1]
		}
	}
	return result
}

// splitIgnoreEmpty splits a string by separator and ignores empty parts
func splitIgnoreEmpty(s string, sep rune) []string {
	var result []string
	current := ""
	for _, r := range s {
		if r == sep {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
