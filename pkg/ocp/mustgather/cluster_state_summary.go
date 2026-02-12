package mustgather

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/dynamic"
)

// criticalOperators is a list of operators that are essential for cluster operation.
// If these operators are missing, a warning is raised as they should exist on every
// supported OpenShift cluster variant and platform.
var criticalOperators = []string{
	"kube-apiserver",
	"etcd",
	"kube-controller-manager",
	"network",
	"config-operator",
	"machine-config",
}

// GetClusterStateSummary retrieves and formats a human-readable summary of the OpenShift cluster state.
// It includes ClusterID, client/cluster versions, and cluster operator health status.
// This is useful for opening support cases, bugzilla reports, or issues.
func GetClusterStateSummary(ctx context.Context, k api.KubernetesClient) (string, error) {
	dynamicClient := k.DynamicClient()
	var result strings.Builder

	result.WriteString("When opening a support case, bugzilla, or issue please include the following summary data along with any other requested information:\n\n")

	// Get ClusterVersion
	clusterVersionGVR := schema.GroupVersionResource{
		Group:    "config.openshift.io",
		Version:  "v1",
		Resource: "clusterversions",
	}

	clusterVersionUnstructured, err := dynamicClient.Resource(clusterVersionGVR).Get(ctx, "version", metav1.GetOptions{})
	if err != nil {
		result.WriteString(fmt.Sprintf("error getting cluster version: %v\n", err))
	}

	if clusterVersionUnstructured != nil {
		clusterID := getClusterID(clusterVersionUnstructured)
		if clusterID != "" {
			result.WriteString(fmt.Sprintf("ClusterID: %v\n", clusterID))
		}
	}

	clientVersion := humanSummaryForClientVersion()
	result.WriteString(fmt.Sprintf("ClientVersion: %v\n", clientVersion))
	result.WriteString(fmt.Sprintf("ClusterVersion: %v\n", humanSummaryForClusterVersion(clusterVersionUnstructured)))

	// List ClusterOperators dynamically
	clusterOperatorSummary, err := getClusterOperatorsSummary(ctx, dynamicClient)
	if err != nil {
		result.WriteString(fmt.Sprintf("error getting cluster operators: %v\n", err))
	} else {
		result.WriteString("ClusterOperators:\n")
		result.WriteString(clusterOperatorSummary + "\n")
	}

	result.WriteString("\n")

	return result.String(), nil
}

