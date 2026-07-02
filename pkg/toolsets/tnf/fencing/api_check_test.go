package fencing

import (
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "connection refused",
			err:      fmt.Errorf("dial tcp 192.168.111.5:6443: connect: connection refused"),
			expected: true,
		},
		{
			name:     "no route to host",
			err:      fmt.Errorf("dial tcp 192.168.111.5:6443: connect: no route to host"),
			expected: true,
		},
		{
			name:     "no such host",
			err:      fmt.Errorf("dial tcp: lookup api.cluster.example.com: no such host"),
			expected: true,
		},
		{
			name:     "i/o timeout",
			err:      fmt.Errorf("dial tcp 192.168.111.5:6443: i/o timeout"),
			expected: true,
		},
		{
			name:     "net.OpError",
			err:      &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")},
			expected: true,
		},
		{
			name:     "wrapped dial error",
			err:      fmt.Errorf("get server version: %w", fmt.Errorf("dial tcp 10.0.0.1:6443: connect: connection refused")),
			expected: true,
		},
		{
			name:     "context deadline exceeded",
			err:      fmt.Errorf("Get \"https://127.0.0.1:6443/version?timeout=5s\": context deadline exceeded"),
			expected: true,
		},
		{
			name:     "connection reset by peer",
			err:      fmt.Errorf("read tcp 127.0.0.1:47428->127.0.0.1:6443: read: connection reset by peer"),
			expected: true,
		},
		{
			name:     "k8s wrapPreviousError style",
			err:      fmt.Errorf("Get \"https://127.0.0.1:16443/version?timeout=5s\": context deadline exceeded - error from a previous attempt: read tcp 127.0.0.1:47428->127.0.0.1:16443: read: connection reset by peer"),
			expected: true,
		},
		{
			name:     "forbidden (API reachable)",
			err:      fmt.Errorf("the server has asked for the client to provide credentials"),
			expected: false,
		},
		{
			name:     "not found (API reachable)",
			err:      fmt.Errorf("the server could not find the requested resource"),
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("something went wrong"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isConnectionError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsAPIUnreachableNilClient(t *testing.T) {
	assert.False(t, IsAPIUnreachable(nil))
}

func TestAPIUnreachableGuide(t *testing.T) {
	guide := APIUnreachableGuide()

	assert.Contains(t, guide, "API Unreachable")
	assert.Contains(t, guide, "Scenario 1")
	assert.Contains(t, guide, "Scenario 2")
	assert.Contains(t, guide, "Scenario 3")
	assert.Contains(t, guide, "BMC")
	assert.Contains(t, guide, "ssh core@")
	assert.Contains(t, guide, "journalctl -u agent")
	assert.Contains(t, guide, "rpm-ostree status")
	assert.Contains(t, guide, "pcs status")
	assert.Contains(t, guide, "etcdctl")
	assert.Contains(t, guide, "tnf_check_fencing_config")
}
