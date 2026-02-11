package oadp

import (
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/oadp"
	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// RepositoryAction represents the action to perform on backup repositories
type RepositoryAction string

const (
	RepositoryActionList   RepositoryAction = "list"
	RepositoryActionGet    RepositoryAction = "get"
	RepositoryActionDelete RepositoryAction = "delete"
)

func initBackupRepositoryTools() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "oadp_repository",
				Description: "Manage Velero BackupRepository resources (connections to backup storage): list, get, or delete",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"action": {
							Type:        "string",
							Enum:        []any{string(RepositoryActionList), string(RepositoryActionGet), string(RepositoryActionDelete)},
							Description: "Action to perform: 'list', 'get', or 'delete'",
						},
						"namespace": {
							Type:        "string",
							Description: "Namespace containing repositories (default: openshift-adp)",
						},
						"name": {
							Type:        "string",
							Description: "Name of the repository (required for get, delete)",
						},
					},
					Required: []string{"action"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "OADP: Backup Repository",
					ReadOnlyHint:    ptr.To(false),
					DestructiveHint: ptr.To(false),
				},
			},
			Handler: repositoryHandler,
		},
	}
}

func repositoryHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	action, err := api.RequiredString(params, "action")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	namespace := api.OptionalString(params, "namespace", oadp.DefaultOADPNamespace)

	switch RepositoryAction(action) {
	case RepositoryActionList:
		repos, err := oadp.ListBackupRepositories(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{})
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to list backup repositories: %w", err)), nil
		}
		return api.NewToolCallResult(params.ListOutput.PrintObj(repos)), nil

	case RepositoryActionGet:
		name, ok := params.GetArguments()["name"].(string)
		if !ok || name == "" {
			return api.NewToolCallResult("", fmt.Errorf("name is required for get action")), nil
		}
		repo, err := oadp.GetBackupRepository(params.Context, params.DynamicClient(), namespace, name)
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to get backup repository: %w", err)), nil
		}
		return api.NewToolCallResult(params.ListOutput.PrintObj(repo)), nil

	case RepositoryActionDelete:
		name, ok := params.GetArguments()["name"].(string)
		if !ok || name == "" {
			return api.NewToolCallResult("", fmt.Errorf("name is required for delete action")), nil
		}
		err := oadp.DeleteBackupRepository(params.Context, params.DynamicClient(), namespace, name)
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to delete backup repository: %w", err)), nil
		}
		return api.NewToolCallResult(fmt.Sprintf("BackupRepository %s/%s deleted", namespace, name), nil), nil

	default:
		return api.NewToolCallResult("", fmt.Errorf("invalid action '%s': must be one of 'list', 'get', 'delete'", action)), nil
	}
}
