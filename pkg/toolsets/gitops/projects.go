package gitops

import (
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/output"
)

func initProjects() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name: "gitops_projects_list",
				Description: "List ArgoCD AppProjects with their allowed source repositories and " +
					"destination clusters/namespaces.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Namespace where ArgoCD projects are defined (auto-detected if not provided)",
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "GitOps: List Projects",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: projectsList,
		},
		{
			Tool: api.Tool{
				Name: "gitops_project_get",
				Description: "Get detailed information about an ArgoCD AppProject including allowed source repos, " +
					"destinations, cluster resource whitelist, and roles.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"name": {
							Type:        "string",
							Description: "Name of the ArgoCD AppProject",
						},
						"namespace": {
							Type:        "string",
							Description: "Namespace where the ArgoCD project is defined (auto-detected if not provided)",
						},
					},
					Required: []string{"name"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "GitOps: Get Project",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: projectGet,
		},
	}
}

func projectsList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := api.OptionalString(params, "namespace", "")

	client := newGitOpsClient(params)
	projects, err := client.appProjectsList(params, namespace, metav1.ListOptions{})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list ArgoCD AppProjects: %w", err)), nil
	}

	yamlOut, err := output.MarshalYaml(projects)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal AppProjects: %w", err)), nil
	}
	return api.NewToolCallResult(yamlOut, nil), nil
}

func projectGet(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	name, err := api.RequiredString(params, "name")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	namespace := api.OptionalString(params, "namespace", "")

	client := newGitOpsClient(params)
	project, err := client.appProjectGet(params, namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get ArgoCD AppProject %q: %w", name, err)), nil
	}

	yamlOut, err := output.MarshalYaml(project)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal AppProject: %w", err)), nil
	}
	return api.NewToolCallResult(yamlOut, nil), nil
}
