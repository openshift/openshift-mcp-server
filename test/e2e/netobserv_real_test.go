//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/containers/kubernetes-mcp-server/internal/test"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

type netobservRealState struct {
	dep       *serverDeployment
	mcpClient *test.McpClient
}

var (
	netobservRealTS          testState[netobservRealState]
	netobservRealManifestDir = getNetobservRealManifestDir()
)

func getNetobservRealManifestDir() string {
	dir, _ := filepath.Abs("testdata/netobserv-real")
	return dir
}

type flowCollectorConfig struct {
	Spec struct {
		Namespace string `json:"namespace"`
		Loki      struct {
			Enable     *bool  `json:"enable"`
			Mode       string `json:"mode"`
			Monolithic struct {
				InstallDemoLoki bool `json:"installDemoLoki"`
			} `json:"monolithic"`
		} `json:"loki"`
	} `json:"spec"`
}

// findNetObservPluginNamespace finds the namespace used by the NetObserv plugin service.
func findNetObservPluginNamespace(ctx context.Context, t *testing.T, clientset kubernetes.Interface) (string, bool) {
	t.Helper()

	// Check common namespaces where NetObserv plugin runs
	namespaces := []string{"netobserv", "openshift-netobserv"}

	for _, ns := range namespaces {
		svc, err := clientset.CoreV1().Services(ns).Get(ctx, "netobserv-plugin", metav1.GetOptions{})
		if err == nil && svc != nil {
			t.Logf("Found NetObserv plugin service in namespace: %s", ns)
			return ns, true
		}
	}

	t.Logf("NetObserv plugin service not found in namespaces: %v", namespaces)
	return "", false
}

func getFlowCollectorConfig(ctx context.Context, t *testing.T, kubeconfig string) (flowCollectorConfig, error) {
	t.Helper()

	cmd := exec.CommandContext(ctx, "kubectl", "get", "flowcollector", "cluster", "-o", "json", "--kubeconfig", kubeconfig)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return flowCollectorConfig{}, fmt.Errorf("get FlowCollector cluster: %w: %s", err, strings.TrimSpace(string(output)))
	}

	var config flowCollectorConfig
	if err := json.Unmarshal(output, &config); err != nil {
		return flowCollectorConfig{}, fmt.Errorf("parse FlowCollector cluster: %w", err)
	}
	return config, nil
}

func requireFlowCollectorDemoLoki(t *testing.T, config flowCollectorConfig) {
	t.Helper()
	require.NotNil(t, config.Spec.Loki.Enable, "FlowCollector spec.loki.enable must be set to true")
	require.True(t, *config.Spec.Loki.Enable, "FlowCollector spec.loki.enable must be true to run real NetObserv tests")
	require.Equal(t, "Monolithic", config.Spec.Loki.Mode, "FlowCollector spec.loki.mode must be Monolithic to run real NetObserv tests")
	require.True(t, config.Spec.Loki.Monolithic.InstallDemoLoki, "FlowCollector spec.loki.monolithic.installDemoLoki must be true to run real NetObserv tests")
}

func netObservPluginNamespace(ctx context.Context, t *testing.T, clientset kubernetes.Interface, config flowCollectorConfig) string {
	t.Helper()
	if config.Spec.Namespace != "" {
		return config.Spec.Namespace
	}
	namespace, found := findNetObservPluginNamespace(ctx, t, clientset)
	require.True(t, found, "FlowCollector spec.namespace is empty and no netobserv-plugin Service was found")
	return namespace
}

