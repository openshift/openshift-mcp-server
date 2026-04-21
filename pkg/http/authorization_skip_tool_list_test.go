package http

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"

	"github.com/containers/kubernetes-mcp-server/internal/test"
)

type IsToolsListRequestSuite struct {
	suite.Suite
}

func (s *IsToolsListRequestSuite) newRequest(body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(body))
	return r
}

func (s *IsToolsListRequestSuite) TestValidToolsListRequest() {
	s.Run("minimal request", func() {
		r := s.newRequest(`{"jsonrpc":"2.0","method":"tools/list","id":1}`)
		s.True(isToolsListRequest(r))
	})
	s.Run("with string id", func() {
		r := s.newRequest(`{"jsonrpc":"2.0","method":"tools/list","id":"abc"}`)
		s.True(isToolsListRequest(r))
	})
	s.Run("with params cursor", func() {
		r := s.newRequest(`{"jsonrpc":"2.0","method":"tools/list","id":1,"params":{"cursor":"next"}}`)
		s.True(isToolsListRequest(r))
	})
	s.Run("with empty params", func() {
		r := s.newRequest(`{"jsonrpc":"2.0","method":"tools/list","id":1,"params":{}}`)
		s.True(isToolsListRequest(r))
	})
}

func (s *IsToolsListRequestSuite) TestRejectsNonToolsListMethods() {
	s.Run("tools/call", func() {
		r := s.newRequest(`{"jsonrpc":"2.0","method":"tools/call","id":1,"params":{"name":"foo"}}`)
		s.False(isToolsListRequest(r))
	})
	s.Run("initialize", func() {
		r := s.newRequest(`{"jsonrpc":"2.0","method":"initialize","id":1}`)
		s.False(isToolsListRequest(r))
	})
	s.Run("resources/read", func() {
		r := s.newRequest(`{"jsonrpc":"2.0","method":"resources/read","id":1}`)
		s.False(isToolsListRequest(r))
	})
}

func (s *IsToolsListRequestSuite) TestRejectsNotifications() {
	s.Run("missing id", func() {
		r := s.newRequest(`{"jsonrpc":"2.0","method":"tools/list"}`)
		s.False(isToolsListRequest(r))
	})
	s.Run("null id", func() {
		r := s.newRequest(`{"jsonrpc":"2.0","method":"tools/list","id":null}`)
		s.False(isToolsListRequest(r))
	})
}

func (s *IsToolsListRequestSuite) TestRejectsBatchedRequests() {
	r := s.newRequest(`[{"jsonrpc":"2.0","method":"tools/list","id":1},{"jsonrpc":"2.0","method":"tools/call","id":2}]`)
	s.False(isToolsListRequest(r))
}

func (s *IsToolsListRequestSuite) TestRejectsInvalidJSON() {
	s.Run("malformed json", func() {
		r := s.newRequest(`{not valid json}`)
		s.False(isToolsListRequest(r))
	})
	s.Run("empty body", func() {
		r := s.newRequest(``)
		s.False(isToolsListRequest(r))
	})
	s.Run("whitespace only", func() {
		r := s.newRequest(`   `)
		s.False(isToolsListRequest(r))
	})
}

func (s *IsToolsListRequestSuite) TestRejectsWrongJSONRPCVersion() {
	r := s.newRequest(`{"jsonrpc":"1.0","method":"tools/list","id":1}`)
	s.False(isToolsListRequest(r))
}

func (s *IsToolsListRequestSuite) TestNilBody() {
	r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	r.Body = nil
	s.False(isToolsListRequest(r))
}

func (s *IsToolsListRequestSuite) TestNoBody() {
	r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	r.Body = http.NoBody
	s.False(isToolsListRequest(r))
}

func (s *IsToolsListRequestSuite) TestBodyRestoredAfterCheck() {
	original := `{"jsonrpc":"2.0","method":"tools/list","id":1}`
	s.Run("body restored when true", func() {
		r := s.newRequest(original)
		s.True(isToolsListRequest(r))
		restored, err := io.ReadAll(r.Body)
		s.NoError(err)
		s.Equal(original, string(restored))
	})
	s.Run("body restored when false", func() {
		body := `{"jsonrpc":"2.0","method":"tools/call","id":1}`
		r := s.newRequest(body)
		s.False(isToolsListRequest(r))
		restored, err := io.ReadAll(r.Body)
		s.NoError(err)
		s.Equal(body, string(restored))
	})
}

func TestIsToolsListRequest(t *testing.T) {
	suite.Run(t, new(IsToolsListRequestSuite))
}

