package core

import (
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/mcpapps"
	openshiftapps "github.com/containers/kubernetes-mcp-server/pkg/mcpapps/openshift"
)

// withAppOverrides selects downstream presentations at the toolset boundary.
// Only the app HTML changes: tool handlers, schemas, RBAC, resource identity,
// descriptions, and metadata are preserved. New tools keep their upstream app
// until explicitly included here.
func withAppOverrides(tools []api.ServerTool) []api.ServerTool {
	for i := range tools {
		app := tools[i].App
		if app == nil {
			continue
		}
		var constructor func(string, string, ...mcpapps.CustomOption) *api.ToolApp
		switch tools[i].Tool.Name {
		case "namespaces_list", "projects_list", "pods_list", "pods_list_in_namespace", "resources_list", "events_list":
			constructor = openshiftapps.Table
		case "pods_top", "nodes_top":
			constructor = openshiftapps.Metrics
		case "pods_get", "resources_get":
			constructor = openshiftapps.Resource
		default:
			continue
		}
		tools[i].App = constructor(app.URI, app.Name,
			mcpapps.WithDescription(app.Description), mcpapps.WithMetadata(app.Meta))
	}
	return tools
}
