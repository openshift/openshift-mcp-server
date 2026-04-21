package resultutil

import (
	"encoding/json"
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Result represents a common tool execution result that can be converted
// to either MCP or Toolset result types.
type Result struct {
	// Data holds the structured result data (only set for successful results)
	Data any
	// JSONText holds the JSON string representation of Data
	JSONText string
	// Error holds any error that occurred (nil for successful results)
	Error error
}

// NewSuccessResult creates a successful result with structured data.
// The data will be automatically marshaled to JSON.
// If marshaling fails, an error result is returned instead.
func NewSuccessResult(data any) *Result {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return &Result{
			Error: fmt.Errorf("failed to marshal result: %w", err),
		}
	}

	return &Result{
		Data:     data,
		JSONText: string(jsonBytes),
	}
}

// NewErrorResult creates an error result with the given error.
func NewErrorResult(err error) *Result {
	return &Result{
		Error: err,
	}
}

// ToMCPResult converts the Result to an MCP CallToolResult.
// Returns (result, nil) following the MCP pattern where errors
// are encoded in the result, not the error return value.
func (r *Result) ToMCPResult() (*mcp.CallToolResult, error) {
	callToolRes := &mcp.CallToolResult{}
	if r.Error != nil {
		callToolRes.SetError(r.Error)
		//nolint:nilerr // MCP pattern encodes errors in result, not error return
		return callToolRes, nil
	}
	callToolRes.Content = []mcp.Content{
		&mcp.ToolResultContent{
			StructuredContent: r.Data, Content: []mcp.Content{&mcp.TextContent{Text: r.JSONText}},
		},
	}
	return callToolRes, nil
}

// ToToolsetResult converts the Result to a Toolset ToolCallResult.
// Returns (result, nil) following the pattern where errors are encoded
// in the ToolCallResult, not the error return value.
func (r *Result) ToToolsetResult() (*api.ToolCallResult, error) {
	if r.Error != nil {
		//nolint:nilerr // Toolset pattern encodes errors in result, not error return
		return api.NewToolCallResult("", r.Error), nil
	}
	return api.NewToolCallResult(r.JSONText, nil), nil
}

// IsError returns true if the result represents an error.
func (r *Result) IsError() bool {
	return r.Error != nil
}

// Unwrap returns the typed data and error.
// If the result contains an error, it returns zero value of T and the error.
// If successful, it returns the data cast to type T and nil error.
func Unwrap[T any](r *Result) (T, error) {
	var zero T
	if r.Error != nil {
		return zero, r.Error
	}
	if data, ok := r.Data.(T); ok {
		return data, nil
	}
	return zero, fmt.Errorf("failed to cast result data to type %T", zero)
}
