package mustgather

import (
	"fmt"
	"strings"
	"sync"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"

	"golang.org/x/sync/singleflight"
)

type providerRegistry struct {
	mu        sync.RWMutex
	providers map[string]*mg.Provider // path -> loaded provider
	flight    singleflight.Group
}

var registry = &providerRegistry{
	providers: make(map[string]*mg.Provider),
}

// loadProvider returns a provider for the given absolute path, lazily
// initializing and caching it. The cache is a pure performance optimization
// keyed by path (no session state); concurrent loads of the same path are
// coalesced via singleflight.
func loadProvider(path string) (*mg.Provider, error) {
	registry.mu.RLock()
	if p, ok := registry.providers[path]; ok {
		registry.mu.RUnlock()
		return p, nil
	}
	registry.mu.RUnlock()

	result, err, _ := registry.flight.Do(path, func() (interface{}, error) {
		p, err := mg.NewProvider(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load must-gather archive: %w", err)
		}
		registry.mu.Lock()
		registry.providers[path] = p
		registry.mu.Unlock()
		return p, nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*mg.Provider), nil
}

// providerForArchive resolves a must-gather archive ID to its provider using the
// directories configured via --mustgather-dirs. It is the stateless entry point
// shared by all mustgather_* tool handlers.
func providerForArchive(cfg api.MustGatherDirsProvider, id string) (*mg.Provider, error) {
	if id == "" {
		return nil, fmt.Errorf("must_gather_archive_id is required; call mustgather_list to discover available archives")
	}
	path, err := resolveArchivePath(cfg.GetMustGatherDirs(), id)
	if err != nil {
		return nil, err
	}
	return loadProvider(path)
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
