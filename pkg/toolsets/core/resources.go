package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	internalk8s "github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/containers/kubernetes-mcp-server/pkg/output"
)

func initResources(o internalk8s.Openshift) []api.ServerTool {
	commonApiVersion := "v1 Pod, v1 Service, v1 Node, apps/v1 Deployment, networking.k8s.io/v1 Ingress"
	if o.IsOpenShift(context.Background()) {
		commonApiVersion += ", route.openshift.io/v1 Route"
	}
	commonApiVersion = fmt.Sprintf("(common apiVersion and kind include: %s)", commonApiVersion)
	return []api.ServerTool{
		{Tool: api.Tool{
			Name:        "resources_list",
			Description: "List Kubernetes resources and objects in the current cluster by providing their apiVersion and kind and optionally the namespace and label selector\n" + commonApiVersion,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"apiVersion": {
						Type:        "string",
						Description: "apiVersion of the resources (examples of valid apiVersion are: v1, apps/v1, networking.k8s.io/v1)",
					},
					"kind": {
						Type:        "string",
						Description: "kind of the resources (examples of valid kind are: Pod, Service, Deployment, Ingress)",
					},
					"namespace": {
						Type:        "string",
						Description: "Optional Namespace to retrieve the namespaced resources from (ignored in case of cluster scoped resources). If not provided, will list resources from all namespaces",
					},
					"labelSelector": {
						Type:        "string",
						Description: "Optional Kubernetes label selector (e.g. 'app=myapp,env=prod' or 'app in (myapp,yourapp)'), use this option when you want to filter the pods by label",
						Pattern:     "([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]",
					},
				},
				Required: []string{"apiVersion", "kind"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "Resources: List",
				ReadOnlyHint:    ptr.To(true),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(false),
				OpenWorldHint:   ptr.To(true),
			},
		}, Handler: resourcesList},
		{Tool: api.Tool{
			Name:        "resources_get",
			Description: "Get a Kubernetes resource in the current cluster by providing its apiVersion, kind, optionally the namespace, and its name\n" + commonApiVersion,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"apiVersion": {
						Type:        "string",
						Description: "apiVersion of the resource (examples of valid apiVersion are: v1, apps/v1, networking.k8s.io/v1)",
					},
					"kind": {
						Type:        "string",
						Description: "kind of the resource (examples of valid kind are: Pod, Service, Deployment, Ingress)",
					},
					"namespace": {
						Type:        "string",
						Description: "Optional Namespace to retrieve the namespaced resource from (ignored in case of cluster scoped resources). If not provided, will get resource from configured namespace",
					},
					"name": {
						Type:        "string",
						Description: "Name of the resource",
					},
				},
				Required: []string{"apiVersion", "kind", "name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "Resources: Get",
				ReadOnlyHint:    ptr.To(true),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(false),
				OpenWorldHint:   ptr.To(true),
			},
		}, Handler: resourcesGet},
		{Tool: api.Tool{
			Name:        "resources_create_or_update",
			Description: "Create or update a Kubernetes resource in the current cluster by providing a YAML or JSON representation of the resource\n" + commonApiVersion,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"resource": {
						Type:        "string",
						Description: "A JSON or YAML containing a representation of the Kubernetes resource. Should include top-level fields such as apiVersion,kind,metadata, and spec",
					},
				},
				Required: []string{"resource"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "Resources: Create or Update",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(true),
				IdempotentHint:  ptr.To(true),
				OpenWorldHint:   ptr.To(true),
			},
		}, Handler: resourcesCreateOrUpdate},
		{Tool: api.Tool{
			Name:        "resources_delete",
			Description: "Delete a Kubernetes resource in the current cluster by providing its apiVersion, kind, optionally the namespace, and its name\n" + commonApiVersion,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"apiVersion": {
						Type:        "string",
						Description: "apiVersion of the resource (examples of valid apiVersion are: v1, apps/v1, networking.k8s.io/v1)",
					},
					"kind": {
						Type:        "string",
						Description: "kind of the resource (examples of valid kind are: Pod, Service, Deployment, Ingress)",
					},
					"namespace": {
						Type:        "string",
						Description: "Optional Namespace to delete the namespaced resource from (ignored in case of cluster scoped resources). If not provided, will delete resource from configured namespace",
					},
					"name": {
						Type:        "string",
						Description: "Name of the resource",
					},
				},
				Required: []string{"apiVersion", "kind", "name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "Resources: Delete",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(true),
				IdempotentHint:  ptr.To(true),
				OpenWorldHint:   ptr.To(true),
			},
		}, Handler: resourcesDelete},
	}
}

