package lvms

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
)

var (
	csiStorageCapacityGVR = schema.GroupVersionResource{
		Group:    "storage.k8s.io",
		Version:  "v1",
		Resource: "csistoragecapacities",
	}
	pvGVR = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "persistentvolumes",
	}
)

// initLVMSCapacity initializes the LVMS capacity monitoring prompt
func initLVMSCapacity() []api.ServerPrompt {
	return []api.ServerPrompt{
		{
			Prompt: api.Prompt{
				Name:        "lvms-capacity",
				Title:       "LVMS Capacity Check",
				Description: "Check LVMS storage capacity across all nodes including available space per node, PVC usage, and capacity warnings",
				Arguments: []api.PromptArgument{
					{
						Name:        "namespace",
						Description: "The LVMS namespace (default: openshift-lvm-storage)",
						Required:    false,
					},
				},
			},
			Handler: lvmsCapacityHandler,
		},
	}
}

// lvmsCapacityHandler implements the LVMS capacity monitoring prompt
func lvmsCapacityHandler(params api.PromptHandlerParams) (*api.PromptCallResult, error) {
	args := params.GetArguments()
	namespace := args["namespace"]
	if namespace == "" {
		namespace = DefaultLVMSNamespace
	}

	ctx := params.Context
	if ctx == nil {
		ctx = context.Background()
	}

	dynamicClient := params.DynamicClient()

	// Fetch all capacity data
	capacitySummary := fetchCSICapacity(ctx, dynamicClient, namespace)
	pvcSummary := fetchPVCCapacitySummary(ctx, dynamicClient)
	pvSummary := fetchPVCapacitySummary(ctx, dynamicClient)

	guideText := fmt.Sprintf(`# LVMS Capacity Report

## Overview

This report shows LVMS storage capacity across all nodes to help with capacity planning
and identify nodes that may need additional storage.

---

## Available Capacity by Node

CSIStorageCapacity reports how much space TopoLVM can provision on each node:

%s

---

## PVC Usage Summary

PVCs using TopoLVM storage classes:

%s

---

## PV Capacity Summary

Provisioned PersistentVolumes by TopoLVM:

%s

---

## Capacity Analysis

Based on the data above, assess:

1. **Available Headroom**: How much free space remains per node?
2. **Usage Distribution**: Is storage evenly distributed or concentrated on few nodes?
3. **Growth Trend**: Are PVCs consuming space faster than expected?
4. **At-Risk Nodes**: Any nodes with less than 20%% free capacity?

---

## Recommendations

- **If a node has < 20%% free**: Consider adding disks to LVMCluster deviceSelector
- **If PVCs are pending**: Check if any node has sufficient capacity
- **For capacity planning**: Total available = sum of all node capacities
- **For HA workloads**: Ensure multiple nodes have sufficient capacity

---

## Report Summary

Provide:
- **Overall Status**: OK / Low Capacity / Critical
- **Total Available**: Sum across all nodes
- **Total Used**: Sum of all PV capacities
- **Utilization**: Used / (Used + Available) %%
- **Action Items**: Any immediate concerns
`, capacitySummary, pvcSummary, pvSummary)

	return api.NewPromptCallResult(
		"LVMS capacity report generated",
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
					Text: "I'll analyze the LVMS capacity data to assess storage health and identify any capacity concerns.",
				},
			},
		},
		nil,
	), nil
}

// fetchCSICapacity fetches CSIStorageCapacity resources for TopoLVM
func fetchCSICapacity(ctx context.Context, dynamicClient dynamic.Interface, namespace string) string {
	var result strings.Builder

	capacities, err := dynamicClient.Resource(csiStorageCapacityGVR).Namespace(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "csi.storage.k8s.io/drivername=topolvm.io",
	})
	if err != nil {
		return fmt.Sprintf("*Error listing CSIStorageCapacity: %v*\n", err)
	}

	if len(capacities.Items) == 0 {
		return "*No CSIStorageCapacity found for TopoLVM — LVMS may not be configured*\n"
	}

	result.WriteString("| Node | Storage Class | Available Capacity |\n")
	result.WriteString("|------|---------------|--------------------|\n")

	var totalCapacity int64
	for _, cap := range capacities.Items {
		storageClass, _, _ := unstructured.NestedString(cap.Object, "storageClassName")
		capacity, _, _ := unstructured.NestedString(cap.Object, "capacity")

		// Get node from topology
		node := "unknown"
		if topology, found, _ := unstructured.NestedMap(cap.Object, "nodeTopology", "matchLabels"); found {
			if n, ok := topology["topology.topolvm.io/node"].(string); ok {
				node = n
			}
		}

		// Parse capacity for total
		totalCapacity += parseCapacity(capacity)

		fmt.Fprintf(&result, "| %s | %s | %s |\n", node, storageClass, capacity)
	}

	fmt.Fprintf(&result, "\n**Total Available: %s**\n", formatCapacity(totalCapacity))

	return result.String()
}

