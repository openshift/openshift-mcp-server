package mcp

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/containers/kubernetes-mcp-server/internal/test"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

type NodesSuite struct {
	BaseMcpSuite
	mockServer *test.MockServer
}

func (s *NodesSuite) SetupTest() {
	s.BaseMcpSuite.SetupTest()
	s.mockServer = test.NewMockServer()
	s.mockServer.Handle(test.NewDiscoveryClientHandler())
	s.Cfg.KubeConfig = s.mockServer.KubeconfigFile(s.T())
}

func (s *NodesSuite) TearDownTest() {
	s.BaseMcpSuite.TearDownTest()
	if s.mockServer != nil {
		s.mockServer.Close()
	}
}

func (s *NodesSuite) TestNodesLog() {
	s.mockServer.Handle(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Get Node response
		if req.URL.Path == "/api/v1/nodes/existing-node" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"apiVersion": "v1",
				"kind": "Node",
				"metadata": {
					"name": "existing-node"
				}
			}`))
			return
		}
		// Get Proxy Logs
		if req.URL.Path == "/api/v1/nodes/existing-node/proxy/logs" {
			w.Header().Set("Content-Type", "text/plain")
			query := req.URL.Query().Get("query")
			var logContent string
			switch query {
			case "/empty.log":
				logContent = ""
			case "/kubelet.log":
				logContent = "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\n"
			default:
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_, err := strconv.Atoi(req.URL.Query().Get("tailLines"))
			if err == nil {
				logContent = "Line 4\nLine 5\n"
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(logContent))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	s.InitMcpClient()
	s.Run("nodes_log(name=nil)", func() {
		toolResult, err := s.CallTool("nodes_log", map[string]interface{}{})
		s.Require().NotNil(toolResult, "toolResult should not be nil")
		s.Run("has error", func() {
			s.Truef(toolResult.IsError, "call tool should fail")
			s.Nilf(err, "call tool should not return error object")
		})
		s.Run("describes missing name", func() {
			expectedMessage := "failed to get node log, missing argument name"
			s.Equalf(expectedMessage, toolResult.Content[0].(mcp.TextContent).Text,
				"expected descriptive error '%s', got %v", expectedMessage, toolResult.Content[0].(mcp.TextContent).Text)
		})
	})
	s.Run("nodes_log(name=existing-node, query=nil)", func() {
		toolResult, err := s.CallTool("nodes_log", map[string]interface{}{
			"name": "existing-node",
		})
		s.Require().NotNil(toolResult, "toolResult should not be nil")
		s.Run("has error", func() {
			s.Truef(toolResult.IsError, "call tool should fail")
			s.Nilf(err, "call tool should not return error object")
		})
		s.Run("describes missing name", func() {
			expectedMessage := "failed to get node log, missing argument query"
			s.Regexpf(expectedMessage, toolResult.Content[0].(mcp.TextContent).Text,
				"expected descriptive error '%s', got %v", expectedMessage, toolResult.Content[0].(mcp.TextContent).Text)
		})
	})
	s.Run("nodes_log(name=inexistent-node, query=/kubelet.log)", func() {
		toolResult, err := s.CallTool("nodes_log", map[string]interface{}{
			"name":  "inexistent-node",
			"query": "/kubelet.log",
		})
		s.Require().NotNil(toolResult, "toolResult should not be nil")
		s.Run("has error", func() {
			s.Truef(toolResult.IsError, "call tool should fail")
			s.Nilf(err, "call tool should not return error object")
		})
		s.Run("describes missing node", func() {
			expectedMessage := "failed to get node log for inexistent-node: failed to get node inexistent-node: the server could not find the requested resource (get nodes inexistent-node)"
			s.Equalf(expectedMessage, toolResult.Content[0].(mcp.TextContent).Text,
				"expected descriptive error '%s', got %v", expectedMessage, toolResult.Content[0].(mcp.TextContent).Text)
		})
	})
	s.Run("nodes_log(name=existing-node, query=/missing.log)", func() {
		toolResult, err := s.CallTool("nodes_log", map[string]interface{}{
			"name":  "existing-node",
			"query": "/missing.log",
		})
		s.Require().NotNil(toolResult, "toolResult should not be nil")
		s.Run("has error", func() {
			s.Truef(toolResult.IsError, "call tool should fail")
			s.Nilf(err, "call tool should not return error object")
		})
		s.Run("describes missing log file", func() {
			expectedMessage := "failed to get node log for existing-node: failed to get node logs: the server could not find the requested resource"
			s.Equalf(expectedMessage, toolResult.Content[0].(mcp.TextContent).Text,
				"expected descriptive error '%s', got %v", expectedMessage, toolResult.Content[0].(mcp.TextContent).Text)
		})
	})
	s.Run("nodes_log(name=existing-node, query=/empty.log)", func() {
		toolResult, err := s.CallTool("nodes_log", map[string]interface{}{
			"name":  "existing-node",
			"query": "/empty.log",
		})
		s.Require().NotNil(toolResult, "toolResult should not be nil")
		s.Run("no error", func() {
			s.Falsef(toolResult.IsError, "call tool should succeed")
			s.Nilf(err, "call tool should not return error object")
		})
		s.Run("describes empty log", func() {
			expectedMessage := "The node existing-node has not logged any message yet or the log file is empty"
			s.Equalf(expectedMessage, toolResult.Content[0].(mcp.TextContent).Text,
				"expected descriptive message '%s', got %v", expectedMessage, toolResult.Content[0].(mcp.TextContent).Text)
		})
	})
	s.Run("nodes_log(name=existing-node, query=/kubelet.log)", func() {
		toolResult, err := s.CallTool("nodes_log", map[string]interface{}{
			"name":  "existing-node",
			"query": "/kubelet.log",
		})
		s.Require().NotNil(toolResult, "toolResult should not be nil")
		s.Run("no error", func() {
			s.Falsef(toolResult.IsError, "call tool should succeed")
			s.Nilf(err, "call tool should not return error object")
		})
		s.Run("returns full log", func() {
			expectedMessage := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\n"
			s.Equalf(expectedMessage, toolResult.Content[0].(mcp.TextContent).Text,
				"expected log content '%s', got %v", expectedMessage, toolResult.Content[0].(mcp.TextContent).Text)
		})
	})
	for _, tailCase := range []interface{}{2, int64(2), float64(2)} {
		s.Run("nodes_log(name=existing-node, query=/kubelet.log, tailLines=2)", func() {
			toolResult, err := s.CallTool("nodes_log", map[string]interface{}{
				"name":      "existing-node",
				"query":     "/kubelet.log",
				"tailLines": tailCase,
			})
			s.Require().NotNil(toolResult, "toolResult should not be nil")
			s.Run("no error", func() {
				s.Falsef(toolResult.IsError, "call tool should succeed")
				s.Nilf(err, "call tool should not return error object")
			})
			s.Run("returns tail log", func() {
				expectedMessage := "Line 4\nLine 5\n"
				s.Equalf(expectedMessage, toolResult.Content[0].(mcp.TextContent).Text,
					"expected log content '%s', got %v", expectedMessage, toolResult.Content[0].(mcp.TextContent).Text)
			})
		})
		s.Run("nodes_log(name=existing-node, query=/kubelet.log, tailLines=-1)", func() {
			toolResult, err := s.CallTool("nodes_log", map[string]interface{}{
				"name":  "existing-node",
				"query": "/kubelet.log",
				"tail":  -1,
			})
			s.Require().NotNil(toolResult, "toolResult should not be nil")
			s.Run("no error", func() {
				s.Falsef(toolResult.IsError, "call tool should succeed")
				s.Nilf(err, "call tool should not return error object")
			})
			s.Run("returns full log", func() {
				expectedMessage := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\n"
				s.Equalf(expectedMessage, toolResult.Content[0].(mcp.TextContent).Text,
					"expected log content '%s', got %v", expectedMessage, toolResult.Content[0].(mcp.TextContent).Text)
			})
		})
	}
}

func (s *NodesSuite) TestNodesLogDenied() {
	s.Require().NoError(toml.Unmarshal([]byte(`
		denied_resources = [ { version = "v1", kind = "Node" } ]
	`), s.Cfg), "Expected to parse denied resources config")
	s.InitMcpClient()
	s.Run("nodes_log (denied)", func() {
		toolResult, err := s.CallTool("nodes_log", map[string]interface{}{
			"name":  "does-not-matter",
			"query": "/does-not-matter-either.log",
		})
		s.Require().NotNil(toolResult, "toolResult should not be nil")
		s.Run("has error", func() {
			s.Truef(toolResult.IsError, "call tool should fail")
			s.Nilf(err, "call tool should not return error object")
		})
		s.Run("describes denial", func() {
			msg := toolResult.Content[0].(mcp.TextContent).Text
			s.Contains(msg, "resource not allowed:")
			expectedMessage := "failed to get node log for does-not-matter:(.+:)? resource not allowed: /v1, Kind=Node"
			s.Regexpf(expectedMessage, msg,
				"expected descriptive error '%s', got %v", expectedMessage, msg)
		})
	})
}

func (s *NodesSuite) TestNodesStatsSummary() {
	s.mockServer.Handle(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Get Node response
		if req.URL.Path == "/api/v1/nodes/existing-node" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"apiVersion": "v1",
				"kind": "Node",
				"metadata": {
					"name": "existing-node"
				}
			}`))
			return
		}
		// Get Stats Summary response
		if req.URL.Path == "/api/v1/nodes/existing-node/proxy/stats/summary" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"node": {
					"nodeName": "existing-node",
					"cpu": {
						"time": "2025-10-27T00:00:00Z",
						"usageNanoCores": 1000000000,
						"usageCoreNanoSeconds": 5000000000
					},
					"memory": {
						"time": "2025-10-27T00:00:00Z",
						"availableBytes": 8000000000,
						"usageBytes": 4000000000,
						"workingSetBytes": 3500000000
					}
				},
				"pods": []
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	s.InitMcpClient()
	s.Run("nodes_stats_summary(name=nil)", func() {
		toolResult, err := s.CallTool("nodes_stats_summary", map[string]interface{}{})
		s.Require().NotNil(toolResult, "toolResult should not be nil")
		s.Run("has error", func() {
			s.Truef(toolResult.IsError, "call tool should fail")
			s.Nilf(err, "call tool should not return error object")
		})
		s.Run("describes missing name", func() {
			expectedMessage := "failed to get node stats summary, missing argument name"
			s.Equalf(expectedMessage, toolResult.Content[0].(mcp.TextContent).Text,
				"expected descriptive error '%s', got %v", expectedMessage, toolResult.Content[0].(mcp.TextContent).Text)
		})
	})
	s.Run("nodes_stats_summary(name=inexistent-node)", func() {
		toolResult, err := s.CallTool("nodes_stats_summary", map[string]interface{}{
			"name": "inexistent-node",
		})
		s.Require().NotNil(toolResult, "toolResult should not be nil")
		s.Run("has error", func() {
			s.Truef(toolResult.IsError, "call tool should fail")
			s.Nilf(err, "call tool should not return error object")
		})
		s.Run("describes missing node", func() {
			expectedMessage := "failed to get node stats summary for inexistent-node: failed to get node inexistent-node: the server could not find the requested resource (get nodes inexistent-node)"
			s.Equalf(expectedMessage, toolResult.Content[0].(mcp.TextContent).Text,
				"expected descriptive error '%s', got %v", expectedMessage, toolResult.Content[0].(mcp.TextContent).Text)
		})
	})
	s.Run("nodes_stats_summary(name=existing-node)", func() {
		toolResult, err := s.CallTool("nodes_stats_summary", map[string]interface{}{
			"name": "existing-node",
		})
		s.Require().NotNil(toolResult, "toolResult should not be nil")
		s.Run("no error", func() {
			s.Falsef(toolResult.IsError, "call tool should succeed")
			s.Nilf(err, "call tool should not return error object")
		})
		s.Run("returns stats summary", func() {
			content := toolResult.Content[0].(mcp.TextContent).Text
			s.Containsf(content, "existing-node", "expected stats to contain node name, got %v", content)
			s.Containsf(content, "usageNanoCores", "expected stats to contain CPU metrics, got %v", content)
			s.Containsf(content, "usageBytes", "expected stats to contain memory metrics, got %v", content)
		})
	})
}

