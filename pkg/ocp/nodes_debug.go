package ocp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/containers/kubernetes-mcp-server/pkg/version"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/utils/ptr"
)

const (
	// DefaultNodeDebugImage is the UBI9 toolbox image that provides comprehensive debugging and troubleshooting utilities.
	// This image includes: systemd tools (systemctl, journalctl), networking tools (ss, ip, ping, traceroute, nmap),
	// process tools (ps, top, lsof, strace), file system tools (find, tar, rsync), debugging tools (gdb),
	// and many other utilities commonly needed for node-level debugging and diagnostics.
	DefaultNodeDebugImage = "registry.access.redhat.com/ubi9/toolbox:latest"
	// NodeDebugContainerName is the name used for the debug container, matching 'oc debug node' defaults.
	NodeDebugContainerName = "debug"
	// DefaultNodeDebugTimeout is the maximum time to wait for the debug pod to finish executing.
	DefaultNodeDebugTimeout = 1 * time.Minute
)

// NodeDebugClient defines the minimal interface for node debug operations.
// This allows for easier testing and decoupling from the concrete kubernetes client.
type NodeDebugClient interface {
	NamespaceOrDefault(namespace string) string
	Pods(namespace string) corev1client.PodInterface
	PodsLog(ctx context.Context, namespace, name, container string, previous bool, tail int64) (string, error)
}

// nodeDebugAdapter adapts api.KubernetesClient to implement NodeDebugClient.
type nodeDebugAdapter struct {
	k api.KubernetesClient
}

// NewNodeDebugClient creates a NodeDebugClient from an api.KubernetesClient.
func NewNodeDebugClient(k api.KubernetesClient) NodeDebugClient {
	return &nodeDebugAdapter{k: k}
}

func (a *nodeDebugAdapter) NamespaceOrDefault(namespace string) string {
	return a.k.NamespaceOrDefault(namespace)
}

func (a *nodeDebugAdapter) Pods(namespace string) corev1client.PodInterface {
	return a.k.CoreV1().Pods(namespace)
}

func (a *nodeDebugAdapter) PodsLog(ctx context.Context, namespace, name, container string, previous bool, tail int64) (string, error) {
	return kubernetes.NewCore(a.k).PodsLog(ctx, namespace, name, container, previous, tail)
}

// NodesDebugExec mimics `oc debug node/<name> -- <command...>` by creating a privileged pod on the target
// node, running the provided command, collecting its output, and removing the pod afterwards.
// The host filesystem is mounted at /host, allowing commands to chroot /host if needed to access node resources.
//
// When namespace is empty, the configured namespace (or "default" if none) is used. When image is empty the
// default debug image is used. Timeout controls how long we wait for the pod to complete.
func NodesDebugExec(
	ctx context.Context,
	k NodeDebugClient,
	namespace string,
	nodeName string,
	image string,
	command []string,
	timeout time.Duration,
) (string, error) {
	if nodeName == "" {
		return "", errors.New("node name is required")
	}
	if len(command) == 0 {
		return "", errors.New("command is required")
	}

	ns := k.NamespaceOrDefault(namespace)
	if ns == "" {
		ns = "default"
	}
	debugImage := image
	if debugImage == "" {
		debugImage = DefaultNodeDebugImage
	}
	if timeout <= 0 {
		timeout = DefaultNodeDebugTimeout
	}

	// Create the debug pod
	created, err := createDebugPod(ctx, k, nodeName, ns, debugImage, command)
	if err != nil {
		return "", err
	}

	// Ensure the pod is deleted regardless of completion state.
	defer func() {
		deleteCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = k.Pods(ns).Delete(deleteCtx, created.Name, metav1.DeleteOptions{})
	}()

	// Poll for debug pod completion
	terminated, lastPod, waitMsg, pollErr := pollForCompletion(ctx, k, ns, created.Name, timeout)

	// Retrieve logs even on poll errors (e.g. timeout) — the pod may have produced partial output.
	logs, _ := retrieveLogs(context.Background(), k, ns, created.Name)

	if pollErr != nil {
		if logs != "" {
			return "", fmt.Errorf("%w\nOutput:\n%s", pollErr, logs)
		}
		return "", pollErr
	}

	// Process the results
	return processResults(terminated, lastPod, waitMsg, logs)
}

// createDebugPod creates a privileged pod on the target node to run debug commands.
func createDebugPod(
	ctx context.Context,
	k NodeDebugClient,
	nodeName string,
	namespace string,
	image string,
	command []string,
) (*corev1.Pod, error) {
	sanitizedNode := sanitizeForName(nodeName)
	hostPathType := corev1.HostPathDirectory

	suffix := rand.String(5)
	maxNodeLen := 63 - len("node-debug-") - 1 - len(suffix)
	if maxNodeLen < 1 {
		maxNodeLen = 1
	}
	if len(sanitizedNode) > maxNodeLen {
		sanitizedNode = strings.TrimRight(sanitizedNode[:maxNodeLen], "-")
	}
	podName := fmt.Sprintf("node-debug-%s-%s", sanitizedNode, suffix)

	debugPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			Labels: map[string]string{
				kubernetes.AppKubernetesManagedBy: version.BinaryName,
				kubernetes.AppKubernetesComponent: "node-debug",
				kubernetes.AppKubernetesName:      fmt.Sprintf("node-debug-%s", sanitizedNode),
			},
		},
		Spec: corev1.PodSpec{
			AutomountServiceAccountToken: ptr.To(false),
			HostNetwork:                  true,
			HostPID:                      true,
			HostIPC:                      true,
			NodeName:                     nodeName,
			RestartPolicy:                corev1.RestartPolicyNever,
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser: ptr.To[int64](0),
			},
			Tolerations: []corev1.Toleration{
				{Operator: corev1.TolerationOpExists},
				{Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule},
				{Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoExecute},
			},
			Volumes: []corev1.Volume{
				{
					Name: "host-root",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/",
							Type: &hostPathType,
						},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name:            NodeDebugContainerName,
					Image:           image,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         command,
					SecurityContext: &corev1.SecurityContext{
						Privileged: ptr.To(true),
						RunAsUser:  ptr.To[int64](0),
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "host-root", MountPath: "/host"},
					},
				},
			},
		},
	}

	created, err := k.Pods(namespace).Create(ctx, debugPod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create debug pod: %w", err)
	}

	return created, nil
}