// deployNetObservOperator deploys NetObserv operator and FlowCollector
func deployNetObservOperator(ctx context.Context, t *testing.T, kubeconfig string, clientset kubernetes.Interface) string {
	t.Helper()

	operatorNamespace := "openshift-netobserv-operator"
	pluginNamespace := "netobserv"

	// Reuse an existing FlowCollector only when its required Loki-backed setup is usable.
	config, flowCollectorErr := getFlowCollectorConfig(ctx, t, kubeconfig)
	if flowCollectorErr == nil {
		t.Logf("FlowCollector already exists; reusing its configuration")
		require.True(t, checkNetObservOperatorStatus(ctx, t, kubeconfig, clientset, operatorNamespace),
			"an existing FlowCollector was found, but the NetObserv operator is not ready")
		requireFlowCollectorDemoLoki(t, config)
		waitForFlowCollector(ctx, t, kubeconfig)
		pluginNamespace = netObservPluginNamespace(ctx, t, clientset, config)
		waitForLokiReady(ctx, t, kubeconfig, clientset, config, pluginNamespace)
		return pluginNamespace
	}
	require.True(t, isFlowCollectorMissingError(flowCollectorErr), "failed to check for an existing FlowCollector: %v", flowCollectorErr)

	// Check if operator is already deployed
	if checkNetObservOperatorStatus(ctx, t, kubeconfig, clientset, operatorNamespace) {
		t.Logf("NetObserv operator already deployed and ready, skipping operator deployment")

		// No FlowCollector exists, so create the test configuration.
		t.Logf("Creating FlowCollector")
		cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", filepath.Join(netobservRealManifestDir, "flowcollector.yaml"), "--kubeconfig", kubeconfig)
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Failed to create FlowCollector: %s", string(output))
		t.Cleanup(func() {
			cmd := exec.Command("kubectl", "delete", "-f", filepath.Join(netobservRealManifestDir, "flowcollector.yaml"), "--kubeconfig", kubeconfig, "--ignore-not-found")
			_ = cmd.Run()
		})
		config, err := getFlowCollectorConfig(ctx, t, kubeconfig)
		require.NoError(t, err, "Failed to read the FlowCollector created by the test")

		requireFlowCollectorDemoLoki(t, config)
		waitForFlowCollector(ctx, t, kubeconfig)
		pluginNamespace = netObservPluginNamespace(ctx, t, clientset, config)
		waitForLokiReady(ctx, t, kubeconfig, clientset, config, pluginNamespace)
		return pluginNamespace
	}
	t.Cleanup(func() {
		cleanupNetObservOperator(t, kubeconfig)
	})

	// Create CatalogSource (y-stream Konflux catalog)
	t.Logf("Creating CatalogSource: netobserv-konflux-fbc")
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", filepath.Join(netobservRealManifestDir, "operator-catalogsource.yaml"), "--kubeconfig", kubeconfig)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to create CatalogSource: %s", string(output))

	// Wait for CatalogSource pod to be ready
	t.Logf("Waiting for CatalogSource pod to be ready...")
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		pods, err := clientset.CoreV1().Pods("openshift-marketplace").List(ctx, metav1.ListOptions{
			LabelSelector: "olm.catalogSource=netobserv-konflux-fbc",
		})
		if err == nil && len(pods.Items) > 0 {
			allReady := true
			for _, pod := range pods.Items {
				if pod.Status.Phase != "Running" {
					allReady = false
					break
				}
			}
			if allReady {
				t.Logf("CatalogSource pod is ready")
				break
			}
		}
		time.Sleep(5 * time.Second)
	}

	// Konflux bundle images are mirrored from registry.redhat.io to quay.io.
	// Match the operator backend tests by installing the cluster-wide IDMS before
	// subscribing to the operator.
	t.Logf("Creating NetObserv ImageDigestMirrorSet")
	cmd = exec.CommandContext(ctx, "kubectl", "apply", "-f", filepath.Join(netobservRealManifestDir, "operator-idms.yaml"), "--kubeconfig", kubeconfig)
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "Failed to create NetObserv ImageDigestMirrorSet: %s", string(output))

	// Create namespaces
	t.Logf("Creating namespaces")
	cmd = exec.CommandContext(ctx, "kubectl", "apply", "-f", filepath.Join(netobservRealManifestDir, "operator-namespace.yaml"), "--kubeconfig", kubeconfig)
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "Failed to create namespaces: %s", string(output))

	// Create OperatorGroup
	t.Logf("Creating OperatorGroup")
	cmd = exec.CommandContext(ctx, "kubectl", "apply", "-f", filepath.Join(netobservRealManifestDir, "operator-group.yaml"), "--kubeconfig", kubeconfig)
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "Failed to create OperatorGroup: %s", string(output))

	// Create Subscription
	t.Logf("Creating Subscription for netobserv-operator")
	cmd = exec.CommandContext(ctx, "kubectl", "apply", "-f", filepath.Join(netobservRealManifestDir, "operator-subscription.yaml"), "--kubeconfig", kubeconfig)
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "Failed to create Subscription: %s", string(output))

	// Wait for operator pod to be ready
	t.Logf("Waiting for operator pod to be ready...")
	deadline = time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		pods, err := clientset.CoreV1().Pods(operatorNamespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app=netobserv-operator",
		})
		if err == nil && len(pods.Items) > 0 {
			allReady := true
			for _, pod := range pods.Items {
				podReady := false
				for _, condition := range pod.Status.Conditions {
					if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
						podReady = true
						break
					}
				}
				if pod.Status.Phase != corev1.PodRunning || !podReady {
					allReady = false
					break
				}
			}
			if allReady {
				t.Logf("Operator pod is ready")
				break
			}
		}
		time.Sleep(10 * time.Second)
	}

	// Wait for FlowCollector CRD to be available
	t.Logf("Waiting for FlowCollector CRD...")
	deadline = time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		cmd = exec.CommandContext(ctx, "kubectl", "get", "crd", "flowcollectors.flows.netobserv.io", "--kubeconfig", kubeconfig)
		if cmd.Run() == nil {
			t.Logf("FlowCollector CRD is available")
			break
		}
		time.Sleep(5 * time.Second)
	}

	// Deploy FlowCollector
	t.Logf("Creating FlowCollector")
	cmd = exec.CommandContext(ctx, "kubectl", "apply", "-f", filepath.Join(netobservRealManifestDir, "flowcollector.yaml"), "--kubeconfig", kubeconfig)
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "Failed to create FlowCollector: %s", string(output))

	config, err = getFlowCollectorConfig(ctx, t, kubeconfig)
	require.NoError(t, err, "Failed to read the FlowCollector created by the test")
	requireFlowCollectorDemoLoki(t, config)

	// Wait for FlowCollector and Loki to be ready.
	waitForFlowCollector(ctx, t, kubeconfig)
	pluginNamespace = netObservPluginNamespace(ctx, t, clientset, config)
	waitForLokiReady(ctx, t, kubeconfig, clientset, config, pluginNamespace)
	return pluginNamespace
}

