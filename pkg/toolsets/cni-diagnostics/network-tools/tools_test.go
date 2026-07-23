package network_tools

import (
	"context"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/config"
	ovnknetmcp "github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/network-tools/mcp"
	"github.com/stretchr/testify/suite"
)

type ToolsSuite struct {
	suite.Suite
}

func TestTools(t *testing.T) {
	suite.Run(t, new(ToolsSuite))
}

type mockToolCallRequest struct {
	args map[string]any
}

func (m *mockToolCallRequest) GetArguments() map[string]any {
	return m.args
}

func (s *ToolsSuite) handlerParams(args map[string]any) api.ToolHandlerParams {
	return api.ToolHandlerParams{
		Context:         context.Background(),
		BaseConfig:      config.BaseDefault(),
		ToolCallRequest: &mockToolCallRequest{args: args},
	}
}

func (s *ToolsSuite) assertReadOnlyNonIdempotentAnnotations(tool api.ServerTool, title string) {
	s.Equal(title, tool.Tool.Annotations.Title)
	s.Require().NotNil(tool.Tool.Annotations.ReadOnlyHint)
	s.True(*tool.Tool.Annotations.ReadOnlyHint)
	s.Require().NotNil(tool.Tool.Annotations.DestructiveHint)
	s.False(*tool.Tool.Annotations.DestructiveHint)
	s.Require().NotNil(tool.Tool.Annotations.IdempotentHint)
	s.False(*tool.Tool.Annotations.IdempotentHint)
}

func (s *ToolsSuite) toolByName(name string) api.ServerTool {
	s.T().Helper()
	for _, tool := range InitNetworkTools() {
		if tool.Tool.Name == name {
			return tool
		}
	}
	s.T().Fatalf("expected tool %q in InitNetworkTools()", name)
	return api.ServerTool{}
}

func (s *ToolsSuite) TestInitNetworkTools() {
	s.Run("returns two network tools", func() {
		tools := InitNetworkTools()
		s.Len(tools, 2)
	})

	s.Run("registers tools with expected names in order", func() {
		tools := InitNetworkTools()
		names := make([]string, len(tools))
		for i, tool := range tools {
			names[i] = tool.Tool.Name
			s.NotNil(tool.Handler, "tool %q should have a handler", tool.Tool.Name)
		}
		s.Equal([]string{"tcpdump", "pwru"}, names)
	})
}

func (s *ToolsSuite) TestInitTcpdump() {
	tool := s.toolByName("tcpdump")

	s.Run("has correct name", func() {
		s.Equal("tcpdump", tool.Tool.Name)
	})

	s.Run("has description", func() {
		s.NotEmpty(tool.Tool.Description)
		s.Contains(tool.Tool.Description, "Capture network packets")
	})

	s.Run("has input schema", func() {
		s.Require().NotNil(tool.Tool.InputSchema)
		s.Equal("object", tool.Tool.InputSchema.Type)
		s.Contains(tool.Tool.InputSchema.Required, "target_type")
		s.Contains(tool.Tool.InputSchema.Required, "name")
	})

	s.Run("has expected parameters", func() {
		props := tool.Tool.InputSchema.Properties
		s.Equal("string", props["target_type"].Type)
		s.Equal("string", props["name"].Type)
		s.Equal("string", props["namespace"].Type)
		s.Equal("string", props["container_name"].Type)
		s.Equal("string", props["interface"].Type)
		s.Equal("integer", props["packet_count"].Type)
		s.Equal("string", props["bpf_filter"].Type)
		s.Equal("integer", props["snaplen"].Type)
		s.Equal("integer", props["timeout_seconds"].Type)
	})

	s.Run("target_type parameter has supported values", func() {
		enum := tool.Tool.InputSchema.Properties["target_type"].Enum
		s.Len(enum, 2)
		s.ElementsMatch([]any{"node", "pod"}, enum)
	})

	s.Run("packet_count has expected bounds", func() {
		prop := tool.Tool.InputSchema.Properties["packet_count"]
		s.Require().NotNil(prop.Minimum)
		s.Equal(float64(0), *prop.Minimum)
		s.Require().NotNil(prop.Maximum)
		s.Equal(float64(ovnknetmcp.MaxPacketCount), *prop.Maximum)
	})

	s.Run("snaplen has expected bounds", func() {
		prop := tool.Tool.InputSchema.Properties["snaplen"]
		s.Require().NotNil(prop.Minimum)
		s.Equal(float64(0), *prop.Minimum)
		s.Require().NotNil(prop.Maximum)
		s.Equal(float64(ovnknetmcp.MaxSnaplen), *prop.Maximum)
	})

	s.Run("has annotations", func() {
		s.assertReadOnlyNonIdempotentAnnotations(tool, "CNI-Diagnostics: Network Packet Capture")
	})

	s.Run("has handler", func() {
		s.NotNil(tool.Handler)
	})
}

