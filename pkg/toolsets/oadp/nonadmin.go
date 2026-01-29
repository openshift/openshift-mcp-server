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

func initNonAdminTools() []api.ServerTool {
	return []api.ServerTool{
		// NonAdminBackup tools
		initNonAdminBackupList(),
		initNonAdminBackupGet(),
		initNonAdminBackupCreate(),
		initNonAdminBackupDelete(),
		// NonAdminRestore tools
		initNonAdminRestoreList(),
		initNonAdminRestoreGet(),
		initNonAdminRestoreCreate(),
		initNonAdminRestoreDelete(),
		// NonAdminBackupStorageLocation tools
		initNonAdminBSLList(),
		initNonAdminBSLGet(),
		initNonAdminBSLCreate(),
		initNonAdminBSLUpdate(),
		initNonAdminBSLDelete(),
		// NonAdminBackupStorageLocationRequest tools
		initNonAdminBSLRequestList(),
		initNonAdminBSLRequestGet(),
		initNonAdminBSLRequestApprove(),
		// NonAdminDownloadRequest tools
		initNonAdminDownloadRequestList(),
		initNonAdminDownloadRequestGet(),
		initNonAdminDownloadRequestCreate(),
		initNonAdminDownloadRequestDelete(),
	}
}

// NonAdminBackup tools

func initNonAdminBackupList() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_non_admin_backup_list",
			Description: "List all NonAdminBackups for non-admin user backup operations",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace containing NonAdminBackups",
					},
					"labelSelector": {
						Type:        "string",
						Description: "Label selector to filter backups",
					},
				},
				Required: []string{"namespace"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: List Non-Admin Backups",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: nonAdminBackupListHandler,
	}
}

func nonAdminBackupListHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace, ok := params.GetArguments()["namespace"].(string)
	if !ok || namespace == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace is required")), nil
	}

	labelSelector := ""
	if v, ok := params.GetArguments()["labelSelector"].(string); ok {
		labelSelector = v
	}

	nabs, err := oadp.ListNonAdminBackups(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list non-admin backups: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(nabs)), nil
}

func initNonAdminBackupGet() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_non_admin_backup_get",
			Description: "Get detailed information about a NonAdminBackup",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the NonAdminBackup",
					},
					"name": {
						Type:        "string",
						Description: "Name of the NonAdminBackup",
					},
				},
				Required: []string{"namespace", "name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Get Non-Admin Backup",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: nonAdminBackupGetHandler,
	}
}

func nonAdminBackupGetHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace, ok := params.GetArguments()["namespace"].(string)
	if !ok || namespace == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace is required")), nil
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	nab, err := oadp.GetNonAdminBackup(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get non-admin backup: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(nab)), nil
}

func initNonAdminBackupCreate() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_non_admin_backup_create",
			Description: "Create a NonAdminBackup for non-admin user backup operations",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace for the NonAdminBackup (user's namespace)",
					},
					"name": {
						Type:        "string",
						Description: "Name for the NonAdminBackup",
					},
					"includedNamespaces": {
						Type:        "array",
						Description: "Namespaces to include in backup",
						Items:       &jsonschema.Schema{Type: "string"},
					},
					"ttl": {
						Type:        "string",
						Description: "Backup TTL (e.g., '720h')",
					},
				},
				Required: []string{"namespace", "name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Create Non-Admin Backup",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(false),
			},
		},
		Handler: nonAdminBackupCreateHandler,
	}
}

func nonAdminBackupCreateHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace, ok := params.GetArguments()["namespace"].(string)
	if !ok || namespace == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace is required")), nil
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	backupSpec := map[string]any{}

	if v, ok := params.GetArguments()["includedNamespaces"].([]any); ok {
		backupSpec["includedNamespaces"] = v
	}
	if v, ok := params.GetArguments()["ttl"].(string); ok && v != "" {
		backupSpec["ttl"] = v
	}

	nab := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": oadp.OADPGroup + "/" + oadp.OADPVersion,
			"kind":       "NonAdminBackup",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"backupSpec": backupSpec,
			},
		},
	}

	created, err := oadp.CreateNonAdminBackup(params.Context, params.DynamicClient(), nab)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create non-admin backup: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(created)), nil
}

func initNonAdminBackupDelete() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_non_admin_backup_delete",
			Description: "Delete a NonAdminBackup",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the NonAdminBackup",
					},
					"name": {
						Type:        "string",
						Description: "Name of the NonAdminBackup to delete",
					},
				},
				Required: []string{"namespace", "name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Delete Non-Admin Backup",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(true),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: nonAdminBackupDeleteHandler,
	}
}

func nonAdminBackupDeleteHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace, ok := params.GetArguments()["namespace"].(string)
	if !ok || namespace == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace is required")), nil
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	err := oadp.DeleteNonAdminBackup(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete non-admin backup: %w", err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("NonAdminBackup %s/%s deleted", namespace, name), nil), nil
}