// isFlowCollectorMissingError reports whether the cluster FlowCollector or its API type is missing.
func isFlowCollectorMissingError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, `flowcollectors.flows.netobserv.io "cluster" not found`) ||
		strings.Contains(message, `no matches for kind "flowcollector" in version "flows.netobserv.io`) ||
		strings.Contains(message, `the server doesn't have a resource type "flowcollector"`)
}

// checkNetObservOperatorStatus checks if NetObserv operator is already deployed and ready
func checkNetObservOperatorStatus(ctx context.Context, t *testing.T, kubeconfig string, clientset kubernetes.Interface, operatorNamespace string) bool {
	t.Helper()

	// Check if operator namespace exists
	_, err := clientset.CoreV1().Namespaces().Get(ctx, operatorNamespace, metav1.GetOptions{})
	if err != nil {
		t.Logf("Operator namespace %s not found, operator will be deployed", operatorNamespace)
		return false
	}

	// Check if operator pod is running
	pods, err := clientset.CoreV1().Pods(operatorNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=netobserv-operator",
	})
	if err != nil || len(pods.Items) == 0 {
		t.Logf("Operator pod not found, operator will be deployed")
		return false
	}

	// Check if all operator pods are ready
	for _, pod := range pods.Items {
		if pod.Status.Phase != "Running" {
			t.Logf("Operator pod %s not running (phase: %s), operator will be deployed", pod.Name, pod.Status.Phase)
			return false
		}
		ready := false
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
				ready = true
				break
			}
		}
		if !ready {
			t.Logf("Operator pod %s is not Ready, operator will be deployed", pod.Name)
			return false
		}
	}

	// Check if FlowCollector CRD exists
	cmd := exec.CommandContext(ctx, "kubectl", "get", "crd", "flowcollectors.flows.netobserv.io", "--kubeconfig", kubeconfig)
	if cmd.Run() != nil {
		t.Logf("FlowCollector CRD not found, operator will be deployed")
		return false
	}

	t.Logf("NetObserv operator is already deployed and ready")
	return true
}

// waitForFlowCollector waits for the FlowCollector and its managed components to be ready.
func waitForFlowCollector(ctx context.Context, t *testing.T, kubeconfig string) {
	t.Helper()

	t.Logf("Waiting for FlowCollector to become Ready...")
	waitCmd := exec.CommandContext(ctx, "kubectl", "wait", "--for=condition=Ready", "flowcollector/cluster", "--timeout=10m", "--kubeconfig", kubeconfig)
	waitOutput, waitErr := waitCmd.CombinedOutput()
	if waitErr == nil {
		t.Logf("FlowCollector is ready: %s", strings.TrimSpace(string(waitOutput)))
		return
	}

	statusCmd := exec.CommandContext(ctx, "kubectl", "get", "flowcollector", "cluster", "-o", "yaml", "--kubeconfig", kubeconfig)
	statusOutput, statusErr := statusCmd.CombinedOutput()
	if statusErr != nil {
		require.NoError(t, waitErr, "FlowCollector did not become Ready: %s; failed to retrieve FlowCollector status: %v: %s",
			strings.TrimSpace(string(waitOutput)), statusErr, strings.TrimSpace(string(statusOutput)))
		return
	}
	require.NoError(t, waitErr, "FlowCollector did not become Ready: %s\nFlowCollector status:\n%s",
		strings.TrimSpace(string(waitOutput)), strings.TrimSpace(string(statusOutput)))
}

