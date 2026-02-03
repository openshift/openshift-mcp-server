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

// ScheduleAction represents the action to perform on schedules
type ScheduleAction string

const (
	ScheduleActionList   ScheduleAction = "list"
	ScheduleActionGet    ScheduleAction = "get"
	ScheduleActionCreate ScheduleAction = "create"
	ScheduleActionUpdate ScheduleAction = "update"
	ScheduleActionDelete ScheduleAction = "delete"
	ScheduleActionPause  ScheduleAction = "pause"
)

func initScheduleTools() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "oadp_schedule",
				Description: "Manage Velero/OADP backup schedules: list, get, create, update, delete, or pause/unpause",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"action": {
							Type:        "string",
							Enum:        []any{string(ScheduleActionList), string(ScheduleActionGet), string(ScheduleActionCreate), string(ScheduleActionUpdate), string(ScheduleActionDelete), string(ScheduleActionPause)},
							Description: "Action to perform: 'list', 'get', 'create', 'update', 'delete', or 'pause' (toggle pause state)",
						},
						"namespace": {
							Type:        "string",
							Description: "Namespace containing schedules (default: openshift-adp)",
						},
						"name": {
							Type:        "string",
							Description: "Name of the schedule (required for get, create, update, delete, pause)",
						},
						"schedule": {
							Type:        "string",
							Description: "Cron expression e.g., '0 1 * * *' for daily at 1am (for create/update action)",
						},
						"includedNamespaces": {
							Type:        "array",
							Description: "Namespaces to include in scheduled backups (for create action)",
							Items:       &jsonschema.Schema{Type: "string"},
						},
						"excludedNamespaces": {
							Type:        "array",
							Description: "Namespaces to exclude from scheduled backups (for create action)",
							Items:       &jsonschema.Schema{Type: "string"},
						},
						"includedResources": {
							Type:        "array",
							Description: "Resource types to include (for create action)",
							Items:       &jsonschema.Schema{Type: "string"},
						},
						"excludedResources": {
							Type:        "array",
							Description: "Resource types to exclude (for create action)",
							Items:       &jsonschema.Schema{Type: "string"},
						},
						"storageLocation": {
							Type:        "string",
							Description: "BackupStorageLocation name (for create action)",
						},
						"ttl": {
							Type:        "string",
							Description: "Backup TTL duration e.g., '720h' for 30 days (for create/update action)",
						},
						"paused": {
							Type:        "boolean",
							Description: "Set to true to pause, false to unpause (for pause action)",
						},
					},
					Required: []string{"action"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "OADP: Schedule",
					ReadOnlyHint:    ptr.To(false),
					DestructiveHint: ptr.To(false),
				},
			},
			Handler: scheduleHandler,
		},
	}
}

func scheduleHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	action, err := api.RequiredString(params, "action")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	switch ScheduleAction(action) {
	case ScheduleActionList:
		return handleScheduleList(params, namespace)
	case ScheduleActionGet:
		return handleScheduleGet(params, namespace)
	case ScheduleActionCreate:
		return handleScheduleCreate(params, namespace)
	case ScheduleActionUpdate:
		return handleScheduleUpdate(params, namespace)
	case ScheduleActionDelete:
		return handleScheduleDelete(params, namespace)
	case ScheduleActionPause:
		return handleSchedulePause(params, namespace)
	default:
		return api.NewToolCallResult("", fmt.Errorf("invalid action '%s': must be one of 'list', 'get', 'create', 'update', 'delete', 'pause'", action)), nil
	}
}

func handleScheduleList(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	schedules, err := oadp.ListSchedules(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list schedules: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(schedules)), nil
}

func handleScheduleGet(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required for get action")), nil
	}

	schedule, err := oadp.GetSchedule(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get schedule: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(schedule)), nil
}

func handleScheduleCreate(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required for create action")), nil
	}

	cronSchedule, ok := params.GetArguments()["schedule"].(string)
	if !ok || cronSchedule == "" {
		return api.NewToolCallResult("", fmt.Errorf("schedule (cron expression) is required for create action")), nil
	}

	template := map[string]any{}

	if v, ok := params.GetArguments()["includedNamespaces"].([]any); ok {
		template["includedNamespaces"] = v
	}
	if v, ok := params.GetArguments()["excludedNamespaces"].([]any); ok {
		template["excludedNamespaces"] = v
	}
	if v, ok := params.GetArguments()["includedResources"].([]any); ok {
		template["includedResources"] = v
	}
	if v, ok := params.GetArguments()["excludedResources"].([]any); ok {
		template["excludedResources"] = v
	}
	if v, ok := params.GetArguments()["storageLocation"].(string); ok && v != "" {
		template["storageLocation"] = v
	}
	if v, ok := params.GetArguments()["ttl"].(string); ok && v != "" {
		template["ttl"] = v
	}

	schedule := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": oadp.VeleroGroup + "/" + oadp.VeleroVersion,
			"kind":       "Schedule",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
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

func handleScheduleUpdate(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required for update action")), nil
	}

	schedule, err := oadp.GetSchedule(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get schedule: %w", err)), nil
	}

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

func handleScheduleDelete(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required for delete action")), nil
	}

	err := oadp.DeleteSchedule(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete schedule: %w", err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("Schedule %s/%s deleted", namespace, name), nil), nil
}

func handleSchedulePause(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required for pause action")), nil
	}

	paused, ok := params.GetArguments()["paused"].(bool)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("paused (true/false) is required for pause action")), nil
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
