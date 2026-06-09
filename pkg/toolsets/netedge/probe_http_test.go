package netedge

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type mockHTTPClient struct {
	resp *http.Response
	err  error
}

func (m *mockHTTPClient) Do(_ *http.Request) (*http.Response, error) {
	return m.resp, m.err
}

func newMockResponse(statusCode int, headers http.Header) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     headers,
		Body:       http.NoBody,
	}
}

func (s *NetEdgeTestSuite) TestProbeHTTPHandler() {
	s.Run("success GET request", func() {
		headers := http.Header{
			"Content-Type": []string{"application/json"},
			"X-Request-Id": []string{"abc-123"},
		}
		mock := &mockHTTPClient{resp: newMockResponse(200, headers)}
		s.SetArgs(map[string]interface{}{"url": "https://example.com"})
		handler := makeProbeHTTPHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NoError(result.Error)

		var res HTTPResult
		jsonErr := json.Unmarshal([]byte(result.Content), &res)
		s.Require().NoError(jsonErr)
		s.Assert().Equal(200, res.StatusCode)
		s.Assert().Equal([]string{"application/json"}, res.Headers["Content-Type"])
		s.Assert().GreaterOrEqual(res.LatencyMS, int64(0))

		structured, ok := result.StructuredContent.(HTTPResult)
		s.Require().True(ok)
		s.Assert().Equal(200, structured.StatusCode)
	})

	s.Run("missing url parameter", func() {
		mock := &mockHTTPClient{}
		s.SetArgs(map[string]interface{}{})
		handler := makeProbeHTTPHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NotNil(result.Error)
		s.Assert().Contains(result.Error.Error(), "url parameter is required")
	})

	s.Run("default method is GET", func() {
		mock := &mockHTTPClient{resp: newMockResponse(200, http.Header{})}
		s.SetArgs(map[string]interface{}{"url": "https://example.com"})
		handler := makeProbeHTTPHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NoError(result.Error)
	})

	s.Run("explicit method POST", func() {
		mock := &mockHTTPClient{resp: newMockResponse(201, http.Header{})}
		s.SetArgs(map[string]interface{}{
			"url":    "https://example.com/resource",
			"method": "post",
		})
		handler := makeProbeHTTPHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NoError(result.Error)

		var res HTTPResult
		jsonErr := json.Unmarshal([]byte(result.Content), &res)
		s.Require().NoError(jsonErr)
		s.Assert().Equal(201, res.StatusCode)
	})

	s.Run("default timeout_seconds is 5", func() {
		mock := &mockHTTPClient{resp: newMockResponse(200, http.Header{})}
		s.SetArgs(map[string]interface{}{"url": "https://example.com"})
		handler := makeProbeHTTPHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NoError(result.Error)
	})

	s.Run("custom timeout_seconds", func() {
		mock := &mockHTTPClient{resp: newMockResponse(200, http.Header{})}
		s.SetArgs(map[string]interface{}{
			"url":             "https://example.com",
			"timeout_seconds": float64(10),
		})
		handler := makeProbeHTTPHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NoError(result.Error)
	})

	s.Run("network error returns structured error", func() {
		mock := &mockHTTPClient{
			resp: nil,
			err:  fmt.Errorf("connection refused"),
		}
		s.SetArgs(map[string]interface{}{"url": "https://unreachable.example.com"})
		handler := makeProbeHTTPHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NotNil(result.Error)
		s.Assert().Contains(result.Error.Error(), "HTTP request failed")
		s.Assert().Contains(result.Error.Error(), "connection refused")
	})

	s.Run("invalid url returns structured error", func() {
		mock := &mockHTTPClient{}
		s.SetArgs(map[string]interface{}{"url": "://invalid-url"})
		handler := makeProbeHTTPHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NotNil(result.Error)
		s.Assert().Contains(result.Error.Error(), "failed to create HTTP request")
	})

	s.Run("non-200 status code is returned without error", func() {
		mock := &mockHTTPClient{resp: newMockResponse(404, http.Header{})}
		s.SetArgs(map[string]interface{}{"url": "https://example.com/notfound"})
		handler := makeProbeHTTPHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NoError(result.Error)

		var res HTTPResult
		jsonErr := json.Unmarshal([]byte(result.Content), &res)
		s.Require().NoError(jsonErr)
		s.Assert().Equal(404, res.StatusCode)
	})

	s.Run("latency is measured", func() {
		mock := &mockHTTPClient{resp: newMockResponse(200, http.Header{})}
		s.SetArgs(map[string]interface{}{"url": "https://example.com"})
		handler := makeProbeHTTPHandler(mock)

		before := time.Now()
		result, err := handler(s.params)
		_ = time.Since(before)

		s.Require().NoError(err)
		s.Require().NoError(result.Error)

		var res HTTPResult
		jsonErr := json.Unmarshal([]byte(result.Content), &res)
		s.Require().NoError(jsonErr)
		s.Assert().GreaterOrEqual(res.LatencyMS, int64(0))
	})
}
