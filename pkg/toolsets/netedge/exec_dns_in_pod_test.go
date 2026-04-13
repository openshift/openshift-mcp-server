package netedge

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockPodExecutor struct {
	createdPod *corev1.Pod
	createErr  error

	waitPod *corev1.Pod
	waitErr error

	logs   string
	logErr error

	deleteErr error

	// Captures for assertions
	lastNamespace string
	lastPodName   string
	lastPod       *corev1.Pod
	deleteCalled  bool
}

func (m *mockPodExecutor) CreatePod(_ context.Context, namespace string, pod *corev1.Pod) (*corev1.Pod, error) {
	m.lastNamespace = namespace
	m.lastPod = pod
	if m.createErr != nil {
		return nil, m.createErr
	}
	if m.createdPod != nil {
		return m.createdPod, nil
	}
	// Return the input pod with phase Pending as default
	created := pod.DeepCopy()
	created.Status.Phase = corev1.PodPending
	return created, nil
}

func (m *mockPodExecutor) WaitForPod(_ context.Context, _ string, name string, _ time.Duration) (*corev1.Pod, error) {
	m.lastPodName = name
	if m.waitErr != nil {
		return nil, m.waitErr
	}
	return m.waitPod, nil
}

func (m *mockPodExecutor) GetPodLogs(_ context.Context, _ string, name string) (string, error) {
	m.lastPodName = name
	if m.logErr != nil {
		return "", m.logErr
	}
	return m.logs, nil
}

func (m *mockPodExecutor) DeletePod(_ context.Context, _ string, _ string) error {
	m.deleteCalled = true
	return m.deleteErr
}

func succeededPod(name, namespace string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodSucceeded,
		},
	}
}

func failedPod(name, namespace string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
		},
	}
}

const sampleDigOutput = `; <<>> DiG 9.16.23 <<>> @172.30.0.10 kubernetes.default.svc.cluster.local A
;; ANSWER SECTION:
kubernetes.default.svc.cluster.local. 30 IN A 172.30.0.1

;; Query time: 1 msec
;; SERVER: 172.30.0.10#53(172.30.0.10)
`

