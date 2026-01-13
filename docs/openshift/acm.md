# Advanced Cluster Management (ACM) Support

The OpenShift MCP Server supports integration with Advanced Cluster Management (ACM) to manage multiple OpenShift clusters from a single hub cluster, starting from version 2.14.0.

## Identity Provider Integration

The OpenShift MCP Server with ACM requires an Identity Provider (IdP) configuration with a separate realm for each cluster (hub and managed clusters).

### User Identity Federation

Users must have their identity **federated between all realms** for clusters they should have access to. Without identity federation, users will only be able to authenticate to a single cluster.

### OAuth Scopes for Dynamic Client Registration

For clients using Dynamic Client Registration (DCR), such as Claude Code, the required OAuth scopes (e.g., `mcp-server`) must be configured as **default client scopes** in the Identity Provider. This ensures dynamically registered clients automatically include the necessary scopes in their token requests.

## Configuration example

Below is an example of how your `config.toml` file could look like:

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
