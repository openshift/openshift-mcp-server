package kernel

import (
	"context"
	"strings"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/config"
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

func (s *ToolsSuite) assertReadOnlyToolAnnotations(tool api.ServerTool, title string) {
	s.Equal(title, tool.Tool.Annotations.Title)
	s.Require().NotNil(tool.Tool.Annotations.ReadOnlyHint)
	s.True(*tool.Tool.Annotations.ReadOnlyHint)
	s.Require().NotNil(tool.Tool.Annotations.DestructiveHint)
	s.False(*tool.Tool.Annotations.DestructiveHint)
	s.Require().NotNil(tool.Tool.Annotations.IdempotentHint)
	s.True(*tool.Tool.Annotations.IdempotentHint)
}

func (s *ToolsSuite) assertCommonOutputParams(schema *api.Tool) {
	s.NotNil(schema.InputSchema.Properties["head"])
	s.Equal("integer", schema.InputSchema.Properties["head"].Type)
	s.NotNil(schema.InputSchema.Properties["tail"])
	s.Equal("integer", schema.InputSchema.Properties["tail"].Type)
	s.NotNil(schema.InputSchema.Properties["apply_tail_first"])
	s.Equal("boolean", schema.InputSchema.Properties["apply_tail_first"].Type)
	s.NotNil(schema.InputSchema.Properties["timeout_seconds"])
	s.Equal("integer", schema.InputSchema.Properties["timeout_seconds"].Type)
}

func (s *ToolsSuite) toolByName(name string) api.ServerTool {
	s.T().Helper()
	for _, tool := range InitKernelTools() {
		if tool.Tool.Name == name {
			return tool
		}
	}
	s.T().Fatalf("expected tool %q in InitKernelTools()", name)
	return api.ServerTool{}
}

func (s *ToolsSuite) TestInitKernelTools() {
	s.Run("returns four kernel tools", func() {
		tools := InitKernelTools()
		s.Len(tools, 4)
	})

	s.Run("registers tools with expected names in order", func() {
		tools := InitKernelTools()
		names := make([]string, len(tools))
		for i, tool := range tools {
			names[i] = tool.Tool.Name
			s.NotNil(tool.Handler, "tool %q should have a handler", tool.Tool.Name)
		}
		s.Equal([]string{"get-conntrack", "get-iptables", "get-nft", "get-ip"}, names)
	})
}

func (s *ToolsSuite) TestInitConntrack() {
	tool := s.toolByName("get-conntrack")

	s.Run("has correct name", func() {
		s.Equal("get-conntrack", tool.Tool.Name)
	})

	s.Run("has description", func() {
		s.NotEmpty(tool.Tool.Description)
		s.Contains(tool.Tool.Description, "connection tracking")
	})

	s.Run("has input schema", func() {
		s.Require().NotNil(tool.Tool.InputSchema)
		s.Equal("object", tool.Tool.InputSchema.Type)
		s.Contains(tool.Tool.InputSchema.Required, "node")
	})

	s.Run("has expected parameters", func() {
		props := tool.Tool.InputSchema.Properties
		s.Equal("string", props["node"].Type)
		s.Equal("string", props["namespace"].Type)
		s.Equal("string", props["command"].Type)
		s.Equal("string", props["filter_parameters"].Type)
		s.assertCommonOutputParams(&tool.Tool)
	})

	s.Run("command parameter has conntrack operations", func() {
		enum := tool.Tool.InputSchema.Properties["command"].Enum
		s.Len(enum, 6)
		s.ElementsMatch([]any{"-L", "--dump", "-C", "--count", "-S", "--stats"}, enum)
	})

	s.Run("has annotations", func() {
		s.assertReadOnlyToolAnnotations(tool, "CNI-Diagnostics: Kernel Connection Tracking")
	})

	s.Run("has handler", func() {
		s.NotNil(tool.Handler)
	})
}

func (s *ToolsSuite) TestConntrackHandler() {
	tool := s.toolByName("get-conntrack")

	s.Run("missing node returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "node parameter required")
	})

	s.Run("invalid node type returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{"node": 123}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "node parameter must be a string")
	})

	s.Run("invalid namespace type returns parameter error", func() {
		args := map[string]any{"node": "worker-0", "namespace": 123}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "namespace parameter must be a string")
	})

	s.Run("invalid command type returns parameter error", func() {
		args := map[string]any{"node": "worker-0", "command": 42}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "command parameter must be a string")
	})

	s.Run("invalid filter_parameters type returns parameter error", func() {
		args := map[string]any{"node": "worker-0", "filter_parameters": true}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "filter_parameters parameter must be a string")
	})

	s.Run("invalid head type returns parameter error", func() {
		args := map[string]any{"node": "worker-0", "head": "many"}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "head parameter must be an integer")
	})

	s.Run("invalid tail type returns parameter error", func() {
		args := map[string]any{"node": "worker-0", "tail": "many"}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "tail parameter must be an integer")
	})

	s.Run("invalid apply_tail_first type returns parameter error", func() {
		args := map[string]any{"node": "worker-0", "apply_tail_first": "yes"}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "apply_tail_first parameter must be a boolean")
	})

	s.Run("invalid timeout_seconds type returns parameter error", func() {
		args := map[string]any{"node": "worker-0", "timeout_seconds": "long"}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "timeout_seconds parameter must be an integer")
	})

	s.Run("negative head returns validation error", func() {
		args := map[string]any{"node": "worker-0", "head": float64(-1)}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Equal("head must be greater than or equal to 0", result.Error.Error())
	})

	s.Run("negative tail returns validation error", func() {
		args := map[string]any{"node": "worker-0", "tail": float64(-1)}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Equal("tail must be greater than or equal to 0", result.Error.Error())
	})

	s.Run("negative timeout_seconds returns validation error", func() {
		args := map[string]any{"node": "worker-0", "timeout_seconds": float64(-1)}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Equal("timeout_seconds must be greater than or equal to 0", result.Error.Error())
	})
}

