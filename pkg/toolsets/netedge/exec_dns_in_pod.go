package netedge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/google/jsonschema-go/jsonschema"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
)

const (
	dnsProbeImage      = "registry.redhat.io/openshift4/network-tools-rhel9"
	dnsProbePodPrefix  = "mcp-dns-probe-"
	dnsProbePollPeriod = 2 * time.Second
	dnsProbeTimeout    = 120 * time.Second
)

var supportedRecordTypes = map[string]bool{
	"A":     true,
	"AAAA":  true,
	"ANY":   true,
	"CNAME": true,
	"MX":    true,
	"NS":    true,
	"PTR":   true,
	"SOA":   true,
	"SRV":   true,
	"TXT":   true,
}

// podExecutor interface abstracts pod lifecycle operations for testability.
type podExecutor interface {
	CreatePod(ctx context.Context, namespace string, pod *corev1.Pod) (*corev1.Pod, error)
	WaitForPod(ctx context.Context, namespace, name string, timeout time.Duration) (*corev1.Pod, error)
	GetPodLogs(ctx context.Context, namespace, name string) (string, error)
	DeletePod(ctx context.Context, namespace, name string) error
}

// defaultPodExecutor wraps a kubernetes.Interface for real cluster operations.
type defaultPodExecutor struct {
	client kubernetes.Interface
}

func (d *defaultPodExecutor) CreatePod(ctx context.Context, namespace string, pod *corev1.Pod) (*corev1.Pod, error) {
	return d.client.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
}

func (d *defaultPodExecutor) WaitForPod(ctx context.Context, namespace, name string, timeout time.Duration) (*corev1.Pod, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(dnsProbePollPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out waiting for pod %s/%s to complete", namespace, name)
		case <-ticker.C:
			pod, err := d.client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to get pod status: %w", err)
			}
			if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
				return pod, nil
			}
		}
	}
}

func (d *defaultPodExecutor) GetPodLogs(ctx context.Context, namespace, name string) (string, error) {
	req := d.client.CoreV1().Pods(namespace).GetLogs(name, &corev1.PodLogOptions{})
	stream, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get pod logs: %w", err)
	}
	defer stream.Close() //nolint:errcheck

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, stream); err != nil {
		return "", fmt.Errorf("failed to read pod logs: %w", err)
	}
	return buf.String(), nil
}

func (d *defaultPodExecutor) DeletePod(ctx context.Context, namespace, name string) error {
	return d.client.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// ExecDNSResult represents the structured JSON response for exec_dns_in_pod.
type ExecDNSResult struct {
	PodName string `json:"pod_name"`
	Output  string `json:"output"`
	Phase   string `json:"phase"`
}

func initExecDNSInPod() []api.ServerTool {
	return initExecDNSInPodWith(nil)
}

// initExecDNSInPodWith creates exec_dns_in_pod tools using the provided podExecutor.
// If executor is nil, a defaultPodExecutor is created at handler call-time from the KubernetesClient.
// Pass a mock executor in tests.
func initExecDNSInPodWith(executor podExecutor) []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "exec_dns_in_pod",
				Description: "Spin up a temporary pod in the cluster to execute a DNS lookup using dig, verifying internal cluster networking and DNS path.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Namespace to run the ephemeral pod in.",
						},
						"target_server": {
							Type:        "string",
							Description: "DNS server IP to query (e.g. 172.30.0.10).",
						},
						"target_name": {
							Type:        "string",
							Description: "DNS name to query (e.g. kubernetes.default.svc.cluster.local).",
						},
						"record_type": {
							Type:        "string",
							Description: "DNS record type (A, AAAA, etc.). Defaults to A.",
							Default:     json.RawMessage(`"A"`),
						},
					},
					Required: []string{"namespace", "target_server", "target_name"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Exec DNS in Pod",
					ReadOnlyHint:    ptr.To(false),
					DestructiveHint: ptr.To(false),
					IdempotentHint:  ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: makeExecDNSInPodHandler(executor),
		},
	}
}

