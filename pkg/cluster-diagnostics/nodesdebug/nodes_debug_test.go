package nodesdebug

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

func (s *NodesDebugSuite) nodeDebug(env *NodeDebugTestEnv) *NodeDebug {
	return &NodeDebug{NodeDebugClient: env.Client}
}

func (s *NodesDebugSuite) TestNodesDebugExecCreatesPrivilegedPod() {
	env := NewNodeDebugTestEnv(s.T())
	env.Pods.Running = true
	env.Pods.ExecStdout = "kernel 6.8"
	command := []string{"uname", "-a"}

	nd := s.nodeDebug(env)
	stdout, _, err := nd.NodesDebugExec(context.Background(), "", "worker-0", "", command, "", "", 2*time.Minute)
	s.Run("returns stdout on success for corresponding command", func() {
		s.Require().NoError(err)
		s.Equal(command, env.Pods.Command)
		s.Equal("kernel 6.8", stdout)
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
		s.Equal([]string{"sleep", "infinity"}, container.Command)
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
		s.Equal("/", created.Spec.Volumes[0].HostPath.Path)
		s.Equal(corev1.HostPathDirectory, *created.Spec.Volumes[0].HostPath.Type)
	})
}

func (s *NodesDebugSuite) TestNodesDebugExecReturnsErrorForNonZeroExit() {
	env := NewNodeDebugTestEnv(s.T())
	env.Pods.Running = true
	env.Pods.ExecError = fmt.Errorf("execution failed")
	env.Pods.Logs = "bad things happened"

	nd := s.nodeDebug(env)
	stdout, _, err := nd.NodesDebugExec(context.Background(), "debug-ns", "infra-node", "registry.example/custom:latest", []string{"journalctl", "-xe"}, "", "", time.Minute)

	s.Run("returns error with logs included", func() {
		s.Require().Error(err)
		s.Contains(err.Error(), "bad things happened")
		s.Contains(err.Error(), "execution failed")
	})
	s.Run("returns empty output on error", func() {
		s.Empty(stdout)
	})
	s.Run("user provided namespace and image", func() {
		s.Require().NotNil(env.Pods.Created)
		s.Equal("debug-ns", env.Pods.Created.Namespace)
		s.Equal("registry.example/custom:latest", env.Pods.Created.Spec.Containers[0].Image)
	})
}

func (s *NodesDebugSuite) TestNodesDebugExecReturnsErrorWhenNodeDoesNotExist() {
	podCreated := false
	env := NewNodeDebugTestEnv(s.T())
	env.Client.NodeExists = false

	nd := s.nodeDebug(env)
	_, _, err := nd.NodesDebugExec(context.Background(), "default", "missing-node", "", []string{"uname"}, "", "", time.Minute)
	s.Require().Error(err)
	s.Contains(err.Error(), "node missing-node does not exist")
	s.False(podCreated, "debug pod should not be created when node does not exist")
}

func (s *NodesDebugSuite) TestNodesDebugExecReturnsErrorWhenNodeLookupFails() {
	podCreated := false
	env := NewNodeDebugTestEnv(s.T())
	env.Client.NodeExistsError = fmt.Errorf("forbidden")

	nd := s.nodeDebug(env)
	_, _, err := nd.NodesDebugExec(context.Background(), "default", "worker-0", "", []string{"uname"}, "", "", time.Minute)
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to check if node worker-0 exists")
	s.False(podCreated, "debug pod should not be created when node lookup fails")
}

func (s *NodesDebugSuite) TestNodesDebugExecRequiresNodeAndCommand() {
	env := NewNodeDebugTestEnv(s.T())
	nd := s.nodeDebug(env)

	s.Run("returns error when node name is empty", func() {
		_, _, err := nd.NodesDebugExec(context.Background(), "", "", "", []string{"uname"}, "", "", time.Minute)
		s.Require().Error(err)
		s.Contains(err.Error(), "node name is required")
	})

	s.Run("returns error when command is empty", func() {
		_, _, err := nd.NodesDebugExec(context.Background(), "", "worker-0", "", nil, "", "", time.Minute)
		s.Require().Error(err)
		s.Contains(err.Error(), "command is required")
	})
}