func getClusterOperatorsSummary(ctx context.Context, dynamicClient dynamic.Interface) (string, error) {
	clusterOperatorGVR := schema.GroupVersionResource{
		Group:    "config.openshift.io",
		Version:  "v1",
		Resource: "clusteroperators",
	}

	operatorList, err := dynamicClient.Resource(clusterOperatorGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	if operatorList == nil || len(operatorList.Items) == 0 {
		return "\tclusteroperators are missing", nil
	}

	// Track which critical operators are present
	missingCriticalOperators := sets.NewString(criticalOperators...)
	operatorStrings := []string{}

	for _, operator := range operatorList.Items {
		name := operator.GetName()
		missingCriticalOperators.Delete(name)

		operatorSummary := humanSummaryForClusterOperator(&operator)
		if len(operatorSummary) == 0 { // not noteworthy (healthy)
			continue
		}
		operatorStrings = append(operatorStrings, operatorSummary)
	}

	// Add warnings for missing critical operators
	missingList := missingCriticalOperators.List()
	sort.Strings(missingList)
	for _, missingOperator := range missingList {
		operatorStrings = append(operatorStrings, fmt.Sprintf("WARNING: clusteroperator/%s is missing (critical operator)", missingOperator))
	}

	if len(operatorStrings) > 0 {
		return "\t" + strings.Join(operatorStrings, "\n\t"), nil
	}
	return "\tAll healthy and stable", nil
}

func humanSummaryForClusterOperator(operator *unstructured.Unstructured) string {
	name := operator.GetName()

	conditions, found, err := unstructured.NestedSlice(operator.Object, "status", "conditions")
	if err != nil || !found {
		return fmt.Sprintf("clusteroperator/%s has no status conditions", name)
	}

	// Parse conditions
	available := false
	progressing := true // default to true (unknown)
	degraded := true    // default to true (unknown)
	upgradeable := true // default to true (unknown)

	availableMessage := "<missing>"
	progressingMessage := "<missing>"
	degradedMessage := "<missing>"
	upgradeableMessage := "<missing>"

	for _, c := range conditions {
		condition, ok := c.(map[string]any)
		if !ok {
			continue
		}

		condType, _, _ := unstructured.NestedString(condition, "type")
		status, _, _ := unstructured.NestedString(condition, "status")
		message, _, _ := unstructured.NestedString(condition, "message")

		switch condType {
		case "Available":
			available = status == "True"
			availableMessage = message
		case "Progressing":
			progressing = status != "False"
			progressingMessage = message
		case "Degraded":
			degraded = status != "False"
			degradedMessage = message
		case "Upgradeable":
			upgradeable = status != "False"
			upgradeableMessage = message
		}
	}

	switch {
	case !available:
		return fmt.Sprintf("clusteroperator/%s is not available (%v) because %v", name, availableMessage, degradedMessage)
	case degraded:
		return fmt.Sprintf("clusteroperator/%s is degraded because %v", name, degradedMessage)
	case !upgradeable:
		return fmt.Sprintf("clusteroperator/%s is not upgradeable because %v", name, upgradeableMessage)
	case progressing:
		return fmt.Sprintf("clusteroperator/%s is progressing: %v", name, progressingMessage)
	case available && !progressing && !degraded && upgradeable:
		return "" // healthy, not noteworthy
	default:
		return fmt.Sprintf("clusteroperator/%s is in an edge case", name)
	}
}

func humanSummaryForClusterVersion(clusterVersion *unstructured.Unstructured) string {
	if clusterVersion == nil {
		return "Cluster is in a version state we don't recognize"
	}

	history, found, err := unstructured.NestedSlice(clusterVersion.Object, "status", "history")
	if err != nil || !found || len(history) == 0 {
		return "Cluster is in a version state we don't recognize"
	}

	// Get desired version
	desiredVersion, _, _ := unstructured.NestedString(clusterVersion.Object, "status", "desired", "version")

	// Parse first history entry
	firstHistory, ok := history[0].(map[string]any)
	if !ok {
		return "Cluster is in a version state we don't recognize"
	}

	firstHistoryState, _, _ := unstructured.NestedString(firstHistory, "state")
	firstHistoryVersion, _, _ := unstructured.NestedString(firstHistory, "version")
	startedTimeStr, _, _ := unstructured.NestedString(firstHistory, "startedTime")

	isInstalling := len(history) == 1 && firstHistoryState != "Completed"
	isUpdating := len(history) > 1 && firstHistoryState != "Completed"
	isStable := firstHistoryState == "Completed"

	sinceDuration := ""
	if startedTimeStr != "" {
		startedTime, err := time.Parse(time.RFC3339, startedTimeStr)
		if err == nil {
			sinceDuration = startedTime.String()
		}
	}

	progressingConditionMessage := "<unknown>"
	conditions, found, _ := unstructured.NestedSlice(clusterVersion.Object, "status", "conditions")
	if found {
		for _, c := range conditions {
			condition, ok := c.(map[string]any)
			if !ok {
				continue
			}
			condType, _, _ := unstructured.NestedString(condition, "type")
			if condType == "Progressing" {
				progressingConditionMessage, _, _ = unstructured.NestedString(condition, "message")
				break
			}
		}
	}

	switch {
	case isInstalling:
		return fmt.Sprintf("Installing %q for %v: %v", sinceDuration,
			desiredVersion, startedTimeStr, progressingConditionMessage)
	case isUpdating:
		previousVersion := ""
		if len(history) > 1 {
			if prevHistory, ok := history[1].(map[string]any); ok {
				previousVersion, _, _ = unstructured.NestedString(prevHistory, "version")
			}
		}
		return fmt.Sprintf("Updating to %q from %q for %v: %v",
			desiredVersion, previousVersion, sinceDuration, progressingConditionMessage)
	case isStable:
		return fmt.Sprintf("Stable at %q", firstHistoryVersion)
	default:
		return "Unknown state"
	}
}

func getClusterID(clusterVersion *unstructured.Unstructured) string {
	clusterID, _, _ := unstructured.NestedString(clusterVersion.Object, "spec", "clusterID")
	return clusterID
}

func humanSummaryForClientVersion() string {
	return version.Version
}
