package certmanager

import (
	"context"
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// GVRs for cert-manager resources (used internally for dynamic client)
var (
	certificateGVR = schema.GroupVersionResource{
		Group:    "cert-manager.io",
		Version:  "v1",
		Resource: "certificates",
	}
	certificateRequestGVR = schema.GroupVersionResource{
		Group:    "cert-manager.io",
		Version:  "v1",
		Resource: "certificaterequests",
	}
	issuerGVR = schema.GroupVersionResource{
		Group:    "cert-manager.io",
		Version:  "v1",
		Resource: "issuers",
	}
	clusterIssuerGVR = schema.GroupVersionResource{
		Group:    "cert-manager.io",
		Version:  "v1",
		Resource: "clusterissuers",
	}
	orderGVR = schema.GroupVersionResource{
		Group:    "acme.cert-manager.io",
		Version:  "v1",
		Resource: "orders",
	}
	challengeGVR = schema.GroupVersionResource{
		Group:    "acme.cert-manager.io",
		Version:  "v1",
		Resource: "challenges",
	}
	deploymentGVR = schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}
	certManagerOperatorGVR = schema.GroupVersionResource{
		Group:    "operator.openshift.io",
		Version:  "v1alpha1",
		Resource: "certmanagers",
	}
)

// GetCertificateDetails fetches a Certificate and all related resources
func GetCertificateDetails(ctx context.Context, client dynamic.Interface, namespace, name string) (*CertificateDetails, error) {
	details := &CertificateDetails{}

	// 1. Get the Certificate
	cert, err := client.Resource(certificateGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate: %w", err)
	}
	details.Certificate = cert

	// 2. Get CertificateRequests for this Certificate
	crList, err := client.Resource(certificateRequestGVR).Namespace(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", LabelCertificateName, name),
	})
	if err == nil && crList != nil {
		for i := range crList.Items {
			details.CertificateRequests = append(details.CertificateRequests, &crList.Items[i])
		}
	}

	// 3. Get the Issuer
	issuerRef := GetIssuerRef(cert)
	if issuerRef.Kind == "ClusterIssuer" {
		issuer, err := client.Resource(clusterIssuerGVR).Get(ctx, issuerRef.Name, metav1.GetOptions{})
		if err == nil {
			details.Issuer = issuer
		}
	} else {
		issuer, err := client.Resource(issuerGVR).Namespace(namespace).Get(ctx, issuerRef.Name, metav1.GetOptions{})
		if err == nil {
			details.Issuer = issuer
		}
	}

	// 4. Get Orders and Challenges (for ACME issuers)
	if details.Issuer != nil && IsACMEIssuer(details.Issuer) {
		// Get Orders related to CertificateRequests
		for _, cr := range details.CertificateRequests {
			orderName := getAnnotation(cr, "cert-manager.io/order-name")
			if orderName != "" {
				order, err := client.Resource(orderGVR).Namespace(namespace).Get(ctx, orderName, metav1.GetOptions{})
				if err == nil {
					details.Orders = append(details.Orders, order)
				}
			}
		}

		// Get Challenges
		challenges, err := client.Resource(challengeGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
		if err == nil && challenges != nil {
			for i := range challenges.Items {
				details.Challenges = append(details.Challenges, &challenges.Items[i])
			}
		}
	}

	// 5. Get Events for the Certificate
	details.Events = GetEventsForResource(ctx, client, namespace, "Certificate", name)

	return details, nil
}