func (s *NodesDebugSuite) TestNodesDebugExecCustomHostAndMountPaths() {
	tests := []struct {
		name              string
		hostPath          string
		mountPath         string
		expectedHostPath  string
		expectedMountPath string
	}{
		{
			name:              "empty host path and mount path",
			hostPath:          "",
			mountPath:         "",
			expectedHostPath:  "/",
			expectedMountPath: "/host",
		},
		{
			name:              "empty host path and non-empty mount path",
			hostPath:          "",
			mountPath:         "/custom/mount",
			expectedHostPath:  "/",
			expectedMountPath: "/custom/mount",
		},
		{
			name:              "non-empty host path and empty mount path",
			hostPath:          "/custom/host",
			mountPath:         "",
			expectedHostPath:  "/custom/host",
			expectedMountPath: "/host",
		},
		{
			name:              "non-empty host path and mount path",
			hostPath:          "/custom/host",
			mountPath:         "/custom/mount",
			expectedHostPath:  "/custom/host",
			expectedMountPath: "/custom/mount",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			env := NewNodeDebugTestEnv(s.T())
			env.Pods.Running = true
			env.Pods.ExecStdout = "ok"

			nd := s.nodeDebug(env)
			_, _, err := nd.NodesDebugExec(context.Background(), "test-ns", "worker-1", "custom:v1", []string{"echo", "ok"}, tt.hostPath, tt.mountPath, time.Minute)
			s.Require().NoError(err)

			created := env.Pods.Created
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
			s.Run("uses specified image", func() {
				s.Require().Len(created.Spec.Containers, 1)
				s.Equal("custom:v1", created.Spec.Containers[0].Image)
				s.Equal([]string{"sleep", "infinity"}, created.Spec.Containers[0].Command)
				s.Require().NotNil(created.Spec.Containers[0].SecurityContext)
				s.True(*created.Spec.Containers[0].SecurityContext.Privileged)
			})
			s.Run("mounts host volume", func() {
				s.Require().Len(created.Spec.Volumes, 1)
				s.Require().NotNil(created.Spec.Volumes[0].HostPath)
				s.Equal(tt.expectedHostPath, created.Spec.Volumes[0].HostPath.Path)
				s.Equal(corev1.HostPathDirectory, *created.Spec.Volumes[0].HostPath.Type)
			})
			s.Run("mounts mount path", func() {
				s.Require().Len(created.Spec.Containers, 1)
				s.Require().NotNil(created.Spec.Containers[0].VolumeMounts)
				s.Equal(tt.expectedMountPath, created.Spec.Containers[0].VolumeMounts[0].MountPath)
			})
		})
	}
}

func (s *NodesDebugSuite) TestNodesDebugExecFailsWhenPodDoesNotBecomeRunning() {
	env := NewNodeDebugTestEnv(s.T())
	env.Pods.Running = false

	nd := s.nodeDebug(env)
	_, _, err := nd.NodesDebugExec(context.Background(), "default", "node-1", DefaultNodeDebugImage, []string{"uname"}, "", "", 10*time.Second)
	s.Require().Error(err)
}

func (s *NodesDebugSuite) TestNodesDebugExecRejectsInvalidPaths() {
	tests := []struct {
		name      string
		hostPath  string
		mountPath string
		wantInErr string
	}{
		{name: "host path traversal", hostPath: "..", mountPath: "", wantInErr: "invalid hostPath"},
		{name: "mount path traversal", hostPath: "", mountPath: "..", wantInErr: "invalid mountPath"},
		{name: "relative host path", hostPath: "relative/path", mountPath: "", wantInErr: "invalid hostPath"},
		{name: "relative mount path", hostPath: "", mountPath: "relative/path", wantInErr: "invalid mountPath"},
		{name: "null byte in host path", hostPath: "\x00", mountPath: "", wantInErr: "invalid hostPath"},
		{name: "control character in mount path", hostPath: "", mountPath: "\x01", wantInErr: "invalid mountPath"},
		{name: "shell special in host path", hostPath: "!", mountPath: "", wantInErr: "invalid hostPath"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			env := NewNodeDebugTestEnv(s.T())
			nd := s.nodeDebug(env)
			_, _, err := nd.NodesDebugExec(context.Background(), "default", "worker-0", "", []string{"uname"}, tt.hostPath, tt.mountPath, time.Minute)
			s.Require().Error(err)
			s.Contains(err.Error(), tt.wantInErr)
			s.Nil(env.Pods.Created, "pod should not be created when path validation fails")
		})
	}
}

func (s *NodesDebugSuite) TestNodesDebugExecSanitizesNodeNameInPodName() {
	tests := []struct {
		nodeName       string
		expectedPrefix string
	}{
		{"worker-0", "node-debug-worker-0-"},
		{"WORKER-0", "node-debug-worker-0-"},
		{"worker.0", "node-debug-worker-0-"},
		{"worker_0", "node-debug-worker-0-"},
		{"ip-10-0-1-42.ec2.internal", "node-debug-ip-10-0-1-42-ec2-internal-"},
		{"Worker-Node_123.domain", "node-debug-worker-node-123-domain-"},
	}

	for _, tt := range tests {
		s.Run(fmt.Sprintf("node %q", tt.nodeName), func() {
			env := NewNodeDebugTestEnv(s.T())
			env.Pods.Running = true
			env.Pods.ExecStdout = "ok"

			nd := s.nodeDebug(env)
			_, _, err := nd.NodesDebugExec(context.Background(), "default", tt.nodeName, DefaultNodeDebugImage, []string{"echo"}, "", "", time.Minute)
			s.Require().NoError(err)
			s.Require().NotNil(env.Pods.Created)
			s.True(strings.HasPrefix(env.Pods.Created.Name, tt.expectedPrefix))
			s.LessOrEqual(len(env.Pods.Created.Name), 63)
		})
	}
}

func TestNodesDebug(t *testing.T) {
	suite.Run(t, new(NodesDebugSuite))
}
