package ocp

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
)

func TestNodesDebugExecCreatesPrivilegedPod(t *testing.T) {
	env := NewNodeDebugTestEnv(t)
	env.Pods.Logs = "kernel 6.8"

	out, err := NodesDebugExec(context.Background(), env.Kubernetes, "", "worker-0", "", []string{"uname", "-a"}, 2*time.Minute)
	if err != nil {
		t.Fatalf("NodesDebugExec returned error: %v", err)
	}
	if out != "kernel 6.8" {
		t.Fatalf("unexpected command output: %q", out)
	}

	created := env.Pods.Created
	if created == nil {
		t.Fatalf("expected debug pod to be created")
	}
	if created.Namespace != "default" {
		t.Fatalf("expected default namespace fallback, got %q", created.Namespace)
	}
	if created.Spec.NodeName != "worker-0" {
		t.Fatalf("expected pod to target node worker-0, got %q", created.Spec.NodeName)
	}
	if !env.Pods.Deleted {
		t.Fatalf("expected debug pod to be deleted after execution")
	}

	if len(created.Spec.Containers) != 1 {
		t.Fatalf("expected single container in debug pod")
	}
	container := created.Spec.Containers[0]
	if container.Image != DefaultNodeDebugImage {
		t.Fatalf("expected default image %q, got %q", DefaultNodeDebugImage, container.Image)
	}
	expectedCommand := []string{"uname", "-a"}
	if len(container.Command) != len(expectedCommand) {
		t.Fatalf("unexpected command length, got %v", container.Command)
	}
	for i, part := range expectedCommand {
		if container.Command[i] != part {
			t.Fatalf("command[%d] = %q, expected %q", i, container.Command[i], part)
		}
	}
	if container.SecurityContext == nil || container.SecurityContext.Privileged == nil || !*container.SecurityContext.Privileged {
		t.Fatalf("expected container to run privileged")
	}
	if len(container.VolumeMounts) != 1 || container.VolumeMounts[0].MountPath != "/host" {
		t.Fatalf("expected container to mount host root at /host")
	}

	if created.Spec.SecurityContext == nil || created.Spec.SecurityContext.RunAsUser == nil || *created.Spec.SecurityContext.RunAsUser != 0 {
		t.Fatalf("expected pod security context to run as root")
	}

	if len(created.Spec.Volumes) != 1 || created.Spec.Volumes[0].HostPath == nil {
		t.Fatalf("expected host root volume to be configured")
	}
}

func TestNodesDebugExecReturnsErrorForNonZeroExit(t *testing.T) {
	env := NewNodeDebugTestEnv(t)
	env.Pods.ExitCode = 5
	env.Pods.TerminatedReason = "Error"
	env.Pods.TerminatedMessage = "some failure"
	env.Pods.Logs = "bad things happened"

	out, err := NodesDebugExec(context.Background(), env.Kubernetes, "debug-ns", "infra-node", "registry.example/custom:latest", []string{"journalctl", "-xe"}, time.Minute)
	if err == nil {
		t.Fatalf("expected error for non-zero exit code")
	}
	// Logs should be included in the error message
	if !strings.Contains(err.Error(), "bad things happened") {
		t.Fatalf("expected error to contain logs, got: %v", err)
	}
	if !strings.Contains(err.Error(), "command exited with code 5") {
		t.Fatalf("expected error to contain exit code, got: %v", err)
	}
	if out != "" {
		t.Fatalf("expected empty output on error, got %q", out)
	}

	created := env.Pods.Created
	if created == nil {
		t.Fatalf("expected pod to be created")
	}
	if created.Namespace != "debug-ns" {
		t.Fatalf("expected provided namespace to be used, got %q", created.Namespace)
	}
	if containerImage := created.Spec.Containers[0].Image; containerImage != "registry.example/custom:latest" {
		t.Fatalf("expected custom image to be used, got %q", containerImage)
	}
}

