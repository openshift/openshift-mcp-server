package utils

import (
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/cni-diagnostics/adapter"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/cni-diagnostics/config"
	"github.com/google/jsonschema-go/jsonschema"
	ovnkkernelmcp "github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/kernel/mcp"
	ovnknetmcp "github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/network-tools/mcp"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils/headtail"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils/timeout"

	"k8s.io/utils/ptr"
)

// ParamsErrorResult returns an invalid-parameters result when p.Err() is non-nil.
// The third return value is true when an error result was produced.
func ParamsErrorResult(p *api.Params) (*api.ToolCallResult, error, bool) {
	if err := p.Err(); err != nil {
		return api.NewToolCallResult("", fmt.Errorf("invalid parameters: %w", err)), nil, true
	}
	return nil, nil, false
}

// ValidationErrorResult builds a tool call result for a validation failure message.
func ValidationErrorResult(message string) (*api.ToolCallResult, error) {
	return api.NewToolCallResult("", fmt.Errorf("%s", message)), nil
}

// ValidateHeadTail populates ht from head, tail, and applyTailFirst.
// Returns a validation error result when head or tail is negative.
func ValidateHeadTail(ht *headtail.HeadTailParams, head, tail int, applyTailFirst bool) (*api.ToolCallResult, error) {
	if head >= 0 {
		ht.Head = head
	} else {
		return ValidationErrorResult("head must be greater than or equal to 0")
	}
	if tail >= 0 {
		ht.Tail = tail
	} else {
		return ValidationErrorResult("tail must be greater than or equal to 0")
	}
	if applyTailFirst {
		ht.ApplyTailFirst = true
	}
	return nil, nil
}

// ValidateTimeout sets tp.TimeoutSeconds from timeoutSecs, capped at the upstream maximum.
// Returns a validation error result when timeoutSecs is negative.
func ValidateTimeout(tp *timeout.TimeoutParams, timeoutSecs int64) (*api.ToolCallResult, error) {
	if timeoutSecs >= 0 {
		tp.TimeoutSeconds = uint32(min(timeoutSecs, int64(timeout.MaxTimeout.Seconds())))
		return nil, nil
	}
	return ValidationErrorResult("timeout_seconds must be greater than or equal to 0")
}

// NormalizeBoundedInt clamps value to maxVal, applies defaultWhenZero when value is zero,
// and returns a validation error result when value is negative.
func NormalizeBoundedInt(fieldName string, value, maxVal, defaultWhenZero int) (int, *api.ToolCallResult, error) {
	if value >= 0 {
		value = min(value, maxVal)
		if value == 0 {
			value = defaultWhenZero
		}
		return value, nil, nil
	}
	result, err := ValidationErrorResult(fmt.Sprintf("%s must be greater than or equal to 0", fieldName))
	return 0, result, err
}

// KernelDebugImage returns the container image for kernel debug node commands,
// using toolset configuration when present.
func KernelDebugImage(params api.ToolHandlerParams) string {
	image := config.DefaultNetshootImage
	if cniCfg := config.GetConfig(params); cniCfg != nil {
		image = cniCfg.KernelDebugImage
	}
	return image
}

// TcpdumpImage returns the container image for tcpdump node commands,
// using toolset configuration when present.
func TcpdumpImage(params api.ToolHandlerParams) string {
	image := config.DefaultNetshootImage
	if cniCfg := config.GetConfig(params); cniCfg != nil {
		image = cniCfg.TcpdumpImage
	}
	return image
}

// PwruImage returns the container image for pwru node commands,
// using toolset configuration when present.
func PwruImage(params api.ToolHandlerParams) string {
	image := config.DefaultPwruImage
	if cniCfg := config.GetConfig(params); cniCfg != nil {
		image = cniCfg.PwruImage
	}
	return image
}

// NewKernelMCPServer creates an upstream kernel MCP server wired to kubernetes-mcp-server
// node debug execution.
func NewKernelMCPServer(params api.ToolHandlerParams, image string) (*ovnkkernelmcp.MCPServer, error) {
	return ovnkkernelmcp.NewMCPServer(
		adapter.NewRunDebugNodeCommand(params.KubernetesClient),
		ovnkkernelmcp.Config{Image: image},
	)
}

// NewNetworkToolsMCPServer creates an upstream network-tools MCP server wired to
// kubernetes-mcp-server node debug and pod exec execution.
func NewNetworkToolsMCPServer(params api.ToolHandlerParams, cfg ovnknetmcp.Config) (*ovnknetmcp.MCPServer, error) {
	return ovnknetmcp.NewMCPServer(
		adapter.NewRunDebugNodeCommand(params.KubernetesClient),
		adapter.NewRunPodExecCommand(params.KubernetesClient),
		cfg,
	)
}

// FormatCommandOutput returns output unchanged when stderr is empty; otherwise it
// labels stdout and stderr in a single string.
func FormatCommandOutput(output, stderr string) string {
	if stderr == "" {
		return output
	}
	return fmt.Sprintf("-- stdout --\n%s\n-- stderr --\n%s", output, stderr)
}

// AddHeadTailProperties adds head, tail, and apply_tail_first JSON schema properties to props.
func AddHeadTailProperties(props map[string]*jsonschema.Schema, headDescription string) {
	props["head"] = &jsonschema.Schema{
		Type:        "integer",
		Description: headDescription,
		Minimum:     ptr.To(float64(0)),
	}
	props["tail"] = &jsonschema.Schema{
		Type:        "integer",
		Description: "Return only last N lines",
		Minimum:     ptr.To(float64(0)),
	}
	props["apply_tail_first"] = &jsonschema.Schema{
		Type:        "boolean",
		Description: "If both head and tail are set and apply_tail_first is true, apply tail before head. Default: false",
	}
}

// AddTimeoutSecondsProperty adds the timeout_seconds JSON schema property to props.
func AddTimeoutSecondsProperty(props map[string]*jsonschema.Schema) {
	props["timeout_seconds"] = TimeoutSecondsSchema()
}

// TimeoutSecondsSchema returns the JSON schema for the timeout_seconds tool parameter.
func TimeoutSecondsSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:        "integer",
		Description: fmt.Sprintf("Timeout in seconds for the command execution. If not specified, server default timeout is used. The maximum value is %d seconds.", uint32(timeout.MaxTimeout.Seconds())),
		Minimum:     ptr.To(float64(0)),
		Maximum:     ptr.To(float64(timeout.MaxTimeout.Seconds())),
	}
}

// ReadOnlyAnnotations returns MCP tool annotations for read-only CNI Diagnostics tools.
func ReadOnlyAnnotations(title string, idempotent bool) api.ToolAnnotations {
	return api.ToolAnnotations{
		Title:           title,
		ReadOnlyHint:    ptr.To(true),
		DestructiveHint: ptr.To(false),
		IdempotentHint:  ptr.To(idempotent),
		OpenWorldHint:   ptr.To(true),
	}
}
