#!/bin/bash
set -eo pipefail

# Generate acm-kubeconfig.toml from Keycloak configuration files
#
# This script reads from:
#   - .keycloak-config/hub-config.env
#   - .keycloak-config/clusters/*.env
#
# And generates: acm-kubeconfig.toml

OUTPUT_FILE="acm-kubeconfig.toml"

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

# Count managed clusters
CLUSTER_COUNT=$(ls -1 .keycloak-config/clusters/*.env 2>/dev/null | wc -l)
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

EOF

# Add managed clusters section header
echo "# Managed Clusters" >> "$OUTPUT_FILE"
echo "" >> "$OUTPUT_FILE"

# Always add local-cluster (hub itself) with same-realm token exchange
echo "Adding local-cluster (hub itself)..."
cat >> "$OUTPUT_FILE" <<EOF
# Cluster: local-cluster (hub itself - same-realm token exchange)
[cluster_provider_configs.acm-kubeconfig.clusters."local-cluster"]
token_url = "$KEYCLOAK_URL/realms/$HUB_REALM/protocol/openid-connect/token"
client_id = "$STS_CLIENT_ID"
client_secret = "$STS_CLIENT_SECRET"
audience = "mcp-server"
subject_token_type = "urn:ietf:params:oauth:token-type:access_token"

EOF

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

        cat >> "$OUTPUT_FILE" <<EOF
# Cluster: $CLUSTER_NAME
[cluster_provider_configs.acm-kubeconfig.clusters."$CLUSTER_NAME"]
token_url = "$KEYCLOAK_URL/realms/$MANAGED_REALM/protocol/openid-connect/token"
client_id = "$CLUSTER_CLIENT_ID"
client_secret = "$CLUSTER_CLIENT_SECRET"
subject_issuer = "$CLUSTER_IDP_ALIAS"
audience = "mcp-server"
subject_token_type = "urn:ietf:params:oauth:token-type:jwt"

EOF
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
echo "Next steps:"
echo "  1. Review the generated file: cat $OUTPUT_FILE"
echo "  2. Update certificate_authority path if using self-signed certs"
echo "  3. Update kubeconfig path if needed"
echo "  4. Test with: npx @modelcontextprotocol/inspector@latest ./kubernetes-mcp-server --config $OUTPUT_FILE"
echo ""
