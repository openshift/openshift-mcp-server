package fencing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/utils/ptr"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
)

var (
	BareMetalHostGVR = schema.GroupVersionResource{
		Group: "metal3.io", Version: "v1alpha1", Resource: "baremetalhosts",
	}
	ClusterOperatorGVR = schema.GroupVersionResource{
		Group: "config.openshift.io", Version: "v1", Resource: "clusteroperators",
	}
	InfrastructureGVR = schema.GroupVersionResource{
		Group: "config.openshift.io", Version: "v1", Resource: "infrastructures",
	}
	MachineGVR = schema.GroupVersionResource{
		Group: "machine.openshift.io", Version: "v1beta1", Resource: "machines",
	}
	FenceAgentsRemediationGVR = schema.GroupVersionResource{
		Group: "fence-agents-remediation.medik8s.io", Version: "v1alpha1",
		Resource: "fenceagentsremediations",
	}
	FenceAgentsRemediationTemplateGVR = schema.GroupVersionResource{
		Group: "fence-agents-remediation.medik8s.io", Version: "v1alpha1",
		Resource: "fenceagentsremediationtemplates",
	}
	NodeHealthCheckGVR = schema.GroupVersionResource{
		Group: "remediation.medik8s.io", Version: "v1alpha1",
		Resource: "nodehealthchecks",
	}
)

func InitFencing() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name: "tnf_check_fencing_config",
				Description: "Check fencing configuration and readiness for a Two-Node Fencing (TNF) cluster. " +
					"Validates cluster topology, critical operator health (etcd, machine-api, baremetal), " +
					"Machine/Node/BareMetalHost correlation, BMC addresses and credentials, " +
					"FenceAgentsRemediation templates and active remediations, and NodeHealthCheck resources. " +
					"Returns a diagnostic summary identifying configuration issues that could " +
					"prevent fencing from functioning correctly.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Namespace containing BareMetalHost resources (e.g. 'openshift-machine-api'). If omitted, searches all namespaces.",
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "TNF: Check Fencing Config",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					IdempotentHint:  ptr.To(true),
					OpenWorldHint:   ptr.To(false),
				},
			},
			Handler: checkFencingHealth,
		},
	}
}

func checkFencingHealth(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	if IsAPIUnreachable(params.KubernetesClient) {
		return api.NewToolCallResult(APIUnreachableGuide(), nil), nil
	}

	p := api.WrapParams(params)
	namespace := p.OptionalString("namespace", "")
	if err := p.Err(); err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to check fencing health: %w", err)), nil
	}

	dynamicClient := params.DynamicClient()
	coreClient := params.CoreV1()

	var report strings.Builder
	var issues []string

	report.WriteString("# TNF Fencing Health Check\n\n")

	topoIssues := checkInfrastructureTopology(params.Context, dynamicClient, coreClient, &report)
	issues = append(issues, topoIssues...)

	coIssues := checkClusterOperatorHealth(params.Context, dynamicClient, &report)
	issues = append(issues, coIssues...)

	bmhToNode, corrIssues := correlateMachinesWithHosts(params.Context, dynamicClient, &report)
	issues = append(issues, corrIssues...)

	hosts, err := listBareMetalHosts(params.Context, dynamicClient, namespace)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list BareMetalHost resources: %w", err)), nil
	}

	if len(hosts) == 0 {
		msg := "No BareMetalHost resources found"
		if namespace != "" {
			msg += fmt.Sprintf(" in namespace %q", namespace)
		}
		report.WriteString(msg + ". This cluster may not be a bare metal deployment.\n\n")
		issues = append(issues, msg)
	} else {
		fmt.Fprintf(&report, "BareMetalHosts found: %d\n\n", len(hosts))
		for _, host := range hosts {
			hostIssues := writeHostSection(params.Context, coreClient, &report, host, bmhToNode)
			issues = append(issues, hostIssues...)
		}
	}

	farIssues := checkFenceAgentsRemediation(params.Context, dynamicClient, &report)
	issues = append(issues, farIssues...)

	nhcIssues := checkNodeHealthChecks(params.Context, dynamicClient, &report)
	issues = append(issues, nhcIssues...)

	report.WriteString("## Summary\n\n")
	if len(issues) == 0 {
		report.WriteString("No fencing health issues detected.\n")
	} else {
		fmt.Fprintf(&report, "Found %d issue(s):\n\n", len(issues))
		for i, issue := range issues {
			fmt.Fprintf(&report, "%d. %s\n", i+1, issue)
		}
	}

	return api.NewToolCallResult(report.String(), nil), nil
}

