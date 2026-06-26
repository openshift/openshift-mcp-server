package mustgather_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/mustgather"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/netedge"
)

type mockRequest struct {
	args map[string]any
}

func (r mockRequest) GetArguments() map[string]any {
	return r.args
}

func TestEvalOffline(t *testing.T) {
	// 1. Load the must-gather provider
	archivePath := "../../../evals/testdata/must-gather"
	p, err := mg.NewProvider(archivePath)
	if err != nil {
		t.Fatalf("failed to create must-gather provider: %v", err)
	}
	mustgather.SetProvider(p)
	fmt.Println("Loaded must-gather archive successfully")

	// 2. Get the "get_service_endpoints" tool from netedge toolset
	netedgeToolset := &netedge.Toolset{}
	tools := netedgeToolset.GetTools(nil)
	var endpointsTool *api.ServerTool
	for _, tool := range tools {
		if tool.Tool.Name == "get_service_endpoints" {
			endpointsTool = &tool
			break
		}
	}
	if endpointsTool == nil {
		t.Fatalf("get_service_endpoints tool not found in netedge toolset")
	}

	// 3. Call get_service_endpoints
	params := api.ToolHandlerParams{
		Context: context.Background(),
		ToolCallRequest: mockRequest{
			args: map[string]any{
				"namespace": "openshift-ingress",
				"service":   "router-default",
			},
		},
	}

	res, err := endpointsTool.Handler(params)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if res.Error != nil {
		t.Fatalf("tool execution error: %v", res.Error)
	}

	fmt.Println("--- RESULT START ---")
	fmt.Println(res.Content)
	fmt.Println("--- RESULT END ---")
}
