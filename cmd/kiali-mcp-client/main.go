package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/kiali/kiali-mcp-server/cmd/kiali-mcp-client/output"
)

func main() {
	var (
		serverURL string
		token     string
		authHdr   string
		toolName  string
		jsonOut   bool
		listKiali bool
		namespace string
		timeout   time.Duration
	)

	flag.StringVar(&serverURL, "server", getenvDefault("MCP_SERVER", "http://localhost:8080/mcp"), "MCP server URL (e.g. http://host:port/mcp)")
	flag.StringVar(&token, "token", os.Getenv("MCP_TOKEN"), "Bearer token (without 'Bearer ' prefix). If empty, tries AUTHORIZATION env var")
	flag.StringVar(&authHdr, "authorization", os.Getenv("AUTHORIZATION"), "Authorization header value (overrides --token). Example: 'Bearer eyJ...' ")
	flag.StringVar(&toolName, "tool", "", "Tool to call (e.g. validations_list). If empty, defaults to validations_list")
	flag.BoolVar(&jsonOut, "json", false, "If true, print JSON output instead of pretty formatting")
	flag.BoolVar(&listKiali, "list-kiali-tools", false, "List all available Kiali-related tools and exit")
	flag.StringVar(&namespace, "namespace", "", "Optional namespace to pass to validations_list tool")
	flag.DurationVar(&timeout, "timeout", 30*time.Second, "Overall request timeout")
	flag.Parse()

	if authHdr == "" && token != "" {
		authHdr = "Bearer " + strings.TrimSpace(token)
	}

	// Configure transport with optional Authorization header
	opts := []transport.StreamableHTTPCOption{
		transport.WithHTTPTimeout(timeout),
	}
	if authHdr != "" {
		opts = append(opts, transport.WithHTTPHeaders(map[string]string{"Authorization": authHdr}))
	}

	client, err := mcpclient.NewStreamableHttpClient(serverURL, opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create client: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := client.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Initialize session
	_, err = client.Initialize(ctx, mcp.InitializeRequest{
		Request: mcp.Request{Method: string(mcp.MethodInitialize)},
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "kiali-mcp-client",
				Version: "dev",
			},
			Capabilities: mcp.ClientCapabilities{},
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize client: %v\n", err)
		os.Exit(1)
	}

	if listKiali {
		// Retrieve tools and list only Kiali-related tools
		toolsRes, err := client.ListTools(ctx, mcp.ListToolsRequest{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to list tools: %v\n", err)
			os.Exit(1)
		}
		// Filter known Kiali tools
		kialiSet := map[string]struct{}{
			"validations_list": {},
		}
		infos := make([]output.ToolInfo, 0)
		for _, t := range toolsRes.Tools {
			if _, ok := kialiSet[t.Name]; ok {
				infos = append(infos, output.ToolInfo{Name: t.Name, Description: t.Description})
			}
		}
		output.PrintToolList(infos, jsonOut)
		return
	}

	// Determine tool
	if strings.TrimSpace(toolName) == "" {
		toolName = "validations_list"
	}

	// Build arguments based on known tools
	args := map[string]any{}
	switch toolName {
	case "validations_list":
		if strings.TrimSpace(namespace) != "" {
			args["namespace"] = namespace
		}
	default:
		// No specific args known; allow empty or future generic args
	}
	argsBytes, _ := json.Marshal(args)

	// Call tool
	result, err := client.CallTool(ctx, mcp.CallToolRequest{
		Request: mcp.Request{Method: "tools/call"},
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: json.RawMessage(argsBytes),
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "tool call failed: %v\n", err)
		os.Exit(2)
	}

	if result.IsError {
		fmt.Fprintf(os.Stderr, "error: %s\n", firstText(result))
		os.Exit(3)
	}

	output.Print(toolName, firstText(result), jsonOut)
}

func firstText(r *mcp.CallToolResult) string {
	if r == nil {
		return ""
	}
	for _, c := range r.Content {
		if t, ok := c.(mcp.TextContent); ok {
			return t.Text
		}
	}
	return ""
}

func getenvDefault(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