func checkInfrastructureTopology(ctx context.Context, client dynamic.Interface, coreClient corev1.CoreV1Interface, report *strings.Builder) []string {
	var issues []string
	report.WriteString("## Cluster Topology\n\n")

	infra, err := client.Resource(InfrastructureGVR).Get(ctx, "cluster", metav1.GetOptions{})
	if err != nil {
		slog.Debug("could not get Infrastructure CR", "error", err)
		if isCRDNotInstalled(err) {
			report.WriteString("- Infrastructure CR: not available (may not be an OpenShift cluster)\n\n")
		} else {
			fmt.Fprintf(report, "- Infrastructure CR: **error** — %v\n\n", err)
			issues = append(issues, fmt.Sprintf("failed to get Infrastructure CR: %v", err))
		}
		return issues
	}

	platform, _, err := unstructured.NestedString(infra.Object, "status", "platform")
	if err != nil {
		slog.Debug("malformed Infrastructure CR field", "field", "status.platform", "error", err)
	}
	infraTopology, _, err := unstructured.NestedString(infra.Object, "status", "infrastructureTopology")
	if err != nil {
		slog.Debug("malformed Infrastructure CR field", "field", "status.infrastructureTopology", "error", err)
	}
	cpTopology, _, err := unstructured.NestedString(infra.Object, "status", "controlPlaneTopology")
	if err != nil {
		slog.Debug("malformed Infrastructure CR field", "field", "status.controlPlaneTopology", "error", err)
	}

	fmt.Fprintf(report, "- Platform: %s\n", ValueOrNA(platform))
	fmt.Fprintf(report, "- Infrastructure Topology: %s\n", ValueOrNA(infraTopology))
	fmt.Fprintf(report, "- Control Plane Topology: %s\n", ValueOrNA(cpTopology))

	nodes, err := coreClient.Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		slog.Debug("could not list nodes for topology check", "error", err)
	} else {
		cpCount := 0
		for _, node := range nodes.Items {
			if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; ok {
				cpCount++
			} else if _, ok := node.Labels["node-role.kubernetes.io/master"]; ok {
				cpCount++
			}
		}
		fmt.Fprintf(report, "- Total nodes: %d\n", len(nodes.Items))
		fmt.Fprintf(report, "- Control-plane nodes: %d\n", cpCount)

		isTNF := strings.EqualFold(platform, "BareMetal") && cpCount == 2
		if isTNF {
			report.WriteString("- TNF Profile: **Yes** (2-node bare metal)\n")
		} else {
			report.WriteString("- TNF Profile: No\n")
			if !strings.EqualFold(platform, "BareMetal") {
				issues = append(issues, fmt.Sprintf("platform is %q, expected BareMetal for TNF", platform))
			}
			if cpCount != 2 {
				issues = append(issues, fmt.Sprintf("found %d control-plane nodes, expected 2 for TNF", cpCount))
			}
		}
	}

	report.WriteString("\n")
	return issues
}