func TestCreateDebugPod(t *testing.T) {
	env := NewNodeDebugTestEnv(t)

	created, err := createDebugPod(context.Background(), env.Kubernetes, "worker-1", "test-ns", "custom:v1", []string{"ls", "-la"})
	if err != nil {
		t.Fatalf("createDebugPod failed: %v", err)
	}
	if created == nil {
		t.Fatalf("expected pod to be created")
	}
	if created.Namespace != "test-ns" {
		t.Fatalf("expected namespace test-ns, got %q", created.Namespace)
	}
	if created.Spec.NodeName != "worker-1" {
		t.Fatalf("expected node worker-1, got %q", created.Spec.NodeName)
	}
	if !strings.HasPrefix(created.Name, "node-debug-worker-1-") {
		t.Fatalf("unexpected pod name: %q", created.Name)
	}
	if len(created.Name) > 63 {
		t.Fatalf("pod name exceeds DNS label length: %d characters", len(created.Name))
	}
	if len(created.Spec.Containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(created.Spec.Containers))
	}
	container := created.Spec.Containers[0]
	if container.Image != "custom:v1" {
		t.Fatalf("expected image custom:v1, got %q", container.Image)
	}
	expectedCmd := []string{"ls", "-la"}
	if len(container.Command) != len(expectedCmd) {
		t.Fatalf("expected %d command parts, got %d", len(expectedCmd), len(container.Command))
	}
	for i, part := range expectedCmd {
		if container.Command[i] != part {
			t.Fatalf("command[%d] = %q, expected %q", i, container.Command[i], part)
		}
	}
	if container.SecurityContext == nil || !*container.SecurityContext.Privileged {
		t.Fatalf("expected privileged container")
	}
}

func TestPollForCompletion(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
			env := NewNodeDebugTestEnv(t)
			env.Pods.ExitCode = tt.exitCode
			env.Pods.TerminatedReason = tt.terminatedReason
			env.Pods.WaitingReason = tt.waitingReason
			env.Pods.WaitingMessage = tt.waitingMessage

			created, _ := createDebugPod(context.Background(), env.Kubernetes, "node-1", "default", DefaultNodeDebugImage, []string{"echo", "test"})

			terminated, lastPod, waitMsg, err := pollForCompletion(context.Background(), env.Kubernetes, "default", created.Name, time.Minute)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				for _, substr := range tt.errorContains {
					if !strings.Contains(err.Error(), substr) {
						t.Fatalf("expected error to contain %q, got: %v", substr, err)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectTerminated {
				if terminated == nil {
					t.Fatalf("expected terminated state")
				}
				if terminated.ExitCode != tt.expectedExitCode {
					t.Fatalf("expected exit code %d, got %d", tt.expectedExitCode, terminated.ExitCode)
				}
				if tt.expectedReason != "" && terminated.Reason != tt.expectedReason {
					t.Fatalf("expected reason %q, got %q", tt.expectedReason, terminated.Reason)
				}
				if lastPod == nil {
					t.Fatalf("expected lastPod to be set")
				}
			}

			if tt.waitingReason == "" && waitMsg != "" {
				t.Fatalf("expected no wait message, got %q", waitMsg)
			}
		})
	}
}

func TestRetrieveLogs(t *testing.T) {
	env := NewNodeDebugTestEnv(t)
	env.Pods.Logs = "  some output with whitespace  \n"

	created, _ := createDebugPod(context.Background(), env.Kubernetes, "node-1", "default", DefaultNodeDebugImage, []string{"echo", "test"})

	logs, err := retrieveLogs(context.Background(), env.Kubernetes, "default", created.Name)
	if err != nil {
		t.Fatalf("retrieveLogs failed: %v", err)
	}
	if logs != "some output with whitespace" {
		t.Fatalf("expected trimmed logs, got %q", logs)
	}
}