func (s *NodesSuite) TestNodesStatsSummaryDenied() {
	s.Require().NoError(toml.Unmarshal([]byte(`
		denied_resources = [ { version = "v1", kind = "Node" } ]
	`), s.Cfg), "Expected to parse denied resources config")
	s.InitMcpClient()
	s.Run("nodes_stats_summary (denied)", func() {
		toolResult, err := s.CallTool("nodes_stats_summary", map[string]interface{}{
			"name": "does-not-matter",
		})
		s.Require().NotNil(toolResult, "toolResult should not be nil")
		s.Run("has error", func() {
			s.Truef(toolResult.IsError, "call tool should fail")
			s.Nilf(err, "call tool should not return error object")
		})
		s.Run("describes denial", func() {
			msg := toolResult.Content[0].(mcp.TextContent).Text
			s.Contains(msg, "resource not allowed:")
			expectedMessage := "failed to get node stats summary for does-not-matter:(.+:)? resource not allowed: /v1, Kind=Node"
			s.Regexpf(expectedMessage, msg,
				"expected descriptive error '%s', got %v", expectedMessage, msg)
		})
	})
}

func TestNodes(t *testing.T) {
	suite.Run(t, new(NodesSuite))
}

// Tests below are for the nodes_debug_exec tool (OpenShift-specific)

