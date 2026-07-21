package lvms

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/containers/kubernetes-mcp-server/pkg/output"
)

const (
	// DefaultLVMSNamespace is the default namespace for LVMS components
	DefaultLVMSNamespace = "openshift-lvm-storage"
)

// GVRs for LVMS resources
var (
	lvmClusterGVR = schema.GroupVersionResource{
		Group:    "lvm.topolvm.io",
		Version:  "v1alpha1",
		Resource: "lvmclusters",
	}
	lvmVolumeGroupNodeStatusGVR = schema.GroupVersionResource{
		Group:    "lvm.topolvm.io",
		Version:  "v1alpha1",
		Resource: "lvmvolumegroupnodestatuses",
	}
	podGVR = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "pods",
	}
	csvGVR = schema.GroupVersionResource{
		Group:    "operators.coreos.com",
		Version:  "v1alpha1",
		Resource: "clusterserviceversions",
	}
	storageClassGVR = schema.GroupVersionResource{
		Group:    "storage.k8s.io",
		Version:  "v1",
		Resource: "storageclasses",
	}
	pvcGVR = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "persistentvolumeclaims",
	}
)

// initLVMSTroubleshoot initializes the LVMS troubleshooting prompt
func initLVMSTroubleshoot() []api.ServerPrompt {
	return []api.ServerPrompt{
		{
			Prompt: api.Prompt{
				Name:        "lvms-troubleshoot",
				Title:       "LVMS Troubleshoot",
				Description: "Diagnose LVMS storage issues by gathering LVMCluster status, volume group health, and node-level LVM data with interpretation of domain-specific fields",
				Arguments: []api.PromptArgument{
					{
						Name:        "namespace",
						Description: "The LVMS namespace (default: openshift-lvm-storage)",
						Required:    false,
					},
					{
						Name:        "node",
						Description: "Specific node to inspect (default: all nodes with LVMS)",
						Required:    false,
					},
				},
			},
			Handler: lvmsTroubleshootHandler,
		},
	}
}

