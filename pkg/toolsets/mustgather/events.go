package mustgather

import (
	"fmt"
	"sort"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
)

var eventGVK = schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Event"}

func initEvents() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "mustgather_events_list",
				Description: "List Kubernetes events from the must-gather archive with optional filtering by type, namespace, resource, and reason",
				Annotations: api.ToolAnnotations{
					Title:        "List Events",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"type":      {Type: "string", Description: "Event type filter: all, Warning, Normal", Enum: []any{"all", "Warning", "Normal"}},
						"namespace": {Type: "string", Description: "Filter by namespace"},
						"resource":  {Type: "string", Description: "Filter by involved resource name (partial match)"},
						"reason":    {Type: "string", Description: "Filter by event reason (partial match)"},
						"limit":     {Type: "integer", Description: "Maximum number of events to return (default: 100)"},
					},
				},
			},
			Handler:      mustgatherEventsList,
			ClusterAware: ptr.To(false),
		},
		{
			Tool: api.Tool{
				Name:        "mustgather_events_by_resource",
				Description: "Get all events related to a specific Kubernetes resource from the must-gather archive",
				Annotations: api.ToolAnnotations{
					Title:        "Events by Resource",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"name":      {Type: "string", Description: "Resource name"},
						"namespace": {Type: "string", Description: "Resource namespace"},
						"kind":      {Type: "string", Description: "Resource kind (optional, narrows search)"},
					},
					Required: []string{"name"},
				},
			},
			Handler:      mustgatherEventsByResource,
			ClusterAware: ptr.To(false),
		},
	}
}

func mustgatherEventsList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p, err := getProvider()
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	args := params.GetArguments()
	typeFilter := getString(args, "type", "all")
	namespace := getString(args, "namespace", "")
	resourceFilter := getString(args, "resource", "")
	reasonFilter := getString(args, "reason", "")
	limit := getInt(args, "limit", 100)

	list := p.ListResources(params.Context, eventGVK, namespace, mg.ListOptions{})

	// Filter and collect events
	var filtered []unstructured.Unstructured
	for i := range list.Items {
		event := &list.Items[i]

		if typeFilter != "all" {
			eventType, _, _ := unstructured.NestedString(event.Object, "type")
			if eventType != typeFilter {
				continue
			}
		}

		if resourceFilter != "" {
			involvedName, _, _ := unstructured.NestedString(event.Object, "involvedObject", "name")
			if !strings.Contains(strings.ToLower(involvedName), strings.ToLower(resourceFilter)) {
				continue
			}
		}

		if reasonFilter != "" {
			reason, _, _ := unstructured.NestedString(event.Object, "reason")
			if !strings.Contains(strings.ToLower(reason), strings.ToLower(reasonFilter)) {
				continue
			}
		}

		filtered = append(filtered, *event)
	}

	// Sort by lastTimestamp descending
	sort.Slice(filtered, func(i, j int) bool {
		ti, _, _ := unstructured.NestedString(filtered[i].Object, "lastTimestamp")
		tj, _, _ := unstructured.NestedString(filtered[j].Object, "lastTimestamp")
		return ti > tj
	})

	// Apply limit
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	if len(filtered) == 0 {
		return api.NewToolCallResult("No events found matching the criteria", nil), nil
	}

	output := fmt.Sprintf("Found %d events:\n\n", len(filtered))
	for i := range filtered {
		output += formatEvent(&filtered[i])
	}

	return api.NewToolCallResult(output, nil), nil
}

func mustgatherEventsByResource(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p, err := getProvider()
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	args := params.GetArguments()
	name := getString(args, "name", "")
	namespace := getString(args, "namespace", "")
	kindFilter := getString(args, "kind", "")

	if name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	list := p.ListResources(params.Context, eventGVK, namespace, mg.ListOptions{})

	// Filter events for the specific resource
	var matched []unstructured.Unstructured
	for i := range list.Items {
		event := &list.Items[i]
		involvedName, _, _ := unstructured.NestedString(event.Object, "involvedObject", "name")
		if involvedName != name {
			continue
		}
		if kindFilter != "" {
			involvedKind, _, _ := unstructured.NestedString(event.Object, "involvedObject", "kind")
			if !strings.EqualFold(involvedKind, kindFilter) {
				continue
			}
		}
		matched = append(matched, *event)
	}

	// Sort chronologically
	sort.Slice(matched, func(i, j int) bool {
		ti, _, _ := unstructured.NestedString(matched[i].Object, "lastTimestamp")
		tj, _, _ := unstructured.NestedString(matched[j].Object, "lastTimestamp")
		return ti < tj
	})

	if len(matched) == 0 {
		return api.NewToolCallResult(fmt.Sprintf("No events found for resource %s", name), nil), nil
	}

	output := fmt.Sprintf("Found %d events for %s:\n\n", len(matched), name)
	for i := range matched {
		output += formatEvent(&matched[i])
	}

	return api.NewToolCallResult(output, nil), nil
}

func formatEvent(event *unstructured.Unstructured) string {
	eventType, _, _ := unstructured.NestedString(event.Object, "type")
	reason, _, _ := unstructured.NestedString(event.Object, "reason")
	message, _, _ := unstructured.NestedString(event.Object, "message")
	lastTimestamp, _, _ := unstructured.NestedString(event.Object, "lastTimestamp")
	involvedName, _, _ := unstructured.NestedString(event.Object, "involvedObject", "name")
	involvedKind, _, _ := unstructured.NestedString(event.Object, "involvedObject", "kind")
	ns := event.GetNamespace()

	marker := "[Normal]"
	if eventType == "Warning" {
		marker = "[Warning]"
	}

	output := fmt.Sprintf("%s %s %s/%s", marker, lastTimestamp, involvedKind, involvedName)
	if ns != "" {
		output += fmt.Sprintf(" (ns: %s)", ns)
	}
	output += "\n"
	output += fmt.Sprintf("  Reason: %s\n", reason)
	output += fmt.Sprintf("  Message: %s\n\n", message)
	return output
}
