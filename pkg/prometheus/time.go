package prometheus

import (
	"fmt"
	"strings"
	"time"
)

// ConvertRelativeTime converts relative time strings to RFC3339 timestamps.
// Supports: "now", "-10m", "-1h", "-1d", or passthrough for RFC3339/Unix timestamps.
func ConvertRelativeTime(timeStr string) (string, error) {
	timeStr = strings.TrimSpace(timeStr)

	// If already a timestamp (contains T) or is numeric (Unix timestamp), return as-is
	if strings.Contains(timeStr, "T") || isNumeric(timeStr) {
		return timeStr, nil
	}

	// Handle 'now'
	if timeStr == "now" {
		return time.Now().UTC().Format(time.RFC3339), nil
	}

	// Handle relative times like '-10m', '-1h', '-1d', '-30s'
	if strings.HasPrefix(timeStr, "-") {
		// Parse duration (Go's time.ParseDuration doesn't support 'd' for days)
		durationStr := timeStr[1:] // Remove leading '-'

		// Handle days specially
		if strings.HasSuffix(durationStr, "d") {
			days, err := parseIntFromString(strings.TrimSuffix(durationStr, "d"))
			if err != nil {
				return "", fmt.Errorf("invalid relative time format: %s", timeStr)
			}
			targetTime := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
			return targetTime.Format(time.RFC3339), nil
		}

		// Parse standard durations (s, m, h)
		duration, err := time.ParseDuration(durationStr)
		if err != nil {
			return "", fmt.Errorf("invalid relative time format: %s", timeStr)
		}
		targetTime := time.Now().UTC().Add(-duration)
		return targetTime.Format(time.RFC3339), nil
	}

	return "", fmt.Errorf("invalid time format: %s; expected 'now', relative time like '-10m', '-1h', '-1d', or RFC3339 timestamp", timeStr)
}

// isNumeric checks if a string contains only digits.
func isNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// parseIntFromString parses an integer from a string with overflow protection.
func parseIntFromString(s string) (int, error) {
	if len(s) == 0 {
		return 0, fmt.Errorf("empty string")
	}
	if len(s) > 10 { // int32 max is 10 digits, prevents overflow
		return 0, fmt.Errorf("number too large: %s", s)
	}
	var result int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid number: %s", s)
		}
		result = result*10 + int(c-'0')
	}
	return result, nil
}
