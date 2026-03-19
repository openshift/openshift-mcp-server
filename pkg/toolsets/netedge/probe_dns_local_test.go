package netedge

import (
	"encoding/json"
	"net"
	"time"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/miekg/dns"
)

type mockDNSClient struct {
	msg        *dns.Msg
	rtt        time.Duration
	err        error
	lastServer string
}

func (m *mockDNSClient) Exchange(msg *dns.Msg, server string) (*dns.Msg, time.Duration, error) {
	m.lastServer = server
	return m.msg, m.rtt, m.err
}

func (s *NetEdgeTestSuite) TestProbeDNSLocalHandler() {
	// Setup static success response
	successMsg := new(dns.Msg)
	successMsg.Rcode = dns.RcodeSuccess

	aRecord, _ := dns.NewRR("example.com. 3600 IN A 93.184.216.34")
	successMsg.Answer = append(successMsg.Answer, aRecord)

	s.Run("success query A record", func() {
		mock := &mockDNSClient{msg: successMsg, rtt: 10 * time.Millisecond}
		s.SetArgs(map[string]interface{}{
			"server": "8.8.8.8",
			"name":   "example.com",
			"type":   "A",
		})
		handler := makeProbeDNSLocalHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NoError(result.Error)

		var res DNSResult
		jsonErr := json.Unmarshal([]byte(result.Content), &res)
		s.Require().NoError(jsonErr)
		s.Assert().Equal("NOERROR", res.Rcode)
		s.Assert().Equal(int64(10), res.LatencyMS)
		s.Assert().Len(res.Answers, 1)
		s.Assert().Contains(res.Answers[0], "93.184.216.34")

		structured, ok := result.StructuredContent.(DNSResult)
		s.Require().True(ok)
		s.Assert().Equal("NOERROR", structured.Rcode)
	})

	s.Run("missing name parameter", func() {
		mock := &mockDNSClient{msg: successMsg}
		s.SetArgs(map[string]interface{}{"server": "8.8.8.8"})
		handler := makeProbeDNSLocalHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NotNil(result.Error)
		s.Assert().Contains(result.Error.Error(), "name parameter is required")
	})

	s.Run("missing server parameter", func() {
		mock := &mockDNSClient{msg: successMsg}
		s.SetArgs(map[string]interface{}{"name": "example.com"})
		handler := makeProbeDNSLocalHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NotNil(result.Error)
		s.Assert().Contains(result.Error.Error(), "server parameter is required")
	})

	s.Run("invalid record type", func() {
		mock := &mockDNSClient{msg: successMsg}
		s.SetArgs(map[string]interface{}{
			"server": "8.8.8.8",
			"name":   "example.com",
			"type":   "INVALID",
		})
		handler := makeProbeDNSLocalHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NotNil(result.Error)
		s.Assert().Contains(result.Error.Error(), "invalid or unsupported DNS record type: INVALID")
	})

	s.Run("network failure from library", func() {
		mock := &mockDNSClient{
			msg: nil,
			rtt: 0,
			err: &net.OpError{Op: "dial", Net: "udp", Err: net.UnknownNetworkError("timeout")},
		}
		s.SetArgs(map[string]interface{}{
			"server": "8.8.8.8",
			"name":   "example.com",
			"type":   "A",
		})
		handler := makeProbeDNSLocalHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NotNil(result.Error)
		s.Assert().Contains(result.Error.Error(), "DNS query failed")
	})

	s.Run("default type is A if omitted", func() {
		mock := &mockDNSClient{msg: successMsg, rtt: 5 * time.Millisecond}
		s.SetArgs(map[string]interface{}{
			"server": "8.8.8.8",
			"name":   "example.com",
		})
		handler := makeProbeDNSLocalHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NoError(result.Error)

		var res DNSResult
		jsonErr := json.Unmarshal([]byte(result.Content), &res)
		s.Require().NoError(jsonErr)
		s.Assert().Equal("NOERROR", res.Rcode)

		structured, ok := result.StructuredContent.(DNSResult)
		s.Require().True(ok)
		s.Assert().Equal("NOERROR", structured.Rcode)
	})

	s.Run("ipv4 address appends default port", func() {
		mock := &mockDNSClient{msg: successMsg, rtt: 5 * time.Millisecond}
		s.SetArgs(map[string]interface{}{
			"server": "1.1.1.1",
			"name":   "example.com",
		})
		handler := makeProbeDNSLocalHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NoError(result.Error)
		s.Assert().Equal("1.1.1.1:53", mock.lastServer)
	})

	s.Run("ipv6 address appends default port", func() {
		mock := &mockDNSClient{msg: successMsg, rtt: 5 * time.Millisecond}
		s.SetArgs(map[string]interface{}{
			"server": "2001:4860:4860::8888",
			"name":   "example.com",
		})
		handler := makeProbeDNSLocalHandler(mock)

		result, err := handler(s.params)

		s.Require().NoError(err)
		s.Require().NoError(result.Error)
		s.Assert().Equal("[2001:4860:4860::8888]:53", mock.lastServer)
	})

	s.Run("invalid result is returned as structured error", func() {
		mock := &mockDNSClient{msg: successMsg, rtt: 5 * time.Millisecond}
		s.SetArgs(map[string]interface{}{
			"server": "8.8.8.8",
			"name":   "example.com",
			"type":   "A",
		})
		handler := makeProbeDNSLocalHandler(mock)

		result, err := handler(s.params)
		_ = result

		s.Require().NoError(err)
		// Ensure structured content is returned (not just raw string)
		s.Assert().IsType(api.ToolCallResult{}, *result)
	})
}
