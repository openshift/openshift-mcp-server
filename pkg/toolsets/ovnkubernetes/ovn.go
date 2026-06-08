package ovnkubernetes

import (
	"context"

	"github.com/google/jsonschema-go/jsonschema"
	gosdk "github.com/modelcontextprotocol/go-sdk/mcp"
	ovnmcp "github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/ovn/mcp"
	"k8s.io/utils/ptr"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
)

// newOVNServer creates an ovnmcp.MCPServer that executes OVN CLI commands
// inside the given container of an OVN pod via core.PodsExec.
func newOVNServer(core *kubernetes.Core, container string) (*ovnmcp.MCPServer, error) {
	return ovnmcp.NewMCPServer(func(ctx context.Context, namespace, name, _ string, command []string) (string, string, error) {
		return core.PodsExec(ctx, namespace, name, container, command)
	})
}

// initOVNTools returns all upstream OVN tool registrations adapted to the
// project's api.ServerTool format.
func initOVNTools() []api.ServerTool {
	regs := ovnmcp.AllToolRegistrations()
	tools := make([]api.ServerTool, len(regs))
	for i, reg := range regs {
		tools[i] = api.ServerTool{
			Tool:    mcpToolToAPITool(reg.Tool),
			Handler: makeOVNHandler(reg),
		}
	}
	return tools
}

// mcpToolToAPITool converts a go-sdk Tool into the project's api.Tool.
func mcpToolToAPITool(t *gosdk.Tool) api.Tool {
	schema, _ := t.InputSchema.(*jsonschema.Schema)
	tool := api.Tool{
		Name:        t.Name,
		Description: t.Description,
		InputSchema: schema,
	}
	if t.Annotations != nil {
		tool.Annotations.ReadOnlyHint = ptr.To(t.Annotations.ReadOnlyHint)
		tool.Annotations.IdempotentHint = ptr.To(t.Annotations.IdempotentHint)
		tool.Annotations.OpenWorldHint = t.Annotations.OpenWorldHint
		tool.Annotations.DestructiveHint = t.Annotations.DestructiveHint
		if t.Annotations.Title != "" {
			tool.Annotations.Title = t.Annotations.Title
		}
	}
	// Top-level Title takes precedence per MCP spec:
	// "Display name precedence order is: title, annotations.title, then name."
	if t.Title != "" {
		tool.Annotations.Title = t.Title
	}
	return tool
}

// makeOVNHandler returns a tool handler that executes OVN commands in the
// appropriate container for each call.
//
// A fresh MCPServer is created per invocation because the target container
// (nbdb, sbdb, or northd) may differ between calls.
func makeOVNHandler(reg ovnmcp.ToolRegistration) api.ToolHandlerFunc {
	return func(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
		args := params.GetArguments()
		target := reg.TargetSelector(args)
		container := containerForTarget(target)
		server, err := newOVNServer(kubernetes.NewCore(params), container)
		if err != nil {
			return api.NewToolCallResult("", err), nil
		}
		result, err := reg.Execute(server, params.Context, args)
		if err != nil {
			return api.NewToolCallResult("", err), nil
		}
		return api.NewToolCallResultStructured(result, nil), nil
	}
}

// containerForTarget maps an upstream execution target to a container name.
func containerForTarget(t ovnmcp.ExecTarget) string {
	return string(t)
}
