package troubleshoot

import (
	"context"
	"fmt"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/containers/kubernetes-mcp-server/pkg/kubevirt"
	"github.com/containers/kubernetes-mcp-server/pkg/output"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/kubevirt/internal/defaults"
	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/utils/ptr"
)

func Tools() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name: "vm_troubleshoot",
				Description: fmt.Sprintf(
					"Diagnose %s VirtualMachine issues by collecting VM status, VMI status, volumes, virt-launcher pod state, pod logs, and related events. "+
						"Returns a structured diagnostic report. Use this tool whenever a user asks why a VM is not starting, crashing, failing to migrate, or exhibiting unexpected behavior. "+
						"This tool should be invoked proactively in troubleshooting mode.",
					defaults.ProductName()),
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "The namespace of the VirtualMachine to troubleshoot",
						},
						"name": {
							Type:        "string",
							Description: "The name of the VirtualMachine to troubleshoot",
						},
					},
					Required: []string{"namespace", "name"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Virtual Machine: Troubleshoot",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					IdempotentHint:  ptr.To(true),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: troubleshoot,
		},
	}
}

func troubleshoot(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace, err := api.RequiredString(params, "namespace")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	name, err := api.RequiredString(params, "name")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	ctx := params.Context
	if ctx == nil {
		ctx = context.Background()
	}

	dynamicClient := params.DynamicClient()

	vmYaml, vm := fetchVMStatus(ctx, dynamicClient, namespace, name)
	vmiYaml, vmi := fetchVMIStatus(ctx, dynamicClient, namespace, name)
	volumesYaml := fetchVolumes(namespace, name, vm, vmi)
	podYaml, podNames := fetchVirtLauncherPod(ctx, dynamicClient, namespace, name)
	podLogsText := fetchVirtLauncherPodLogs(ctx, params.KubernetesClient, namespace, podNames)
	eventsYaml := fetchEvents(ctx, params.KubernetesClient, namespace, name)

	report := fmt.Sprintf(`# VirtualMachine Diagnostic Report: %s/%s

%s

%s

%s

%s

%s

%s
`, namespace, name, vmYaml, vmiYaml, volumesYaml, podYaml, podLogsText, eventsYaml)

	return api.NewToolCallResult(report, nil), nil
}

func fetchVMStatus(ctx context.Context, dynamicClient dynamic.Interface, namespace, name string) (string, *unstructured.Unstructured) {
	vm, err := dynamicClient.Resource(kubevirt.VirtualMachineGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Sprintf("## VirtualMachine\n\n*Error: %v*", err), nil
	}

	status, found, err := unstructured.NestedMap(vm.Object, "status")
	if err != nil {
		return fmt.Sprintf("## VirtualMachine\n\n*Error extracting status: %v*", err), vm
	}
	if !found {
		return fmt.Sprintf("## VirtualMachine: %s/%s\n\n*No status found (VM may not have been reconciled yet)*", namespace, name), vm
	}

	yamlStr, err := output.MarshalYaml(status)
	if err != nil {
		return fmt.Sprintf("## VirtualMachine\n\n*Error marshaling status: %v*", err), vm
	}

	return fmt.Sprintf("## VirtualMachine Status: %s/%s\n\n```yaml\n%s```", namespace, name, yamlStr), vm
}

func fetchVMIStatus(ctx context.Context, dynamicClient dynamic.Interface, namespace, name string) (string, *unstructured.Unstructured) {
	vmi, err := dynamicClient.Resource(kubevirt.VirtualMachineInstanceGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Sprintf("## VirtualMachineInstance\n\n*VMI not found: %v*\n\n(Expected if VM is stopped or stuck provisioning)", err), nil
	}

	status, found, err := unstructured.NestedMap(vmi.Object, "status")
	if err != nil {
		return fmt.Sprintf("## VirtualMachineInstance\n\n*Error extracting status: %v*", err), vmi
	}
	if !found {
		return fmt.Sprintf("## VirtualMachineInstance: %s/%s\n\n*No status found*", namespace, name), vmi
	}

	yamlStr, err := output.MarshalYaml(status)
	if err != nil {
		return fmt.Sprintf("## VirtualMachineInstance\n\n*Error marshaling status: %v*", err), vmi
	}

	return fmt.Sprintf("## VirtualMachineInstance Status: %s/%s\n\n```yaml\n%s```", namespace, name, yamlStr), vmi
}