func (s *NetEdgeTestSuite) TestExecDNSInPodHandler() {
	s.Run("success query", func() {
		mock := &mockPodExecutor{
			waitPod: succeededPod("mcp-dns-probe-abc123", "test-ns"),
			logs:    sampleDigOutput,
		}
		s.SetArgs(map[string]interface{}{
			"namespace":     "test-ns",
			"target_server": "172.30.0.10",
			"target_name":   "kubernetes.default.svc.cluster.local",
			"record_type":   "A",
		})
		handler := makeExecDNSInPodHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NoError(result.Error)

		var res ExecDNSResult
		jsonErr := json.Unmarshal([]byte(result.Content), &res)
		s.Require().NoError(jsonErr)
		s.Assert().Equal("Succeeded", res.Phase)
		s.Assert().Contains(res.Output, "172.30.0.1")
		s.Assert().NotEmpty(res.PodName)

		structured, ok := result.StructuredContent.(ExecDNSResult)
		s.Require().True(ok)
		s.Assert().Equal("Succeeded", structured.Phase)
		s.Assert().True(mock.deleteCalled, "pod should be cleaned up")
	})

	s.Run("missing namespace parameter", func() {
		mock := &mockPodExecutor{}
		s.SetArgs(map[string]interface{}{
			"target_server": "172.30.0.10",
			"target_name":   "example.com",
		})
		handler := makeExecDNSInPodHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NotNil(result.Error)
		s.Assert().Contains(result.Error.Error(), "namespace parameter is required")
	})

	s.Run("missing target_server parameter", func() {
		mock := &mockPodExecutor{}
		s.SetArgs(map[string]interface{}{
			"namespace":   "test-ns",
			"target_name": "example.com",
		})
		handler := makeExecDNSInPodHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NotNil(result.Error)
		s.Assert().Contains(result.Error.Error(), "target_server parameter is required")
	})

	s.Run("missing target_name parameter", func() {
		mock := &mockPodExecutor{}
		s.SetArgs(map[string]interface{}{
			"namespace":     "test-ns",
			"target_server": "172.30.0.10",
		})
		handler := makeExecDNSInPodHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NotNil(result.Error)
		s.Assert().Contains(result.Error.Error(), "target_name parameter is required")
	})

	s.Run("default record_type is A", func() {
		mock := &mockPodExecutor{
			waitPod: succeededPod("mcp-dns-probe-abc123", "test-ns"),
			logs:    sampleDigOutput,
		}
		s.SetArgs(map[string]interface{}{
			"namespace":     "test-ns",
			"target_server": "172.30.0.10",
			"target_name":   "kubernetes.default.svc.cluster.local",
		})
		handler := makeExecDNSInPodHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NoError(result.Error)

		// Verify the pod command includes "A" as the record type
		s.Require().NotNil(mock.lastPod)
		s.Require().Len(mock.lastPod.Spec.Containers, 1)
		cmd := mock.lastPod.Spec.Containers[0].Command
		s.Assert().Equal("A", cmd[len(cmd)-1])
	})

	s.Run("pod creation failure", func() {
		mock := &mockPodExecutor{
			createErr: fmt.Errorf("forbidden: namespace test-ns not found"),
		}
		s.SetArgs(map[string]interface{}{
			"namespace":     "test-ns",
			"target_server": "172.30.0.10",
			"target_name":   "example.com",
		})
		handler := makeExecDNSInPodHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NotNil(result.Error)
		s.Assert().Contains(result.Error.Error(), "failed to create DNS probe pod")
		s.Assert().Contains(result.Error.Error(), "namespace test-ns not found")
	})

	s.Run("pod wait timeout", func() {
		mock := &mockPodExecutor{
			waitErr: fmt.Errorf("timed out waiting for pod test-ns/mcp-dns-probe-abc123 to complete"),
		}
		s.SetArgs(map[string]interface{}{
			"namespace":     "test-ns",
			"target_server": "172.30.0.10",
			"target_name":   "example.com",
		})
		handler := makeExecDNSInPodHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NotNil(result.Error)
		s.Assert().Contains(result.Error.Error(), "error waiting for DNS probe pod")
	})

	s.Run("pod fails with Failed phase", func() {
		mock := &mockPodExecutor{
			waitPod: failedPod("mcp-dns-probe-abc123", "test-ns"),
			logs:    "dig: command failed",
		}
		s.SetArgs(map[string]interface{}{
			"namespace":     "test-ns",
			"target_server": "172.30.0.10",
			"target_name":   "example.com",
		})
		handler := makeExecDNSInPodHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NoError(result.Error)

		var res ExecDNSResult
		jsonErr := json.Unmarshal([]byte(result.Content), &res)
		s.Require().NoError(jsonErr)
		s.Assert().Equal("Failed", res.Phase)
		s.Assert().Contains(res.Output, "command failed")
	})

	s.Run("log retrieval failure returns partial result", func() {
		mock := &mockPodExecutor{
			waitPod: succeededPod("mcp-dns-probe-abc123", "test-ns"),
			logErr:  fmt.Errorf("container not found"),
		}
		s.SetArgs(map[string]interface{}{
			"namespace":     "test-ns",
			"target_server": "172.30.0.10",
			"target_name":   "example.com",
		})
		handler := makeExecDNSInPodHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		// Log retrieval failure should NOT be a protocol error
		s.Require().NoError(result.Error)

		var res ExecDNSResult
		jsonErr := json.Unmarshal([]byte(result.Content), &res)
		s.Require().NoError(jsonErr)
		s.Assert().Equal("Succeeded", res.Phase)
		s.Assert().Contains(res.Output, "failed to retrieve pod logs")
	})

	s.Run("cleanup failure does not affect result", func() {
		mock := &mockPodExecutor{
			waitPod:   succeededPod("mcp-dns-probe-abc123", "test-ns"),
			logs:      sampleDigOutput,
			deleteErr: fmt.Errorf("delete permission denied"),
		}
		s.SetArgs(map[string]interface{}{
			"namespace":     "test-ns",
			"target_server": "172.30.0.10",
			"target_name":   "example.com",
		})
		handler := makeExecDNSInPodHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NoError(result.Error)

		var res ExecDNSResult
		jsonErr := json.Unmarshal([]byte(result.Content), &res)
		s.Require().NoError(jsonErr)
		s.Assert().Equal("Succeeded", res.Phase)
		s.Assert().Contains(res.Output, "172.30.0.1")
		s.Assert().True(mock.deleteCalled)
	})

	s.Run("correct pod spec is constructed", func() {
		mock := &mockPodExecutor{
			waitPod: succeededPod("mcp-dns-probe-abc123", "my-namespace"),
			logs:    sampleDigOutput,
		}
		s.SetArgs(map[string]interface{}{
			"namespace":     "my-namespace",
			"target_server": "10.0.0.10",
			"target_name":   "myservice.default.svc.cluster.local",
			"record_type":   "AAAA",
		})
		handler := makeExecDNSInPodHandler(mock)

		_, err := handler(s.params)
		s.Require().NoError(err)

		s.Require().NotNil(mock.lastPod)
		s.Assert().Equal("my-namespace", mock.lastNamespace)
		s.Assert().Equal(corev1.RestartPolicyNever, mock.lastPod.Spec.RestartPolicy)
		s.Require().Len(mock.lastPod.Spec.Containers, 1)

		container := mock.lastPod.Spec.Containers[0]
		s.Assert().Equal(dnsProbeImage, container.Image)
		s.Assert().Equal([]string{"/usr/bin/dig", "@10.0.0.10", "myservice.default.svc.cluster.local", "AAAA"}, container.Command)
	})

	s.Run("pod spec disables service account token mount", func() {
		mock := &mockPodExecutor{
			waitPod: succeededPod("mcp-dns-probe-abc123", "test-ns"),
			logs:    sampleDigOutput,
		}
		s.SetArgs(map[string]interface{}{
			"namespace":     "test-ns",
			"target_server": "172.30.0.10",
			"target_name":   "example.com",
		})
		handler := makeExecDNSInPodHandler(mock)

		_, err := handler(s.params)
		s.Require().NoError(err)

		s.Require().NotNil(mock.lastPod)
		s.Require().NotNil(mock.lastPod.Spec.AutomountServiceAccountToken)
		s.Assert().False(*mock.lastPod.Spec.AutomountServiceAccountToken, "SA token should not be mounted")
	})

	s.Run("rejects non-IP target_server", func() {
		mock := &mockPodExecutor{}
		s.SetArgs(map[string]interface{}{
			"namespace":     "test-ns",
			"target_server": "not-an-ip",
			"target_name":   "example.com",
		})
		handler := makeExecDNSInPodHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NotNil(result.Error)
		s.Assert().Contains(result.Error.Error(), "target_server must be a valid IP address")
	})

	s.Run("rejects option-shaped target_name", func() {
		testCases := []struct {
			name       string
			targetName string
		}{
			{"dash prefix", "-f /etc/hostname"},
			{"plus prefix", "+trace"},
			{"at prefix", "@8.8.8.8"},
		}
		for _, tc := range testCases {
			s.Run(tc.name, func() {
				mock := &mockPodExecutor{}
				s.SetArgs(map[string]interface{}{
					"namespace":     "test-ns",
					"target_server": "172.30.0.10",
					"target_name":   tc.targetName,
				})
				handler := makeExecDNSInPodHandler(mock)

				result, err := handler(s.params)

				s.Require().NoError(err)
				s.Require().NotNil(result.Error)
				s.Assert().Contains(result.Error.Error(), "target_name must be a DNS name")
			})
		}
	})

	s.Run("rejects unsupported record_type", func() {
		testCases := []struct {
			name       string
			recordType string
		}{
			{"dash prefix", "-f"},
			{"plus prefix", "+short"},
			{"at prefix", "@server"},
			{"unknown type", "UNKNOWN"},
		}
		for _, tc := range testCases {
			s.Run(tc.name, func() {
				mock := &mockPodExecutor{}
				s.SetArgs(map[string]interface{}{
					"namespace":     "test-ns",
					"target_server": "172.30.0.10",
					"target_name":   "example.com",
					"record_type":   tc.recordType,
				})
				handler := makeExecDNSInPodHandler(mock)

				result, err := handler(s.params)

				s.Require().NoError(err)
				s.Require().NotNil(result.Error)
				s.Assert().Contains(result.Error.Error(), "unsupported record_type")
			})
		}
	})

	s.Run("pod spec includes security context and resources", func() {
		mock := &mockPodExecutor{
			waitPod: succeededPod("mcp-dns-probe-abc123", "test-ns"),
			logs:    sampleDigOutput,
		}
		s.SetArgs(map[string]interface{}{
			"namespace":     "test-ns",
			"target_server": "172.30.0.10",
			"target_name":   "example.com",
		})
		handler := makeExecDNSInPodHandler(mock)

		_, err := handler(s.params)
		s.Require().NoError(err)

		s.Require().NotNil(mock.lastPod)
		s.Require().Len(mock.lastPod.Spec.Containers, 1)
		container := mock.lastPod.Spec.Containers[0]

		s.Require().NotNil(container.SecurityContext)
		s.Assert().True(*container.SecurityContext.RunAsNonRoot)
		s.Assert().False(*container.SecurityContext.AllowPrivilegeEscalation)
		s.Require().NotNil(container.SecurityContext.Capabilities)
		s.Assert().Contains(container.SecurityContext.Capabilities.Drop, corev1.Capability("ALL"))

		s.Assert().False(container.Resources.Requests.Cpu().IsZero())
		s.Assert().False(container.Resources.Requests.Memory().IsZero())
		s.Assert().False(container.Resources.Limits.Cpu().IsZero())
		s.Assert().False(container.Resources.Limits.Memory().IsZero())
	})
}