func TestProcessResults(t *testing.T) {
	tests := []struct {
		name           string
		terminated     *corev1.ContainerStateTerminated
		pod            *corev1.Pod
		waitMsg        string
		logs           string
		expectError    bool
		errorContains  []string
		expectLogs     bool
		expectedResult string
	}{
		{
			name: "successful completion",
			terminated: &corev1.ContainerStateTerminated{
				ExitCode: 0,
			},
			logs:           "success output",
			expectError:    false,
			expectLogs:     true,
			expectedResult: "success output",
		},
		{
			name: "non-zero exit code with logs",
			terminated: &corev1.ContainerStateTerminated{
				ExitCode: 127,
				Reason:   "CommandNotFound",
				Message:  "command not found",
			},
			logs:           "error logs",
			expectError:    true,
			errorContains:  []string{"127", "CommandNotFound", "command not found", "error logs", "Output:"},
			expectLogs:     false,
			expectedResult: "",
		},
		{
			name: "non-zero exit code without reason or message but with logs",
			terminated: &corev1.ContainerStateTerminated{
				ExitCode: 1,
			},
			logs:           "failed output",
			expectError:    true,
			errorContains:  []string{"command exited with code 1", "failed output", "Output:"},
			expectLogs:     false,
			expectedResult: "",
		},
		{
			name: "non-zero exit code without logs",
			terminated: &corev1.ContainerStateTerminated{
				ExitCode: 1,
			},
			logs:           "",
			expectError:    true,
			errorContains:  []string{"command exited with code 1"},
			expectLogs:     false,
			expectedResult: "",
		},
		{
			name: "pod failed with logs",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Reason: "Evicted",
				},
			},
			logs:           "pod evicted logs",
			expectError:    true,
			errorContains:  []string{"Evicted", "pod evicted logs", "Output:"},
			expectLogs:     false,
			expectedResult: "",
		},
		{
			name: "pod failed without logs",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Reason: "Evicted",
				},
			},
			logs:           "",
			expectError:    true,
			errorContains:  []string{"Evicted"},
			expectLogs:     false,
			expectedResult: "",
		},
		{
			name:           "container waiting with logs",
			waitMsg:        "container waiting: ImagePullBackOff",
			logs:           "waiting logs",
			expectError:    true,
			errorContains:  []string{"did not complete", "waiting logs", "Output:"},
			expectLogs:     false,
			expectedResult: "",
		},
		{
			name:           "container waiting without logs",
			waitMsg:        "container waiting: ImagePullBackOff",
			logs:           "",
			expectError:    true,
			errorContains:  []string{"did not complete"},
			expectLogs:     false,
			expectedResult: "",
		},
		{
			name:           "no terminal state with logs",
			logs:           "incomplete logs",
			expectError:    true,
			errorContains:  []string{"did not reach a terminal state", "incomplete logs", "Output:"},
			expectLogs:     false,
			expectedResult: "",
		},
		{
			name:           "no terminal state without logs",
			logs:           "",
			expectError:    true,
			errorContains:  []string{"did not reach a terminal state"},
			expectLogs:     false,
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processResults(tt.terminated, tt.pod, tt.waitMsg, tt.logs)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				for _, substr := range tt.errorContains {
					if !strings.Contains(err.Error(), substr) {
						t.Fatalf("expected error to contain %q, got: %v", substr, err)
					}
				}
				// Verify logs are NOT in the result when there's an error
				if result != "" {
					t.Fatalf("expected empty result on error, got %q", result)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				// Verify logs ARE in the result when successful
				if result != tt.expectedResult {
					t.Fatalf("expected result %q, got %q", tt.expectedResult, result)
				}
			}
		})
	}
}

func TestSanitizeForName(t *testing.T) {
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
		t.Run(fmt.Sprintf("sanitize(%q)", tt.input), func(t *testing.T) {
			result := sanitizeForName(tt.input)
			if result != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
