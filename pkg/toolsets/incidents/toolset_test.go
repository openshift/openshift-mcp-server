package incidents_test

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/containers/kubernetes-mcp-server/internal/test"
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/config"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	servermcp "github.com/containers/kubernetes-mcp-server/pkg/mcp"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/incidents"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type ToolsetSuite struct {
	suite.Suite
}

type pluginResponse struct {
	status   int
	analyzer any
}

func (s *ToolsetSuite) newInspector(responses ...pluginResponse) api.ClusterInspector {
	kubeconfig := clientcmdapi.NewConfig()
	for i, response := range responses {
		server := test.NewMockServer()
		s.T().Cleanup(server.Close)
		server.Handle(test.NewDiscoveryClientHandler(metav1.APIResourceList{
			GroupVersion: "observability.openshift.io/v1alpha1",
			APIResources: []metav1.APIResource{
				{Name: "uiplugins", Kind: "UIPlugin", Verbs: metav1.Verbs{"get"}},
			},
		}))
		server.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/apis/observability.openshift.io/v1alpha1/uiplugins/monitoring" {
				return
			}
			if response.status != http.StatusOK {
				http.Error(w, http.StatusText(response.status), response.status)
				return
			}
			plugin := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "observability.openshift.io/v1alpha1",
				"kind":       "UIPlugin",
				"metadata":   map[string]any{"name": "monitoring"},
			}}
			if response.analyzer != nil {
				plugin.Object["spec"] = map[string]any{
					"monitoring": map[string]any{
						"clusterHealthAnalyzer": response.analyzer,
					},
				}
			}
			test.WriteObject(w, plugin)
		}))

		name := strconv.Itoa(i)
		kubeconfig.Clusters[name] = &clientcmdapi.Cluster{Server: server.Config().Host}
		kubeconfig.AuthInfos[name] = clientcmdapi.NewAuthInfo()
		kubeconfig.Contexts[name] = &clientcmdapi.Context{Cluster: name, AuthInfo: name}
		if i == 0 {
			kubeconfig.CurrentContext = name
		}
	}

	cfg := config.BaseDefault()
	cfg.KubeConfig.SetForTest(test.KubeconfigFile(s.T(), kubeconfig))
	cfg.ValidationEnabled.SetForTest(false)
	provider, err := kubernetes.NewProvider(s.T().Context(), cfg)
	s.Require().NoError(err)
	s.T().Cleanup(provider.Close)
	return kubernetes.NewClusterInspector(provider)
}

func (s *ToolsetSuite) TestGetToolsWithoutInspector() {
	tools := (&incidents.Toolset{}).GetTools(s.T().Context(), api.ToolsetContext{})
	s.Nil(tools)
}

func (s *ToolsetSuite) TestUIPluginGate() {
	enabled := pluginResponse{http.StatusOK, map[string]any{"enabled": true}}
	disabled := pluginResponse{http.StatusOK, map[string]any{"enabled": false}}
	missing := pluginResponse{status: http.StatusNotFound}
	forbidden := pluginResponse{status: http.StatusForbidden}
	failed := pluginResponse{status: http.StatusInternalServerError}

	cases := []struct {
		name      string
		responses []pluginResponse
		wantTools int
	}{
		{"enabled", []pluginResponse{enabled}, 1},
		{"disabled", []pluginResponse{disabled}, 0},
		{"missing spec", []pluginResponse{{status: http.StatusOK}}, 0},
		{"missing enabled field", []pluginResponse{
			{http.StatusOK, map[string]any{}},
		}, 0},
		{"non-boolean enabled field", []pluginResponse{
			{http.StatusOK, map[string]any{"enabled": "true"}},
		}, 0},
		{"malformed analyzer field", []pluginResponse{
			{http.StatusOK, "invalid"},
		}, 0},
		{"missing UIPlugin", []pluginResponse{missing}, 0},
		{"permission denied", []pluginResponse{forbidden}, 0},
		{"API failure", []pluginResponse{failed}, 0},
		{"one target enabled", []pluginResponse{disabled, enabled}, 1},
		{"enabled target with failing target", []pluginResponse{failed, enabled}, 1},
		{"no enabled targets", []pluginResponse{disabled, missing}, 0},
		{"disabled target with failing target", []pluginResponse{disabled, failed}, 0},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			tools := (&incidents.Toolset{}).GetTools(s.T().Context(), api.ToolsetContext{
				Inspector: s.newInspector(tc.responses...),
			})
			s.Len(tools, tc.wantTools)
		})
	}
}

