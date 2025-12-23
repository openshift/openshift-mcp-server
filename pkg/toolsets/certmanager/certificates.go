package certmanager

import (
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/certmanager"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/containers/kubernetes-mcp-server/pkg/output"
)

func initCertificates() []api.ServerTool {
	return []api.ServerTool{
		// certificates_list
		{
			Tool: api.Tool{
				Name:        "certmanager_certificates_list",
				Description: "List all cert-manager Certificates in the cluster. Returns certificates with their status including ready state, expiry time, and issuer reference.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Optional namespace to list certificates from. If not provided, lists from all namespaces",
						},
						"labelSelector": {
							Type:        "string",
							Description: "Optional label selector to filter certificates (e.g., 'app=myapp')",
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Cert-Manager: List Certificates",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: certificatesList,
		},
		// certificates_get
		{
			Tool: api.Tool{
				Name:        "certmanager_certificate_get",
				Description: "Get a specific cert-manager Certificate with its full status, conditions, expiry time, and renewal information.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Namespace of the Certificate",
						},
						"name": {
							Type:        "string",
							Description: "Name of the Certificate",
						},
					},
					Required: []string{"name", "namespace"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Cert-Manager: Get Certificate",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: certificateGet,
		},
		// certificaterequests_list
		{
			Tool: api.Tool{
				Name:        "certmanager_certificaterequests_list",
				Description: "List CertificateRequests in the cluster. CertificateRequests represent a request for a certificate from an issuer. Useful for debugging certificate issuance.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Optional namespace to list CertificateRequests from. If not provided, lists from all namespaces",
						},
						"certificateName": {
							Type:        "string",
							Description: "Optional: filter CertificateRequests for a specific Certificate name",
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Cert-Manager: List CertificateRequests",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: certificateRequestsList,
		},
		// orders_list
		{
			Tool: api.Tool{
				Name:        "certmanager_orders_list",
				Description: "List ACME Orders in the cluster. Orders represent an ACME certificate order and are created when using ACME issuers (like Let's Encrypt).",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Optional namespace to list Orders from. If not provided, lists from all namespaces",
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Cert-Manager: List ACME Orders",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: ordersList,
		},
		// challenges_list
		{
			Tool: api.Tool{
				Name:        "certmanager_challenges_list",
				Description: "List ACME Challenges in the cluster. Challenges represent domain validation challenges (HTTP-01 or DNS-01) and are created during ACME certificate issuance.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Optional namespace to list Challenges from. If not provided, lists from all namespaces",
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Cert-Manager: List ACME Challenges",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: challengesList,
		},
		// certificate_renew
		{
			Tool: api.Tool{
				Name:        "certmanager_certificate_renew",
				Description: "Trigger renewal of a cert-manager Certificate by deleting its Secret. Cert-manager will automatically detect the missing Secret and issue a new certificate.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Namespace of the Certificate",
						},
						"name": {
							Type:        "string",
							Description: "Name of the Certificate to renew",
						},
					},
					Required: []string{"name", "namespace"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Cert-Manager: Renew Certificate",
					ReadOnlyHint:    ptr.To(false),
					DestructiveHint: ptr.To(true),
					IdempotentHint:  ptr.To(true),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: certificateRenew,
		},
	}
}

func certificatesList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := ""
	if ns := params.GetArguments()["namespace"]; ns != nil {
		namespace = ns.(string)
	}

	listOptions := kubernetes.ResourceListOptions{
		AsTable: params.ListOutput.AsTable(),
	}

	if labelSelector := params.GetArguments()["labelSelector"]; labelSelector != nil {
		listOptions.LabelSelector = labelSelector.(string)
	}

	ret, err := params.ResourcesList(params.Context, &certmanager.CertificateGVK, namespace, listOptions)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list certificates: %v", err)), nil
	}
	return api.NewToolCallResult(params.ListOutput.PrintObj(ret)), nil
}

func certificateGet(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := params.GetArguments()["namespace"].(string)
	name := params.GetArguments()["name"].(string)

	ret, err := params.ResourcesGet(params.Context, &certmanager.CertificateGVK, namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get certificate %s/%s: %v", namespace, name, err)), nil
	}
	return api.NewToolCallResult(output.MarshalYaml(ret)), nil
}

func certificateRequestsList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := ""
	if ns := params.GetArguments()["namespace"]; ns != nil {
		namespace = ns.(string)
	}

	listOptions := kubernetes.ResourceListOptions{
		AsTable: params.ListOutput.AsTable(),
	}

	// Filter by certificate name if provided
	if certName := params.GetArguments()["certificateName"]; certName != nil {
		listOptions.LabelSelector = fmt.Sprintf("%s=%s", certmanager.LabelCertificateName, certName.(string))
	}

	ret, err := params.ResourcesList(params.Context, &certmanager.CertificateRequestGVK, namespace, listOptions)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list certificate requests: %v", err)), nil
	}
	return api.NewToolCallResult(params.ListOutput.PrintObj(ret)), nil
}

func ordersList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := ""
	if ns := params.GetArguments()["namespace"]; ns != nil {
		namespace = ns.(string)
	}

	listOptions := kubernetes.ResourceListOptions{
		AsTable: params.ListOutput.AsTable(),
	}

	ret, err := params.ResourcesList(params.Context, &certmanager.OrderGVK, namespace, listOptions)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list ACME orders: %v", err)), nil
	}
	return api.NewToolCallResult(params.ListOutput.PrintObj(ret)), nil
}

func challengesList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := ""
	if ns := params.GetArguments()["namespace"]; ns != nil {
		namespace = ns.(string)
	}

	listOptions := kubernetes.ResourceListOptions{
		AsTable: params.ListOutput.AsTable(),
	}

	ret, err := params.ResourcesList(params.Context, &certmanager.ChallengeGVK, namespace, listOptions)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list ACME challenges: %v", err)), nil
	}
	return api.NewToolCallResult(params.ListOutput.PrintObj(ret)), nil
}

func certificateRenew(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := params.GetArguments()["namespace"].(string)
	name := params.GetArguments()["name"].(string)

	// Get the certificate to find the secret name
	cert, err := params.ResourcesGet(params.Context, &certmanager.CertificateGVK, namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get certificate %s/%s: %v", namespace, name, err)), nil
	}

	// Get the secret name from the certificate spec
	secretName, found, _ := unstructured.NestedString(cert.Object, "spec", "secretName")
	if !found || secretName == "" {
		return api.NewToolCallResult("", fmt.Errorf("certificate %s/%s has no secretName specified", namespace, name)), nil
	}

	// Delete the secret to trigger renewal
	err = params.ResourcesDelete(params.Context, &certmanager.SecretGVK, namespace, secretName)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete secret %s/%s to trigger renewal: %v", namespace, secretName, err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("✅ Deleted Secret '%s/%s' to trigger certificate renewal.\n\nCert-manager will now detect the missing Secret and issue a new certificate.\nUse `certmanager_certificate_get` to monitor the renewal progress.", namespace, secretName), nil), nil
}