func checkClusterOperatorHealth(ctx context.Context, client dynamic.Interface, report *strings.Builder) []string {
	var issues []string
	report.WriteString("## Cluster Operator Health\n\n")

	tnfOperators := map[string]bool{
		"etcd":        true,
		"machine-api": true,
		"baremetal":   true,
	}

	list, err := client.Resource(ClusterOperatorGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		slog.Debug("could not list ClusterOperators", "error", err)
		if isCRDNotInstalled(err) {
			report.WriteString("- ClusterOperators: not available (may not be an OpenShift cluster)\n\n")
		} else {
			fmt.Fprintf(report, "- ClusterOperators: **error** — %v\n\n", err)
			issues = append(issues, fmt.Sprintf("failed to list ClusterOperators: %v", err))
		}
		return issues
	}

	foundOps := make(map[string]bool)

	for _, item := range list.Items {
		name := item.GetName()
		if !tnfOperators[name] {
			continue
		}
		foundOps[name] = true

		conditions, _, _ := unstructured.NestedSlice(item.Object, "status", "conditions")
		available := "Unknown"
		degraded := "Unknown"
		progressing := "Unknown"
		var opIssues []string

		for _, cond := range conditions {
			condMap, ok := cond.(map[string]interface{})
			if !ok {
				continue
			}
			condType, _ := condMap["type"].(string)
			condStatus, _ := condMap["status"].(string)
			message, _ := condMap["message"].(string)

			switch condType {
			case "Available":
				available = condStatus
				if condStatus != "True" {
					opIssues = append(opIssues, fmt.Sprintf("not available: %s", message))
				}
			case "Degraded":
				degraded = condStatus
				if condStatus == "True" {
					opIssues = append(opIssues, fmt.Sprintf("degraded: %s", message))
				}
			case "Progressing":
				progressing = condStatus
			}
		}

		fmt.Fprintf(report, "- %s: Available=%s, Degraded=%s, Progressing=%s\n",
			name, available, degraded, progressing)
		for _, oi := range opIssues {
			fmt.Fprintf(report, "  - **Issue**: %s\n", oi)
			issues = append(issues, fmt.Sprintf("operator %s: %s", name, oi))
		}
	}

	for op := range tnfOperators {
		if !foundOps[op] {
			fmt.Fprintf(report, "- %s: **not found**\n", op)
			issues = append(issues, fmt.Sprintf("operator %s not found", op))
		}
	}

	report.WriteString("\n")
	return issues
}

// correlateMachinesWithHosts writes a Machine/Node/BMH correlation table and
// returns a mapping from "namespace/bmhName" to the Node name resolved via
// Machine.status.nodeRef.name, along with any issues found.
func correlateMachinesWithHosts(ctx context.Context, client dynamic.Interface, report *strings.Builder) (map[string]string, []string) {
	bmhToNode := make(map[string]string)
	var issues []string
	report.WriteString("## Machine/Node/BMH Correlation\n\n")

	list, err := client.Resource(MachineGVR).Namespace("openshift-machine-api").List(ctx, metav1.ListOptions{})
	if err != nil {
		slog.Debug("could not list Machines", "error", err)
		if isCRDNotInstalled(err) {
			report.WriteString("- Machines: not available (Machine API not installed)\n\n")
		} else {
			fmt.Fprintf(report, "- Machines: **error** — %v\n\n", err)
			issues = append(issues, fmt.Sprintf("failed to list Machines: %v", err))
		}
		return bmhToNode, issues
	}

	if len(list.Items) == 0 {
		report.WriteString("No Machine resources found in openshift-machine-api.\n\n")
		return bmhToNode, issues
	}

	report.WriteString("| Machine | Node | BMH | Phase |\n")
	report.WriteString("|---------|------|-----|-------|\n")

	for _, machine := range list.Items {
		machineName := machine.GetName()
		annotations := machine.GetAnnotations()

		nodeRef, _, _ := unstructured.NestedString(machine.Object, "status", "nodeRef", "name")
		phase, _, _ := unstructured.NestedString(machine.Object, "status", "phase")
		bmhRef := annotations["metal3.io/BareMetalHost"]

		nodeDisplay := ValueOrNA(nodeRef)
		bmhDisplay := ValueOrNA(bmhRef)

		fmt.Fprintf(report, "| %s | %s | %s | %s |\n",
			machineName, nodeDisplay, bmhDisplay, ValueOrNA(phase))

		if nodeRef != "" && bmhRef != "" {
			bmhToNode[bmhRef] = nodeRef
		}

		if nodeRef == "" {
			issues = append(issues, fmt.Sprintf("machine %s has no nodeRef (not yet provisioned or failed)", machineName))
		}
		if bmhRef == "" {
			issues = append(issues, fmt.Sprintf("machine %s has no metal3.io/BareMetalHost annotation", machineName))
		}
	}

	report.WriteString("\n")
	return bmhToNode, issues
}

