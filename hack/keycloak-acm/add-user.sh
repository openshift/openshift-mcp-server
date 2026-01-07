#!/bin/bash
# Add a new user to Keycloak realms
#
# Two modes of operation:
#   1. Federated mode (default): Creates user in hub realm and federates to all managed clusters
#   2. Standalone mode (CLUSTER_NAME set): Creates standalone user in a specific cluster realm only
#
# Required environment variables:
#   KEYCLOAK_USER      - Username for the new user
#   KEYCLOAK_PASS      - Password for the new user
#
# Optional environment variables:
#   CLUSTER_NAME       - If set, creates standalone user in this cluster's realm only (no federation)
#   KEYCLOAK_USER_ROLE - ClusterRole to bind (default: cluster-admin)
#   KEYCLOAK_CA_CERT   - Path to CA certificate for HTTPS verification
#
# Examples:
#   # Federated user (hub + all managed clusters)
#   KEYCLOAK_USER=alice KEYCLOAK_PASS=secret ./add-user.sh
#
#   # Standalone user for specific cluster (like kubeadmin)
#   CLUSTER_NAME=managed-cluster-one KEYCLOAK_USER=kubeadmin KEYCLOAK_PASS=secret ./add-user.sh

set -euo pipefail

# Validate required variables
: "${KEYCLOAK_USER:?Error: KEYCLOAK_USER environment variable is required}"
: "${KEYCLOAK_PASS:?Error: KEYCLOAK_PASS environment variable is required}"

# Optional variables with defaults
KEYCLOAK_USER_ROLE="${KEYCLOAK_USER_ROLE:-cluster-admin}"
CLUSTER_NAME="${CLUSTER_NAME:-}"

# Get script directory and repo root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
HUB_CONFIG_ENV="$REPO_ROOT/.keycloak-config/hub-config.env"
CLUSTER_CONFIG_DIR="$REPO_ROOT/.keycloak-config/clusters"

# Load hub configuration
if [ ! -f "$HUB_CONFIG_ENV" ]; then
    echo "Error: Hub configuration not found at $HUB_CONFIG_ENV"
    echo "Please run 'make keycloak-acm-setup-hub' first"
    exit 1
fi

source "$HUB_CONFIG_ENV"

# Set curl options based on CA cert availability
CURL_OPTS="-sk"
if [ -n "${KEYCLOAK_CA_CERT:-}" ]; then
    CURL_OPTS="--cacert $KEYCLOAK_CA_CERT"
fi

#=============================================================================
# Helper function to create user in a realm
#=============================================================================
create_user_in_realm() {
    local realm="$1"
    local username="$2"
    local password="$3"
    local set_password="${4:-true}"

    # Check if user already exists
    local existing_user
    existing_user=$(curl $CURL_OPTS -X GET "$KEYCLOAK_URL/admin/realms/$realm/users?username=$username&exact=true" \
        -H "Authorization: Bearer $ADMIN_TOKEN")

    local user_id
    user_id=$(echo "$existing_user" | jq -r '.[0].id // empty')

    if [ -n "$user_id" ] && [ "$user_id" != "null" ]; then
        echo "  User already exists in realm $realm (ID: $user_id)"
        echo "$user_id"
        return 0
    fi

    # Create user
    local user_create_response
    user_create_response=$(curl $CURL_OPTS -w "%{http_code}" -X POST "$KEYCLOAK_URL/admin/realms/$realm/users" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{
            \"username\": \"$username\",
            \"enabled\": true,
            \"emailVerified\": true,
            \"email\": \"$username@example.com\",
            \"firstName\": \"$username\",
            \"lastName\": \"User\",
            \"requiredActions\": []
        }")

    local user_create_code
    user_create_code=$(echo "$user_create_response" | tail -c 4)

    if [ "$user_create_code" != "201" ]; then
        echo "  Error: Failed to create user in realm $realm (HTTP $user_create_code)" >&2
        return 1
    fi

    # Get the new user's ID
    existing_user=$(curl $CURL_OPTS -X GET "$KEYCLOAK_URL/admin/realms/$realm/users?username=$username&exact=true" \
        -H "Authorization: Bearer $ADMIN_TOKEN")

    user_id=$(echo "$existing_user" | jq -r '.[0].id')

    # Set password if requested
    if [ "$set_password" = "true" ]; then
        curl $CURL_OPTS -X PUT "$KEYCLOAK_URL/admin/realms/$realm/users/$user_id/reset-password" \
            -H "Authorization: Bearer $ADMIN_TOKEN" \
            -H "Content-Type: application/json" \
            -d "{
                \"type\": \"password\",
                \"value\": \"$password\",
                \"temporary\": false
            }" > /dev/null
    fi

    echo "  User created in realm $realm (ID: $user_id)"
    echo "$user_id"
}

