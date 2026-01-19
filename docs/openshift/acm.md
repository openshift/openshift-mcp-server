# Advanced Cluster Management (ACM) Support

The OpenShift MCP Server supports integration with Advanced Cluster Management (ACM) to manage multiple OpenShift clusters from a single hub cluster, starting from version 2.14.0.

## Prerequisites

Before deploying the MCP Server with ACM support, ensure you have:

1. **ACM installed** on your hub cluster (version 2.14.0 or higher)
2. **Managed clusters** imported into ACM
3. **Identity Provider** (e.g., Keycloak) configured with:
   - Separate realm for each cluster (hub + managed clusters)
   - User identity federation between all realms
   - OAuth scopes (e.g., `mcp-server`) configured as default client scopes
4. **Client credentials** for OAuth token exchange in each realm
5. **CA certificate** for your Identity Provider (if using self-signed certificates)

## Identity Provider Integration

The OpenShift MCP Server with ACM requires an Identity Provider (IdP) configuration with a separate realm for each cluster (hub and managed clusters).

### User Identity Federation

Users must have their identity **federated between all realms** for clusters they should have access to. Without identity federation, users will only be able to authenticate to a single cluster.

### OAuth Scopes for Dynamic Client Registration

For clients using Dynamic Client Registration (DCR), such as Claude Code, the required OAuth scopes (e.g., `mcp-server`) must be configured as **default client scopes** in the Identity Provider. This ensures dynamically registered clients automatically include the necessary scopes in their token requests.

## Deployment with Helm

The recommended way to deploy the MCP Server with ACM support is using Helm charts. The configuration is provided through Helm values files in YAML format.

### Step 1: Create a Keycloak CA Secret

If your Identity Provider uses self-signed certificates, create a Kubernetes secret with the CA certificate:

```bash
kubectl create secret generic keycloak-ca \
  --from-file=keycloak-ca.crt=/path/to/your/keycloak-ca.crt \
  -n <namespace>
```

### Step 2: Create Custom Values File

Create a custom values file (e.g., `acm-custom-values.yaml`) with your ACM configuration. The Helm chart converts YAML configuration under the `config:` section to TOML format automatically.

**Example `acm-custom-values.yaml`:**

```yaml
# Keycloak CA certificate volume and mount
extraVolumes:
  - name: keycloak-ca
    secret:
      secretName: keycloak-ca
      optional: false

extraVolumeMounts:
  - name: keycloak-ca
    mountPath: /etc/keycloak-ca
    readOnly: true

# MCP Server configuration
config:
  port: "8080"
  cluster_provider_strategy: acm

  # Hub OAuth Configuration
  require_oauth: true
  oauth_audience: mcp-server
  oauth_scopes:
    - openid
    - mcp-server
  authorization_url: https://your-keycloak-route/realms/hub
  token_url: https://your-keycloak-route/realms/hub/protocol/openid-connect/token

  # Hub Client Credentials
  sts_client_id: mcp-sts
  sts_client_secret: <your-client-secret>

  # Keycloak CA certificate
  certificate_authority: /etc/keycloak-ca/keycloak-ca.crt

  # Deny access to sensitive resources
  denied_resources:
    - group: ""
      version: v1
      kind: Secret
    - group: rbac.authorization.k8s.io
      version: v1
      kind: Role
    - group: rbac.authorization.k8s.io
      version: v1
      kind: RoleBinding
    - group: rbac.authorization.k8s.io
      version: v1
      kind: ClusterRole
    - group: rbac.authorization.k8s.io
      version: v1
      kind: ClusterRoleBinding

  # ACM Provider Configuration
  cluster_provider_configs:
    acm:
      cluster_proxy_addon_skip_tls_verify: true
      token_exchange_strategy: keycloak-v1

      # Managed Clusters
      clusters:
        # Hub cluster (local-cluster) - same-realm token exchange
        local-cluster:
          token_url: https://your-keycloak-route/realms/hub/protocol/openid-connect/token
          client_id: mcp-sts
          client_secret: <your-client-secret>
          audience: mcp-server
          subject_token_type: "urn:ietf:params:oauth:token-type:access_token"
          ca_file: /etc/keycloak-ca/keycloak-ca.crt

        # Managed cluster - cross-realm token exchange
        managed-cluster-1:
          token_url: https://your-keycloak-route/realms/managed-cluster-1/protocol/openid-connect/token
          client_id: mcp-server
          client_secret: <managed-cluster-client-secret>
          subject_issuer: hub-realm
          audience: mcp-server
          subject_token_type: "urn:ietf:params:oauth:token-type:jwt"
          ca_file: /etc/keycloak-ca/keycloak-ca.crt
```