func resourcesList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := params.GetArguments()["namespace"]
	if namespace == nil {
		namespace = ""
	}
	labelSelector := params.GetArguments()["labelSelector"]
	resourceListOptions := internalk8s.ResourceListOptions{
		AsTable: params.ListOutput.AsTable(),
	}

	if labelSelector != nil {
		l, ok := labelSelector.(string)
		if !ok {
			return api.NewToolCallResult("", fmt.Errorf("labelSelector is not a string")), nil
		}
		resourceListOptions.LabelSelector = l
	}
	gvk, err := parseGroupVersionKind(params.GetArguments())
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list resources, %s", err)), nil
	}

	ns, ok := namespace.(string)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("namespace is not a string")), nil
	}

	ret, err := params.ResourcesList(params, gvk, ns, resourceListOptions)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list resources: %v", err)), nil
	}
	return api.NewToolCallResult(params.ListOutput.PrintObj(ret)), nil
}

func resourcesGet(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := params.GetArguments()["namespace"]
	if namespace == nil {
		namespace = ""
	}
	gvk, err := parseGroupVersionKind(params.GetArguments())
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get resource, %s", err)), nil
	}
	name := params.GetArguments()["name"]
	if name == nil {
		return api.NewToolCallResult("", errors.New("failed to get resource, missing argument name")), nil
	}

	ns, ok := namespace.(string)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("namespace is not a string")), nil
	}

	n, ok := name.(string)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("name is not a string")), nil
	}

	ret, err := params.ResourcesGet(params, gvk, ns, n)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get resource: %v", err)), nil
	}
	return api.NewToolCallResult(output.MarshalYaml(ret)), nil
}

func resourcesCreateOrUpdate(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	resource := params.GetArguments()["resource"]
	if resource == nil || resource == "" {
		return api.NewToolCallResult("", errors.New("failed to create or update resources, missing argument resource")), nil
	}

	r, ok := resource.(string)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("resource is not a string")), nil
	}

	resources, err := params.ResourcesCreateOrUpdate(params, r)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create or update resources: %v", err)), nil
	}
	marshalledYaml, err := output.MarshalYaml(resources)
	if err != nil {
		err = fmt.Errorf("failed to create or update resources:: %v", err)
	}
	return api.NewToolCallResult("# The following resources (YAML) have been created or updated successfully\n"+marshalledYaml, err), nil
}

func resourcesDelete(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := params.GetArguments()["namespace"]
	if namespace == nil {
		namespace = ""
	}
	gvk, err := parseGroupVersionKind(params.GetArguments())
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete resource, %s", err)), nil
	}
	name := params.GetArguments()["name"]
	if name == nil {
		return api.NewToolCallResult("", errors.New("failed to delete resource, missing argument name")), nil
	}

	ns, ok := namespace.(string)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("namespace is not a string")), nil
	}

	n, ok := name.(string)
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("name is not a string")), nil
	}

	err = params.ResourcesDelete(params, gvk, ns, n)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete resource: %v", err)), nil
	}
	return api.NewToolCallResult("Resource deleted successfully", err), nil
}

func parseGroupVersionKind(arguments map[string]interface{}) (*schema.GroupVersionKind, error) {
	apiVersion := arguments["apiVersion"]
	if apiVersion == nil {
		return nil, errors.New("missing argument apiVersion")
	}
	kind := arguments["kind"]
	if kind == nil {
		return nil, errors.New("missing argument kind")
	}

	a, ok := apiVersion.(string)
	if !ok {
		return nil, fmt.Errorf("name is not a string")
	}

	gv, err := schema.ParseGroupVersion(a)
	if err != nil {
		return nil, errors.New("invalid argument apiVersion")
	}
	return &schema.GroupVersionKind{Group: gv.Group, Version: gv.Version, Kind: kind.(string)}, nil
}