#=============================================================================
# Helper function to create RBAC
#=============================================================================
create_rbac() {
    local kubeconfig="$1"
    local username="$2"
    local role="$3"

    local kubectl_opts=""
    if [ -n "$kubeconfig" ]; then
        kubectl_opts="--kubeconfig=$kubeconfig"
    fi

    kubectl $kubectl_opts apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: keycloak-user-$username
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: $role
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: User
  name: $username
EOF
}

#=============================================================================
# Get Keycloak Admin Token
#=============================================================================
echo "=========================================="
if [ -n "$CLUSTER_NAME" ]; then
    echo "Add Standalone Cluster User"
else
    echo "Add Federated Keycloak User"
fi
echo "=========================================="
echo "Username: $KEYCLOAK_USER"
echo "ClusterRole: $KEYCLOAK_USER_ROLE"
if [ -n "$CLUSTER_NAME" ]; then
    echo "Target Cluster: $CLUSTER_NAME (standalone, no federation)"
else
    echo "Mode: Federated (hub + all managed clusters)"
fi
echo "Keycloak URL: $KEYCLOAK_URL"
echo ""

echo "Getting Keycloak admin token..."

RESPONSE=$(curl $CURL_OPTS -X POST "$KEYCLOAK_URL/realms/master/protocol/openid-connect/token" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "username=$ADMIN_USER" \
    -d "password=$ADMIN_PASSWORD" \
    -d "grant_type=password" \
    -d "client_id=admin-cli")

ADMIN_TOKEN=$(echo "$RESPONSE" | jq -r '.access_token // empty' 2>/dev/null)

if [ -z "$ADMIN_TOKEN" ] || [ "$ADMIN_TOKEN" = "null" ]; then
    echo "Error: Failed to get admin token"
    echo "Response: $RESPONSE"
    exit 1
fi

echo "  Admin token obtained"
echo ""