// fetchPVCCapacitySummary fetches PVC summary for TopoLVM storage classes
func fetchPVCCapacitySummary(ctx context.Context, dynamicClient dynamic.Interface) string {
	var result strings.Builder

	// Get TopoLVM storage classes first
	scList, err := dynamicClient.Resource(storageClassGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Sprintf("*Error listing StorageClasses: %v*\n", err)
	}

	topoLVMSCs := make(map[string]bool)
	for _, sc := range scList.Items {
		provisioner, _, _ := unstructured.NestedString(sc.Object, "provisioner")
		if strings.Contains(provisioner, "topolvm") {
			topoLVMSCs[sc.GetName()] = true
		}
	}

	if len(topoLVMSCs) == 0 {
		return "*No TopoLVM storage classes found*\n"
	}

	// Get PVCs
	pvcList, err := dynamicClient.Resource(pvcGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Sprintf("*Error listing PVCs: %v*\n", err)
	}

	var boundCount, pendingCount int
	var totalRequested int64
	pendingPVCs := []string{}

	for _, pvc := range pvcList.Items {
		scName, _, _ := unstructured.NestedString(pvc.Object, "spec", "storageClassName")
		if !topoLVMSCs[scName] {
			continue
		}

		phase, _, _ := unstructured.NestedString(pvc.Object, "status", "phase")
		requestedStorage, _, _ := unstructured.NestedString(pvc.Object, "spec", "resources", "requests", "storage")

		switch phase {
		case "Bound":
			boundCount++
			totalRequested += parseCapacity(requestedStorage)
		case "Pending":
			pendingCount++
			pendingPVCs = append(pendingPVCs, fmt.Sprintf("%s/%s (%s)",
				pvc.GetNamespace(), pvc.GetName(), requestedStorage))
		}
	}

	fmt.Fprintf(&result, "- **Bound PVCs**: %d (Total requested: %s)\n", boundCount, formatCapacity(totalRequested))
	fmt.Fprintf(&result, "- **Pending PVCs**: %d\n", pendingCount)

	if len(pendingPVCs) > 0 {
		result.WriteString("\n**Pending PVCs (need attention):**\n")
		for _, pvc := range pendingPVCs {
			fmt.Fprintf(&result, "- %s\n", pvc)
		}
	}

	return result.String()
}

// fetchPVCapacitySummary fetches PV summary for TopoLVM
func fetchPVCapacitySummary(ctx context.Context, dynamicClient dynamic.Interface) string {
	var result strings.Builder

	pvList, err := dynamicClient.Resource(pvGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Sprintf("*Error listing PVs: %v*\n", err)
	}

	var topoLVMCount int
	var totalCapacity int64
	nodeCapacity := make(map[string]int64)

	for _, pv := range pvList.Items {
		csiDriver, _, _ := unstructured.NestedString(pv.Object, "spec", "csi", "driver")
		if csiDriver != "topolvm.io" {
			continue
		}

		topoLVMCount++
		capacity, _, _ := unstructured.NestedString(pv.Object, "spec", "capacity", "storage")
		capBytes := parseCapacity(capacity)
		totalCapacity += capBytes

		// Get node affinity
		terms, found, _ := unstructured.NestedSlice(pv.Object, "spec", "nodeAffinity", "required", "nodeSelectorTerms")
		if found && len(terms) > 0 {
			if term, ok := terms[0].(map[string]any); ok {
				if exprs, ok := term["matchExpressions"].([]any); ok {
					for _, expr := range exprs {
						if e, ok := expr.(map[string]any); ok {
							if e["key"] == "topology.topolvm.io/node" {
								if values, ok := e["values"].([]any); ok && len(values) > 0 {
									if node, ok := values[0].(string); ok {
										nodeCapacity[node] += capBytes
									}
								}
							}
						}
					}
				}
			}
		}
	}

	fmt.Fprintf(&result, "- **Total PVs**: %d\n", topoLVMCount)
	fmt.Fprintf(&result, "- **Total Provisioned**: %s\n", formatCapacity(totalCapacity))

	if len(nodeCapacity) > 0 {
		result.WriteString("\n**Provisioned by Node:**\n\n")
		result.WriteString("| Node | Provisioned |\n")
		result.WriteString("|------|-------------|\n")
		for node, cap := range nodeCapacity {
			fmt.Fprintf(&result, "| %s | %s |\n", node, formatCapacity(cap))
		}
	}

	return result.String()
}

// parseCapacity parses a Kubernetes quantity string to bytes
func parseCapacity(s string) int64 {
	if s == "" {
		return 0
	}
	q, err := resource.ParseQuantity(s)
	if err != nil {
		return 0
	}
	return q.Value()
}

// formatCapacity formats bytes to human readable string
func formatCapacity(bytes int64) string {
	if bytes == 0 {
		return "0"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