// SkipToolListAuthSuite tests the full middleware integration with
// experimental_skip_tool_list_auth enabled.
type SkipToolListAuthSuite struct {
	BaseHttpSuite
}

func (s *SkipToolListAuthSuite) SetupTest() {
	s.BaseHttpSuite.SetupTest()
	s.OidcProvider = nil
	s.StaticConfig.RequireOAuth = true
	s.StaticConfig.ExperimentalSkipToolListAuth = true
}

func (s *SkipToolListAuthSuite) TearDownTest() {
	s.BaseHttpSuite.TearDownTest()
}

func (s *SkipToolListAuthSuite) TestToolsListWithoutAuth() {
	s.StartServer()
	endpoint := fmt.Sprintf("http://127.0.0.1:%s/mcp", s.StaticConfig.Port)

	s.Run("tools/list succeeds without auth header", func() {
		body := bytes.NewBufferString(`{"jsonrpc":"2.0","method":"tools/list","id":1}`)
		req, err := http.NewRequest(http.MethodPost, endpoint, body)
		s.Require().NoError(err)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		s.Require().NoError(err)
		s.T().Cleanup(func() { _ = resp.Body.Close() })
		s.NotEqual(http.StatusUnauthorized, resp.StatusCode, "tools/list should bypass auth")
	})
}

func (s *SkipToolListAuthSuite) TestToolsCallStillRequiresAuth() {
	s.StartServer()
	endpoint := fmt.Sprintf("http://127.0.0.1:%s/mcp", s.StaticConfig.Port)

	s.Run("tools/call without auth header returns 401", func() {
		body := bytes.NewBufferString(`{"jsonrpc":"2.0","method":"tools/call","id":1,"params":{"name":"events_list"}}`)
		req, err := http.NewRequest(http.MethodPost, endpoint, body)
		s.Require().NoError(err)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		s.Require().NoError(err)
		s.T().Cleanup(func() { _ = resp.Body.Close() })
		s.Equal(http.StatusUnauthorized, resp.StatusCode, "tools/call must still require auth")
	})
}

func (s *SkipToolListAuthSuite) TestInitializeStillRequiresAuth() {
	s.StartServer()
	endpoint := fmt.Sprintf("http://127.0.0.1:%s/mcp", s.StaticConfig.Port)

	s.Run("initialize without auth header returns 401", func() {
		body := bytes.NewBufferString(`{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`)
		req, err := http.NewRequest(http.MethodPost, endpoint, body)
		s.Require().NoError(err)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		s.Require().NoError(err)
		s.T().Cleanup(func() { _ = resp.Body.Close() })
		s.Equal(http.StatusUnauthorized, resp.StatusCode, "initialize must still require auth")
	})
}

func (s *SkipToolListAuthSuite) TestFlagDisabledStillRequiresAuth() {
	s.StaticConfig.ExperimentalSkipToolListAuth = false
	s.StartServer()
	endpoint := fmt.Sprintf("http://127.0.0.1:%s/mcp", s.StaticConfig.Port)

	s.Run("tools/list without auth returns 401 when flag is off", func() {
		body := bytes.NewBufferString(`{"jsonrpc":"2.0","method":"tools/list","id":1}`)
		req, err := http.NewRequest(http.MethodPost, endpoint, body)
		s.Require().NoError(err)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		s.Require().NoError(err)
		s.T().Cleanup(func() { _ = resp.Body.Close() })
		s.Equal(http.StatusUnauthorized, resp.StatusCode, "tools/list must require auth when flag is off")
	})
}

func (s *SkipToolListAuthSuite) TestToolsListWithAuthStillWorks() {
	s.StaticConfig.OAuthAudience = ""
	s.StaticConfig.SkipJWTVerification = true
	s.StartServer()
	endpoint := fmt.Sprintf("http://127.0.0.1:%s/mcp", s.StaticConfig.Port)

	options := []test.McpClientOption{
		test.WithEndpoint(endpoint),
		test.WithHTTPHeaders(map[string]string{
			"Authorization": "Bearer " + tokenBasicNotExpired,
		}),
	}
	mcpClient := test.NewMcpClient(s.T(), nil, options...)

	s.Run("authenticated tools/list still succeeds", func() {
		s.Require().NotNil(mcpClient.Session, "Expected session for successful authentication")
		tools, err := mcpClient.Session.ListTools(s.T().Context(), &mcp.ListToolsParams{})
		s.Require().NoError(err, "Expected no error listing tools")
		s.Greater(len(tools.Tools), 0, "Expected at least one tool")
	})
	mcpClient.Close()
}

func TestSkipToolListAuth(t *testing.T) {
	suite.Run(t, new(SkipToolListAuthSuite))
}