func (s *ToolsSuite) TestInitIPtables() {
	tool := s.toolByName("get-iptables")

	s.Run("has correct name", func() {
		s.Equal("get-iptables", tool.Tool.Name)
	})

	s.Run("has description", func() {
		s.NotEmpty(tool.Tool.Description)
		s.Contains(tool.Tool.Description, "packet filter")
	})

	s.Run("has input schema", func() {
		s.Require().NotNil(tool.Tool.InputSchema)
		s.Equal("object", tool.Tool.InputSchema.Type)
		s.Contains(tool.Tool.InputSchema.Required, "node")
	})

	s.Run("has expected parameters", func() {
		props := tool.Tool.InputSchema.Properties
		s.Equal("string", props["node"].Type)
		s.Equal("string", props["namespace"].Type)
		s.Equal("string", props["table"].Type)
		s.Equal("string", props["command"].Type)
		s.Equal("string", props["filter_parameters"].Type)
		s.assertCommonOutputParams(&tool.Tool)
	})

	s.Run("table parameter has iptables tables", func() {
		enum := tool.Tool.InputSchema.Properties["table"].Enum
		s.Len(enum, 5)
		s.ElementsMatch([]any{"filter", "nat", "mangle", "raw", "security"}, enum)
	})

	s.Run("has annotations", func() {
		s.assertReadOnlyToolAnnotations(tool, "CNI-Diagnostics: Kernel IPtables")
	})

	s.Run("has handler", func() {
		s.NotNil(tool.Handler)
	})
}