// lvmsTroubleshootHandler implements the LVMS troubleshooting prompt
func lvmsTroubleshootHandler(params api.PromptHandlerParams) (*api.PromptCallResult, error) {
	args := params.GetArguments()
	namespace := args["namespace"]
	if namespace == "" {
		namespace = DefaultLVMSNamespace
	}
	targetNode := args["node"]

	ctx := params.Context
	if ctx == nil {
		ctx = context.Background()
	}

	dynamicClient := params.DynamicClient()

	// Fetch all relevant LVMS resources
	operatorHealthYaml := fetchOperatorHealth(ctx, dynamicClient, namespace)
	lvmClusterYaml := fetchLVMClusterStatus(ctx, dynamicClient, namespace)
	vgNodeStatusYaml := fetchVolumeGroupNodeStatus(ctx, dynamicClient, namespace, targetNode)
	storageClassYaml := fetchTopoLVMStorageClasses(ctx, dynamicClient)
	pvcStatusYaml := fetchTopoLVMPVCs(ctx, dynamicClient)
	vgManagerPodsYaml, podNames := fetchVGManagerPods(ctx, dynamicClient, namespace)
	vgManagerLogsText := fetchVGManagerLogs(ctx, params.KubernetesClient, namespace, podNames)
	eventsYaml := fetchLVMSEvents(ctx, params.KubernetesClient, namespace)

	guideText := fmt.Sprintf(`# LVMS Troubleshooting Guide

## Namespace: %s

Use this guide to diagnose LVMS storage issues. The relevant resource data and LVM-specific field interpretations are provided below.

---

## LVM Attribute Field Reference

Understanding these fields is critical for LVMS troubleshooting:

### Volume Group Attributes (vg_attr)
The vg_attr field is a 6-character string where each position has meaning:

| Position | Character | Meaning |
|----------|-----------|---------|
| 1 | w/r | Writeable / Read-only |
| 2 | z/- | Resizeable / Fixed |
| 3 | x/- | Exported / Normal |
| 4 | p/- | Partial (missing PVs) / Complete |
| 5 | c/l/n/s | Allocation: contiguous/cling/normal/anywhere |
| 6 | c/- | Clustered / Local |

**Healthy VG:** "wz--n-" (writeable, resizeable, not partial, normal allocation)
**Degraded VG:** "wz-pn-" (the 'p' in position 4 means missing PVs!)

### Logical Volume Attributes (lv_attr)
The lv_attr field is a 10-character string. Key positions for LVMS:

| Position | Key Values | Meaning |
|----------|------------|---------|
| 1 | V/v/t | Virtual (thin), thin pool, or other type |
| 5 | a/- | Active / Inactive |
| 9 | p/- | Partial activation (RAID degraded!) / Normal |

**Healthy LV:** Position 9 is '-'
**Degraded RAID LV:** Position 9 is 'p' (partial)

### Key Diagnostic Fields
- **vg_missing_pv_count > 0**: Volume group has missing physical volumes (disk failure)
- **vg_free**: Available space in volume group
- **lv_health_status**: "partial" indicates RAID degradation
- **raid_sync_percent**: < 100 means RAID is rebuilding

---

## Step 1: Operator Health

Check LVMS operator status:
- CSV should be "Succeeded"
- All operator pods should be "Running"

%s

---

## Step 2: LVMCluster Status

The LVMCluster CR is the primary configuration. Check:
- Does an LVMCluster exist?
- What is the state? ("Ready" = healthy)
- Are all deviceClasses configured correctly?
- **Device paths**: Use /dev/disk/by-id/ not /dev/sdX (sdX paths change on reboot!)

%s

---

## Step 3: Volume Group Node Status

LVMVolumeGroupNodeStatus shows per-node volume group health:
- Check status for each node
- Look for "Degraded" or "Failed" states
- Review raidStatus if RAID is configured

%s

---

## Step 4: Storage Classes

TopoLVM storage classes provisioned by LVMS:

%s

---

## Step 5: PVC Status

PVCs using TopoLVM provisioner:
- Pending PVCs indicate capacity or configuration issues
- Check events on Pending PVCs for root cause

%s

---

## Step 6: vg-manager Pod Health

The vg-manager pods manage volume groups on each node:
- Pods should be "Running" with all containers ready
- CrashLoopBackOff indicates disk discovery issues
- High restart counts suggest persistent problems

%s

---

## Step 7: vg-manager Pod Logs

Review logs for errors:
- Look for disk discovery failures
- Check for LVM command errors
- Review any "error" or "failed" messages

%s

---

## Step 8: Events

Review events in the LVMS namespace:
- Warning events indicate problems
- Look for scheduling, disk, or permission issues

%s

---

## Troubleshooting Analysis

Based on the data above, analyze:

1. **LVMCluster State**: Is it "Ready"? If not, what deviceClass is failing?
2. **Node Status**: Are all nodes healthy? Look for "Degraded" or "Failed" states in LVMVolumeGroupNodeStatus.
3. **vg-manager Pods**: Are they running? Check for CrashLoopBackOff or high restart counts.
4. **Events**: Are there Warning events indicating problems?

---

## Common Issues and Recovery Procedures

### 1. VG has missing PVs (vg_attr contains 'p')
Disk failure detected. Recovery options:
- **Option A - Reduce VG**: Remove failed disk from VG
  `+"`"+`vgchange --activate n <VG_NAME>`+"`"+`
  `+"`"+`vgreduce --removemissing --force <VG_NAME>`+"`"+`
  Then patch LVMCluster to remove failed device path.

- **Option B - Replace disk**: If replacement disk available
  `+"`"+`pvcreate --restorefile /etc/lvm/backup/<VG_NAME> --uuid <ORIGINAL_UUID> /dev/<new_device>`+"`"+`
  `+"`"+`vgextend --restoremissing <VG_NAME> /dev/<new_device>`+"`"+`

### 2. Thin pool nearly full (data_percent > 80%%)
- Extend the thin pool or add disks
- Delete unused PVCs to free space
- Check for stuck snapshots consuming space

### 3. vg-manager CrashLoopBackOff
- Check if deviceSelector.paths point to valid disks
- Run `+"`"+`lsblk --paths --json`+"`"+` on the node to verify disk availability
- Ensure disks are not in use by other systems

### 4. RAID degraded (lv_health_status: partial)
- Check raid_sync_percent - if syncing, wait for completion
- If stuck at 0%%, the RAID may need repair:
  `+"`"+`lvconvert --repair <VG_NAME>/<LV_NAME>`+"`"+`

### 5. LVMCluster deletion stuck
Perform forced cleanup (in order):
1. Delete all PVCs using LVMS
2. Remove finalizers from LogicalVolumes, then delete them
3. Remove finalizers from LVMVolumeGroups, then delete them
4. Remove finalizers from LVMVolumeGroupNodeStatus, then delete them
5. Delete LVMCluster

### 6. PVC stuck in Pending
- Check if StorageClass exists
- Verify LVMCluster is Ready
- Check node has available capacity
- If using WaitForFirstConsumer, PVC stays Pending until a Pod requests it (this is normal)

---

## Report Findings

After analysis, report:
- **Status:** Healthy / Degraded / Failed
- **Root Cause:** Description of the issue found (or "None found")
- **Affected Nodes:** List of nodes with problems
- **Recommended Action:** Steps to resolve the issue
`, namespace, operatorHealthYaml, lvmClusterYaml, vgNodeStatusYaml, storageClassYaml, pvcStatusYaml, vgManagerPodsYaml, vgManagerLogsText, eventsYaml)

	return api.NewPromptCallResult(
		"LVMS troubleshooting guide generated",
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
					Text: "I'll analyze the collected LVMS data to diagnose storage issues, paying special attention to vg_attr flags and node-level health indicators.",
				},
			},
		},
		nil,
	), nil
}