func (s *ToolsSuite) TestTcpdumpHandler() {
	tool := s.toolByName("tcpdump")

	s.Run("missing target_type returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "target_type parameter required")
	})

	s.Run("invalid target_type type returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{
			"target_type": 123,
		}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "target_type parameter must be a string")
	})

	s.Run("missing name returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{
			"target_type": "node",
		}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "name parameter required")
	})

	s.Run("invalid name type returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{
			"target_type": "node",
			"name":        42,
		}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "name parameter must be a string")
	})

	s.Run("invalid target_type value returns validation error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{
			"target_type": "vm",
			"name":        "worker-0",
		}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Equal("target_type must be 'node' or 'pod'", result.Error.Error())
	})

	s.Run("empty name returns validation error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{
			"target_type": "node",
			"name":        "",
		}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Equal("name is required when target_type is 'node'", result.Error.Error())
	})

	s.Run("pod target without namespace returns validation error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{
			"target_type": "pod",
			"name":        "my-pod",
		}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Equal("namespace is required when target_type is 'pod'", result.Error.Error())
	})

	s.Run("invalid namespace type returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{
			"target_type": "pod",
			"name":        "my-pod",
			"namespace":   123,
		}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "namespace parameter must be a string")
	})

	s.Run("invalid container_name type returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{
			"target_type":    "pod",
			"name":           "my-pod",
			"namespace":      "default",
			"container_name": 8080,
		}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "container_name parameter must be a string")
	})

	s.Run("invalid interface type returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{
			"target_type": "node",
			"name":        "worker-0",
			"interface":   true,
		}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "interface parameter must be a string")
	})

	s.Run("invalid bpf_filter type returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{
			"target_type": "node",
			"name":        "worker-0",
			"bpf_filter":  8080,
		}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "bpf_filter parameter must be a string")
	})

	s.Run("invalid packet_count type returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{
			"target_type":  "node",
			"name":         "worker-0",
			"packet_count": "many",
		}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "packet_count parameter must be an integer")
	})

	s.Run("negative packet_count returns validation error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{
			"target_type":  "node",
			"name":         "worker-0",
			"packet_count": float64(-1),
		}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Equal("packet_count must be greater than or equal to 0", result.Error.Error())
	})

	s.Run("invalid snaplen type returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{
			"target_type": "node",
			"name":        "worker-0",
			"snaplen":     "large",
		}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "snaplen parameter must be an integer")
	})

	s.Run("negative snaplen returns validation error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{
			"target_type": "node",
			"name":        "worker-0",
			"snaplen":     float64(-1),
		}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Equal("snaplen must be greater than or equal to 0", result.Error.Error())
	})

	s.Run("invalid timeout_seconds type returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{
			"target_type":     "node",
			"name":            "worker-0",
			"timeout_seconds": "long",
		}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "timeout_seconds parameter must be an integer")
	})

	s.Run("negative timeout_seconds returns validation error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{
			"target_type":     "node",
			"name":            "worker-0",
			"timeout_seconds": float64(-1),
		}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Equal("timeout_seconds must be greater than or equal to 0", result.Error.Error())
	})
}

func (s *ToolsSuite) TestInitPwru() {
	tool := s.toolByName("pwru")

	s.Run("has correct name", func() {
		s.Equal("pwru", tool.Tool.Name)
	})

	s.Run("has description", func() {
		s.NotEmpty(tool.Tool.Description)
		s.Contains(tool.Tool.Description, "eBPF")
	})

	s.Run("has input schema", func() {
		s.Require().NotNil(tool.Tool.InputSchema)
		s.Equal("object", tool.Tool.InputSchema.Type)
		s.Contains(tool.Tool.InputSchema.Required, "node_name")
	})

	s.Run("has expected parameters", func() {
		props := tool.Tool.InputSchema.Properties
		s.Equal("string", props["node_name"].Type)
		s.Equal("string", props["node_pod_namespace"].Type)
		s.Equal("string", props["bpf_filter"].Type)
		s.Equal("integer", props["output_limit_lines"].Type)
		s.Equal("integer", props["timeout_seconds"].Type)
	})

	s.Run("output_limit_lines has expected bounds", func() {
		prop := tool.Tool.InputSchema.Properties["output_limit_lines"]
		s.Require().NotNil(prop.Minimum)
		s.Equal(float64(0), *prop.Minimum)
		s.Require().NotNil(prop.Maximum)
		s.Equal(float64(ovnknetmcp.MaxOutputLimitLines), *prop.Maximum)
	})

	s.Run("has annotations", func() {
		s.assertReadOnlyNonIdempotentAnnotations(tool, "CNI-Diagnostics: Network eBPF Packet Tracing")
	})

	s.Run("has handler", func() {
		s.NotNil(tool.Handler)
	})
}

func (s *ToolsSuite) TestPwruHandler() {
	tool := s.toolByName("pwru")

	s.Run("missing node_name returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "node_name parameter required")
	})

	s.Run("invalid node_name type returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{
			"node_name": 42,
		}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "node_name parameter must be a string")
	})

	s.Run("invalid node_pod_namespace type returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{
			"node_name":          "worker-0",
			"node_pod_namespace": 123,
		}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "node_pod_namespace parameter must be a string")
	})

	s.Run("invalid bpf_filter type returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{
			"node_name":  "worker-0",
			"bpf_filter": 8080,
		}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "bpf_filter parameter must be a string")
	})

	s.Run("invalid output_limit_lines type returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{
			"node_name":          "worker-0",
			"output_limit_lines": "many",
		}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "output_limit_lines parameter must be an integer")
	})

	s.Run("negative output_limit_lines returns validation error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{
			"node_name":          "worker-0",
			"output_limit_lines": float64(-1),
		}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Equal("output_limit_lines must be greater than or equal to 0", result.Error.Error())
	})

	s.Run("invalid timeout_seconds type returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{
			"node_name":       "worker-0",
			"timeout_seconds": "long",
		}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "timeout_seconds parameter must be an integer")
	})

	s.Run("negative timeout_seconds returns validation error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{
			"node_name":       "worker-0",
			"timeout_seconds": float64(-1),
		}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Equal("timeout_seconds must be greater than or equal to 0", result.Error.Error())
	})
}
