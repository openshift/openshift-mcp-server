package prometheus

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	// DefaultTimeout is the default HTTP timeout.
	DefaultTimeout = 30 * time.Second

	// MaxResponseSize is the maximum response size (10MB).
	MaxResponseSize = 10 * 1024 * 1024
)

// Client is an HTTP client for Prometheus and Alertmanager APIs.
type Client struct {
	baseURL     string
	bearerToken string
	tlsConfig   *tls.Config
	timeout     time.Duration
}

// NewClient creates a new Prometheus client with the specified base URL and options.
func NewClient(baseURL string, opts ...ClientOption) *Client {
	c := &Client{
		baseURL:   baseURL,
		tlsConfig: newDefaultTLSConfig(),
		timeout:   DefaultTimeout,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Query executes an instant PromQL query at the specified time.
// If timeStr is empty, the current time is used.
func (c *Client) Query(ctx context.Context, query string, timeStr string) (*QueryResult, error) {
	params := url.Values{}
	params.Set("query", query)
	if timeStr != "" {
		params.Set("time", timeStr)
	}

	body, err := c.executeRequest(ctx, "/api/v1/query", params)
	if err != nil {
		return nil, err
	}

	var result QueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse query response: %w", err)
	}

	return &result, nil
}

// QueryRange executes a range PromQL query over the specified time range.
func (c *Client) QueryRange(ctx context.Context, query, start, end, step string) (*QueryResult, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("start", start)
	params.Set("end", end)
	params.Set("step", step)

	body, err := c.executeRequest(ctx, "/api/v1/query_range", params)
	if err != nil {
		return nil, err
	}

	var result QueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse query_range response: %w", err)
	}

	return &result, nil
}

// QueryRaw executes a query and returns the raw JSON response.
func (c *Client) QueryRaw(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	return c.executeRequest(ctx, endpoint, params)
}

// executeRequest executes an HTTP GET request with authentication.
func (c *Client) executeRequest(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	// Build URL
	requestURL := c.baseURL + endpoint
	if len(params) > 0 {
		requestURL += "?" + params.Encode()
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication
	if c.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	}

	// Execute request
	client := c.createHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response with size limit
	limitedReader := io.LimitReader(resp.Body, MaxResponseSize+1)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if len(body) > MaxResponseSize {
		return nil, fmt.Errorf("response size exceeds maximum of %d bytes", MaxResponseSize)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncateString(string(body), 200))
	}

	return body, nil
}

// createHTTPClient creates an HTTP client with the configured TLS and timeout settings.
func (c *Client) createHTTPClient() *http.Client {
	return &http.Client{
		Timeout: c.timeout,
		Transport: &http.Transport{
			TLSClientConfig: c.tlsConfig,
		},
	}
}

// truncateString truncates a string to the specified length.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
