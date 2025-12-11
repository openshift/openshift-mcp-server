#!/bin/bash
set -euo pipefail

# ACM Managed Cluster Registration Script (Declarative)
# This script registers a managed cluster realm and configures cross-realm token exchange
#
# Required environment variables:
#   CLUSTER_NAME           - Name of the managed cluster (e.g., managed-cluster-one)
#   HUB_KUBECONFIG         - Path to hub cluster kubeconfig
#   MANAGED_KUBECONFIG     - Path to managed cluster kubeconfig
#
# Optional environment variables:
#   KEYCLOAK_CA_CERT       - Path to CA certificate for HTTPS verification (optional)

# Validate required variables
: "${CLUSTER_NAME:?Error: CLUSTER_NAME environment variable is required}"
: "${HUB_KUBECONFIG:?Error: HUB_KUBECONFIG environment variable is required}"
: "${MANAGED_KUBECONFIG:?Error: MANAGED_KUBECONFIG environment variable is required}"

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
KEYCLOAK_CONFIG_DIR="$REPO_ROOT/dev/config/openshift/keycloak"
HUB_CONFIG_ENV="$REPO_ROOT/.keycloak-config/hub-config.env"
CLUSTER_CONFIG_DIR="$REPO_ROOT/.keycloak-config/clusters"

# Load hub configuration
if [ ! -f "$HUB_CONFIG_ENV" ]; then
    echo "❌ Hub configuration not found at $HUB_CONFIG_ENV"
    echo "Please run ./hack/acm/acm-keycloak-setup-hub-declarative.sh first"
    exit 1
fi

source "$HUB_CONFIG_ENV"

# Set curl options based on CA cert availability
CURL_OPTS="-sk"
if [ -n "${KEYCLOAK_CA_CERT:-}" ]; then
    CURL_OPTS="--cacert $KEYCLOAK_CA_CERT"
fi

MANAGED_REALM="$CLUSTER_NAME"
IDP_ALIAS="hub-realm"

echo "========================================="
echo "ACM Managed Cluster Registration (Declarative)"
echo "========================================="
echo "Cluster Name: $CLUSTER_NAME"
echo "Managed Realm: $MANAGED_REALM"
echo "Hub Realm: $HUB_REALM"
echo "Keycloak URL: $KEYCLOAK_URL"
echo ""

#=============================================================================
# ACM ManagedCluster Registration
#=============================================================================
echo "========================================="
echo "ACM ManagedCluster Registration"
echo "========================================="
echo ""

# Step 2: Create ManagedCluster in ACM
echo "Step 2: Creating ManagedCluster in ACM..."
cat <<EOF | kubectl --kubeconfig="$HUB_KUBECONFIG" apply -f -
apiVersion: cluster.open-cluster-management.io/v1
kind: ManagedCluster
metadata:
  name: $CLUSTER_NAME
  labels:
    cloud: auto-detect
    vendor: auto-detect
spec:
  hubAcceptsClient: true
  leaseDurationSeconds: 60
EOF

if [ $? -eq 0 ]; then
    echo "  ✅ ManagedCluster created/updated"
else
    echo "  ❌ Failed to create ManagedCluster"
    exit 1
fi

echo ""
echo "Waiting for import secret to be generated (30 seconds)..."
sleep 30

# Verify import secret exists
if ! kubectl --kubeconfig="$HUB_KUBECONFIG" get secret ${CLUSTER_NAME}-import -n ${CLUSTER_NAME} &>/dev/null; then
    echo "  ⚠️  Warning: Import secret not found yet, waiting longer..."
    sleep 30
fi

echo ""

# Step 3: Apply import manifests to managed cluster
echo "Step 3: Applying ACM import manifests to managed cluster..."

# Extract manifests
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

kubectl --kubeconfig="$HUB_KUBECONFIG" get secret ${CLUSTER_NAME}-import -n ${CLUSTER_NAME} \
    -o jsonpath='{.data.crds\.yaml}' | base64 -d > "$TEMP_DIR/crds.yaml"
kubectl --kubeconfig="$HUB_KUBECONFIG" get secret ${CLUSTER_NAME}-import -n ${CLUSTER_NAME} \
    -o jsonpath='{.data.import\.yaml}' | base64 -d > "$TEMP_DIR/import.yaml"

