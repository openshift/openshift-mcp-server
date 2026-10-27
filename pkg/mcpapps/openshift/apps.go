// Package openshift provides the downstream OpenShift presentation for MCP Apps.
// It owns its bundled HTML, React components, styles, and runtime; upstream's
// generic applications and MCP registration remain unchanged.
package openshift

import (
	_ "embed"
	"html"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	upstream "github.com/containers/kubernetes-mcp-server/pkg/mcpapps"
)

//go:embed app.html
var template string

// runtimeScript is a downstream-owned adaptation of the upstream standard
// runtime. Changes to upstream lifecycle handling should be reviewed here too.
//
//go:embed runtime.js
var runtimeScript string

//go:embed presentation.js
var presentationScript string

//go:embed presentation.css
var presentationStyles string

// Table creates a PatternFly table using the same structured data as the
// upstream table app.
func Table(uri, name string, options ...upstream.CustomOption) *api.ToolApp {
	return customApp(uri, name, "table", options...)
}

// Metrics creates a PatternFly CPU and memory metrics table.
func Metrics(uri, name string, options ...upstream.CustomOption) *api.ToolApp {
	return customApp(uri, name, "metrics", options...)
}

// Resource creates a PatternFly details view for one Kubernetes resource.
func Resource(uri, name string, options ...upstream.CustomOption) *api.ToolApp {
	return customApp(uri, name, "resource", options...)
}

func customApp(uri, name, kind string, options ...upstream.CustomOption) *api.ToolApp {
	// Tool discovery also runs with Apps disabled and on reload. Keep the
	// embedded assets shared and assemble each document only when it is read.
	return upstream.Custom(uri, name, func(_ api.ResourceHandlerParams) (string, error) {
		return strings.NewReplacer(
			"{{KIND}}", kind,
			"{{NAME}}", html.EscapeString(name),
			"{{SCRIPT}}", runtimeScript,
			"{{PRESENTATION_STYLES}}", presentationStyles,
			"{{PRESENTATION_SCRIPT}}", presentationScript,
		).Replace(template), nil
	}, options...)
}