// waitForLokiReady waits until the Monolithic demo Loki service accepts requests.
func waitForLokiReady(ctx context.Context, t *testing.T, kubeconfig string, clientset kubernetes.Interface, config flowCollectorConfig, pluginNamespace string) {
	t.Helper()

	serviceNamespace := config.Spec.Namespace
	if serviceNamespace == "" {
		serviceNamespace = pluginNamespace
	}
	const serviceName = "loki"
	const servicePort = 3100

	restCfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	require.NoError(t, err, "build rest config for Loki readiness check")

	deadline := time.Now().Add(3 * time.Minute)
	var serviceErr error
	for time.Now().Before(deadline) {
		_, serviceErr = clientset.CoreV1().Services(serviceNamespace).Get(ctx, serviceName, metav1.GetOptions{})
		if serviceErr == nil {
			break
		}
		select {
		case <-ctx.Done():
			require.Fail(t, "context cancelled while waiting for Loki service: "+ctx.Err().Error())
			return
		case <-time.After(2 * time.Second):
		}
	}
	require.NoError(t, serviceErr, "Loki service %s/%s was not created within 3 minutes", serviceNamespace, serviceName)

	localURL, stopPortForward := portForwardNetObservServiceWithTimeout(ctx, t, restCfg, clientset, serviceNamespace, serviceName, servicePort, 3*time.Minute)
	defer stopPortForward()
	checkURL, err := url.Parse(localURL)
	require.NoError(t, err, "parse local Loki port-forward URL")
	checkURL.Path = "/ready"

	client := &http.Client{Timeout: 5 * time.Second}
	readinessDeadline := time.Now().Add(3 * time.Minute)
	lastStatus := "no response"
	t.Logf("Waiting for Loki /ready endpoint at %s...", checkURL.Redacted())
	for time.Now().Before(readinessDeadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, checkURL.String(), nil)
		require.NoError(t, err, "create Loki readiness request")

		response, err := client.Do(req)
		if err != nil {
			lastStatus = err.Error()
		} else {
			lastStatus = response.Status
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK || response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
				t.Logf("Loki is ready: %s", response.Status)
				return
			}
		}
		time.Sleep(2 * time.Second)
	}

	require.Fail(t, "Loki did not become ready within 3 minutes; last /ready response: "+lastStatus)
}

// cleanupNetObservOperator removes the NetObserv operator, FlowCollector, and IDMS.
func cleanupNetObservOperator(t *testing.T, kubeconfig string) {
	t.Helper()

	// Best effort cleanup - delete in reverse order
	t.Logf("Cleaning up NetObserv operator")

	// Delete FlowCollector
	cmd := exec.Command("kubectl", "delete", "-f", filepath.Join(netobservRealManifestDir, "flowcollector.yaml"), "--kubeconfig", kubeconfig, "--ignore-not-found")
	_ = cmd.Run()

	// Delete Subscription
	cmd = exec.Command("kubectl", "delete", "-f", filepath.Join(netobservRealManifestDir, "operator-subscription.yaml"), "--kubeconfig", kubeconfig, "--ignore-not-found")
	_ = cmd.Run()

	// Delete OperatorGroup
	cmd = exec.Command("kubectl", "delete", "-f", filepath.Join(netobservRealManifestDir, "operator-group.yaml"), "--kubeconfig", kubeconfig, "--ignore-not-found")
	_ = cmd.Run()

	// Delete namespaces
	cmd = exec.Command("kubectl", "delete", "-f", filepath.Join(netobservRealManifestDir, "operator-namespace.yaml"), "--kubeconfig", kubeconfig, "--ignore-not-found")
	_ = cmd.Run()

	// Delete CatalogSource
	cmd = exec.Command("kubectl", "delete", "-f", filepath.Join(netobservRealManifestDir, "operator-catalogsource.yaml"), "--kubeconfig", kubeconfig, "--ignore-not-found")
	_ = cmd.Run()

	// Delete the ImageDigestMirrorSet.
	cmd = exec.Command("kubectl", "delete", "-f", filepath.Join(netobservRealManifestDir, "operator-idms.yaml"), "--kubeconfig", kubeconfig, "--ignore-not-found")
	_ = cmd.Run()
}

