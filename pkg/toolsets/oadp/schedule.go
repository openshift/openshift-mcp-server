package oadp

import (
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/oadp"
	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"
)

func initScheduleTools() []api.ServerTool {
	return []api.ServerTool{
		initScheduleList(),
		initScheduleGet(),
		initScheduleCreate(),
		initScheduleUpdate(),
		initScheduleDelete(),
		initSchedulePause(),
	}
}

func initScheduleList() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_schedule_list",
			Description: "List all Velero/OADP backup schedules",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace containing schedules (default: openshift-adp)",
					},
				},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: List Schedules",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: scheduleListHandler,
	}
}

func scheduleListHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	schedules, err := oadp.ListSchedules(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list schedules: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(schedules)), nil
}

func initScheduleGet() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_schedule_get",
			Description: "Get detailed information about a specific backup schedule including cron expression, last backup time, and template",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the schedule (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the schedule",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Get Schedule",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: scheduleGetHandler,
	}
}

func scheduleGetHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	schedule, err := oadp.GetSchedule(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get schedule: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(schedule)), nil
}

func initScheduleCreate() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_schedule_create",
			Description: "Create a new backup schedule with a cron expression",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "OADP namespace (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name for the schedule",
					},
					"schedule": {
						Type:        "string",
						Description: "Cron expression (e.g., '0 1 * * *' for daily at 1am, '@every 6h' for every 6 hours)",
					},
					"includedNamespaces": {
						Type:        "array",
						Description: "Namespaces to include in scheduled backups",
						Items:       &jsonschema.Schema{Type: "string"},
					},
					"excludedNamespaces": {
						Type:        "array",
						Description: "Namespaces to exclude from scheduled backups",
						Items:       &jsonschema.Schema{Type: "string"},
					},
					"includedResources": {
						Type:        "array",
						Description: "Resource types to include",
						Items:       &jsonschema.Schema{Type: "string"},
					},
					"excludedResources": {
						Type:        "array",
						Description: "Resource types to exclude",
						Items:       &jsonschema.Schema{Type: "string"},
					},
					"storageLocation": {
						Type:        "string",
						Description: "BackupStorageLocation name",
					},
					"ttl": {
						Type:        "string",
						Description: "Backup TTL duration (e.g., '720h' for 30 days)",
					},
				},
				Required: []string{"name", "schedule"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Create Schedule",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(false),
			},
		},
		Handler: scheduleCreateHandler,
	}
}

func scheduleCreateHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	cronSchedule, ok := params.GetArguments()["schedule"].(string)
	if !ok || cronSchedule == "" {
		return api.NewToolCallResult("", fmt.Errorf("schedule is required")), nil
	}

	// Build the backup template spec
	template := map[string]interface{}{}

	if v, ok := params.GetArguments()["includedNamespaces"].([]interface{}); ok {
		template["includedNamespaces"] = v
	}
	if v, ok := params.GetArguments()["excludedNamespaces"].([]interface{}); ok {
		template["excludedNamespaces"] = v
	}
	if v, ok := params.GetArguments()["includedResources"].([]interface{}); ok {
		template["includedResources"] = v
	}
	if v, ok := params.GetArguments()["excludedResources"].([]interface{}); ok {
		template["excludedResources"] = v
	}
	if v, ok := params.GetArguments()["storageLocation"].(string); ok && v != "" {
		template["storageLocation"] = v
	}
	if v, ok := params.GetArguments()["ttl"].(string); ok && v != "" {
		template["ttl"] = v
	}

	schedule := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": oadp.VeleroGroup + "/" + oadp.VeleroVersion,
			"kind":       "Schedule",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"schedule": cronSchedule,
				"template": template,
			},
		},
	}

	created, err := oadp.CreateSchedule(params.Context, params.DynamicClient(), schedule)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create schedule: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(created)), nil
}

func initScheduleUpdate() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_schedule_update",
			Description: "Update a backup schedule's cron expression or backup template",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the schedule (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the schedule to update",
					},
					"schedule": {
						Type:        "string",
						Description: "New cron expression (e.g., '0 1 * * *' for daily at 1am)",
					},
					"ttl": {
						Type:        "string",
						Description: "New backup TTL duration (e.g., '720h' for 30 days)",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Update Schedule",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: scheduleUpdateHandler,
	}
}

func scheduleUpdateHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	// Get the existing schedule
	schedule, err := oadp.GetSchedule(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get schedule: %w", err)), nil
	}

	// Apply updates
	if cronSchedule, ok := params.GetArguments()["schedule"].(string); ok && cronSchedule != "" {
		if err := unstructured.SetNestedField(schedule.Object, cronSchedule, "spec", "schedule"); err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to set schedule field: %w", err)), nil
		}
	}

	if ttl, ok := params.GetArguments()["ttl"].(string); ok && ttl != "" {
		if err := unstructured.SetNestedField(schedule.Object, ttl, "spec", "template", "ttl"); err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to set ttl field: %w", err)), nil
		}
	}

	updated, err := oadp.UpdateSchedule(params.Context, params.DynamicClient(), schedule)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to update schedule: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(updated)), nil
}

func initScheduleDelete() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_schedule_delete",
			Description: "Delete a backup schedule. Existing backups created by this schedule are not affected.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the schedule (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the schedule to delete",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Delete Schedule",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(true),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: scheduleDeleteHandler,
	}
}

func scheduleDeleteHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	err := oadp.DeleteSchedule(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete schedule: %w", err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("Schedule %s/%s deleted", namespace, name), nil), nil
}

func initSchedulePause() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_schedule_pause",
			Description: "Pause or unpause a backup schedule",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the schedule (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the schedule",
					},
					"paused": {
						Type:        "boolean",
						Description: "Set to true to pause, false to unpause",
					},
				},
				Required: []string{"name", "paused"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Pause/Unpause Schedule",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: schedulePauseHandler,
	}
}

func schedulePauseHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	paused, ok := params.GetArguments()["paused"].(bool)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("paused is required")), nil
	}

	updated, err := oadp.PauseSchedule(params.Context, params.DynamicClient(), namespace, name, paused)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to update schedule: %w", err)), nil
	}

	action := "unpaused"
	if paused {
		action = "paused"
	}

	output, outputErr := params.ListOutput.PrintObj(updated)
	if outputErr != nil {
		return api.NewToolCallResult("", outputErr), nil
	}
	return api.NewToolCallResult(fmt.Sprintf("Schedule %s/%s %s\n\n%s", namespace, name, action, output), nil), nil
}
