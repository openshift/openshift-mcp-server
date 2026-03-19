package ocp

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
)

type NodesDebugSuite struct {
	suite.Suite
}

func (s *NodesDebugSuite) TestNodesDebugExecCreatesPrivilegedPod() {
	env := NewNodeDebugTestEnv(s.T())
	env.Pods.Logs = "kernel 6.8"

	out, err := NodesDebugExec(context.Background(), env.Client, "", "worker-0", "", []string{"uname", "-a"}, 2*time.Minute)
	s.Run("returns logs on success", func() {
		s.Require().NoError(err)
		s.Equal("kernel 6.8", out)
	})

	created := env.Pods.Created
	s.Require().NotNil(created, "expected debug pod to be created")

	s.Run("uses default namespace fallback", func() {
		s.Equal("default", created.Namespace)
	})
	s.Run("targets correct node", func() {
		s.Equal("worker-0", created.Spec.NodeName)
	})
	s.Run("deletes pod after execution", func() {
		s.True(env.Pods.Deleted)
	})
	s.Run("creates single container with correct defaults", func() {
		s.Require().Len(created.Spec.Containers, 1)
		container := created.Spec.Containers[0]
		s.Equal(DefaultNodeDebugImage, container.Image)
		s.Equal([]string{"uname", "-a"}, container.Command)
		s.Require().NotNil(container.SecurityContext)
		s.Require().NotNil(container.SecurityContext.Privileged)
		s.True(*container.SecurityContext.Privileged)
		s.Require().Len(container.VolumeMounts, 1)
		s.Equal("/host", container.VolumeMounts[0].MountPath)
	})
	s.Run("runs as root", func() {
		s.Require().NotNil(created.Spec.SecurityContext)
		s.Require().NotNil(created.Spec.SecurityContext.RunAsUser)
		s.Equal(int64(0), *created.Spec.SecurityContext.RunAsUser)
	})
	s.Run("mounts host root volume", func() {
		s.Require().Len(created.Spec.Volumes, 1)
		s.Require().NotNil(created.Spec.Volumes[0].HostPath)
	})
}

func (s *NodesDebugSuite) TestNodesDebugExecReturnsErrorForNonZeroExit() {
	env := NewNodeDebugTestEnv(s.T())
	env.Pods.ExitCode = 5
	env.Pods.TerminatedReason = "Error"
	env.Pods.TerminatedMessage = "some failure"
	env.Pods.Logs = "bad things happened"

	out, err := NodesDebugExec(context.Background(), env.Client, "debug-ns", "infra-node", "registry.example/custom:latest", []string{"journalctl", "-xe"}, time.Minute)

	s.Run("returns error with logs included", func() {
		s.Require().Error(err)
		s.Contains(err.Error(), "bad things happened")
		s.Contains(err.Error(), "command exited with code 5")
	})
	s.Run("returns empty output on error", func() {
		s.Empty(out)
	})
	s.Run("uses provided namespace and image", func() {
		s.Require().NotNil(env.Pods.Created)
		s.Equal("debug-ns", env.Pods.Created.Namespace)
		s.Equal("registry.example/custom:latest", env.Pods.Created.Spec.Containers[0].Image)
	})
}

func (s *NodesDebugSuite) TestCreateDebugPod() {
	env := NewNodeDebugTestEnv(s.T())

	created, err := createDebugPod(context.Background(), env.Client, "worker-1", "test-ns", "custom:v1", []string{"ls", "-la"})
	s.Require().NoError(err)
	s.Require().NotNil(created)

	s.Run("sets correct namespace", func() {
		s.Equal("test-ns", created.Namespace)
	})
	s.Run("targets correct node", func() {
		s.Equal("worker-1", created.Spec.NodeName)
	})
	s.Run("generates valid pod name", func() {
		s.True(strings.HasPrefix(created.Name, "node-debug-worker-1-"))
		s.LessOrEqual(len(created.Name), 63, "pod name exceeds DNS label length")
	})
	s.Run("uses specified image and command", func() {
		s.Require().Len(created.Spec.Containers, 1)
		s.Equal("custom:v1", created.Spec.Containers[0].Image)
		s.Equal([]string{"ls", "-la"}, created.Spec.Containers[0].Command)
		s.Require().NotNil(created.Spec.Containers[0].SecurityContext)
		s.True(*created.Spec.Containers[0].SecurityContext.Privileged)
	})
}

