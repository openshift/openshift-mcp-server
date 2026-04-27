package mcp

import (
	"context"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type resourceRegistrar struct {
	server *mcp.Server
}

var _ api.ResourceRegistrar = &resourceRegistrar{}

func (r *resourceRegistrar) AddResource(uri, name, description, mimeType, content string) {
	r.server.AddResource(
		&mcp.Resource{URI: uri, Name: name, Description: description, MIMEType: mimeType},
		func(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{{URI: uri, Text: content}},
			}, nil
		},
	)
}

func (r *resourceRegistrar) RemoveResources(uris ...string) {
	r.server.RemoveResources(uris...)
}

type resourceTemplateRegistrar struct {
	server *mcp.Server
}

var _ api.ResourceTemplateRegistrar = &resourceTemplateRegistrar{}

func (r *resourceTemplateRegistrar) AddResourceTemplate(uriTemplate, name, description, mimeType string, handler api.ResourceTemplateHandler) {
	r.server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: uriTemplate,
			Name:        name,
			Description: description,
			MIMEType:    mimeType,
		},
		func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			content, err := handler(ctx, req.Params.URI)
			if err != nil {
				return nil, err
			}
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{{URI: req.Params.URI, MIMEType: mimeType, Text: content}},
			}, nil
		},
	)
}

func (r *resourceTemplateRegistrar) RemoveResourceTemplates(uriTemplates ...string) {
	r.server.RemoveResourceTemplates(uriTemplates...)
}
