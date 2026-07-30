package netedge

import (
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/mustgather"
	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
)

func initCoreDNS() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "get_coredns_config",
				Description: "Retrieve the current CoreDNS configuration (Corefile) from the cluster.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
				},
				Annotations: api.ToolAnnotations{
					Title:           "Get CoreDNS Config",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: getCoreDNSConfig,
		},
	}
}

func getCoreDNSConfig(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	gvr := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "configmaps",
	}

	var cm *unstructured.Unstructured
	var err error

	if p, mgErr := mustgather.GetProvider(); mgErr == nil && p != nil {
		gvk := schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"}
		cm = p.GetResource(gvk, "dns-default", "openshift-dns")
		if cm == nil {
			return api.NewToolCallResult("", fmt.Errorf("ConfigMap openshift-dns/dns-default not found in must-gather archive")), nil
		}
	} else {
		cm, err = params.DynamicClient().Resource(gvr).Namespace("openshift-dns").Get(params.Context, "dns-default", metav1.GetOptions{})
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to get dns-default ConfigMap: %w", err)), nil
		}
	}

	data, found, err := unstructured.NestedStringMap(cm.Object, "data")
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to parse ConfigMap data: %w", err)), nil
	}
	if !found {
		return api.NewToolCallResult("", fmt.Errorf("ConfigMap has no data")), nil
	}

	corefile, ok := data["Corefile"]
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("corefile not found in dns-default ConfigMap")), nil
	}

	return api.NewToolCallResult(corefile, nil), nil
}
