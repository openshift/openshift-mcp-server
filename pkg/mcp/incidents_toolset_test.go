package mcp

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/containers/kubernetes-mcp-server/internal/test"
	"github.com/containers/kubernetes-mcp-server/pkg/config/configtest"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func (s *ToolsetsSuite) TestIncidentToolsetVisibility() {
	cases := []struct {
		name       string
		configured bool
		present    bool
		enabled    bool
		want       bool
	}{
		{name: "toolset not configured", present: true, enabled: true},
		{name: "UIPlugin absent", configured: true},
		{name: "UIPlugin disabled", configured: true, present: true},
		{name: "UIPlugin enabled", configured: true, present: true, enabled: true, want: true},
	}
	for _, tt := range cases {
		s.Run(tt.name, func() {
			s.ResetHandlers()
			s.Handle(test.NewDiscoveryClientHandler(metav1.APIResourceList{
				GroupVersion: "observability.openshift.io/v1alpha1",
				APIResources: []metav1.APIResource{{
					Name: "uiplugins", Kind: "UIPlugin", Namespaced: false,
					Verbs: metav1.Verbs{"get", "list"},
				}},
			}))
			s.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/apis/observability.openshift.io/v1alpha1/uiplugins/monitoring" {
					return
				}
				if !tt.present {
					http.Error(w, "not found", http.StatusNotFound)
					return
				}
				test.WriteObject(w, &unstructured.Unstructured{Object: map[string]any{
					"apiVersion": "observability.openshift.io/v1alpha1",
					"kind":       "UIPlugin",
					"metadata":   map[string]any{"name": "monitoring"},
					"spec": map[string]any{
						"monitoring": map[string]any{
							"clusterHealthAnalyzer": map[string]any{"enabled": tt.enabled},
						},
					},
				}})
			}))
			if tt.configured {
				s.Cfg.Toolsets.SetForTest([]string{"observability/incidents"})
			} else {
				s.Cfg.Toolsets.SetForTest([]string{})
			}
			s.InitMcpClient()
			tools, err := s.ListTools()
			s.Require().NoError(err)
			found := false
			for _, tool := range tools.Tools {
				if tool.Name == "get_incidents" {
					found = true
				}
			}
			s.Equal(tt.want, found)
			if tt.want {
				s.assertJsonSnapshot("toolsets-incidents-tools.json", tools.Tools)
			}
		})
	}
}

func (s *ToolsetsSuite) TestIncidentToolsetRejectsHTTPBackend() {
	requests := make(chan struct{}, 1)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		select {
		case requests <- struct{}{}:
		default:
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer backend.Close()

	s.ResetHandlers()
	s.Handle(test.NewDiscoveryClientHandler(metav1.APIResourceList{
		GroupVersion: "observability.openshift.io/v1alpha1",
		APIResources: []metav1.APIResource{{
			Name: "uiplugins", Kind: "UIPlugin", Namespaced: false,
			Verbs: metav1.Verbs{"get", "list"},
		}},
	}))
	s.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/apis/observability.openshift.io/v1alpha1/uiplugins/monitoring" {
			test.WriteObject(w, &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "observability.openshift.io/v1alpha1",
				"kind":       "UIPlugin",
				"metadata":   map[string]any{"name": "monitoring"},
				"spec": map[string]any{
					"monitoring": map[string]any{
						"clusterHealthAnalyzer": map[string]any{"enabled": true},
					},
				},
			}})
		}
	}))

	configtest.OverlayTOML(s.T(), &s.Cfg, fmt.Sprintf(`
		toolsets = ["observability/incidents"]
		[toolset_configs."observability/metrics"]
		prometheus_url = %q
		alertmanager_url = %q
	`, backend.URL, backend.URL))
	s.InitMcpClient()

	result, err := s.CallTool("get_incidents", map[string]any{})
	s.Require().NoError(err)
	s.Require().True(result.IsError, "HTTP backend should be rejected")
	s.Require().NotEmpty(result.Content)
	s.Contains(result.Content[0].(*mcp.TextContent).Text, "must use HTTPS")
	select {
	case <-requests:
		s.Fail("HTTP backend must not be contacted")
	default:
	}
}