// fetchLVMClusterStatus fetches LVMCluster resources and returns their status
func fetchLVMClusterStatus(ctx context.Context, dynamicClient dynamic.Interface, namespace string) string {
	clusters, err := dynamicClient.Resource(lvmClusterGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Sprintf("### LVMCluster\n\n*Error listing LVMClusters: %v*", err)
	}

	if len(clusters.Items) == 0 {
		return "### LVMCluster\n\n*No LVMCluster found — LVMS may not be configured*"
	}

	var result strings.Builder
	result.WriteString("### LVMCluster\n\n")

	for _, cluster := range clusters.Items {
		fmt.Fprintf(&result, "#### %s\n\n", cluster.GetName())

		// Get overall state
		state, _, _ := unstructured.NestedString(cluster.Object, "status", "state")
		fmt.Fprintf(&result, "**State:** %s\n\n", valueOrNA(state))

		// Get deviceClassStatuses
		deviceStatuses, found, _ := unstructured.NestedSlice(cluster.Object, "status", "deviceClassStatuses")
		if found && len(deviceStatuses) > 0 {
			result.WriteString("**Device Class Statuses:**\n\n")
			result.WriteString("| Name | Node | Status |\n")
			result.WriteString("|------|------|--------|\n")

			for _, ds := range deviceStatuses {
				dsMap, ok := ds.(map[string]any)
				if !ok {
					continue
				}
				name, _ := dsMap["name"].(string)
				nodeStatuses, _ := dsMap["nodeStatus"].([]any)
				for _, ns := range nodeStatuses {
					nsMap, ok := ns.(map[string]any)
					if !ok {
						continue
					}
					nodeName, _ := nsMap["node"].(string)
					status, _ := nsMap["status"].(string)
					fmt.Fprintf(&result, "| %s | %s | %s |\n", name, nodeName, status)
				}
			}
			result.WriteString("\n")
		}

		// Get spec for device configuration
		spec, found, _ := unstructured.NestedMap(cluster.Object, "spec", "storage")
		if found {
			specYaml, err := output.MarshalYaml(spec)
			if err == nil {
				fmt.Fprintf(&result, "**Storage Spec:**\n```yaml\n%s```\n\n", specYaml)
			}
		}
	}

	return result.String()
}

