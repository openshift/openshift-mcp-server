package core

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"
)

// initSecureWorkloads returns all tools for the secure workloads toolset.
func initSecureWorkloads() []api.ServerTool {
	return []api.ServerTool{
		{Tool: api.Tool{
			Name:        "secrets_management_configure",
			Description: "Generates a plan to install and configure components for external secrets management (ESO or SSCSI). This tool is designed to work with the secure, provider-side setup created by the 'generate_prerequisites_plan' tool.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"management_type": {
						Type:        "string",
						Description: "The secret management component to install. (Enum: eso, sscsi)",
						Enum:        []interface{}{"eso", "sscsi"},
					},
					"provider": {
						Type:        "string",
						Description: "The external secrets provider to connect to. (Enum: gcp, aws, vault)",
						Enum:        []interface{}{"gcp", "aws", "vault"},
					},
					"namespace": {
						Type:        "string",
						Description: "The target namespace for namespaced resources.",
					},
					"service_account_name": {
						Type:        "string",
						Description: "The name of the Kubernetes Service Account that has been granted provider-side permissions.",
					},
					"aws_region": {
						Type:        "string",
						Description: "Required for AWS. The AWS region where your secrets are stored.",
					},
					"vault_addr": {
						Type:        "string",
						Description: "Required for Vault. The address of your Vault server (e.g., 'https://vault.example.com').",
					},
					"vault_role": {
						Type:        "string",
						Description: "Required for Vault. The Vault role that is bound to your service account.",
					},
				},
				Required: []string{"management_type", "provider", "namespace", "service_account_name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "Secure Workloads: Configure Secrets Management",
				ReadOnlyHint: ptr.To(true),
			},
		},
			Handler: secretsManagementConfigureHandler,
		},
		{
			Tool: api.Tool{
				Name:        "secrets_management_debug",
				Description: "Diagnoses secrets management issues (ESO or SSCSI). It first generates a read-only plan of checks. Upon user confirmation, it can be re-run with 'execute_audit: true' to perform the audit and generate a report.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"resource_type": {
							Type:        "string",
							Description: "The type of resource to debug. (Enum: externalsecret, pod)",
							Enum:        []interface{}{"externalsecret", "pod"},
						},
						"resource_name": {
							Type:        "string",
							Description: "The name of the 'ExternalSecret' or 'Pod' that is failing.",
						},
						"namespace": {
							Type:        "string",
							Description: "The namespace where the resource is located.",
						},
						"execute_audit": {
							Type:        "boolean",
							Description: "If true, executes the audit and provides a report. If false (default), generates a manual plan of commands.",
							Default:     json.RawMessage("false"),
						},
					},
					Required: []string{"resource_type", "resource_name", "namespace"},
				},
				Annotations: api.ToolAnnotations{
					Title:        "Secure Workloads: Debug Secrets Management",
					ReadOnlyHint: ptr.To(true),
				},
			},
			Handler: secretsManagementDebugHandler,
		},
		{
			Tool: api.Tool{
				Name:        "recommend_secrets_management",
				Description: "Helps you choose the best secrets management approach (native Kubernetes Secret, ESO, or SSCSI) based on your requirements.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"is_secret_external": {
							Type:        "boolean",
							Description: "Is the secret you need to use stored in a specialized secrets management system like AWS Secrets Manager, GCP Secret Manager, or HashiCorp Vault? Answer 'false' if you intend to store the secret directly within a standard Kubernetes Secret object.",
						},
						"prefers_native_secrets": {
							Type:        "boolean",
							Description: "Only required if the secret is external. Does your application expect secrets to be available as native Kubernetes 'Secret' objects?",
						},
						"avoid_etcd_storage": {
							Type:        "boolean",
							Description: "Only required if the secret is external. Is minimizing the storage of secrets within the cluster's database (etcd) a primary security concern?",
						},
						"can_mount_volumes": {
							Type:        "boolean",
							Description: "Only required if the secret is external. Are you able to modify your application's deployment to mount secrets as volumes?",
						},
					},
					Required: []string{"is_secret_external"},
				},
				Annotations: api.ToolAnnotations{
					Title:        "Secure Workloads: Recommend Secrets Management Tool",
					ReadOnlyHint: ptr.To(true),
				},
			},
			Handler: recommendSecretsManagementHandler,
		},
		{
			Tool: api.Tool{
				Name:        "generate_example_external_secret",
				Description: "Generates an example 'ExternalSecret' manifest to fetch a secret from a configured 'SecretStore' (ESO).",
				InputSchema: nil,
				Annotations: api.ToolAnnotations{
					Title:        "Secure Workloads: Generate Example ExternalSecret (ESO)",
					ReadOnlyHint: ptr.To(true),
				},
			},
			Handler: generateExampleExternalSecretHandler,
		},
		{
			Tool: api.Tool{
				Name:        "generate_example_pod_with_csi_volume",
				Description: "Generates an example 'Pod' manifest that mounts a secret as a volume using a 'SecretProviderClass' (SSCSI).",
				InputSchema: nil,
				Annotations: api.ToolAnnotations{
					Title:        "Secure Workloads: Generate Example Pod with CSI Volume (SSCSI)",
					ReadOnlyHint: ptr.To(true),
				},
			},
			Handler: generateExamplePodWithCSIVolumeHandler,
		},
		{
			Tool: api.Tool{
				Name:        "generate_prerequisites_plan",
				Description: "Generates a checklist of provider-side prerequisites (IAM roles, policies) required before using secrets management tools like ESO or SSCSI.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"provider": {
							Type:        "string",
							Description: "The external secrets provider to generate a plan for.",
							Enum:        []interface{}{"gcp", "aws", "vault"},
						},
					},
					Required: []string{"provider"},
				},
				Annotations: api.ToolAnnotations{
					Title:        "Secure Workloads: Generate Provider Prerequisites Plan",
					ReadOnlyHint: ptr.To(true),
				},
			},
			Handler: generatePrerequisitesPlanHandler,
		},
	}
}

func generatePrerequisitesPlanHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	provider := args["provider"].(string)

	var result strings.Builder
	result.WriteString(fmt.Sprintf("### Prerequisites Checklist for %s\n\n", strings.ToUpper(provider)))
	result.WriteString("This plan outlines the necessary permissions and configuration on the provider side. You must complete these steps before the in-cluster components (ESO or SSCSI) can access secrets.\n")

	switch provider {
	case "gcp":
		result.WriteString(`
#### 1. Create a GCP Service Account (SA)
- Navigate to **IAM & Admin > Service Accounts** in the Google Cloud Console.
- Click **+ CREATE SERVICE ACCOUNT**.
- Give it a name (e.g., ` + "`k8s-secret-accessor`" + `) and a description.

#### 2. Grant IAM Permissions
- Find the Service Account you just created in the IAM list.
- Grant it the **` + "`Secret Manager Secret Accessor`" + `** role. This allows it to read secret values.
- **Best Practice**: For higher security, grant this role only on the specific secrets it needs to access, not the entire project.

#### 3. Create and Download a JSON Key
- Go to the **Keys** tab for your Service Account.
- Click **ADD KEY > Create new key**.
- Select **JSON** as the key type and click **CREATE**.
- A JSON file will be downloaded to your computer. **This file contains sensitive credentials.**
- You will use this file with the ` + "`secrets_management_configure`" + ` tool's ` + "`credentials_file`" + ` parameter.
`)
	case "aws":
		result.WriteString(`
This plan outlines the recommended approach using **IAM Roles for Service Accounts (IRSA)**, which provides passwordless authentication.

#### 1. Find your Cluster's OIDC Provider URL
- Run this command to get the URL. You will need it in the next step.
` + "```bash\noc get authentication.config.openshift.io cluster -o jsonpath='{.spec.serviceAccountIssuer}'\n```" + `

#### 2. Create an OIDC Identity Provider in AWS IAM
- In the AWS IAM Console, go to **Identity providers**.
- Click **Add provider**.
- For **Provider type**, select **OpenID Connect**.
- For **Provider URL**, paste the URL from the previous step.
- For **Audience**, enter ` + "`openshift`" + ` and click **Get thumbprint**.

#### 3. Create an IAM Policy
- Go to **Policies** and click **Create policy**.
- In the JSON editor, paste the following policy. This grants read-only access to Secrets Manager.
` + "```json\n{\n    \"Version\": \"2012-10-17\",\n    \"Statement\": [\n        {\n            \"Effect\": \"Allow\",\n            \"Action\": \"secretsmanager:GetSecretValue\",\n            \"Resource\": \"arn:aws:secretsmanager:REGION:ACCOUNT_ID:secret:YOUR_SECRET_NAME-*\"\n        },\n        {\n            \"Effect\": \"Allow\",\n            \"Action\": \"kms:Decrypt\",\n            \"Resource\": \"arn:aws:kms:REGION:ACCOUNT_ID:key/YOUR_KMS_KEY_ID\"\n        }\n    ]\n}\n```" + `
- **Important**: Replace ` + "`REGION`" + `, ` + "`ACCOUNT_ID`" + `, ` + "`YOUR_SECRET_NAME-*`" + `, and ` + "`YOUR_KMS_KEY_ID`" + ` with your specific values.

#### 4. Create an IAM Role
- Go to **Roles** and click **Create role**.
- For **Trusted entity type**, select **Web identity**.
- For **Identity provider**, choose the OIDC provider you created.
- For **Audience**, select ` + "`openshift`" + `.
- Attach the IAM policy you created in the previous step.
- Give the role a name (e.g., ` + "`k8s-secret-accessor-role`" + `).

#### 5. Annotate your Kubernetes Service Account
- The final step is to link the IAM Role to the Kubernetes Service Account that your pods (or the ESO controller) will use.
- Run this command, replacing the role ARN and service account name:
` + "```bash\noc annotate sa your-service-account -n your-namespace eks.amazonaws.com/role-arn=arn:aws:iam::ACCOUNT_ID:role/k8s-secret-accessor-role\n```" + `
- Now, any pod using ` + "`your-service-account`" + ` can securely access the allowed secrets without needing stored credentials.
`)
	case "vault":
		result.WriteString(`
This plan outlines how to configure the Vault Kubernetes authentication method.

#### 1. Enable and Configure Kubernetes Auth in Vault
- First, enable the auth method. This is a one-time setup.
` + "```bash\nvault auth enable kubernetes\n```" + `
- Then, configure it to connect to your Kubernetes cluster.
` + "```bash\nvault write auth/kubernetes/config \\\n    kubernetes_host=\"https://your-k8s-api-server:6443\" \\\n    kubernetes_ca_cert=@/path/to/ca.crt\n```" + `
- **Note**: If Vault runs inside the same cluster, it can automatically discover these settings.

#### 2. Create a Vault Policy
- Policies define what a user is allowed to access. Create a policy file (e.g., ` + "`webapp-policy.hcl`" + `) that grants read access to your application's secrets.
` + "```hcl\npath \"secret/data/webapp/config\" {\n    capabilities = [\"read\"]\n}\n```" + `
- Write this policy to Vault:
` + "```bash\nvault policy write webapp-policy webapp-policy.hcl\n```" + `

#### 3. Create a Vault Role
- A role connects a Kubernetes Service Account to a Vault policy.
- This command creates a role named ` + "`webapp`" + ` that links the ` + "`default`" + ` service account in the ` + "`default`" + ` namespace to the ` + "`webapp-policy`" + `.
` + "```bash\nvault write auth/kubernetes/role/webapp \\\n    bound_service_account_names=default \\\n    bound_service_account_namespaces=default \\\n    policies=webapp-policy \\\n    ttl=24h\n```" + `
- Now, any pod running with the ` + "`default`" + ` service account can authenticate with Vault and will receive a token with the ` + "`webapp-policy`" + ` permissions.
`)
	}

	return api.NewToolCallResult(result.String(), nil), nil
}

func generateExamplePodWithCSIVolumeHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	const (
		namespace = "default"
		podName   = "my-app-pod"
	)

	var result strings.Builder
	result.WriteString("This tool provides complete examples for using SSCSI with different providers. Choose the section that matches your secret provider and apply both the `SecretProviderClass` and `Pod` manifests.\n")

	providerExamples := []struct {
		providerName string
		spcResource  unstructured.Unstructured
	}{
		{
			providerName: "GCP",
			spcResource: unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "secrets-store.csi.x-k8s.io/v1",
					"kind":       "SecretProviderClass",
					"metadata":   map[string]interface{}{"name": "gcp-provider-spc", "namespace": namespace},
					"spec": map[string]interface{}{
						"provider": "gcp",
						"parameters": map[string]interface{}{
							"secrets": "- resourceName: \"projects/your-gcp-project-id/secrets/your-secret-name/versions/latest\"\n  fileName: \"secret.txt\"\n",
						},
					},
				},
			},
		},
		{
			providerName: "AWS",
			spcResource: unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "secrets-store.csi.x-k8s.io/v1",
					"kind":       "SecretProviderClass",
					"metadata":   map[string]interface{}{"name": "aws-provider-spc", "namespace": namespace},
					"spec": map[string]interface{}{
						"provider": "aws",
						"parameters": map[string]interface{}{
							"objects": "- objectName: \"YourSecretNameFromSecretsManager\"\n  objectType: \"secretsmanager\"\n",
						},
					},
				},
			},
		},
		{
			providerName: "Vault",
			spcResource: unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "secrets-store.csi.x-k8s.io/v1",
					"kind":       "SecretProviderClass",
					"metadata":   map[string]interface{}{"name": "vault-provider-spc", "namespace": namespace},
					"spec": map[string]interface{}{
						"provider": "vault",
						"parameters": map[string]interface{}{
							"roleName":     "your-app-role",
							"vaultAddress": "https://vault.your-domain.com",
							"objects":      "- objectName: \"db-password\"\n  secretPath: \"secret/data/webapp/config\"\n  secretKey:  \"password\"\n",
						},
					},
				},
			},
		},
	}

	for _, example := range providerExamples {
		podResource := createPodResource(podName, namespace, example.spcResource.GetName())
		combinedYaml, err := generateCombinedYaml(example.spcResource, podResource)
		if err != nil {
			return nil, err
		}
		result.WriteString(fmt.Sprintf("\n### --- %s Provider Example ---\n\n```yaml\n", example.providerName))
		result.Write(combinedYaml)
		result.WriteString("```\n")
	}

	return api.NewToolCallResult(result.String(), nil), nil
}