type NodesDebugExecSuite struct {
	BaseMcpSuite
	mockServer *test.MockServer
}

func (s *NodesDebugExecSuite) SetupTest() {
	s.BaseMcpSuite.SetupTest()
	s.mockServer = test.NewMockServer()
	s.Cfg.KubeConfig = s.mockServer.KubeconfigFile(s.T())
}

func (s *NodesDebugExecSuite) TearDownTest() {
	s.BaseMcpSuite.TearDownTest()
	if s.mockServer != nil {
		s.mockServer.Close()
	}
}

func (s *NodesDebugExecSuite) TestNodesDebugExecTool() {
	s.Run("nodes_debug_exec with successful execution", func() {

		var (
			createdPod   v1.Pod
			deleteCalled bool
		)
		const namespace = "debug"
		const logOutput = "filesystem repaired"

		scheme := runtime.NewScheme()
		_ = v1.AddToScheme(scheme)
		codec := serializer.NewCodecFactory(scheme).UniversalDeserializer()

		s.mockServer.Handle(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch {
			case req.URL.Path == "/api":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"kind":"APIVersions","versions":["v1"],"serverAddressByClientCIDRs":[{"clientCIDR":"0.0.0.0/0"}]}`))
			case req.URL.Path == "/apis":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"kind":"APIGroupList","apiVersion":"v1","groups":[]}`))
			case req.URL.Path == "/api/v1":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"kind":"APIResourceList","apiVersion":"v1","resources":[{"name":"pods","singularName":"","namespaced":true,"kind":"Pod","verbs":["get","list","watch","create","update","patch","delete"]}]}`))
			case req.Method == http.MethodPatch && strings.HasPrefix(req.URL.Path, "/api/v1/namespaces/"+namespace+"/pods/"):
				// Handle server-side apply (PATCH with fieldManager query param)
				body, err := io.ReadAll(req.Body)
				if err != nil {
					s.T().Fatalf("failed to read apply body: %v", err)
				}
				created := &v1.Pod{}
				if _, _, err = codec.Decode(body, nil, created); err != nil {
					s.T().Fatalf("failed to decode apply body: %v", err)
				}
				createdPod = *created
				// Keep the name from the request URL if it was provided
				pathParts := strings.Split(req.URL.Path, "/")
				if len(pathParts) > 0 {
					createdPod.Name = pathParts[len(pathParts)-1]
				}
				createdPod.Namespace = namespace
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(&createdPod)
			case req.Method == http.MethodPost && req.URL.Path == "/api/v1/namespaces/"+namespace+"/pods":
				body, err := io.ReadAll(req.Body)
				if err != nil {
					s.T().Fatalf("failed to read create body: %v", err)
				}
				created := &v1.Pod{}
				if _, _, err = codec.Decode(body, nil, created); err != nil {
					s.T().Fatalf("failed to decode create body: %v", err)
				}
				createdPod = *created
				createdPod.ObjectMeta = metav1.ObjectMeta{
					Namespace: namespace,
					Name:      createdPod.GenerateName + "abc",
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(&createdPod)
			case req.Method == http.MethodGet && createdPod.Name != "" && req.URL.Path == "/api/v1/namespaces/"+namespace+"/pods/"+createdPod.Name:
				podStatus := createdPod.DeepCopy()
				podStatus.Status = v1.PodStatus{
					Phase: v1.PodSucceeded,
					ContainerStatuses: []v1.ContainerStatus{{
						Name: "debug",
						State: v1.ContainerState{Terminated: &v1.ContainerStateTerminated{
							ExitCode: 0,
						}},
					}},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(podStatus)
			case req.Method == http.MethodDelete && createdPod.Name != "" && req.URL.Path == "/api/v1/namespaces/"+namespace+"/pods/"+createdPod.Name:
				deleteCalled = true
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(&metav1.Status{Status: "Success"})
			case req.Method == http.MethodGet && createdPod.Name != "" && req.URL.Path == "/api/v1/namespaces/"+namespace+"/pods/"+createdPod.Name+"/log":
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write([]byte(logOutput))
			}
		}))

		s.InitMcpClient()
		toolResult, err := s.CallTool("nodes_debug_exec", map[string]interface{}{
			"node":      "worker-0",
			"namespace": namespace,
			"command":   []interface{}{"uname", "-a"},
		})

		s.Run("call succeeds", func() {
			s.Nilf(err, "call tool should not error: %v", err)
			s.Falsef(toolResult.IsError, "tool should not return error: %v", toolResult.Content)
			s.NotEmpty(toolResult.Content, "expected output content")
			text := toolResult.Content[0].(mcp.TextContent).Text
			s.Equalf(logOutput, text, "unexpected tool output %q", text)
		})

		s.Run("debug pod shaped correctly", func() {
			s.Require().NotNil(createdPod.Spec.Containers, "expected containers in debug pod")
			s.Require().Len(createdPod.Spec.Containers, 1, "expected single container in debug pod")
			container := createdPod.Spec.Containers[0]
			expectedCommand := []string{"uname", "-a"}
			s.Truef(equalStringSlices(container.Command, expectedCommand),
				"unexpected debug command: %v", container.Command)
			s.Require().NotNil(container.SecurityContext, "expected security context")
			s.Require().NotNil(container.SecurityContext.Privileged, "expected privileged field")
			s.Truef(*container.SecurityContext.Privileged, "expected privileged container")
			s.Require().NotEmpty(createdPod.Spec.Volumes, "expected volumes on debug pod")
			s.Require().NotNil(createdPod.Spec.Volumes[0].HostPath, "expected hostPath volume")
			s.Truef(deleteCalled, "expected debug pod to be deleted")
		})
	})
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (s *NodesDebugExecSuite) TestNodesDebugExecToolNonZeroExit() {
	s.Run("nodes_debug_exec with non-zero exit code", func() {
		const namespace = "default"
		const errorMessage = "failed"

		scheme := runtime.NewScheme()
		_ = v1.AddToScheme(scheme)
		codec := serializer.NewCodecFactory(scheme).UniversalDeserializer()

		s.mockServer.Handle(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch {
			case req.URL.Path == "/api":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"kind":"APIVersions","versions":["v1"],"serverAddressByClientCIDRs":[{"clientCIDR":"0.0.0.0/0"}]}`))
			case req.URL.Path == "/apis":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"kind":"APIGroupList","apiVersion":"v1","groups":[]}`))
			case req.URL.Path == "/api/v1":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"kind":"APIResourceList","apiVersion":"v1","resources":[{"name":"pods","singularName":"","namespaced":true,"kind":"Pod","verbs":["get","list","watch","create","update","patch","delete"]}]}`))
			case req.Method == http.MethodPatch && strings.HasPrefix(req.URL.Path, "/api/v1/namespaces/"+namespace+"/pods/"):
				// Handle server-side apply (PATCH with fieldManager query param)
				body, err := io.ReadAll(req.Body)
				if err != nil {
					s.T().Fatalf("failed to read apply body: %v", err)
				}
				pod := &v1.Pod{}
				if _, _, err = codec.Decode(body, nil, pod); err != nil {
					s.T().Fatalf("failed to decode apply body: %v", err)
				}
				// Keep the name from the request URL if it was provided
				pathParts := strings.Split(req.URL.Path, "/")
				if len(pathParts) > 0 {
					pod.Name = pathParts[len(pathParts)-1]
				}
				pod.Namespace = namespace
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(pod)
			case req.Method == http.MethodPost && req.URL.Path == "/api/v1/namespaces/"+namespace+"/pods":
				body, err := io.ReadAll(req.Body)
				if err != nil {
					s.T().Fatalf("failed to read create body: %v", err)
				}
				pod := &v1.Pod{}
				if _, _, err = codec.Decode(body, nil, pod); err != nil {
					s.T().Fatalf("failed to decode create body: %v", err)
				}
				pod.ObjectMeta = metav1.ObjectMeta{Name: pod.GenerateName + "xyz", Namespace: namespace}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(pod)
			case strings.HasPrefix(req.URL.Path, "/api/v1/namespaces/"+namespace+"/pods/") && strings.HasSuffix(req.URL.Path, "/log"):
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write([]byte(errorMessage))
			case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/api/v1/namespaces/"+namespace+"/pods/"):
				pathParts := strings.Split(req.URL.Path, "/")
				podName := pathParts[len(pathParts)-1]
				pod := &v1.Pod{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Pod",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      podName,
						Namespace: namespace,
					},
				}
				pod.Status = v1.PodStatus{
					Phase: v1.PodSucceeded,
					ContainerStatuses: []v1.ContainerStatus{{
						Name: "debug",
						State: v1.ContainerState{Terminated: &v1.ContainerStateTerminated{
							ExitCode: 2,
							Reason:   "Error",
						}},
					}},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(pod)
			}
		}))

		s.InitMcpClient()
		toolResult, err := s.CallTool("nodes_debug_exec", map[string]interface{}{
			"node":    "infra-1",
			"command": []interface{}{"journalctl"},
		})

		s.Nilf(err, "call tool should not error: %v", err)
		s.Truef(toolResult.IsError, "expected tool to return error")
		text := toolResult.Content[0].(mcp.TextContent).Text
		s.Containsf(text, "command exited with code 2", "expected exit code message, got %q", text)
		s.Containsf(text, "Error", "expected error reason included, got %q", text)
	})
}

func TestNodesDebugExec(t *testing.T) {
	suite.Run(t, new(NodesDebugExecSuite))
}