func writeHostSection(ctx context.Context, coreClient corev1.CoreV1Interface, report *strings.Builder, host unstructured.Unstructured, bmhToNode map[string]string) []string {
	var issues []string
	hostName := host.GetName()
	hostNamespace := host.GetNamespace()

	fmt.Fprintf(report, "## Host: %s/%s\n\n", hostNamespace, hostName)

	provisioningState, _, _ := unstructured.NestedString(host.Object, "status", "provisioning", "state")
	operationalStatus, _, _ := unstructured.NestedString(host.Object, "status", "operationalStatus")
	poweredOn, _, _ := unstructured.NestedBool(host.Object, "status", "poweredOn")
	errorMessage, _, _ := unstructured.NestedString(host.Object, "status", "errorMessage")
	errorType, _, _ := unstructured.NestedString(host.Object, "status", "errorType")
	bmcAddress, _, _ := unstructured.NestedString(host.Object, "spec", "bmc", "address")
	credentialsName, _, _ := unstructured.NestedString(host.Object, "spec", "bmc", "credentialsName")
	online, _, _ := unstructured.NestedBool(host.Object, "spec", "online")
	goodCredsName, _, _ := unstructured.NestedString(host.Object, "status", "goodCredentials", "credentials", "name")

	consumerKind, _, _ := unstructured.NestedString(host.Object, "spec", "consumerRef", "kind")
	consumerName, _, _ := unstructured.NestedString(host.Object, "spec", "consumerRef", "name")

	manufacturer, _, _ := unstructured.NestedString(host.Object, "status", "hardware", "systemVendor", "manufacturer")
	productName, _, _ := unstructured.NestedString(host.Object, "status", "hardware", "systemVendor", "productName")

	isUnmanaged := provisioningState == "" || provisioningState == "unmanaged"

	if isUnmanaged && bmcAddress == "" {
		report.WriteString("- State: **Unmanaged** (not controlled by baremetal-operator)\n")
		report.WriteString("- BMC/credentials: not applicable for unmanaged hosts\n")
		if operationalStatus != "" {
			fmt.Fprintf(report, "- Operational Status: %s\n", operationalStatus)
		}
		if consumerKind != "" {
			fmt.Fprintf(report, "- Consumer: %s/%s\n", consumerKind, consumerName)
		}
	} else {
		fmt.Fprintf(report, "- Provisioning State: %s\n", ValueOrNA(provisioningState))
		fmt.Fprintf(report, "- Operational Status: %s\n", ValueOrNA(operationalStatus))
		fmt.Fprintf(report, "- Online (desired): %t\n", online)
		fmt.Fprintf(report, "- Powered On (actual): %t\n", poweredOn)
		fmt.Fprintf(report, "- BMC Address: %s\n", ValueOrNA(bmcAddress))
		fmt.Fprintf(report, "- Credentials Secret: %s\n", ValueOrNA(credentialsName))

		if goodCredsName != "" {
			fmt.Fprintf(report, "- Good Credentials: %s (verified by BMO)\n", goodCredsName)
		}

		if consumerKind != "" {
			fmt.Fprintf(report, "- Consumer: %s/%s\n", consumerKind, consumerName)
		}

		if manufacturer != "" || productName != "" {
			fmt.Fprintf(report, "- Hardware: %s %s\n", manufacturer, productName)
		}

		if errorMessage != "" {
			fmt.Fprintf(report, "- **Error**: [%s] %s\n", errorType, errorMessage)
			issues = append(issues, fmt.Sprintf("%s/%s: BMH error — [%s] %s", hostNamespace, hostName, errorType, errorMessage))
		}

		if bmcAddress == "" {
			issues = append(issues, fmt.Sprintf("%s/%s: no BMC address configured", hostNamespace, hostName))
		}

		if credentialsName != "" {
			credIssues := checkCredentials(ctx, coreClient, hostNamespace, credentialsName, hostName)
			issues = append(issues, credIssues...)
			if len(credIssues) == 0 {
				report.WriteString("- Credentials: present and valid\n")
			} else {
				for _, ci := range credIssues {
					fmt.Fprintf(report, "- **Credentials Issue**: %s\n", ci)
				}
			}
		} else if !isUnmanaged {
			issues = append(issues, fmt.Sprintf("%s/%s: no BMC credentials secret configured", hostNamespace, hostName))
		}
	}

	nodeName := hostName
	bmhKey := fmt.Sprintf("%s/%s", hostNamespace, hostName)
	if resolved, ok := bmhToNode[bmhKey]; ok {
		nodeName = resolved
	}

	nodeIssues := checkNodeHealth(ctx, coreClient, nodeName)
	issues = append(issues, nodeIssues...)
	for _, ni := range nodeIssues {
		fmt.Fprintf(report, "- **Node Issue**: %s\n", ni)
	}
	if len(nodeIssues) == 0 {
		report.WriteString("- Node: healthy\n")
	}

	report.WriteString("\n")
	return issues
}