func portForwardNetObservServiceWithTimeout(
	ctx context.Context,
	t *testing.T,
	restCfg *rest.Config,
	clientset kubernetes.Interface,
	namespace, serviceName string,
	servicePort int,
	podReadyTimeout time.Duration,
) (string, func()) {
	t.Helper()

	svc, err := clientset.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	require.NoError(t, err, "get service %s/%s", namespace, serviceName)
	selector := labels.SelectorFromSet(svc.Spec.Selector)

	deadline := time.Now().Add(podReadyTimeout)
	var podName string
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			require.FailNow(t, "context cancelled while waiting for a ready Loki pod: "+err.Error())
		}
		pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
		require.NoError(t, err, "list pods for service %s/%s", namespace, serviceName)
		for i := range pods.Items {
			pod := &pods.Items[i]
			if pod.Status.Phase != corev1.PodRunning {
				continue
			}
			for _, condition := range pod.Status.Conditions {
				if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
					podName = pod.Name
					break
				}
			}
			if podName != "" {
				break
			}
		}
		if podName != "" {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	require.NotEmpty(t, podName, "no ready pod found for service %s/%s within %s", namespace, serviceName, podReadyTimeout)
	localPort, stopFn := startPortForward(ctx, t, restCfg, namespace, podName, servicePort)
	return fmt.Sprintf("http://127.0.0.1:%d", localPort), stopFn
}

// TestNetObservReal tests NetObserv MCP tools against a FlowCollector with Loki enabled.
// Set NETOBSERV_OPERATOR=deploy to reuse a ready installation or deploy missing prerequisites.
func TestNetObservReal(t *testing.T) {
	operatorMode := os.Getenv("NETOBSERV_OPERATOR")
	if operatorMode == "" {
		t.Skip("Skipping real plugin tests - set NETOBSERV_OPERATOR=deploy to run")
	}

	f := features.New("netobserv-real").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			kubeconfig := cfg.KubeconfigFile()
			clientset, err := clientsetFromKubeconfig(kubeconfig)
			require.NoError(t, err, "create clientset")

			if operatorMode != "deploy" {
				t.Skipf("Invalid NETOBSERV_OPERATOR value: %s (use 'deploy')", operatorMode)
			}

			t.Logf("Ensuring NetObserv operator, FlowCollector, and Loki are ready")
			pluginNamespace := deployNetObservOperator(ctx, t, kubeconfig, clientset)

			// Deploy MCP server configured to use real plugin service
			configTOML := fmt.Sprintf(`
toolsets = ["core", "netobserv"]

[toolset_configs.netobserv]
namespace = "%s"
`, pluginNamespace)

			dep := deployServer(ctx, t, cfg, "netobserv-real",
				withNamespace("e2e-netobserv-real"),
				withConfig(configTOML),
				withValues(viewClusterRoleBindingValues()),
			)
			mcpClient := test.NewMcpClient(t, nil, test.WithEndpoint(dep.serverURL+"/mcp"))
			t.Cleanup(mcpClient.Close)
			return netobservRealTS.set(ctx, &netobservRealState{dep: dep, mcpClient: mcpClient})
		}).
		// Tier 1: Smoke Tests
		Assess("list_flows returns actual flow data", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			s := netobservRealTS.get(ctx)

			assertNetobservListFlows(t, s.mcpClient, map[string]any{
				"timeRange": makeTimeRange(15),
			})

			return ctx
		}).
		Assess("get_flow_metrics returns actual metrics", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			s := netobservRealTS.get(ctx)

			assertNetobservGetMetrics(t, s.mcpClient, map[string]any{
				"timeRange":   makeTimeRange(15),
				"aggregateBy": "namespace",
				"type":        "Bytes",
				"function":    "rate",
			})

			return ctx
		}).
		// Tier 3: Optional Feature Sample (DNS enrichment)
		Assess("DNS enrichment is available in flows", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			s := netobservRealTS.get(ctx)

			result, err := s.mcpClient.CallTool("netobserv_list_flows", map[string]any{
				"timeRange": makeTimeRange(30),
				"filters":   makeFilters("DnsFlagsResponseCode!="),
			})

			// This may return empty if no DNS flows exist, which is okay
			// We're just checking the filter works without error
			require.NoError(t, err, "DNS filter should not error")
			require.NotNil(t, result, "should return result")

			return ctx
		}).
		Feature()

	testenv.Test(t, f)
}
