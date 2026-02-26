package mustgather

import (
	"fmt"
	"strings"
	"sync"

	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
)

var (
	providerMu sync.RWMutex
	provider   *mg.Provider
)

// setProvider stores the loaded must-gather provider
func setProvider(p *mg.Provider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	provider = p
}

// getProvider returns the loaded must-gather provider or an error
func getProvider() (*mg.Provider, error) {
	providerMu.RLock()
	defer providerMu.RUnlock()
	if provider == nil {
		return nil, fmt.Errorf("no must-gather archive loaded. Call mustgather_use first with a path to a must-gather archive")
	}
	return provider, nil
}

// getString extracts a string argument with a default
func getString(args map[string]any, key, defaultValue string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return defaultValue
}

// getInt extracts an integer argument with a default
func getInt(args map[string]any, key string, defaultValue int) int {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case int64:
			return int(n)
		}
	}
	return defaultValue
}

// getBool extracts a boolean argument with a default
func getBool(args map[string]any, key string, defaultValue bool) bool {
	if v, ok := args[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// formatBytes formats bytes as human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatNumber formats a number with thousands separators
func formatNumber(n int64) string {
	str := fmt.Sprintf("%d", n)
	if len(str) <= 3 {
		return str
	}
	result := ""
	for i, c := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result += ","
		}
		result += string(c)
	}
	return result
}

// formatDuration formats duration in seconds to human-readable string
func formatDuration(seconds float64) string {
	if seconds < 0.001 {
		return fmt.Sprintf("%.2fus", seconds*1000000)
	} else if seconds < 1 {
		return fmt.Sprintf("%.2fms", seconds*1000)
	} else if seconds < 60 {
		return fmt.Sprintf("%.2fs", seconds)
	} else if seconds < 3600 {
		return fmt.Sprintf("%.1fm", seconds/60)
	} else if seconds < 86400 {
		return fmt.Sprintf("%.1fh", seconds/3600)
	}
	return fmt.Sprintf("%.1fd", seconds/86400)
}

// truncate truncates a string to maxLen
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// healthSymbol returns a symbol for health status
func healthSymbol(health string) string {
	switch strings.ToLower(health) {
	case "up", "healthy", "ok", "true", "firing":
		return "[OK]"
	case "down", "unhealthy", "error", "false":
		return "[FAIL]"
	default:
		return "[WARN]"
	}
}

// severitySymbol returns a symbol for severity level
func severitySymbol(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return "[CRITICAL]"
	case "warning":
		return "[WARNING]"
	case "info":
		return "[INFO]"
	default:
		return "[UNKNOWN]"
	}
}
