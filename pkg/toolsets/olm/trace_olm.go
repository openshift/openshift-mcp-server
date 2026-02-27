package olm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/utils/ptr"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
)

var (
	subscriptionGVR = schema.GroupVersionResource{
		Group: "operators.coreos.com", Version: "v1alpha1", Resource: "subscriptions",
	}
	catalogSourceGVR = schema.GroupVersionResource{
		Group: "operators.coreos.com", Version: "v1alpha1", Resource: "catalogsources",
	}
	installPlanGVR = schema.GroupVersionResource{
		Group: "operators.coreos.com", Version: "v1alpha1", Resource: "installplans",
	}
	csvGVR = schema.GroupVersionResource{
		Group: "operators.coreos.com", Version: "v1alpha1", Resource: "clusterserviceversions",
	}
)

func initTraceOLM() []api.ServerTool {
	return []api.ServerTool{
		{Tool: api.Tool{
			Name: "trace_olm_subscription",
			Description: "Trace the full OLM (Operator Lifecycle Manager) chain for a Subscription. " +
				"Walks Subscription → CatalogSource → InstallPlan → ClusterServiceVersion and " +
				"identifies the exact step where the operator installation is failing. " +
				"Returns structured JSON with the chain status and the failing_step.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"subscription_name": {
						Type:        "string",
						Description: "Name of the OLM Subscription to trace",
					},
					"subscription_namespace": {
						Type:        "string",
						Description: "Namespace of the Subscription (default: openshift-operators)",
					},
				},
				Required: []string{"subscription_name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OLM: Trace Subscription",
				ReadOnlyHint:    ptr.To(true),
				DestructiveHint: ptr.To(false),
				OpenWorldHint:   ptr.To(true),
			},
		}, Handler: traceOLMSubscription},
	}
}

type olmTrace struct {
	Subscription  *olmStepStatus  `json:"subscription"`
	CatalogSource *olmStepStatus  `json:"catalog_source,omitempty"`
	InstallPlans  []olmStepStatus `json:"install_plans,omitempty"`
	CSV           *olmStepStatus  `json:"csv,omitempty"`
	FailingStep   string          `json:"failing_step"`
	Summary       string          `json:"summary"`
}

type olmStepStatus struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
}

func traceOLMSubscription(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	subName, _ := params.GetArguments()["subscription_name"].(string)
	if subName == "" {
		return api.NewToolCallResult("", fmt.Errorf("subscription_name is required")), nil
	}
	subNS, _ := params.GetArguments()["subscription_namespace"].(string)
	if subNS == "" {
		subNS = "openshift-operators"
	}

	dynClient := params.DynamicClient()
	ctx := params.Context

	trace := &olmTrace{}

	sub, err := dynClient.Resource(subscriptionGVR).Namespace(subNS).Get(ctx, subName, metav1.GetOptions{})
	if err != nil {
		trace.FailingStep = "subscription"
		trace.Summary = fmt.Sprintf("Subscription %s/%s not found: %v", subNS, subName, err)
		trace.Subscription = &olmStepStatus{
			Name: subName, Namespace: subNS, Status: "NotFound", Message: err.Error(),
		}
		return marshalTrace(trace)
	}

	subState := extractSubscriptionState(sub)
	trace.Subscription = subState

	traceCatalogSource(ctx, dynClient, sub, trace)
	traceInstallPlans(ctx, dynClient, sub, trace)
	traceCSV(ctx, dynClient, sub, trace)

	if trace.FailingStep == "" {
		if trace.Subscription.Status != "AtLatestKnown" {
			trace.FailingStep = "subscription"
			trace.Summary = fmt.Sprintf("Subscription is in state %q, not AtLatestKnown", trace.Subscription.Status)
		} else if trace.CSV != nil && trace.CSV.Status != "Succeeded" {
			trace.FailingStep = "csv"
			trace.Summary = fmt.Sprintf("CSV %s is in phase %q", trace.CSV.Name, trace.CSV.Status)
		} else {
			trace.FailingStep = "none"
			trace.Summary = "OLM chain is healthy. Subscription is at latest known and CSV succeeded."
		}
	}

	return marshalTrace(trace)
}

func extractSubscriptionState(sub *unstructured.Unstructured) *olmStepStatus {
	state, _, _ := unstructured.NestedString(sub.Object, "status", "state")
	currentCSV, _, _ := unstructured.NestedString(sub.Object, "status", "currentCSV")
	installedCSV, _, _ := unstructured.NestedString(sub.Object, "status", "installedCSV")

	msg := fmt.Sprintf("currentCSV=%s, installedCSV=%s", currentCSV, installedCSV)

	conditions, _, _ := unstructured.NestedSlice(sub.Object, "status", "conditions")
	for _, c := range conditions {
		cMap, ok := c.(map[string]any)
		if !ok {
			continue
		}
		cType, _ := cMap["type"].(string)
		cStatus, _ := cMap["status"].(string)
		cMsg, _ := cMap["message"].(string)
		if cStatus != "True" || cType == "" {
			continue
		}
		if cMsg != "" {
			msg += fmt.Sprintf("; condition %s: %s", cType, truncate(cMsg, 200))
		}
	}

	return &olmStepStatus{
		Name:      sub.GetName(),
		Namespace: sub.GetNamespace(),
		Status:    state,
		Message:   msg,
	}
}