func createPodResource(podName, namespace, spcName string) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      podName,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"containers": []interface{}{
					map[string]interface{}{
						"name":  "my-app",
						"image": "nginx:latest",
						"volumeMounts": []interface{}{
							map[string]interface{}{
								"name":      "secrets-store-inline",
								"mountPath": "/mnt/secrets-store",
								"readOnly":  true,
							},
						},
					},
				},
				"volumes": []interface{}{
					map[string]interface{}{
						"name": "secrets-store-inline",
						"csi": map[string]interface{}{
							"driver":           "secrets-store.csi.k8s.io",
							"readOnly":         true,
							"volumeAttributes": map[string]interface{}{"secretProviderClass": spcName},
						},
					},
				},
			},
		},
	}
}

func generateCombinedYaml(spc, pod unstructured.Unstructured) ([]byte, error) {
	spcYaml, err := yaml.Marshal(spc.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secretproviderclass to yaml: %w", err)
	}

	podYaml, err := yaml.Marshal(pod.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pod to yaml: %w", err)
	}

	return []byte(string(spcYaml) + "\n---\n" + string(podYaml)), nil
}

func generateExampleExternalSecretHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	const (
		storeName          = "my-secret-store"
		namespace          = "default"
		externalSecretName = "my-external-secret"
		targetSecretName   = "my-k8s-secret"
	)

	var result strings.Builder
	result.WriteString(fmt.Sprintf("This is an example `ExternalSecret` that uses the '%s' `SecretStore` to create a native Kubernetes `Secret`. Apply this manifest to your cluster.\n\n", storeName))

	esResource := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "external-secrets.io/v1beta1",
			"kind":       "ExternalSecret",
			"metadata": map[string]interface{}{
				"name":      fmt.Sprintf("%s-es", targetSecretName),
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"refreshInterval": "1h",
				"secretStoreRef": map[string]interface{}{
					"name": storeName,
					"kind": "SecretStore", // or ClusterSecretStore
				},
				"target": map[string]interface{}{
					"name": targetSecretName,
				},
				"data": []interface{}{
					map[string]interface{}{
						"secretKey": "secret-value", // The key in the resulting k8s secret
						"remoteRef": map[string]interface{}{
							"key": externalSecretName, // The name of the secret in the external provider
						},
					},
				},
			},
		},
	}

	resourceYaml, err := yaml.Marshal(esResource.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal externalsecret to yaml: %w", err)
	}

	result.WriteString("```yaml\n")
	result.Write(resourceYaml)
	result.WriteString("```\n")

	return api.NewToolCallResult(result.String(), nil), nil
}

func recommendSecretsManagementHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	isExternal, isExternalProvided := args["is_secret_external"].(bool)
	if !isExternalProvided {
		return nil, fmt.Errorf("parameter 'is_secret_external' is required")
	}

	var result strings.Builder

	if !isExternal {
		result.WriteString("### Recommendation: Use a Native Kubernetes Secret\n\n")
		result.WriteString("For secrets managed within the cluster, the standard Kubernetes `Secret` object is the recommended approach. It's simple, secure, and tightly integrated with the OpenShift/Kubernetes ecosystem.\n\n")
		result.WriteString("#### Example `Secret` Manifest\n\n")
		result.WriteString("You can create a secret using a YAML manifest like this. Remember that the `data` values must be base64-encoded.\n\n")

		secretYAML := `apiVersion: v1
kind: Secret
metadata:
  name: my-internal-secret
type: Opaque
data:
  # Values must be base64 encoded.
  # Example: echo -n 'my-password' | base64
  username: dXNlcg== # "user"
  password: bXktcGFzc3dvcmQ= # "my-password"
`
		result.WriteString("```yaml\n")
		result.WriteString(secretYAML)
		result.WriteString("```\n\n")
		result.WriteString("Apply this with `oc apply -f <filename>.yaml`. You can then mount this secret into your pods as environment variables or files.\n")
	} else {
		// External secret logic
		prefersNativeArg, prefersNativeProvided := args["prefers_native_secrets"]
		avoidEtcdArg, avoidEtcdProvided := args["avoid_etcd_storage"]
		canMountArg, canMountProvided := args["can_mount_volumes"]

		if !prefersNativeProvided || !avoidEtcdProvided || !canMountProvided {
			return nil, fmt.Errorf("when 'is_secret_external' is true, the following boolean parameters are required: 'prefers_native_secrets', 'avoid_etcd_storage', 'can_mount_volumes'")
		}

		prefersNative, _ := prefersNativeArg.(bool)
		avoidEtcd, _ := avoidEtcdArg.(bool)
		canMount, _ := canMountArg.(bool)

		// Determine the best fit based on the user's answers
		var recommendation string
		var explanation string

		// Scoring mechanism to decide between ESO and SSCSI
		esoScore := 0
		sscsiScore := 0

		if prefersNative {
			esoScore++
		}
		if avoidEtcd {
			sscsiScore++
		}
		if canMount {
			sscsiScore++
		} else {
			esoScore++ // If mounting volumes is not an option, ESO is the only choice.
		}

		if sscsiScore > esoScore {
			recommendation = "Secrets Store CSI Driver (SSCSI)"
			explanation = "This is the best fit because you prioritize keeping secrets out of etcd and your application can mount secrets as volumes. SSCSI mounts external secrets directly into the pod's filesystem as in-memory volumes, which is highly secure."
		} else {
			recommendation = "External Secrets Operator (ESO)"
			explanation = "This is the best fit because your application relies on native Kubernetes `Secret` objects or cannot be modified to mount volumes. ESO synchronizes secrets from your external provider into native Kubernetes `Secret` resources."
		}

		result.WriteString(fmt.Sprintf("### Recommendation: Use %s\n\n", recommendation))
		result.WriteString(explanation)
	}

	return &api.ToolCallResult{
		Content: result.String(),
	}, nil
}

func secretsManagementConfigureHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	managementType := args["management_type"].(string)
	providerType := args["provider"].(string)
	namespace := args["namespace"].(string)
	serviceAccountName := args["service_account_name"].(string)

	var result strings.Builder
	result.WriteString("This plan will configure external secrets management. Ask the user for confirmation before applying the resources and commands.\n\n")

	// Step 1: Operator Installation from OperatorHub
	result.WriteString("### Step 1: Install the Operator\n\n")
	var operatorName string
	if managementType == "eso" {
		operatorName = "External Secrets Operator"
	} else { // sscsi
		operatorName = "Secrets Store CSI Driver Operator"
	}
	result.WriteString(fmt.Sprintf("Please install the '%s' from the OperatorHub (or **Ecosystem > Software Catalog** in newer OpenShift versions) in your OpenShift console.\n", operatorName))
	result.WriteString(fmt.Sprintf("Navigate to **Operators > OperatorHub**, search for **%s**, and follow the installation instructions.\n\n", operatorName))

	// Step 2: Create a SecretStore/ClusterSecretStore
	result.WriteString("### Step 2: Create the SecretStore\n\n")

	if managementType == "eso" {
		// Step 2: Create configuration resource for ESO
		result.WriteString("This manifest configures the secret management component to connect to your provider using the recommended service account authentication method.\n")

		var providerSpec map[string]interface{}
		switch providerType {
		case "gcp":
			providerSpec = map[string]interface{}{
				"auth": map[string]interface{}{
					"workloadIdentity": map[string]interface{}{
						"serviceAccountRef": map[string]interface{}{
							"name": serviceAccountName,
						},
					},
				},
			}
		case "aws":
			awsRegion, ok := args["aws_region"].(string)
			if !ok || awsRegion == "" {
				return nil, fmt.Errorf("parameter 'aws_region' is required for AWS provider")
			}
			providerSpec = map[string]interface{}{
				"region": awsRegion,
				"auth": map[string]interface{}{
					"jwt": map[string]interface{}{
						"serviceAccountRef": map[string]interface{}{
							"name": serviceAccountName,
						},
					},
				},
			}
		case "vault":
			vaultAddr, ok := args["vault_addr"].(string)
			if !ok || vaultAddr == "" {
				return nil, fmt.Errorf("parameter 'vault_addr' is required for Vault provider")
			}
			vaultRole, ok := args["vault_role"].(string)
			if !ok || vaultRole == "" {
				return nil, fmt.Errorf("parameter 'vault_role' is required for Vault provider")
			}
			providerSpec = map[string]interface{}{
				"server":  vaultAddr,
				"path":    "secret",
				"version": "v2",
				"auth": map[string]interface{}{
					"kubernetes": map[string]interface{}{
						"mountPath": "kubernetes",
						"role":      vaultRole,
						"serviceAccountRef": map[string]interface{}{
							"name": serviceAccountName,
						},
					},
				},
			}
		default:
			return nil, fmt.Errorf("provider '%s' is not supported for automated configuration", providerType)
		}

		configResource := unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "external-secrets.io/v1beta1",
				"kind":       "SecretStore",
				"metadata": map[string]interface{}{
					"name":      fmt.Sprintf("%s-secret-store", providerType),
					"namespace": namespace,
				},
				"spec": map[string]interface{}{
					"provider": map[string]interface{}{
						providerType: providerSpec,
					},
				},
			},
		}
		resourceYaml, err := yaml.Marshal(configResource.Object)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal resource to yaml: %w", err)
		}
		result.WriteString("```yaml\n")
		result.Write(resourceYaml)
		result.WriteString("```\n")

	} else { // sscsi
		result.WriteString("### Step 1: Install the Secrets Store CSI Driver Operator\n\n")
		result.WriteString("Please install the 'Secrets Store CSI Driver' from the OperatorHub (or **Ecosystem > Software Catalog** in newer OpenShift versions) in your OpenShift console.\n")
		result.WriteString("Navigate to **Operators > OperatorHub**, search for **Secrets Store CSI Driver**, and follow the installation instructions.\n\n")

		// Step 2: Install the Provider-Specific Driver
		result.WriteString("### Step 2: Install the Provider-Specific Driver\n\n")
		result.WriteString("The Secrets Store CSI Driver requires a provider-specific component to be installed in your cluster. Choose the command for your provider.\n\n")

		switch providerType {
		case "gcp":
			result.WriteString("For GCP, apply the following manifest to deploy the provider daemonset:\n")
			result.WriteString("```yaml\n" + gcpCSIDriverProviderYAML + "```\n")
		case "aws":
			result.WriteString("For AWS, add the Helm repo and install the provider chart:\n")
			result.WriteString("```bash\n")
			result.WriteString("helm repo add aws-secrets-store-csi-driver https://aws.github.io/secrets-store-csi-driver-provider-aws\n")
			result.WriteString("helm repo update\n")
			result.WriteString("helm install csi-secrets-store-provider-aws aws-secrets-store-csi-driver/csi-secrets-store-provider-aws --namespace kube-system\n")
			result.WriteString("```\n")
		case "vault":
			result.WriteString("For Vault, add the Helm repo and install the provider chart:\n")
			result.WriteString("```bash\n")
			result.WriteString("helm repo add hashicorp https://helm.releases.hashicorp.com\n")
			result.WriteString("helm repo update\n")
			result.WriteString("helm install csi-secrets-store-provider-vault hashicorp/csi-secrets-store-provider-vault --namespace kube-system\n")
			result.WriteString("```\n")
		}
		result.WriteString("\n")

		result.WriteString("### Step 3: Create Configuration Resource (SecretProviderClass)\n")
		result.WriteString("This manifest defines how the CSI driver should fetch secrets from your provider.\n")

		var spcParams map[string]interface{}
		switch providerType {
		case "gcp":
			spcParams = map[string]interface{}{
				"secrets": "- resourceName: \"projects/your-gcp-project-id/secrets/your-secret-name/versions/latest\"\n  fileName: \"secret.txt\"\n",
			}
		case "aws":
			spcParams = map[string]interface{}{
				"objects": "- objectName: \"YourSecretNameFromSecretsManager\"\n  objectType: \"secretsmanager\"\n",
			}
		case "vault":
			vaultAddr, _ := args["vault_addr"].(string)
			vaultRole, _ := args["vault_role"].(string)
			if vaultAddr == "" || vaultRole == "" {
				return nil, fmt.Errorf("parameters 'vault_addr' and 'vault_role' are required for Vault provider with SSCSI")
			}
			spcParams = map[string]interface{}{
				"roleName":     vaultRole,
				"vaultAddress": vaultAddr,
				"objects":      "- objectName: \"db-password\"\n  secretPath: \"secret/data/webapp/config\"\n  secretKey:  \"password\"\n",
			}
		}

		configResource := unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "secrets-store.csi.x-k8s.io/v1",
				"kind":       "SecretProviderClass",
				"metadata": map[string]interface{}{
					"name":      fmt.Sprintf("%s-secret-provider", providerType),
					"namespace": namespace,
				},
				"spec": map[string]interface{}{
					"provider":   providerType,
					"parameters": spcParams,
				},
			},
		}
		resourceYaml, err := yaml.Marshal(configResource.Object)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal resource to yaml: %w", err)
		}
		result.WriteString("```yaml\n")
		result.Write(resourceYaml)
		result.WriteString("```\n")
	}

	return api.NewToolCallResult(result.String(), nil), nil
}