func checkFenceAgentsRemediation(ctx context.Context, client dynamic.Interface, report *strings.Builder) []string {
	var issues []string
	report.WriteString("## Fencing Remediation\n\n")

	report.WriteString("### Templates\n\n")
	templates, err := client.Resource(FenceAgentsRemediationTemplateGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		slog.Debug("could not list FenceAgentsRemediationTemplates", "error", err)
		if isCRDNotInstalled(err) {
			report.WriteString("FenceAgentsRemediationTemplate CRD not installed. Cluster may use traditional pacemaker/STONITH fencing instead.\n\n")
		} else {
			fmt.Fprintf(report, "FenceAgentsRemediationTemplates: **error** — %v\n\n", err)
			issues = append(issues, fmt.Sprintf("failed to list FenceAgentsRemediationTemplates: %v", err))
		}
	} else if len(templates.Items) == 0 {
		report.WriteString("No FenceAgentsRemediationTemplates found. Cluster may use traditional pacemaker/STONITH fencing instead.\n\n")
	} else {
		for _, tmpl := range templates.Items {
			name := tmpl.GetName()
			ns := tmpl.GetNamespace()
			agent, _, _ := unstructured.NestedString(tmpl.Object, "spec", "template", "spec", "agent")
			sharedParams, _, _ := unstructured.NestedMap(tmpl.Object, "spec", "template", "spec", "sharedparameters")
			retryCount, _, _ := unstructured.NestedInt64(tmpl.Object, "spec", "template", "spec", "retrycount")

			location := name
			if ns != "" {
				location = fmt.Sprintf("%s/%s", ns, name)
			}
			fmt.Fprintf(report, "- **%s**: agent=%s", location, ValueOrNA(agent))
			if retryCount > 0 {
				fmt.Fprintf(report, ", retryCount=%d", retryCount)
			}
			if len(sharedParams) > 0 {
				var paramKeys []string
				for k := range sharedParams {
					paramKeys = append(paramKeys, k)
				}
				fmt.Fprintf(report, ", sharedParams={%s}", strings.Join(paramKeys, ", "))
			}
			report.WriteString("\n")
		}
	}

	report.WriteString("\n### Active Remediations\n\n")
	remediations, err := client.Resource(FenceAgentsRemediationGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		slog.Debug("could not list FenceAgentsRemediations", "error", err)
		if isCRDNotInstalled(err) {
			report.WriteString("FenceAgentsRemediation CRD not installed.\n\n")
		} else {
			fmt.Fprintf(report, "Active Remediations: **error** — %v\n\n", err)
			issues = append(issues, fmt.Sprintf("failed to list FenceAgentsRemediations: %v", err))
		}
		return issues
	}

	if len(remediations.Items) == 0 {
		report.WriteString("No active fencing remediations.\n\n")
	} else {
		for _, rem := range remediations.Items {
			remName := rem.GetName()
			remNs := rem.GetNamespace()
			agent, _, _ := unstructured.NestedString(rem.Object, "spec", "agent")
			conditions, _, _ := unstructured.NestedSlice(rem.Object, "status", "conditions")

			fmt.Fprintf(report, "- **%s/%s**: agent=%s\n", remNs, remName, ValueOrNA(agent))
			for _, cond := range conditions {
				condMap, ok := cond.(map[string]interface{})
				if !ok {
					continue
				}
				condType, _ := condMap["type"].(string)
				condStatus, _ := condMap["status"].(string)
				message, _ := condMap["message"].(string)
				fmt.Fprintf(report, "  - %s=%s: %s\n", condType, condStatus, message)
			}
			issues = append(issues, fmt.Sprintf("active fencing remediation in progress: %s/%s", remNs, remName))
		}
	}

	report.WriteString("\n")
	return issues
}

