package openshift_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	upstream "github.com/containers/kubernetes-mcp-server/pkg/mcpapps"
	openshiftapps "github.com/containers/kubernetes-mcp-server/pkg/mcpapps/openshift"
)

type OpenShiftAppsSuite struct{ suite.Suite }

func (s *OpenShiftAppsSuite) TestAssetsAreSafeToInline() {
	for _, tc := range []struct {
		file, closingTag string
	}{
		{"presentation.js", "</script"},
		{"runtime.js", "</script"},
		{"presentation.css", "</style"},
	} {
		s.Run(tc.file, func() {
			content, err := os.ReadFile(tc.file)
			s.Require().NoError(err)
			// HTML recognizes closing tags even inside JS strings or CSS URLs.
			// Only the enclosing element's closing tag matters.
			s.NotContains(strings.ToLower(string(content)), tc.closingTag)
		})
	}
}

func (s *OpenShiftAppsSuite) TestCustomPresentations() {
	for _, tc := range []struct {
		kind string
		new  func(string, string, ...upstream.CustomOption) *api.ToolApp
	}{
		{"table", openshiftapps.Table},
		{"metrics", openshiftapps.Metrics},
		{"resource", openshiftapps.Resource},
	} {
		s.Run(tc.kind, func() {
			app := tc.new("ui://example/view", `View "quoted" <name>`,
				upstream.WithDescription("Custom view"),
				upstream.WithMetadata(map[string]any{"ui": map[string]any{"prefersBorder": true}}))
			s.Require().NoError(app.Validate())
			s.Equal("Custom view", app.Description)
			s.Equal(map[string]any{"ui": map[string]any{"prefersBorder": true}}, app.Meta)
			content, err := app.Handler(api.ResourceHandlerParams{Context: s.T().Context()})
			s.Require().NoError(err)
			s.Contains(content.Text, `data-app-kind="`+tc.kind+`"`)
			s.Contains(content.Text, `data-app-name="View &#34;quoted&#34; &lt;name&gt;"`)
			s.Contains(content.Text, "pf-v6-c-card")
			s.NotRegexp(`\{\{(?:KIND|NAME|SCRIPT|PRESENTATION_STYLES|PRESENTATION_SCRIPT)\}\}`, content.Text)
			s.NotRegexp(`(?i)(?:src|href)\s*=\s*["'](?:https?:)?//`, content.Text)
			s.NotRegexp(`(?i)url\(\s*["']?(?:https?:)?//`, content.Text)
		})
	}
}

func (s *OpenShiftAppsSuite) TestUpstreamPresentationIsIndependent() {
	app := upstream.NamespacesList()
	content, err := app.Handler(api.ResourceHandlerParams{Context: s.T().Context()})
	s.Require().NoError(err)
	s.NotContains(content.Text, "pf-v6-c-card")
	s.NotContains(content.Text, "mcpAppPresentation")
}

func TestOpenShiftApps(t *testing.T) {
	suite.Run(t, new(OpenShiftAppsSuite))
}

var benchmarkApp *api.ToolApp

// BenchmarkAppRegistration tracks allocations during tool discovery, when no
// client has requested the HTML yet (including when Apps are disabled).
func BenchmarkAppRegistration(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchmarkApp = openshiftapps.Table("ui://example/view", "Resource table")
	}
}
