// Package korrel8r adapts the upstream Korrel8r MCP tools to this server.
package korrel8r

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/config"
	"github.com/containers/kubernetes-mcp-server/pkg/klogutil"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/containers/kubernetes-mcp-server/pkg/obs/auth"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
	"github.com/google/jsonschema-go/jsonschema"
	upstream "github.com/korrel8r/korrel8r/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s.io/utils/ptr"
)

const ToolsetName = "observability/korrel8r"

// toolPrefix disambiguates the upstream tool names from other toolsets.
const toolPrefix = "korrel8r_"

// defaultTimeout is a backstop against a hung connection to the Korrel8r
// backend. It is deliberately longer than Korrel8r's own request deadline
// (requestTimeout: 1m in the shipped OpenShift configs, unset by default) so
// that the server's timeout, error, or truncated partial result wins over a
// client-side cut-off. Raise it with the timeout option if the backend is
// tuned for longer traversals.
const defaultTimeout = 2 * time.Minute

// Config selects the Korrel8r REST server, authentication source and request timeout.
type Config struct {
	URL      string        `toml:"url"`
	AuthMode auth.AuthMode `toml:"auth_mode,omitempty"`
	// Timeout bounds a single REST call, e.g. "5m". Defaults to defaultTimeout.
	Timeout time.Duration `toml:"timeout,omitempty"`
}

func (c *Config) getAuthMode() auth.AuthMode {
	if c.AuthMode == "" {
		return auth.AuthModeHeader
	}
	return c.AuthMode
}

func (c *Config) getTimeout() time.Duration {
	if c.Timeout <= 0 {
		return defaultTimeout
	}
	return c.Timeout
}

// parseURL returns the validated backend URL.
func (c *Config) parseURL() (*url.URL, error) {
	u, err := url.Parse(c.URL)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, fmt.Errorf("invalid korrel8r url %q: an http(s) URL is required", c.URL)
	}
	return u, nil
}

func (c *Config) Validate() error {
	if c.AuthMode != "" {
		if _, err := auth.ParseAuthMode(string(c.AuthMode)); err != nil {
			return err
		}
	}
	if c.Timeout < 0 {
		return fmt.Errorf("invalid korrel8r timeout %v: must not be negative", c.Timeout)
	}
	_, err := c.parseURL()
	return err
}

func init() {
	config.RegisterToolsetConfig(ToolsetName, func(_ context.Context, p toml.Primitive, md toml.MetaData) (config.ExtendedConfig, error) {
		var cfg Config
		if err := md.PrimitiveDecode(p, &cfg); err != nil {
			return nil, err
		}
		return &cfg, nil
	})
	toolsets.Register(&Toolset{})
}

type Toolset struct{}

var (
	_ api.Toolset           = (*Toolset)(nil)
	_ config.ExtendedConfig = (*Config)(nil)
)

func (*Toolset) GetName() string { return ToolsetName }
func (*Toolset) GetDescription() string {
	return "Correlate Kubernetes resources with logs, metrics, alerts, and traces using Korrel8r; inspect and update the caller's connected console."
}
func (*Toolset) GetPrompts(context.Context, api.ToolsetContext) []api.ServerPrompt     { return nil }
func (*Toolset) GetResources(context.Context, api.ToolsetContext) []api.ServerResource { return nil }
func (*Toolset) GetResourceTemplates(context.Context, api.ToolsetContext) []api.ServerResourceTemplate {
	return nil
}

// GetTools mechanically converts the upstream tool definitions.
// Upstream remains the sole owner of schemas, descriptions and handlers.
func (*Toolset) GetTools(_ context.Context, _ api.ToolsetContext) []api.ServerTool {
	tools := make([]api.ServerTool, 0, len(upstream.ToolNames))
	for _, name := range upstream.ToolNames {
		t := bind(nil, name).Tool
		tools = append(tools, api.ServerTool{
			Tool: api.Tool{
				Name:         toolPrefix + name,
				Description:  toolNameReplacer.Replace(t.Description),
				Annotations:  annotations(t.Annotations),
				InputSchema:  schema(t.InputSchema),
				OutputSchema: schema(t.OutputSchema),
			},
			RBAC:         api.RBACNone(),
			ClusterAware: ptr.To(false),
			Handler:      handler(name),
		})
	}
	return tools
}

// bind returns the upstream binding for name. Each call allocates a fresh tool
// definition and a handler bound to client, so nothing is shared between callers.
func bind(client *upstream.Client, name string) upstream.Binding {
	binding, ok := upstream.BindTool(client, name)
	if !ok {
		panic("unknown korrel8r tool " + name) // Names come from upstream.ToolNames.
	}
	return binding
}

// Only quoted tool references are rewritten; ordinary prose is left alone.
var toolNameReplacer = func() *strings.Replacer {
	replacements := make([]string, 0, len(upstream.ToolNames)*2)
	for _, name := range upstream.ToolNames {
		replacements = append(replacements, "'"+name+"'", "'"+toolPrefix+name+"'")
	}
	return strings.NewReplacer(replacements...)
}()

