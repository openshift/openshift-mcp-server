package mcp

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/mustgather"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/netedge"
)

type MustGatherOfflineSuite struct {
	suite.Suite
}

type mockRequest struct {
	args map[string]any
}

func (r mockRequest) GetArguments() map[string]any {
	return r.args
}

func (s *MustGatherOfflineSuite) TestOfflineMustGather() {
	// 1. Load the must-gather provider via mustgather_use tool
	mustgatherToolset := &mustgather.Toolset{}
	mustgatherTools := mustgatherToolset.GetTools(nil)
	var useTool *api.ServerTool
	for _, tool := range mustgatherTools {
		if tool.Tool.Name == "mustgather_use" {
			useTool = &tool
			break
		}
	}
	s.Require().NotNil(useTool, "mustgather_use tool not found")

	_, thisFile, _, _ := runtime.Caller(0)
	archivePath := filepath.Join(filepath.Dir(thisFile), "..", "..", "evals", "testdata", "must-gather")

	useParams := api.ToolHandlerParams{
		Context: context.Background(),
		ToolCallRequest: mockRequest{
			args: map[string]any{
				"path": archivePath,
			},
		},
	}

	useRes, err := useTool.Handler(useParams)
	s.Require().NoError(err, "mustgather_use returned error")
	s.Require().Nil(useRes.Error, "mustgather_use execution error")
	s.T().Log("Loaded must-gather archive successfully via tool")

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
	s.Require().NotNil(endpointsTool, "get_service_endpoints tool not found in netedge toolset")

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
	s.Require().NoError(err, "handler returned error")
	s.Require().Nil(res.Error, "tool execution error")

	s.T().Log("--- RESULT START ---")
	s.T().Log(res.Content)
	s.T().Log("--- RESULT END ---")
}

func TestMustGatherOffline(t *testing.T) {
	suite.Run(t, new(MustGatherOfflineSuite))
}