func traceCatalogSource(ctx context.Context, dynClient dynamic.Interface, sub *unstructured.Unstructured, trace *olmTrace) {
	catName, _, _ := unstructured.NestedString(sub.Object, "spec", "source")
	catNS, _, _ := unstructured.NestedString(sub.Object, "spec", "sourceNamespace")
	if catName == "" {
		return
	}
	if catNS == "" {
		catNS = "openshift-marketplace"
	}

	cat, err := dynClient.Resource(catalogSourceGVR).Namespace(catNS).Get(ctx, catName, metav1.GetOptions{})
	if err != nil {
		trace.CatalogSource = &olmStepStatus{
			Name: catName, Namespace: catNS, Status: "NotFound", Message: err.Error(),
		}
		trace.FailingStep = "catalog_source"
		trace.Summary = fmt.Sprintf("CatalogSource %s/%s not found: %v", catNS, catName, err)
		return
	}

	connectionState, _, _ := unstructured.NestedMap(cat.Object, "status", "connectionState")
	lastConnect, _ := connectionState["lastObservedState"].(string)

	trace.CatalogSource = &olmStepStatus{
		Name:      catName,
		Namespace: catNS,
		Status:    lastConnect,
		Message:   fmt.Sprintf("lastObservedState=%s", lastConnect),
	}

	if !strings.EqualFold(lastConnect, "READY") {
		trace.FailingStep = "catalog_source"
		trace.Summary = fmt.Sprintf("CatalogSource %s is not READY (state=%s)", catName, lastConnect)
	}
}

func traceInstallPlans(ctx context.Context, dynClient dynamic.Interface, sub *unstructured.Unstructured, trace *olmTrace) {
	ipRef, _, _ := unstructured.NestedString(sub.Object, "status", "installPlanRef", "name")
	ns := sub.GetNamespace()

	if ipRef != "" {
		ip, err := dynClient.Resource(installPlanGVR).Namespace(ns).Get(ctx, ipRef, metav1.GetOptions{})
		if err != nil {
			trace.InstallPlans = append(trace.InstallPlans, olmStepStatus{
				Name: ipRef, Namespace: ns, Status: "NotFound", Message: err.Error(),
			})
			if trace.FailingStep == "" {
				trace.FailingStep = "install_plan"
				trace.Summary = fmt.Sprintf("InstallPlan %s not found: %v", ipRef, err)
			}
			return
		}
		phase, _, _ := unstructured.NestedString(ip.Object, "status", "phase")
		trace.InstallPlans = append(trace.InstallPlans, olmStepStatus{
			Name: ipRef, Namespace: ns, Status: phase,
		})
		if phase != "Complete" && trace.FailingStep == "" {
			trace.FailingStep = "install_plan"
			trace.Summary = fmt.Sprintf("InstallPlan %s is in phase %q (expected Complete)", ipRef, phase)
		}
		return
	}

	ipList, err := dynClient.Resource(installPlanGVR).Namespace(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return
	}
	for _, ip := range ipList.Items {
		phase, _, _ := unstructured.NestedString(ip.Object, "status", "phase")
		trace.InstallPlans = append(trace.InstallPlans, olmStepStatus{
			Name: ip.GetName(), Namespace: ns, Status: phase,
		})
	}
}

func traceCSV(ctx context.Context, dynClient dynamic.Interface, sub *unstructured.Unstructured, trace *olmTrace) {
	csvName, _, _ := unstructured.NestedString(sub.Object, "status", "currentCSV")
	if csvName == "" {
		csvName, _, _ = unstructured.NestedString(sub.Object, "status", "installedCSV")
	}
	if csvName == "" {
		return
	}

	ns := sub.GetNamespace()
	csv, err := dynClient.Resource(csvGVR).Namespace(ns).Get(ctx, csvName, metav1.GetOptions{})
	if err != nil {
		trace.CSV = &olmStepStatus{
			Name: csvName, Namespace: ns, Status: "NotFound", Message: err.Error(),
		}
		if trace.FailingStep == "" {
			trace.FailingStep = "csv"
			trace.Summary = fmt.Sprintf("CSV %s not found: %v", csvName, err)
		}
		return
	}

	phase, _, _ := unstructured.NestedString(csv.Object, "status", "phase")
	reason, _, _ := unstructured.NestedString(csv.Object, "status", "reason")
	csvMsg, _, _ := unstructured.NestedString(csv.Object, "status", "message")

	msg := fmt.Sprintf("phase=%s", phase)
	if reason != "" {
		msg += fmt.Sprintf(", reason=%s", reason)
	}
	if csvMsg != "" {
		msg += fmt.Sprintf(", message=%s", truncate(csvMsg, 200))
	}

	trace.CSV = &olmStepStatus{
		Name:      csvName,
		Namespace: ns,
		Status:    phase,
		Message:   msg,
	}

	if phase != "Succeeded" && trace.FailingStep == "" {
		trace.FailingStep = "csv"
		trace.Summary = fmt.Sprintf("CSV %s is in phase %q: %s", csvName, phase, reason)
	}
}

func marshalTrace(trace *olmTrace) (*api.ToolCallResult, error) {
	data, err := json.MarshalIndent(trace, "", "  ")
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal OLM trace: %v", err)), nil
	}
	return api.NewToolCallResult(string(data), nil), nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
