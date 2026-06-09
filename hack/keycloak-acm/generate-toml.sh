#!/bin/bash
set -eo pipefail

# Generate ACM configuration files from Keycloak configuration
#
# This script reads from:
#   - .keycloak-config/hub-config.env
#   - .keycloak-config/clusters/*.env
#
# And generates:
#   - _output/acm-kubeconfig.toml (for local development)
#   - _output/acm.toml (for in-cluster deployment)

# Get script directory and repo root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Create _output directory if it doesn't exist
mkdir -p "$REPO_ROOT/_output"

OUTPUT_FILE="$REPO_ROOT/_output/acm-kubeconfig.toml"

echo "==========================================="
echo "Generating acm-kubeconfig.toml"
echo "==========================================="
echo ""

# Check if hub config exists
if [ ! -f ".keycloak-config/hub-config.env" ]; then
    echo "❌ Error: Hub configuration not found"
    echo "   Run: make keycloak-acm-setup-hub"
    exit 1
fi

# Load hub config
source .keycloak-config/hub-config.env

# Detect hub kubeconfig
if [ -n "${HUB_KUBECONFIG:-}" ]; then
    HUB_KUBECONFIG_PATH="$HUB_KUBECONFIG"
elif [ -n "${KUBECONFIG:-}" ]; then
    HUB_KUBECONFIG_PATH="$KUBECONFIG"
else
    HUB_KUBECONFIG_PATH="$HOME/.kube/config"
fi

# Detect context name from kubeconfig
if [ -f "$HUB_KUBECONFIG_PATH" ]; then
    CONTEXT_NAME=$(kubectl --kubeconfig="$HUB_KUBECONFIG_PATH" config current-context 2>/dev/null || echo "admin")
else
    CONTEXT_NAME="admin"
fi

echo "Configuration:"
echo "  Hub Kubeconfig: $HUB_KUBECONFIG_PATH"
echo "  Context Name: $CONTEXT_NAME"
echo "  Keycloak URL: $KEYCLOAK_URL"
echo "  Hub Realm: $HUB_REALM"
echo ""

# Count managed clusters (handle case where directory is empty or doesn't exist)
if [ -d ".keycloak-config/clusters" ]; then
    CLUSTER_COUNT=$(find .keycloak-config/clusters -maxdepth 1 -name "*.env" -type f 2>/dev/null | wc -l | tr -d ' ')
else
    CLUSTER_COUNT=0
fi
echo "  Managed Clusters: $CLUSTER_COUNT"
echo ""

# Generate TOML file
cat > "$OUTPUT_FILE" <<EOF
# ACM Multi-Cluster Configuration with Single Keycloak
# Generated: $(date -Iseconds)
#
# This configuration uses:
#   - Single Keycloak instance on hub cluster
#   - Multi-realm architecture (hub + managed cluster realms)
#   - V1 token exchange with subject_issuer parameter
#   - Full JWKS signature validation (CA trust configured)

cluster_provider_strategy = "acm-kubeconfig"
kubeconfig = "$HUB_KUBECONFIG_PATH"

# Hub OAuth Configuration
require_oauth = true
oauth_audience = "mcp-server"
oauth_scopes = ["openid", "mcp-server"]
authorization_url = "$KEYCLOAK_URL/realms/$HUB_REALM"
token_url = "$KEYCLOAK_URL/realms/$HUB_REALM/protocol/openid-connect/token"

# Hub Client Credentials (from .keycloak-config/hub-config.env)
sts_client_id = "$STS_CLIENT_ID"
sts_client_secret = "$STS_CLIENT_SECRET"

EOF

# Extract and add Keycloak CA certificate
echo "Extracting Keycloak CA certificate..."

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CA_DIR="$REPO_ROOT/_output"
mkdir -p "$CA_DIR"

CA_FILE="$CA_DIR/keycloak-ca.crt"

