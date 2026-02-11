package oadp

import (
	"fmt"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/oadp"
	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"
)

// BackupAction represents the action to perform on backups
type BackupAction string

const (
	BackupActionList   BackupAction = "list"
	BackupActionGet    BackupAction = "get"
	BackupActionCreate BackupAction = "create"
	BackupActionDelete BackupAction = "delete"
	BackupActionStatus BackupAction = "status"
)

func initBackupTools() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "oadp_backup",
				Description: "Manage Velero/OADP backups: list, get, create, delete, or get status",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"action": {
							Type:        "string",
							Enum:        []any{string(BackupActionList), string(BackupActionGet), string(BackupActionCreate), string(BackupActionDelete), string(BackupActionStatus)},
							Description: "Action to perform: 'list' (list all backups), 'get' (get backup details), 'create' (create new backup), 'delete' (delete backup), 'status' (get detailed backup status)",
						},
						"namespace": {
							Type:        "string",
							Description: "Namespace containing backups (default: openshift-adp)",
						},
						"name": {
							Type:        "string",
							Description: "Name of the backup (required for get, create, delete, status)",
						},
						"labelSelector": {
							Type:        "string",
							Description: "Label selector to filter backups (for list action)",
						},
						"includedNamespaces": {
							Type:        "array",
							Description: "Namespaces to include in the backup (for create action)",
							Items:       &jsonschema.Schema{Type: "string"},
						},
						"excludedNamespaces": {
							Type:        "array",
							Description: "Namespaces to exclude from the backup (for create action)",
							Items:       &jsonschema.Schema{Type: "string"},
						},
						"includedResources": {
							Type:        "array",
							Description: "Resource types to include (for create action)",
							Items:       &jsonschema.Schema{Type: "string"},
						},
						"excludedResources": {
							Type:        "array",
							Description: "Resource types to exclude (for create action)",
							Items:       &jsonschema.Schema{Type: "string"},
						},
						"storageLocation": {
							Type:        "string",
							Description: "BackupStorageLocation name to use (for create action)",
						},
						"volumeSnapshotLocations": {
							Type:        "array",
							Description: "VolumeSnapshotLocation names to use (for create action)",
							Items:       &jsonschema.Schema{Type: "string"},
						},
						"snapshotVolumes": {
							Type:        "boolean",
							Description: "Whether to snapshot persistent volumes (for create action)",
						},
						"defaultVolumesToFsBackup": {
							Type:        "boolean",
							Description: "Use file system backup for volumes instead of snapshots (for create action)",
						},
						"ttl": {
							Type:        "string",
							Description: "Backup TTL duration e.g., '720h' for 30 days (for create action)",
						},
					},
					Required: []string{"action"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "OADP: Backup",
					ReadOnlyHint:    ptr.To(false),
					DestructiveHint: ptr.To(false),
				},
			},
			Handler: backupHandler,
		},
	}
}

func backupHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	action, err := api.RequiredString(params, "action")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	namespace := api.OptionalString(params, "namespace", oadp.DefaultOADPNamespace)

	switch BackupAction(action) {
	case BackupActionList:
		return handleBackupList(params, namespace)
	case BackupActionGet:
		return handleBackupGet(params, namespace)
	case BackupActionCreate:
		return handleBackupCreate(params, namespace)
	case BackupActionDelete:
		return handleBackupDelete(params, namespace)
	case BackupActionStatus:
		return handleBackupStatus(params, namespace)
	default:
		return api.NewToolCallResult("", fmt.Errorf("invalid action '%s': must be one of 'list', 'get', 'create', 'delete', 'status'", action)), nil
	}
}

func handleBackupList(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	labelSelector := api.OptionalString(params, "labelSelector", "")

	backups, err := oadp.ListBackups(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list backups: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(backups)), nil
}

func handleBackupGet(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required for get action")), nil
	}

	backup, err := oadp.GetBackup(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get backup: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(backup)), nil
}

func handleBackupCreate(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required for create action")), nil
	}

	spec := map[string]any{}

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
	if v, ok := params.GetArguments()["labelSelector"].(string); ok && v != "" {
		matchLabels, err := parseLabelSelector(v)
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("invalid labelSelector: %w", err)), nil
		}
		spec["labelSelector"] = map[string]any{
			"matchLabels": matchLabels,
		}
	}
	if v, ok := params.GetArguments()["storageLocation"].(string); ok && v != "" {
		spec["storageLocation"] = v
	}
	if v, ok := params.GetArguments()["volumeSnapshotLocations"].([]any); ok {
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
		Object: map[string]any{
			"apiVersion": oadp.VeleroGroup + "/" + oadp.VeleroVersion,
			"kind":       "Backup",
			"metadata": map[string]any{
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

func handleBackupDelete(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required for delete action")), nil
	}

	err := oadp.DeleteBackup(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete backup: %w", err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("DeleteBackupRequest created for backup %s/%s", namespace, name), nil), nil
}

func handleBackupStatus(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required for status action")), nil
	}

	status, err := oadp.GetBackupStatus(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get backup status: %w", err)), nil
	}

	return api.NewToolCallResult(status, nil), nil
}

// parseLabelSelector parses a label selector string like "app=myapp,env=prod" into a map
// Returns an error if the selector contains non-equality expressions (!=, in, notin)
func parseLabelSelector(selector string) (map[string]string, error) {
	result := make(map[string]string)
	if selector == "" {
		return result, nil
	}

	// Check for unsupported selector syntax
	if strings.Contains(selector, "!=") || strings.Contains(selector, " in ") ||
		strings.Contains(selector, " notin ") || strings.Contains(selector, "(") {
		return nil, fmt.Errorf("only equality-based selectors (key=value) are supported; got: %s", selector)
	}

	pairs := splitIgnoreEmpty(selector, ',')
	for _, pair := range pairs {
		kv := splitIgnoreEmpty(pair, '=')
		if len(kv) == 2 {
			result[kv[0]] = kv[1]
		} else {
			return nil, fmt.Errorf("invalid label selector format: %s (expected key=value)", pair)
		}
	}
	return result, nil
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
