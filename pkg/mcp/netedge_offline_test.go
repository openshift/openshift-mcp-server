package mcp

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/config/configtest"
	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/netedge"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
)

type NetEdgeOfflineSuite struct {
	BaseMcpSuite
	firstID  string
	secondID string
}

func (s *NetEdgeOfflineSuite) SetupTest() {
	s.BaseMcpSuite.SetupTest()
	root, err := filepath.Abs(filepath.Join("..", "..", "evals", "testdata"))
	s.Require().NoError(err)
	first := filepath.Join(root, "must-gather")
	second := filepath.Join(root, "must-gather-second")
	s.firstID, err = mg.ArchiveIDFromLocalPath(first)
	s.Require().NoError(err)
	s.secondID, err = mg.ArchiveIDFromLocalPath(second)
	s.Require().NoError(err)

	configtest.OverlayTOML(s.T(), &s.Cfg, fmt.Sprintf(`
		[toolset_configs."openshift/mustgather"]
		mustgather_dirs = [%q, %q]
	`, first, second))
	s.Cfg.Toolsets.SetForTest(append(s.Cfg.Toolsets.Get(), (&netedge.Toolset{}).GetName(), "openshift/mustgather"))
}

func (s *NetEdgeOfflineSuite) callOffline(name string, args map[string]any, id string) string {
	s.T().Helper()
	selected := make(map[string]any, len(args)+1)
	for k, v := range args {
		selected[k] = v
	}
	selected["archive_id"] = id
	result, err := s.CallTool(name, selected)
	s.Require().NoError(err)
	s.Require().False(result.IsError, "%s returned an error: %v", name, result.Content)
	s.Require().Len(result.Content, 1)
	return result.Content[0].(*mcp.TextContent).Text
}

func (s *NetEdgeOfflineSuite) TestExplicitArchiveSelection() {
	s.InitMcpClient()
	listing, err := s.CallTool("mustgather_list", map[string]any{})
	s.Require().NoError(err)
	s.Require().False(listing.IsError)
	s.Contains(listing.Content[0].(*mcp.TextContent).Text, s.firstID)
	s.Contains(listing.Content[0].(*mcp.TextContent).Text, s.secondID)

	cases := []struct {
		name, firstValue, secondValue string
		args                          map[string]any
	}{
		{"inspect_route", "console-openshift-console.apps.example.com", "console-second.apps.example.com", map[string]any{"namespace": "openshift-console", "route": "console"}},
		{"get_coredns_config", "prometheus :9153", "forward . 192.0.2.53", map[string]any{}},
		{"get_service_endpoints", "10.128.0.5", "10.129.0.9", map[string]any{"namespace": "openshift-ingress", "service": "router-default"}},
	}
	for _, tc := range cases {
		s.Run(tc.name, func() {
			first := s.callOffline(tc.name, tc.args, s.firstID)
			second := s.callOffline(tc.name, tc.args, s.secondID)
			s.Contains(first, tc.firstValue)
			s.NotContains(first, tc.secondValue)
			s.Contains(second, tc.secondValue)
			s.NotContains(second, tc.firstValue)
			// Re-read the first archive after selecting the second to catch
			// shared active-archive state leaking between calls.
			s.Contains(s.callOffline(tc.name, tc.args, s.firstID), tc.firstValue)
		})
	}
}

func (s *NetEdgeOfflineSuite) TestUnknownArchiveDoesNotUseLiveCluster() {
	s.InitMcpClient()
	result, err := s.CallTool("get_coredns_config", map[string]any{"archive_id": "mg-000000000000"})
	s.Require().NoError(err)
	s.Require().True(result.IsError)
	s.Contains(result.Content[0].(*mcp.TextContent).Text, "not found")
}

func (s *NetEdgeOfflineSuite) TestMalformedArchiveIDDoesNotUseLiveCluster() {
	s.InitMcpClient()
	result, err := s.CallTool("get_coredns_config", map[string]any{"archive_id": 123})
	s.Require().NoError(err)
	s.Require().True(result.IsError)
	s.Contains(result.Content[0].(*mcp.TextContent).Text, "archive_id")
}

func (s *NetEdgeOfflineSuite) TestOfflineRouteRedactsTLSKey() {
	s.InitMcpClient()
	text := s.callOffline("inspect_route", map[string]any{"namespace": "openshift-console", "route": "console"}, s.secondID)
	s.NotContains(text, "second-archive-private-key")
	s.Contains(text, "<redacted>")
}

func TestNetEdgeOffline(t *testing.T) {
	suite.Run(t, new(NetEdgeOfflineSuite))
}
