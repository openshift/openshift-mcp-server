package oadp

import (
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/oadp"
	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func initBackupRepositoryTools() []api.ServerTool {
	return []api.ServerTool{
		initBackupRepositoryList(),
		initBackupRepositoryGet(),
		initBackupRepositoryDelete(),
	}
}

func initBackupRepositoryList() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_backup_repository_list",
			Description: "List all Velero BackupRepositories which manage connections to backup storage",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace containing backup repositories (default: openshift-adp)",
					},
				},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: List Backup Repositories",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: backupRepositoryListHandler,
	}
}

func backupRepositoryListHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	repos, err := oadp.ListBackupRepositories(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list backup repositories: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(repos)), nil
}

func initBackupRepositoryGet() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_backup_repository_get",
			Description: "Get detailed information about a BackupRepository including status and restic/kopia repository info",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the backup repository (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the BackupRepository",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Get Backup Repository",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: backupRepositoryGetHandler,
	}
}

func backupRepositoryGetHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	repo, err := oadp.GetBackupRepository(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get backup repository: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(repo)), nil
}

func initBackupRepositoryDelete() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_backup_repository_delete",
			Description: "Delete a BackupRepository. Use with caution as this removes the repository connection.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the backup repository (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the BackupRepository to delete",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Delete Backup Repository",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(true),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: backupRepositoryDeleteHandler,
	}
}

func backupRepositoryDeleteHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	err := oadp.DeleteBackupRepository(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete backup repository: %w", err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("BackupRepository %s/%s deleted", namespace, name), nil), nil
}
