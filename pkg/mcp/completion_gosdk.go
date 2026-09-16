package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/klogutil"
)

// maxCompletionValues bounds a single completion/complete response, per the MCP
// specification (a page returns at most 100 values).
const maxCompletionValues = 100

// completionRegistry holds the argument-completion handlers currently installed,
// indexed by the reference a client uses in a completion/complete request. It is
// a struct (rather than a bare map) so additional reference kinds — e.g. prompt
// completions keyed by name — can be added without disturbing the atomic-pointer
// plumbing that publishes it across hot reloads.
type completionRegistry struct {
	// resourceTemplates is keyed by URITemplate (the CompleteReference.URI a
	// client sends for a ref/resource completion).
	resourceTemplates map[string]api.ArgumentCompletionHandler
}

// buildCompletionRegistry collects the completion handlers from the applicable
// resource templates. Templates without a CompletionHandler are skipped. The
// returned registry is always non-nil so readers never have to nil-check the
// published pointer's target.
func buildCompletionRegistry(rts []api.ServerResourceTemplate) *completionRegistry {
	reg := &completionRegistry{resourceTemplates: map[string]api.ArgumentCompletionHandler{}}
	for _, rt := range rts {
		if rt.CompletionHandler != nil {
			reg.resourceTemplates[rt.ResourceTemplate.URITemplate] = rt.CompletionHandler
		}
	}
	return reg
}

// handleComplete is the single server-wide completion/complete dispatcher wired
// into mcp.ServerOptions.CompletionHandler. It routes a request to the handler
// registered for its reference and shapes the result per the MCP spec.
// Completion is advisory: a missing handler or a handler error yields an empty
// result rather than a failed RPC.
func (s *Server) handleComplete(ctx context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	if req == nil || req.Params == nil || req.Params.Ref == nil {
		return emptyCompleteResult(), nil
	}

	reg := s.completions.Load()
	ref := req.Params.Ref

	var handler api.ArgumentCompletionHandler
	switch ref.Type {
	case "ref/resource":
		if reg != nil {
			handler = reg.resourceTemplates[ref.URI]
		}
	default:
		// ref/prompt and any other reference kinds are reserved: no handlers
		// are registered for them yet.
	}
	if handler == nil {
		return emptyCompleteResult(), nil
	}

	resolved := map[string]string{}
	if req.Params.Context != nil && req.Params.Context.Arguments != nil {
		resolved = req.Params.Context.Arguments
	}

	values, err := handler(ctx, req.Params.Argument.Name, req.Params.Argument.Value, resolved)
	if err != nil {
		klogutil.FromContext(ctx).V(2).Info("completion handler failed", "ref", ref.URI, "argument", req.Params.Argument.Name, "error", err)
		return emptyCompleteResult(), nil
	}

	total := len(values)
	hasMore := false
	if total > maxCompletionValues {
		values = values[:maxCompletionValues]
		hasMore = true
	}
	if values == nil {
		values = []string{}
	}
	return &mcp.CompleteResult{
		Completion: mcp.CompletionResultDetails{
			Values:  values,
			Total:   total,
			HasMore: hasMore,
		},
	}, nil
}

// emptyCompleteResult returns a well-formed empty completion response. Values is
// a non-nil empty slice so it serializes as [] rather than null.
func emptyCompleteResult() *mcp.CompleteResult {
	return &mcp.CompleteResult{Completion: mcp.CompletionResultDetails{Values: []string{}}}
}
