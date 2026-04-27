package mcp

import (
	"context"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/suite"
	"k8s.io/utils/ptr"
)

type ResourceRegistrarSuite struct {
	BaseMcpSuite
	originalToolsets []api.Toolset
	resourceReg      api.ResourceRegistrar
	templateReg      api.ResourceTemplateRegistrar
}

func (s *ResourceRegistrarSuite) SetupTest() {
	s.BaseMcpSuite.SetupTest()
	s.originalToolsets = toolsets.Toolsets()
}

func (s *ResourceRegistrarSuite) TearDownTest() {
	s.BaseMcpSuite.TearDownTest()
	toolsets.Clear()
	for _, toolset := range s.originalToolsets {
		toolsets.Register(toolset)
	}
}

// initWithRegistrars sets up a tool that captures the registrars, then initializes the client.
func (s *ResourceRegistrarSuite) initWithRegistrars() {
	testToolset := &mockResourceToolset{
		tools: []api.ServerTool{
			{
				Tool: api.Tool{
					Name:        "test_init",
					Description: "Initialization call to capture fields for testing",
					Annotations: api.ToolAnnotations{ReadOnlyHint: ptr.To(true)},
					InputSchema: &jsonschema.Schema{
						Type:       "object",
						Properties: make(map[string]*jsonschema.Schema),
					},
				},
				Handler: func(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
					s.resourceReg = params.ResourceRegistrar
					s.templateReg = params.ResourceTemplateRegistrar
					return api.NewToolCallResult("ok", nil), nil
				},
			},
		},
	}

	toolsets.Clear()
	toolsets.Register(testToolset)
	s.Cfg.Toolsets = []string{"resource-test"}
	s.InitMcpClient()

	// Call once to capture the registrars
	_, err := s.CallTool("test_init", map[string]any{})
	s.Require().NoError(err)
}

func (s *ResourceRegistrarSuite) TestAddResource() {
	s.initWithRegistrars()

	txt1 := "Content 1"
	json2 := `{"key": "value"}`

	s.resourceReg.AddResource("test://example/resource1", "Resource One", "First", "text/plain", txt1)
	s.resourceReg.AddResource("test://example/resource2", "Resource Two", "Second", "application/json", json2)

	s.Run("all resources appear in list", func() {
		result, err := s.ListResources()
		s.NoError(err)
		s.Require().Len(result.Resources, 2)

		uris := make(map[string]bool)
		for _, r := range result.Resources {
			uris[r.URI] = true
		}
		s.True(uris["test://example/resource1"])
		s.True(uris["test://example/resource2"])
	})

	s.Run("each resource has correct content and mimeType", func() {
		result1, err := s.ReadResource("test://example/resource1")
		s.NoError(err)
		s.Equal(txt1, result1.Contents[0].Text)
		s.Equal("text/plain", result1.Contents[0].MIMEType)

		result2, err := s.ReadResource("test://example/resource2")
		s.NoError(err)
		s.Equal(json2, result2.Contents[0].Text)
		s.Equal("application/json", result2.Contents[0].MIMEType)
	})
}

func (s *ResourceRegistrarSuite) TestRemoveResources() {
	s.initWithRegistrars()

	resToRemove := "test://example/to-remove"
	resToKeep := "test://example/to-keep"

	s.resourceReg.AddResource(resToRemove, "To Remove", "", "text/plain", "Temporary")
	s.resourceReg.AddResource(resToKeep, "To Keep", "", "text/plain", "Permanent")

	s.Run("both resources exist initially", func() {
		result, err := s.ListResources()
		s.NoError(err)
		s.Len(result.Resources, 2)
	})

	s.resourceReg.RemoveResources(resToRemove)

	s.Run("removed resource no longer in list", func() {
		result, err := s.ListResources()
		s.NoError(err)
		s.Require().Len(result.Resources, 1)
		s.Equal(resToKeep, result.Resources[0].URI)
	})
}

func (s *ResourceRegistrarSuite) TestAddResourceTemplate() {
	s.initWithRegistrars()

	uriTempl := "test://example/{name}"
	txtFoo := "foo-dynamic-resource"

	s.templateReg.AddResourceTemplate(
		uriTempl,
		txtFoo,
		txtFoo,
		"text/plain",
		func(_ context.Context, uri string) (string, error) {
			return "content for: " + uri, nil
		},
	)

	s.Run("template appears in list", func() {
		result, err := s.ListResourceTemplates()
		s.NoError(err)
		s.Require().Len(result.ResourceTemplates, 1)
		s.Equal(uriTempl, result.ResourceTemplates[0].URITemplate)
		s.Equal(txtFoo, result.ResourceTemplates[0].Name)
		s.Equal(txtFoo, result.ResourceTemplates[0].Description)
		s.Equal("text/plain", result.ResourceTemplates[0].MIMEType)
	})

	s.Run("handler receives correct URI for different URIs", func() {
		uri1 := "test://example/foo"
		result1, err := s.ReadResource(uri1)
		s.NoError(err)
		s.Equal(uri1, result1.Contents[0].URI)
		s.Equal("content for: "+uri1, result1.Contents[0].Text)

		uri2 := "test://example/bar"
		result2, err := s.ReadResource(uri2)
		s.NoError(err)
		s.Equal(uri2, result2.Contents[0].URI)
		s.Equal("content for: "+uri2, result2.Contents[0].Text)
	})
}

func (s *ResourceRegistrarSuite) TestRemoveResourceTemplates() {
	s.initWithRegistrars()

	uriToRemove := "test://remove/{id}"
	uriToKeep := "test://keep/{id}"

	handler := func(_ context.Context, uri string) (string, error) { return uri, nil }
	s.templateReg.AddResourceTemplate(uriToRemove, "To Remove", "", "text/plain", handler)
	s.templateReg.AddResourceTemplate(uriToKeep, "To Keep", "", "text/plain", handler)

	s.Run("both templates exist initially", func() {
		result, err := s.ListResourceTemplates()
		s.NoError(err)
		s.Len(result.ResourceTemplates, 2)
	})

	s.templateReg.RemoveResourceTemplates(uriToRemove)

	s.Run("removed template no longer in list", func() {
		result, err := s.ListResourceTemplates()
		s.NoError(err)
		s.Require().Len(result.ResourceTemplates, 1)
		s.Equal(uriToKeep, result.ResourceTemplates[0].URITemplate)
	})
}

type mockResourceToolset struct {
	tools []api.ServerTool
}

func (m *mockResourceToolset) GetName() string                           { return "resource-test" }
func (m *mockResourceToolset) GetDescription() string                    { return "Test toolset for resource registrar" }
func (m *mockResourceToolset) GetTools(_ api.Openshift) []api.ServerTool { return m.tools }
func (m *mockResourceToolset) GetPrompts() []api.ServerPrompt            { return nil }

func TestResourceRegistrarSuite(t *testing.T) {
	suite.Run(t, new(ResourceRegistrarSuite))
}