func fetchVolumes(namespace, name string, vm, vmi *unstructured.Unstructured) string {
	var volumes []any
	var found bool
	var err error
	source := "VirtualMachine"

	if vm != nil {
		volumes, found, err = unstructured.NestedSlice(vm.Object, "spec", "template", "spec", "volumes")
		if err != nil {
			return "*Error extracting volumes from VirtualMachine: " + err.Error() + "*"
		}
	}

	if (!found || len(volumes) == 0) && vmi != nil {
		volumes, found, err = unstructured.NestedSlice(vmi.Object, "spec", "volumes")
		if err != nil {
			return "*Error extracting volumes from VirtualMachineInstance: " + err.Error() + "*"
		}
		if found && len(volumes) > 0 {
			source = "VirtualMachineInstance"
		}
	}

	if !found || len(volumes) == 0 {
		return "## Volumes\n\n*No volumes configured*"
	}

	yamlStr, err := output.MarshalYaml(volumes)
	if err != nil {
		return "*Error marshaling volumes: " + err.Error() + "*"
	}

	return fmt.Sprintf("## Volumes (from %s: %s/%s)\n\n```yaml\n%s```", source, namespace, name, yamlStr)
}

func fetchVirtLauncherPod(ctx context.Context, dynamicClient dynamic.Interface, namespace, name string) (string, []string) {
	labelSelector := fmt.Sprintf("kubevirt.io=virt-launcher,vm.kubevirt.io/name=%s", name)
	podList, err := dynamicClient.Resource(kubevirt.PodGVR).Namespace(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Sprintf("## virt-launcher Pod\n\n*Error listing pods: %v*", err), nil
	}

	if len(podList.Items) == 0 {
		return "## virt-launcher Pod\n\n*No virt-launcher pod found (VM may be stopped or not yet scheduled)*", nil
	}

	var result strings.Builder
	var podNames []string
	result.WriteString("## virt-launcher Pod\n\n")
	for _, pod := range podList.Items {
		podNames = append(podNames, pod.GetName())
		yamlStr, err := output.MarshalYaml(&pod)
		if err != nil {
			fmt.Fprintf(&result, "*Error marshaling pod %s: %v*\n\n", pod.GetName(), err)
			continue
		}
		fmt.Fprintf(&result, "### %s\n\n```yaml\n%s```\n\n", pod.GetName(), yamlStr)
	}

	return result.String(), podNames
}

func fetchVirtLauncherPodLogs(ctx context.Context, client api.KubernetesClient, namespace string, podNames []string) string {
	if len(podNames) == 0 {
		return "## virt-launcher Pod Logs\n\n*No pod found — no logs available*"
	}

	core := kubernetes.NewCore(client)
	var result strings.Builder
	result.WriteString("## virt-launcher Pod Logs\n\n")

	containerName := "compute"
	for _, podName := range podNames {
		logs, err := core.PodsLog(ctx, namespace, podName, containerName, false, 50)
		if err != nil {
			fmt.Fprintf(&result, "### %s\n\n*Error fetching logs: %v*\n\n", podName, err)
			continue
		}
		fmt.Fprintf(&result, "### %s (container: %s)\n\n```\n%s\n```\n\n", podName, containerName, logs)
	}

	return result.String()
}

func fetchEvents(ctx context.Context, client api.KubernetesClient, namespace, vmName string) string {
	core := kubernetes.NewCore(client)

	vmEvents, err := core.EventsList(ctx, namespace, api.ListOptions{
		ListOptions: metav1.ListOptions{FieldSelector: "involvedObject.name=" + vmName},
	})
	if err != nil {
		return fmt.Sprintf("## Events\n\n*Error listing events: %v*", err)
	}

	var relatedEvents []map[string]any
	for _, event := range vmEvents {
		involvedObj, ok := event["InvolvedObject"].(map[string]string)
		if !ok {
			continue
		}
		objKind := involvedObj["Kind"]
		if objKind == "VirtualMachine" || objKind == "VirtualMachineInstance" {
			relatedEvents = append(relatedEvents, event)
		}
	}

	allEvents, err := core.EventsList(ctx, namespace, api.ListOptions{})
	if err != nil {
		return fmt.Sprintf("## Events\n\n*Error listing events: %v*", err)
	}
	for _, event := range allEvents {
		involvedObj, ok := event["InvolvedObject"].(map[string]string)
		if !ok {
			continue
		}
		objName := involvedObj["Name"]
		if strings.HasPrefix(objName, vmName+"-") || strings.HasPrefix(objName, "virt-launcher-"+vmName) {
			relatedEvents = append(relatedEvents, event)
		}
	}

	if len(relatedEvents) == 0 {
		return "## Events\n\n*No events found related to this VM*"
	}

	yamlStr, err := output.MarshalYaml(relatedEvents)
	if err != nil {
		return fmt.Sprintf("## Events\n\n*Error marshaling events: %v*", err)
	}

	return fmt.Sprintf("## Events (related to %s)\n\n```yaml\n%s```", vmName, yamlStr)
}
