package prometheus

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"k8s.io/client-go/rest"
)

type PrometheusSuite struct {
	suite.Suite
}

func (s *PrometheusSuite) TestNewClient() {
	s.Run("creates client with defaults", func() {
		client := NewClient("https://prometheus.example.com")

		s.Equal("https://prometheus.example.com", client.baseURL)
		s.Equal("", client.bearerToken)
		s.Equal(DefaultTimeout, client.timeout)
		s.NotNil(client.tlsConfig)
	})

	s.Run("applies bearer token option", func() {
		client := NewClient("https://prometheus.example.com",
			WithBearerToken("test-token"),
		)

		s.Equal("test-token", client.bearerToken)
	})

	s.Run("applies timeout option", func() {
		client := NewClient("https://prometheus.example.com",
			WithTimeout(60*time.Second),
		)

		s.Equal(60*time.Second, client.timeout)
	})

	s.Run("applies insecure option", func() {
		client := NewClient("https://prometheus.example.com",
			WithInsecure(true),
		)

		s.True(client.tlsConfig.InsecureSkipVerify)
	})

	s.Run("trims whitespace from bearer token", func() {
		client := NewClient("https://prometheus.example.com",
			WithBearerToken("  test-token  "),
		)

		s.Equal("test-token", client.bearerToken)
	})
}

func (s *PrometheusSuite) TestWithBearerTokenFromRESTConfig() {
	s.Run("uses token from BearerToken field", func() {
		config := &rest.Config{
			BearerToken: "direct-token",
		}

		client := NewClient("https://prometheus.example.com",
			WithBearerTokenFromRESTConfig(config),
		)

		s.Equal("direct-token", client.bearerToken)
	})

	s.Run("handles nil config gracefully", func() {
		client := NewClient("https://prometheus.example.com",
			WithBearerTokenFromRESTConfig(nil),
		)

		s.Equal("", client.bearerToken)
	})
}

func (s *PrometheusSuite) TestWithTLSFromRESTConfig() {
	s.Run("handles nil config gracefully", func() {
		client := NewClient("https://prometheus.example.com",
			WithTLSFromRESTConfig(nil),
		)

		s.NotNil(client.tlsConfig)
	})

	s.Run("uses CAData when available", func() {
		// Create a minimal PEM certificate for testing
		caPEM := []byte(`-----BEGIN CERTIFICATE-----
MIIBkTCB+wIJAKHBfpegPjMCMA0GCSqGSIb3DQEBCwUAMBExDzANBgNVBAMMBnRl
c3RjYTAeFw0yNDAxMDEwMDAwMDBaFw0yNTAxMDEwMDAwMDBaMBExDzANBgNVBAMM
BnRlc3RjYTBcMA0GCSqGSIb3DQEBAQUAA0sAMEgCQQC7o96FCFhP2RxnNwj7mVXh
qGYXt9L9BJVjjTpD2hCRVEJgqGYb3bSoGiK4MYpqnLJDt9IBSfJz7JBkjHDvDZLX
AgMBAAGjUzBRMB0GA1UdDgQWBBQS0P3hKf3cG8XKBQMO3F/3GmZ7wjAfBgNVHSME
GDAWgBQS0P3hKf3cG8XKBQMO3F/3GmZ7wjAPBgNVHRMBAf8EBTADAQH/MA0GCSqG
SIb3DQEBCwUAA0EAFHbN1pWPxvCqVTH1gHCJdNlHqY3hg3PA2PIzv1NiaP3qmJk0
cDq6b5fP0Z3e6Q1OvH5hEYnD6W8fXG5M8CxHjg==
-----END CERTIFICATE-----`)

		config := &rest.Config{
			TLSClientConfig: rest.TLSClientConfig{
				CAData: caPEM,
			},
		}

		client := NewClient("https://prometheus.example.com",
			WithTLSFromRESTConfig(config),
		)

		s.NotNil(client.tlsConfig.RootCAs)
	})
}