func (s *NodesDebugSuite) TestPollForCompletion() {
	tests := []struct {
		name             string
		exitCode         int32
		terminatedReason string
		waitingReason    string
		waitingMessage   string
		expectError      bool
		expectTerminated bool
		errorContains    []string
		expectedExitCode int32
		expectedReason   string
	}{
		{
			name:             "successful completion",
			exitCode:         0,
			expectTerminated: true,
			expectedExitCode: 0,
		},
		{
			name:             "non-zero exit code",
			exitCode:         42,
			terminatedReason: "Error",
			expectTerminated: true,
			expectedExitCode: 42,
			expectedReason:   "Error",
		},
		{
			name:           "image pull error",
			waitingReason:  "ErrImagePull",
			waitingMessage: "image not found",
			expectError:    true,
			errorContains:  []string{"ErrImagePull", "image not found"},
		},
		{
			name:           "image pull backoff",
			waitingReason:  "ImagePullBackOff",
			waitingMessage: "back-off pulling image",
			expectError:    true,
			errorContains:  []string{"ImagePullBackOff", "back-off pulling image"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			env := NewNodeDebugTestEnv(s.T())
			env.Pods.ExitCode = tt.exitCode
			env.Pods.TerminatedReason = tt.terminatedReason
			env.Pods.WaitingReason = tt.waitingReason
			env.Pods.WaitingMessage = tt.waitingMessage

			created, _ := createDebugPod(context.Background(), env.Client, "node-1", "default", DefaultNodeDebugImage, []string{"echo", "test"})

			terminated, lastPod, waitMsg, err := pollForCompletion(context.Background(), env.Client, "default", created.Name, time.Minute)

			if tt.expectError {
				s.Require().Error(err)
				for _, substr := range tt.errorContains {
					s.Contains(err.Error(), substr)
				}
				return
			}

			s.Require().NoError(err)

			if tt.expectTerminated {
				s.Require().NotNil(terminated)
				s.Equal(tt.expectedExitCode, terminated.ExitCode)
				if tt.expectedReason != "" {
					s.Equal(tt.expectedReason, terminated.Reason)
				}
				s.NotNil(lastPod)
			}

			if tt.waitingReason == "" {
				s.Empty(waitMsg)
			}
		})
	}
}

func (s *NodesDebugSuite) TestRetrieveLogs() {
	env := NewNodeDebugTestEnv(s.T())
	env.Pods.Logs = "  some output with whitespace  \n"

	created, _ := createDebugPod(context.Background(), env.Client, "node-1", "default", DefaultNodeDebugImage, []string{"echo", "test"})

	logs, err := retrieveLogs(context.Background(), env.Client, "default", created.Name)
	s.Require().NoError(err)
	s.Equal("some output with whitespace", logs)
}

func (s *NodesDebugSuite) TestProcessResults() {
	tests := []struct {
		name           string
		terminated     *corev1.ContainerStateTerminated
		pod            *corev1.Pod
		waitMsg        string
		logs           string
		expectError    bool
		errorContains  []string
		expectedResult string
	}{
		{
			name:           "successful completion",
			terminated:     &corev1.ContainerStateTerminated{ExitCode: 0},
			logs:           "success output",
			expectedResult: "success output",
		},
		{
			name: "non-zero exit code with logs",
			terminated: &corev1.ContainerStateTerminated{
				ExitCode: 127,
				Reason:   "CommandNotFound",
				Message:  "command not found",
			},
			logs:          "error logs",
			expectError:   true,
			errorContains: []string{"127", "CommandNotFound", "command not found", "error logs", "Output:"},
		},
		{
			name:          "non-zero exit code without reason but with logs",
			terminated:    &corev1.ContainerStateTerminated{ExitCode: 1},
			logs:          "failed output",
			expectError:   true,
			errorContains: []string{"command exited with code 1", "failed output", "Output:"},
		},
		{
			name:          "non-zero exit code without logs",
			terminated:    &corev1.ContainerStateTerminated{ExitCode: 1},
			expectError:   true,
			errorContains: []string{"command exited with code 1"},
		},
		{
			name:          "pod failed with logs",
			pod:           &corev1.Pod{Status: corev1.PodStatus{Reason: "Evicted"}},
			logs:          "pod evicted logs",
			expectError:   true,
			errorContains: []string{"Evicted", "pod evicted logs", "Output:"},
		},
		{
			name:          "pod failed without logs",
			pod:           &corev1.Pod{Status: corev1.PodStatus{Reason: "Evicted"}},
			expectError:   true,
			errorContains: []string{"Evicted"},
		},
		{
			name:          "container waiting with logs",
			waitMsg:       "container waiting: ImagePullBackOff",
			logs:          "waiting logs",
			expectError:   true,
			errorContains: []string{"did not complete", "waiting logs", "Output:"},
		},
		{
			name:          "container waiting without logs",
			waitMsg:       "container waiting: ImagePullBackOff",
			expectError:   true,
			errorContains: []string{"did not complete"},
		},
		{
			name:          "no terminal state with logs",
			logs:          "incomplete logs",
			expectError:   true,
			errorContains: []string{"did not reach a terminal state", "incomplete logs", "Output:"},
		},
		{
			name:          "no terminal state without logs",
			expectError:   true,
			errorContains: []string{"did not reach a terminal state"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result, err := processResults(tt.terminated, tt.pod, tt.waitMsg, tt.logs)

			if tt.expectError {
				s.Require().Error(err)
				for _, substr := range tt.errorContains {
					s.Contains(err.Error(), substr)
				}
				s.Empty(result)
			} else {
				s.Require().NoError(err)
				s.Equal(tt.expectedResult, result)
			}
		})
	}
}

func (s *NodesDebugSuite) TestSanitizeForName() {
	tests := []struct {
		input    string
		expected string
	}{
		{"worker-0", "worker-0"},
		{"WORKER-0", "worker-0"},
		{"worker.0", "worker-0"},
		{"worker_0", "worker-0"},
		{"ip-10-0-1-42.ec2.internal", "ip-10-0-1-42-ec2-internal"},
		{"", "node"},
		{"---", "node"},
		{strings.Repeat("a", 50), strings.Repeat("a", 40)},
		{"Worker-Node_123.domain", "worker-node-123-domain"},
	}

	for _, tt := range tests {
		s.Run(fmt.Sprintf("sanitize(%q)", tt.input), func() {
			s.Equal(tt.expected, sanitizeForName(tt.input))
		})
	}
}

func TestNodesDebug(t *testing.T) {
	suite.Run(t, new(NodesDebugSuite))
}