// GetIssuerDetails fetches an Issuer and related information
func GetIssuerDetails(ctx context.Context, client dynamic.Interface, namespace, name string, isClusterIssuer bool) (*IssuerDetails, error) {
	details := &IssuerDetails{}

	var issuer *unstructured.Unstructured
	var err error

	if isClusterIssuer {
		issuer, err = client.Resource(clusterIssuerGVR).Get(ctx, name, metav1.GetOptions{})
	} else {
		issuer, err = client.Resource(issuerGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get issuer: %w", err)
	}
	details.Issuer = issuer

	// Get Events
	kind := "Issuer"
	if isClusterIssuer {
		kind = "ClusterIssuer"
		namespace = "" // ClusterIssuers have cluster-scoped events
	}
	details.Events = GetEventsForResource(ctx, client, namespace, kind, name)

	return details, nil
}

// GetIssuerRef extracts the issuer reference from a Certificate
func GetIssuerRef(cert *unstructured.Unstructured) IssuerRef {
	issuerRef := IssuerRef{
		Kind:  "Issuer", // default
		Group: "cert-manager.io",
	}

	if ref, found, _ := unstructured.NestedMap(cert.Object, "spec", "issuerRef"); found {
		if name, ok := ref["name"].(string); ok {
			issuerRef.Name = name
		}
		if kind, ok := ref["kind"].(string); ok {
			issuerRef.Kind = kind
		}
		if group, ok := ref["group"].(string); ok {
			issuerRef.Group = group
		}
	}

	return issuerRef
}

// IsACMEIssuer checks if an issuer is an ACME issuer
func IsACMEIssuer(issuer *unstructured.Unstructured) bool {
	if issuer == nil {
		return false
	}
	_, found, _ := unstructured.NestedMap(issuer.Object, "spec", "acme")
	return found
}

// GetIssuerType returns the type of issuer (selfSigned, ca, acme, vault, venafi)
func GetIssuerType(issuer *unstructured.Unstructured) string {
	if issuer == nil {
		return "unknown"
	}

	spec, found, _ := unstructured.NestedMap(issuer.Object, "spec")
	if !found {
		return "unknown"
	}

	// Check for each issuer type
	issuerTypes := []string{"selfSigned", "ca", "acme", "vault", "venafi"}
	for _, t := range issuerTypes {
		if _, found := spec[t]; found {
			return t
		}
	}

	return "unknown"
}

// ExtractConditions extracts conditions from a resource's status
func ExtractConditions(obj *unstructured.Unstructured) []Condition {
	if obj == nil {
		return nil
	}

	conditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if !found {
		return nil
	}

	result := make([]Condition, 0, len(conditions))
	for _, c := range conditions {
		cond, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		result = append(result, Condition{
			Type:               getString(cond, "type"),
			Status:             getString(cond, "status"),
			Reason:             getString(cond, "reason"),
			Message:            getString(cond, "message"),
			LastTransitionTime: getString(cond, "lastTransitionTime"),
		})
	}
	return result
}

// GetEventsForResource fetches events for a specific resource
func GetEventsForResource(ctx context.Context, client dynamic.Interface, namespace, kind, name string) []Event {
	eventGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "events"}

	var events *unstructured.UnstructuredList
	var err error

	fieldSelector := fmt.Sprintf("involvedObject.kind=%s,involvedObject.name=%s", kind, name)

	if namespace != "" {
		events, err = client.Resource(eventGVR).Namespace(namespace).List(ctx, metav1.ListOptions{
			FieldSelector: fieldSelector,
		})
	} else {
		events, err = client.Resource(eventGVR).List(ctx, metav1.ListOptions{
			FieldSelector: fieldSelector,
		})
	}

	if err != nil || events == nil {
		return nil
	}

	result := make([]Event, 0, len(events.Items))
	for _, e := range events.Items {
		result = append(result, Event{
			Type:      getString(e.Object, "type"),
			Reason:    getString(e.Object, "reason"),
			Message:   getString(e.Object, "message"),
			Timestamp: getString(e.Object, "lastTimestamp"),
			Count:     getInt32(e.Object, "count"),
		})
	}

	// Sort by timestamp descending (most recent first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp > result[j].Timestamp
	})

	return result
}