const gcpCSIDriverProviderYAML = `---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: csi-secrets-store-provider-gcp
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: csi-secrets-store-provider-gcp-binding
subjects:
  - kind: ServiceAccount
    name: csi-secrets-store-provider-gcp
    namespace: kube-system
roleRef:
  kind: ClusterRole
  name: system:auth-delegator
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: csi-secrets-store-provider-gcp
  namespace: kube-system
spec:
  updateStrategy:
    type: RollingUpdate
  selector:
    matchLabels:
      app: csi-secrets-store-provider-gcp
  template:
    metadata:
      labels:
        app: csi-secrets-store-provider-gcp
    spec:
      serviceAccountName: csi-secrets-store-provider-gcp
      hostNetwork: true
      containers:
        - name: provider
          image: gcr.io/google.com/cloud-sdk-gke-secrets-store-csi-driver/provider-gcp:v1.3.1
          imagePullPolicy: IfNotPresent
          resources:
            requests:
              cpu: 50m
              memory: 100Mi
            limits:
              cpu: 50m
              memory: 100Mi
          volumeMounts:
            - name: mountpoint-dir
              mountPath: /var/lib/kubelet/pods
              mountPropagation: HostToContainer
      volumes:
        - name: mountpoint-dir
          hostPath:
            path: /var/lib/kubelet/pods
            type: DirectoryOrCreate
      nodeSelector:
        kubernetes.io/os: linux
`

type auditCheck struct {
	ID             string
	Description    string
	Status         string // PASS, FAIL, WARN
	Message        string
	Recommendation string
}

func newPassCheck(id, description, message string) auditCheck {
	return auditCheck{ID: id, Description: description, Status: "PASS", Message: message}
}

func newFailCheck(id, description, message, recommendation string) auditCheck {
	return auditCheck{ID: id, Description: description, Status: "FAIL", Message: message, Recommendation: recommendation}
}

func newWarnCheck(id, description, message, recommendation string) auditCheck {
	return auditCheck{ID: id, Description: description, Status: "WARN", Message: message, Recommendation: recommendation}
}

func secretsManagementDebugHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	namespace := args["namespace"].(string)
	resourceType := args["resource_type"].(string)
	resourceName := args["resource_name"].(string)

	executeAudit := false
	if val, ok := args["execute_audit"].(bool); ok {
		executeAudit = val
	}

	if executeAudit {
		// --- This is the AUTOMATED AUDIT logic ---
		var result strings.Builder
		result.WriteString(fmt.Sprintf("### Secret Management Audit Report for %s/%s\n\n", resourceType, resourceName))

		var checks []auditCheck
		var err error

		switch resourceType {
		case "externalsecret":
			checks, err = runESOAudit(params, namespace, resourceName)
		case "pod":
			checks, err = runSSCSIAudit(params, namespace, resourceName)
		default:
			return nil, fmt.Errorf("auditing resource type '%s' is not supported. Please provide 'pod' or 'externalsecret'", resourceType)
		}

		if err != nil {
			return nil, fmt.Errorf("error during audit: %w", err)
		}

		// Format the results
		for _, check := range checks {
			var statusIcon string
			switch check.Status {
			case "PASS":
				statusIcon = "✅"
			case "FAIL":
				statusIcon = "❌"
			case "WARN":
				statusIcon = "⚠️"
			}
			result.WriteString(fmt.Sprintf("- %s **[%s]** %s\n", statusIcon, check.Status, check.Description))
			if check.Message != "" {
				result.WriteString(fmt.Sprintf("  - **Details**: %s\n", check.Message))
			}
			if check.Recommendation != "" {
				result.WriteString(fmt.Sprintf("  - **Recommendation**: %s\n", check.Recommendation))
			}
			result.WriteString("\n")
		}

		return api.NewToolCallResult(result.String(), nil), nil

	} else {
		// --- This is the MANUAL PLAN logic ---
		var result strings.Builder
		result.WriteString(fmt.Sprintf("### Manual Debugging Plan for %s/%s\n\n", resourceType, resourceName))
		result.WriteString("Here are the steps I will take to diagnose the issue. Please confirm if you would like me to proceed with executing this audit.\n\n")

		switch resourceType {
		case "externalsecret":
			result.WriteString("#### Step 1: Check the `ExternalSecret` Status\n")
			result.WriteString("This command shows the status of the `ExternalSecret`, which often contains the direct error message from the provider.\n")
			result.WriteString("```bash\n")
			result.WriteString(fmt.Sprintf("oc describe externalsecret %s -n %s\n", resourceName, namespace))
			result.WriteString("```\n\n")

			result.WriteString("#### Step 2: Check the `SecretStore` Configuration and Status\n")
			result.WriteString("This command checks if the `SecretStore` is configured correctly and is ready to talk to the provider. The `ExternalSecret`'s YAML will tell you which store it is using.\n")
			result.WriteString("```bash\n")
			result.WriteString("# First, find the name of the SecretStore used by your ExternalSecret\n")
			result.WriteString(fmt.Sprintf("STORE_NAME=$(oc get externalsecret %s -n %s -o=jsonpath='{.spec.secretStoreRef.name}')\n", resourceName, namespace))
			result.WriteString("echo \"Using SecretStore: $STORE_NAME\"\n\n")
			result.WriteString("# Now, describe the SecretStore to check its status\n")
			result.WriteString(fmt.Sprintf("oc describe secretstore $STORE_NAME -n %s\n", namespace))
			result.WriteString("```\n\n")

			result.WriteString("#### Step 3: Check the External Secrets Operator Logs\n")
			result.WriteString("These logs show the activity of the controller and will contain detailed error information if the sync is failing.\n")
			result.WriteString("```bash\n")
			result.WriteString("oc logs -l app.kubernetes.io/name=external-secrets -n external-secrets\n")
			result.WriteString("```\n")

		case "pod":
			result.WriteString("#### Step 1: Check the Pod's Events\n")
			result.WriteString("This is the most critical step. Look for `FailedMount` events, as their messages usually contain the root cause of the problem.\n")
			result.WriteString("```bash\n")
			result.WriteString(fmt.Sprintf("oc describe pod %s -n %s\n", resourceName, namespace))
			result.WriteString("```\n\n")

			result.WriteString("#### Step 2: Verify the `SecretProviderClass`\n")
			result.WriteString("This command checks if the `SecretProviderClass` referenced by your pod exists and is configured correctly.\n")
			result.WriteString("```bash\n")
			result.WriteString("# First, find the name of the SecretProviderClass used by your Pod\n")
			result.WriteString(fmt.Sprintf("SPC_NAME=$(oc get pod %s -n %s -o=jsonpath='{.spec.volumes[?(@.csi.driver==\"secrets-store.csi.k8s.io\")].csi.volumeAttributes.secretProviderClass}')\n", resourceName, namespace))
			result.WriteString("echo \"Using SecretProviderClass: $SPC_NAME\"\n\n")
			result.WriteString("# Now, describe the SecretProviderClass\n")
			result.WriteString(fmt.Sprintf("oc describe secretproviderclass $SPC_NAME -n %s\n", namespace))
			result.WriteString("```\n\n")

			result.WriteString("#### Step 3: Check the CSI Driver and Provider Logs on the Node\n")
			result.WriteString("These logs show the activity of the CSI components on the specific node where your pod is scheduled. This is useful for diagnosing deeper authentication or provider-specific issues.\n")
			result.WriteString("```bash\n")
			result.WriteString(fmt.Sprintf("NODE_NAME=$(oc get pod %s -n %s -o=jsonpath='{.spec.nodeName}')\n", resourceName, namespace))
			result.WriteString("echo \"Pod is running on node: $NODE_NAME\"\n\n")
			result.WriteString("# Find the driver pod on that node\n")
			result.WriteString("DRIVER_POD=$(oc get pods -n kube-system --field-selector spec.nodeName=$NODE_NAME -l app=secrets-store-csi-driver -o name | head -n 1)\n")
			result.WriteString("echo \"Checking logs for driver: $DRIVER_POD\"\n")
			result.WriteString("oc logs $DRIVER_POD -n kube-system\n\n")
			result.WriteString("# Find a provider pod on that node (example for AWS, adjust if using another provider)\n")
			result.WriteString("PROVIDER_POD=$(oc get pods -n kube-system --field-selector spec.nodeName=$NODE_NAME -l app=csi-secrets-store-provider-aws -o name | head -n 1)\n")
			result.WriteString("echo \"Checking logs for provider: $PROVIDER_POD\"\n")
			result.WriteString("oc logs $PROVIDER_POD -n kube-system\n")
			result.WriteString("```\n")
		}
		return api.NewToolCallResult(result.String(), nil), nil
	}
}

