package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/mcplog"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ErrElicitationNotSupported is returned when the MCP client does not support elicitation.
// Tool authors can check for this error using errors.Is() to implement fallback behavior.
var ErrElicitationNotSupported = errors.New("client does not support elicitation")

type sessionElicitor struct{}

var _ api.Elicitor = &sessionElicitor{}

func (s *sessionElicitor) Elicit(ctx context.Context, message string, requestedSchema *jsonschema.Schema) (*api.ElicitResult, error) {
	session, ok := ctx.Value(mcplog.MCPSessionContextKey).(*mcp.ServerSession)
	if !ok || session == nil {
		return nil, fmt.Errorf("no MCP session found in context")
	}

	result, err := session.Elicit(ctx, &mcp.ElicitParams{Message: message, RequestedSchema: requestedSchema})
	if err != nil {
		if strings.Contains(err.Error(), "does not support elicitation") {
			return nil, fmt.Errorf("%w: %s", ErrElicitationNotSupported, err.Error())
		}
		return nil, err
	}

	return &api.ElicitResult{Action: result.Action, Content: result.Content}, nil
}