func (s *ToolsSuite) TestIPtablesHandler() {
	tool := s.toolByName("get-iptables")

	s.Run("missing node returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "node parameter required")
	})

	s.Run("invalid node type returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{"node": true}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "node parameter must be a string")
	})

	s.Run("invalid namespace type returns parameter error", func() {
		args := map[string]any{"node": "worker-0", "namespace": 123}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "namespace parameter must be a string")
	})

	s.Run("invalid table type returns parameter error", func() {
		args := map[string]any{"node": "worker-0", "table": 8080}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "table parameter must be a string")
	})

	s.Run("invalid command type returns parameter error", func() {
		args := map[string]any{"node": "worker-0", "command": 42}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "command parameter must be a string")
	})

	s.Run("invalid filter_parameters type returns parameter error", func() {
		args := map[string]any{"node": "worker-0", "filter_parameters": []string{"-n"}}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "filter_parameters parameter must be a string")
	})

	s.Run("invalid head type returns parameter error", func() {
		args := map[string]any{"node": "worker-0", "head": "many"}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "head parameter must be an integer")
	})

	s.Run("negative head returns validation error", func() {
		args := map[string]any{"node": "worker-0", "head": float64(-1)}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Equal("head must be greater than or equal to 0", result.Error.Error())
	})

	s.Run("negative tail returns validation error", func() {
		args := map[string]any{"node": "worker-0", "tail": float64(-1)}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Equal("tail must be greater than or equal to 0", result.Error.Error())
	})

	s.Run("negative timeout_seconds returns validation error", func() {
		args := map[string]any{"node": "worker-0", "timeout_seconds": float64(-1)}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Equal("timeout_seconds must be greater than or equal to 0", result.Error.Error())
	})
}

func (s *ToolsSuite) TestInitNFT() {
	tool := s.toolByName("get-nft")

	s.Run("has correct name", func() {
		s.Equal("get-nft", tool.Tool.Name)
	})

	s.Run("has description", func() {
		s.NotEmpty(tool.Tool.Description)
		s.Contains(strings.ToLower(tool.Tool.Description), "nftables")
	})

	s.Run("has input schema", func() {
		s.Require().NotNil(tool.Tool.InputSchema)
		s.Equal("object", tool.Tool.InputSchema.Type)
		s.Contains(tool.Tool.InputSchema.Required, "node")
		s.Contains(tool.Tool.InputSchema.Required, "command")
	})

	s.Run("has expected parameters", func() {
		props := tool.Tool.InputSchema.Properties
		s.Equal("string", props["node"].Type)
		s.Equal("string", props["namespace"].Type)
		s.Equal("string", props["command"].Type)
		s.Equal("string", props["address_families"].Type)
		s.assertCommonOutputParams(&tool.Tool)
	})

	s.Run("command parameter has nft operations", func() {
		enum := tool.Tool.InputSchema.Properties["command"].Enum
		s.Len(enum, 6)
		s.ElementsMatch([]any{"list ruleset", "list tables", "list chains", "list sets", "list maps", "list flowtables"}, enum)
	})

	s.Run("address_families parameter has supported families", func() {
		enum := tool.Tool.InputSchema.Properties["address_families"].Enum
		s.Len(enum, 6)
		s.ElementsMatch([]any{"ip", "ip6", "inet", "arp", "bridge", "netdev"}, enum)
	})

	s.Run("has annotations", func() {
		s.assertReadOnlyToolAnnotations(tool, "CNI-Diagnostics: Kernel NFtables")
	})

	s.Run("has handler", func() {
		s.NotNil(tool.Handler)
	})
}

func (s *ToolsSuite) TestNFTHandler() {
	tool := s.toolByName("get-nft")

	s.Run("missing node returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{"command": "list ruleset"}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "node parameter required")
	})

	s.Run("missing command returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{"node": "worker-0"}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "command parameter required")
	})

	s.Run("invalid command type returns parameter error", func() {
		args := map[string]any{"node": "worker-0", "command": []string{"list ruleset"}}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "command parameter must be a string")
	})

	s.Run("invalid namespace type returns parameter error", func() {
		args := map[string]any{"node": "worker-0", "command": "list ruleset", "namespace": 123}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "namespace parameter must be a string")
	})

	s.Run("invalid address_families type returns parameter error", func() {
		args := map[string]any{"node": "worker-0", "command": "list ruleset", "address_families": 4}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "address_families parameter must be a string")
	})

	s.Run("negative head returns validation error", func() {
		args := map[string]any{"node": "worker-0", "command": "list ruleset", "head": float64(-1)}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Equal("head must be greater than or equal to 0", result.Error.Error())
	})

	s.Run("negative tail returns validation error", func() {
		args := map[string]any{"node": "worker-0", "command": "list ruleset", "tail": float64(-1)}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Equal("tail must be greater than or equal to 0", result.Error.Error())
	})

	s.Run("negative timeout_seconds returns validation error", func() {
		args := map[string]any{"node": "worker-0", "command": "list ruleset", "timeout_seconds": float64(-1)}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Equal("timeout_seconds must be greater than or equal to 0", result.Error.Error())
	})
}

