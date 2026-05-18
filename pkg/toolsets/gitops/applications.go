package gitops

import (
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/output"
)

func initApplications() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name: "gitops_applications_list",
				Description: "List ArgoCD/OpenShift GitOps applications with sync and health status overview. " +
					"Returns application names, sync status (Synced/OutOfSync/Unknown), health status " +
					"(Healthy/Degraded/Progressing/Missing/Suspended/Unknown), source repository, and destination.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Namespace where ArgoCD applications are defined (auto-detected if not provided: 'openshift-gitops' for OpenShift GitOps, 'argocd' for generic ArgoCD)",
						},
						"project": {
							Type:        "string",
							Description: "ArgoCD project name to filter applications by (Optional)",
						},
						"labelSelector": {
							Type:        "string",
							Description: "Kubernetes label selector to filter applications (e.g. 'app.kubernetes.io/instance=my-app') (Optional)",
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "GitOps: List Applications",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: applicationsList,
		},
		{
			Tool: api.Tool{
				Name: "gitops_application_get",
				Description: "Get detailed information about a specific ArgoCD application including its full spec " +
					"(source, destination, project) and status (sync, health, conditions, operation state).",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"name": {
							Type:        "string",
							Description: "Name of the ArgoCD application",
						},
						"namespace": {
							Type:        "string",
							Description: "Namespace where the ArgoCD application is defined (auto-detected if not provided)",
						},
					},
					Required: []string{"name"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "GitOps: Get Application",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: applicationGet,
		},
	}
}

func applicationsList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := api.OptionalString(params, "namespace", "")
	project := api.OptionalString(params, "project", "")
	labelSelector := api.OptionalString(params, "labelSelector", "")

	opts := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	client := newGitOpsClient(params)
	apps, err := client.applicationsList(params, namespace, opts)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list ArgoCD applications: %w", err)), nil
	}

	if project != "" {
		filtered := apps.Items[:0]
		for i := range apps.Items {
			if spec, ok := apps.Items[i].UnstructuredContent()["spec"].(map[string]any); ok {
				if p, ok := spec["project"].(string); ok && p == project {
					filtered = append(filtered, apps.Items[i])
				}
			}
		}
		apps.Items = filtered
	}

	yamlOut, err := output.MarshalYaml(apps)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal applications: %w", err)), nil
	}
	return api.NewToolCallResult(yamlOut, nil), nil
}

func applicationGet(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	name, err := api.RequiredString(params, "name")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	namespace := api.OptionalString(params, "namespace", "")

	client := newGitOpsClient(params)
	app, err := client.applicationGet(params, namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get ArgoCD application %q: %w", name, err)), nil
	}

	yamlOut, err := output.MarshalYaml(app)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal application: %w", err)), nil
	}
	return api.NewToolCallResult(yamlOut, nil), nil
}