// GetOperatorStatus fetches the status of cert-manager operator components
func GetOperatorStatus(ctx context.Context, client dynamic.Interface) (*OperatorStatus, error) {
	status := &OperatorStatus{}

	// Get Controller deployment
	controller, err := client.Resource(deploymentGVR).Namespace(CertManagerNamespace).Get(ctx, ControllerDeploymentName, metav1.GetOptions{})
	if err == nil {
		status.Controller = extractDeploymentStatus(controller, "Controller")
	} else {
		status.Controller = ComponentStatus{Name: "Controller", Ready: false, Message: fmt.Sprintf("Not found: %v", err)}
	}

	// Get Webhook deployment
	webhook, err := client.Resource(deploymentGVR).Namespace(CertManagerNamespace).Get(ctx, WebhookDeploymentName, metav1.GetOptions{})
	if err == nil {
		status.Webhook = extractDeploymentStatus(webhook, "Webhook")
	} else {
		status.Webhook = ComponentStatus{Name: "Webhook", Ready: false, Message: fmt.Sprintf("Not found: %v", err)}
	}

	// Get CAInjector deployment
	cainjector, err := client.Resource(deploymentGVR).Namespace(CertManagerNamespace).Get(ctx, CAInjectorDeploymentName, metav1.GetOptions{})
	if err == nil {
		status.CAInjector = extractDeploymentStatus(cainjector, "CAInjector")
	} else {
		status.CAInjector = ComponentStatus{Name: "CAInjector", Ready: false, Message: fmt.Sprintf("Not found: %v", err)}
	}

	// Try to get the CertManager operator CR
	certManagerCR, err := client.Resource(certManagerOperatorGVR).Get(ctx, "cluster", metav1.GetOptions{})
	if err == nil {
		status.Conditions = ExtractConditions(certManagerCR)
	}

	return status, nil
}

// extractDeploymentStatus extracts status from a deployment
func extractDeploymentStatus(deployment *unstructured.Unstructured, name string) ComponentStatus {
	status := ComponentStatus{Name: name}

	spec, _, _ := unstructured.NestedMap(deployment.Object, "spec")
	statusMap, _, _ := unstructured.NestedMap(deployment.Object, "status")

	if replicas, ok := spec["replicas"].(int64); ok {
		status.DesiredReplicas = int32(replicas)
	}
	if available, ok := statusMap["availableReplicas"].(int64); ok {
		status.AvailableReplicas = int32(available)
	}

	status.Ready = status.AvailableReplicas >= status.DesiredReplicas && status.DesiredReplicas > 0

	if status.Ready {
		status.Message = fmt.Sprintf("%d/%d replicas available", status.AvailableReplicas, status.DesiredReplicas)
	} else {
		status.Message = fmt.Sprintf("%d/%d replicas available (not ready)", status.AvailableReplicas, status.DesiredReplicas)
	}

	return status
}