if kubectl --kubeconfig="$HUB_KUBECONFIG_PATH" get configmap router-ca -n keycloak -o jsonpath='{.data.router-ca\.crt}' > "$CA_FILE" 2>/dev/null; then
    echo "  ✅ Keycloak CA extracted to $CA_FILE"
    cat >> "$OUTPUT_FILE" <<EOF
# Keycloak CA certificate (OpenShift router CA)
certificate_authority = "$CA_FILE"

EOF
else
    echo "  ⚠️  Could not extract Keycloak CA, TLS verification may fail"
    cat >> "$OUTPUT_FILE" <<EOF
# Optional: Keycloak CA certificate (if using self-signed certs)
# Uncomment and update path if needed:
# certificate_authority = "$CA_DIR/keycloak-ca.crt"

EOF
fi

# Add ACM Provider Configuration
cat >> "$OUTPUT_FILE" <<EOF
# ACM Provider Configuration
[cluster_provider_configs.acm-kubeconfig]
context_name = "$CONTEXT_NAME"
cluster_proxy_addon_skip_tls_verify = true
# Token exchange strategy: keycloak-v1, rfc8693, or external-account
token_exchange_strategy = "keycloak-v1"

EOF

# Add managed clusters section header
echo "# Managed Clusters" >> "$OUTPUT_FILE"
echo "" >> "$OUTPUT_FILE"

# Always add local-cluster (hub itself) with same-realm token exchange
echo "Adding local-cluster (hub itself)..."
if [ -f "$CA_FILE" ]; then
cat >> "$OUTPUT_FILE" <<EOF
# Cluster: local-cluster (hub itself - same-realm token exchange)
[cluster_provider_configs.acm-kubeconfig.clusters."local-cluster"]
token_url = "$KEYCLOAK_URL/realms/$HUB_REALM/protocol/openid-connect/token"
client_id = "$STS_CLIENT_ID"
client_secret = "$STS_CLIENT_SECRET"
audience = "mcp-server"
subject_token_type = "urn:ietf:params:oauth:token-type:access_token"
ca_file = "$CA_FILE"

EOF
else
cat >> "$OUTPUT_FILE" <<EOF
# Cluster: local-cluster (hub itself - same-realm token exchange)
[cluster_provider_configs.acm-kubeconfig.clusters."local-cluster"]
token_url = "$KEYCLOAK_URL/realms/$HUB_REALM/protocol/openid-connect/token"
client_id = "$STS_CLIENT_ID"
client_secret = "$STS_CLIENT_SECRET"
audience = "mcp-server"
subject_token_type = "urn:ietf:params:oauth:token-type:access_token"

EOF
fi