func annotations(a *mcp.ToolAnnotations) api.ToolAnnotations {
	if a == nil {
		return api.ToolAnnotations{}
	}
	return api.ToolAnnotations{
		Title:           a.Title,
		ReadOnlyHint:    ptr.To(a.ReadOnlyHint),
		DestructiveHint: a.DestructiveHint,
		IdempotentHint:  ptr.To(a.IdempotentHint),
		OpenWorldHint:   a.OpenWorldHint,
	}
}

// schema converts an upstream schema, typed as any by the MCP SDK, and renames
// tool references in its property descriptions.
func schema(v any) *jsonschema.Schema {
	s, _ := v.(*jsonschema.Schema)
	if s == nil {
		return nil
	}
	for _, property := range s.Properties {
		property.Description = toolNameReplacer.Replace(property.Description)
	}
	return s
}

// handler binds the named upstream tool to a per-call client that carries the
// caller's identity. Configuration problems are reported as tool errors, not
// transport errors, so the model can recover (e.g. by telling the user what to
// configure).
func handler(name string) api.ToolHandlerFunc {
	return func(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
		var client *upstream.Client
		// The upstream instructions tool is static. It must work even when
		// Korrel8r is not configured or the backend is unavailable.
		if name != upstream.InstructionsTool {
			var err error
			if client, err = newClient(params); err != nil {
				klogutil.FromContext(params.Context).Error(err, "Korrel8r client unavailable", "tool", name)
				return api.NewToolCallResult("", err), nil
			}
		}
		args, err := json.Marshal(params.GetArguments())
		if err != nil {
			return nil, fmt.Errorf("marshal korrel8r arguments: %w", err)
		}
		// Upstream reports tool failures in the result, not as an error.
		result, err := bind(client, name).Handler(params.Context,
			&mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: name, Arguments: args}})
		if err != nil {
			return nil, err
		}
		rewriteResult(result)
		return convertResult(result)
	}
}

// rewriteResult renames tool references in results that document other tools:
// instructions and help. Results of the other tools are cluster data, not
// guidance, and are passed through unchanged.
func rewriteResult(result *mcp.CallToolResult) {
	if result == nil || result.IsError {
		return
	}
	// Structured content is freshly allocated per call, safe to modify in place.
	// Upstream returns one of these by value or by pointer; accept both so a
	// change of shape cannot silently disable the rewrite.
	switch content := result.StructuredContent.(type) {
	case upstream.InstructionsResult:
		content.Instructions = toolNameReplacer.Replace(content.Instructions)
		result.StructuredContent = content
	case *upstream.InstructionsResult:
		content.Instructions = toolNameReplacer.Replace(content.Instructions)
	case upstream.HelpResult:
		content.Documentation = toolNameReplacer.Replace(content.Documentation)
		result.StructuredContent = content
	case *upstream.HelpResult:
		content.Documentation = toolNameReplacer.Replace(content.Documentation)
	default:
		return
	}
	// Keep the JSON text in step with the structured content, as upstream does.
	// Upstream text is the sole source of guidance: leave it if it cannot encode.
	if text, err := json.Marshal(result.StructuredContent); err == nil {
		result.Content = []mcp.Content{&mcp.TextContent{Text: string(text)}}
	}
}

// newClient creates a Korrel8r REST client that carries the caller's identity.
func newClient(params api.ToolHandlerParams) (*upstream.Client, error) {
	c, ok := params.GetToolsetConfig(ToolsetName)
	if !ok {
		return nil, fmt.Errorf("korrel8r endpoint not configured: set [toolset_configs.%q].url", ToolsetName)
	}
	cfg, ok := c.(*Config)
	if !ok {
		return nil, fmt.Errorf("invalid korrel8r configuration type %T", c)
	}
	u, err := cfg.parseURL()
	if err != nil {
		return nil, err
	}
	rt, err := auth.BuildRoundTripper(params.Context, params.Config, params.RESTConfig(), cfg.getAuthMode(), u.Scheme == "https", false)
	if err != nil {
		return nil, err
	}
	// Both the transport and upstream client bound each backend response.
	rt = kubernetes.NewBackendResponseRoundTripper(rt, kubernetes.LiveBackendLimit(params.Config))
	return upstream.NewClient(strings.TrimRight(u.String(), "/"), &http.Client{Transport: rt, Timeout: cfg.getTimeout()}), nil
}

// convertResult converts an upstream result. Korrel8r tools only return text.
func convertResult(result *mcp.CallToolResult) (*api.ToolCallResult, error) {
	if result == nil {
		return nil, fmt.Errorf("korrel8r returned no result")
	}
	out := &api.ToolCallResult{StructuredContent: result.StructuredContent}
	var messages []string
	for _, content := range result.Content {
		text, ok := content.(*mcp.TextContent)
		if !ok {
			return nil, fmt.Errorf("unsupported korrel8r content type %T", content)
		}
		out.ContentBlocks = append(out.ContentBlocks, api.NewTextToolContent(text.Text))
		messages = append(messages, text.Text)
	}
	if result.IsError {
		out.Error = fmt.Errorf("korrel8r: %s", strings.Join(messages, "\n"))
	}
	return out, nil
}
