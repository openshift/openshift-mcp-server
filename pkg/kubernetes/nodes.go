package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/containers/kubernetes-mcp-server/pkg/version"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const (
	// defaultNodeDebugImage is a lightweight image that provides the tooling required to run chroot.
	defaultNodeDebugImage = "quay.io/fedora/fedora:latest"
	// nodeDebugContainerName is the name used for the debug container, matching oc debug defaults.
	nodeDebugContainerName = "debug"
	// defaultNodeDebugTimeout is the maximum time to wait for the debug pod to finish executing.
	defaultNodeDebugTimeout = 5 * time.Minute
)

// NodesDebugExec mimics `oc debug node/<name> -- <command...>` by creating a privileged pod on the target
// node, running the provided command within a chroot of the host filesystem, collecting its output, and
// removing the pod afterwards.
//
// When namespace is empty, the configured namespace (or "default" if none) is used. When image is empty the
// default debug image is used. Timeout controls how long we wait for the pod to complete.
func (k *Kubernetes) NodesDebugExec(
	ctx context.Context,
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
		debugImage = defaultNodeDebugImage
	}
	if timeout <= 0 {
		timeout = defaultNodeDebugTimeout
	}

	podsClient, err := k.podsClient(ns)
	if err != nil {
		return "", fmt.Errorf("failed to get pod client for namespace %s: %w", ns, err)
	}

	sanitizedNode := sanitizeForName(nodeName)
	hostPathType := corev1.HostPathDirectory

	debugPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("node-debug-%s-", sanitizedNode),
			Namespace:    ns,
			Labels: map[string]string{
				AppKubernetesManagedBy: version.BinaryName,
				AppKubernetesComponent: "node-debug",
				AppKubernetesName:      fmt.Sprintf("node-debug-%s", sanitizedNode),
			},
		},
		Spec: corev1.PodSpec{
			AutomountServiceAccountToken: ptr.To(false),
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
					Name:            nodeDebugContainerName,
					Image:           debugImage,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         append([]string{"chroot", "/host"}, command...),
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

	created, err := podsClient.Create(ctx, debugPod, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create debug pod: %w", err)
	}

	// Ensure the pod is deleted regardless of completion state.
	defer func() {
		deleteCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		grace := int64(0)
		_ = podsClient.Delete(deleteCtx, created.Name, metav1.DeleteOptions{GracePeriodSeconds: &grace})
	}()

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
		select {
		case <-pollCtx.Done():
			return "", fmt.Errorf("timed out waiting for debug pod %s to complete: %w", created.Name, pollCtx.Err())
		default:
		}

		current, getErr := podsClient.Get(pollCtx, created.Name, metav1.GetOptions{})
		if getErr != nil {
			return "", fmt.Errorf("failed to get debug pod status: %w", getErr)
		}
		lastPod = current

		if status := containerStatusByName(current.Status.ContainerStatuses, nodeDebugContainerName); status != nil {
			if status.State.Waiting != nil {
				waitMsg = fmt.Sprintf("container waiting: %s", status.State.Waiting.Reason)
				// Image pull issues should fail fast.
				if status.State.Waiting.Reason == "ErrImagePull" || status.State.Waiting.Reason == "ImagePullBackOff" {
					return "", fmt.Errorf("debug container failed to start (%s): %s", status.State.Waiting.Reason, status.State.Waiting.Message)
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

		select {
		case <-pollCtx.Done():
			return "", fmt.Errorf("timed out waiting for debug pod %s to complete: %w", created.Name, pollCtx.Err())
		case <-ticker.C:
		}
	}

	logCtx, logCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer logCancel()
	logs, logErr := k.PodsLog(logCtx, ns, created.Name, nodeDebugContainerName, false, 0)
	if logErr != nil {
		return "", fmt.Errorf("failed to retrieve debug pod logs: %w", logErr)
	}
	logs = strings.TrimSpace(logs)

	if terminated != nil {
		if terminated.ExitCode != 0 {
			errMsg := fmt.Sprintf("command exited with code %d", terminated.ExitCode)
			if terminated.Reason != "" {
				errMsg = fmt.Sprintf("%s (%s)", errMsg, terminated.Reason)
			}
			if terminated.Message != "" {
				errMsg = fmt.Sprintf("%s: %s", errMsg, terminated.Message)
			}
			return logs, errors.New(errMsg)
		}
		return logs, nil
	}

	if lastPod != nil && lastPod.Status.Reason != "" {
		return logs, fmt.Errorf("debug pod failed: %s", lastPod.Status.Reason)
	}
	if waitMsg != "" {
		return logs, fmt.Errorf("debug container did not complete: %s", waitMsg)
	}
	return logs, errors.New("debug container did not reach a terminal state")
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