func makeExecDNSInPodHandler(executor podExecutor) api.ToolHandlerFunc {
	return func(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
		namespace, ok := params.GetArguments()["namespace"].(string)
		if !ok || namespace == "" {
			return api.NewToolCallResultStructured(nil, fmt.Errorf("namespace parameter is required")), nil
		}

		targetServer, ok := params.GetArguments()["target_server"].(string)
		if !ok || targetServer == "" {
			return api.NewToolCallResultStructured(nil, fmt.Errorf("target_server parameter is required")), nil
		}
		if net.ParseIP(targetServer) == nil {
			return api.NewToolCallResultStructured(nil, fmt.Errorf("target_server must be a valid IP address")), nil
		}

		targetName, ok := params.GetArguments()["target_name"].(string)
		if !ok || targetName == "" {
			return api.NewToolCallResultStructured(nil, fmt.Errorf("target_name parameter is required")), nil
		}
		if strings.ContainsAny(targetName, " \t\r\n") || strings.HasPrefix(targetName, "-") || strings.HasPrefix(targetName, "+") || strings.HasPrefix(targetName, "@") {
			return api.NewToolCallResultStructured(nil, fmt.Errorf("target_name must be a DNS name, not a dig option")), nil
		}

		recordType, ok := params.GetArguments()["record_type"].(string)
		if !ok || recordType == "" {
			recordType = "A"
		}
		recordType = strings.ToUpper(recordType)
		if !supportedRecordTypes[recordType] {
			return api.NewToolCallResultStructured(nil, fmt.Errorf("unsupported record_type: %s", recordType)), nil
		}

		// Use provided executor or create one from the KubernetesClient
		exec := executor
		if exec == nil {
			exec = &defaultPodExecutor{client: params.KubernetesClient}
		}

		podName := dnsProbePodPrefix + rand.String(6)

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: namespace,
			},
			Spec: corev1.PodSpec{
				AutomountServiceAccountToken: ptr.To(false),
				RestartPolicy:                corev1.RestartPolicyNever,
				Containers: []corev1.Container{
					{
						Name:    "dns-probe",
						Image:   dnsProbeImage,
						Command: []string{"/usr/bin/dig", fmt.Sprintf("@%s", targetServer), targetName, recordType},
						SecurityContext: &corev1.SecurityContext{
							RunAsNonRoot:             ptr.To(true),
							AllowPrivilegeEscalation: ptr.To(false),
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{"ALL"},
							},
							SeccompProfile: &corev1.SeccompProfile{
								Type: corev1.SeccompProfileTypeRuntimeDefault,
							},
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("50m"),
								corev1.ResourceMemory: resource.MustParse("64Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("100m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
					},
				},
			},
		}

		createdPod, err := exec.CreatePod(params.Context, namespace, pod)
		if err != nil {
			return api.NewToolCallResultStructured(nil, fmt.Errorf("failed to create DNS probe pod: %w", err)), nil
		}
		podName = createdPod.Name

		// Always attempt cleanup
		defer func() {
			// Use a background context for cleanup since the original context may be cancelled
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cleanupCancel()
			_ = exec.DeletePod(cleanupCtx, namespace, podName)
		}()

		completedPod, err := exec.WaitForPod(params.Context, namespace, podName, dnsProbeTimeout)
		if err != nil {
			return api.NewToolCallResultStructured(nil, fmt.Errorf("error waiting for DNS probe pod: %w", err)), nil
		}

		phase := string(completedPod.Status.Phase)

		logs, err := exec.GetPodLogs(params.Context, namespace, podName)
		if err != nil {
			// Return what we have even if log retrieval fails
			result := ExecDNSResult{
				PodName: podName,
				Output:  fmt.Sprintf("failed to retrieve pod logs: %v", err),
				Phase:   phase,
			}
			return api.NewToolCallResultStructured(result, nil), nil
		}

		result := ExecDNSResult{
			PodName: podName,
			Output:  logs,
			Phase:   phase,
		}

		return api.NewToolCallResultStructured(result, nil), nil
	}
}
