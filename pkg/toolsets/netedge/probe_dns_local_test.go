package netedge

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDNSClient struct {
	msg *dns.Msg
	rtt time.Duration
	err error
}

func (m *mockDNSClient) Exchange(msg *dns.Msg, server string) (*dns.Msg, time.Duration, error) {
	return m.msg, m.rtt, m.err
}

func TestProbeDNSLocalHandler(t *testing.T) {
	// Setup static success response
	successMsg := new(dns.Msg)
	successMsg.Rcode = dns.RcodeSuccess

	aRecord, _ := dns.NewRR("example.com. 3600 IN A 93.184.216.34")
	successMsg.Answer = append(successMsg.Answer, aRecord)

	tests := []struct {
		name          string
		args          map[string]interface{}
		mockClient    *mockDNSClient
		expectedError string
		validate      func(t *testing.T, result string)
	}{
		{
			name: "success query A record",
			args: map[string]interface{}{
				"server": "8.8.8.8",
				"name":   "example.com",
				"type":   "A",
			},
			mockClient: &mockDNSClient{
				msg: successMsg,
				rtt: 10 * time.Millisecond,
				err: nil,
			},
			validate: func(t *testing.T, content string) {
				var res DNSResult
				err := json.Unmarshal([]byte(content), &res)
				require.NoError(t, err)
				assert.Equal(t, "NOERROR", res.Rcode)
				assert.Equal(t, int64(10), res.LatencyMS)
				assert.Len(t, res.Answers, 1)
				assert.Contains(t, res.Answers[0], "93.184.216.34")
			},
		},
		{
			name: "missing name parameter",
			args: map[string]interface{}{
				"server": "8.8.8.8",
			},
			expectedError: "name parameter is required",
		},
		{
			name: "missing server parameter",
			args: map[string]interface{}{
				"name": "example.com",
			},
			expectedError: "server parameter is required",
		},
		{
			name: "invalid record type",
			args: map[string]interface{}{
				"server": "8.8.8.8",
				"name":   "example.com",
				"type":   "INVALID",
			},
			expectedError: "invalid or unsupported DNS record type: INVALID",
		},
		{
			name: "network failure from library",
			args: map[string]interface{}{
				"server": "8.8.8.8",
				"name":   "example.com",
				"type":   "A",
			},
			mockClient: &mockDNSClient{
				msg: nil,
				rtt: 0,
				err: &net.OpError{Op: "dial", Net: "udp", Err: net.UnknownNetworkError("timeout")},
			},
			expectedError: "DNS query failed",
		},
		{
			name: "default type is A if omitted",
			args: map[string]interface{}{
				"server": "8.8.8.8",
				"name":   "example.com",
			},
			mockClient: &mockDNSClient{
				msg: successMsg,
				rtt: 5 * time.Millisecond,
				err: nil,
			},
			validate: func(t *testing.T, content string) {
				var res DNSResult
				err := json.Unmarshal([]byte(content), &res)
				require.NoError(t, err)
				assert.Equal(t, "NOERROR", res.Rcode)
			},
		},
	}

	// Stash original
	origClient := activeDNSClient
	defer func() {
		activeDNSClient = origClient
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockClient != nil {
				activeDNSClient = tt.mockClient
			}

			// Mock toolcall params
			toolReq := &mockToolCallRequest{args: tt.args}
			params := api.ToolHandlerParams{
				Context:         context.Background(),
				ToolCallRequest: toolReq,
			}

			result, err := probeDNSLocalHandler(params)

			// The handler wraps the validation error into NewToolCallResult("", err) or returns err directly?
			// Actually our handler returns api.NewToolCallResult("", err), nil for errors so err is always nil.
			require.NoError(t, err)

			if tt.expectedError != "" {
				require.NotNil(t, result.Error)
				assert.Contains(t, result.Error.Error(), tt.expectedError)
			} else {
				require.NotNil(t, result)
				require.NoError(t, result.Error)
				if tt.validate != nil {
					tt.validate(t, result.Content)
				}
			}
		})
	}
}

// Add mockToolCallRequest so tests can run without importing cycle
type mockToolCallRequest struct {
	args map[string]interface{}
}

func (m *mockToolCallRequest) GetArguments() map[string]interface{} {
	return m.args
}

func (m *mockToolCallRequest) GetName() string {
	return "mock_tool"
}
