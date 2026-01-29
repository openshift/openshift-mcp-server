package oadp

import (
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/oadp"
	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func initPodVolumeTools() []api.ServerTool {
	return []api.ServerTool{
		initPodVolumeBackupList(),
		initPodVolumeBackupGet(),
		initPodVolumeRestoreList(),
		initPodVolumeRestoreGet(),
	}
}

func initPodVolumeBackupList() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_pod_volume_backup_list",
			Description: "List all PodVolumeBackups which track Restic/Kopia volume backup operations",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace containing PodVolumeBackups (default: openshift-adp)",
					},
					"labelSelector": {
						Type:        "string",
						Description: "Label selector to filter (e.g., 'velero.io/backup-name=my-backup')",
					},
				},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: List Pod Volume Backups",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: podVolumeBackupListHandler,
	}
}

func podVolumeBackupListHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	labelSelector := ""
	if v, ok := params.GetArguments()["labelSelector"].(string); ok {
		labelSelector = v
	}

	pvbs, err := oadp.ListPodVolumeBackups(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list pod volume backups: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(pvbs)), nil
}

func initPodVolumeBackupGet() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_pod_volume_backup_get",
			Description: "Get detailed information about a PodVolumeBackup including progress and status",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the PodVolumeBackup (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the PodVolumeBackup",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Get Pod Volume Backup",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: podVolumeBackupGetHandler,
	}
}

func podVolumeBackupGetHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	pvb, err := oadp.GetPodVolumeBackup(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get pod volume backup: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(pvb)), nil
}

func initPodVolumeRestoreList() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_pod_volume_restore_list",
			Description: "List all PodVolumeRestores which track Restic/Kopia volume restore operations",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace containing PodVolumeRestores (default: openshift-adp)",
					},
					"labelSelector": {
						Type:        "string",
						Description: "Label selector to filter (e.g., 'velero.io/restore-name=my-restore')",
					},
				},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: List Pod Volume Restores",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: podVolumeRestoreListHandler,
	}
}

func podVolumeRestoreListHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	labelSelector := ""
	if v, ok := params.GetArguments()["labelSelector"].(string); ok {
		labelSelector = v
	}

	pvrs, err := oadp.ListPodVolumeRestores(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list pod volume restores: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(pvrs)), nil
}

func initPodVolumeRestoreGet() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_pod_volume_restore_get",
			Description: "Get detailed information about a PodVolumeRestore including progress and status",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the PodVolumeRestore (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the PodVolumeRestore",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Get Pod Volume Restore",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: podVolumeRestoreGetHandler,
	}
}

func podVolumeRestoreGetHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	pvr, err := oadp.GetPodVolumeRestore(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get pod volume restore: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(pvr)), nil
}
