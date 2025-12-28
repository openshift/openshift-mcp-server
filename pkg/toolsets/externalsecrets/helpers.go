package externalsecrets

import (
	"time"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
)

// getStringArg retrieves a string argument from the tool parameters with a default value
func getStringArg(params api.ToolHandlerParams, key, defaultVal string) string {
	if val, ok := params.GetArguments()[key].(string); ok && val != "" {
		return val
	}
	return defaultVal
}

// getBoolArg retrieves a boolean argument from the tool parameters with a default value
func getBoolArg(params api.ToolHandlerParams, key string, defaultVal bool) bool {
	if val, ok := params.GetArguments()[key].(bool); ok {
		return val
	}
	return defaultVal
}

// getIntArg retrieves an integer argument from the tool parameters with a default value
func getIntArg(params api.ToolHandlerParams, key string, defaultVal int) int {
	if val, ok := params.GetArguments()[key].(float64); ok {
		return int(val)
	}
	if val, ok := params.GetArguments()[key].(int); ok {
		return val
	}
	return defaultVal
}

// getCurrentTimestamp returns the current Unix timestamp
func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}
