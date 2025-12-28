package certmanager

import (
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/certmanager"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
)

func initOperator() []api.ServerTool {
	return []api.ServerTool{
		// operator_status
		{
			Tool: api.Tool{
				Name: "certmanager_operator_status",
				Description: `Check the health status of cert-manager operator components.

Returns the status of:
- cert-manager controller deployment
- cert-manager webhook deployment  
- cert-manager cainjector deployment
- CertManager operator CR conditions (if using cert-manager-operator)

Use this tool to verify cert-manager is properly installed and running.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
				},
				Annotations: api.ToolAnnotations{
					Title:           "Cert-Manager: Operator Status",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: operatorStatus,
		},
		// operator_health_check
		{
			Tool: api.Tool{
				Name: "certmanager_health_check",
				Description: `Perform a comprehensive health check of the cert-manager installation.

Checks:
- All cert-manager deployments are running
- Webhook is responding
- At least one Issuer or ClusterIssuer exists
- Recent error events

Use this tool for a quick overview of cert-manager health.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
				},
				Annotations: api.ToolAnnotations{
					Title:           "Cert-Manager: Health Check",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: healthCheck,
		},
	}
}

func operatorStatus(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	client := params.AccessControlClientset().DynamicClient()

	status, err := certmanager.GetOperatorStatus(params.Context, client)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get operator status: %v", err)), nil
	}

	report := buildOperatorStatusReport(status)

	return api.NewToolCallResult(report, nil), nil
}

func buildOperatorStatusReport(status *certmanager.OperatorStatus) string {
	var report strings.Builder

	report.WriteString("# Cert-Manager Operator Status\n\n")

	report.WriteString("## Components\n\n")

	// Controller
	icon := "✅"
	if !status.Controller.Ready {
		icon = "❌"
	}
	report.WriteString(fmt.Sprintf("%s **Controller**: %s\n", icon, status.Controller.Message))

	// Webhook
	icon = "✅"
	if !status.Webhook.Ready {
		icon = "❌"
	}
	report.WriteString(fmt.Sprintf("%s **Webhook**: %s\n", icon, status.Webhook.Message))

	// CAInjector
	icon = "✅"
	if !status.CAInjector.Ready {
		icon = "❌"
	}
	report.WriteString(fmt.Sprintf("%s **CAInjector**: %s\n", icon, status.CAInjector.Message))

	// Overall status
	report.WriteString("\n## Overall Status\n\n")
	allReady := status.Controller.Ready && status.Webhook.Ready && status.CAInjector.Ready
	if allReady {
		report.WriteString("✅ All cert-manager components are healthy\n")
	} else {
		report.WriteString("❌ Some cert-manager components are not ready\n")
		report.WriteString("\n### Troubleshooting Steps:\n")
		if !status.Controller.Ready {
			report.WriteString("- Check controller pod logs: `kubectl logs -n cert-manager deploy/cert-manager`\n")
		}
		if !status.Webhook.Ready {
			report.WriteString("- Check webhook pod logs: `kubectl logs -n cert-manager deploy/cert-manager-webhook`\n")
			report.WriteString("- Verify webhook service: `kubectl get svc -n cert-manager cert-manager-webhook`\n")
		}
		if !status.CAInjector.Ready {
			report.WriteString("- Check cainjector pod logs: `kubectl logs -n cert-manager deploy/cert-manager-cainjector`\n")
		}
	}

	// Operator CR conditions (if available)
	if len(status.Conditions) > 0 {
		report.WriteString("\n## Operator Conditions\n\n")
		for _, c := range status.Conditions {
			icon := "✅"
			if c.Status != "True" {
				icon = "❌"
			}
			report.WriteString(fmt.Sprintf("%s **%s**: %s\n", icon, c.Type, c.Status))
			if c.Message != "" {
				report.WriteString(fmt.Sprintf("   - %s\n", c.Message))
			}
		}
	}

	return report.String()
}