// NonAdminRestore tools

func initNonAdminRestoreList() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_non_admin_restore_list",
			Description: "List all NonAdminRestores for non-admin user restore operations",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace containing NonAdminRestores",
					},
				},
				Required: []string{"namespace"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: List Non-Admin Restores",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: nonAdminRestoreListHandler,
	}
}

func nonAdminRestoreListHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace, ok := params.GetArguments()["namespace"].(string)
	if !ok || namespace == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace is required")), nil
	}

	nars, err := oadp.ListNonAdminRestores(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list non-admin restores: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(nars)), nil
}

func initNonAdminRestoreGet() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_non_admin_restore_get",
			Description: "Get detailed information about a NonAdminRestore",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the NonAdminRestore",
					},
					"name": {
						Type:        "string",
						Description: "Name of the NonAdminRestore",
					},
				},
				Required: []string{"namespace", "name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Get Non-Admin Restore",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: nonAdminRestoreGetHandler,
	}
}

func nonAdminRestoreGetHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace, ok := params.GetArguments()["namespace"].(string)
	if !ok || namespace == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace is required")), nil
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	nar, err := oadp.GetNonAdminRestore(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get non-admin restore: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(nar)), nil
}

func initNonAdminRestoreCreate() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_non_admin_restore_create",
			Description: "Create a NonAdminRestore for non-admin user restore operations",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace for the NonAdminRestore",
					},
					"name": {
						Type:        "string",
						Description: "Name for the NonAdminRestore",
					},
					"backupName": {
						Type:        "string",
						Description: "Name of the backup to restore from",
					},
				},
				Required: []string{"namespace", "name", "backupName"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Create Non-Admin Restore",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(true),
				IdempotentHint:  ptr.To(false),
			},
		},
		Handler: nonAdminRestoreCreateHandler,
	}
}

func nonAdminRestoreCreateHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace, ok := params.GetArguments()["namespace"].(string)
	if !ok || namespace == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace is required")), nil
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	backupName, ok := params.GetArguments()["backupName"].(string)
	if !ok || backupName == "" {
		return api.NewToolCallResult("", fmt.Errorf("backupName is required")), nil
	}

	nar := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": oadp.OADPGroup + "/" + oadp.OADPVersion,
			"kind":       "NonAdminRestore",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"restoreSpec": map[string]any{
					"backupName": backupName,
				},
			},
		},
	}

	created, err := oadp.CreateNonAdminRestore(params.Context, params.DynamicClient(), nar)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create non-admin restore: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(created)), nil
}

func initNonAdminRestoreDelete() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_non_admin_restore_delete",
			Description: "Delete a NonAdminRestore",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the NonAdminRestore",
					},
					"name": {
						Type:        "string",
						Description: "Name of the NonAdminRestore to delete",
					},
				},
				Required: []string{"namespace", "name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Delete Non-Admin Restore",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(true),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: nonAdminRestoreDeleteHandler,
	}
}

func nonAdminRestoreDeleteHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace, ok := params.GetArguments()["namespace"].(string)
	if !ok || namespace == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace is required")), nil
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	err := oadp.DeleteNonAdminRestore(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete non-admin restore: %w", err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("NonAdminRestore %s/%s deleted", namespace, name), nil), nil
}

// NonAdminBackupStorageLocation tools

func initNonAdminBSLList() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_non_admin_bsl_list",
			Description: "List all NonAdminBackupStorageLocations",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace containing NonAdminBackupStorageLocations",
					},
				},
				Required: []string{"namespace"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: List Non-Admin BSLs",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: nonAdminBSLListHandler,
	}
}

func nonAdminBSLListHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace, ok := params.GetArguments()["namespace"].(string)
	if !ok || namespace == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace is required")), nil
	}

	nabsls, err := oadp.ListNonAdminBackupStorageLocations(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list non-admin BSLs: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(nabsls)), nil
}

func initNonAdminBSLGet() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_non_admin_bsl_get",
			Description: "Get detailed information about a NonAdminBackupStorageLocation",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the NonAdminBackupStorageLocation",
					},
					"name": {
						Type:        "string",
						Description: "Name of the NonAdminBackupStorageLocation",
					},
				},
				Required: []string{"namespace", "name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Get Non-Admin BSL",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: nonAdminBSLGetHandler,
	}
}

func nonAdminBSLGetHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace, ok := params.GetArguments()["namespace"].(string)
	if !ok || namespace == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace is required")), nil
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	nabsl, err := oadp.GetNonAdminBackupStorageLocation(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get non-admin BSL: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(nabsl)), nil
}