// BuildDiagnosticReport creates a human-readable diagnostic report from certificate details
func BuildDiagnosticReport(details *CertificateDetails) string {
	var report strings.Builder

	certName := details.Certificate.GetName()
	certNamespace := details.Certificate.GetNamespace()

	report.WriteString(fmt.Sprintf("# Certificate Troubleshooting Report: %s/%s\n\n", certNamespace, certName))

	// Certificate Status
	report.WriteString("## Certificate Status\n\n")
	conditions := ExtractConditions(details.Certificate)
	if len(conditions) == 0 {
		report.WriteString("⚠️ No status conditions found - certificate may not have been reconciled yet\n\n")
	} else {
		for _, c := range conditions {
			icon := getStatusIcon(c.Status)
			report.WriteString(fmt.Sprintf("%s **%s**: %s\n", icon, c.Type, c.Status))
			if c.Reason != "" {
				report.WriteString(fmt.Sprintf("   - Reason: %s\n", c.Reason))
			}
			if c.Message != "" {
				report.WriteString(fmt.Sprintf("   - Message: %s\n", c.Message))
			}
		}
	}

	// Expiry information
	notAfter, found, _ := unstructured.NestedString(details.Certificate.Object, "status", "notAfter")
	if found {
		report.WriteString(fmt.Sprintf("\n📅 **Expires**: %s\n", notAfter))
	}
	renewalTime, found, _ := unstructured.NestedString(details.Certificate.Object, "status", "renewalTime")
	if found {
		report.WriteString(fmt.Sprintf("🔄 **Renewal scheduled**: %s\n", renewalTime))
	}

	// CertificateRequest Status
	report.WriteString("\n## CertificateRequest Status\n\n")
	if len(details.CertificateRequests) == 0 {
		report.WriteString("⚠️ No CertificateRequests found for this certificate\n")
	} else {
		for _, cr := range details.CertificateRequests {
			crName := cr.GetName()
			crConditions := ExtractConditions(cr)
			report.WriteString(fmt.Sprintf("### %s\n", crName))
			for _, c := range crConditions {
				icon := getStatusIcon(c.Status)
				report.WriteString(fmt.Sprintf("%s **%s**: %s\n", icon, c.Type, c.Status))
				if c.Message != "" {
					report.WriteString(fmt.Sprintf("   - %s\n", c.Message))
				}
			}
		}
	}

	// Issuer Status
	report.WriteString("\n## Issuer Status\n\n")
	if details.Issuer == nil {
		issuerRef := GetIssuerRef(details.Certificate)
		report.WriteString(fmt.Sprintf("❌ **Issuer not found**: %s/%s\n", issuerRef.Kind, issuerRef.Name))
		report.WriteString("   - Verify the issuer exists and is spelled correctly\n")
	} else {
		issuerName := details.Issuer.GetName()
		issuerKind := details.Issuer.GetKind()
		issuerType := GetIssuerType(details.Issuer)
		issuerConditions := ExtractConditions(details.Issuer)

		report.WriteString(fmt.Sprintf("**%s**: %s (type: %s)\n", issuerKind, issuerName, issuerType))
		for _, c := range issuerConditions {
			icon := getStatusIcon(c.Status)
			report.WriteString(fmt.Sprintf("%s **%s**: %s\n", icon, c.Type, c.Status))
			if c.Message != "" {
				report.WriteString(fmt.Sprintf("   - %s\n", c.Message))
			}
		}
	}

	// ACME Orders (if applicable)
	if len(details.Orders) > 0 {
		report.WriteString("\n## ACME Orders\n\n")
		for _, order := range details.Orders {
			orderName := order.GetName()
			state, _, _ := unstructured.NestedString(order.Object, "status", "state")
			report.WriteString(fmt.Sprintf("- **%s**: %s\n", orderName, state))
		}
	}

	// ACME Challenges (if applicable)
	if len(details.Challenges) > 0 {
		report.WriteString("\n## ACME Challenges\n\n")
		for _, challenge := range details.Challenges {
			challengeName := challenge.GetName()
			challengeType, _, _ := unstructured.NestedString(challenge.Object, "spec", "type")
			state, _, _ := unstructured.NestedString(challenge.Object, "status", "state")
			reason, _, _ := unstructured.NestedString(challenge.Object, "status", "reason")

			icon := "⏳"
			switch state {
			case "valid":
				icon = "✅"
			case "invalid":
				icon = "❌"
			}

			report.WriteString(fmt.Sprintf("%s **%s** (%s): %s\n", icon, challengeName, challengeType, state))
			if reason != "" {
				report.WriteString(fmt.Sprintf("   - %s\n", reason))
			}
		}
	}

	// Recent Events
	report.WriteString("\n## Recent Events\n\n")
	if len(details.Events) == 0 {
		report.WriteString("No events found\n")
	} else {
		// Limit to last 10 events
		maxEvents := 10
		if len(details.Events) < maxEvents {
			maxEvents = len(details.Events)
		}
		for _, e := range details.Events[:maxEvents] {
			icon := "ℹ️"
			if e.Type == string(corev1.EventTypeWarning) {
				icon = "⚠️"
			}
			report.WriteString(fmt.Sprintf("%s **%s**: %s\n", icon, e.Reason, e.Message))
		}
	}

	// Suggested next steps
	report.WriteString("\n## Suggested Next Steps\n\n")
	report.WriteString(generateSuggestions(details))

	return report.String()
}

