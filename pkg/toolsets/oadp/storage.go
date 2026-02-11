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

// StorageAction represents the action to perform on storage locations
type StorageAction string

const (
	StorageActionList   StorageAction = "list"
	StorageActionGet    StorageAction = "get"
	StorageActionCreate StorageAction = "create"
	StorageActionUpdate StorageAction = "update"
	StorageActionDelete StorageAction = "delete"
)

// StorageType represents the type of storage location
type StorageType string

const (
	StorageTypeBSL StorageType = "bsl"
	StorageTypeVSL StorageType = "vsl"
)

func initStorageTools() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "oadp_storage_location",
				Description: "Manage Velero storage locations (BackupStorageLocation and VolumeSnapshotLocation): list, get, create, update, or delete",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"action": {
							Type:        "string",
							Enum:        []any{string(StorageActionList), string(StorageActionGet), string(StorageActionCreate), string(StorageActionUpdate), string(StorageActionDelete)},
							Description: "Action to perform: 'list', 'get', 'create', 'update', or 'delete'",
						},
						"type": {
							Type:        "string",
							Enum:        []any{string(StorageTypeBSL), string(StorageTypeVSL)},
							Description: "Storage location type: 'bsl' (BackupStorageLocation) or 'vsl' (VolumeSnapshotLocation)",
						},
						"namespace": {
							Type:        "string",
							Description: "Namespace containing storage locations (default: openshift-adp)",
						},
						"name": {
							Type:        "string",
							Description: "Name of the storage location (required for get, create, update, delete)",
						},
						"provider": {
							Type:        "string",
							Description: "Storage provider e.g., aws, azure, gcp (for create)",
						},
						"bucket": {
							Type:        "string",
							Description: "Bucket name for object storage (for BSL create)",
						},
						"prefix": {
							Type:        "string",
							Description: "Optional prefix within the bucket (for BSL create)",
						},
						"region": {
							Type:        "string",
							Description: "Region for the storage (for create/update)",
						},
						"credentialSecretName": {
							Type:        "string",
							Description: "Name of the secret containing credentials (for create)",
						},
						"credentialSecretKey": {
							Type:        "string",
							Description: "Key in the secret containing credentials (default: cloud)",
						},
						"default": {
							Type:        "boolean",
							Description: "Set as the default storage location (for BSL create/update)",
						},
						"accessMode": {
							Type:        "string",
							Description: "Access mode: ReadWrite or ReadOnly (for BSL update)",
						},
					},
					Required: []string{"action", "type"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "OADP: Storage Location",
					ReadOnlyHint:    ptr.To(false),
					DestructiveHint: ptr.To(false),
				},
			},
			Handler: storageHandler,
		},
	}
}

func storageHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	action, err := api.RequiredString(params, "action")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	storageType, err := api.RequiredString(params, "type")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	namespace := api.OptionalString(params, "namespace", oadp.DefaultOADPNamespace)

	switch StorageType(storageType) {
	case StorageTypeBSL:
		return handleBSL(params, namespace, StorageAction(action))
	case StorageTypeVSL:
		return handleVSL(params, namespace, StorageAction(action))
	default:
		return api.NewToolCallResult("", fmt.Errorf("invalid type '%s': must be 'bsl' or 'vsl'", storageType)), nil
	}
}

func handleBSL(params api.ToolHandlerParams, namespace string, action StorageAction) (*api.ToolCallResult, error) {
	switch action {
	case StorageActionList:
		bsls, err := oadp.ListBackupStorageLocations(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{})
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to list backup storage locations: %w", err)), nil
		}
		return api.NewToolCallResult(params.ListOutput.PrintObj(bsls)), nil

	case StorageActionGet:
		name, ok := params.GetArguments()["name"].(string)
		if !ok || name == "" {
			return api.NewToolCallResult("", fmt.Errorf("name is required for get action")), nil
		}
		bsl, err := oadp.GetBackupStorageLocation(params.Context, params.DynamicClient(), namespace, name)
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to get backup storage location: %w", err)), nil
		}
		return api.NewToolCallResult(params.ListOutput.PrintObj(bsl)), nil

	case StorageActionCreate:
		return handleBSLCreate(params, namespace)

	case StorageActionUpdate:
		return handleBSLUpdate(params, namespace)

	case StorageActionDelete:
		name, ok := params.GetArguments()["name"].(string)
		if !ok || name == "" {
			return api.NewToolCallResult("", fmt.Errorf("name is required for delete action")), nil
		}
		err := oadp.DeleteBackupStorageLocation(params.Context, params.DynamicClient(), namespace, name)
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to delete backup storage location: %w", err)), nil
		}
		return api.NewToolCallResult(fmt.Sprintf("BackupStorageLocation %s/%s deleted", namespace, name), nil), nil

	default:
		return api.NewToolCallResult("", fmt.Errorf("invalid action '%s'", action)), nil
	}
}

func handleBSLCreate(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required for create action")), nil
	}

	provider, ok := params.GetArguments()["provider"].(string)
	if !ok || provider == "" {
		return api.NewToolCallResult("", fmt.Errorf("provider is required for create action")), nil
	}

	bucket, ok := params.GetArguments()["bucket"].(string)
	if !ok || bucket == "" {
		return api.NewToolCallResult("", fmt.Errorf("bucket is required for BSL create action")), nil
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

func handleBSLUpdate(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required for update action")), nil
	}

	bsl, err := oadp.GetBackupStorageLocation(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get backup storage location: %w", err)), nil
	}

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

func handleVSL(params api.ToolHandlerParams, namespace string, action StorageAction) (*api.ToolCallResult, error) {
	switch action {
	case StorageActionList:
		vsls, err := oadp.ListVolumeSnapshotLocations(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{})
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to list volume snapshot locations: %w", err)), nil
		}
		return api.NewToolCallResult(params.ListOutput.PrintObj(vsls)), nil

	case StorageActionGet:
		name, ok := params.GetArguments()["name"].(string)
		if !ok || name == "" {
			return api.NewToolCallResult("", fmt.Errorf("name is required for get action")), nil
		}
		vsl, err := oadp.GetVolumeSnapshotLocation(params.Context, params.DynamicClient(), namespace, name)
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to get volume snapshot location: %w", err)), nil
		}
		return api.NewToolCallResult(params.ListOutput.PrintObj(vsl)), nil

	case StorageActionCreate:
		return handleVSLCreate(params, namespace)

	case StorageActionUpdate:
		return handleVSLUpdate(params, namespace)

	case StorageActionDelete:
		name, ok := params.GetArguments()["name"].(string)
		if !ok || name == "" {
			return api.NewToolCallResult("", fmt.Errorf("name is required for delete action")), nil
		}
		err := oadp.DeleteVolumeSnapshotLocation(params.Context, params.DynamicClient(), namespace, name)
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to delete volume snapshot location: %w", err)), nil
		}
		return api.NewToolCallResult(fmt.Sprintf("VolumeSnapshotLocation %s/%s deleted", namespace, name), nil), nil

	default:
		return api.NewToolCallResult("", fmt.Errorf("invalid action '%s'", action)), nil
	}
}

func handleVSLCreate(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required for create action")), nil
	}

	provider, ok := params.GetArguments()["provider"].(string)
	if !ok || provider == "" {
		return api.NewToolCallResult("", fmt.Errorf("provider is required for create action")), nil
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

func handleVSLUpdate(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required for update action")), nil
	}

	vsl, err := oadp.GetVolumeSnapshotLocation(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get volume snapshot location: %w", err)), nil
	}

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