func (s *ToolsetSuite) TestIncidentToolMetadata() {
	tools := (&incidents.Toolset{}).GetTools(s.T().Context(), api.ToolsetContext{
		Inspector: s.newInspector(pluginResponse{
			http.StatusOK, map[string]any{"enabled": true},
		}),
	})
	s.Require().Len(tools, 1)
	tool := tools[0]

	s.Equal("get_incidents", tool.Tool.Name)
	s.NotEmpty(tool.Tool.Description)
	s.NotNil(tool.Handler)
	s.Require().NotNil(tool.Tool.Annotations.ReadOnlyHint)
	s.True(*tool.Tool.Annotations.ReadOnlyHint)
	s.Require().NotNil(tool.Tool.Annotations.OpenWorldHint)
	s.True(*tool.Tool.Annotations.OpenWorldHint)
	s.Require().NotNil(tool.Tool.InputSchema)
	s.Equal("object", tool.Tool.InputSchema.Type)
	s.Contains(tool.Tool.InputSchema.Properties, "time_range")
	s.Contains(tool.Tool.InputSchema.Properties, "min_severity")
}

func (s *ToolsetSuite) callIncidents(tomlConfig, arguments string, caData []byte) *api.ToolCallResult {
	ctx, cancel := context.WithTimeout(s.T().Context(), 5*time.Second)
	defer cancel()
	ctx = context.WithValue(ctx, kubernetes.OAuthAuthorizationHeader, "Bearer header-token")

	cfg, err := config.ReadToml(ctx, []byte(tomlConfig), config.WithBaseDefault())
	s.Require().NoError(err)
	request, err := servermcp.GoSdkToolCallParamsToToolCallRequest(&mcp.CallToolParamsRaw{
		Name:      "get_incidents",
		Arguments: json.RawMessage(arguments),
	})
	s.Require().NoError(err)
	client, err := kubernetes.NewKubernetes(ctx, cfg,
		clientcmd.NewDefaultClientConfig(*test.KubeConfigFake(), nil),
		&rest.Config{
			Host:            "https://kubernetes.example",
			BearerToken:     "kubeconfig-token",
			TLSClientConfig: rest.TLSClientConfig{CAData: caData},
		},
	)
	s.Require().NoError(err)
	tools := (&incidents.Toolset{}).GetTools(ctx, api.ToolsetContext{
		Inspector: s.newInspector(pluginResponse{
			http.StatusOK, map[string]any{"enabled": true},
		}),
	})
	s.Require().Len(tools, 1)
	result, err := tools[0].Handler(api.ToolHandlerParams{
		Context:          ctx,
		Config:           cfg,
		KubernetesClient: client,
		Request:          request,
	})
	s.Require().NoError(err)
	s.Require().NotNil(result)
	return result
}

func (s *ToolsetSuite) TestIncidentConfiguration() {
	cases := []struct {
		name      string
		promURL   string
		alertURL  string
		wantError string
	}{
		{"missing Prometheus URL", "", "https://alerts.example", "requires prometheus_url and alertmanager_url"},
		{"missing Alertmanager URL", "https://prometheus.example", "", "requires prometheus_url and alertmanager_url"},
		{"HTTP Prometheus URL", "http://prometheus.example", "https://alerts.example", "must use HTTPS"},
		{"HTTP Alertmanager URL", "https://prometheus.example", "http://alerts.example", "must use HTTPS"},
		{"relative URL", "/prometheus", "https://alerts.example", "must use HTTPS"},
		{"malformed URL", "https://%", "https://alerts.example", "invalid URL escape"},
	}
	s.Run("missing metrics configuration", func() {
		result := s.callIncidents("", `{}`, nil)
		s.ErrorContains(result.Error, "requires prometheus_url and alertmanager_url")
	})
	for _, tc := range cases {
		s.Run(tc.name, func() {
			tomlConfig := fmt.Sprintf(`
[toolset_configs."observability/metrics"]
prometheus_url = %q
alertmanager_url = %q
`, tc.promURL, tc.alertURL)
			result := s.callIncidents(tomlConfig, `{}`, nil)
			s.ErrorContains(result.Error, tc.wantError)
		})
	}
}

func (s *ToolsetSuite) TestIncidentArguments() {
	const tomlConfig = `
[toolset_configs."observability/metrics"]
prometheus_url = "https://prometheus.example"
alertmanager_url = "https://alerts.example"
`
	cases := []struct {
		name      string
		arguments string
		wantError string
	}{
		{"string time range", `{"time_range":"1"}`, "time_range must be a number"},
		{"null time range", `{"time_range":null}`, "time_range must be a number"},
		{"zero time range", `{"time_range":0}`, "time_range must be between 1 and 360"},
		{"negative time range", `{"time_range":-1}`, "time_range must be between 1 and 360"},
		{"below minimum", `{"time_range":0.5}`, "time_range must be between 1 and 360"},
		{"above maximum", `{"time_range":361}`, "time_range must be between 1 and 360"},
		{"numeric severity", `{"min_severity":1}`, "min_severity must be a string"},
		{"null severity", `{"min_severity":null}`, "min_severity must be a string"},
		{"empty severity", `{"min_severity":""}`, "min_severity must be info, warning, or critical"},
		{"unknown severity", `{"min_severity":"fatal"}`, "min_severity must be info, warning, or critical"},
	}
	for _, tc := range cases {
		s.Run(tc.name, func() {
			result := s.callIncidents(tomlConfig, tc.arguments, nil)
			s.ErrorContains(result.Error, tc.wantError)
		})
	}
}

