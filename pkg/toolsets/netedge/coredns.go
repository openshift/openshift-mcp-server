package netedge

import (
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/google/jsonschema-go/jsonschema"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

// newClientFunc is the function used to create a controller-runtime client.
// It is a variable to allow overriding in tests.
var newClientFunc = func(config *rest.Config, options client.Options) (client.Client, error) {
	return client.New(config, options)
}

func getCoreDNSConfig(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	cfg := params.RESTConfig()
	if cfg == nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get REST config")), nil
	}

	cl, err := newClientFunc(cfg, client.Options{Scheme: kubernetes.Scheme})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create controller-runtime client: %w", err)), nil
	}

	cm := &corev1.ConfigMap{}
	err = cl.Get(params.Context, types.NamespacedName{Name: "dns-default", Namespace: "openshift-dns"}, cm)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get dns-default ConfigMap: %w", err)), nil
	}

	corefile, ok := cm.Data["Corefile"]
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("Corefile not found in dns-default ConfigMap")), nil
	}

	return api.NewToolCallResult(corefile, nil), nil
}
