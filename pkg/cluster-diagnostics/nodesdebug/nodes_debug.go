package nodesdebug

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/containers/kubernetes-mcp-server/pkg/version"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/klog/v2"

	"github.com/containers/kubernetes-mcp-server/pkg/klogutil"
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

var (
	// pathUnsafeChar matches the first rune not allowed in a mount path (alphanumeric, /, -, _, ., ~).
	pathUnsafeChar = regexp.MustCompile(`[^a-zA-Z0-9/_.~-]`)
)

// NodeDebugClient defines the minimal interface for node debug operations.
// This allows for easier testing and decoupling from the concrete kubernetes client.
type NodeDebugClient interface {
	NamespaceOrDefault(namespace string) string
	Pods(namespace string) corev1client.PodInterface
	PodsLog(ctx context.Context, namespace, name, container string, previous bool, tail int64) (string, error)
	PodsExec(ctx context.Context, namespace, name, container string, command []string) (string, string, error)
	DoesNodeExist(ctx context.Context, name string) (bool, error)
}

// NodeDebug wraps a NodeDebugClient and provides a more user-friendly interface.
type NodeDebug struct {
	NodeDebugClient
}

// NewNodeDebug creates a new NodeDebug client from an api.KubernetesClient.
func NewNodeDebug(k api.KubernetesClient) *NodeDebug {
	return &NodeDebug{NodeDebugClient: NewNodeDebugClient(k)}
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

func (a *nodeDebugAdapter) PodsExec(ctx context.Context, namespace, name, container string, command []string) (string, string, error) {
	return kubernetes.NewCore(a.k).PodsExec(ctx, namespace, name, container, command)
}

func (a *nodeDebugAdapter) DoesNodeExist(ctx context.Context, name string) (bool, error) {
	_, err := kubernetes.NewCore(a.k).ResourcesGet(ctx, &schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Node"}, "", name)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if node %s exists: %w", name, err)
	}
	return true, nil
}

// NodesDebugExec mimics `oc debug node/<name> -- <command...>` by creating a privileged pod on the target
// node, running the provided command, collecting its output, and removing the pod afterwards.
// The host filesystem is mounted at /host, allowing commands to chroot /host if needed to access node resources.
//
// When namespace is empty, the configured namespace (or "default" if none) is used. When image is empty the
// default debug image is used. Timeout controls how long we wait for the pod to complete.
func (n *NodeDebug) NodesDebugExec(
	ctx context.Context,
	namespace string,
	nodeName string,
	image string,
	command []string,
	hostPath string,
	mountPath string,
	timeout time.Duration,
) (string, string, error) {
	if nodeName == "" {
		return "", "", errors.New("node name is required")
	}
	if len(command) == 0 {
		return "", "", errors.New("command is required")
	}

	if err := validatePath(hostPath, "hostPath"); err != nil {
		return "", "", fmt.Errorf("invalid hostPath: %w", err)
	}
	if err := validatePath(mountPath, "mountPath"); err != nil {
		return "", "", fmt.Errorf("invalid mountPath: %w", err)
	}

	ns := n.NamespaceOrDefault(namespace)
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

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	exists, err := n.DoesNodeExist(timeoutCtx, nodeName)
	if err != nil {
		return "", "", fmt.Errorf("failed to check if node %s exists: %w", nodeName, err)
	}
	if !exists {
		return "", "", fmt.Errorf("node %s does not exist", nodeName)
	}

	// Create the debug pod
	created, err := n.createDebugPod(timeoutCtx, nodeName, ns, debugImage, hostPath, mountPath)
	if err != nil {
		return "", "", err
	}

	// Capture logger before defer — ctx may be cancelled when the defer runs
	logger := klog.FromContext(ctx)

	// Ensure the pod is deleted regardless of completion state.
	defer func() {
		deleteCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		deleteErr := n.Pods(ns).Delete(deleteCtx, created.Name, metav1.DeleteOptions{})
		if deleteErr != nil {
			klogutil.LogWarn(logger, "Failed to delete debug pod", klogutil.Field("pod", created.Name), klogutil.Err(deleteErr))
		}
	}()

	// Poll for debug pod running
	pollErr := n.pollForRunning(timeoutCtx, ns, created.Name)
	if pollErr != nil {
		return "", "", pollErr
	}

	stdout, stderr, execErr := n.PodsExec(timeoutCtx, ns, created.Name, NodeDebugContainerName, command)
	if execErr != nil {
		// Retrieve logs on exec errors — the pod may have produced partial output.
		logs, logsErr := n.retrieveLogs(context.Background(), ns, created.Name)
		if logsErr != nil {
			klogutil.LogWarn(logger, "Failed to retrieve logs from debug pod", klogutil.Field("pod", created.Name), klogutil.Field("namespace", ns), klogutil.Err(logsErr))
		} else if logs != "" {
			return stdout, stderr, fmt.Errorf("failed to execute command in debug pod %s in namespace %s: %w\nlogs:\n%s", created.Name, ns, execErr, logs)
		}
		return stdout, stderr, fmt.Errorf("failed to execute command in debug pod %s in namespace %s: %w", created.Name, ns, execErr)
	}

	return stdout, stderr, nil
}

