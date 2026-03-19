package openshift

import (
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/stretchr/testify/suite"
)

type NodesHandlerSuite struct {
	suite.Suite
}

type staticRequest struct {
	args map[string]any
}

func (s staticRequest) GetArguments() map[string]any {
	return s.args
}

func (s *NodesHandlerSuite) TestValidatesInput() {
	s.Run("missing node", func() {
		params := api.ToolHandlerParams{
			ToolCallRequest: staticRequest{args: map[string]any{}},
		}
		result, err := nodesDebugExec(params)
		s.Require().NoError(err)
		s.Require().NotNil(result.Error)
		s.Equal("missing required argument: node", result.Error.Error())
	})

	s.Run("invalid command type", func() {
		params := api.ToolHandlerParams{
			ToolCallRequest: staticRequest{args: map[string]any{
				"node":    "worker-0",
				"command": "ls -la",
			}},
		}
		result, err := nodesDebugExec(params)
		s.Require().NoError(err)
		s.Require().NotNil(result.Error)
		s.Equal("invalid command argument: command must be an array of strings", result.Error.Error())
	})

	s.Run("missing command", func() {
		params := api.ToolHandlerParams{
			ToolCallRequest: staticRequest{args: map[string]any{
				"node": "worker-0",
			}},
		}
		result, err := nodesDebugExec(params)
		s.Require().NoError(err)
		s.Require().NotNil(result.Error)
		s.Contains(result.Error.Error(), "command is required")
	})

	s.Run("empty command array", func() {
		params := api.ToolHandlerParams{
			ToolCallRequest: staticRequest{args: map[string]any{
				"node":    "worker-0",
				"command": []interface{}{},
			}},
		}
		result, err := nodesDebugExec(params)
		s.Require().NoError(err)
		s.Require().NotNil(result.Error)
		s.Contains(result.Error.Error(), "command array cannot be empty")
	})
}

func TestNodesHandler(t *testing.T) {
	suite.Run(t, new(NodesHandlerSuite))
}
