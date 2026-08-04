package tnf

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/tnf/fencing"
)

const defaultTroubleshootTimeout = 120 * time.Second

func initTNFTroubleshoot() []api.ServerPrompt {
	return []api.ServerPrompt{
		{
			Prompt: api.Prompt{
				Name:  "tnf-troubleshoot",
				Title: "TNF Fencing Troubleshoot",
				Description: "Generate a step-by-step troubleshooting guide for diagnosing " +
					"Two-Node Fencing (TNF) cluster issues including STONITH configuration, " +
					"quorum status, BMC health, and split-brain risk assessment",
				Arguments: []api.PromptArgument{
					{
						Name:        "node",
						Description: "Node to run STONITH diagnostics on (auto-detects if omitted)",
						Required:    false,
					},
					{
						Name:        "namespace",
						Description: "Namespace for BareMetalHost resources (default: openshift-machine-api)",
						Required:    false,
					},
				},
			},
			Handler: tnfTroubleshootHandler,
		},
	}
}

func tnfTroubleshootHandler(params api.PromptHandlerParams) (*api.PromptCallResult, error) {
	if fencing.IsAPIUnreachable(params.KubernetesClient) {
		guide := fencing.APIUnreachableGuide()
		return api.NewPromptCallResult(
			"Kubernetes API is unreachable — returning out-of-band troubleshooting guide",
			[]api.PromptMessage{
				{
					Role: "user",
					Content: api.PromptContent{
						Type: "text",
						Text: guide,
					},
				},
				{
					Role: "assistant",
					Content: api.PromptContent{
						Type: "text",
						Text: "The Kubernetes API is unreachable. I'll walk through the out-of-band troubleshooting guide to help determine whether this is a failed install, an upgrade/reboot, or a crash scenario.",
					},
				},
			},
			nil,
		), nil
	}

	args := params.GetArguments()
	nodeName := args["node"]
	namespace := args["namespace"]
	if namespace == "" {
		namespace = "openshift-machine-api"
	}

	ctx := params.Context

	dynamicClient := params.DynamicClient()
	coreClient := params.CoreV1()

	topologyData, isTNF := fetchClusterTopology(ctx, dynamicClient, coreClient)
	nodeHealthData := fetchNodeHealth(ctx, coreClient)
	operatorData := fetchOperatorHealth(ctx, dynamicClient)
	bmhData := fetchBMHStatus(ctx, dynamicClient, coreClient, namespace)

	stonithData := fetchSTONITHData(ctx, params.KubernetesClient, coreClient, nodeName)

	var remediationData string
	if isTNF {
		remediationData = "TNF clusters use Pacemaker/STONITH for fencing — see STONITH data above.\n"
	} else {
		remediationData = fetchRemediationStatus(ctx, dynamicClient)
	}

	guideText := fmt.Sprintf(`# TNF Fencing Troubleshooting Guide

Use this guide to diagnose issues with Two-Node Fencing (TNF) clusters. All relevant
cluster data has been collected below.

---

## Step 1: Cluster Topology

Verify this is a TNF cluster (2-node bare metal with pacemaker fencing).
- Expect: Platform=BareMetal, 2 control-plane nodes, TNF Profile=Yes
- If not TNF: this guide may not apply

%s

---

## Step 2: Node Health

Both nodes must be Ready for the cluster to function. A NotReady node may trigger fencing.
- Both nodes Ready = healthy
- One node NotReady = potential fencing trigger (check pacemaker)
- Both nodes NotReady = critical — cluster is down

%s

---

## Step 3: Critical Operators

These operators are essential for TNF cluster health:
- **etcd**: Stores all cluster state; losing quorum is fatal for 2-node clusters
- **machine-api**: Manages BareMetalHost lifecycle
- **baremetal**: Manages BMC communication and provisioning

%s

---

## Step 4: BareMetalHost & BMC Health

BareMetalHost resources represent the physical nodes. BMC credentials must be valid
for STONITH fencing to work — the fence agent uses BMC to power-cycle nodes.
- Check: BMH provisioning state, BMC address configured, credentials secret exists
- Missing or invalid BMC credentials = fencing will fail when needed

%s

---

## Step 5: Pacemaker & STONITH Status

This section shows the live pacemaker cluster state gathered from the node via a
debug pod. Key things to check:
- **STONITH enabled**: Must be true for production. Disabling STONITH is a split-brain risk.
- **Nodes online**: Both nodes should be in the pacemaker cluster
- **Fence devices**: Each node needs a working fence device targeting the OTHER node
- **Quorum**: Cluster must be quorate. Two-node mode uses special quorum rules.

%s

---

## Step 6: Remediation Operators

%s

---

## TNF Domain Knowledge

For detailed domain knowledge about two-node fencing mechanics, split-brain
risk assessment, and common recovery procedures, read the MCP resource:

  tnf://domain-knowledge/fencing

Reference this resource to interpret the data collected above and assess
split-brain risk.

---

## Troubleshooting Analysis

Based on the data collected above (and the domain knowledge resource), analyze:

1. **Is this a TNF cluster?** Check topology — 2-node bare metal with pacemaker
2. **Are both nodes healthy?** Check node Ready status and pacemaker online list
3. **Is STONITH properly configured?** Devices exist, started, targeting correct nodes
4. **Are BMC credentials valid?** Secrets exist with username/password keys
5. **Is the cluster quorate?** Check corosync quorum status
6. **Any active issues?** Failed resources, active remediations, fencing events
7. **Split-brain risk level?** Use the risk matrix from the domain knowledge resource

---

## Report Findings

After analysis, report:
- **Status:** Healthy / At Risk / Degraded / Critical
- **Split-Brain Risk:** None / Low / Medium / High / Critical
- **Issues Found:** List of specific problems
- **Root Cause:** Description of the primary issue (or "None found")
- **Recommended Actions:** Specific steps to resolve issues
- **Verification:** How to confirm the fix worked
`, topologyData, nodeHealthData, operatorData, bmhData, stonithData, remediationData)

	return api.NewPromptCallResult(
		"TNF fencing troubleshooting guide generated",
		[]api.PromptMessage{
			{
				Role: "user",
				Content: api.PromptContent{
					Type: "text",
					Text: guideText,
				},
			},
			{
				Role: "assistant",
				Content: api.PromptContent{
					Type: "text",
					Text: "I'll analyze the collected TNF cluster data to assess fencing health, identify split-brain risks, and provide specific remediation guidance.",
				},
			},
		},
		nil,
	), nil
}