func (s *PrometheusSuite) TestQuery() {
	s.Run("executes instant query", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.Equal("/api/v1/query", r.URL.Path)
			s.Equal("up", r.URL.Query().Get("query"))

			response := QueryResult{
				Status: "success",
				Data: Data{
					ResultType: "vector",
					Result: []Result{
						{
							Metric: map[string]string{"__name__": "up", "job": "apiserver"},
							Value:  []any{1234567890.0, "1"},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		result, err := client.Query(context.Background(), "up", "")

		s.NoError(err)
		s.Equal("success", result.Status)
		s.Len(result.Data.Result, 1)
		s.Equal("up", result.Data.Result[0].Metric["__name__"])
	})

	s.Run("includes time parameter when specified", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.Equal("1234567890", r.URL.Query().Get("time"))

			response := QueryResult{Status: "success"}
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		_, err := client.Query(context.Background(), "up", "1234567890")

		s.NoError(err)
	})

	s.Run("includes bearer token in request", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.Equal("Bearer test-token", r.Header.Get("Authorization"))

			response := QueryResult{Status: "success"}
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewClient(server.URL, WithBearerToken("test-token"))
		_, err := client.Query(context.Background(), "up", "")

		s.NoError(err)
	})

	s.Run("returns error for HTTP error status", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		_, err := client.Query(context.Background(), "up", "")

		s.Error(err)
		s.Contains(err.Error(), "500")
	})
}