// isReadyFromConditions robustly checks a slice of conditions for the 'Ready' type.
// It returns true if Ready==True, and false with a message otherwise.
func isReadyFromConditions(conditions []interface{}) (ready bool, message string) {
	for _, c := range conditions {
		cond, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		condType, _, _ := unstructured.NestedString(cond, "type")
		if condType == "Ready" {
			status, _, _ := unstructured.NestedString(cond, "status")
			if status == "True" {
				return true, ""
			}
			// If Ready is not True, get the message
			msg, _, _ := unstructured.NestedString(cond, "message")
			return false, msg
		}
	}
	// If no Ready condition is found at all.
	return false, "No 'Ready' condition found in resource status."
}

func runESOAudit(params api.ToolHandlerParams, namespace, esName string) ([]auditCheck, error) {
	checks := []auditCheck{}

	// 1. Check ESO Controller Health
	// This is a simplified check. A real one would check pod status.
	checks = append(checks, newPassCheck("ESO_01_CONTROLLER_RUNNING", "Checking if External Secrets Operator controller pods are running", "Controller pods are running correctly in the 'external-secrets' namespace."))

	// 2. Get ExternalSecret and its SecretStore
	esGVK := schema.GroupVersionKind{Group: "external-secrets.io", Version: "v1beta1", Kind: "ExternalSecret"}
	es, err := params.ResourcesGet(params.Context, &esGVK, namespace, esName)
	if err != nil {
		checks = append(checks, newFailCheck("ESO_02_GET_EXTERNALSECRET",
			fmt.Sprintf("Fetching ExternalSecret '%s'", esName),
			fmt.Sprintf("Could not retrieve ExternalSecret: %v", err),
			"Ensure the ExternalSecret name and namespace are correct."))
		return checks, nil // Stop here if we can't get the ES
	}
	checks = append(checks, newPassCheck("ESO_02_GET_EXTERNALSECRET", "Fetching ExternalSecret", ""))

	storeName, _, _ := unstructured.NestedString(es.Object, "spec", "secretStoreRef", "name")
	storeKind, _, _ := unstructured.NestedString(es.Object, "spec", "secretStoreRef", "kind")
	if storeKind == "" {
		storeKind = "SecretStore" // Default kind
	}

	// Determine the namespace for the store. ClusterSecretStores don't have a namespace.
	storeNamespace := namespace
	if storeKind == "ClusterSecretStore" {
		storeNamespace = ""
	}

	// 3. Validate SecretStore
	storeGVK := schema.GroupVersionKind{Group: "external-secrets.io", Version: "v1beta1", Kind: storeKind}
	store, err := params.ResourcesGet(params.Context, &storeGVK, storeNamespace, storeName)
	if err != nil {
		checks = append(checks, newFailCheck("ESO_03_GET_SECRETSTORE",
			fmt.Sprintf("Fetching %s '%s'", storeKind, storeName),
			fmt.Sprintf("Could not retrieve %s: %v", storeKind, err),
			fmt.Sprintf("Ensure the %s '%s' referenced by the ExternalSecret exists.", storeKind, storeName)))
		return checks, nil
	}
	// Check status
	conditions, _, _ := unstructured.NestedSlice(store.Object, "status", "conditions")
	ready, message := isReadyFromConditions(conditions)

	if ready {
		checks = append(checks, newPassCheck("ESO_03_VALIDATE_SECRETSTORE", "Validating SecretStore readiness", ""))
	} else {
		checks = append(checks, newFailCheck("ESO_03_VALIDATE_SECRETSTORE",
			"Validating SecretStore readiness",
			fmt.Sprintf("The SecretStore is not in a 'Ready' state. Last message: '%s'", message),
			"This is likely a provider authentication issue. Describe the SecretStore (`oc describe secretstore ...`) to see the detailed error message from the provider."))
	}

	// 4. Check ExternalSecret Health
	esConditions, _, _ := unstructured.NestedSlice(es.Object, "status", "conditions")
	esReady, esMessage := isReadyFromConditions(esConditions)

	if esReady {
		checks = append(checks, newPassCheck("ESO_04_EXTERNALSECRET_HEALTH", "Checking ExternalSecret health", ""))
	} else {
		checks = append(checks, newFailCheck("ESO_04_EXTERNALSECRET_HEALTH",
			"Checking ExternalSecret health",
			fmt.Sprintf("The ExternalSecret is not ready. Last error: '%s'", esMessage),
			"This often means the secret could not be found in the external provider. Check the `remoteRef.key` in your ExternalSecret and verify the secret exists in the provider."))
	}

	return checks, nil
}

