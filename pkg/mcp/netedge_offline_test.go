package mcp

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"github.com/containers/kubernetes-mcp-server/internal/test"
	"github.com/containers/kubernetes-mcp-server/pkg/config/configtest"
	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/netedge"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

func (s *NetEdgeOfflineSuite) assertOfflineToolSchemas(enabled bool) {
	s.T().Helper()
	listing, err := s.ListTools()
	s.Require().NoError(err)
	for _, name := range []string{"get_coredns_config", "get_service_endpoints", "inspect_route"} {
		s.Run(name, func() {
			index := slices.IndexFunc(listing.Tools, func(tool *mcp.Tool) bool { return tool.Name == name })
			s.Require().NotEqual(-1, index, "tool must remain available for live-cluster calls")
			tool := listing.Tools[index]
			schema, ok := tool.InputSchema.(map[string]any)
			s.Require().True(ok)
			properties, ok := schema["properties"].(map[string]any)
			s.Require().True(ok)
			if enabled {
				s.Contains(properties, "archive_id")
				s.Contains(tool.Description, "must-gather")
			} else {
				s.NotContains(properties, "archive_id")
				s.NotContains(tool.Description, "must-gather")
			}
		})
	}
}

func (s *NetEdgeOfflineSuite) TestOfflineSupportFollowsEnabledToolsets() {
	s.InitMcpClient()
	s.Run("enabled", func() { s.assertOfflineToolSchemas(true) })

	// Keep the archive directories configured while disabling the toolset.
	disabledCfg := *s.Cfg
	disabledCfg.Toolsets.SetForTest(slices.DeleteFunc(slices.Clone(s.Cfg.Toolsets.Get()), func(name string) bool {
		return name == "openshift/mustgather"
	}))
	s.Require().NoError(s.mcpServer.ReloadConfiguration(s.T().Context(), &disabledCfg))
	s.Run("disabled", func() {
		s.assertOfflineToolSchemas(false)
		cases := []struct {
			name string
			args map[string]any
		}{
			{"get_coredns_config", map[string]any{"archive_id": s.firstID}},
			{"get_service_endpoints", map[string]any{"namespace": "openshift-ingress", "service": "router-default", "archive_id": s.firstID}},
			{"inspect_route", map[string]any{"namespace": "openshift-console", "route": "console", "archive_id": s.firstID}},
		}
		for _, tc := range cases {
			s.Run(tc.name+" rejects archive requests", func() {
				result, err := s.CallTool(tc.name, tc.args)
				s.Require().NoError(err)
				s.Require().True(result.IsError)
				s.Contains(result.Content[0].(*mcp.TextContent).Text, "openshift/mustgather toolset to be enabled")
			})
		}
	})

	s.Require().NoError(s.mcpServer.ReloadConfiguration(s.T().Context(), s.Cfg))
	s.Run("re-enabled", func() {
		s.assertOfflineToolSchemas(true)
		s.Contains(s.callOffline("get_coredns_config", map[string]any{}, s.firstID), "prometheus :9153")
	})
}

func (s *NetEdgeOfflineSuite) TestLiveRequestAfterOfflineRequest() {
	client := kubernetes.NewForConfigOrDie(test.EnvTestRestConfig())
	ctx := s.T().Context()
	_, err := client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "openshift-dns"},
	}, metav1.CreateOptions{})
	if !apierrors.IsAlreadyExists(err) {
		s.Require().NoError(err)
	}
	const liveCorefile = ".:5353 { forward . 192.0.2.100 }"
	_, err = client.CoreV1().ConfigMaps("openshift-dns").Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "dns-default"},
		Data:       map[string]string{"Corefile": liveCorefile},
	}, metav1.CreateOptions{})
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		s.NoError(client.CoreV1().ConfigMaps("openshift-dns").Delete(context.Background(), "dns-default", metav1.DeleteOptions{}))
	})
	s.InitMcpClient()
	s.Contains(s.callOffline("get_coredns_config", map[string]any{}, s.firstID), "prometheus :9153")

	result, err := s.CallTool("get_coredns_config", map[string]any{})
	s.Require().NoError(err)
	s.Require().False(result.IsError)
	s.Equal(liveCorefile, result.Content[0].(*mcp.TextContent).Text)
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
