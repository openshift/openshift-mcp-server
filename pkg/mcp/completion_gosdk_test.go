package mcp

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
)

type CompletionSuite struct {
	suite.Suite
}

// newServerWithRegistry returns a Server whose completion registry is seeded
// from rts, without spinning up the full SDK server.
func (s *CompletionSuite) newServerWithRegistry(rts ...api.ServerResourceTemplate) *Server {
	srv := &Server{}
	srv.completions.Store(buildCompletionRegistry(rts))
	return srv
}

func completeRequest(refType, uri, arg, value string, resolved map[string]string) *mcp.CompleteRequest {
	params := &mcp.CompleteParams{
		Ref:      &mcp.CompleteReference{Type: refType, URI: uri},
		Argument: mcp.CompleteParamsArgument{Name: arg, Value: value},
	}
	if resolved != nil {
		params.Context = &mcp.CompleteContext{Arguments: resolved}
	}
	return &mcp.CompleteRequest{Params: params}
}

func (s *CompletionSuite) TestBuildCompletionRegistry() {
	s.Run("skips templates without a completion handler", func() {
		reg := buildCompletionRegistry([]api.ServerResourceTemplate{
			{ResourceTemplate: api.ResourceTemplate{URITemplate: "x://a"}},
			{
				ResourceTemplate:  api.ResourceTemplate{URITemplate: "x://b"},
				CompletionHandler: func(context.Context, string, string, map[string]string) ([]string, error) { return nil, nil },
			},
		})
		s.Require().NotNil(reg)
		s.NotContains(reg.resourceTemplates, "x://a")
		s.Contains(reg.resourceTemplates, "x://b")
	})

	s.Run("nil input yields a non-nil empty registry", func() {
		reg := buildCompletionRegistry(nil)
		s.Require().NotNil(reg)
		s.Empty(reg.resourceTemplates)
	})
}

func (s *CompletionSuite) TestHandleComplete() {
	tmpl := "x://{id}/dir{/path*}"
	handler := func(_ context.Context, argument, value string, resolved map[string]string) ([]string, error) {
		if argument != "path" {
			return nil, nil
		}
		return []string{"got:" + value + ":id=" + resolved["id"]}, nil
	}
	srv := s.newServerWithRegistry(api.ServerResourceTemplate{
		ResourceTemplate:  api.ResourceTemplate{URITemplate: tmpl},
		CompletionHandler: handler,
	})

	s.Run("dispatches ref/resource to the matching handler", func() {
		res, err := srv.handleComplete(context.Background(),
			completeRequest("ref/resource", tmpl, "path", "au", map[string]string{"id": "mg-1"}))
		s.Require().NoError(err)
		s.Equal([]string{"got:au:id=mg-1"}, res.Completion.Values)
		s.Equal(1, res.Completion.Total)
		s.False(res.Completion.HasMore)
	})

	s.Run("nil context arguments are handled", func() {
		res, err := srv.handleComplete(context.Background(),
			completeRequest("ref/resource", tmpl, "path", "", nil))
		s.Require().NoError(err)
		s.Equal([]string{"got::id="}, res.Completion.Values)
	})

	s.Run("unknown template URI yields empty result", func() {
		res, err := srv.handleComplete(context.Background(),
			completeRequest("ref/resource", "x://other", "path", "", nil))
		s.Require().NoError(err)
		s.Empty(res.Completion.Values)
		s.NotNil(res.Completion.Values, "values must serialize as [] not null")
	})

	s.Run("ref/prompt is reserved and yields empty result", func() {
		res, err := srv.handleComplete(context.Background(),
			completeRequest("ref/prompt", "", "path", "", nil))
		s.Require().NoError(err)
		s.Empty(res.Completion.Values)
	})

	s.Run("nil ref and nil params yield empty result", func() {
		res, err := srv.handleComplete(context.Background(), &mcp.CompleteRequest{Params: &mcp.CompleteParams{}})
		s.Require().NoError(err)
		s.Empty(res.Completion.Values)

		res, err = srv.handleComplete(context.Background(), &mcp.CompleteRequest{})
		s.Require().NoError(err)
		s.Empty(res.Completion.Values)
	})

	s.Run("handler error yields empty result, not an RPC error", func() {
		errSrv := s.newServerWithRegistry(api.ServerResourceTemplate{
			ResourceTemplate: api.ResourceTemplate{URITemplate: tmpl},
			CompletionHandler: func(context.Context, string, string, map[string]string) ([]string, error) {
				return nil, errors.New("boom")
			},
		})
		res, err := errSrv.handleComplete(context.Background(),
			completeRequest("ref/resource", tmpl, "path", "", nil))
		s.Require().NoError(err)
		s.Empty(res.Completion.Values)
	})

	s.Run("values are capped and HasMore/Total are set", func() {
		big := make([]string, maxCompletionValues+5)
		for i := range big {
			big[i] = strconv.Itoa(i)
		}
		capSrv := s.newServerWithRegistry(api.ServerResourceTemplate{
			ResourceTemplate: api.ResourceTemplate{URITemplate: tmpl},
			CompletionHandler: func(context.Context, string, string, map[string]string) ([]string, error) {
				return big, nil
			},
		})
		res, err := capSrv.handleComplete(context.Background(),
			completeRequest("ref/resource", tmpl, "path", "", nil))
		s.Require().NoError(err)
		s.Len(res.Completion.Values, maxCompletionValues)
		s.Equal(maxCompletionValues+5, res.Completion.Total)
		s.True(res.Completion.HasMore)
	})
}

func TestCompletion(t *testing.T) {
	suite.Run(t, new(CompletionSuite))
}