// fetchVolumeGroupNodeStatus fetches LVMVolumeGroupNodeStatus resources
func fetchVolumeGroupNodeStatus(ctx context.Context, dynamicClient dynamic.Interface, namespace, targetNode string) string {
	statuses, err := dynamicClient.Resource(lvmVolumeGroupNodeStatusGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Sprintf("### LVMVolumeGroupNodeStatus\n\n*Error listing node statuses: %v*", err)
	}

	if len(statuses.Items) == 0 {
		return "### LVMVolumeGroupNodeStatus\n\n*No LVMVolumeGroupNodeStatus found*"
	}

	var result strings.Builder
	result.WriteString("### LVMVolumeGroupNodeStatus\n\n")

	for _, status := range statuses.Items {
		nodeName := status.GetName()

		// Filter by target node if specified
		if targetNode != "" && nodeName != targetNode {
			continue
		}

		fmt.Fprintf(&result, "#### Node: %s\n\n", nodeName)

		// Get spec which contains the actual status data
		spec, found, _ := unstructured.NestedMap(status.Object, "spec")
		if !found {
			result.WriteString("*No spec found*\n\n")
			continue
		}

		specYaml, err := output.MarshalYaml(spec)
		if err == nil {
			fmt.Fprintf(&result, "```yaml\n%s```\n\n", specYaml)
		}
	}

	return result.String()
}

// fetchVGManagerPods fetches vg-manager pods
func fetchVGManagerPods(ctx context.Context, dynamicClient dynamic.Interface, namespace string) (string, []string) {
	pods, err := dynamicClient.Resource(podGVR).Namespace(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=vg-manager",
	})
	if err != nil {
		return fmt.Sprintf("### vg-manager Pods\n\n*Error listing pods: %v*", err), nil
	}

	if len(pods.Items) == 0 {
		return "### vg-manager Pods\n\n*No vg-manager pods found — LVMS may not be installed*", nil
	}

	var result strings.Builder
	var podNames []string
	result.WriteString("### vg-manager Pods\n\n")
	result.WriteString("| Name | Node | Phase | Ready | Restarts |\n")
	result.WriteString("|------|------|-------|-------|----------|\n")

	for _, pod := range pods.Items {
		podNames = append(podNames, pod.GetName())
		phase, _, _ := unstructured.NestedString(pod.Object, "status", "phase")
		nodeName, _, _ := unstructured.NestedString(pod.Object, "spec", "nodeName")
		ready, restarts := getPodReadyAndRestarts(&pod)

		fmt.Fprintf(&result, "| %s | %s | %s | %s | %d |\n",
			pod.GetName(), valueOrNA(nodeName), valueOrNA(phase), ready, restarts)
	}
	result.WriteString("\n")

	return result.String(), podNames
}

// fetchVGManagerLogs fetches logs from vg-manager pods
func fetchVGManagerLogs(ctx context.Context, client api.KubernetesClient, namespace string, podNames []string) string {
	if len(podNames) == 0 {
		return "### vg-manager Logs\n\n*No vg-manager pods found*"
	}

	core := kubernetes.NewCore(client)
	var result strings.Builder
	result.WriteString("### vg-manager Logs\n\n")

	for _, podName := range podNames {
		fmt.Fprintf(&result, "#### Pod: %s\n\n", podName)
		// Use empty container name to get logs from the default container
		logs, err := core.PodsLog(ctx, namespace, podName, "", false, 100)
		if err != nil {
			fmt.Fprintf(&result, "*Error fetching logs: %v*\n\n", err)
			continue
		}

		errorLines := filterLogLines(logs, []string{"error", "failed", "unable", "cannot", "no available"})
		if errorLines != "" {
			fmt.Fprintf(&result, "**Error lines (last 100 log lines):**\n```\n%s\n```\n\n", errorLines)
		} else {
			result.WriteString("*No error lines found in recent logs*\n\n")
		}
	}

	return result.String()
}

// fetchLVMSEvents fetches events in the LVMS namespace
func fetchLVMSEvents(ctx context.Context, client api.KubernetesClient, namespace string) string {
	core := kubernetes.NewCore(client)
	eventMap, err := core.EventsList(ctx, namespace, api.ListOptions{})
	if err != nil {
		return fmt.Sprintf("### Events\n\n*Error listing events: %v*", err)
	}

	if len(eventMap) == 0 {
		return "### Events\n\n*No events found in namespace*"
	}

	// Filter for warning events
	var warningEvents []map[string]any
	for _, event := range eventMap {
		eventType, _ := event["Type"].(string)
		if eventType == "Warning" {
			warningEvents = append(warningEvents, event)
		}
	}

	if len(warningEvents) == 0 {
		return "### Events\n\n*No Warning events found — this is good*"
	}

	// Limit to most recent 10
	if len(warningEvents) > 10 {
		warningEvents = warningEvents[len(warningEvents)-10:]
	}

	yamlStr, err := output.MarshalYaml(warningEvents)
	if err != nil {
		return fmt.Sprintf("### Events\n\n*Error marshaling events: %v*", err)
	}

	return fmt.Sprintf("### Warning Events (last 10)\n\n```yaml\n%s```", yamlStr)
}