echo "  Applying CRDs to managed cluster..."
kubectl --kubeconfig="$MANAGED_KUBECONFIG" apply -f "$TEMP_DIR/crds.yaml"

echo "  Applying import resources to managed cluster..."
kubectl --kubeconfig="$MANAGED_KUBECONFIG" apply -f "$TEMP_DIR/import.yaml"

echo "  ✅ Import manifests applied"
echo ""
echo "  Waiting for klusterlet agents to start (30 seconds)..."
sleep 30

# Verify klusterlet pods are starting
echo "  Klusterlet agent pods:"
kubectl --kubeconfig="$MANAGED_KUBECONFIG" get pods -n open-cluster-management-agent 2>/dev/null || echo "    (starting...)"
echo ""
echo "  Klusterlet addon pods (including cluster-proxy):"
kubectl --kubeconfig="$MANAGED_KUBECONFIG" get pods -n open-cluster-management-agent-addon 2>/dev/null || echo "    (starting...)"
echo ""

# Check if cluster-proxy agent is running or starting
PROXY_PODS=$(kubectl --kubeconfig="$MANAGED_KUBECONFIG" get pods -n open-cluster-management-agent-addon \
    -l component=cluster-proxy-proxy-agent --no-headers 2>/dev/null | wc -l)
if [ "$PROXY_PODS" -gt 0 ]; then
    echo "  ✅ Cluster-proxy agent pods detected"
else
    echo "  ⚠️  Cluster-proxy agent pods not yet running (may take a few minutes)"
fi

echo ""

#=============================================================================
# Keycloak Realm Configuration
#=============================================================================
echo "========================================="
echo "Keycloak Realm Configuration"
echo "========================================="
echo ""

# Get admin token (do this right before Keycloak operations to avoid expiration)
echo "Step 4: Getting Keycloak admin access token..."
echo "  Keycloak URL: $KEYCLOAK_URL"
echo "  Checking Keycloak availability..."

# Test if Keycloak is reachable
if ! curl $CURL_OPTS -sf "$KEYCLOAK_URL/realms/master" > /dev/null 2>&1; then
    echo "  ⚠️  Keycloak not yet reachable, waiting 30 seconds..."
    sleep 30

    if ! curl $CURL_OPTS -sf "$KEYCLOAK_URL/realms/master" > /dev/null 2>&1; then
        echo "  ❌ Keycloak still not reachable at $KEYCLOAK_URL"
        echo "  Please ensure Keycloak is running: make keycloak-status"
        exit 1
    fi
fi

echo "  ✅ Keycloak is reachable"
echo "  Requesting admin token..."

RESPONSE=$(curl $CURL_OPTS -X POST "$KEYCLOAK_URL/realms/master/protocol/openid-connect/token" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "username=$ADMIN_USER" \
    -d "password=$ADMIN_PASSWORD" \
    -d "grant_type=password" \
    -d "client_id=admin-cli")

ADMIN_TOKEN=$(echo "$RESPONSE" | jq -r '.access_token // empty' 2>/dev/null)

if [ -z "$ADMIN_TOKEN" ] || [ "$ADMIN_TOKEN" = "null" ]; then
    echo "  ❌ Failed to get access token"
    echo "  Response: $RESPONSE"
    echo ""
    echo "  Please check:"
    echo "    - Keycloak admin credentials in .keycloak-config/hub-config.env"
    echo "    - ADMIN_USER: $ADMIN_USER"
    echo "    - Keycloak status: make keycloak-status"
    exit 1
fi

echo "  ✅ Admin token obtained"
echo ""

# Step 5: Create managed cluster realm
echo "Step 5: Creating managed cluster realm..."
REALM_JSON=$(cat "$KEYCLOAK_CONFIG_DIR/realm/managed-realm-create.json" | sed "s/\"managed-cluster-one\"/\"$MANAGED_REALM\"/")
REALM_RESPONSE=$(curl $CURL_OPTS -w "%{http_code}" -X POST "$KEYCLOAK_URL/admin/realms" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "$REALM_JSON")

REALM_CODE=$(echo "$REALM_RESPONSE" | tail -c 4)

if [ "$REALM_CODE" = "201" ]; then
    echo "  ✅ Managed realm created"
elif [ "$REALM_CODE" = "409" ]; then
    echo "  ✅ Managed realm already exists"