// createDebugPod creates a privileged pod on the target node to run debug commands.
func (n *NodeDebug) createDebugPod(
	ctx context.Context,
	nodeName string,
	namespace string,
	image string,
	hostPath string,
	mountPath string,
) (*corev1.Pod, error) {
	sanitizedNode := sanitizeForName(nodeName)
	hostPathType := corev1.HostPathDirectory
	sleepCommand := []string{"sleep", "infinity"}

	suffix := rand.String(5)
	maxNodeLen := 63 - len("node-debug-") - 1 - len(suffix)
	if maxNodeLen < 1 {
		maxNodeLen = 1
	}
	if len(sanitizedNode) > maxNodeLen {
		sanitizedNode = strings.TrimRight(sanitizedNode[:maxNodeLen], "-")
	}
	podName := fmt.Sprintf("node-debug-%s-%s", sanitizedNode, suffix)

	if hostPath == "" {
		hostPath = "/"
	}
	if mountPath == "" {
		mountPath = "/host"
	}

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
			},
			Volumes: []corev1.Volume{
				{
					Name: "host-root",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: hostPath,
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
					Command:         sleepCommand,
					SecurityContext: &corev1.SecurityContext{
						Privileged: ptr.To(true),
						RunAsUser:  ptr.To[int64](0),
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "host-root", MountPath: mountPath},
					},
				},
			},
		},
	}

	created, err := n.Pods(namespace).Create(ctx, debugPod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create debug pod: %w", err)
	}

	return created, nil
}

// pollForRunning polls the debug pod until it becomes running or the context is cancelled.
func (n *NodeDebug) pollForRunning(ctx context.Context, namespace string, podName string) error {
	// Poll until the pod is running or the context is cancelled.
	err := wait.PollUntilContextCancel(ctx, 2*time.Second, true, func(ctx context.Context) (bool, error) {
		pod, err := n.Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to get debug pod %s status: %w", podName, err)
		}

		switch pod.Status.Phase {
		case corev1.PodFailed:
			// Exit early if pod has failed.
			return false, fmt.Errorf("debug pod %s failed: %s", podName, pod.Status.Message)
		case corev1.PodRunning:
			// Return true if the pod is running.
			return true, nil
		default:
			// Continue polling if the pod is not running or failed.
			return false, nil
		}
	})
	return err
}

// retrieveLogs retrieves the logs from the debug pod.
func (n *NodeDebug) retrieveLogs(ctx context.Context, namespace, podName string) (string, error) {
	logCtx, logCancel := context.WithTimeout(ctx, 30*time.Second)
	defer logCancel()
	logs, logErr := n.PodsLog(logCtx, namespace, podName, NodeDebugContainerName, false, 0)
	if logErr != nil {
		return "", fmt.Errorf("failed to retrieve debug pod logs: %w", logErr)
	}
	return strings.TrimSpace(logs), nil
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

// validatePath validates that a path is safe to use as a filesystem path.
// It ensures the path:
// - Is absolute (starts with /)
// - Does not contain path traversal patterns (..)
// - Contains only safe characters
func validatePath(path, pathType string) error {
	if path == "" {
		return nil
	}

	// Ensure path is absolute
	if !filepath.IsAbs(path) {
		return fmt.Errorf("%s must be an absolute path (start with /), got: %s", pathType, path)
	}

	// Check for path traversal patterns: reject any path element that is exactly ".."
	// Use "/" explicitly since Kubernetes paths are always Unix-style
	if slices.Contains(strings.Split(path, "/"), "..") {
		return fmt.Errorf("%s contains path traversal element '..': %s", pathType, path)
	}

	// Reject null bytes, control characters, shell specials, and other disallowed runes.
	if loc := pathUnsafeChar.FindStringIndex(path); loc != nil {
		i := loc[0]
		r, size := utf8.DecodeRuneInString(path[i:])
		if r == utf8.RuneError && size <= 1 {
			return fmt.Errorf("%s contains invalid/unsafe byte at position %d: 0x%02X", pathType, i, path[i])
		}
		return fmt.Errorf("%s contains unsafe character at position %d: %c (U+%04X)", pathType, i, r, r)
	}

	return nil
}