func checkNodeHealthChecks(ctx context.Context, client dynamic.Interface, report *strings.Builder) []string {
	var issues []string
	report.WriteString("## Node Health Checks\n\n")

	list, err := client.Resource(NodeHealthCheckGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		slog.Debug("could not list NodeHealthChecks", "error", err)
		if isCRDNotInstalled(err) {
			report.WriteString("NodeHealthCheck CRD not installed. Cluster may use traditional pacemaker-based health monitoring.\n\n")
		} else {
			fmt.Fprintf(report, "NodeHealthChecks: **error** — %v\n\n", err)
			issues = append(issues, fmt.Sprintf("failed to list NodeHealthChecks: %v", err))
		}
		return issues
	}

	if len(list.Items) == 0 {
		report.WriteString("No NodeHealthCheck resources found. Cluster may use traditional pacemaker-based health monitoring.\n\n")
		return issues
	}

	for _, nhc := range list.Items {
		nhcName := nhc.GetName()
		phase, _, _ := unstructured.NestedString(nhc.Object, "status", "phase")
		minHealthy, _, _ := unstructured.NestedString(nhc.Object, "spec", "minHealthy")

		selectorLabels, _, _ := unstructured.NestedStringMap(nhc.Object, "spec", "selector", "matchLabels")

		var selectorStr string
		if len(selectorLabels) > 0 {
			var parts []string
			for k, v := range selectorLabels {
				if v == "" {
					parts = append(parts, k)
				} else {
					parts = append(parts, fmt.Sprintf("%s=%s", k, v))
				}
			}
			selectorStr = strings.Join(parts, ", ")
		} else {
			selectorStr = "(all nodes)"
		}

		fmt.Fprintf(report, "- **%s**: selector={%s}\n", nhcName, selectorStr)
		fmt.Fprintf(report, "  - Phase: %s\n", ValueOrNA(phase))
		if minHealthy != "" {
			fmt.Fprintf(report, "  - Min Healthy: %s\n", minHealthy)
		}

		escalating, found, _ := unstructured.NestedSlice(nhc.Object, "spec", "escalatingRemediations")
		if found && len(escalating) > 0 {
			report.WriteString("  - Escalating Remediations:\n")
			for i, rem := range escalating {
				remMap, ok := rem.(map[string]interface{})
				if !ok {
					continue
				}
				remRef, _ := remMap["remediationTemplate"].(map[string]interface{})
				kind, _ := remRef["kind"].(string)
				name, _ := remRef["name"].(string)
				ns, _ := remRef["namespace"].(string)
				timeout, _ := remMap["timeout"].(string)

				location := name
				if ns != "" {
					location = fmt.Sprintf("%s/%s", ns, name)
				}
				fmt.Fprintf(report, "    %d. %s/%s", i+1, kind, location)
				if timeout != "" {
					fmt.Fprintf(report, " (timeout=%s)", timeout)
				}
				report.WriteString("\n")
			}
		}

		unhealthyConditions, found, _ := unstructured.NestedSlice(nhc.Object, "spec", "unhealthyConditions")
		if found && len(unhealthyConditions) > 0 {
			report.WriteString("  - Unhealthy Conditions:\n")
			for _, uc := range unhealthyConditions {
				ucMap, ok := uc.(map[string]interface{})
				if !ok {
					continue
				}
				ucType, _ := ucMap["type"].(string)
				ucStatus, _ := ucMap["status"].(string)
				ucDuration, _ := ucMap["duration"].(string)
				fmt.Fprintf(report, "    - %s=%s (duration=%s)\n", ucType, ucStatus, ucDuration)
			}
		}
	}

	report.WriteString("\n")
	return issues
}