else
    echo "  ❌ Failed to create managed realm (HTTP $REALM_CODE)"
    exit 1
fi
echo ""

# Step 3: Create client scopes
echo "Step 3: Creating client scopes in managed realm..."

SCOPE_RESPONSE=$(curl $CURL_OPTS -w "%{http_code}" -X POST "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/client-scopes" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d @"$KEYCLOAK_CONFIG_DIR/client-scopes/openid.json")

SCOPE_CODE=$(echo "$SCOPE_RESPONSE" | tail -c 4)
if [ "$SCOPE_CODE" = "201" ] || [ "$SCOPE_CODE" = "409" ]; then
    echo "  ✅ openid scope created/exists"
fi

SCOPE_RESPONSE=$(curl $CURL_OPTS -w "%{http_code}" -X POST "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/client-scopes" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d @"$KEYCLOAK_CONFIG_DIR/client-scopes/mcp-server.json")

SCOPE_CODE=$(echo "$SCOPE_RESPONSE" | tail -c 4)
if [ "$SCOPE_CODE" = "201" ] || [ "$SCOPE_CODE" = "409" ]; then
    echo "  ✅ mcp-server scope created/exists"
fi
echo ""

# Step 4: Add protocol mappers to mcp-server scope
echo "Step 4: Adding protocol mappers..."

