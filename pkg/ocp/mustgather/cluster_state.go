package mustgather

import (
	"fmt"
	"strings"
	"time"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	internalk8s "github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/containers/kubernetes-mcp-server/pkg/version"
	"github.com/docker/go-units"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/library-go/pkg/config/clusteroperator/v1helpers"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
)

// longExistingOperators is a list of operators that should be present on every variant of every platform and have
// existed for all supported version.  We use this to find when things are missing (like if they fail install).
var longExistingOperators = []string{
	"authentication",
	"cloud-credential",
	"cluster-autoscaler",
	"config-operator",
	"console",
	"dns",
	"etcd",
	"image-registry",
	"ingress",
	"insights",
	"kube-apiserver",
	"kube-controller-manager",
	"kube-scheduler",
	"kube-storage-version-migrator",
	"machine-api",
	"machine-approver",
	"machine-config",
	"marketplace",
	"monitoring",
	"network",
	"openshift-apiserver",
	"openshift-controller-manager",
	"operator-lifecycle-manager",
	"service-ca",
	"storage",
}

// GetClusterStateSummary prints a human-readable highlight of some basic information about the openshift cluster that
// is valuable to every caller of `oc must-gather`.  Even different products.
// This is NOT a place to add your pet conditions.  We have three places for components to report their errors,
//  1. clusterversion - shows errors applying payloads to transition versions
//  2. clusteroperators - shows whether every operand is functioning properly
//  3. alerts - show whether something might be at risk
//
// if you find yourself wanting to add an additional piece of information here, what you really want to do is add it
// to one of those three spots.  Doing so improves the self-diagnosis capabilities of our platform and lets *every*
// client benefit.
func GetClusterStateSummary(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	var result strings.Builder

	result.WriteString("When opening a support case, bugzilla, or issue please include the following summary data along with any other requested information:\n")

	clusterVersionUnstructured, err := params.ResourcesGet(params, &schema.GroupVersionKind{
		Group:   "config.openshift.io",
		Version: "v1",
		Kind:    "ClusterVersion",
	}, "", "version")
	if err != nil {
		result.WriteString(fmt.Sprintf("error getting cluster version: %v\n", err))
	}

	var clusterVersion *configv1.ClusterVersion
	if clusterVersionUnstructured != nil {
		clusterVersion = &configv1.ClusterVersion{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(clusterVersionUnstructured.Object, clusterVersion); err != nil {
			result.WriteString(fmt.Sprintf("error converting cluster version: %v\n", err))
		}
	}

	if clusterVersion != nil {
		result.WriteString(fmt.Sprintf("ClusterID: %v\n", clusterVersion.Spec.ClusterID))
	}

	clientVersion := humanSummaryForClientVersion()
	result.WriteString(fmt.Sprintf("ClientVersion: %v\n", clientVersion))
	result.WriteString(fmt.Sprintf("ClusterVersion: %v\n", humanSummaryForClusterVersion(clusterVersion)))

	clusterOperatorsUnstructured, err := params.ResourcesList(params, &schema.GroupVersionKind{
		Group:   "config.openshift.io",
		Version: "v1",
		Kind:    "ClusterOperator",
	}, "", internalk8s.ResourceListOptions{})
	if err != nil {
		result.WriteString(fmt.Sprintf("error getting cluster operators: %v\n", err))
	}

	var clusterOperators *configv1.ClusterOperatorList
	if clusterOperatorsUnstructured != nil {
		clusterOperators = &configv1.ClusterOperatorList{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(clusterOperatorsUnstructured.UnstructuredContent(), clusterOperators); err != nil {
			result.WriteString(fmt.Sprintf("error converting cluster operators: %v\n", err))
		}
	}

	result.WriteString("ClusterOperators:\n")
	result.WriteString(humanSummaryForInterestingClusterOperators(clusterOperators) + "\n")

	// TODO gather and display firing alerts
	result.WriteString("\n\n")

	return api.NewToolCallResult(result.String(), nil), nil
}

func humanSummaryForInterestingClusterOperators(clusterOperators *configv1.ClusterOperatorList) string {
	if clusterOperators == nil {
		return "\tclusteroperators not found"
	}
	if len(clusterOperators.Items) == 0 {
		return "\tclusteroperators are missing"
	}

	missingOperators := sets.NewString(longExistingOperators...)
	clusterOperatorStrings := []string{}
	for _, clusterOperator := range clusterOperators.Items {
		missingOperators.Delete(clusterOperator.Name)
		clusterOperatorSummary := humanSummaryForClusterOperator(clusterOperator)
		if len(clusterOperatorSummary) == 0 { // not noteworthy
			continue
		}
		clusterOperatorStrings = append(clusterOperatorStrings, humanSummaryForClusterOperator(clusterOperator))
	}

	for _, missingOperator := range missingOperators.List() {
		clusterOperatorStrings = append(clusterOperatorStrings, fmt.Sprintf("clusteroperator/%s is missing", missingOperator))
	}

	if len(clusterOperatorStrings) > 0 {
		return "\t" + strings.Join(clusterOperatorStrings, "\n\t")
	}
	return "\tAll healthy and stable"
}

func humanSummaryForClusterOperator(clusterOperator configv1.ClusterOperator) string {
	available := v1helpers.IsStatusConditionTrue(clusterOperator.Status.Conditions, configv1.OperatorAvailable)
	progressing := !v1helpers.IsStatusConditionFalse(clusterOperator.Status.Conditions, configv1.OperatorProgressing)
	degraded := !v1helpers.IsStatusConditionFalse(clusterOperator.Status.Conditions, configv1.OperatorDegraded)
	upgradeable := !v1helpers.IsStatusConditionFalse(clusterOperator.Status.Conditions, configv1.OperatorUpgradeable)

	availableMessage := "<missing>"
	progressingMessage := "<missing>"
	degradedMessage := "<missing>"
	upgradeableMessage := "<missing>"
	if condition := v1helpers.FindStatusCondition(clusterOperator.Status.Conditions, configv1.OperatorAvailable); condition != nil {
		availableMessage = condition.Message
	}
	if condition := v1helpers.FindStatusCondition(clusterOperator.Status.Conditions, configv1.OperatorProgressing); condition != nil {
		progressingMessage = condition.Message
	}
	if condition := v1helpers.FindStatusCondition(clusterOperator.Status.Conditions, configv1.OperatorDegraded); condition != nil {
		degradedMessage = condition.Message
	}
	if condition := v1helpers.FindStatusCondition(clusterOperator.Status.Conditions, configv1.OperatorUpgradeable); condition != nil {
		upgradeableMessage = condition.Message
	}

	switch {
	case !available:
		return fmt.Sprintf("clusteroperator/%s is not available (%v) because %v", clusterOperator.Name, availableMessage, degradedMessage)
	case degraded:
		return fmt.Sprintf("clusteroperator/%s is degraded because %v", clusterOperator.Name, degradedMessage)
	case !upgradeable:
		return fmt.Sprintf("clusteroperator/%s is not upgradeable because %v", clusterOperator.Name, upgradeableMessage)
	case progressing:
		return fmt.Sprintf("clusteroperator/%s is progressing: %v", clusterOperator.Name, progressingMessage)
	case available && !progressing && !degraded && upgradeable:
		return ""
	default:
		return fmt.Sprintf("clusteroperator/%s is in an edge case", clusterOperator.Name)
	}
}

func humanSummaryForClusterVersion(clusterVersion *configv1.ClusterVersion) string {
	if clusterVersion == nil {
		return "Cluster is in a version state we don't recognize"
	}

	isInstalling :=
		len(clusterVersion.Status.History) == 0 ||
			(len(clusterVersion.Status.History) == 1 && clusterVersion.Status.History[0].State != configv1.CompletedUpdate)
	isUpdating := len(clusterVersion.Status.History) > 1 && clusterVersion.Status.History[0].State != configv1.CompletedUpdate
	isStable := len(clusterVersion.Status.History) > 0 && clusterVersion.Status.History[0].State == configv1.CompletedUpdate

	lastChangeHumanDuration := "<unknown>"
	if len(clusterVersion.Status.History) > 0 {
		lastChangeHumanDuration = units.HumanDuration(time.Since(clusterVersion.Status.History[0].StartedTime.Time))
	}

	progressingConditionMessage := "<unknown>"
	for _, condition := range clusterVersion.Status.Conditions {
		if condition.Type == "Progressing" {
			progressingConditionMessage = condition.Message
			break
		}
	}
	switch {
	case isInstalling:
		return fmt.Sprintf("Installing %q for %v: %v",
			clusterVersion.Status.Desired.Version, lastChangeHumanDuration, progressingConditionMessage)
	case isUpdating:
		return fmt.Sprintf("Updating to %q from %q for %v: %v",
			clusterVersion.Status.Desired.Version, clusterVersion.Status.History[1].Version, lastChangeHumanDuration, progressingConditionMessage)
	case isStable:
		return fmt.Sprintf("Stable at %q", clusterVersion.Status.History[0].Version)
	default:
		return "Unknown state"
	}
}

func humanSummaryForClientVersion() string {
	return version.Version
}

