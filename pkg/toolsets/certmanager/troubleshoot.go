package certmanager

import (
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/certmanager"
)

func initTroubleshoot() []api.ServerTool {
	return []api.ServerTool{
		// certificate_troubleshoot
		{
			Tool: api.Tool{
				Name: "certmanager_certificate_troubleshoot",
				Description: `Comprehensive troubleshooting for a cert-manager Certificate.

Analyzes the Certificate and all related resources including:
- Certificate status and conditions
- CertificateRequest status
- Issuer/ClusterIssuer status
- ACME Order and Challenge status (for ACME issuers)
- Recent events
- Suggested next steps

Use this tool when a Certificate is not Ready or you need to diagnose issuance problems.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Namespace of the Certificate",
						},
						"name": {
							Type:        "string",
							Description: "Name of the Certificate to troubleshoot",
						},
					},
					Required: []string{"name", "namespace"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Cert-Manager: Troubleshoot Certificate",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: certificateTroubleshoot,
		},
		// issuer_troubleshoot
		{
			Tool: api.Tool{
				Name: "certmanager_issuer_troubleshoot",
				Description: `Troubleshoot a cert-manager Issuer or ClusterIssuer.

Analyzes the Issuer configuration and status including:
- Issuer type (SelfSigned, CA, ACME, Vault, etc.)
- Ready status and conditions
- ACME account registration status (for ACME issuers)
- Referenced Secret status (for CA/Vault issuers)
- Recent events

Use this tool when an Issuer is not Ready or Certificates are failing to issue.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Namespace of the Issuer (leave empty for ClusterIssuer)",
						},
						"name": {
							Type:        "string",
							Description: "Name of the Issuer or ClusterIssuer",
						},
						"isClusterIssuer": {
							Type:        "boolean",
							Description: "Set to true if this is a ClusterIssuer (default: false)",
							Default:     api.ToRawMessage(false),
						},
					},
					Required: []string{"name"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Cert-Manager: Troubleshoot Issuer",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: issuerTroubleshoot,
		},
		// challenge_troubleshoot
		{
			Tool: api.Tool{
				Name: "certmanager_challenge_troubleshoot",
				Description: `Troubleshoot an ACME Challenge.

Analyzes the Challenge status and provides debugging information for:
- HTTP-01 challenges: Ingress configuration, port 80 accessibility
- DNS-01 challenges: DNS provider configuration, TXT record propagation

Use this tool when ACME certificate issuance is stuck on challenge validation.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Namespace of the Challenge",
						},
						"name": {
							Type:        "string",
							Description: "Name of the Challenge to troubleshoot",
						},
					},
					Required: []string{"name", "namespace"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Cert-Manager: Troubleshoot ACME Challenge",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: challengeTroubleshoot,
		},
	}
}

func certificateTroubleshoot(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := params.GetArguments()["namespace"].(string)
	name := params.GetArguments()["name"].(string)

	client := params.AccessControlClientset().DynamicClient()

	details, err := certmanager.GetCertificateDetails(params.Context, client, namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get certificate details: %v", err)), nil
	}

	report := certmanager.BuildDiagnosticReport(details)

	return api.NewToolCallResult(report, nil), nil
}

func issuerTroubleshoot(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	name := params.GetArguments()["name"].(string)

	namespace := ""
	if ns := params.GetArguments()["namespace"]; ns != nil {
		namespace = ns.(string)
	}

	isClusterIssuer := false
	if ici := params.GetArguments()["isClusterIssuer"]; ici != nil {
		isClusterIssuer = ici.(bool)
	}

	// If namespace is empty and not explicitly a ClusterIssuer, assume ClusterIssuer
	if namespace == "" {
		isClusterIssuer = true
	}

	client := params.AccessControlClientset().DynamicClient()

	details, err := certmanager.GetIssuerDetails(params.Context, client, namespace, name, isClusterIssuer)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get issuer details: %v", err)), nil
	}

	report := buildIssuerReport(details, isClusterIssuer)

	return api.NewToolCallResult(report, nil), nil
}

func buildIssuerReport(details *certmanager.IssuerDetails, isClusterIssuer bool) string {
	var report strings.Builder

	issuerKind := "Issuer"
	if isClusterIssuer {
		issuerKind = "ClusterIssuer"
	}

	issuerName := details.Issuer.GetName()
	issuerNamespace := details.Issuer.GetNamespace()

	if isClusterIssuer {
		report.WriteString(fmt.Sprintf("# %s Troubleshooting Report: %s\n\n", issuerKind, issuerName))
	} else {
		report.WriteString(fmt.Sprintf("# %s Troubleshooting Report: %s/%s\n\n", issuerKind, issuerNamespace, issuerName))
	}

	// Issuer Type
	issuerType := certmanager.GetIssuerType(details.Issuer)
	report.WriteString(fmt.Sprintf("**Type**: %s\n\n", issuerType))

	// Status
	report.WriteString("## Status\n\n")
	conditions := certmanager.ExtractConditions(details.Issuer)
	if len(conditions) == 0 {
		report.WriteString("⚠️ No status conditions found\n")
	} else {
		for _, c := range conditions {
			icon := "✅"
			if c.Status != "True" {
				icon = "❌"
			}
			report.WriteString(fmt.Sprintf("%s **%s**: %s\n", icon, c.Type, c.Status))
			if c.Reason != "" {
				report.WriteString(fmt.Sprintf("   - Reason: %s\n", c.Reason))
			}
			if c.Message != "" {
				report.WriteString(fmt.Sprintf("   - Message: %s\n", c.Message))
			}
		}
	}

	// Type-specific information
	report.WriteString("\n## Configuration\n\n")
	switch issuerType {
	case "selfSigned":
		report.WriteString("Self-signed issuer - no external dependencies.\n")
	case "ca":
		report.WriteString("CA issuer - issues certificates signed by a CA stored in a Secret.\n")
		report.WriteString("Check that the referenced Secret exists and contains valid ca.crt and tls.key.\n")
	case "acme":
		report.WriteString("ACME issuer - issues certificates from an ACME server (e.g., Let's Encrypt).\n")
		report.WriteString("Check ACME account registration status in conditions above.\n")
	case "vault":
		report.WriteString("Vault issuer - issues certificates from HashiCorp Vault PKI.\n")
		report.WriteString("Check Vault connectivity and authentication configuration.\n")
	}

	// Events
	report.WriteString("\n## Recent Events\n\n")
	if len(details.Events) == 0 {
		report.WriteString("No events found\n")
	} else {
		maxEvents := 10
		if len(details.Events) < maxEvents {
			maxEvents = len(details.Events)
		}
		for _, e := range details.Events[:maxEvents] {
			icon := "ℹ️"
			if e.Type == "Warning" {
				icon = "⚠️"
			}
			report.WriteString(fmt.Sprintf("%s **%s**: %s\n", icon, e.Reason, e.Message))
		}
	}

	return report.String()
}

func challengeTroubleshoot(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := params.GetArguments()["namespace"].(string)
	name := params.GetArguments()["name"].(string)

	challenge, err := params.ResourcesGet(params.Context, &certmanager.ChallengeGVK, namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get challenge %s/%s: %v", namespace, name, err)), nil
	}

	report := buildChallengeReport(challenge)

	return api.NewToolCallResult(report, nil), nil
}

func buildChallengeReport(challenge *unstructured.Unstructured) string {
	var report strings.Builder

	name := challenge.GetName()
	namespace := challenge.GetNamespace()

	report.WriteString(fmt.Sprintf("# ACME Challenge Troubleshooting Report: %s/%s\n\n", namespace, name))

	// Challenge type and domain
	challengeType, _, _ := unstructured.NestedString(challenge.Object, "spec", "type")
	domain, _, _ := unstructured.NestedString(challenge.Object, "spec", "dnsName")
	token, _, _ := unstructured.NestedString(challenge.Object, "spec", "token")

	report.WriteString(fmt.Sprintf("**Challenge Type**: %s\n", challengeType))
	report.WriteString(fmt.Sprintf("**Domain**: %s\n", domain))
	report.WriteString(fmt.Sprintf("**Token**: %s\n\n", token))

	// Status
	report.WriteString("## Status\n\n")
	state, _, _ := unstructured.NestedString(challenge.Object, "status", "state")
	reason, _, _ := unstructured.NestedString(challenge.Object, "status", "reason")
	presented, _, _ := unstructured.NestedBool(challenge.Object, "status", "presented")

	stateIcon := "⏳"
	switch state {
	case "valid":
		stateIcon = "✅"
	case "invalid":
		stateIcon = "❌"
	}

	report.WriteString(fmt.Sprintf("%s **State**: %s\n", stateIcon, state))
	report.WriteString(fmt.Sprintf("**Presented**: %v\n", presented))
	if reason != "" {
		report.WriteString(fmt.Sprintf("**Reason**: %s\n", reason))
	}

	// Type-specific debugging
	report.WriteString("\n## Debugging Steps\n\n")
	switch challengeType {
	case "HTTP-01":
		report.WriteString("### HTTP-01 Challenge Debugging\n\n")
		report.WriteString("1. **Verify the challenge URL is accessible**:\n")
		report.WriteString(fmt.Sprintf("   ```\n   curl -v http://%s/.well-known/acme-challenge/%s\n   ```\n\n", domain, token))
		report.WriteString("2. **Check Ingress/Route configuration**:\n")
		report.WriteString("   - Ensure an Ingress or Route exists for this domain\n")
		report.WriteString("   - Verify it routes to the cert-manager solver service\n\n")
		report.WriteString("3. **Common issues**:\n")
		report.WriteString("   - Port 80 blocked by firewall\n")
		report.WriteString("   - Ingress controller not handling the solver Ingress\n")
		report.WriteString("   - DNS not pointing to the cluster\n")
		report.WriteString("   - NetworkPolicy blocking traffic\n")
	case "DNS-01":
		report.WriteString("### DNS-01 Challenge Debugging\n\n")
		report.WriteString("1. **Check if TXT record exists**:\n")
		report.WriteString(fmt.Sprintf("   ```\n   dig +short TXT _acme-challenge.%s\n   ```\n\n", domain))
		report.WriteString("2. **Verify DNS provider credentials**:\n")
		report.WriteString("   - Check the Secret referenced in the Issuer\n")
		report.WriteString("   - Verify API keys/tokens are valid\n\n")
		report.WriteString("3. **Common issues**:\n")
		report.WriteString("   - Invalid DNS provider credentials\n")
		report.WriteString("   - Wrong zone ID or domain\n")
		report.WriteString("   - DNS propagation delay (wait 5-60 minutes)\n")
		report.WriteString("   - Insufficient permissions to create TXT records\n")
	}

	return report.String()
}
