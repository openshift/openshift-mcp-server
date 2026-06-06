package nodes

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
		s.Equal("failed to execute command on node: node parameter required", result.Error.Error())
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
		s.Equal("failed to execute command on node: invalid command argument: command must be an array of strings", result.Error.Error())
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
		s.Equal("failed to execute command on node: invalid command argument: command is required", result.Error.Error())
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
		s.Equal("failed to execute command on node: invalid command argument: command array cannot be empty", result.Error.Error())
	})

	s.Run("fractional timeout rejected", func() {
		params := api.ToolHandlerParams{
			ToolCallRequest: staticRequest{args: map[string]any{
				"node":            "worker-0",
				"command":         []interface{}{"uname"},
				"timeout_seconds": 1.5,
			}},
		}
		result, err := nodesDebugExec(params)
		s.Require().NoError(err)
		s.Require().NotNil(result.Error)
		s.Equal("failed to execute command on node: timeout_seconds must be an integer >= 1", result.Error.Error())
	})

	s.Run("zero timeout rejected", func() {
		params := api.ToolHandlerParams{
			ToolCallRequest: staticRequest{args: map[string]any{
				"node":            "worker-0",
				"command":         []interface{}{"uname"},
				"timeout_seconds": float64(0),
			}},
		}
		result, err := nodesDebugExec(params)
		s.Require().NoError(err)
		s.Require().NotNil(result.Error)
		s.Equal("failed to execute command on node: timeout_seconds must be an integer >= 1", result.Error.Error())
	})

	s.Run("non-numeric timeout rejected", func() {
		params := api.ToolHandlerParams{
			ToolCallRequest: staticRequest{args: map[string]any{
				"node":            "worker-0",
				"command":         []interface{}{"uname"},
				"timeout_seconds": "sixty",
			}},
		}
		result, err := nodesDebugExec(params)
		s.Require().NoError(err)
		s.Require().NotNil(result.Error)
		s.Equal("failed to execute command on node: timeout_seconds must be a numeric value", result.Error.Error())
	})
}

func TestNodesHandler(t *testing.T) {
	suite.Run(t, new(NodesHandlerSuite))
}