func initNonAdminBSLCreate() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_non_admin_bsl_create",
			Description: "Create a NonAdminBackupStorageLocation",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace for the NonAdminBackupStorageLocation",
					},
					"name": {
						Type:        "string",
						Description: "Name for the NonAdminBackupStorageLocation",
					},
					"provider": {
						Type:        "string",
						Description: "Storage provider (e.g., aws, azure, gcp)",
					},
					"bucket": {
						Type:        "string",
						Description: "Bucket name",
					},
					"credentialSecretName": {
						Type:        "string",
						Description: "Name of secret containing credentials",
					},
					"credentialSecretKey": {
						Type:        "string",
						Description: "Key in secret (default: cloud)",
					},
				},
				Required: []string{"namespace", "name", "provider", "bucket", "credentialSecretName"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Create Non-Admin BSL",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(false),
			},
		},
		Handler: nonAdminBSLCreateHandler,
	}
}

func nonAdminBSLCreateHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace, ok := params.GetArguments()["namespace"].(string)
	if !ok || namespace == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace is required")), nil
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	provider, ok := params.GetArguments()["provider"].(string)
	if !ok || provider == "" {
		return api.NewToolCallResult("", fmt.Errorf("provider is required")), nil
	}

	bucket, ok := params.GetArguments()["bucket"].(string)
	if !ok || bucket == "" {
		return api.NewToolCallResult("", fmt.Errorf("bucket is required")), nil
	}

	credSecretName, ok := params.GetArguments()["credentialSecretName"].(string)
	if !ok || credSecretName == "" {
		return api.NewToolCallResult("", fmt.Errorf("credentialSecretName is required")), nil
	}

	credSecretKey := "cloud"
	if v, ok := params.GetArguments()["credentialSecretKey"].(string); ok && v != "" {
		credSecretKey = v
	}

	nabsl := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": oadp.OADPGroup + "/" + oadp.OADPVersion,
			"kind":       "NonAdminBackupStorageLocation",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"backupStorageLocationSpec": map[string]any{
					"provider": provider,
					"objectStorage": map[string]any{
						"bucket": bucket,
					},
					"credential": map[string]any{
						"name": credSecretName,
						"key":  credSecretKey,
					},
				},
			},
		},
	}

	created, err := oadp.CreateNonAdminBackupStorageLocation(params.Context, params.DynamicClient(), nabsl)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create non-admin BSL: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(created)), nil
}

func initNonAdminBSLUpdate() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_non_admin_bsl_update",
			Description: "Update a NonAdminBackupStorageLocation",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the NonAdminBackupStorageLocation",
					},
					"name": {
						Type:        "string",
						Description: "Name of the NonAdminBackupStorageLocation",
					},
					"bucket": {
						Type:        "string",
						Description: "New bucket name",
					},
				},
				Required: []string{"namespace", "name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Update Non-Admin BSL",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: nonAdminBSLUpdateHandler,
	}
}

func nonAdminBSLUpdateHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace, ok := params.GetArguments()["namespace"].(string)
	if !ok || namespace == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace is required")), nil
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	nabsl, err := oadp.GetNonAdminBackupStorageLocation(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get non-admin BSL: %w", err)), nil
	}

	if bucket, ok := params.GetArguments()["bucket"].(string); ok && bucket != "" {
		if err := unstructured.SetNestedField(nabsl.Object, bucket, "spec", "backupStorageLocationSpec", "objectStorage", "bucket"); err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to set bucket: %w", err)), nil
		}
	}

	updated, err := oadp.UpdateNonAdminBackupStorageLocation(params.Context, params.DynamicClient(), nabsl)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to update non-admin BSL: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(updated)), nil
}

func initNonAdminBSLDelete() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_non_admin_bsl_delete",
			Description: "Delete a NonAdminBackupStorageLocation",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the NonAdminBackupStorageLocation",
					},
					"name": {
						Type:        "string",
						Description: "Name of the NonAdminBackupStorageLocation to delete",
					},
				},
				Required: []string{"namespace", "name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Delete Non-Admin BSL",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(true),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: nonAdminBSLDeleteHandler,
	}
}

func nonAdminBSLDeleteHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace, ok := params.GetArguments()["namespace"].(string)
	if !ok || namespace == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace is required")), nil
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	err := oadp.DeleteNonAdminBackupStorageLocation(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete non-admin BSL: %w", err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("NonAdminBackupStorageLocation %s/%s deleted", namespace, name), nil), nil
}

// NonAdminBackupStorageLocationRequest tools

func initNonAdminBSLRequestList() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_non_admin_bsl_request_list",
			Description: "List all NonAdminBackupStorageLocationRequests pending admin approval",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace containing requests (default: openshift-adp)",
					},
				},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: List Non-Admin BSL Requests",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: nonAdminBSLRequestListHandler,
	}
}

func nonAdminBSLRequestListHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	reqs, err := oadp.ListNonAdminBackupStorageLocationRequests(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list non-admin BSL requests: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(reqs)), nil
}

func initNonAdminBSLRequestGet() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_non_admin_bsl_request_get",
			Description: "Get detailed information about a NonAdminBackupStorageLocationRequest",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the request (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the request",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Get Non-Admin BSL Request",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: nonAdminBSLRequestGetHandler,
	}
}

func nonAdminBSLRequestGetHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	req, err := oadp.GetNonAdminBackupStorageLocationRequest(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get non-admin BSL request: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(req)), nil
}

func initNonAdminBSLRequestApprove() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_non_admin_bsl_request_approve",
			Description: "Approve or reject a NonAdminBackupStorageLocationRequest",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the request (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the request",
					},
					"decision": {
						Type:        "string",
						Description: "Approval decision: approve, reject, or pending",
					},
				},
				Required: []string{"name", "decision"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Approve/Reject Non-Admin BSL Request",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: nonAdminBSLRequestApproveHandler,
	}
}

func nonAdminBSLRequestApproveHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	decision, ok := params.GetArguments()["decision"].(string)
	if !ok || decision == "" {
		return api.NewToolCallResult("", fmt.Errorf("decision is required")), nil
	}

	updated, err := oadp.ApproveNonAdminBackupStorageLocationRequest(params.Context, params.DynamicClient(), namespace, name, decision)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to update approval decision: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(updated)), nil
}

// NonAdminDownloadRequest tools

func initNonAdminDownloadRequestList() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_non_admin_download_request_list",
			Description: "List all NonAdminDownloadRequests",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace containing requests",
					},
				},
				Required: []string{"namespace"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: List Non-Admin Download Requests",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: nonAdminDownloadRequestListHandler,
	}
}

func nonAdminDownloadRequestListHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace, ok := params.GetArguments()["namespace"].(string)
	if !ok || namespace == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace is required")), nil
	}

	reqs, err := oadp.ListNonAdminDownloadRequests(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list non-admin download requests: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(reqs)), nil
}

func initNonAdminDownloadRequestGet() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_non_admin_download_request_get",
			Description: "Get detailed information about a NonAdminDownloadRequest",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the request",
					},
					"name": {
						Type:        "string",
						Description: "Name of the request",
					},
				},
				Required: []string{"namespace", "name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Get Non-Admin Download Request",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: nonAdminDownloadRequestGetHandler,
	}
}

func nonAdminDownloadRequestGetHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace, ok := params.GetArguments()["namespace"].(string)
	if !ok || namespace == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace is required")), nil
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	req, err := oadp.GetNonAdminDownloadRequest(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get non-admin download request: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(req)), nil
}

func initNonAdminDownloadRequestCreate() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_non_admin_download_request_create",
			Description: "Create a NonAdminDownloadRequest to get logs or data",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace for the request",
					},
					"name": {
						Type:        "string",
						Description: "Name for the request",
					},
					"targetKind": {
						Type:        "string",
						Description: "Kind of data: BackupLog, RestoreLog, etc.",
					},
					"targetName": {
						Type:        "string",
						Description: "Name of backup/restore to download from",
					},
				},
				Required: []string{"namespace", "name", "targetKind", "targetName"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Create Non-Admin Download Request",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(false),
			},
		},
		Handler: nonAdminDownloadRequestCreateHandler,
	}
}

func nonAdminDownloadRequestCreateHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace, ok := params.GetArguments()["namespace"].(string)
	if !ok || namespace == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace is required")), nil
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

	nadr := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": oadp.OADPGroup + "/" + oadp.OADPVersion,
			"kind":       "NonAdminDownloadRequest",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"target": map[string]any{
					"kind": targetKind,
					"name": targetName,
				},
			},
		},
	}

	created, err := oadp.CreateNonAdminDownloadRequest(params.Context, params.DynamicClient(), nadr)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create non-admin download request: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(created)), nil
}

func initNonAdminDownloadRequestDelete() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_non_admin_download_request_delete",
			Description: "Delete a NonAdminDownloadRequest",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the request",
					},
					"name": {
						Type:        "string",
						Description: "Name of the request to delete",
					},
				},
				Required: []string{"namespace", "name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Delete Non-Admin Download Request",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(true),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: nonAdminDownloadRequestDeleteHandler,
	}
}

func nonAdminDownloadRequestDeleteHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace, ok := params.GetArguments()["namespace"].(string)
	if !ok || namespace == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace is required")), nil
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	err := oadp.DeleteNonAdminDownloadRequest(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete non-admin download request: %w", err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("NonAdminDownloadRequest %s/%s deleted", namespace, name), nil), nil
}