// pollForCompletion polls the debug pod until it completes or times out.
func pollForCompletion(
	ctx context.Context,
	k NodeDebugClient,
	namespace string,
	podName string,
	timeout time.Duration,
) (*corev1.ContainerStateTerminated, *corev1.Pod, string, error) {
	pollCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var (
		lastPod    *corev1.Pod
		terminated *corev1.ContainerStateTerminated
		waitMsg    string
	)

	for {
		current, err := k.Pods(namespace).Get(pollCtx, podName, metav1.GetOptions{})
		if err != nil {
			return nil, nil, "", fmt.Errorf("failed to get debug pod status: %w", err)
		}
		lastPod = current

		if status := containerStatusByName(current.Status.ContainerStatuses, NodeDebugContainerName); status != nil {
			if status.State.Waiting != nil {
				waitMsg = fmt.Sprintf("container waiting: %s", status.State.Waiting.Reason)
				// Image pull issues should fail fast.
				if status.State.Waiting.Reason == "ErrImagePull" || status.State.Waiting.Reason == "ImagePullBackOff" {
					return nil, nil, "", fmt.Errorf("debug container failed to start (%s): %s", status.State.Waiting.Reason, status.State.Waiting.Message)
				}
			}
			if status.State.Terminated != nil {
				terminated = status.State.Terminated
				break
			}
		}

		if current.Status.Phase == corev1.PodFailed {
			break
		}

		// Wait for the next tick interval before checking pod status again, or timeout if context is done.
		select {
		case <-pollCtx.Done():
			return nil, lastPod, waitMsg, fmt.Errorf("timed out waiting for debug pod %s to complete: %w", podName, pollCtx.Err())
		case <-ticker.C:
		}
	}

	return terminated, lastPod, waitMsg, nil
}

// retrieveLogs retrieves the logs from the debug pod.
func retrieveLogs(ctx context.Context, k NodeDebugClient, namespace, podName string) (string, error) {
	logCtx, logCancel := context.WithTimeout(ctx, 30*time.Second)
	defer logCancel()
	logs, logErr := k.PodsLog(logCtx, namespace, podName, NodeDebugContainerName, false, 0)
	if logErr != nil {
		return "", fmt.Errorf("failed to retrieve debug pod logs: %w", logErr)
	}
	return strings.TrimSpace(logs), nil
}

// processResults processes the debug pod completion status and returns the appropriate result.
func processResults(terminated *corev1.ContainerStateTerminated, lastPod *corev1.Pod, waitMsg, logs string) (string, error) {
	if terminated != nil {
		if terminated.ExitCode != 0 {
			errMsg := fmt.Sprintf("command exited with code %d", terminated.ExitCode)
			if terminated.Reason != "" {
				errMsg = fmt.Sprintf("%s (%s)", errMsg, terminated.Reason)
			}
			if terminated.Message != "" {
				errMsg = fmt.Sprintf("%s: %s", errMsg, terminated.Message)
			}
			if logs != "" {
				errMsg = fmt.Sprintf("%s\nOutput:\n%s", errMsg, logs)
			}
			return "", errors.New(errMsg)
		}
		return logs, nil
	}

	if lastPod != nil && lastPod.Status.Reason != "" {
		if logs != "" {
			return "", fmt.Errorf("debug pod failed: %s\nOutput:\n%s", lastPod.Status.Reason, logs)
		}
		return "", fmt.Errorf("debug pod failed: %s", lastPod.Status.Reason)
	}
	if waitMsg != "" {
		if logs != "" {
			return "", fmt.Errorf("debug container did not complete: %s\nOutput:\n%s", waitMsg, logs)
		}
		return "", fmt.Errorf("debug container did not complete: %s", waitMsg)
	}
	if logs != "" {
		return "", fmt.Errorf("debug container did not reach a terminal state\nOutput:\n%s", logs)
	}
	return "", errors.New("debug container did not reach a terminal state")
}

func sanitizeForName(name string) string {
	lower := strings.ToLower(name)
	var b strings.Builder
	b.Grow(len(lower))
	for _, r := range lower {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
			continue
		}
		b.WriteRune('-')
	}
	sanitized := strings.Trim(b.String(), "-")
	if sanitized == "" {
		sanitized = "node"
	}
	if len(sanitized) > 40 {
		sanitized = sanitized[:40]
	}
	return sanitized
}

func containerStatusByName(statuses []corev1.ContainerStatus, name string) *corev1.ContainerStatus {
	for idx := range statuses {
		if statuses[idx].Name == name {
			return &statuses[idx]
		}
	}
	return nil
}