func fetchClusterTopology(ctx context.Context, dynamicClient dynamic.Interface, coreClient corev1client.CoreV1Interface) (string, bool) {
	var result strings.Builder
	result.WriteString("### Cluster Topology\n\n")

	infra, err := dynamicClient.Resource(fencing.InfrastructureGVR).Get(ctx, "cluster", metav1.GetOptions{})
	if err != nil {
		slog.Debug("could not get Infrastructure CR", "error", err)
		result.WriteString("*Infrastructure CR not available*\n")
		return result.String(), false
	}

	platform, _, _ := unstructured.NestedString(infra.Object, "status", "platform")
	infraTopology, _, _ := unstructured.NestedString(infra.Object, "status", "infrastructureTopology")
	cpTopology, _, _ := unstructured.NestedString(infra.Object, "status", "controlPlaneTopology")

	fmt.Fprintf(&result, "- **Platform:** %s\n", fencing.ValueOrNA(platform))
	fmt.Fprintf(&result, "- **Infrastructure Topology:** %s\n", fencing.ValueOrNA(infraTopology))
	fmt.Fprintf(&result, "- **Control Plane Topology:** %s\n", fencing.ValueOrNA(cpTopology))

	nodes, err := coreClient.Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		slog.Debug("could not list nodes", "error", err)
		return result.String(), false
	}

	cpCount := 0
	for _, node := range nodes.Items {
		if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; ok {
			cpCount++
		} else if _, ok := node.Labels["node-role.kubernetes.io/master"]; ok {
			cpCount++
		}
	}
	fmt.Fprintf(&result, "- **Total Nodes:** %d\n", len(nodes.Items))
	fmt.Fprintf(&result, "- **Control-Plane Nodes:** %d\n", cpCount)

	isTNF := strings.EqualFold(platform, "BareMetal") && cpCount == 2
	if isTNF {
		result.WriteString("- **TNF Profile:** Yes (2-node bare metal)\n")
	} else {
		result.WriteString("- **TNF Profile:** No\n")
	}

	return result.String(), isTNF
}

