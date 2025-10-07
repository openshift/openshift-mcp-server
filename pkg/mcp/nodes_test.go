package mcp

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/containers/kubernetes-mcp-server/internal/test"
)

func TestNodesDebugExecTool(t *testing.T) {
	testCase(t, func(c *mcpContext) {
		mockServer := test.NewMockServer()
		defer mockServer.Close()
		c.withKubeConfig(mockServer.Config())

		var (
			createdPod   v1.Pod
			deleteCalled bool
		)
		const namespace = "debug"
		const logOutput = "filesystem repaired"

		scheme := runtime.NewScheme()
		_ = v1.AddToScheme(scheme)
		codec := serializer.NewCodecFactory(scheme).UniversalDeserializer()

		mockServer.Handle(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
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
					t.Fatalf("failed to read apply body: %v", err)
				}
				created := &v1.Pod{}
				if _, _, err = codec.Decode(body, nil, created); err != nil {
					t.Fatalf("failed to decode apply body: %v", err)
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
					t.Fatalf("failed to read create body: %v", err)
				}
				created := &v1.Pod{}
				if _, _, err = codec.Decode(body, nil, created); err != nil {
					t.Fatalf("failed to decode create body: %v", err)
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

		toolResult, err := c.callTool("nodes_debug_exec", map[string]any{
			"node":      "worker-0",
			"namespace": namespace,
			"command":   []any{"uname", "-a"},
		})

		t.Run("call succeeds", func(t *testing.T) {
			if err != nil {
				t.Fatalf("call tool failed: %v", err)
			}
			if toolResult.IsError {
				t.Fatalf("tool returned error: %v", toolResult.Content)
			}
			if len(toolResult.Content) == 0 {
				t.Fatalf("expected output content")
			}
			text := toolResult.Content[0].(mcp.TextContent).Text
			if text != logOutput {
				t.Fatalf("unexpected tool output %q", text)
			}
		})

		t.Run("debug pod shaped correctly", func(t *testing.T) {
			if createdPod.Spec.Containers == nil || len(createdPod.Spec.Containers) != 1 {
				t.Fatalf("expected single container in debug pod")
			}
			container := createdPod.Spec.Containers[0]
			expectedPrefix := []string{"chroot", "/host", "uname", "-a"}
			if !equalStringSlices(container.Command, expectedPrefix) {
				t.Fatalf("unexpected debug command: %v", container.Command)
			}
			if container.SecurityContext == nil || container.SecurityContext.Privileged == nil || !*container.SecurityContext.Privileged {
				t.Fatalf("expected privileged container")
			}
			if len(createdPod.Spec.Volumes) == 0 || createdPod.Spec.Volumes[0].HostPath == nil {
				t.Fatalf("expected hostPath volume on debug pod")
			}
			if !deleteCalled {
				t.Fatalf("expected debug pod to be deleted")
			}
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

func TestNodesDebugExecToolNonZeroExit(t *testing.T) {
	testCase(t, func(c *mcpContext) {
		mockServer := test.NewMockServer()
		defer mockServer.Close()
		c.withKubeConfig(mockServer.Config())

		const namespace = "default"
		const errorMessage = "failed"

		scheme := runtime.NewScheme()
		_ = v1.AddToScheme(scheme)
		codec := serializer.NewCodecFactory(scheme).UniversalDeserializer()

		mockServer.Handle(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
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
					t.Fatalf("failed to read apply body: %v", err)
				}
				pod := &v1.Pod{}
				if _, _, err = codec.Decode(body, nil, pod); err != nil {
					t.Fatalf("failed to decode apply body: %v", err)
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
					t.Fatalf("failed to read create body: %v", err)
				}
				pod := &v1.Pod{}
				if _, _, err = codec.Decode(body, nil, pod); err != nil {
					t.Fatalf("failed to decode create body: %v", err)
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

		toolResult, err := c.callTool("nodes_debug_exec", map[string]any{
			"node":    "infra-1",
			"command": []any{"journalctl"},
		})

		if err != nil {
			t.Fatalf("call tool failed: %v", err)
		}
		if !toolResult.IsError {
			t.Fatalf("expected tool to return error")
		}
		text := toolResult.Content[0].(mcp.TextContent).Text
		if !strings.Contains(text, "command exited with code 2") {
			t.Fatalf("expected exit code message, got %q", text)
		}
		if !strings.Contains(text, "Error") {
			t.Fatalf("expected error reason included, got %q", text)
		}
	})
}
