package resultutil

import (
	"encoding/json"
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/mark3labs/mcp-go/mcp"
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
	if r.Error != nil {
		//nolint:nilerr // MCP pattern encodes errors in result, not error return
		return mcp.NewToolResultError(r.Error.Error()), nil
	}
	return mcp.NewToolResultStructured(r.Data, r.JSONText), nil
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