func fetchNodeHealth(ctx context.Context, coreClient corev1client.CoreV1Interface) string {
	var result strings.Builder
	result.WriteString("### Node Status\n\n")

	nodes, err := coreClient.Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(&result, "*Error listing nodes: %v*\n", err)
		return result.String()
	}

	if len(nodes.Items) == 0 {
		result.WriteString("*No nodes found*\n")
		return result.String()
	}

	result.WriteString("| Node | Ready | Roles | Conditions |\n")
	result.WriteString("|------|-------|-------|------------|\n")

	for _, node := range nodes.Items {
		ready := "Unknown"
		var conditions []string

		for _, cond := range node.Status.Conditions {
			switch cond.Type {
			case "Ready":
				if cond.Status == "True" {
					ready = "Yes"
				} else {
					ready = "**No**"
				}
			case "MemoryPressure", "DiskPressure", "PIDPressure":
				if cond.Status == "True" {
					conditions = append(conditions, string(cond.Type))
				}
			}
		}

		var roles []string
		for label := range node.Labels {
			if strings.HasPrefix(label, "node-role.kubernetes.io/") {
				role := strings.TrimPrefix(label, "node-role.kubernetes.io/")
				roles = append(roles, role)
			}
		}

		condStr := "None"
		if len(conditions) > 0 {
			condStr = strings.Join(conditions, ", ")
		}

		fmt.Fprintf(&result, "| %s | %s | %s | %s |\n",
			node.Name, ready, strings.Join(roles, ", "), condStr)
	}

	return result.String()
}

func fetchOperatorHealth(ctx context.Context, dynamicClient dynamic.Interface) string {
	var result strings.Builder
	result.WriteString("### Cluster Operators\n\n")

	tnfOperators := []string{"etcd", "machine-api", "baremetal"}

	list, err := dynamicClient.Resource(fencing.ClusterOperatorGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		slog.Debug("could not list ClusterOperators", "error", err)
		result.WriteString("*ClusterOperators not available (may not be an OpenShift cluster)*\n")
		return result.String()
	}

	result.WriteString("| Operator | Available | Degraded | Progressing |\n")
	result.WriteString("|----------|-----------|----------|-------------|\n")

	found := make(map[string]bool)
	for _, item := range list.Items {
		name := item.GetName()
		isRelevant := false
		for _, op := range tnfOperators {
			if name == op {
				isRelevant = true
				break
			}
		}
		if !isRelevant {
			continue
		}
		found[name] = true

		conditions, _, _ := unstructured.NestedSlice(item.Object, "status", "conditions")
		available, degraded, progressing := "Unknown", "Unknown", "Unknown"

		for _, cond := range conditions {
			condMap, ok := cond.(map[string]interface{})
			if !ok {
				continue
			}
			condType, _ := condMap["type"].(string)
			condStatus, _ := condMap["status"].(string)

			switch condType {
			case "Available":
				available = condStatus
			case "Degraded":
				degraded = condStatus
			case "Progressing":
				progressing = condStatus
			}
		}

		fmt.Fprintf(&result, "| %s | %s | %s | %s |\n",
			name, available, degraded, progressing)
	}

	for _, op := range tnfOperators {
		if !found[op] {
			fmt.Fprintf(&result, "| %s | **NOT FOUND** | - | - |\n", op)
		}
	}

	return result.String()
}