func listBareMetalHosts(ctx context.Context, client dynamic.Interface, namespace string) ([]unstructured.Unstructured, error) {
	var list *unstructured.UnstructuredList
	var err error

	if namespace != "" {
		list, err = client.Resource(BareMetalHostGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	} else {
		list, err = client.Resource(BareMetalHostGVR).List(ctx, metav1.ListOptions{})
	}

	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func checkCredentials(ctx context.Context, client corev1.CoreV1Interface, namespace, secretName, hostName string) []string {
	var issues []string

	secret, err := client.Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		issues = append(issues, fmt.Sprintf("%s/%s: credentials secret %q not found — %v", namespace, hostName, secretName, err))
		return issues
	}

	if v, ok := secret.Data["username"]; !ok || len(v) == 0 {
		issues = append(issues, fmt.Sprintf("%s/%s: credentials secret %q missing or empty 'username' key", namespace, hostName, secretName))
	}
	if v, ok := secret.Data["password"]; !ok || len(v) == 0 {
		issues = append(issues, fmt.Sprintf("%s/%s: credentials secret %q missing or empty 'password' key", namespace, hostName, secretName))
	}

	return issues
}

func checkNodeHealth(ctx context.Context, client corev1.CoreV1Interface, hostName string) []string {
	var issues []string

	node, err := client.Nodes().Get(ctx, hostName, metav1.GetOptions{})
	if err != nil {
		slog.Debug("could not find Node matching BareMetalHost", "host", hostName, "error", err)
		issues = append(issues, fmt.Sprintf("%s: no matching Node resource found", hostName))
		return issues
	}

	for _, condition := range node.Status.Conditions {
		switch condition.Type {
		case "Ready":
			if condition.Status != "True" {
				issues = append(issues, fmt.Sprintf("node %s: NotReady — %s", hostName, condition.Message))
			}
		case "MemoryPressure", "DiskPressure", "PIDPressure":
			if condition.Status == "True" {
				issues = append(issues, fmt.Sprintf("node %s: %s — %s", hostName, condition.Type, condition.Message))
			}
		}
	}

	return issues
}

// ValueOrNA returns s if non-empty, otherwise "N/A".
func ValueOrNA(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}

func isCRDNotInstalled(err error) bool {
	if apierrors.IsNotFound(err) || apimeta.IsNoMatchError(err) {
		return true
	}
	var ve *api.ValidationError
	return errors.As(err, &ve) && ve.Code == api.ErrorCodeResourceNotFound
}
