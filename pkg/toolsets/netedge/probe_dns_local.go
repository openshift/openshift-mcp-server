package netedge

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/miekg/dns"
	"k8s.io/utils/ptr"
)

// dnsExchange interface allows mocking the dns client for tests.
type dnsExchange interface {
	Exchange(m *dns.Msg, a string) (r *dns.Msg, rtt time.Duration, err error)
}

// defaultDNSClient wraps the miekg/dns client.
type defaultDNSClient struct {
	client *dns.Client
}

func (d *defaultDNSClient) Exchange(m *dns.Msg, a string) (*dns.Msg, time.Duration, error) {
	return d.client.Exchange(m, a)
}

// DNSResult represents the required JSON response format for probe_dns_local
type DNSResult struct {
	Answers   []string `json:"answers"`
	Rcode     string   `json:"rcode"`
	LatencyMS int64    `json:"latency_ms"`
}

func initProbeDNSLocal() []api.ServerTool {
	return initProbeDNSLocalWith(&defaultDNSClient{client: new(dns.Client)})
}

// initProbeDNSLocalWith creates probe_dns_local tools using the provided dnsExchange.
// Use initProbeDNSLocal() for production; pass a mock to this function in tests.
func initProbeDNSLocalWith(client dnsExchange) []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "probe_dns_local",
				Description: "Run a DNS query using local libraries on the MCP server host to verify connectivity and resolution.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"server": {
							Type:        "string",
							Description: "DNS server IP (e.g. 8.8.8.8, 10.0.0.10)",
						},
						"name": {
							Type:        "string",
							Description: "FQDN to query",
						},
						"type": {
							Type:        "string",
							Description: "Record type (A, AAAA, CNAME, TXT, SRV, etc.). Defaults to A.",
							Default:     json.RawMessage(`"A"`),
						},
					},
					Required: []string{"server", "name"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Probe DNS (Local)",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			ClusterAware: ptr.To(false),
			Handler:      makeProbeDNSLocalHandler(client),
		},
	}
}

func makeProbeDNSLocalHandler(client dnsExchange) api.ToolHandlerFunc {
	return func(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
		serverParam, ok := params.GetArguments()["server"].(string)
		if !ok || serverParam == "" {
			return api.NewToolCallResultStructured(nil, fmt.Errorf("server parameter is required")), nil
		}

		nameParam, ok := params.GetArguments()["name"].(string)
		if !ok || nameParam == "" {
			return api.NewToolCallResultStructured(nil, fmt.Errorf("name parameter is required")), nil
		}

		typeParam, ok := params.GetArguments()["type"].(string)
		if !ok || typeParam == "" {
			typeParam = "A"
		}

		// Ensure name falls back to a FQDN format
		fqdn := dns.Fqdn(nameParam)

		// Ensure server parameter has a port
		if _, _, err := net.SplitHostPort(serverParam); err != nil {
			// If port is missing, try adding default DNS port 53
			appended := net.JoinHostPort(serverParam, "53")
			if _, _, err2 := net.SplitHostPort(appended); err2 == nil {
				serverParam = appended
			} else {
				return api.NewToolCallResultStructured(nil, fmt.Errorf("invalid server address format: %w", err)), nil
			}
		}

		recordType, ok := dns.StringToType[strings.ToUpper(typeParam)]
		if !ok {
			return api.NewToolCallResultStructured(nil, fmt.Errorf("invalid or unsupported DNS record type: %s", typeParam)), nil
		}

		msg := new(dns.Msg)
		msg.SetQuestion(fqdn, recordType)
		msg.RecursionDesired = true

		resp, rtt, err := client.Exchange(msg, serverParam)

		if err != nil {
			// Log network level errors directly to the tool output so agent can interpret it
			return api.NewToolCallResultStructured(nil, fmt.Errorf("DNS query failed: %w", err)), nil
		}

		result := DNSResult{
			Answers:   make([]string, 0, len(resp.Answer)),
			Rcode:     dns.RcodeToString[resp.Rcode],
			LatencyMS: rtt.Milliseconds(),
		}

		for _, answer := range resp.Answer {
			// Replace tabs with spaces for prettier JSON presentation if needed
			ans := strings.ReplaceAll(answer.String(), "\t", " ")
			result.Answers = append(result.Answers, ans)
		}

		return api.NewToolCallResultStructured(result, nil), nil
	}
}