func fetchBMHStatus(ctx context.Context, dynamicClient dynamic.Interface, coreClient corev1client.CoreV1Interface, namespace string) string {
	var result strings.Builder
	result.WriteString("### BareMetalHost Status\n\n")

	var list *unstructured.UnstructuredList
	var err error
	if namespace != "" {
		list, err = dynamicClient.Resource(fencing.BareMetalHostGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	} else {
		list, err = dynamicClient.Resource(fencing.BareMetalHostGVR).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		fmt.Fprintf(&result, "*Error listing BareMetalHosts: %v*\n", err)
		return result.String()
	}

	if len(list.Items) == 0 {
		result.WriteString("*No BareMetalHost resources found — this may not be a bare metal cluster*\n")
		return result.String()
	}

	fmt.Fprintf(&result, "Found %d BareMetalHost(s):\n\n", len(list.Items))

	for _, host := range list.Items {
		hostName := host.GetName()
		hostNS := host.GetNamespace()

		provState, _, _ := unstructured.NestedString(host.Object, "status", "provisioning", "state")
		opStatus, _, _ := unstructured.NestedString(host.Object, "status", "operationalStatus")
		poweredOn, _, _ := unstructured.NestedBool(host.Object, "status", "poweredOn")
		errorMsg, _, _ := unstructured.NestedString(host.Object, "status", "errorMessage")
		bmcAddr, _, _ := unstructured.NestedString(host.Object, "spec", "bmc", "address")
		credName, _, _ := unstructured.NestedString(host.Object, "spec", "bmc", "credentialsName")

		fmt.Fprintf(&result, "#### %s/%s\n\n", hostNS, hostName)

		isUnmanaged := provState == "" || provState == "unmanaged"
		if isUnmanaged && bmcAddr == "" {
			result.WriteString("- State: Unmanaged (not controlled by baremetal-operator)\n")
		} else {
			fmt.Fprintf(&result, "- Provisioning State: %s\n", fencing.ValueOrNA(provState))
			fmt.Fprintf(&result, "- Operational Status: %s\n", fencing.ValueOrNA(opStatus))
			fmt.Fprintf(&result, "- Powered On: %t\n", poweredOn)
			fmt.Fprintf(&result, "- BMC Address: %s\n", fencing.ValueOrNA(bmcAddr))
			fmt.Fprintf(&result, "- Credentials Secret: %s\n", fencing.ValueOrNA(credName))

			if errorMsg != "" {
				fmt.Fprintf(&result, "- **Error:** %s\n", errorMsg)
			}

			if credName != "" {
				credStatus := checkCredentialSecret(ctx, coreClient, hostNS, credName)
				fmt.Fprintf(&result, "- Credentials Check: %s\n", credStatus)
			} else if !isUnmanaged {
				result.WriteString("- Credentials Check: **No credentials secret configured**\n")
			}
		}
		result.WriteString("\n")
	}

	return result.String()
}

func checkCredentialSecret(ctx context.Context, coreClient corev1client.CoreV1Interface, namespace, secretName string) string {
	secret, err := coreClient.Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Sprintf("**Secret %q not found** — %v", secretName, err)
	}

	userVal, hasUser := secret.Data["username"]
	passVal, hasPass := secret.Data["password"]
	userValid := hasUser && len(userVal) > 0
	passValid := hasPass && len(passVal) > 0

	if !userValid && !passValid {
		return fmt.Sprintf("**Secret %q missing or empty 'username' and 'password'**", secretName)
	}
	if !userValid {
		return fmt.Sprintf("**Secret %q missing or empty 'username'**", secretName)
	}
	if !passValid {
		return fmt.Sprintf("**Secret %q missing or empty 'password'**", secretName)
	}
	return "Valid (username and password present)"
}

