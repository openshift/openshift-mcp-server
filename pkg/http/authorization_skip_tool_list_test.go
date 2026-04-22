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

type IsGatewayDiscoveryRequestSuite struct {
	suite.Suite
}

func (s *IsGatewayDiscoveryRequestSuite) newRequest(body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(body))
	return r
}

func (s *IsGatewayDiscoveryRequestSuite) TestAcceptsToolsList() {
	s.Run("minimal request", func() {
		r := s.newRequest(`{"jsonrpc":"2.0","method":"tools/list","id":1}`)
		s.True(isGatewayDiscoveryRequest(r))
	})
	s.Run("with string id", func() {
		r := s.newRequest(`{"jsonrpc":"2.0","method":"tools/list","id":"abc"}`)
		s.True(isGatewayDiscoveryRequest(r))
	})
	s.Run("with params cursor", func() {
		r := s.newRequest(`{"jsonrpc":"2.0","method":"tools/list","id":1,"params":{"cursor":"next"}}`)
		s.True(isGatewayDiscoveryRequest(r))
	})
}

func (s *IsGatewayDiscoveryRequestSuite) TestAcceptsInitialize() {
	s.Run("minimal", func() {
		r := s.newRequest(`{"jsonrpc":"2.0","method":"initialize","id":1}`)
		s.True(isGatewayDiscoveryRequest(r))
	})
	s.Run("with params", func() {
		r := s.newRequest(`{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`)
		s.True(isGatewayDiscoveryRequest(r))
	})
}

func (s *IsGatewayDiscoveryRequestSuite) TestAcceptsPing() {
	r := s.newRequest(`{"jsonrpc":"2.0","method":"ping","id":1}`)
	s.True(isGatewayDiscoveryRequest(r))
}

