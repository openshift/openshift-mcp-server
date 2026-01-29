package oadp

import (
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/oadp"
	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func initStorageTools() []api.ServerTool {
	return []api.ServerTool{
		initBSLList(),
		initBSLGet(),
		initVSLList(),
		initVSLGet(),
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