**Configuration Notes:**
- Replace `https://your-keycloak-route` with your actual Keycloak URL
- Replace `<your-client-secret>` with your actual client secrets
- Add additional managed clusters by duplicating the cluster configuration block
- The `config:` section in the YAML values file is automatically converted to TOML format by the Helm chart

### Step 3: Deploy with Helm

Deploy the MCP Server using Helm with the base values, OpenShift values, and your custom ACM values:

```bash
helm install kubernetes-mcp-server ./charts/kubernetes-mcp-server \
  -f ./charts/kubernetes-mcp-server/values.yaml \
  -f ./charts/kubernetes-mcp-server/values-openshift.yaml \
  -f acm-custom-values.yaml \
  -n <namespace> \
  --set ingress.host=kubernetes-mcp-server.apps.<cluster-domain>
```

**Important:** The `values-openshift.yaml` file includes the necessary RBAC permissions for ACM to function properly.

### Step 4: Verify Deployment

Check that the MCP Server is running:

```bash
kubectl get pods -n <namespace>
kubectl logs -n <namespace> deployment/kubernetes-mcp-server
```

Access the MCP Server through the ingress URL:
```
https://kubernetes-mcp-server.apps.<cluster-domain>
```

## Configuration Reference (TOML Format)

For reference, here's what the configuration looks like in TOML format (the Helm chart converts the YAML `config:` section to this format automatically):

```toml
# ACM Multi-Cluster Configuration with Single Keycloak
#
# This configuration uses:
#   - Single Keycloak instance on hub cluster
#   - Multi-realm architecture (hub + managed cluster realms)
#   - V1 token exchange with subject_issuer parameter
#   - Full JWKS signature validation (CA trust configured)

cluster_provider_strategy = "acm"
# In-cluster: kubeconfig is not needed, uses service account credentials

# HTTP server port
port = "8080"

# Hub OAuth Configuration (MUST be before [[denied_resources]] arrays)
require_oauth = true
oauth_audience = "mcp-server"
oauth_scopes = ["openid", "mcp-server"]
authorization_url = "https://your-keycloak-route/realms/hub"
token_url = "https://your-keycloak-route/realms/hub/protocol/openid-connect/token"

# Hub Client Credentials (from .keycloak-config/hub-config.env)
sts_client_id = "mcp-sts"
sts_client_secret = "<MCP client secret (from keycloak)>"

# Keycloak CA certificate (OpenShift router CA)
certificate_authority = "/etc/keycloak-ca/keycloak-ca.crt" # note: this needs to be mounted to the pod at this place

# Deny access to sensitive resources
[[denied_resources]]
group = ""
version = "v1"
kind = "Secret"

[[denied_resources]]
group = "rbac.authorization.k8s.io"
version = "v1"
kind = "Role"

[[denied_resources]]
group = "rbac.authorization.k8s.io"
version = "v1"
kind = "RoleBinding"

[[denied_resources]]
group = "rbac.authorization.k8s.io"
version = "v1"
kind = "ClusterRole"

[[denied_resources]]
group = "rbac.authorization.k8s.io"
version = "v1"
kind = "ClusterRoleBinding"

# ACM Provider Configuration
[cluster_provider_configs.acm]
cluster_proxy_addon_skip_tls_verify = true # alternatively, mount the CA cert for the cluster proxy to the pod
# Token exchange strategy: keycloak-v1, rfc8693, or external-account
token_exchange_strategy = "keycloak-v1"

# Managed Clusters

# Cluster: local-cluster (hub itself - same-realm token exchange)
[cluster_provider_configs.acm.clusters."local-cluster"]
token_url = "https://your-keycloak-route-url/realms/hub/protocol/openid-connect/token"
client_id = "mcp-sts"
client_secret = "<MCP client secret in keycloak>"
audience = "mcp-server"
subject_token_type = "urn:ietf:params:oauth:token-type:access_token" # this can also be auto-detected, feel free to omit
ca_file = "/etc/keycloak-ca/keycloak-ca.crt" # note: this needs to be mounted to the pod

# Cluster: cmurray-managed
[cluster_provider_configs.acm.clusters."managed-cluster"]
token_url = "https://your-keycloak-route-url/realms/manager-cluster-realm/protocol/openid-connect/token"
client_id = "mcp-server"
client_secret = "<MCP client secret in keycloak for the managed-cluster realm>"
subject_issuer = "hub-realm"
audience = "mcp-server"
subject_token_type = "urn:ietf:params:oauth:token-type:jwt" # this can be also be auto-detected, feel free to omit
ca_file = "/etc/keycloak-ca/keycloak-ca.crt" # note: this needs to be mounted to the pod
```
