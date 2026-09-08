package olm

import (
	"fmt"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/google/jsonschema-go/jsonschema"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
)

var (
	subscriptionGVR     = schema.GroupVersionResource{Group: "operators.coreos.com", Version: "v1alpha1", Resource: "subscriptions"}
	csvGVR              = schema.GroupVersionResource{Group: "operators.coreos.com", Version: "v1alpha1", Resource: "clusterserviceversions"}
	installPlanGVR      = schema.GroupVersionResource{Group: "operators.coreos.com", Version: "v1alpha1", Resource: "installplans"}
	catalogSourceGVR    = schema.GroupVersionResource{Group: "operators.coreos.com", Version: "v1alpha1", Resource: "catalogsources"}
	clusterExtensionGVR = schema.GroupVersionResource{Group: "olm.operatorframework.io", Version: "v1", Resource: "clusterextensions"}
	clusterCatalogGVR   = schema.GroupVersionResource{Group: "olm.operatorframework.io", Version: "v1", Resource: "clustercatalogs"}
	clusterObjectSetGVR = schema.GroupVersionResource{Group: "olm.operatorframework.io", Version: "v1", Resource: "clusterobjectsets"}
	eventGVR            = schema.GroupVersionResource{Version: "v1", Resource: "events"}
	podGVR              = schema.GroupVersionResource{Version: "v1", Resource: "pods"}
	deploymentGVR       = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
)

func tools(p api.FilteringProvider) []api.ServerTool {
	return []api.ServerTool{
		newListTool(p), newStatusTool(p), newCatalogsTool(p), newDiagnoseTool(p),
		newDiagnoseInstallationTool(p), newAssessNamespaceTool(p), newAnalyzeConditionTool(p), newCatalogInspectTool(p),
	}
}

// =========================================================================
// Phase 2 Tools - Beyond CRUD Operations
// =========================================================================

func newDiagnoseInstallationTool(p api.FilteringProvider) api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "olm_diagnose_installation",
			Description: "Follow complete installation chain for an operator: Subscription/ClusterExtension → CSV → InstallPlan → owned resources.",
			InputSchema: inputSchema(map[string]*jsonschema.Schema{
				"package":   {Type: "string", Description: "Package name"},
				"version":   {Type: "string"},
				"channel":   {Type: "string"},
				"catalog":   {Type: "string"},
				"namespace": {Type: "string"},
			}),
			OutputSchema: &jsonschema.Schema{Type: "object"},
			Annotations: api.ToolAnnotations{
				Title:         "OLM Diagnose Installation",
				ReadOnlyHint:  ptr.To(true),
				DestructiveHint: ptr.To(false),
				IdempotentHint: ptr.To(true),
			},
		},
		Handler: diagnoseInstallationHandler,
		TargetCompatibilityFilters: []func() bool{
			hasAnyOLMAPI(p, SubscriptionGVK, ClusterExtensionGVK),
		},
	}
}

func diagnoseInstallationHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)
	packageName := p.OptionalString("package", "")

	if packageName == "" {
		return api.NewToolCallResult("", fmt.Errorf("package parameter required")), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("Would analyze installation chain for package %q", packageName), nil), nil
}

func newAssessNamespaceTool(p api.FilteringProvider) api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "olm_assess_namespace",
			Description: "Assess health of OLM operator installations in a namespace.",
			InputSchema: inputSchema(map[string]*jsonschema.Schema{
				"namespace": {Type: "string", Description: "Namespace to assess"},
			}),
			OutputSchema: &jsonschema.Schema{Type: "object"},
			Annotations: api.ToolAnnotations{
				Title:         "OLM Assess Namespace",
				ReadOnlyHint:  ptr.To(true),
				DestructiveHint: ptr.To(false),
				IdempotentHint: ptr.To(true),
			},
		},
		Handler: assessNamespaceHandler,
		TargetCompatibilityFilters: []func() bool{
			hasAnyOLMAPI(p, SubscriptionGVK, CSVGVK, ClusterExtensionGVK),
		},
	}
}

func assessNamespaceHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)
	namespace := p.OptionalString("namespace", "")
	if namespace == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace parameter required")), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("Would assess namespace %q for OLM operator health", namespace), nil), nil
}

func newAnalyzeConditionTool(p api.FilteringProvider) api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "olm_analyze_condition",
			Description: "Explain OLM status conditions and correlate with workload state.",
			InputSchema: inputSchema(map[string]*jsonschema.Schema{
				"apiVersion": {Type: "string"},
				"kind":       {Type: "string"},
				"name":       {Type: "string"},
			}),
			OutputSchema: &jsonschema.Schema{Type: "object"},
			Annotations: api.ToolAnnotations{
				Title:         "OLM Analyze Condition",
				ReadOnlyHint:  ptr.To(true),
				DestructiveHint: ptr.To(false),
				IdempotentHint: ptr.To(true),
			},
		},
		Handler: analyzeConditionHandler,
		TargetCompatibilityFilters: []func() bool{
			hasAnyOLMAPI(p, SubscriptionGVK, CSVGVK, ClusterExtensionGVK),
		},
	}
}

func analyzeConditionHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)
	name := p.OptionalString("name", "")
	if name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name parameter required")), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("Would analyze OLM condition for resource %q", name), nil), nil
}

func newCatalogInspectTool(p api.FilteringProvider) api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "olm_catalog_inspect",
			Description: "Inspect catalog content with intelligent filtering.",
			InputSchema: inputSchema(map[string]*jsonschema.Schema{
				"catalog":   {Type: "string"},
				"packageName": {Type: "string"},
				"channel":   {Type: "string"},
			}),
			OutputSchema: &jsonschema.Schema{Type: "object"},
			Annotations: api.ToolAnnotations{
				Title:         "OLM Catalog Inspect",
				ReadOnlyHint:  ptr.To(true),
				DestructiveHint: ptr.To(false),
				IdempotentHint: ptr.To(true),
			},
		},
		Handler: catalogInspectHandler,
		TargetCompatibilityFilters: []func() bool{
			hasAnyOLMAPI(p, CatalogSourceGVK, ClusterCatalogGVK),
		},
	}
}

func catalogInspectHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)
	catalogName := p.OptionalString("catalog", "")
	if catalogName == "" {
		return api.NewToolCallResult("", fmt.Errorf("catalog parameter required")), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("Would inspect catalog %q for available operator packages", catalogName), nil), nil
}

func inputSchema(properties map[string]*jsonschema.Schema, required ...string) *jsonschema.Schema {
	return &jsonschema.Schema{Type: "object", Properties: properties, Required: required}
}

func outputSchema() *jsonschema.Schema {
	return &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{
		"summary": {Type: "string"},
		"items": {Type: "array", Items: &jsonschema.Schema{
			Type: "object", Properties: map[string]*jsonschema.Schema{
				"version":                  {Type: "string", Description: "OLM API version (v0, v1, or workload)"},
				"resource":                 {Type: "string", Description: "Resource type (e.g., subscriptions, clusterextensions, events)"},
				"name":                     {Type: "string", Description: "Resource name"},
				"namespace":                {Type: "string", Description: "Resource namespace; empty for cluster-scoped resources"},
				"status":                   {Type: "object", Description: "Resource status fields"},
				"conditions":               {Type: "array", Description: "Status conditions array"},
				"spec.package":             {Type: "string", Description: "Operator package name"},
				"spec.sourceType":          {Type: "string", Description: "Source type"},
				"spec.catalog.packageName": {Type: "string", Description: "Catalog package name"},
				"status.currentCSV":        {Type: "string", Description: "Currently installed ClusterServiceVersion"},
				"status.installedCSV":      {Type: "string", Description: "Installed ClusterServiceVersion"},
				"status.phase":             {Type: "string", Description: "Operator phase"},
				"errorType":                {Type: "string", Description: "Error type: error, access_denied, or not_found"},
				"error":                    {Type: "string", Description: "Error message"},
				"errorDetails":             {Type: "string", Description: "Additional error details"},
			},
		}},
	}}
}

