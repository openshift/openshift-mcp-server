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

func initStorageTools() []api.ServerTool {
	return []api.ServerTool{
		initBSLList(),
		initBSLGet(),
		initBSLCreate(),
		initBSLUpdate(),
		initBSLDelete(),
		initVSLList(),
		initVSLGet(),
		initVSLCreate(),
		initVSLUpdate(),
		initVSLDelete(),
	}
}

func initBSLList() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_backup_storage_location_list",
			Description: "List all BackupStorageLocations configured for OADP",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace containing BSLs (default: openshift-adp)",
					},
				},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: List Backup Storage Locations",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: bslListHandler,
	}
}

func bslListHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	bsls, err := oadp.ListBackupStorageLocations(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list backup storage locations: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(bsls)), nil
}

func initBSLGet() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_backup_storage_location_get",
			Description: "Get detailed information about a BackupStorageLocation including provider, bucket, and status",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the BSL (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the BackupStorageLocation",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Get Backup Storage Location",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: bslGetHandler,
	}
}

func bslGetHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	bsl, err := oadp.GetBackupStorageLocation(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get backup storage location: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(bsl)), nil
}

func initVSLList() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_volume_snapshot_location_list",
			Description: "List all VolumeSnapshotLocations configured for OADP",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace containing VSLs (default: openshift-adp)",
					},
				},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: List Volume Snapshot Locations",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: vslListHandler,
	}
}

func vslListHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	vsls, err := oadp.ListVolumeSnapshotLocations(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list volume snapshot locations: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(vsls)), nil
}

func initVSLGet() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_volume_snapshot_location_get",
			Description: "Get detailed information about a VolumeSnapshotLocation",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the VSL (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the VolumeSnapshotLocation",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Get Volume Snapshot Location",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: vslGetHandler,
	}
}

func vslGetHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	vsl, err := oadp.GetVolumeSnapshotLocation(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get volume snapshot location: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(vsl)), nil
}

func initBSLCreate() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_backup_storage_location_create",
			Description: "Create a BackupStorageLocation for storing backups",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace for the BSL (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name for the BackupStorageLocation",
					},
					"provider": {
						Type:        "string",
						Description: "Storage provider (e.g., aws, azure, gcp)",
					},
					"bucket": {
						Type:        "string",
						Description: "Bucket name for object storage",
					},
					"prefix": {
						Type:        "string",
						Description: "Optional prefix within the bucket",
					},
					"region": {
						Type:        "string",
						Description: "Region for the storage",
					},
					"credentialSecretName": {
						Type:        "string",
						Description: "Name of the secret containing credentials",
					},
					"credentialSecretKey": {
						Type:        "string",
						Description: "Key in the secret containing credentials (default: cloud)",
					},
					"default": {
						Type:        "boolean",
						Description: "Set as the default backup storage location",
					},
				},
				Required: []string{"name", "provider", "bucket"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Create Backup Storage Location",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(false),
			},
		},
		Handler: bslCreateHandler,
	}
}

func bslCreateHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
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

	objectStorage := map[string]any{
		"bucket": bucket,
	}

	if prefix, ok := params.GetArguments()["prefix"].(string); ok && prefix != "" {
		objectStorage["prefix"] = prefix
	}

	spec := map[string]any{
		"provider":      provider,
		"objectStorage": objectStorage,
	}

	if region, ok := params.GetArguments()["region"].(string); ok && region != "" {
		if spec["config"] == nil {
			spec["config"] = map[string]any{}
		}
		spec["config"].(map[string]any)["region"] = region
	}

	if credSecretName, ok := params.GetArguments()["credentialSecretName"].(string); ok && credSecretName != "" {
		credSecretKey := "cloud"
		if v, ok := params.GetArguments()["credentialSecretKey"].(string); ok && v != "" {
			credSecretKey = v
		}
		spec["credential"] = map[string]any{
			"name": credSecretName,
			"key":  credSecretKey,
		}
	}

	if isDefault, ok := params.GetArguments()["default"].(bool); ok {
		spec["default"] = isDefault
	}

	bsl := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": oadp.VeleroGroup + "/" + oadp.VeleroVersion,
			"kind":       "BackupStorageLocation",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": spec,
		},
	}

	created, err := oadp.CreateBackupStorageLocation(params.Context, params.DynamicClient(), bsl)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create backup storage location: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(created)), nil
}

func initBSLUpdate() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_backup_storage_location_update",
			Description: "Update a BackupStorageLocation configuration",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the BSL (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the BackupStorageLocation to update",
					},
					"default": {
						Type:        "boolean",
						Description: "Set as the default backup storage location",
					},
					"accessMode": {
						Type:        "string",
						Description: "Access mode: ReadWrite or ReadOnly",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Update Backup Storage Location",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: bslUpdateHandler,
	}
}

func bslUpdateHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	// Get the existing BSL
	bsl, err := oadp.GetBackupStorageLocation(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get backup storage location: %w", err)), nil
	}

	// Apply updates
	if isDefault, ok := params.GetArguments()["default"].(bool); ok {
		if err := unstructured.SetNestedField(bsl.Object, isDefault, "spec", "default"); err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to set default field: %w", err)), nil
		}
	}

	if accessMode, ok := params.GetArguments()["accessMode"].(string); ok && accessMode != "" {
		if err := unstructured.SetNestedField(bsl.Object, accessMode, "spec", "accessMode"); err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to set accessMode field: %w", err)), nil
		}
	}

	updated, err := oadp.UpdateBackupStorageLocation(params.Context, params.DynamicClient(), bsl)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to update backup storage location: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(updated)), nil
}

func initBSLDelete() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_backup_storage_location_delete",
			Description: "Delete a BackupStorageLocation. Warning: This does not delete backups stored in the location.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the BSL (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the BackupStorageLocation to delete",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Delete Backup Storage Location",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(true),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: bslDeleteHandler,
	}
}

func bslDeleteHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	err := oadp.DeleteBackupStorageLocation(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete backup storage location: %w", err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("BackupStorageLocation %s/%s deleted", namespace, name), nil), nil
}

func initVSLCreate() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_volume_snapshot_location_create",
			Description: "Create a VolumeSnapshotLocation for storing volume snapshots",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace for the VSL (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name for the VolumeSnapshotLocation",
					},
					"provider": {
						Type:        "string",
						Description: "Snapshot provider (e.g., aws, azure, gcp)",
					},
					"region": {
						Type:        "string",
						Description: "Region for snapshots",
					},
					"credentialSecretName": {
						Type:        "string",
						Description: "Name of the secret containing credentials",
					},
					"credentialSecretKey": {
						Type:        "string",
						Description: "Key in the secret containing credentials (default: cloud)",
					},
				},
				Required: []string{"name", "provider"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Create Volume Snapshot Location",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(false),
			},
		},
		Handler: vslCreateHandler,
	}
}

func vslCreateHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	provider, ok := params.GetArguments()["provider"].(string)
	if !ok || provider == "" {
		return api.NewToolCallResult("", fmt.Errorf("provider is required")), nil
	}

	spec := map[string]any{
		"provider": provider,
	}

	config := map[string]any{}
	if region, ok := params.GetArguments()["region"].(string); ok && region != "" {
		config["region"] = region
	}
	if len(config) > 0 {
		spec["config"] = config
	}

	if credSecretName, ok := params.GetArguments()["credentialSecretName"].(string); ok && credSecretName != "" {
		credSecretKey := "cloud"
		if v, ok := params.GetArguments()["credentialSecretKey"].(string); ok && v != "" {
			credSecretKey = v
		}
		spec["credential"] = map[string]any{
			"name": credSecretName,
			"key":  credSecretKey,
		}
	}

	vsl := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": oadp.VeleroGroup + "/" + oadp.VeleroVersion,
			"kind":       "VolumeSnapshotLocation",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": spec,
		},
	}

	created, err := oadp.CreateVolumeSnapshotLocation(params.Context, params.DynamicClient(), vsl)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create volume snapshot location: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(created)), nil
}

func initVSLUpdate() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_volume_snapshot_location_update",
			Description: "Update a VolumeSnapshotLocation configuration",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the VSL (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the VolumeSnapshotLocation to update",
					},
					"region": {
						Type:        "string",
						Description: "Update the region for snapshots",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Update Volume Snapshot Location",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: vslUpdateHandler,
	}
}

func vslUpdateHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	// Get the existing VSL
	vsl, err := oadp.GetVolumeSnapshotLocation(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get volume snapshot location: %w", err)), nil
	}

	// Apply updates
	if region, ok := params.GetArguments()["region"].(string); ok && region != "" {
		config, _, _ := unstructured.NestedMap(vsl.Object, "spec", "config")
		if config == nil {
			config = map[string]any{}
		}
		config["region"] = region
		if err := unstructured.SetNestedMap(vsl.Object, config, "spec", "config"); err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to set config: %w", err)), nil
		}
	}

	updated, err := oadp.UpdateVolumeSnapshotLocation(params.Context, params.DynamicClient(), vsl)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to update volume snapshot location: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(updated)), nil
}

func initVSLDelete() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_volume_snapshot_location_delete",
			Description: "Delete a VolumeSnapshotLocation",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the VSL (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the VolumeSnapshotLocation to delete",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Delete Volume Snapshot Location",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(true),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: vslDeleteHandler,
	}
}

func vslDeleteHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	err := oadp.DeleteVolumeSnapshotLocation(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete volume snapshot location: %w", err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("VolumeSnapshotLocation %s/%s deleted", namespace, name), nil), nil
}
