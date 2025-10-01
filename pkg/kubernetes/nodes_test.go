package kubernetes

import (
	"context"
	"testing"
	"time"
)

func TestNodesDebugExecCreatesPrivilegedChrootPod(t *testing.T) {
	env := NewNodeDebugTestEnv(t)
	env.Pods.Logs = "kernel 6.8"

	out, err := env.Kubernetes.NodesDebugExec(context.Background(), "", "worker-0", "", []string{"uname", "-a"}, 2*time.Minute)
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
	if container.Image != defaultNodeDebugImage {
		t.Fatalf("expected default image %q, got %q", defaultNodeDebugImage, container.Image)
	}
	expectedCommand := []string{"chroot", "/host", "uname", "-a"}
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

	if len(created.Spec.Volumes) != 1 || created.Spec.Volumes[0].VolumeSource.HostPath == nil {
		t.Fatalf("expected host root volume to be configured")
	}
}

func TestNodesDebugExecReturnsErrorForNonZeroExit(t *testing.T) {
	env := NewNodeDebugTestEnv(t)
	env.Pods.ExitCode = 5
	env.Pods.TerminatedReason = "Error"
	env.Pods.TerminatedMessage = "some failure"
	env.Pods.Logs = "bad things happened"

	out, err := env.Kubernetes.NodesDebugExec(context.Background(), "debug-ns", "infra-node", "registry.example/custom:latest", []string{"journalctl", "-xe"}, time.Minute)
	if err == nil {
		t.Fatalf("expected error for non-zero exit code")
	}
	if out != "bad things happened" {
		t.Fatalf("expected logs to be returned alongside error, got %q", out)
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