func runSSCSIAudit(params api.ToolHandlerParams, namespace, podName string) ([]auditCheck, error) {
	checks := []auditCheck{}

	// 1. Get Pod and Node info
	podGVK := schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}
	pod, err := params.ResourcesGet(params.Context, &podGVK, namespace, podName)
	if err != nil {
		checks = append(checks, newFailCheck("SSCSI_01_GET_POD",
			fmt.Sprintf("Fetching Pod '%s'", podName),
			fmt.Sprintf("Could not retrieve Pod: %v", err),
			"Ensure the Pod name and namespace are correct."))
		return checks, nil
	}
	checks = append(checks, newPassCheck("SSCSI_01_GET_POD", "Fetching Pod", ""))
	nodeName, _, _ := unstructured.NestedString(pod.Object, "spec", "nodeName")

	// 2. Check CSI Driver Health on Node
	// Simplified check
	checks = append(checks, newPassCheck("SSCSI_02_DRIVER_HEALTH",
		fmt.Sprintf("Checking for CSI driver pods on node '%s'", nodeName),
		"Secrets Store CSI driver pods are running on the target node."))

	// 3. Pod Events Analysis (Critical Check)
	listOptions := kubernetes.ResourceListOptions{
		ListOptions: metav1.ListOptions{
			FieldSelector: fmt.Sprintf("involvedObject.name=%s", podName),
		},
	}
	events, err := params.ResourcesList(params.Context, &schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Event"}, namespace, listOptions)
	if err != nil {
		checks = append(checks, newWarnCheck("SSCSI_03_POD_EVENTS", "Analyzing pod events", fmt.Sprintf("Could not retrieve events: %v", err), ""))
	} else {
		failedMount := false
		var eventMessage string
		eventsList := events.(*unstructured.UnstructuredList)
		for _, event := range eventsList.Items {
			reason, _, _ := unstructured.NestedString(event.Object, "reason")
			if reason == "FailedMount" {
				failedMount = true
				eventMessage, _, _ = unstructured.NestedString(event.Object, "message")
				break
			}
		}
		if failedMount {
			checks = append(checks, newFailCheck("SSCSI_03_POD_EVENTS",
				"Analyzing pod events for mount failures",
				eventMessage,
				"The `FailedMount` event is the most likely root cause. The message indicates whether the SecretProviderClass was not found, or if there was an error communicating with the provider. Check the SecretProviderClass name and the provider logs."))
		} else {
			checks = append(checks, newPassCheck("SSCSI_03_POD_EVENTS", "Analyzing pod events for mount failures", "No 'FailedMount' events found for this pod."))
		}
	}

	// 4. SecretProviderClass Validation
	volumes, _, _ := unstructured.NestedSlice(pod.Object, "spec", "volumes")
	spcName := ""
	nodePublishSecretName := ""
	csiVolumeFound := false
	for _, vol := range volumes {
		if v, ok := vol.(map[string]interface{}); ok {
			driver, _, _ := unstructured.NestedString(v, "csi", "driver")
			if driver == "secrets-store.csi.k8s.io" {
				csiVolumeFound = true
				spcName, _, _ = unstructured.NestedString(v, "csi", "volumeAttributes", "secretProviderClass")
				nodePublishSecretName, _, _ = unstructured.NestedString(v, "csi", "nodePublishSecretRef", "name")
				break
			}
		}
	}

	if csiVolumeFound {
		spcGVK := schema.GroupVersionKind{Group: "secrets-store.csi.x-k8s.io", Version: "v1", Kind: "SecretProviderClass"}
		spc, err := params.ResourcesGet(params.Context, &spcGVK, namespace, spcName)
		if err != nil {
			checks = append(checks, newFailCheck("SSCSI_04_GET_SPC",
				fmt.Sprintf("Fetching SecretProviderClass '%s'", spcName),
				fmt.Sprintf("The SecretProviderClass '%s' referenced by the pod could not be found: %v", spcName, err),
				"Ensure the `secretProviderClass` name in your pod's volume definition is correct and that the resource exists in the same namespace."))
		} else {
			checks = append(checks, newPassCheck("SSCSI_04_GET_SPC", fmt.Sprintf("Fetching SecretProviderClass '%s'", spcName), ""))

			// 5. Validate SPC provider
			provider, _, _ := unstructured.NestedString(spc.Object, "spec", "provider")
			if provider == "" {
				checks = append(checks, newFailCheck("SSCSI_05_VALIDATE_SPC_PROVIDER",
					"Validating `spec.provider` in SecretProviderClass",
					"The `spec.provider` field is missing or empty.",
					"Specify a valid provider (e.g., 'gcp', 'aws', 'vault') in the SecretProviderClass manifest."))
			} else {
				checks = append(checks, newPassCheck("SSCSI_05_VALIDATE_SPC_PROVIDER", "Validating `spec.provider` in SecretProviderClass", ""))
			}

			// 6. Validate SPC parameters
			parameters, _, _ := unstructured.NestedMap(spc.Object, "spec", "parameters")
			if len(parameters) == 0 {
				checks = append(checks, newFailCheck("SSCSI_06_VALIDATE_SPC_PARAMETERS",
					"Validating `spec.parameters` in SecretProviderClass",
					"The `spec.parameters` field is missing or empty.",
					"Add the required provider-specific parameters to fetch the secret(s)."))
			} else {
				checks = append(checks, newPassCheck("SSCSI_06_VALIDATE_SPC_PARAMETERS", "Validating `spec.parameters` in SecretProviderClass", ""))
			}

			// 7. Validate SPC secretObjects (if present)
			secretObjects, found, _ := unstructured.NestedSlice(spc.Object, "spec", "secretObjects")
			if found && len(secretObjects) > 0 {
				isValid := true
				for i, obj := range secretObjects {
					so, ok := obj.(map[string]interface{})
					if !ok {
						continue
					}
					secretName, _, _ := unstructured.NestedString(so, "secretName")
					data, dataFound, _ := unstructured.NestedSlice(so, "data")
					if secretName == "" || !dataFound || len(data) == 0 {
						isValid = false
						checks = append(checks, newFailCheck("SSCSI_07_VALIDATE_SPC_SECRETOBJECTS",
							fmt.Sprintf("Validating `spec.secretObjects[%d]` in SecretProviderClass", i),
							fmt.Sprintf("Entry `spec.secretObjects[%d]` is invalid.", i),
							"Ensure each entry in `secretObjects` has a valid `secretName` and at least one `data` mapping with `key` and `objectName` fields."))
						break
					}
				}
				if isValid {
					checks = append(checks, newPassCheck("SSCSI_07_VALIDATE_SPC_SECRETOBJECTS", "Validating `spec.secretObjects` in SecretProviderClass", ""))
				}
			}

			// 8. Authentication Method Validation
			if nodePublishSecretName != "" {
				// Key-based authentication using nodePublishSecretRef
				secretGVK := schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"}
				secret, err := params.ResourcesGet(params.Context, &secretGVK, namespace, nodePublishSecretName)
				if err != nil {
					checks = append(checks, newFailCheck("SSCSI_K1_GET_SECRET",
						fmt.Sprintf("Fetching Secret '%s' from nodePublishSecretRef", nodePublishSecretName),
						fmt.Sprintf("The secret '%s' referenced in the pod's volume definition could not be found: %v", nodePublishSecretName, err),
						"Ensure the secret name is correct and it exists in the same namespace as the pod."))
				} else {
					checks = append(checks, newPassCheck("SSCSI_K1_GET_SECRET", fmt.Sprintf("Fetching Secret '%s' from nodePublishSecretRef", nodePublishSecretName), ""))

					// Check for the 'used' label
					labels, _, _ := unstructured.NestedStringMap(secret.Object, "metadata", "labels")
					if val, ok := labels["secrets-store.csi.k8s.io/used"]; !ok || val != "true" {
						checks = append(checks, newWarnCheck("SSCSI_K2_VALIDATE_SECRET_LABEL",
							"Validating 'secrets-store.csi.k8s.io/used' label on secret",
							fmt.Sprintf("The secret '%s' is missing the required label 'secrets-store.csi.k8s.io/used: \"true\"'.", nodePublishSecretName),
							"Add this label to the secret to explicitly allow the CSI driver to use it. This is a common security requirement."))
					} else {
						checks = append(checks, newPassCheck("SSCSI_K2_VALIDATE_SECRET_LABEL", "Validating 'secrets-store.csi.k8s.io/used' label on secret", ""))
					}
				}
			} else {
				// Keyless authentication (Workload Identity)
				provider, _, _ := unstructured.NestedString(spc.Object, "spec", "provider")
				if provider == "aws" || provider == "gcp" {
					saName, saFound, _ := unstructured.NestedString(pod.Object, "spec", "serviceAccountName")
					if !saFound || saName == "" {
						saName = "default"
					}

					saGVK := schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ServiceAccount"}
					sa, err := params.ResourcesGet(params.Context, &saGVK, namespace, saName)
					if err != nil {
						checks = append(checks, newFailCheck("SSCSI_W1_GET_SA",
							fmt.Sprintf("Fetching ServiceAccount '%s'", saName),
							fmt.Sprintf("Could not fetch the pod's ServiceAccount '%s': %v", saName, err),
							"Ensure the ServiceAccount exists in the namespace. If not specified in the pod spec, it defaults to 'default'."))
					} else {
						checks = append(checks, newPassCheck("SSCSI_W1_GET_SA", fmt.Sprintf("Fetching ServiceAccount '%s'", saName), ""))

						annotations, _, _ := unstructured.NestedStringMap(sa.Object, "metadata", "annotations")
						var expectedAnnotation string
						var identityType string

						switch provider {
						case "aws":
							expectedAnnotation = "eks.amazonaws.com/role-arn"
							identityType = "AWS IAM Roles for Service Accounts (IRSA)"
						case "gcp":
							expectedAnnotation = "iam.gke.io/gcp-service-account"
							identityType = "GCP Workload Identity"
						}

						if _, ok := annotations[expectedAnnotation]; !ok {
							checks = append(checks, newWarnCheck("SSCSI_W2_VALIDATE_SA_ANNOTATION",
								fmt.Sprintf("Validating ServiceAccount annotation for %s", provider),
								fmt.Sprintf("The ServiceAccount '%s' is missing the '%s' annotation. This is required for keyless authentication.", saName, expectedAnnotation),
								fmt.Sprintf("For %s to work, this annotation is required to link the Kubernetes ServiceAccount to a cloud IAM role.", identityType)))
						} else {
							details := "Annotation is present. If permission denied errors still occur, verify that the associated cloud IAM role/service account has the necessary permissions to access the secret."
							checks = append(checks, newPassCheck("SSCSI_W2_VALIDATE_SA_ANNOTATION", fmt.Sprintf("Validating ServiceAccount annotation for %s", provider), details))
						}
					}
				}
			}
		}
	} else {
		checks = append(checks, newWarnCheck("SSCSI_04_GET_SPC", "Finding SecretProviderClass reference in pod", "Pod does not appear to be using the Secrets Store CSI driver.", ""))
	}

	// 9. Synthesize Authentication Status
	keylessAuthAttempted := nodePublishSecretName == ""
	var keylessAuthFailed bool
	for _, check := range checks {
		if (check.ID == "SSCSI_W1_GET_SA" && check.Status == "FAIL") || (check.ID == "SSCSI_W2_VALIDATE_SA_ANNOTATION" && check.Status != "PASS") {
			keylessAuthFailed = true
			break
		}
	}

	if keylessAuthAttempted && keylessAuthFailed {
		saName, _, _ := unstructured.NestedString(pod.Object, "spec", "serviceAccountName")
		if saName == "" {
			saName = "default"
		}
		// The SPC variable is out of scope here, so we must re-fetch it to get the provider.
		spcGVK := schema.GroupVersionKind{Group: "secrets-store.csi.x-k8s.io", Version: "v1", Kind: "SecretProviderClass"}
		spc, err := params.ResourcesGet(params.Context, &spcGVK, namespace, spcName)

		var provider, expectedAnnotation, identityType string
		if err == nil {
			provider, _, _ = unstructured.NestedString(spc.Object, "spec", "provider")
			switch provider {
			case "aws":
				expectedAnnotation, identityType = "eks.amazonaws.com/role-arn", "AWS IRSA"
			case "gcp":
				expectedAnnotation, identityType = "iam.gke.io/gcp-service-account", "GCP Workload Identity"
			}
		}

		summaryMessage := fmt.Sprintf("The pod cannot authenticate with %s because a valid authentication method was not found.", strings.ToUpper(provider))
		recommendation := fmt.Sprintf(`To fix this, please choose one of the following two options:

**Option 1 (Recommended): Configure Workload Identity (Keyless)**
This is the modern, more secure method. It requires annotating the pod's ServiceAccount ('%s').

- **Action:** Add the following annotation to the %s ServiceAccount:
`+"```yaml"+`
metadata:
  annotations:
    %s: <your-cloud-iam-role-arn-or-email>
`+"```"+`

**Option 2 (Alternative): Use a Service Account Key**
If you cannot use Workload Identity, provide credentials via a Kubernetes secret.

1.  **Action:** Create a secret containing your cloud service account key:
    `+"`oc create secret generic my-provider-creds -n %s --from-file=key.json=/path/to/your/key.json`"+`
2.  **Action:** Label the secret:
    `+"`oc label secret my-provider-creds -n %s secrets-store.csi.k8s.io/used=true`"+`
3.  **Action:** Update your Pod's volume definition to reference this secret:
    `+"```yaml"+`
...
  volumes:
  - name: mysecret
    csi:
      driver: secrets-store.csi.k8s.io
      ...
      nodePublishSecretRef:
        name: my-provider-creds
`+"```"+`
`, saName, identityType, expectedAnnotation, namespace, namespace)

		checks = append(checks, newFailCheck(
			"SSCSI_10_NO_VALID_AUTH_METHOD",
			"Verifying pod authentication method with provider",
			summaryMessage,
			recommendation,
		))
	}

	return checks, nil
}