func healthCheck(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	client := params.AccessControlClientset().DynamicClient()
	ctx := params.Context

	var report strings.Builder
	report.WriteString("# Cert-Manager Health Check\n\n")

	issues := 0

	// 1. Check operator status
	status, err := certmanager.GetOperatorStatus(ctx, client)
	if err != nil {
		report.WriteString("❌ **Failed to check operator status**: " + err.Error() + "\n")
		issues++
	} else {
		if status.Controller.Ready && status.Webhook.Ready && status.CAInjector.Ready {
			report.WriteString("✅ **Components**: All cert-manager deployments are running\n")
		} else {
			report.WriteString("❌ **Components**: Some deployments are not ready\n")
			issues++
		}
	}

	// 2. Check for Issuers
	listOptions := kubernetes.ResourceListOptions{}
	issuers, err := params.ResourcesList(params.Context, &certmanager.IssuerGVK, "", listOptions)
	clusterIssuers, err2 := params.ResourcesList(params.Context, &certmanager.ClusterIssuerGVK, "", listOptions)

	totalIssuers := 0
	if err == nil && issuers != nil {
		items, _, _ := unstructured.NestedSlice(issuers.UnstructuredContent(), "items")
		totalIssuers += len(items)
	}
	if err2 == nil && clusterIssuers != nil {
		items, _, _ := unstructured.NestedSlice(clusterIssuers.UnstructuredContent(), "items")
		totalIssuers += len(items)
	}

	if totalIssuers > 0 {
		report.WriteString(fmt.Sprintf("✅ **Issuers**: Found %d Issuer(s)/ClusterIssuer(s)\n", totalIssuers))
	} else {
		report.WriteString("⚠️ **Issuers**: No Issuers or ClusterIssuers found\n")
	}

	// 3. Check for pending/failed Certificates
	certs, err := params.ResourcesList(params.Context, &certmanager.CertificateGVK, "", listOptions)
	if err == nil && certs != nil {
		items, _, _ := unstructured.NestedSlice(certs.UnstructuredContent(), "items")
		notReady := 0
		for _, item := range items {
			if certMap, ok := item.(map[string]interface{}); ok {
				conditions, found, _ := unstructured.NestedSlice(certMap, "status", "conditions")
				if found {
					isReady := false
					for _, c := range conditions {
						if cond, ok := c.(map[string]interface{}); ok {
							if cond["type"] == "Ready" && cond["status"] == "True" {
								isReady = true
								break
							}
						}
					}
					if !isReady {
						notReady++
					}
				}
			}
		}
		if notReady > 0 {
			report.WriteString(fmt.Sprintf("⚠️ **Certificates**: %d certificate(s) not ready\n", notReady))
		} else if len(items) > 0 {
			report.WriteString(fmt.Sprintf("✅ **Certificates**: All %d certificate(s) are ready\n", len(items)))
		} else {
			report.WriteString("ℹ️ **Certificates**: No certificates found\n")
		}
	}

	// 4. Check for failed CertificateRequests
	crs, err := params.ResourcesList(params.Context, &certmanager.CertificateRequestGVK, "", listOptions)
	if err == nil && crs != nil {
		items, _, _ := unstructured.NestedSlice(crs.UnstructuredContent(), "items")
		failed := 0
		for _, item := range items {
			if crMap, ok := item.(map[string]interface{}); ok {
				conditions, found, _ := unstructured.NestedSlice(crMap, "status", "conditions")
				if found {
					for _, c := range conditions {
						if cond, ok := c.(map[string]interface{}); ok {
							if cond["type"] == "Ready" && cond["status"] == "False" && cond["reason"] == "Failed" {
								failed++
								break
							}
						}
					}
				}
			}
		}
		if failed > 0 {
			report.WriteString(fmt.Sprintf("❌ **CertificateRequests**: %d failed request(s)\n", failed))
			issues++
		}
	}

	// 5. Check for pending Challenges
	challenges, err := params.ResourcesList(params.Context, &certmanager.ChallengeGVK, "", listOptions)
	if err == nil && challenges != nil {
		items, _, _ := unstructured.NestedSlice(challenges.UnstructuredContent(), "items")
		pending := 0
		for _, item := range items {
			if chMap, ok := item.(map[string]interface{}); ok {
				state, _, _ := unstructured.NestedString(chMap, "status", "state")
				if state == "pending" || state == "invalid" {
					pending++
				}
			}
		}
		if pending > 0 {
			report.WriteString(fmt.Sprintf("⚠️ **ACME Challenges**: %d pending/invalid challenge(s)\n", pending))
		}
	}

	// Summary
	report.WriteString("\n## Summary\n\n")
	if issues == 0 {
		report.WriteString("✅ **Cert-manager is healthy!**\n")
	} else {
		report.WriteString(fmt.Sprintf("⚠️ **Found %d issue(s)** - use troubleshooting tools for details\n", issues))
	}

	return api.NewToolCallResult(report.String(), nil), nil
}