#=============================================================================
# Mode: Standalone Cluster User
#=============================================================================
if [ -n "$CLUSTER_NAME" ]; then
    # Find the cluster configuration
    CLUSTER_CONFIG="$CLUSTER_CONFIG_DIR/$CLUSTER_NAME.env"

    # Determine the target realm
    if [ "$CLUSTER_NAME" = "hub" ]; then
        # Special case: create standalone user in hub realm
        TARGET_REALM="$HUB_REALM"
        MANAGED_KUBECONFIG=""
    elif [ -f "$CLUSTER_CONFIG" ]; then
        source "$CLUSTER_CONFIG"
        TARGET_REALM="$MANAGED_REALM"
    else
        echo "Error: Cluster configuration not found: $CLUSTER_CONFIG"
        echo ""
        echo "Available clusters:"
        echo "  - hub (hub realm)"
        if [ -d "$CLUSTER_CONFIG_DIR" ]; then
            for env_file in "$CLUSTER_CONFIG_DIR"/*.env; do
                if [ -f "$env_file" ]; then
                    basename "$env_file" .env | sed 's/^/  - /'
                fi
            done
        fi
        exit 1
    fi

    echo "Creating standalone user in realm: $TARGET_REALM"
    echo ""

    # Create user with password in target realm
    USER_ID=$(create_user_in_realm "$TARGET_REALM" "$KEYCLOAK_USER" "$KEYCLOAK_PASS" "true")

    # Clear user cache
    curl $CURL_OPTS -X POST "$KEYCLOAK_URL/admin/realms/$TARGET_REALM/clear-user-cache" \
        -H "Authorization: Bearer $ADMIN_TOKEN" > /dev/null 2>&1
    echo "  User cache cleared"
    echo ""

    # Create RBAC
    echo "Creating RBAC..."
    if [ "$CLUSTER_NAME" = "hub" ]; then
        create_rbac "" "$KEYCLOAK_USER" "$KEYCLOAK_USER_ROLE"
        echo "  ClusterRoleBinding created on hub cluster"
    elif [ -n "${MANAGED_KUBECONFIG:-}" ] && [ -f "${MANAGED_KUBECONFIG:-}" ]; then
        create_rbac "$MANAGED_KUBECONFIG" "$KEYCLOAK_USER" "$KEYCLOAK_USER_ROLE"
        echo "  ClusterRoleBinding created on $CLUSTER_NAME"
    else
        echo "  Note: Managed cluster kubeconfig not found, skipping RBAC creation"
        echo "  You may need to create RBAC manually on cluster: $CLUSTER_NAME"
    fi
    echo ""

    # Summary
    echo "=========================================="
    echo "Standalone User Added Successfully"
    echo "=========================================="
    echo ""
    echo "Username: $KEYCLOAK_USER"
    echo "Password: $KEYCLOAK_PASS"
    echo "Realm: $TARGET_REALM"
    echo "ClusterRole: $KEYCLOAK_USER_ROLE"
    echo ""
    echo "This is a standalone user for cluster '$CLUSTER_NAME' only."
    echo "The user can authenticate directly to this cluster's realm."
    echo ""

    exit 0
fi

#=============================================================================
# Mode: Federated User (Hub + All Managed Clusters)
#=============================================================================

echo "Creating user in hub realm ($HUB_REALM)..."
NEW_HUB_USER_ID=$(create_user_in_realm "$HUB_REALM" "$KEYCLOAK_USER" "$KEYCLOAK_PASS" "true")
echo ""

# Create RBAC on Hub Cluster
echo "Creating RBAC for user on hub cluster..."
create_rbac "" "$KEYCLOAK_USER" "$KEYCLOAK_USER_ROLE"
echo "  ClusterRoleBinding created (role: $KEYCLOAK_USER_ROLE)"
echo ""

# Federate User to Managed Clusters
if [ -d "$CLUSTER_CONFIG_DIR" ] && [ "$(ls -A $CLUSTER_CONFIG_DIR/*.env 2>/dev/null)" ]; then
    echo "Federating user to managed clusters..."
    echo ""

    for cluster_env in "$CLUSTER_CONFIG_DIR"/*.env; do
        if [ ! -f "$cluster_env" ]; then
            continue
        fi

        # Source cluster config (provides CLUSTER_NAME, MANAGED_REALM, IDP_ALIAS)
        source "$cluster_env"

        echo "  Processing cluster: $CLUSTER_NAME (realm: $MANAGED_REALM)"

        # Create user in managed realm (without password - will use federation)
        MANAGED_USER_ID=$(create_user_in_realm "$MANAGED_REALM" "$KEYCLOAK_USER" "" "false" 2>/dev/null | tail -1)

        if [ -n "$MANAGED_USER_ID" ] && [ "$MANAGED_USER_ID" != "null" ]; then
            # Create federated identity link
            # Use NEW_HUB_USER_ID (not HUB_USER_ID which gets overwritten by sourcing cluster config)
            FED_IDENTITY_JSON="{
                \"identityProvider\": \"$IDP_ALIAS\",
                \"userId\": \"$NEW_HUB_USER_ID\",
                \"userName\": \"$KEYCLOAK_USER\"
            }"

            FED_RESPONSE=$(curl $CURL_OPTS -w "%{http_code}" -X POST "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/users/$MANAGED_USER_ID/federated-identity/$IDP_ALIAS" \
                -H "Authorization: Bearer $ADMIN_TOKEN" \
                -H "Content-Type: application/json" \
                -d "$FED_IDENTITY_JSON")

            FED_CODE=$(echo "$FED_RESPONSE" | tail -c 4)

            if [ "$FED_CODE" = "204" ]; then
                echo "    Federated identity link created"
            elif [ "$FED_CODE" = "409" ]; then
                echo "    Federated identity link already exists"
            else
                echo "    Warning: Federated identity link returned HTTP $FED_CODE"
            fi

            # Clear user cache
            curl $CURL_OPTS -X POST "$KEYCLOAK_URL/admin/realms/$MANAGED_REALM/clear-user-cache" \
                -H "Authorization: Bearer $ADMIN_TOKEN" > /dev/null 2>&1

            echo "    User cache cleared"
        fi

        # Create RBAC on managed cluster if kubeconfig is available
        if [ -n "${MANAGED_KUBECONFIG:-}" ] && [ -f "${MANAGED_KUBECONFIG:-}" ]; then
            echo "    Creating RBAC on managed cluster..."
            create_rbac "$MANAGED_KUBECONFIG" "$KEYCLOAK_USER" "$KEYCLOAK_USER_ROLE"
            echo "    ClusterRoleBinding created on managed cluster"
        else
            echo "    Note: Managed cluster kubeconfig not found, skipping RBAC creation"
            echo "    You may need to create RBAC manually on cluster: $CLUSTER_NAME"
        fi

        echo ""
    done
else
    echo "No managed clusters registered."
    echo "To register managed clusters, run:"
    echo "  make keycloak-acm-register-managed-cluster CLUSTER_NAME=<name> MANAGED_KUBECONFIG=<path>"
    echo ""
fi

# Summary
echo "=========================================="
echo "Federated User Added Successfully"
echo "=========================================="
echo ""
echo "Username: $KEYCLOAK_USER"
echo "Password: $KEYCLOAK_PASS"
echo "ClusterRole: $KEYCLOAK_USER_ROLE"
echo ""
echo "Hub Realm User ID: $NEW_HUB_USER_ID"
echo ""
echo "The user can authenticate via the hub Keycloak realm and access"
echo "all federated clusters based on the assigned ClusterRole ($KEYCLOAK_USER_ROLE)."
echo ""