SCOPES_LIST=$(curl $CURL_OPTS -X GET "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/client-scopes" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
MCP_SERVER_SCOPE_ID=$(echo "$SCOPES_LIST" | jq -r '.[] | select(.name == "mcp-server") | .id // empty')

MAPPER_RESPONSE=$(curl $CURL_OPTS -w "%{http_code}" -X POST "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/client-scopes/$MCP_SERVER_SCOPE_ID/protocol-mappers/models" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d @"$KEYCLOAK_CONFIG_DIR/mappers/mcp-server-audience-mapper.json")

MAPPER_CODE=$(echo "$MAPPER_RESPONSE" | tail -c 4)
if [ "$MAPPER_CODE" = "201" ] || [ "$MAPPER_CODE" = "409" ]; then
    echo "  ✅ mcp-server audience mapper added"
fi
echo ""

# Step 5: Create mcp-server client in managed realm
echo "Step 5: Creating mcp-server client in managed realm..."

CLIENT_RESPONSE=$(curl $CURL_OPTS -w "%{http_code}" -X POST "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/clients" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d @"$KEYCLOAK_CONFIG_DIR/clients/mcp-server.json")

CLIENT_CODE=$(echo "$CLIENT_RESPONSE" | tail -c 4)
if [ "$CLIENT_CODE" = "201" ] || [ "$CLIENT_CODE" = "409" ]; then
    echo "  ✅ mcp-server client created/exists"
fi
echo ""

# Step 6: Get managed mcp-server client UUID and secret
echo "Step 6: Retrieving managed cluster client details..."

CLIENTS_LIST=$(curl $CURL_OPTS -X GET "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/clients" \
    -H "Authorization: Bearer $ADMIN_TOKEN")

MANAGED_CLIENT_UUID=$(echo "$CLIENTS_LIST" | jq -r '.[] | select(.clientId == "mcp-server") | .id')
MANAGED_SECRET_RESPONSE=$(curl $CURL_OPTS -X GET "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/clients/$MANAGED_CLIENT_UUID/client-secret" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
MANAGED_CLIENT_SECRET=$(echo "$MANAGED_SECRET_RESPONSE" | jq -r '.value')

echo "  ✅ Managed mcp-server UUID: $MANAGED_CLIENT_UUID"
echo ""

# Step 6a: Add protocol mapper for sub claim (required for token exchange)
echo "Step 6a: Adding protocol mapper for sub claim..."

# Check if user-id mapper already exists
MAPPERS=$(curl $CURL_OPTS -X GET "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/clients/$MANAGED_CLIENT_UUID/protocol-mappers/models" \
    -H "Authorization: Bearer $ADMIN_TOKEN")

USER_ID_MAPPER=$(echo "$MAPPERS" | jq -r '.[] | select(.name == "user-id") | .id')

if [ -z "$USER_ID_MAPPER" ] || [ "$USER_ID_MAPPER" = "null" ]; then
    MAPPER_RESPONSE=$(curl $CURL_OPTS -w "%{http_code}" -X POST "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/clients/$MANAGED_CLIENT_UUID/protocol-mappers/models" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{
          "name": "user-id",
          "protocol": "openid-connect",
          "protocolMapper": "oidc-usermodel-property-mapper",
          "config": {
            "user.attribute": "id",
            "claim.name": "sub",
            "jsonType.label": "String",
            "id.token.claim": "true",
            "access.token.claim": "true",
            "userinfo.token.claim": "true"
          }
        }')

    MAPPER_CODE=$(echo "$MAPPER_RESPONSE" | tail -c 4)
    if [ "$MAPPER_CODE" = "201" ]; then
        echo "  ✅ Created user-id mapper for sub claim"
    else
        echo "  ⚠️  Failed to create user-id mapper (code: $MAPPER_CODE)"
    fi
else
    echo "  ✅ user-id mapper already exists"
fi
echo ""

# Step 7: Create identity provider pointing to hub realm
echo "Step 7: Creating identity provider (hub realm)..."

IDP_JSON=$(cat "$KEYCLOAK_CONFIG_DIR/identity-providers/hub-realm-idp-template.json" | \
    sed "s|\${KEYCLOAK_URL}|$KEYCLOAK_URL|g" | \
    sed "s|\${HUB_CLIENT_SECRET}|$CLIENT_SECRET|g" | \
    sed "s/\"hub-realm\"/\"$IDP_ALIAS\"/")

IDP_RESPONSE=$(curl $CURL_OPTS -w "%{http_code}" -X POST "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/identity-provider/instances" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "$IDP_JSON")

IDP_CODE=$(echo "$IDP_RESPONSE" | tail -c 4)

if [ "$IDP_CODE" = "201" ]; then
    echo "  ✅ Identity provider created"
elif [ "$IDP_CODE" = "409" ]; then
    echo "  ✅ Identity provider already exists"
    # Update existing IDP
    curl $CURL_OPTS -X PUT "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/identity-provider/instances/$IDP_ALIAS" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -H "Content-Type: application/json" \
        -d "$IDP_JSON" > /dev/null
    echo "  ✅ Identity provider updated"
else
    echo "  ❌ Failed to create identity provider (HTTP $IDP_CODE)"
    exit 1
fi
echo ""

# Step 8: Create federated identity link
echo "Step 8: Creating federated identity link..."

# Get hub user ID
HUB_USERS=$(curl $CURL_OPTS -X GET "$KEYCLOAK_URL/admin/realms/$HUB_REALM/users?username=$MCP_USERNAME" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
HUB_USER_ID=$(echo "$HUB_USERS" | jq -r '.[0].id // empty')

if [ -z "$HUB_USER_ID" ]; then
    echo "  ❌ Hub user $MCP_USERNAME not found"
    exit 1
fi

echo "  ✅ Found hub user: $MCP_USERNAME (ID: $HUB_USER_ID)"

# For V1 cross-realm token exchange, we need a regular user in the managed realm
# with a federated identity link to the hub user. The sub claim from the hub token
# must match the userId in the federated identity link.

# Check if user already exists in managed realm
MANAGED_USERS=$(curl $CURL_OPTS -X GET "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/users?username=$MCP_USERNAME" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
MANAGED_USER_ID=$(echo "$MANAGED_USERS" | jq -r '.[0].id // empty')

if [ -z "$MANAGED_USER_ID" ] || [ "$MANAGED_USER_ID" = "null" ]; then
    # Create user in managed realm
    echo "  Creating user $MCP_USERNAME in managed realm..."
    USER_CREATE_RESPONSE=$(curl $CURL_OPTS -w "%{http_code}" -X POST "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/users" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{
            \"username\": \"$MCP_USERNAME\",
            \"enabled\": true,
            \"email\": \"$MCP_USERNAME@example.com\",
            \"firstName\": \"MCP\",
            \"lastName\": \"User\"
        }")
    USER_CREATE_CODE=$(echo "$USER_CREATE_RESPONSE" | tail -c 4)

    if [ "$USER_CREATE_CODE" = "201" ]; then
        echo "  ✅ User $MCP_USERNAME created in managed realm"
    else
        echo "  ⚠️  User creation returned HTTP $USER_CREATE_CODE"
    fi

    # Get the new user ID
    MANAGED_USERS=$(curl $CURL_OPTS -X GET "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/users?username=$MCP_USERNAME" \
        -H "Authorization: Bearer $ADMIN_TOKEN")
    MANAGED_USER_ID=$(echo "$MANAGED_USERS" | jq -r '.[0].id // empty')
else
    echo "  ✅ User $MCP_USERNAME already exists in managed realm (ID: $MANAGED_USER_ID)"
fi

if [ -z "$MANAGED_USER_ID" ] || [ "$MANAGED_USER_ID" = "null" ]; then
    echo "  ❌ Failed to get managed realm user ID"
    exit 1
fi

# Create federated identity link between managed user and hub user
# The userId here is the hub user's ID - this is how Keycloak matches the sub claim
FED_IDENTITY_JSON="{
  \"identityProvider\": \"$IDP_ALIAS\",
  \"userId\": \"$HUB_USER_ID\",
  \"userName\": \"$MCP_USERNAME\"
}"

FED_RESPONSE=$(curl $CURL_OPTS -w "%{http_code}" -X POST "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/users/$MANAGED_USER_ID/federated-identity/$IDP_ALIAS" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "$FED_IDENTITY_JSON")

FED_CODE=$(echo "$FED_RESPONSE" | tail -c 4)
if [ "$FED_CODE" = "204" ] || [ "$FED_CODE" = "409" ]; then
    echo "  ✅ Federated identity link created (hub: $MCP_USERNAME/$HUB_USER_ID → managed: $MCP_USERNAME/$MANAGED_USER_ID)"
else
    echo "  ⚠️  Federated identity link returned HTTP $FED_CODE"
fi

# Remove any federated identity link from service account user to prevent
# "More results found" error during token exchange. The mcp-server client's
# service account should NOT have a federated identity link.
echo "  Checking for service account federated identity..."
SA_USER=$(curl $CURL_OPTS -X GET "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/clients/$MANAGED_CLIENT_UUID/service-account-user" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
SA_USER_ID=$(echo "$SA_USER" | jq -r '.id // empty')

if [ -n "$SA_USER_ID" ] && [ "$SA_USER_ID" != "null" ]; then
    # Check if service account has a federated identity link
    SA_FED=$(curl $CURL_OPTS -X GET "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/users/$SA_USER_ID/federated-identity" \
        -H "Authorization: Bearer $ADMIN_TOKEN")
    SA_FED_IDP=$(echo "$SA_FED" | jq -r '.[] | select(.identityProvider == "'"$IDP_ALIAS"'") | .identityProvider // empty')

    if [ -n "$SA_FED_IDP" ]; then
        echo "  Removing federated identity from service account..."
        curl $CURL_OPTS -X DELETE "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/users/$SA_USER_ID/federated-identity/$IDP_ALIAS" \
            -H "Authorization: Bearer $ADMIN_TOKEN" > /dev/null
        echo "  ✅ Removed federated identity from service account"
    else
        echo "  ✅ Service account has no conflicting federated identity"
    fi
fi

# Clear Keycloak user cache to ensure federated identity changes take effect immediately
echo "  Clearing Keycloak user cache..."
curl $CURL_OPTS -X POST "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/clear-user-cache" \
    -H "Authorization: Bearer $ADMIN_TOKEN" > /dev/null
echo "  ✅ User cache cleared"
echo ""

# Step 9: Configure cross-realm token exchange permissions
echo "Step 9: Configuring cross-realm token exchange permissions..."

# Enable fine-grained permissions on IDP
curl $CURL_OPTS -X PUT "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/identity-provider/instances/$IDP_ALIAS/management/permissions" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"enabled": true}' > /dev/null

# Get IDP permissions
IDP_PERMS=$(curl $CURL_OPTS -X GET "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/identity-provider/instances/$IDP_ALIAS/management/permissions" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
TOKEN_EXCHANGE_PERM_ID=$(echo "$IDP_PERMS" | jq -r '.scopePermissions."token-exchange"')

# Get realm-management client ID
REALM_MGMT_ID=$(curl $CURL_OPTS -X GET "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/clients?clientId=realm-management" \
    -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[0].id')

# Create client policy for managed mcp-server
POLICY_RESPONSE=$(curl $CURL_OPTS -X POST "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/clients/$REALM_MGMT_ID/authz/resource-server/policy/client" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"type\": \"client\",
      \"logic\": \"POSITIVE\",
      \"decisionStrategy\": \"UNANIMOUS\",
      \"name\": \"allow-mcp-server-token-exchange\",
      \"description\": \"Allow hub realm mcp-server to exchange to managed cluster\",
      \"clients\": [\"$MANAGED_CLIENT_UUID\"]
    }" 2>/dev/null)

POLICY_ID=$(echo "$POLICY_RESPONSE" | jq -r '.id // empty')

# If policy creation failed, try to find existing policy
if [ -z "$POLICY_ID" ] || [ "$POLICY_ID" = "null" ]; then
    ALL_POLICIES=$(curl $CURL_OPTS -X GET "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/clients/$REALM_MGMT_ID/authz/resource-server/policy?type=client" \
        -H "Authorization: Bearer $ADMIN_TOKEN")
    POLICY_ID=$(echo "$ALL_POLICIES" | jq -r '.[] | select(.name == "allow-mcp-server-token-exchange") | .id')
fi

# Link policy to token-exchange permission
CURRENT_PERM=$(curl $CURL_OPTS -X GET "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/clients/$REALM_MGMT_ID/authz/resource-server/permission/$TOKEN_EXCHANGE_PERM_ID" \
    -H "Authorization: Bearer $ADMIN_TOKEN")

UPDATED_PERM=$(echo "$CURRENT_PERM" | jq --arg policy_id "$POLICY_ID" '. + {policies: [$policy_id]}')

curl $CURL_OPTS -X PUT "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/clients/$REALM_MGMT_ID/authz/resource-server/permission/$TOKEN_EXCHANGE_PERM_ID" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "$UPDATED_PERM" > /dev/null

echo "  ✅ Cross-realm token exchange configured"
echo ""

# Step 10: Save configuration
echo "Step 10: Saving configuration..."

mkdir -p "$CLUSTER_CONFIG_DIR"
cat > "$CLUSTER_CONFIG_DIR/$CLUSTER_NAME.env" <<EOF
# Managed Cluster Configuration: $CLUSTER_NAME
# Generated: $(date -Iseconds)

CLUSTER_NAME="$CLUSTER_NAME"
MANAGED_REALM="$MANAGED_REALM"
IDP_ALIAS="$IDP_ALIAS"
CLIENT_ID="mcp-server"
CLIENT_SECRET="$MANAGED_CLIENT_SECRET"
CLIENT_UUID="$MANAGED_CLIENT_UUID"

MCP_USERNAME="$MCP_USERNAME"
MCP_PASSWORD="$MCP_PASSWORD"

HUB_REALM="$HUB_REALM"
HUB_USER_ID="$HUB_USER_ID"
MANAGED_USER_ID="$MANAGED_USER_ID"
EOF

echo "  ✅ Configuration saved to $CLUSTER_CONFIG_DIR/$CLUSTER_NAME.env"
echo ""

#=============================================================================
# Configure OIDC Authentication on Managed Cluster
#=============================================================================
echo "========================================="
echo "Configuring OIDC on Managed Cluster"
echo "========================================="
echo ""

# Step 1: Enable TechPreviewNoUpgrade feature gate
echo "Step 1: Enabling TechPreviewNoUpgrade feature gate on managed cluster..."
CURRENT_FEATURE_SET=$(kubectl --kubeconfig="$MANAGED_KUBECONFIG" get featuregate cluster -o jsonpath='{.spec.featureSet}' 2>/dev/null || echo "")
if [ "$CURRENT_FEATURE_SET" != "TechPreviewNoUpgrade" ]; then
    kubectl --kubeconfig="$MANAGED_KUBECONFIG" patch featuregate cluster \
        --type=merge -p='{"spec":{"featureSet":"TechPreviewNoUpgrade"}}'
    echo "  ✅ TechPreviewNoUpgrade enabled"
    echo "  ⚠️  Control plane will restart (10-15 minutes)"
    echo "  ⚠️  Waiting 2 minutes for initial rollout..."
    sleep 120
else
    echo "  ✅ TechPreviewNoUpgrade already enabled"
fi

echo ""
echo "Waiting for kube-apiserver on managed cluster..."
for i in $(seq 1 30); do
    if kubectl --kubeconfig="$MANAGED_KUBECONFIG" wait --for=condition=Available --timeout=10s clusteroperator/kube-apiserver 2>/dev/null; then
        echo "  ✅ kube-apiserver is ready"
        break
    fi
    echo "  Waiting for kube-apiserver (attempt $i/30)..."
    sleep 10
done

# Step 2: Create Keycloak CA certificate ConfigMap
echo ""
echo "Step 2: Creating Keycloak CA certificate ConfigMap..."
KEYCLOAK_CA=$(kubectl --kubeconfig="$HUB_KUBECONFIG" get configmap router-ca -n keycloak \
    -o jsonpath='{.data.router-ca\.crt}' 2>/dev/null || \
    kubectl --kubeconfig="$HUB_KUBECONFIG" get configmap -n openshift-config-managed default-ingress-cert \
    -o jsonpath='{.data.ca-bundle\.crt}' 2>/dev/null)

if [ -z "$KEYCLOAK_CA" ]; then
    echo "  ⚠️  Could not extract Keycloak CA certificate"
    echo "  You may need to manually create ConfigMap: keycloak-oidc-ca in openshift-config"
else
    echo "$KEYCLOAK_CA" | kubectl --kubeconfig="$MANAGED_KUBECONFIG" create configmap keycloak-oidc-ca \
        -n openshift-config --from-file=ca-bundle.crt=/dev/stdin --dry-run=client -o yaml | \
        kubectl --kubeconfig="$MANAGED_KUBECONFIG" apply -f -
    echo "  ✅ CA certificate ConfigMap created"
fi

# Step 3: Create RBAC for mcp user (the user from token exchange)
echo ""
echo "Step 3: Creating RBAC for $MCP_USERNAME user..."
kubectl --kubeconfig="$MANAGED_KUBECONFIG" create clusterrolebinding mcp-user-admin \
    --clusterrole=cluster-admin --user="$MCP_USERNAME" \
    --dry-run=client -o yaml | kubectl --kubeconfig="$MANAGED_KUBECONFIG" apply -f -
echo "  ✅ RBAC created for $MCP_USERNAME"

# Step 4: Configure OIDC provider
echo ""
echo "Step 4: Configuring OIDC provider..."
ISSUER_URL="$KEYCLOAK_URL/realms/$MANAGED_REALM"

CURRENT_ISSUER=$(kubectl --kubeconfig="$MANAGED_KUBECONFIG" get authentication.config.openshift.io/cluster \
    -o jsonpath='{.spec.oidcProviders[0].issuer.issuerURL}' 2>/dev/null || echo "")

if [ "$CURRENT_ISSUER" = "$ISSUER_URL" ]; then
    echo "  ✅ OIDC provider already configured"
else
    if [ -n "$CURRENT_ISSUER" ]; then
        echo "  Updating existing OIDC provider..."
        printf '[{"op":"replace","path":"/spec/oidcProviders/0/issuer/issuerURL","value":"%s"},{"op":"replace","path":"/spec/oidcProviders/0/issuer/audiences","value":["mcp-server"]}]' "$ISSUER_URL" > /tmp/oidc-patch-$CLUSTER_NAME.json
    else
        echo "  Creating new OIDC provider..."
        printf '[{"op":"remove","path":"/spec/webhookTokenAuthenticator"},{"op":"replace","path":"/spec/type","value":"OIDC"},{"op":"add","path":"/spec/oidcProviders","value":[{"name":"keycloak","issuer":{"issuerURL":"%s","audiences":["mcp-server"],"issuerCertificateAuthority":{"name":"keycloak-oidc-ca"}},"claimMappings":{"username":{"claim":"preferred_username","prefixPolicy":"NoPrefix"}}}]}]' "$ISSUER_URL" > /tmp/oidc-patch-$CLUSTER_NAME.json
    fi

    kubectl --kubeconfig="$MANAGED_KUBECONFIG" patch authentication.config.openshift.io/cluster \
        --type=json -p="$(cat /tmp/oidc-patch-$CLUSTER_NAME.json)"
    echo "  ✅ OIDC provider configured"
    echo ""
    echo "  Verifying kube-apiserver operator picked up OIDC configuration..."

    # Get current revision before verification
    BEFORE_REV=$(kubectl --kubeconfig="$MANAGED_KUBECONFIG" get kubeapiserver cluster \
        -o jsonpath='{.status.latestAvailableRevision}' 2>/dev/null || echo "0")

    # Wait up to 2 minutes for a new revision to be created
    echo "  Waiting for new kube-apiserver revision to be created..."
    for i in $(seq 1 24); do
        sleep 5
        CURRENT_REV=$(kubectl --kubeconfig="$MANAGED_KUBECONFIG" get kubeapiserver cluster \
            -o jsonpath='{.status.latestAvailableRevision}' 2>/dev/null || echo "0")
        if [ "$CURRENT_REV" -gt "$BEFORE_REV" ]; then
            echo "  ✅ New revision $CURRENT_REV created"
            break
        fi
        if [ $i -eq 24 ]; then
            echo "  ⚠️  No new revision created after 2 minutes"
            echo "  Applying workaround: remove/re-add OIDC provider to force reconciliation..."

            # Remove OIDC provider
            printf '[{"op":"replace","path":"/spec/oidcProviders","value":[]}]' > /tmp/oidc-remove-$CLUSTER_NAME.json
            kubectl --kubeconfig="$MANAGED_KUBECONFIG" patch authentication.config.openshift.io/cluster \
                --type=json -p="$(cat /tmp/oidc-remove-$CLUSTER_NAME.json)"
            sleep 10

            # Re-add OIDC provider
            kubectl --kubeconfig="$MANAGED_KUBECONFIG" patch authentication.config.openshift.io/cluster \
                --type=json -p="$(cat /tmp/oidc-patch-$CLUSTER_NAME.json)"
            echo "  ✅ OIDC provider re-applied"

            # Wait for new revision again
            for j in $(seq 1 12); do
                sleep 5
                CURRENT_REV=$(kubectl --kubeconfig="$MANAGED_KUBECONFIG" get kubeapiserver cluster \
                    -o jsonpath='{.status.latestAvailableRevision}' 2>/dev/null || echo "0")
                if [ "$CURRENT_REV" -gt "$BEFORE_REV" ]; then
                    echo "  ✅ New revision $CURRENT_REV created after workaround"
                    break
                fi
            done
        fi
    done

    echo ""
    echo "  ⚠️  IMPORTANT: kube-apiserver will now roll out with OIDC configuration"
    echo "  This takes 10-15 minutes as each master node updates sequentially."
    echo ""
    echo "  You can monitor the rollout with:"
    echo "    kubectl --kubeconfig=$MANAGED_KUBECONFIG get co kube-apiserver -w"
    echo ""
    echo "  Wait until: Available=True, Progressing=False, Degraded=False"
fi

echo ""
echo "========================================="
echo "✅ Managed Cluster Registration Complete!"
echo "========================================="
echo ""
echo "Cluster: $CLUSTER_NAME"
echo "Managed Realm: $MANAGED_REALM"
echo "Identity Provider: $IDP_ALIAS (hub realm)"
echo "Federated User: $MCP_USERNAME (hub: $HUB_USER_ID → managed: $MANAGED_USER_ID)"
echo "ACM ManagedCluster: Created"
echo "ACM Import Manifests: Applied"
echo "Cluster-Proxy Agents: Starting"
echo "Cross-Realm Exchange: Configured"
echo "OIDC Authentication: Configured (rolling out)"
echo ""
echo "Configuration saved to: $CLUSTER_CONFIG_DIR/$CLUSTER_NAME.env"
echo ""
echo "⚠️  IMPORTANT: Wait for rollouts to complete:"
echo "  1. Feature gate rollout: ~10-15 minutes"
echo "  2. OIDC rollout: ~10-15 minutes"
echo "  3. Cluster-proxy agents: ~2-5 minutes"
echo "  Total: ~25-30 minutes"
echo ""
echo "Monitor rollout status:"
echo "  kubectl --kubeconfig=$MANAGED_KUBECONFIG get co kube-apiserver -w"
echo "  kubectl --kubeconfig=$MANAGED_KUBECONFIG get pods -n open-cluster-management-agent-addon -w"
echo ""
echo "After rollout completes:"
echo "  1. Run: make keycloak-acm-generate-toml"
echo "  2. Start MCP server: ./kubernetes-mcp-server --port 8080 --config _output/acm-kubeconfig.toml"
echo ""