func (s *ToolsetSuite) TestIncidentBackends() {
	cases := []struct {
		name      string
		settings  string
		arguments string
		trustCA   bool
		wantToken string
		wantHours float64
		failPath  string
		wantError string
	}{
		{
			name:      "default header authentication and time range",
			arguments: `{}`, trustCA: true,
			wantToken: "header-token", wantHours: 360,
		},
		{
			name:      "kubeconfig authentication and minimum time range",
			settings:  `auth_mode = "kubeconfig"`,
			arguments: `{"time_range":1,"min_severity":"info"}`, trustCA: true,
			wantToken: "kubeconfig-token", wantHours: 1,
		},
		{
			name:      "fractional time range",
			arguments: `{"time_range":1.5,"min_severity":"critical"}`, trustCA: true,
			wantToken: "header-token", wantHours: 1.5,
		},
		{
			name:      "maximum time range and case-insensitive severity",
			arguments: `{"time_range":360,"min_severity":"WARNING"}`, trustCA: true,
			wantToken: "header-token", wantHours: 360,
		},
		{
			name:     "insecure permits an untrusted certificate",
			settings: `insecure = true`, arguments: `{}`,
			wantToken: "header-token", wantHours: 360,
		},
		{
			name:      "certificate verification enabled by default",
			arguments: `{}`, wantError: "certificate signed by unknown authority",
		},
		{
			name:      "Prometheus failure",
			arguments: `{}`, trustCA: true,
			failPath: "/api/v1/query_range", wantError: "503",
		},
		{
			name:      "Alertmanager failure",
			arguments: `{}`, trustCA: true,
			failPath: "/api/v2/alerts", wantError: "503",
		},
	}
	for _, tc := range cases {
		s.Run(tc.name, func() {
			requests := make(chan *http.Request, 16)
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := r.ParseForm(); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				requests <- r
				if r.URL.Path == tc.failPath {
					http.Error(w, "backend unavailable", http.StatusServiceUnavailable)
					return
				}
				var body string
				switch r.URL.Path {
				case "/api/v1/query_range":
					body = `{"status":"success","data":{"resultType":"matrix","result":[]}}`
				case "/api/v1/query":
					body = `{"status":"success","data":{"resultType":"vector","result":[]}}`
				case "/api/v2/alerts":
					body = `[]`
				default:
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(json.RawMessage(body)); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			}))
			s.T().Cleanup(server.Close)

			var caData []byte
			if tc.trustCA {
				caData = pem.EncodeToMemory(&pem.Block{
					Type: "CERTIFICATE", Bytes: server.Certificate().Raw,
				})
			}
			tomlConfig := fmt.Sprintf(`
[toolset_configs."observability/metrics"]
prometheus_url = %q
alertmanager_url = %q
%s
`, server.URL, server.URL, tc.settings)
			result := s.callIncidents(tomlConfig, tc.arguments, caData)
			if tc.wantError != "" {
				s.ErrorContains(result.Error, tc.wantError)
				return
			}
			s.Require().NoError(result.Error)
			s.Contains(result.Content, "<DATA>")
			s.Contains(result.Content, "<INSTRUCTIONS>")
			structured, err := json.Marshal(result.StructuredContent)
			s.Require().NoError(err)
			s.JSONEq(`{"incidents":{"total":0,"items":[]}}`, string(structured))
			s.Contains(result.Content, string(structured))

			s.Require().NotEmpty(requests)
			close(requests)
			for request := range requests {
				s.Equal("Bearer "+tc.wantToken, request.Header.Get("Authorization"))
				if request.URL.Path == "/api/v1/query_range" {
					start, err := strconv.ParseFloat(request.Form.Get("start"), 64)
					s.Require().NoError(err)
					end, err := strconv.ParseFloat(request.Form.Get("end"), 64)
					s.Require().NoError(err)
					s.InDelta(tc.wantHours*3600, end-start, 0.001)
				}
			}
		})
	}
}

func TestToolset(t *testing.T) {
	suite.Run(t, new(ToolsetSuite))
}