// Helper functions

func valueOrNA(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}

func getPodReadyAndRestarts(pod *unstructured.Unstructured) (string, int64) {
	containerStatuses, found, _ := unstructured.NestedSlice(pod.Object, "status", "containerStatuses")
	if !found {
		return "N/A", 0
	}

	totalContainers := len(containerStatuses)
	readyCount := 0
	var totalRestarts int64

	for _, cs := range containerStatuses {
		csMap, ok := cs.(map[string]any)
		if !ok {
			continue
		}
		if ready, ok := csMap["ready"].(bool); ok && ready {
			readyCount++
		}
		switch restarts := csMap["restartCount"].(type) {
		case int64:
			totalRestarts += restarts
		case float64:
			totalRestarts += int64(restarts)
		case json.Number:
			if r, err := restarts.Int64(); err == nil {
				totalRestarts += r
			}
		}
	}

	return fmt.Sprintf("%d/%d", readyCount, totalContainers), totalRestarts
}

func filterLogLines(logs string, keywords []string) string {
	var matched []string
	for _, line := range strings.Split(logs, "\n") {
		lower := strings.ToLower(line)
		for _, kw := range keywords {
			if strings.Contains(lower, strings.ToLower(kw)) {
				matched = append(matched, line)
				break
			}
		}
	}
	return strings.Join(matched, "\n")
}

// fetchOperatorHealth fetches LVMS operator CSV and pod status
func fetchOperatorHealth(ctx context.Context, dynamicClient dynamic.Interface, namespace string) string {
	var result strings.Builder
	result.WriteString("### Operator Health\n\n")

	// Get CSV status
	csvList, err := dynamicClient.Resource(csvGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(&result, "*Error listing CSVs: %v*\n\n", err)
	} else {
		var lvmsCSV *unstructured.Unstructured
		for i := range csvList.Items {
			csv := &csvList.Items[i]
			name := csv.GetName()
			if strings.HasPrefix(name, "lvms-operator") || strings.HasPrefix(name, "lvm-operator") {
				lvmsCSV = csv
				break
			}
		}
		if lvmsCSV == nil {
			result.WriteString("*No LVMS operator CSV found*\n\n")
		} else {
			phase, _, _ := unstructured.NestedString(lvmsCSV.Object, "status", "phase")
			reason, _, _ := unstructured.NestedString(lvmsCSV.Object, "status", "reason")
			fmt.Fprintf(&result, "**CSV:** %s\n- **Phase:** %s\n- **Reason:** %s\n\n",
				lvmsCSV.GetName(), valueOrNA(phase), valueOrNA(reason))
		}
	}

	// Get operator pods
	labelSelector := "app.kubernetes.io/name=lvms-operator"
	podList, err := dynamicClient.Resource(podGVR).Namespace(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		fmt.Fprintf(&result, "*Error listing operator pods: %v*\n\n", err)
	} else if len(podList.Items) == 0 {
		result.WriteString("*No LVMS operator pods found*\n\n")
	} else {
		result.WriteString("**Operator Pods:**\n\n")
		for _, pod := range podList.Items {
			phase, _, _ := unstructured.NestedString(pod.Object, "status", "phase")
			ready, restarts := getPodReadyAndRestarts(&pod)
			fmt.Fprintf(&result, "- %s: %s (Ready: %s, Restarts: %d)\n",
				pod.GetName(), phase, ready, restarts)
		}
		result.WriteString("\n")
	}

	return result.String()
}

