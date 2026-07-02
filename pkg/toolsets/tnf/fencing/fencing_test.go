package fencing

import (
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/stretchr/testify/suite"
)

type FencingHandlerSuite struct {
	suite.Suite
}

type staticRequest struct {
	args map[string]any
}

func (s staticRequest) GetArguments() map[string]any {
	return s.args
}

func (s *FencingHandlerSuite) TestInvalidNamespaceType() {
	params := api.ToolHandlerParams{
		ToolCallRequest: staticRequest{args: map[string]any{
			"namespace": 12345,
		}},
	}
	result, err := checkFencingHealth(params)
	s.Require().NoError(err)
	s.Require().NotNil(result.Error)
	s.Contains(result.Error.Error(), "failed to check fencing health")
}

func TestFencingHandler(t *testing.T) {
	suite.Run(t, new(FencingHandlerSuite))
}
