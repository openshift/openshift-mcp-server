package mcp

import (
	"bytes"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/containers/kubernetes-mcp-server/internal/test"
)

const (
	ovnTestNamespace = "openshift-ovn-kubernetes"
	ovnTestPodName   = "ovnkube-node-abc12"
)

type OVNKubernetesSuite struct {
	BaseMcpSuite
	mockServer    *test.MockServer
	lastContainer atomic.Value
}

func (s *OVNKubernetesSuite) SetupTest() {
	s.BaseMcpSuite.SetupTest()
	s.mockServer = test.NewMockServer()
	s.mockServer.Handle(test.NewDiscoveryClientHandler())
	s.Cfg.KubeConfig = s.mockServer.KubeconfigFile(s.T())
	s.Require().NoError(toml.Unmarshal([]byte(`
		toolsets = [ "ovn-kubernetes" ]
	`), s.Cfg), "Expected to parse toolsets config")
	s.setupPodHandler()
}

func (s *OVNKubernetesSuite) TearDownTest() {
	s.BaseMcpSuite.TearDownTest()
	if s.mockServer != nil {
		s.mockServer.Close()
	}
}

func ovnkubeNodePod() *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ovnTestNamespace,
			Name:      ovnTestPodName,
			Annotations: map[string]string{
				"kubectl.kubernetes.io/default-container": "ovnkube-controller",
			},
		},
		Spec: v1.PodSpec{Containers: []v1.Container{
			{Name: "ovn-controller"},
			{Name: "northd"},
			{Name: "nbdb"},
			{Name: "sbdb"},
			{Name: "ovnkube-controller"},
		}},
	}
}

func (s *OVNKubernetesSuite) setupPodHandler() {
	s.mockServer.Handle(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/v1/namespaces/"+ovnTestNamespace+"/pods/"+ovnTestPodName {
			return
		}
		test.WriteObject(w, ovnkubeNodePod())
	}))
}

func (s *OVNKubernetesSuite) setupRecordingExecHandler() {
	s.mockServer.Handle(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/v1/namespaces/"+ovnTestNamespace+"/pods/"+ovnTestPodName+"/exec" {
			return
		}
		s.lastContainer.Store(req.URL.Query().Get("container"))
		var stdin, stdout bytes.Buffer
		ctx, err := test.CreateHTTPStreams(w, req, &test.StreamOptions{
			Stdin:  &stdin,
			Stdout: &stdout,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		defer func() { _ = ctx.Close() }()
		_, _ = io.WriteString(ctx.StdoutStream, "mock-output\n")
	}))
}

func (s *OVNKubernetesSuite) setupStderrExecHandler() {
	s.mockServer.Handle(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/v1/namespaces/"+ovnTestNamespace+"/pods/"+ovnTestPodName+"/exec" {
			return
		}
		var stdin, stdout bytes.Buffer
		ctx, err := test.CreateHTTPStreams(w, req, &test.StreamOptions{
			Stdin:  &stdin,
			Stdout: &stdout,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		defer func() { _ = ctx.Close() }()
		_, _ = io.WriteString(ctx.StderrStream, "simulated error\n")
	}))
}