func (s *PrometheusSuite) TestQueryRange() {
	s.Run("executes range query with all parameters", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.Equal("/api/v1/query_range", r.URL.Path)
			s.Equal("rate(http_requests_total[5m])", r.URL.Query().Get("query"))
			s.Equal("2024-01-01T00:00:00Z", r.URL.Query().Get("start"))
			s.Equal("2024-01-01T01:00:00Z", r.URL.Query().Get("end"))
			s.Equal("1m", r.URL.Query().Get("step"))

			response := QueryResult{
				Status: "success",
				Data: Data{
					ResultType: "matrix",
					Result: []Result{
						{
							Metric: map[string]string{"__name__": "http_requests_total"},
							Values: [][]any{
								{1234567890.0, "10"},
								{1234567950.0, "15"},
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		result, err := client.QueryRange(context.Background(),
			"rate(http_requests_total[5m])",
			"2024-01-01T00:00:00Z",
			"2024-01-01T01:00:00Z",
			"1m",
		)

		s.NoError(err)
		s.Equal("success", result.Status)
		s.Equal("matrix", result.Data.ResultType)
		s.Len(result.Data.Result, 1)
		s.Len(result.Data.Result[0].Values, 2)
	})
}

func (s *PrometheusSuite) TestGetAlerts() {
	s.Run("retrieves alerts with parameters", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.Equal("/api/v2/alerts", r.URL.Path)
			s.Equal("true", r.URL.Query().Get("active"))
			s.Equal("false", r.URL.Query().Get("silenced"))
			s.Equal("false", r.URL.Query().Get("inhibited"))

			alerts := []Alert{
				{
					Labels:      map[string]string{"alertname": "HighCPU", "severity": "warning"},
					Annotations: map[string]string{"summary": "CPU usage is high"},
					StartsAt:    "2024-01-01T00:00:00Z",
					Status: AlertStatus{
						State: "active",
					},
				},
			}
			_ = json.NewEncoder(w).Encode(alerts)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		alerts, err := client.GetAlerts(context.Background(), true, false, false, "")

		s.NoError(err)
		s.Len(alerts, 1)
		s.Equal("HighCPU", alerts[0].Labels["alertname"])
		s.Equal("active", alerts[0].Status.State)
	})

	s.Run("includes filter parameter when specified", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.Equal("alertname=Watchdog", r.URL.Query().Get("filter"))

			alerts := []Alert{}
			_ = json.NewEncoder(w).Encode(alerts)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		_, err := client.GetAlerts(context.Background(), true, false, false, "alertname=Watchdog")

		s.NoError(err)
	})
}

func (s *PrometheusSuite) TestConvertRelativeTime() {
	s.Run("handles 'now' keyword", func() {
		before := time.Now().UTC()
		result, err := ConvertRelativeTime("now")
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
		result, err := ConvertRelativeTime(input)

		s.NoError(err)
		s.Equal(input, result, "RFC3339 timestamp should be returned unchanged")
	})

	s.Run("handles Unix timestamp unchanged", func() {
		input := "1704110400"
		result, err := ConvertRelativeTime(input)

		s.NoError(err)
		s.Equal(input, result, "Unix timestamp should be returned unchanged")
	})

	s.Run("handles relative time -10m", func() {
		before := time.Now().UTC().Add(-10 * time.Minute)
		result, err := ConvertRelativeTime("-10m")
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
		result, err := ConvertRelativeTime("-1h")
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
		result, err := ConvertRelativeTime("-1d")
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
		result, err := ConvertRelativeTime("-30s")
		after := time.Now().UTC().Add(-30 * time.Second)

		s.NoError(err)
		s.Contains(result, "T", "Result should be RFC3339 format")

		parsed, err := time.Parse(time.RFC3339, result)
		s.NoError(err)
		s.True(parsed.After(before.Add(-time.Second)) && parsed.Before(after.Add(time.Second)),
			"Parsed time should be approximately 30 seconds ago")
	})

	s.Run("handles whitespace around input", func() {
		result, err := ConvertRelativeTime("  now  ")

		s.NoError(err)
		s.Contains(result, "T", "Result should be RFC3339 format")
	})

	s.Run("returns error for invalid format", func() {
		_, err := ConvertRelativeTime("invalid")

		s.Error(err)
		s.Contains(err.Error(), "invalid time format")
	})

	s.Run("returns error for malformed relative time", func() {
		_, err := ConvertRelativeTime("-abc")

		s.Error(err)
		s.Contains(err.Error(), "invalid relative time format")
	})
}

func (s *PrometheusSuite) TestTruncateString() {
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

func (s *PrometheusSuite) TestCreateHTTPClient() {
	s.Run("creates client with timeout", func() {
		client := NewClient("https://example.com", WithTimeout(60*time.Second))
		httpClient := client.createHTTPClient()

		s.Equal(60*time.Second, httpClient.Timeout)
	})

	s.Run("creates client with TLS config", func() {
		client := NewClient("https://example.com", WithInsecure(true))
		httpClient := client.createHTTPClient()

		transport, ok := httpClient.Transport.(*http.Transport)
		s.True(ok)
		s.True(transport.TLSClientConfig.InsecureSkipVerify)
	})
}

func (s *PrometheusSuite) TestNewDefaultTLSConfig() {
	s.Run("sets minimum TLS version", func() {
		config := newDefaultTLSConfig()
		s.Equal(uint16(tls.VersionTLS12), config.MinVersion)
	})
}

func (s *PrometheusSuite) TestParseIntFromString() {
	s.Run("returns error for empty string", func() {
		_, err := parseIntFromString("")
		s.Error(err)
		s.Contains(err.Error(), "empty string")
	})

	s.Run("returns error for number too large", func() {
		_, err := parseIntFromString("12345678901") // 11 digits
		s.Error(err)
		s.Contains(err.Error(), "number too large")
	})

	s.Run("parses valid number", func() {
		result, err := parseIntFromString("365")
		s.NoError(err)
		s.Equal(365, result)
	})

	s.Run("parses max allowed digits", func() {
		result, err := parseIntFromString("1234567890") // exactly 10 digits
		s.NoError(err)
		s.Equal(1234567890, result)
	})
}

func TestPrometheusSuite(t *testing.T) {
	suite.Run(t, new(PrometheusSuite))
}