func (s *IsGatewayDiscoveryRequestSuite) TestAcceptsNotificationsInitialized() {
	s.Run("without id", func() {
		r := s.newRequest(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
		s.True(isGatewayDiscoveryRequest(r))
	})
}

func (s *IsGatewayDiscoveryRequestSuite) TestRejectsUnsafeMethods() {
	s.Run("tools/call", func() {
		r := s.newRequest(`{"jsonrpc":"2.0","method":"tools/call","id":1,"params":{"name":"foo"}}`)
		s.False(isGatewayDiscoveryRequest(r))
	})
	s.Run("resources/read", func() {
		r := s.newRequest(`{"jsonrpc":"2.0","method":"resources/read","id":1}`)
		s.False(isGatewayDiscoveryRequest(r))
	})
	s.Run("prompts/get", func() {
		r := s.newRequest(`{"jsonrpc":"2.0","method":"prompts/get","id":1}`)
		s.False(isGatewayDiscoveryRequest(r))
	})
}

func (s *IsGatewayDiscoveryRequestSuite) TestRejectsBatchedRequests() {
	r := s.newRequest(`[{"jsonrpc":"2.0","method":"tools/list","id":1},{"jsonrpc":"2.0","method":"tools/call","id":2}]`)
	s.False(isGatewayDiscoveryRequest(r))
}

func (s *IsGatewayDiscoveryRequestSuite) TestRejectsInvalidJSON() {
	s.Run("malformed json", func() {
		r := s.newRequest(`{not valid json}`)
		s.False(isGatewayDiscoveryRequest(r))
	})
	s.Run("empty body", func() {
		r := s.newRequest(``)
		s.False(isGatewayDiscoveryRequest(r))
	})
	s.Run("whitespace only", func() {
		r := s.newRequest(`   `)
		s.False(isGatewayDiscoveryRequest(r))
	})
}

func (s *IsGatewayDiscoveryRequestSuite) TestRejectsWrongJSONRPCVersion() {
	r := s.newRequest(`{"jsonrpc":"1.0","method":"tools/list","id":1}`)
	s.False(isGatewayDiscoveryRequest(r))
}

func (s *IsGatewayDiscoveryRequestSuite) TestNilBody() {
	r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	r.Body = nil
	s.False(isGatewayDiscoveryRequest(r))
}

func (s *IsGatewayDiscoveryRequestSuite) TestNoBody() {
	r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	r.Body = http.NoBody
	s.False(isGatewayDiscoveryRequest(r))
}

func (s *IsGatewayDiscoveryRequestSuite) TestBodyRestoredAfterCheck() {
	original := `{"jsonrpc":"2.0","method":"tools/list","id":1}`
	s.Run("body restored when true", func() {
		r := s.newRequest(original)
		s.True(isGatewayDiscoveryRequest(r))
		restored, err := io.ReadAll(r.Body)
		s.NoError(err)
		s.Equal(original, string(restored))
	})
	s.Run("body restored when false", func() {
		body := `{"jsonrpc":"2.0","method":"tools/call","id":1}`
		r := s.newRequest(body)
		s.False(isGatewayDiscoveryRequest(r))
		restored, err := io.ReadAll(r.Body)
		s.NoError(err)
		s.Equal(body, string(restored))
	})
}

func TestIsGatewayDiscoveryRequest(t *testing.T) {
	suite.Run(t, new(IsGatewayDiscoveryRequestSuite))
}

// SkipGatewayAuthSuite tests the full middleware integration with
// experimental_skip_gateway_auth enabled.
type SkipGatewayAuthSuite struct {
	BaseHttpSuite
}

func (s *SkipGatewayAuthSuite) SetupTest() {
	s.BaseHttpSuite.SetupTest()
	s.OidcProvider = nil
	s.StaticConfig.RequireOAuth = true
	s.StaticConfig.ExperimentalSkipGatewayAuth = true
}

func (s *SkipGatewayAuthSuite) TearDownTest() {
	s.BaseHttpSuite.TearDownTest()
}

func (s *SkipGatewayAuthSuite) TestToolsListWithoutAuth() {
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

func (s *SkipGatewayAuthSuite) TestInitializeWithoutAuth() {
	s.StartServer()
	endpoint := fmt.Sprintf("http://127.0.0.1:%s/mcp", s.StaticConfig.Port)

	s.Run("initialize succeeds without auth header", func() {
		body := bytes.NewBufferString(`{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`)
		req, err := http.NewRequest(http.MethodPost, endpoint, body)
		s.Require().NoError(err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		resp, err := http.DefaultClient.Do(req)
		s.Require().NoError(err)
		s.T().Cleanup(func() { _ = resp.Body.Close() })
		s.NotEqual(http.StatusUnauthorized, resp.StatusCode, "initialize should bypass auth")
	})
}

func (s *SkipGatewayAuthSuite) TestPingWithoutAuth() {
	s.StartServer()
	endpoint := fmt.Sprintf("http://127.0.0.1:%s/mcp", s.StaticConfig.Port)

	s.Run("ping succeeds without auth header", func() {
		body := bytes.NewBufferString(`{"jsonrpc":"2.0","method":"ping","id":1}`)
		req, err := http.NewRequest(http.MethodPost, endpoint, body)
		s.Require().NoError(err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		resp, err := http.DefaultClient.Do(req)
		s.Require().NoError(err)
		s.T().Cleanup(func() { _ = resp.Body.Close() })
		s.NotEqual(http.StatusUnauthorized, resp.StatusCode, "ping should bypass auth")
	})
}

func (s *SkipGatewayAuthSuite) TestGetWithoutAuth() {
	s.StartServer()
	endpoint := fmt.Sprintf("http://127.0.0.1:%s/mcp", s.StaticConfig.Port)

	s.Run("GET request does not return 401", func() {
		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		s.Require().NoError(err)
		req.Header.Set("Accept", "text/event-stream")
		resp, err := http.DefaultClient.Do(req)
		s.Require().NoError(err)
		s.T().Cleanup(func() { _ = resp.Body.Close() })
		s.NotEqual(http.StatusUnauthorized, resp.StatusCode, "GET should bypass auth")
	})
}

func (s *SkipGatewayAuthSuite) TestProtectedMethodsStillRequireAuth() {
	s.StartServer()
	endpoint := fmt.Sprintf("http://127.0.0.1:%s/mcp", s.StaticConfig.Port)

	protectedMethods := []struct {
		name string
		body string
	}{
		{"tools/call", `{"jsonrpc":"2.0","method":"tools/call","id":1,"params":{"name":"events_list"}}`},
		{"prompts/list", `{"jsonrpc":"2.0","method":"prompts/list","id":1}`},
		{"prompts/get", `{"jsonrpc":"2.0","method":"prompts/get","id":1,"params":{"name":"foo"}}`},
		{"resources/list", `{"jsonrpc":"2.0","method":"resources/list","id":1}`},
		{"resources/read", `{"jsonrpc":"2.0","method":"resources/read","id":1,"params":{"uri":"file:///foo"}}`},
		{"resources/templates/list", `{"jsonrpc":"2.0","method":"resources/templates/list","id":1}`},
		{"completion/complete", `{"jsonrpc":"2.0","method":"completion/complete","id":1,"params":{"ref":{"type":"ref/prompt","name":"foo"},"argument":{"name":"arg","value":"val"}}}`},
		{"logging/setLevel", `{"jsonrpc":"2.0","method":"logging/setLevel","id":1,"params":{"level":"info"}}`},
	}

	for _, tc := range protectedMethods {
		s.Run(tc.name+" without auth header returns 401", func() {
			body := bytes.NewBufferString(tc.body)
			req, err := http.NewRequest(http.MethodPost, endpoint, body)
			s.Require().NoError(err)
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			s.Require().NoError(err)
			s.T().Cleanup(func() { _ = resp.Body.Close() })
			s.Equal(http.StatusUnauthorized, resp.StatusCode, "%s must still require auth", tc.name)
		})
	}
}

func (s *SkipGatewayAuthSuite) TestFlagDisabledStillRequiresAuth() {
	s.StaticConfig.ExperimentalSkipGatewayAuth = false
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

	s.Run("initialize without auth returns 401 when flag is off", func() {
		body := bytes.NewBufferString(`{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`)
		req, err := http.NewRequest(http.MethodPost, endpoint, body)
		s.Require().NoError(err)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		s.Require().NoError(err)
		s.T().Cleanup(func() { _ = resp.Body.Close() })
		s.Equal(http.StatusUnauthorized, resp.StatusCode, "initialize must require auth when flag is off")
	})

	s.Run("GET without auth returns 401 when flag is off", func() {
		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		s.Require().NoError(err)
		req.Header.Set("Accept", "text/event-stream")
		resp, err := http.DefaultClient.Do(req)
		s.Require().NoError(err)
		s.T().Cleanup(func() { _ = resp.Body.Close() })
		s.Equal(http.StatusUnauthorized, resp.StatusCode, "GET must require auth when flag is off")
	})
}

func (s *SkipGatewayAuthSuite) TestToolsListWithAuthStillWorks() {
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

func TestSkipGatewayAuth(t *testing.T) {
	suite.Run(t, new(SkipGatewayAuthSuite))
}
