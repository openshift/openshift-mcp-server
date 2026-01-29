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

func initVMRestoreTools() []api.ServerTool {
	return []api.ServerTool{
		initVMBackupDiscoveryList(),
		initVMBackupDiscoveryGet(),
		initVMBackupDiscoveryCreate(),
		initVMBackupDiscoveryDelete(),
		initVMFileRestoreList(),
		initVMFileRestoreGet(),
		initVMFileRestoreCreate(),
		initVMFileRestoreDelete(),
	}
}

// VirtualMachineBackupsDiscovery tools

func initVMBackupDiscoveryList() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_vm_backup_discovery_list",
			Description: "List all VirtualMachineBackupsDiscovery resources for VM backup discovery",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace containing VirtualMachineBackupsDiscoveries (default: openshift-adp)",
					},
				},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: List VM Backup Discoveries",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: vmBackupDiscoveryListHandler,
	}
}

func vmBackupDiscoveryListHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	vmbds, err := oadp.ListVirtualMachineBackupsDiscoveries(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list VM backup discoveries: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(vmbds)), nil
}

func initVMBackupDiscoveryGet() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_vm_backup_discovery_get",
			Description: "Get detailed information about a VirtualMachineBackupsDiscovery including discovered backups",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the VirtualMachineBackupsDiscovery (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the VirtualMachineBackupsDiscovery",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Get VM Backup Discovery",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: vmBackupDiscoveryGetHandler,
	}
}

func vmBackupDiscoveryGetHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	vmbd, err := oadp.GetVirtualMachineBackupsDiscovery(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get VM backup discovery: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(vmbd)), nil
}

func initVMBackupDiscoveryCreate() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_vm_backup_discovery_create",
			Description: "Create a VirtualMachineBackupsDiscovery to discover backups for a virtual machine",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace for the VirtualMachineBackupsDiscovery (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name for the VirtualMachineBackupsDiscovery",
					},
					"vmName": {
						Type:        "string",
						Description: "Name of the virtual machine to discover backups for",
					},
					"vmNamespace": {
						Type:        "string",
						Description: "Namespace of the virtual machine",
					},
				},
				Required: []string{"name", "vmName", "vmNamespace"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Create VM Backup Discovery",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(false),
			},
		},
		Handler: vmBackupDiscoveryCreateHandler,
	}
}

func vmBackupDiscoveryCreateHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	vmName, ok := params.GetArguments()["vmName"].(string)
	if !ok || vmName == "" {
		return api.NewToolCallResult("", fmt.Errorf("vmName is required")), nil
	}

	vmNamespace, ok := params.GetArguments()["vmNamespace"].(string)
	if !ok || vmNamespace == "" {
		return api.NewToolCallResult("", fmt.Errorf("vmNamespace is required")), nil
	}

	vmbd := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": oadp.OADPGroup + "/" + oadp.OADPVersion,
			"kind":       "VirtualMachineBackupsDiscovery",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"virtualMachine": map[string]any{
					"name":      vmName,
					"namespace": vmNamespace,
				},
			},
		},
	}

	created, err := oadp.CreateVirtualMachineBackupsDiscovery(params.Context, params.DynamicClient(), vmbd)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create VM backup discovery: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(created)), nil
}

func initVMBackupDiscoveryDelete() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_vm_backup_discovery_delete",
			Description: "Delete a VirtualMachineBackupsDiscovery",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the VirtualMachineBackupsDiscovery (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the VirtualMachineBackupsDiscovery to delete",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Delete VM Backup Discovery",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(true),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: vmBackupDiscoveryDeleteHandler,
	}
}

func vmBackupDiscoveryDeleteHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	err := oadp.DeleteVirtualMachineBackupsDiscovery(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete VM backup discovery: %w", err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("VirtualMachineBackupsDiscovery %s/%s deleted", namespace, name), nil), nil
}

// VirtualMachineFileRestore tools

func initVMFileRestoreList() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_vm_file_restore_list",
			Description: "List all VirtualMachineFileRestore resources for VM file-level restore",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace containing VirtualMachineFileRestores (default: openshift-adp)",
					},
				},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: List VM File Restores",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: vmFileRestoreListHandler,
	}
}

func vmFileRestoreListHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	vmfrs, err := oadp.ListVirtualMachineFileRestores(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list VM file restores: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(vmfrs)), nil
}

func initVMFileRestoreGet() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_vm_file_restore_get",
			Description: "Get detailed information about a VirtualMachineFileRestore including restore status",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the VirtualMachineFileRestore (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the VirtualMachineFileRestore",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Get VM File Restore",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: vmFileRestoreGetHandler,
	}
}

func vmFileRestoreGetHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	vmfr, err := oadp.GetVirtualMachineFileRestore(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get VM file restore: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(vmfr)), nil
}

func initVMFileRestoreCreate() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_vm_file_restore_create",
			Description: "Create a VirtualMachineFileRestore to restore specific files from a VM backup",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace for the VirtualMachineFileRestore (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name for the VirtualMachineFileRestore",
					},
					"backupName": {
						Type:        "string",
						Description: "Name of the backup to restore files from",
					},
					"vmName": {
						Type:        "string",
						Description: "Name of the virtual machine",
					},
					"vmNamespace": {
						Type:        "string",
						Description: "Namespace of the virtual machine",
					},
					"filePaths": {
						Type:        "array",
						Description: "List of file paths to restore",
						Items:       &jsonschema.Schema{Type: "string"},
					},
				},
				Required: []string{"name", "backupName", "vmName", "vmNamespace"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Create VM File Restore",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(false),
			},
		},
		Handler: vmFileRestoreCreateHandler,
	}
}

func vmFileRestoreCreateHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
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

	vmName, ok := params.GetArguments()["vmName"].(string)
	if !ok || vmName == "" {
		return api.NewToolCallResult("", fmt.Errorf("vmName is required")), nil
	}

	vmNamespace, ok := params.GetArguments()["vmNamespace"].(string)
	if !ok || vmNamespace == "" {
		return api.NewToolCallResult("", fmt.Errorf("vmNamespace is required")), nil
	}

	spec := map[string]any{
		"backupName": backupName,
		"virtualMachine": map[string]any{
			"name":      vmName,
			"namespace": vmNamespace,
		},
	}

	if filePaths, ok := params.GetArguments()["filePaths"].([]any); ok && len(filePaths) > 0 {
		spec["filePaths"] = filePaths
	}

	vmfr := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": oadp.OADPGroup + "/" + oadp.OADPVersion,
			"kind":       "VirtualMachineFileRestore",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": spec,
		},
	}

	created, err := oadp.CreateVirtualMachineFileRestore(params.Context, params.DynamicClient(), vmfr)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create VM file restore: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(created)), nil
}

func initVMFileRestoreDelete() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_vm_file_restore_delete",
			Description: "Delete a VirtualMachineFileRestore",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the VirtualMachineFileRestore (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the VirtualMachineFileRestore to delete",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Delete VM File Restore",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(true),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: vmFileRestoreDeleteHandler,
	}
}

func vmFileRestoreDeleteHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	err := oadp.DeleteVirtualMachineFileRestore(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete VM file restore: %w", err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("VirtualMachineFileRestore %s/%s deleted", namespace, name), nil), nil
}