func (s *OVNKubernetesSuite) TestContainerRouting() {
	s.setupRecordingExecHandler()
	s.InitMcpClient()

	s.Run("ovn_show with nbdb routes to nbdb container", func() {
		result, err := s.CallTool("ovn_show", map[string]any{
			"namespace": ovnTestNamespace,
			"name":      ovnTestPodName,
			"database":  "nbdb",
		})
		s.Require().NoError(err)
		s.Require().NotNil(result)
		s.Falsef(result.IsError, "call tool failed: %v", result.Content)
		s.Equal("nbdb", s.lastContainer.Load().(string))
	})

	s.Run("ovn_show with sbdb routes to sbdb container", func() {
		result, err := s.CallTool("ovn_show", map[string]any{
			"namespace": ovnTestNamespace,
			"name":      ovnTestPodName,
			"database":  "sbdb",
		})
		s.Require().NoError(err)
		s.Require().NotNil(result)
		s.Falsef(result.IsError, "call tool failed: %v", result.Content)
		s.Equal("sbdb", s.lastContainer.Load().(string))
	})

	s.Run("ovn_get with nbdb routes to nbdb container", func() {
		result, err := s.CallTool("ovn_get", map[string]any{
			"namespace": ovnTestNamespace,
			"name":      ovnTestPodName,
			"database":  "nbdb",
			"table":     "Logical_Switch",
		})
		s.Require().NoError(err)
		s.Require().NotNil(result)
		s.Falsef(result.IsError, "call tool failed: %v", result.Content)
		s.Equal("nbdb", s.lastContainer.Load().(string))
	})

	s.Run("ovn_get with sbdb routes to sbdb container", func() {
		result, err := s.CallTool("ovn_get", map[string]any{
			"namespace": ovnTestNamespace,
			"name":      ovnTestPodName,
			"database":  "sbdb",
			"table":     "Chassis",
		})
		s.Require().NoError(err)
		s.Require().NotNil(result)
		s.Falsef(result.IsError, "call tool failed: %v", result.Content)
		s.Equal("sbdb", s.lastContainer.Load().(string))
	})

	s.Run("ovn_lflow_list routes to sbdb container", func() {
		result, err := s.CallTool("ovn_lflow_list", map[string]any{
			"namespace": ovnTestNamespace,
			"name":      ovnTestPodName,
		})
		s.Require().NoError(err)
		s.Require().NotNil(result)
		s.Falsef(result.IsError, "call tool failed: %v", result.Content)
		s.Equal("sbdb", s.lastContainer.Load().(string))
	})

	s.Run("ovn_trace routes to northd container", func() {
		result, err := s.CallTool("ovn_trace", map[string]any{
			"namespace": ovnTestNamespace,
			"name":      ovnTestPodName,
			"datapath":  "test-datapath",
			"microflow": `inport=="port1" && eth.src==00:00:00:00:00:01 && ip4.src==10.244.0.5 && ip4.dst==10.244.1.5`,
		})
		s.Require().NoError(err)
		s.Require().NotNil(result)
		s.Falsef(result.IsError, "call tool failed: %v", result.Content)
		s.Equal("northd", s.lastContainer.Load().(string))
	})
}

func (s *OVNKubernetesSuite) TestDefaultNamespace() {
	s.setupRecordingExecHandler()
	s.InitMcpClient()

	s.Run("ovn_show uses default namespace when omitted", func() {
		result, err := s.CallTool("ovn_show", map[string]any{
			"name":     ovnTestPodName,
			"database": "nbdb",
		})
		s.Require().NoError(err)
		s.Require().NotNil(result)
		s.Falsef(result.IsError, "call tool failed: %v", result.Content)
		s.Equal("nbdb", s.lastContainer.Load().(string))
	})
}

func (s *OVNKubernetesSuite) TestExecStderrReturnsError() {
	s.setupStderrExecHandler()
	s.InitMcpClient()

	for _, tc := range []struct {
		tool   string
		params map[string]any
	}{
		{"ovn_show", map[string]any{"namespace": ovnTestNamespace, "name": ovnTestPodName, "database": "nbdb"}},
		{"ovn_get", map[string]any{"namespace": ovnTestNamespace, "name": ovnTestPodName, "database": "nbdb", "table": "Logical_Switch"}},
		{"ovn_lflow_list", map[string]any{"namespace": ovnTestNamespace, "name": ovnTestPodName}},
		{"ovn_trace", map[string]any{"namespace": ovnTestNamespace, "name": ovnTestPodName, "datapath": "test-dp", "microflow": `inport=="port1" && ip4.src==10.0.0.1`}},
	} {
		s.Run(tc.tool+" returns error on stderr", func() {
			result, err := s.CallTool(tc.tool, tc.params)
			s.Require().NoError(err)
			s.True(result.IsError)
		})
	}
}

