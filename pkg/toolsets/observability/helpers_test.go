package observability

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type HelpersSuite struct {
	suite.Suite
}

func (s *HelpersSuite) TestConvertRelativeTime() {
	s.Run("handles 'now' keyword", func() {
		before := time.Now().UTC()
		result, err := convertRelativeTime("now")
		after := time.Now().UTC()

		s.NoError(err)
		s.Contains(result, "T", "Result should be RFC3339 format")

		// Parse and verify it's within the expected time range
		parsed, err := time.Parse(time.RFC3339, result)
		s.NoError(err)
		s.True(parsed.After(before.Add(-time.Second)) && parsed.Before(after.Add(time.Second)),
			"Parsed time should be close to current time")
	})

	s.Run("handles RFC3339 timestamp unchanged", func() {
		input := "2024-01-01T12:00:00Z"
		result, err := convertRelativeTime(input)

		s.NoError(err)
		s.Equal(input, result, "RFC3339 timestamp should be returned unchanged")
	})

	s.Run("handles Unix timestamp unchanged", func() {
		input := "1704110400"
		result, err := convertRelativeTime(input)

		s.NoError(err)
		s.Equal(input, result, "Unix timestamp should be returned unchanged")
	})

	s.Run("handles relative time -10m", func() {
		before := time.Now().UTC().Add(-10 * time.Minute)
		result, err := convertRelativeTime("-10m")
		after := time.Now().UTC().Add(-10 * time.Minute)

		s.NoError(err)
		s.Contains(result, "T", "Result should be RFC3339 format")

		parsed, err := time.Parse(time.RFC3339, result)
		s.NoError(err)
		s.True(parsed.After(before.Add(-time.Second)) && parsed.Before(after.Add(time.Second)),
			"Parsed time should be approximately 10 minutes ago")
	})

	s.Run("handles relative time -1h", func() {
		before := time.Now().UTC().Add(-1 * time.Hour)
		result, err := convertRelativeTime("-1h")
		after := time.Now().UTC().Add(-1 * time.Hour)

		s.NoError(err)
		s.Contains(result, "T", "Result should be RFC3339 format")

		parsed, err := time.Parse(time.RFC3339, result)
		s.NoError(err)
		s.True(parsed.After(before.Add(-time.Second)) && parsed.Before(after.Add(time.Second)),
			"Parsed time should be approximately 1 hour ago")
	})

	s.Run("handles relative time -1d (days)", func() {
		before := time.Now().UTC().Add(-24 * time.Hour)
		result, err := convertRelativeTime("-1d")
		after := time.Now().UTC().Add(-24 * time.Hour)

		s.NoError(err)
		s.Contains(result, "T", "Result should be RFC3339 format")

		parsed, err := time.Parse(time.RFC3339, result)
		s.NoError(err)
		s.True(parsed.After(before.Add(-time.Second)) && parsed.Before(after.Add(time.Second)),
			"Parsed time should be approximately 1 day ago")
	})

	s.Run("handles relative time -30s (seconds)", func() {
		before := time.Now().UTC().Add(-30 * time.Second)
		result, err := convertRelativeTime("-30s")
		after := time.Now().UTC().Add(-30 * time.Second)

		s.NoError(err)
		s.Contains(result, "T", "Result should be RFC3339 format")

		parsed, err := time.Parse(time.RFC3339, result)
		s.NoError(err)
		s.True(parsed.After(before.Add(-time.Second)) && parsed.Before(after.Add(time.Second)),
			"Parsed time should be approximately 30 seconds ago")
	})

	s.Run("handles whitespace around input", func() {
		result, err := convertRelativeTime("  now  ")

		s.NoError(err)
		s.Contains(result, "T", "Result should be RFC3339 format")
	})

	s.Run("returns error for invalid format", func() {
		_, err := convertRelativeTime("invalid")

		s.Error(err)
		s.Contains(err.Error(), "invalid time format")
	})

	s.Run("returns error for malformed relative time", func() {
		_, err := convertRelativeTime("-abc")

		s.Error(err)
		s.Contains(err.Error(), "invalid relative time format")
	})
}

