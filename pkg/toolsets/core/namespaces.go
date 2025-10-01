package core

import (
	"context"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/utils/ptr"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	internalk8s "github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
)

func initNamespaces(o internalk8s.Openshift) []api.ServerTool {
	ret := make([]api.ServerTool, 0)
	ret = append(ret, api.ServerTool{
		Tool: api.Tool{
			Name:        "namespaces_list",
			Description: "List all the Kubernetes namespaces in the current cluster",
			InputSchema: &jsonschema.Schema{
				Type: "object",
			},
			Annotations: api.ToolAnnotations{
				Title:           "Namespaces: List",
				ReadOnlyHint:    ptr.To(true),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(false),
				OpenWorldHint:   ptr.To(true),
			},
		}, Handler: namespacesList,
	})
	if o.IsOpenShift(context.Background()) {
		ret = append(ret, api.ServerTool{
			Tool: api.Tool{
				Name:        "projects_list",
				Description: "List all the OpenShift projects in the current cluster",
				InputSchema: &jsonschema.Schema{
					Type: "object",
				},
				Annotations: api.ToolAnnotations{
					Title:           "Projects: List",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					IdempotentHint:  ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			}, Handler: projectsList,
		})
	}
	return ret
}

func namespacesList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	ret, err := params.NamespacesList(params, internalk8s.ResourceListOptions{AsTable: params.ListOutput.AsTable()})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list namespaces: %v", err)), nil
	}
	return api.NewToolCallResult(params.ListOutput.PrintObj(ret)), nil
}

func projectsList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	ret, err := params.ProjectsList(params, internalk8s.ResourceListOptions{AsTable: params.ListOutput.AsTable()})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list projects: %v", err)), nil
	}
	return api.NewToolCallResult(params.ListOutput.PrintObj(ret)), nil
}