// fetchTopoLVMStorageClasses fetches storage classes provisioned by TopoLVM
func fetchTopoLVMStorageClasses(ctx context.Context, dynamicClient dynamic.Interface) string {
	var result strings.Builder
	result.WriteString("### TopoLVM Storage Classes\n\n")

	scList, err := dynamicClient.Resource(storageClassGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(&result, "*Error listing StorageClasses: %v*\n\n", err)
		return result.String()
	}

	var topoLVMSCs []map[string]any
	for _, sc := range scList.Items {
		provisioner, _, _ := unstructured.NestedString(sc.Object, "provisioner")
		if strings.Contains(provisioner, "topolvm") {
			scData := map[string]any{
				"name":                 sc.GetName(),
				"provisioner":          provisioner,
				"reclaimPolicy":        sc.Object["reclaimPolicy"],
				"volumeBindingMode":    sc.Object["volumeBindingMode"],
				"allowVolumeExpansion": sc.Object["allowVolumeExpansion"],
			}
			if params, found, _ := unstructured.NestedStringMap(sc.Object, "parameters"); found {
				scData["parameters"] = params
			}
			topoLVMSCs = append(topoLVMSCs, scData)
		}
	}

	if len(topoLVMSCs) == 0 {
		result.WriteString("*No TopoLVM storage classes found*\n\n")
		return result.String()
	}

	yamlStr, err := output.MarshalYaml(topoLVMSCs)
	if err != nil {
		fmt.Fprintf(&result, "*Error marshaling storage classes: %v*\n\n", err)
		return result.String()
	}

	fmt.Fprintf(&result, "```yaml\n%s```\n\n", yamlStr)
	return result.String()
}

// fetchTopoLVMPVCs fetches PVCs using TopoLVM storage classes
func fetchTopoLVMPVCs(ctx context.Context, dynamicClient dynamic.Interface) string {
	var result strings.Builder
	result.WriteString("### PVCs Using TopoLVM\n\n")

	// First get TopoLVM storage class names
	scList, err := dynamicClient.Resource(storageClassGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(&result, "*Error listing StorageClasses: %v*\n\n", err)
		return result.String()
	}

	topoLVMSCNames := make(map[string]bool)
	for _, sc := range scList.Items {
		provisioner, _, _ := unstructured.NestedString(sc.Object, "provisioner")
		if strings.Contains(provisioner, "topolvm") {
			topoLVMSCNames[sc.GetName()] = true
		}
	}

	if len(topoLVMSCNames) == 0 {
		result.WriteString("*No TopoLVM storage classes found*\n\n")
		return result.String()
	}

	// Get all PVCs
	pvcList, err := dynamicClient.Resource(pvcGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(&result, "*Error listing PVCs: %v*\n\n", err)
		return result.String()
	}

	var pendingPVCs, boundPVCs []map[string]any
	for _, pvc := range pvcList.Items {
		scName, _, _ := unstructured.NestedString(pvc.Object, "spec", "storageClassName")
		if !topoLVMSCNames[scName] {
			continue
		}

		phase, _, _ := unstructured.NestedString(pvc.Object, "status", "phase")
		capacity, _, _ := unstructured.NestedString(pvc.Object, "status", "capacity", "storage")

		pvcData := map[string]any{
			"namespace":    pvc.GetNamespace(),
			"name":         pvc.GetName(),
			"storageClass": scName,
			"phase":        phase,
			"capacity":     capacity,
		}

		if phase == "Pending" {
			pendingPVCs = append(pendingPVCs, pvcData)
		} else {
			boundPVCs = append(boundPVCs, pvcData)
		}
	}

	if len(pendingPVCs) > 0 {
		result.WriteString("**Pending PVCs (needs attention):**\n\n")
		yamlStr, _ := output.MarshalYaml(pendingPVCs)
		fmt.Fprintf(&result, "```yaml\n%s```\n\n", yamlStr)
	}

	if len(boundPVCs) > 0 {
		// Limit bound PVCs to 20 most recent
		if len(boundPVCs) > 20 {
			boundPVCs = boundPVCs[len(boundPVCs)-20:]
			result.WriteString("**Bound PVCs (showing last 20):**\n\n")
		} else {
			result.WriteString("**Bound PVCs:**\n\n")
		}
		yamlStr, _ := output.MarshalYaml(boundPVCs)
		fmt.Fprintf(&result, "```yaml\n%s```\n\n", yamlStr)
	}

	if len(pendingPVCs) == 0 && len(boundPVCs) == 0 {
		result.WriteString("*No PVCs found using TopoLVM storage classes*\n\n")
	}

	return result.String()
}