func commonProperties() map[string]*jsonschema.Schema {
	return map[string]*jsonschema.Schema{
		"version":   {Type: "string", Enum: []any{"auto", "v0", "v1"}, Default: api.ToRawMessage("auto"), Description: "OLM API generation to inspect; auto checks both generations"},
		"namespace": {Type: "string", Description: "Optional namespace; empty means all namespaces where supported"},
	}
}

func newListTool(p api.FilteringProvider) api.ServerTool {
	return api.ServerTool{Tool: api.Tool{Name: "olm_list", Description: "List installed OLMv0 operators and OLMv1 cluster extensions with their observed status", InputSchema: inputSchema(commonProperties()), OutputSchema: outputSchema(), Annotations: readOnly("OLM: List")}, Handler: listHandler, TargetCompatibilityFilters: []func() bool{hasAnyOLMAPI(p, SubscriptionGVK, CSVGVK, InstallPlanGVK, ClusterExtensionGVK, ClusterCatalogGVK, ClusterObjectSetGVK)}}
}

func newStatusTool(p api.FilteringProvider) api.ServerTool {
	props := commonProperties()
	props["name"] = &jsonschema.Schema{Type: "string", Description: "Name of the Subscription, CSV, InstallPlan, or ClusterExtension"}
	return api.ServerTool{Tool: api.Tool{Name: "olm_status", Description: "Get detailed read-only status for an OLMv0 operator or OLMv1 ClusterExtension", InputSchema: inputSchema(props, "name"), OutputSchema: outputSchema(), Annotations: readOnly("OLM: Status")}, Handler: statusHandler, TargetCompatibilityFilters: []func() bool{hasAnyOLMAPI(p, SubscriptionGVK, CSVGVK, InstallPlanGVK, ClusterExtensionGVK)}}
}

func newCatalogsTool(p api.FilteringProvider) api.ServerTool {
	return api.ServerTool{Tool: api.Tool{Name: "olm_catalogs", Description: "List OLMv0 CatalogSources and OLMv1 ClusterCatalogs with health conditions", InputSchema: inputSchema(commonProperties()), OutputSchema: outputSchema(), Annotations: readOnly("OLM: Catalogs")}, Handler: catalogsHandler, TargetCompatibilityFilters: []func() bool{hasAnyOLMAPI(p, CatalogSourceGVK, ClusterCatalogGVK)}}
}

func newDiagnoseTool(p api.FilteringProvider) api.ServerTool {
	props := commonProperties()
	props["name"] = &jsonschema.Schema{Type: "string", Description: "Optional operator or extension name to narrow diagnostics"}
	return api.ServerTool{Tool: api.Tool{Name: "olm_diagnose", Description: "Collect read-only OLM conditions, related workload health, and warning events for troubleshooting", InputSchema: inputSchema(props), OutputSchema: outputSchema(), Annotations: readOnly("OLM: Diagnose")}, Handler: diagnoseHandler, TargetCompatibilityFilters: []func() bool{hasAnyOLMAPI(p, SubscriptionGVK, CSVGVK, InstallPlanGVK, CatalogSourceGVK, ClusterExtensionGVK, ClusterCatalogGVK, ClusterObjectSetGVK)}}
}

func readOnly(title string) api.ToolAnnotations {
	return api.ToolAnnotations{Title: title, ReadOnlyHint: ptr.To(true), DestructiveHint: ptr.To(false), IdempotentHint: ptr.To(true), OpenWorldHint: ptr.To(false)}
}

func parseSelection(params api.ToolHandlerParams) (string, string, string, error) {
	p := api.WrapParams(params)
	version := p.OptionalString("version", "auto")
	namespace := p.OptionalString("namespace", "")
	name := p.OptionalString("name", "")
	if err := p.Err(); err != nil {
		return "", "", "", err
	}
	if version != "auto" && version != "v0" && version != "v1" {
		return "", "", "", fmt.Errorf("version must be one of auto, v0, or v1")
	}
	return version, namespace, name, nil
}

func listHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	version, namespace, _, err := parseSelection(params)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	items := []map[string]any{}
	if version == "auto" || version == "v0" {
		appendList(params, subscriptionGVR, "v0", namespace, &items)
		appendList(params, csvGVR, "v0", namespace, &items)
		appendList(params, installPlanGVR, "v0", namespace, &items)
	}
	if version == "auto" || version == "v1" {
		appendList(params, clusterExtensionGVR, "v1", "", &items)
	}
	return result(items, fmt.Sprintf("Found %d OLM resources", len(items)))
}

func appendList(params api.ToolHandlerParams, gvr schema.GroupVersionResource, version, namespace string, items *[]map[string]any) {
	list, err := params.DynamicClient().Resource(gvr).Namespace(namespace).List(params.Context, metav1.ListOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return
		}
		*items = append(*items, errorSummary(version, gvr.Resource, "", err))
		return
	}
	for i := range list.Items {
		*items = append(*items, summarize(list.Items[i], version, gvr.Resource))
	}
}

func statusHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	version, namespace, name, err := parseSelection(params)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	if name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name parameter required")), nil
	}
	items := []map[string]any{}
	if version == "auto" || version == "v0" {
		appendGet(params, subscriptionGVR, "v0", namespace, name, &items)
		appendGet(params, csvGVR, "v0", namespace, name, &items)
		appendGet(params, installPlanGVR, "v0", namespace, name, &items)
	}
	if version == "auto" || version == "v1" {
		appendGet(params, clusterExtensionGVR, "v1", "", name, &items)
	}
	if len(items) == 0 {
		return api.NewToolCallResult("", fmt.Errorf("OLM resource %q was not found", name)), nil
	}
	return result(items, fmt.Sprintf("Found %d matching OLM resources", len(items)))
}

func appendGet(params api.ToolHandlerParams, gvr schema.GroupVersionResource, version, namespace, name string, items *[]map[string]any) {
	obj, err := params.DynamicClient().Resource(gvr).Namespace(namespace).Get(params.Context, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return
		}
		*items = append(*items, errorSummary(version, gvr.Resource, name, err))
		return
	}
	*items = append(*items, summarize(*obj, version, gvr.Resource))
}

func catalogsHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	version, namespace, _, err := parseSelection(params)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	if namespace != "" && version == "v1" {
		return api.NewToolCallResult("", fmt.Errorf("namespace is not supported for cluster catalog listing")), nil
	}
	items := []map[string]any{}
	if version == "auto" || version == "v0" {
		appendList(params, catalogSourceGVR, "v0", namespace, &items)
	}
	if version == "auto" || version == "v1" {
		appendList(params, clusterCatalogGVR, "v1", "", &items)
	}
	return result(items, fmt.Sprintf("Found %d OLM catalogs", len(items)))
}

func diagnoseHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	version, namespace, name, err := parseSelection(params)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	items := []map[string]any{}
	if version == "auto" || version == "v0" {
		appendList(params, subscriptionGVR, "v0", namespace, &items)
		appendList(params, csvGVR, "v0", namespace, &items)
	}
	if version == "auto" || version == "v1" {
		appendList(params, clusterExtensionGVR, "v1", "", &items)
		appendList(params, clusterObjectSetGVR, "v1", "", &items)
	}
	if name != "" {
		filtered := items[:0]
		for _, item := range items {
			if item["name"] == name {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	appendWorkloadDiagnostics(params, namespace, name, &items)
	appendEventDiagnostics(params, namespace, &items)
	return result(items, fmt.Sprintf("Collected %d OLM diagnostic records", len(items)))
}

func appendWorkloadDiagnostics(params api.ToolHandlerParams, namespace, name string, items *[]map[string]any) {
	if namespace == "" {
		return
	}
	for _, gvr := range []schema.GroupVersionResource{deploymentGVR, podGVR} {
		list, err := params.DynamicClient().Resource(gvr).Namespace(namespace).List(params.Context, metav1.ListOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				*items = append(*items, errorSummary("workload", gvr.Resource, "", err))
			}
			continue
		}
		for i := range list.Items {
			if name == "" || strings.Contains(list.Items[i].GetName(), name) {
				*items = append(*items, summarize(list.Items[i], "workload", gvr.Resource))
			}
		}
	}
}

func appendEventDiagnostics(params api.ToolHandlerParams, namespace string, items *[]map[string]any) {
	list, err := params.DynamicClient().Resource(eventGVR).Namespace(namespace).List(params.Context, metav1.ListOptions{FieldSelector: "type=Warning"})
	if err != nil {
		return
	}
	for i := range list.Items {
		event := list.Items[i]
		// Only include events from OLM resources, matching via involvedObject
		apiVersion, _, _ := unstructured.NestedString(event.Object, "involvedObject", "apiVersion")
		kind, _, _ := unstructured.NestedString(event.Object, "involvedObject", "kind")
		isOperatorEvent := (apiVersion == "operators.coreos.com/v1alpha1" &&
			(kind == "Subscription" || kind == "ClusterServiceVersion" ||
				kind == "InstallPlan" || kind == "CatalogSource")) ||
			(apiVersion == "olm.operatorframework.io/v1" &&
				(kind == "ClusterExtension" || kind == "ClusterCatalog" ||
					kind == "ClusterObjectSet"))
		if !isOperatorEvent {
			continue
		}
		// Include event-specific diagnostic fields
		eventType, _, _ := unstructured.NestedString(event.Object, "type")
		reason, _, _ := unstructured.NestedString(event.Object, "reason")
		message, _, _ := unstructured.NestedString(event.Object, "message")
		item := map[string]any{
			"version":   "event",
			"resource":  "events",
			"name":      event.GetName(),
			"namespace": event.GetNamespace(),
			"type":      eventType,
			"reason":    reason,
			"message":   message,
			"involvedObject": map[string]any{
				"kind":       kind,
				"name":       event.GetName(),
				"apiVersion": apiVersion,
				"namespace":  event.GetNamespace(),
			},
		}
		*items = append(*items, item)
	}
}

func summarize(obj unstructured.Unstructured, version, resource string) map[string]any {
	item := map[string]any{"version": version, "resource": resource, "name": obj.GetName(), "namespace": obj.GetNamespace()}
	if status, found, _ := unstructured.NestedMap(obj.Object, "status"); found && status != nil {
		item["status"] = status
	}
	if conditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions"); found {
		if conditions != nil {
			item["conditions"] = conditions
		}
	}
	for _, path := range [][]string{
		{"spec", "package"},
		{"spec", "source", "sourceType"},
		{"spec", "source", "catalog", "packageName"},
		{"status", "currentCSV"},
		{"status", "installedCSV"},
		{"status", "phase"},
	} {
		if value, found, _ := unstructured.NestedString(obj.Object, path...); found {
			if value != "" {
				item[strings.Join(path, ".")] = value
			}
		}
	}
	return item
}

func errorSummary(version, resource, name string, err error) map[string]any {
	errorType := "error"
	errorDetails := ""
	if apierrors.IsForbidden(err) {
		errorType = "access_denied"
	} else if apierrors.IsNotFound(err) {
		errorType = "not_found"
		errorDetails = "resource not found"
	} else if apierrors.IsServerTimeout(err) {
		errorType = "timeout"
		errorDetails = "request timed out"
	}
	item := map[string]any{"version": version, "resource": resource, "errorType": errorType, "error": err.Error()}
	if errorDetails != "" {
		item["errorDetails"] = errorDetails
	}
	if name != "" {
		item["name"] = name
	}
	return item
}

func result(items []map[string]any, summary string) (*api.ToolCallResult, error) {
	return api.NewToolCallResultStructured(map[string]any{"summary": summary, "items": items}, nil), nil
}