func fetchSTONITHData(ctx context.Context, k8sClient api.KubernetesClient, coreClient corev1client.CoreV1Interface, nodeName string) string {
	report, issues, err := fencing.RunSTONITHDiagnostics(ctx, k8sClient, coreClient, nodeName, "", defaultTroubleshootTimeout)
	if err != nil {
		return fmt.Sprintf("### STONITH Diagnostics\n\n*Error running STONITH diagnostics: %v*\n\n"+
			"This may indicate:\n- No control-plane nodes found\n- Debug pod could not be created\n"+
			"- pcs/corosync not installed (not a pacemaker-based cluster)\n", err)
	}

	if len(issues) > 0 {
		var issueSection strings.Builder
		issueSection.WriteString("\n### Detected Issues\n\n")
		for i, issue := range issues {
			fmt.Fprintf(&issueSection, "%d. **%s**\n", i+1, issue)
		}
		return report + issueSection.String()
	}

	return report
}

func fetchRemediationStatus(ctx context.Context, dynamicClient dynamic.Interface) string {
	var result strings.Builder
	result.WriteString("### Remediation Operators\n\n")

	// FenceAgentsRemediation Templates
	result.WriteString("#### FenceAgentsRemediation Templates\n\n")
	templates, err := dynamicClient.Resource(fencing.FenceAgentsRemediationTemplateGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		slog.Debug("could not list FAR templates", "error", err)
		result.WriteString("*FenceAgentsRemediationTemplate CRD not installed — cluster may use traditional pacemaker/STONITH fencing*\n\n")
	} else if len(templates.Items) == 0 {
		result.WriteString("*No FenceAgentsRemediationTemplates configured*\n\n")
	} else {
		for _, tmpl := range templates.Items {
			name := tmpl.GetName()
			ns := tmpl.GetNamespace()
			agent, _, _ := unstructured.NestedString(tmpl.Object, "spec", "template", "spec", "agent")
			location := name
			if ns != "" {
				location = fmt.Sprintf("%s/%s", ns, name)
			}
			fmt.Fprintf(&result, "- %s: agent=%s\n", location, fencing.ValueOrNA(agent))
		}
		result.WriteString("\n")
	}

	// Active remediations
	result.WriteString("#### Active Remediations\n\n")
	remediations, err := dynamicClient.Resource(fencing.FenceAgentsRemediationGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		slog.Debug("could not list FAR remediations", "error", err)
		result.WriteString("*Could not check active remediations*\n\n")
	} else if len(remediations.Items) == 0 {
		result.WriteString("*No active fencing remediations*\n\n")
	} else {
		for _, rem := range remediations.Items {
			agent, _, _ := unstructured.NestedString(rem.Object, "spec", "agent")
			fmt.Fprintf(&result, "- **%s/%s**: agent=%s\n", rem.GetNamespace(), rem.GetName(), fencing.ValueOrNA(agent))
		}
		result.WriteString("\n")
	}

	// NodeHealthChecks
	result.WriteString("#### NodeHealthCheck\n\n")
	nhcList, err := dynamicClient.Resource(fencing.NodeHealthCheckGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		slog.Debug("could not list NHC", "error", err)
		result.WriteString("*NodeHealthCheck CRD not installed*\n\n")
	} else if len(nhcList.Items) == 0 {
		result.WriteString("*No NodeHealthCheck resources configured*\n\n")
	} else {
		for _, nhc := range nhcList.Items {
			phase, _, _ := unstructured.NestedString(nhc.Object, "status", "phase")
			fmt.Fprintf(&result, "- %s: phase=%s\n", nhc.GetName(), fencing.ValueOrNA(phase))
		}
		result.WriteString("\n")
	}

	return result.String()
}