func (s *HelpersSuite) TestIsNumeric() {
	s.Run("returns true for numeric strings", func() {
		s.True(isNumeric("123"))
		s.True(isNumeric("0"))
		s.True(isNumeric("9876543210"))
	})

	s.Run("returns false for non-numeric strings", func() {
		s.False(isNumeric(""))
		s.False(isNumeric("abc"))
		s.False(isNumeric("12a3"))
		s.False(isNumeric("-123"))
		s.False(isNumeric("12.3"))
	})
}

func (s *HelpersSuite) TestValidatePrometheusEndpoint() {
	s.Run("allows valid query endpoint", func() {
		err := validatePrometheusEndpoint("/api/v1/query")
		s.NoError(err)
	})

	s.Run("allows valid query_range endpoint", func() {
		err := validatePrometheusEndpoint("/api/v1/query_range")
		s.NoError(err)
	})

	s.Run("allows valid series endpoint", func() {
		err := validatePrometheusEndpoint("/api/v1/series")
		s.NoError(err)
	})

	s.Run("allows valid labels endpoint", func() {
		err := validatePrometheusEndpoint("/api/v1/labels")
		s.NoError(err)
	})

	s.Run("allows valid label values endpoint", func() {
		err := validatePrometheusEndpoint("/api/v1/label/job/values")
		s.NoError(err)
	})

	s.Run("rejects invalid endpoint", func() {
		err := validatePrometheusEndpoint("/api/v1/admin/tsdb/delete_series")
		s.Error(err)
		s.Contains(err.Error(), "not allowed")
	})

	s.Run("rejects arbitrary path", func() {
		err := validatePrometheusEndpoint("/some/random/path")
		s.Error(err)
	})
}

func (s *HelpersSuite) TestValidateAlertmanagerEndpoint() {
	s.Run("allows valid v2 alerts endpoint", func() {
		err := validateAlertmanagerEndpoint("/api/v2/alerts")
		s.NoError(err)
	})

	s.Run("allows valid v2 silences endpoint", func() {
		err := validateAlertmanagerEndpoint("/api/v2/silences")
		s.NoError(err)
	})

	s.Run("allows valid v1 alerts endpoint", func() {
		err := validateAlertmanagerEndpoint("/api/v1/alerts")
		s.NoError(err)
	})

	s.Run("rejects invalid endpoint", func() {
		err := validateAlertmanagerEndpoint("/api/v2/status")
		s.Error(err)
		s.Contains(err.Error(), "not allowed")
	})
}

func (s *HelpersSuite) TestBuildQueryURL() {
	s.Run("builds URL without parameters", func() {
		result := buildQueryURL("https://example.com", "/api/v1/query", nil)
		s.Equal("https://example.com/api/v1/query", result)
	})

	s.Run("builds URL with parameters", func() {
		params := make(map[string][]string)
		params["query"] = []string{"up"}
		result := buildQueryURL("https://example.com", "/api/v1/query", params)
		s.Contains(result, "https://example.com/api/v1/query?")
		s.Contains(result, "query=up")
	})
}

func (s *HelpersSuite) TestTruncateString() {
	s.Run("returns original string if shorter than max", func() {
		result := truncateString("hello", 10)
		s.Equal("hello", result)
	})

	s.Run("returns original string if equal to max", func() {
		result := truncateString("hello", 5)
		s.Equal("hello", result)
	})

	s.Run("truncates and adds ellipsis if longer than max", func() {
		result := truncateString("hello world", 5)
		s.Equal("hello...", result)
	})
}

func (s *HelpersSuite) TestPrettyJSON() {
	s.Run("formats valid JSON", func() {
		input := []byte(`{"key":"value","number":123}`)
		result, err := prettyJSON(input)

		s.NoError(err)
		s.Contains(result, "\"key\"")
		s.Contains(result, "\"value\"")
		s.Contains(result, "\n") // Should have newlines for pretty printing
	})

	s.Run("returns original for invalid JSON", func() {
		input := []byte("not valid json")
		result, err := prettyJSON(input)

		s.NoError(err)
		s.Equal("not valid json", result)
	})
}

func TestHelpersSuite(t *testing.T) {
	suite.Run(t, new(HelpersSuite))
}