// generateSuggestions creates context-aware suggestions based on the certificate state
func generateSuggestions(details *CertificateDetails) string {
	var suggestions strings.Builder

	conditions := ExtractConditions(details.Certificate)
	isReady := false
	isIssuing := false

	for _, c := range conditions {
		if c.Type == "Ready" && c.Status == "True" {
			isReady = true
		}
		if c.Type == "Issuing" && c.Status == "True" {
			isIssuing = true
		}
	}

	if isReady {
		suggestions.WriteString("✅ Certificate is healthy. No action required.\n")
		return suggestions.String()
	}

	if isIssuing {
		suggestions.WriteString("1. Certificate is currently being issued - this is normal\n")
		suggestions.WriteString("2. Wait a few minutes and check status again\n")
		suggestions.WriteString("3. If stuck, check CertificateRequest status above\n")
	}

	if details.Issuer == nil {
		suggestions.WriteString("1. Create the missing Issuer/ClusterIssuer\n")
		suggestions.WriteString("2. Verify the issuerRef in the Certificate spec is correct\n")
	} else {
		issuerConditions := ExtractConditions(details.Issuer)
		issuerReady := false
		for _, c := range issuerConditions {
			if c.Type == "Ready" && c.Status == "True" {
				issuerReady = true
			}
		}
		if !issuerReady {
			suggestions.WriteString("1. **Fix the Issuer first** - it's not ready\n")
			suggestions.WriteString("2. Check Issuer configuration and referenced Secrets\n")
		}
	}

	if len(details.CertificateRequests) == 0 && !isIssuing {
		suggestions.WriteString("1. Check if cert-manager controller is running\n")
		suggestions.WriteString("2. Use `certmanager_operator_status` to verify components\n")
		suggestions.WriteString("3. Check cert-manager controller logs with `certmanager_controller_logs`\n")
	}

	// ACME-specific suggestions
	for _, challenge := range details.Challenges {
		state, _, _ := unstructured.NestedString(challenge.Object, "status", "state")
		if state == "pending" || state == "invalid" {
			challengeType, _, _ := unstructured.NestedString(challenge.Object, "spec", "type")
			switch challengeType {
			case "HTTP-01":
				suggestions.WriteString("\n**HTTP-01 Challenge Debugging:**\n")
				suggestions.WriteString("- Verify Ingress/Route is configured and accessible\n")
				suggestions.WriteString("- Check that port 80 is reachable from the internet\n")
				suggestions.WriteString("- Test: `curl http://<domain>/.well-known/acme-challenge/test`\n")
			case "DNS-01":
				suggestions.WriteString("\n**DNS-01 Challenge Debugging:**\n")
				suggestions.WriteString("- Verify DNS provider credentials are correct\n")
				suggestions.WriteString("- Check that the zone ID/domain is correct\n")
				suggestions.WriteString("- Test: `dig +short TXT _acme-challenge.<domain>`\n")
			}
		}
	}

	return suggestions.String()
}

// Helper functions
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt32(m map[string]interface{}, key string) int32 {
	if v, ok := m[key].(int64); ok {
		return int32(v)
	}
	return 0
}

func getAnnotation(obj *unstructured.Unstructured, key string) string {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		return ""
	}
	return annotations[key]
}

func getStatusIcon(status string) string {
	if status == "True" {
		return "✅"
	}
	return "❌"
}