# Add other managed clusters
if [ "$CLUSTER_COUNT" -gt 0 ]; then
    for cluster_env in .keycloak-config/clusters/*.env; do
        if [ ! -f "$cluster_env" ]; then
            continue
        fi

        # Extract variables from cluster env file
        CLUSTER_NAME=$(grep '^CLUSTER_NAME=' "$cluster_env" | cut -d'=' -f2 | tr -d '"')
        MANAGED_REALM=$(grep '^MANAGED_REALM=' "$cluster_env" | cut -d'=' -f2 | tr -d '"')
        CLUSTER_CLIENT_ID=$(grep '^CLIENT_ID=' "$cluster_env" | cut -d'=' -f2 | tr -d '"')
        CLUSTER_CLIENT_SECRET=$(grep '^CLIENT_SECRET=' "$cluster_env" | cut -d'=' -f2 | tr -d '"')
        CLUSTER_IDP_ALIAS=$(grep '^IDP_ALIAS=' "$cluster_env" | cut -d'=' -f2 | tr -d '"')

        echo "  Adding cluster: $CLUSTER_NAME"

        echo "# Cluster: $CLUSTER_NAME" >> "$OUTPUT_FILE"
        echo "[cluster_provider_configs.acm-kubeconfig.clusters.\"$CLUSTER_NAME\"]" >> "$OUTPUT_FILE"
        echo "token_url = \"$KEYCLOAK_URL/realms/$MANAGED_REALM/protocol/openid-connect/token\"" >> "$OUTPUT_FILE"
        echo "client_id = \"$CLUSTER_CLIENT_ID\"" >> "$OUTPUT_FILE"
        echo "client_secret = \"$CLUSTER_CLIENT_SECRET\"" >> "$OUTPUT_FILE"
        echo "subject_issuer = \"$CLUSTER_IDP_ALIAS\"" >> "$OUTPUT_FILE"
        echo "audience = \"mcp-server\"" >> "$OUTPUT_FILE"
        echo "subject_token_type = \"urn:ietf:params:oauth:token-type:jwt\"" >> "$OUTPUT_FILE"
        if [ -f "$CA_FILE" ]; then
            echo "ca_file = \"$CA_FILE\"" >> "$OUTPUT_FILE"
        fi
        echo "" >> "$OUTPUT_FILE"
    done
fi

echo ""
echo "==========================================="
echo "✅ Generated: $OUTPUT_FILE"
echo "==========================================="
echo ""
echo "Configuration includes:"
echo "  ✅ Hub OAuth configuration"
echo "  ✅ Hub client credentials"
echo "  ✅ local-cluster (hub itself - same-realm token exchange)"
echo "  ✅ $CLUSTER_COUNT managed cluster(s) (cross-realm token exchange)"
echo ""

# Now generate in-cluster version (acm.toml)
echo "==========================================="
echo "Generating acm.toml (in-cluster)"
echo "==========================================="
echo ""

INCLUSTER_OUTPUT_FILE="$REPO_ROOT/_output/acm.toml"
INCLUSTER_CA_PATH="/etc/keycloak-ca/keycloak-ca.crt"

# Generate in-cluster TOML file
cat > "$INCLUSTER_OUTPUT_FILE" <<EOF
# ACM Multi-Cluster Configuration with Single Keycloak (In-Cluster)
# Generated: $(date -Iseconds)
#
# This configuration uses:
#   - Single Keycloak instance on hub cluster
#   - Multi-realm architecture (hub + managed cluster realms)
#   - V1 token exchange with subject_issuer parameter
#   - Full JWKS signature validation (CA trust configured)
#   - In-cluster service account for authentication

cluster_provider_strategy = "acm"
# In-cluster: kubeconfig not needed, uses service account credentials

# Hub OAuth Configuration
require_oauth = true
oauth_audience = "mcp-server"
oauth_scopes = ["openid", "mcp-server"]
authorization_url = "$KEYCLOAK_URL/realms/$HUB_REALM"
token_url = "$KEYCLOAK_URL/realms/$HUB_REALM/protocol/openid-connect/token"

# Hub Client Credentials (from .keycloak-config/hub-config.env)
sts_client_id = "$STS_CLIENT_ID"
sts_client_secret = "$STS_CLIENT_SECRET"

# Keycloak CA certificate (OpenShift router CA - mounted in pod)
certificate_authority = "$INCLUSTER_CA_PATH"

# ACM Provider Configuration
[cluster_provider_configs.acm]
# In-cluster: context_name not needed, uses current context from service account
cluster_proxy_addon_skip_tls_verify = true
# Token exchange strategy: keycloak-v1, rfc8693, or external-account
token_exchange_strategy = "keycloak-v1"

# Managed Clusters

# Cluster: local-cluster (hub itself - same-realm token exchange)
[cluster_provider_configs.acm.clusters."local-cluster"]
token_url = "$KEYCLOAK_URL/realms/$HUB_REALM/protocol/openid-connect/token"
client_id = "$STS_CLIENT_ID"
client_secret = "$STS_CLIENT_SECRET"
audience = "mcp-server"
subject_token_type = "urn:ietf:params:oauth:token-type:access_token"
ca_file = "$INCLUSTER_CA_PATH"

EOF

# Add other managed clusters to in-cluster config
if [ "$CLUSTER_COUNT" -gt 0 ]; then
    for cluster_env in .keycloak-config/clusters/*.env; do
        if [ ! -f "$cluster_env" ]; then
            continue
        fi

        # Extract variables from cluster env file
        CLUSTER_NAME=$(grep '^CLUSTER_NAME=' "$cluster_env" | cut -d'=' -f2 | tr -d '"')
        MANAGED_REALM=$(grep '^MANAGED_REALM=' "$cluster_env" | cut -d'=' -f2 | tr -d '"')
        CLUSTER_CLIENT_ID=$(grep '^CLIENT_ID=' "$cluster_env" | cut -d'=' -f2 | tr -d '"')
        CLUSTER_CLIENT_SECRET=$(grep '^CLIENT_SECRET=' "$cluster_env" | cut -d'=' -f2 | tr -d '"')
        CLUSTER_IDP_ALIAS=$(grep '^IDP_ALIAS=' "$cluster_env" | cut -d'=' -f2 | tr -d '"')

        echo "# Cluster: $CLUSTER_NAME" >> "$INCLUSTER_OUTPUT_FILE"
        echo "[cluster_provider_configs.acm.clusters.\"$CLUSTER_NAME\"]" >> "$INCLUSTER_OUTPUT_FILE"
        echo "token_url = \"$KEYCLOAK_URL/realms/$MANAGED_REALM/protocol/openid-connect/token\"" >> "$INCLUSTER_OUTPUT_FILE"
        echo "client_id = \"$CLUSTER_CLIENT_ID\"" >> "$INCLUSTER_OUTPUT_FILE"
        echo "client_secret = \"$CLUSTER_CLIENT_SECRET\"" >> "$INCLUSTER_OUTPUT_FILE"
        echo "subject_issuer = \"$CLUSTER_IDP_ALIAS\"" >> "$INCLUSTER_OUTPUT_FILE"
        echo "audience = \"mcp-server\"" >> "$INCLUSTER_OUTPUT_FILE"
        echo "subject_token_type = \"urn:ietf:params:oauth:token-type:jwt\"" >> "$INCLUSTER_OUTPUT_FILE"
        echo "ca_file = \"$INCLUSTER_CA_PATH\"" >> "$INCLUSTER_OUTPUT_FILE"
        echo "" >> "$INCLUSTER_OUTPUT_FILE"
    done
fi

echo ""
echo "==========================================="
echo "✅ Generated: $INCLUSTER_OUTPUT_FILE"
echo "==========================================="
echo ""

# Now generate Helm values YAML file (acm-values.yaml)
echo "==========================================="
echo "Generating acm-values.yaml (Helm values)"
echo "==========================================="
echo ""

HELM_VALUES_FILE="$REPO_ROOT/_output/acm-values.yaml"

# Generate Helm values YAML file
cat > "$HELM_VALUES_FILE" <<'EOF'
# ACM Multi-Cluster Configuration with Single Keycloak (Helm Values)
# Generated from Keycloak configuration
#
# This configuration uses:
#   - Single Keycloak instance on hub cluster
#   - Multi-realm architecture (hub + managed cluster realms)
#   - V1 token exchange with subject_issuer parameter
#   - Full JWKS signature validation (CA trust configured)
#
# Usage:
#   helm install kubernetes-mcp-server ./charts/kubernetes-mcp-server \
#     -f ./charts/kubernetes-mcp-server/values.yaml \
#     -f ./charts/kubernetes-mcp-server/values-openshift.yaml \
#     -f _output/acm-values.yaml \
#     -n <namespace> \
#     --set ingress.host=kubernetes-mcp-server.apps.<cluster-domain>

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
  port: "{{ .Values.service.port }}"
  cluster_provider_strategy: acm

  # Hub OAuth Configuration
  require_oauth: true
  oauth_audience: mcp-server
  oauth_scopes:
    - openid
    - mcp-server
EOF

# Add Keycloak URLs
cat >> "$HELM_VALUES_FILE" <<EOF
  authorization_url: $KEYCLOAK_URL/realms/$HUB_REALM
  token_url: $KEYCLOAK_URL/realms/$HUB_REALM/protocol/openid-connect/token

  # Hub Client Credentials
  sts_client_id: $STS_CLIENT_ID
  sts_client_secret: $STS_CLIENT_SECRET

  # Keycloak CA certificate
  certificate_authority: $INCLUSTER_CA_PATH

  # ACM Provider Configuration
  cluster_provider_configs:
    acm:
      cluster_proxy_addon_skip_tls_verify: true
      token_exchange_strategy: keycloak-v1

      # Managed Clusters
      clusters:
        # Cluster: local-cluster (hub itself - same-realm token exchange)
        local-cluster:
          token_url: $KEYCLOAK_URL/realms/$HUB_REALM/protocol/openid-connect/token
          client_id: $STS_CLIENT_ID
          client_secret: $STS_CLIENT_SECRET
          audience: mcp-server
          subject_token_type: "urn:ietf:params:oauth:token-type:access_token"
          ca_file: $INCLUSTER_CA_PATH
EOF

# Add other managed clusters to Helm values
if [ "$CLUSTER_COUNT" -gt 0 ]; then
    for cluster_env in .keycloak-config/clusters/*.env; do
        if [ ! -f "$cluster_env" ]; then
            continue
        fi

        # Extract variables from cluster env file
        CLUSTER_NAME=$(grep '^CLUSTER_NAME=' "$cluster_env" | cut -d'=' -f2 | tr -d '"')
        MANAGED_REALM=$(grep '^MANAGED_REALM=' "$cluster_env" | cut -d'=' -f2 | tr -d '"')
        CLUSTER_CLIENT_ID=$(grep '^CLIENT_ID=' "$cluster_env" | cut -d'=' -f2 | tr -d '"')
        CLUSTER_CLIENT_SECRET=$(grep '^CLIENT_SECRET=' "$cluster_env" | cut -d'=' -f2 | tr -d '"')
        CLUSTER_IDP_ALIAS=$(grep '^IDP_ALIAS=' "$cluster_env" | cut -d'=' -f2 | tr -d '"')

        cat >> "$HELM_VALUES_FILE" <<EOF

        # Cluster: $CLUSTER_NAME
        $CLUSTER_NAME:
          token_url: $KEYCLOAK_URL/realms/$MANAGED_REALM/protocol/openid-connect/token
          client_id: $CLUSTER_CLIENT_ID
          client_secret: $CLUSTER_CLIENT_SECRET
          subject_issuer: $CLUSTER_IDP_ALIAS
          audience: mcp-server
          subject_token_type: "urn:ietf:params:oauth:token-type:jwt"
          ca_file: $INCLUSTER_CA_PATH
EOF
    done
fi

echo ""
echo "==========================================="
echo "✅ Generated: $HELM_VALUES_FILE"
echo "==========================================="
echo ""
echo "Summary:"
echo "  📄 Local development config:  $OUTPUT_FILE"
echo "  📄 In-cluster TOML config:    $INCLUSTER_OUTPUT_FILE"
echo "  📄 Helm values file:          $HELM_VALUES_FILE"
echo ""
echo "Next steps:"
echo "  Local development:"
echo "    1. Review: cat $OUTPUT_FILE"
echo "    2. Test with: npx @modelcontextprotocol/inspector@latest ./kubernetes-mcp-server --config $OUTPUT_FILE"
echo ""
echo "  In-cluster deployment (Option 1 - Helm with YAML values - RECOMMENDED):"
echo "    1. Create Secret: kubectl create secret generic keycloak-ca --from-file=keycloak-ca.crt=$CA_FILE -n <namespace>"
echo "    2. Deploy with Helm: helm install kubernetes-mcp-server ./charts/kubernetes-mcp-server -f $HELM_VALUES_FILE -n <namespace>"
echo ""
echo "  In-cluster deployment (Option 2 - Helm with --set-file):"
echo "    1. Create Secret: kubectl create secret generic keycloak-ca --from-file=keycloak-ca.crt=$CA_FILE -n <namespace>"
echo "    2. Deploy with Helm: helm install kubernetes-mcp-server ./charts/kubernetes-mcp-server --set-file configToml=$INCLUSTER_OUTPUT_FILE -n <namespace>"
echo ""