func (s *ToolsSuite) TestInitIP() {
	tool := s.toolByName("get-ip")

	s.Run("has correct name", func() {
		s.Equal("get-ip", tool.Tool.Name)
	})

	s.Run("has description", func() {
		s.NotEmpty(tool.Tool.Description)
		s.Contains(tool.Tool.Description, "ip commands")
	})

	s.Run("has input schema", func() {
		s.Require().NotNil(tool.Tool.InputSchema)
		s.Equal("object", tool.Tool.InputSchema.Type)
		s.Contains(tool.Tool.InputSchema.Required, "node")
		s.Contains(tool.Tool.InputSchema.Required, "command")
	})

	s.Run("has expected parameters", func() {
		props := tool.Tool.InputSchema.Properties
		s.Equal("string", props["node"].Type)
		s.Equal("string", props["namespace"].Type)
		s.Equal("string", props["options"].Type)
		s.Equal("string", props["command"].Type)
		s.Equal("string", props["filter_parameters"].Type)
		s.assertCommonOutputParams(&tool.Tool)
	})

	s.Run("command parameter has ip subcommands", func() {
		enum := tool.Tool.InputSchema.Properties["command"].Enum
		s.Len(enum, 9)
		s.ElementsMatch([]any{
			"address show", "link show", "neighbour show", "netns show",
			"route show", "rule show", "vrf show", "xfrm state list", "xfrm policy list",
		}, enum)
	})

	s.Run("has annotations", func() {
		s.assertReadOnlyToolAnnotations(tool, "CNI-Diagnostics: Kernel IP Command")
	})

	s.Run("has handler", func() {
		s.NotNil(tool.Handler)
	})
}

func (s *ToolsSuite) TestIPHandler() {
	tool := s.toolByName("get-ip")

	s.Run("missing node returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{"command": "route show"}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "node parameter required")
	})

	s.Run("missing command returns parameter error", func() {
		result, err := tool.Handler(s.handlerParams(map[string]any{"node": "worker-0"}))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "command parameter required")
	})

	s.Run("invalid command type returns parameter error", func() {
		args := map[string]any{"node": "worker-0", "command": 42}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "command parameter must be a string")
	})

	s.Run("invalid namespace type returns parameter error", func() {
		args := map[string]any{"node": "worker-0", "command": "route show", "namespace": 123}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "namespace parameter must be a string")
	})

	s.Run("invalid options type returns parameter error", func() {
		args := map[string]any{"node": "worker-0", "command": "route show", "options": 4}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "options parameter must be a string")
	})

	s.Run("invalid filter_parameters type returns parameter error", func() {
		args := map[string]any{"node": "worker-0", "command": "route show", "filter_parameters": true}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Contains(result.Error.Error(), "invalid parameters")
		s.Contains(result.Error.Error(), "filter_parameters parameter must be a string")
	})

	s.Run("negative head returns validation error", func() {
		args := map[string]any{"node": "worker-0", "command": "route show", "head": float64(-1)}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Equal("head must be greater than or equal to 0", result.Error.Error())
	})

	s.Run("negative tail returns validation error", func() {
		args := map[string]any{"node": "worker-0", "command": "route show", "tail": float64(-1)}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Equal("tail must be greater than or equal to 0", result.Error.Error())
	})

	s.Run("negative timeout_seconds returns validation error", func() {
		args := map[string]any{"node": "worker-0", "command": "route show", "timeout_seconds": float64(-1)}
		result, err := tool.Handler(s.handlerParams(args))
		s.Require().NoError(err)
		s.Require().Error(result.Error)
		s.Equal("timeout_seconds must be greater than or equal to 0", result.Error.Error())
	})
}