func (s *OVNKubernetesSuite) TestMissingRequiredParams() {
	s.setupRecordingExecHandler()
	s.InitMcpClient()

	for _, tc := range []struct {
		tool      string
		required  []string
		allParams map[string]any
	}{
		{
			tool:     "ovn_show",
			required: []string{"name", "database"},
			allParams: map[string]any{
				"namespace": ovnTestNamespace, "name": ovnTestPodName, "database": "nbdb",
			},
		},
		{
			tool:     "ovn_get",
			required: []string{"name", "database", "table"},
			allParams: map[string]any{
				"namespace": ovnTestNamespace, "name": ovnTestPodName, "database": "nbdb", "table": "Logical_Switch",
			},
		},
		{
			tool:     "ovn_lflow_list",
			required: []string{"name"},
			allParams: map[string]any{
				"namespace": ovnTestNamespace, "name": ovnTestPodName,
			},
		},
		{
			tool:     "ovn_trace",
			required: []string{"name", "datapath", "microflow"},
			allParams: map[string]any{
				"namespace": ovnTestNamespace, "name": ovnTestPodName, "datapath": "test-dp",
				"microflow": `inport=="port1" && ip4.src==10.0.0.1`,
			},
		},
	} {
		s.Run(tc.tool+" missing required params", func() {
			for _, param := range tc.required {
				s.Run("missing "+param, func() {
					params := make(map[string]any, len(tc.allParams))
					for k, v := range tc.allParams {
						params[k] = v
					}
					delete(params, param)
					result, err := s.CallTool(tc.tool, params)
					s.Require().NoError(err)
					s.Require().NotNil(result)
					s.Truef(result.IsError, "expected error for missing %s", param)
					s.Contains(result.Content[0].(*mcp.TextContent).Text, param+" parameter required")
				})
			}
		})
	}
}

func (s *OVNKubernetesSuite) TestOutputContent() {
	s.setupRecordingExecHandler()
	s.InitMcpClient()

	for _, tc := range []struct {
		tool     string
		params   map[string]any
		expected []string
	}{
		{
			tool:     "ovn_show",
			params:   map[string]any{"namespace": ovnTestNamespace, "name": ovnTestPodName, "database": "nbdb"},
			expected: []string{`"database":"nbdb"`},
		},
		{
			tool:     "ovn_get",
			params:   map[string]any{"namespace": ovnTestNamespace, "name": ovnTestPodName, "database": "sbdb", "table": "Chassis"},
			expected: []string{`"database":"sbdb"`, `"table":"Chassis"`},
		},
		{
			tool:   "ovn_lflow_list",
			params: map[string]any{"namespace": ovnTestNamespace, "name": ovnTestPodName},
		},
		{
			tool:     "ovn_trace",
			params:   map[string]any{"namespace": ovnTestNamespace, "name": ovnTestPodName, "datapath": "test-datapath", "microflow": `inport=="port1" && eth.src==00:00:00:00:00:01 && ip4.src==10.244.0.5 && ip4.dst==10.244.1.5`},
			expected: []string{`"datapath":"test-datapath"`},
		},
	} {
		s.Run(tc.tool+" returns output", func() {
			result, err := s.CallTool(tc.tool, tc.params)
			s.Require().NoError(err)
			s.Require().NotNil(result)
			s.Falsef(result.IsError, "call tool failed: %v", result.Content)
			text := result.Content[0].(*mcp.TextContent).Text
			s.Contains(text, "mock-output")
			for _, expected := range tc.expected {
				s.Contains(text, expected)
			}
		})
	}
}

func TestOVNKubernetes(t *testing.T) {
	suite.Run(t, new(OVNKubernetesSuite))
}
