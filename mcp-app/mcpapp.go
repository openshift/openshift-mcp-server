package mcpapp

import _ "embed"

// ResourceMIMEType is the MIME type for MCP App HTML resources per the MCP Apps spec.
const ResourceMIMEType = "text/html;profile=mcp-app"

// PodsTopResourceURI is the MCP resource URI for the pods-top UI.
const PodsTopResourceURI = "ui://kubernetes-mcp-server/pods-top.html"

//go:embed dist/pods-top-app.html
var podsTopHTML string

// AppResource represents an embedded MCP App UI resource.
type AppResource struct {
	URI  string
	Name string
	HTML string
}

// ToolMeta returns a _meta map with the MCP Apps ui.resourceUri field set.
func ToolMeta(resourceURI string) map[string]any {
	return map[string]any{
		"ui": map[string]any{
			"resourceUri": resourceURI,
		},
	}
}

// Resources returns all registered MCP App resources.
func Resources() []AppResource {
	return []AppResource{
		{
			URI:  PodsTopResourceURI,
			Name: "Pods Top UI",
			HTML: podsTopHTML,
		},
	}
}
