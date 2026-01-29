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

func initServerStatusRequestTools() []api.ServerTool {
	return []api.ServerTool{
		initServerStatusRequestList(),
		initServerStatusRequestGet(),
		initServerStatusRequestCreate(),
		initServerStatusRequestDelete(),
	}
}

func initServerStatusRequestList() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_server_status_request_list",
			Description: "List all ServerStatusRequests to check Velero server health",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace containing ServerStatusRequests (default: openshift-adp)",
					},
				},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: List Server Status Requests",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: serverStatusRequestListHandler,
	}
}

func serverStatusRequestListHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	ssrs, err := oadp.ListServerStatusRequests(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list server status requests: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(ssrs)), nil
}

func initServerStatusRequestGet() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_server_status_request_get",
			Description: "Get detailed information about a ServerStatusRequest including server version and plugins",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the ServerStatusRequest (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the ServerStatusRequest",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Get Server Status Request",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: serverStatusRequestGetHandler,
	}
}

func serverStatusRequestGetHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	ssr, err := oadp.GetServerStatusRequest(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get server status request: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(ssr)), nil
}

func initServerStatusRequestCreate() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_server_status_request_create",
			Description: "Create a ServerStatusRequest to check Velero server health and get server information",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace for the ServerStatusRequest (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name for the ServerStatusRequest",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Create Server Status Request",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(false),
			},
		},
		Handler: serverStatusRequestCreateHandler,
	}
}

func serverStatusRequestCreateHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	ssr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": oadp.VeleroGroup + "/" + oadp.VeleroVersion,
			"kind":       "ServerStatusRequest",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{},
		},
	}

	created, err := oadp.CreateServerStatusRequest(params.Context, params.DynamicClient(), ssr)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create server status request: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(created)), nil
}

func initServerStatusRequestDelete() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_server_status_request_delete",
			Description: "Delete a ServerStatusRequest",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the ServerStatusRequest (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the ServerStatusRequest to delete",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Delete Server Status Request",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(true),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: serverStatusRequestDeleteHandler,
	}
}

func serverStatusRequestDeleteHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	err := oadp.DeleteServerStatusRequest(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete server status request: %w", err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("ServerStatusRequest %s/%s deleted", namespace, name), nil), nil
}
