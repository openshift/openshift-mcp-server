#!/bin/bash
# Setup Keycloak instance and hub realm for ACM multi-cluster
# This script:
# 1. Deploys Keycloak (PostgreSQL + Keycloak instance)
# 2. Configures OpenShift Authentication CR to use Keycloak as OIDC provider
# 3. Creates hub realm with all V1 token exchange requirements

set -e

# Configuration
HUB_REALM="${HUB_REALM:-hub}"
CLIENT_ID="${CLIENT_ID:-mcp-server}"
MCP_USERNAME="${MCP_USERNAME:-mcp}"
MCP_PASSWORD="${MCP_PASSWORD:-mcp}"
KEYCLOAK_VERSION="${KEYCLOAK_VERSION:-26.4}"
KEYCLOAK_NAMESPACE="${KEYCLOAK_NAMESPACE:-keycloak}"

# Keycloak admin credentials (consistent with deployment)
ADMIN_USER="admin"
ADMIN_PASSWORD="admin"

echo "=========================================="
echo "Keycloak Hub Setup for ACM"
echo "=========================================="
echo "Hub Realm: $HUB_REALM"
echo "Client: $CLIENT_ID"
echo "Keycloak Version: $KEYCLOAK_VERSION"
echo ""
echo "This script will:"
echo "  1. Deploy Keycloak (PostgreSQL + Keycloak instance)"
echo "  2. Configure OpenShift Authentication CR"
echo "  3. Create hub realm with V1 token exchange support"
echo ""

#=============================================================================
# STEP 1: Deploy Keycloak
#=============================================================================
echo "========================================"
echo "STEP 1: Deploying Keycloak"
echo "========================================"
echo ""

# Create namespace
echo "Creating namespace..."
if oc get namespace "$KEYCLOAK_NAMESPACE" >/dev/null 2>&1; then
    echo "✅ Namespace $KEYCLOAK_NAMESPACE already exists"
else
    oc create namespace "$KEYCLOAK_NAMESPACE"
    echo "✅ Namespace $KEYCLOAK_NAMESPACE created"
fi

# Deploy PostgreSQL
echo ""
echo "Deploying PostgreSQL..."

# Check if PostgreSQL secret already exists
if oc get secret postgresql-credentials -n "$KEYCLOAK_NAMESPACE" >/dev/null 2>&1; then
    echo "  Using existing PostgreSQL credentials"
    POSTGRESQL_PASSWORD=$(oc get secret postgresql-credentials -n "$KEYCLOAK_NAMESPACE" -o jsonpath='{.data.POSTGRESQL_PASSWORD}' | base64 -d)
else
    echo "  Generating new PostgreSQL credentials"
    POSTGRESQL_PASSWORD="$(openssl rand -base64 24 | tr -d '=+/' | cut -c1-24)"
fi

cat <<EOF | oc apply -n "$KEYCLOAK_NAMESPACE" -f -
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: postgresql-data
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
---
apiVersion: v1
kind: Secret
metadata:
  name: postgresql-credentials
type: Opaque
stringData:
  POSTGRESQL_DATABASE: keycloak
  POSTGRESQL_USER: keycloak
  POSTGRESQL_PASSWORD: $POSTGRESQL_PASSWORD
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgresql
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgresql
  template:
    metadata:
      labels:
        app: postgresql
    spec:
      containers:
      - name: postgresql
        image: registry.redhat.io/rhel9/postgresql-16:latest
        ports:
        - containerPort: 5432
          name: postgresql
        envFrom:
        - secretRef:
            name: postgresql-credentials
        volumeMounts:
        - name: postgresql-data
          mountPath: /var/lib/pgsql/data
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          tcpSocket:
            port: 5432
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          exec:
            command:
            - /bin/sh
            - -c
            - pg_isready -U keycloak
          initialDelaySeconds: 10
          periodSeconds: 5
      volumes:
      - name: postgresql-data
        persistentVolumeClaim:
          claimName: postgresql-data
---
apiVersion: v1
kind: Service
metadata:
  name: postgresql
spec:
  ports:
  - port: 5432
    targetPort: 5432
    name: postgresql
  selector:
    app: postgresql
  type: ClusterIP
EOF

echo "✅ PostgreSQL deployment created"

echo ""
echo "Waiting for PostgreSQL to be ready..."
oc wait --for=condition=ready pod -l app=postgresql -n "$KEYCLOAK_NAMESPACE" --timeout=300s
echo "✅ PostgreSQL is ready"

# Deploy Keycloak
echo ""
echo "Deploying Keycloak with V1 features enabled..."

cat <<EOF | oc apply -n "$KEYCLOAK_NAMESPACE" -f -
apiVersion: v1
kind: Service
metadata:
  name: keycloak
  labels:
    app: keycloak
spec:
  ports:
  - name: https
    port: 8443
    targetPort: 8443
  - name: http
    port: 8080
    targetPort: 8080
  selector:
    app: keycloak
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: keycloak
  labels:
    app: keycloak
spec:
  replicas: 1
  selector:
    matchLabels:
      app: keycloak
  template:
    metadata:
      labels:
        app: keycloak
    spec:
      containers:
      - name: keycloak
        image: quay.io/keycloak/keycloak:$KEYCLOAK_VERSION
        args:
        - start-dev
        - --features=token-exchange:v1,admin-fine-grained-authz:v1
        env:
        - name: KC_DB
          value: postgres
        - name: KC_DB_URL
          value: jdbc:postgresql://postgresql:5432/keycloak
        - name: KC_DB_USERNAME
          valueFrom:
            secretKeyRef:
              name: postgresql-credentials
              key: POSTGRESQL_USER
        - name: KC_DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgresql-credentials
              key: POSTGRESQL_PASSWORD
        - name: KC_BOOTSTRAP_ADMIN_USERNAME
          value: "$ADMIN_USER"
        - name: KC_BOOTSTRAP_ADMIN_PASSWORD
          value: "$ADMIN_PASSWORD"
        - name: KC_PROXY_HEADERS
          value: "xforwarded"
        - name: KC_HTTP_ENABLED
          value: "true"
        - name: KC_HOSTNAME_STRICT
          value: "false"
        ports:
        - name: http
          containerPort: 8080
        - name: https
          containerPort: 8443
        readinessProbe:
          httpGet:
            path: /realms/master
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        livenessProbe:
          httpGet:
            path: /realms/master
            port: 8080
          initialDelaySeconds: 60
          periodSeconds: 30
---
apiVersion: route.openshift.io/v1
kind: Route
metadata:
  name: keycloak
  labels:
    app: keycloak
spec:
  to:
    kind: Service
    name: keycloak
  port:
    targetPort: http
  tls:
    termination: edge
    insecureEdgeTerminationPolicy: Redirect
EOF

echo "✅ Keycloak deployment created with V1 features: token-exchange:v1,admin-fine-grained-authz:v1"

echo ""
echo "Waiting for Keycloak pod to be ready..."
oc wait --for=condition=ready pod -l app=keycloak -n "$KEYCLOAK_NAMESPACE" --timeout=300s
echo "✅ Keycloak pod is ready"

# Get Keycloak URL
echo ""
echo "Getting Keycloak route..."
KEYCLOAK_ROUTE=$(oc get route keycloak -n "$KEYCLOAK_NAMESPACE" -o jsonpath='{.spec.host}')
KEYCLOAK_URL="https://$KEYCLOAK_ROUTE"
echo "✅ Keycloak URL: $KEYCLOAK_URL"

# Wait for Keycloak HTTP endpoint
echo ""
echo "Waiting for Keycloak HTTP endpoint..."
for i in $(seq 1 30); do
    STATUS=$(curl -sk -o /dev/null -w "%{http_code}" "$KEYCLOAK_URL/realms/master" 2>/dev/null || echo "000")
    if [ "$STATUS" = "200" ]; then
        echo "✅ Keycloak HTTP endpoint ready"
        break
    fi
    echo "  Attempt $i/30: Waiting (status: $STATUS)..."
    sleep 5
done

if [ "$STATUS" != "200" ]; then
    echo "❌ Keycloak endpoint not responding"
    exit 1
fi

#=============================================================================
# STEP 2: Configure OpenShift Authentication CR
#=============================================================================
echo ""
echo "========================================"
echo "STEP 2: Configuring OpenShift Authentication"
echo "========================================"
echo ""

echo "Enabling TechPreviewNoUpgrade feature gate..."
CURRENT_FEATURE_SET=$(oc get featuregate cluster -o jsonpath='{.spec.featureSet}' 2>/dev/null || echo "")
if [ "$CURRENT_FEATURE_SET" != "TechPreviewNoUpgrade" ]; then
    echo "  Enabling TechPreviewNoUpgrade..."
    oc patch featuregate cluster --type=merge -p='{"spec":{"featureSet":"TechPreviewNoUpgrade"}}'
    echo "  ✅ Feature gate enabled"
    echo "  ⚠️  Control plane will restart (10-15 minutes)"
    echo "  ⚠️  Waiting 2 minutes for initial rollout..."
    sleep 120
else
    echo "  ✅ TechPreviewNoUpgrade already enabled"
fi

echo ""
echo "Waiting for kube-apiserver..."
for i in $(seq 1 30); do
    if oc wait --for=condition=Available --timeout=10s clusteroperator/kube-apiserver 2>/dev/null; then
        echo "  ✅ kube-apiserver is ready"
        break
    fi
    echo "  Waiting for kube-apiserver (attempt $i/30)..."
    sleep 10
done

echo ""
echo "Configuring OIDC provider CA certificate..."
kubectl get configmap -n openshift-config-managed default-ingress-cert -o jsonpath='{.data.ca-bundle\.crt}' > /tmp/keycloak-ca.crt
echo "  Extracted OpenShift ingress CA ($(wc -l < /tmp/keycloak-ca.crt) lines)"
oc delete configmap keycloak-oidc-ca -n openshift-config 2>/dev/null || true
oc create configmap keycloak-oidc-ca -n openshift-config --from-file=ca-bundle.crt=/tmp/keycloak-ca.crt
echo "  ✅ CA certificate configmap created"

echo ""
echo "Configuring OIDC provider..."
ISSUER_URL="$KEYCLOAK_URL/realms/$HUB_REALM"
echo "  Issuer URL: $ISSUER_URL"
echo "  Audiences: openshift, $CLIENT_ID"

CURRENT_ISSUER=$(oc get authentication.config.openshift.io/cluster -o jsonpath='{.spec.oidcProviders[0].issuer.issuerURL}' 2>/dev/null || echo "")
if [ "$CURRENT_ISSUER" = "$ISSUER_URL" ]; then
    echo "  ✅ OIDC provider already configured"
else
    if [ -n "$CURRENT_ISSUER" ]; then
        echo "  Updating existing OIDC provider..."
        printf '[{"op":"replace","path":"/spec/oidcProviders/0/issuer/issuerURL","value":"%s"},{"op":"replace","path":"/spec/oidcProviders/0/issuer/audiences","value":["openshift","%s"]}]' "$ISSUER_URL" "$CLIENT_ID" > /tmp/oidc-patch.json
    else
        echo "  Creating new OIDC provider..."
        printf '[{"op":"remove","path":"/spec/webhookTokenAuthenticator"},{"op":"replace","path":"/spec/type","value":"OIDC"},{"op":"add","path":"/spec/oidcProviders","value":[{"name":"keycloak","issuer":{"issuerURL":"%s","audiences":["openshift","%s"],"issuerCertificateAuthority":{"name":"keycloak-oidc-ca"}},"claimMappings":{"username":{"claim":"preferred_username","prefixPolicy":"NoPrefix"}}}]}]' "$ISSUER_URL" "$CLIENT_ID" > /tmp/oidc-patch.json
    fi
    oc patch authentication.config.openshift.io/cluster --type=json -p="$(cat /tmp/oidc-patch.json)"
    echo "  ✅ Authentication CR configured"
    echo ""
    echo "  ⚠️  IMPORTANT: kube-apiserver will now roll out with OIDC configuration"
    echo "  This takes 10-15 minutes as each master node updates sequentially."
    echo ""
    echo "  You can monitor the rollout with:"
    echo "    oc get co kube-apiserver -w"
    echo ""
    echo "  The MCP server will not be able to authenticate until the rollout completes."
    echo "  Wait until all conditions show: Available=True, Progressing=False, Degraded=False"
    echo ""
fi

#=============================================================================
# STEP 3: Create Hub Realm
#=============================================================================
echo ""
echo "========================================"
echo "STEP 3: Creating Hub Realm"
echo "========================================"
echo ""

# Get admin token
echo "Getting admin token..."
ADMIN_TOKEN=$(curl -sk -X POST "$KEYCLOAK_URL/realms/master/protocol/openid-connect/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "username=$ADMIN_USER" \
  -d "password=$ADMIN_PASSWORD" \
  -d "grant_type=password" \
  -d "client_id=admin-cli" | jq -r '.access_token')

if [ -z "$ADMIN_TOKEN" ] || [ "$ADMIN_TOKEN" = "null" ]; then
  echo "❌ Failed to get admin token"
  exit 1
fi
echo "✅ Got admin token"

# Create hub realm
echo ""
echo "Creating hub realm..."
EXISTING_REALM=$(curl -sk -X GET "$KEYCLOAK_URL/admin/realms/$HUB_REALM" \
  -H "Authorization: Bearer $ADMIN_TOKEN" 2>/dev/null)

# Check if realm exists by looking for the .realm field (not just valid JSON)
if echo "$EXISTING_REALM" | jq -e '.realm' > /dev/null 2>&1; then
  echo "  ✅ Hub realm already exists: $HUB_REALM"
else
  echo "  Creating hub realm..."
  curl -sk -X POST "$KEYCLOAK_URL/admin/realms" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"realm\": \"$HUB_REALM\",
      \"enabled\": true,
      \"displayName\": \"Hub Cluster Realm\",
      \"accessTokenLifespan\": 3600
    }" > /dev/null
  echo "  ✅ Created hub realm: $HUB_REALM"
fi

# Create client scopes (openid and mcp-server)
echo ""
echo "Creating client scopes..."

# Check if client-scopes endpoint is ready
SCOPES_RESPONSE=$(curl -sk -X GET "$KEYCLOAK_URL/admin/realms/$HUB_REALM/client-scopes" \
  -H "Authorization: Bearer $ADMIN_TOKEN")

# Check if response is valid JSON array (not an error object)
# We check if it's an array by attempting to get its length
SCOPES_COUNT=$(echo "$SCOPES_RESPONSE" | jq 'if type == "array" then length else -1 end' 2>/dev/null || echo "-1")

if [ "$SCOPES_COUNT" = "-1" ]; then
  echo "  ⚠️  Realm may not be fully ready, waiting 5 seconds..."
  sleep 5
  SCOPES_RESPONSE=$(curl -sk -X GET "$KEYCLOAK_URL/admin/realms/$HUB_REALM/client-scopes" \
    -H "Authorization: Bearer $ADMIN_TOKEN")

  # Check again after retry
  SCOPES_COUNT=$(echo "$SCOPES_RESPONSE" | jq 'if type == "array" then length else -1 end' 2>/dev/null || echo "-1")
  if [ "$SCOPES_COUNT" = "-1" ]; then
    echo "  ❌ Failed to get client scopes from Keycloak"
    echo "  Response: $SCOPES_RESPONSE"
    exit 1
  fi
fi

# Create openid scope
OPENID_SCOPE_UUID=$(echo "$SCOPES_RESPONSE" | jq -r '.[] | select(.name == "openid") | .id // empty')

if [ -z "$OPENID_SCOPE_UUID" ] || [ "$OPENID_SCOPE_UUID" = "null" ]; then
  curl -sk -X POST "$KEYCLOAK_URL/admin/realms/$HUB_REALM/client-scopes" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
      "name": "openid",
      "description": "OpenID Connect scope",
      "protocol": "openid-connect",
      "attributes": {
        "include.in.token.scope": "true",
        "display.on.consent.screen": "false"
      }
    }' > /dev/null

  # Wait a moment for scope to be created, then fetch UUID
  sleep 2
  SCOPES_RESPONSE=$(curl -sk -X GET "$KEYCLOAK_URL/admin/realms/$HUB_REALM/client-scopes" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
  OPENID_SCOPE_UUID=$(echo "$SCOPES_RESPONSE" | jq -r '.[] | select(.name == "openid") | .id // empty')
  echo "  ✅ Created openid scope"
else
  echo "  ✅ openid scope already exists"
fi

# Create mcp-server scope (for audience validation)
MCP_SERVER_SCOPE_UUID=$(echo "$SCOPES_RESPONSE" | jq -r '.[] | select(.name == "mcp-server") | .id // empty')

if [ -z "$MCP_SERVER_SCOPE_UUID" ] || [ "$MCP_SERVER_SCOPE_UUID" = "null" ]; then
  curl -sk -X POST "$KEYCLOAK_URL/admin/realms/$HUB_REALM/client-scopes" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
      "name": "mcp-server",
      "description": "MCP Server audience scope",
      "protocol": "openid-connect",
      "attributes": {
        "include.in.token.scope": "true",
        "display.on.consent.screen": "false"
      }
    }' > /dev/null

  # Wait and fetch UUID
  sleep 2
  SCOPES_RESPONSE=$(curl -sk -X GET "$KEYCLOAK_URL/admin/realms/$HUB_REALM/client-scopes" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
  MCP_SERVER_SCOPE_UUID=$(echo "$SCOPES_RESPONSE" | jq -r '.[] | select(.name == "mcp-server") | .id // empty')
  echo "  ✅ Created mcp-server scope"
else
  echo "  ✅ mcp-server scope already exists"
fi

# Create mcp-server client
echo ""
echo "Creating mcp-server client..."
CLIENT_UUID=$(curl -sk -X GET "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients?clientId=$CLIENT_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[0].id // empty')

if [ -z "$CLIENT_UUID" ] || [ "$CLIENT_UUID" = "null" ]; then
  CLIENT_SECRET=$(openssl rand -hex 32)

  curl -sk -X POST "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"clientId\": \"$CLIENT_ID\",
      \"enabled\": true,
      \"protocol\": \"openid-connect\",
      \"publicClient\": false,
      \"directAccessGrantsEnabled\": true,
      \"serviceAccountsEnabled\": true,
      \"standardFlowEnabled\": true,
      \"secret\": \"$CLIENT_SECRET\",
      \"redirectUris\": [\"http://localhost:*\", \"https://*\"],
      \"webOrigins\": [\"*\"],
      \"attributes\": {
        \"token.exchange.grant.enabled\": \"true\"
      }
    }" > /dev/null

  CLIENT_UUID=$(curl -sk -X GET "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients?clientId=$CLIENT_ID" \
    -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[0].id')

  echo "  ✅ Created client: $CLIENT_ID"
  echo "  📝 Client Secret: $CLIENT_SECRET"
else
  echo "  ✅ Client already exists: $CLIENT_UUID"
  CLIENT_SECRET=$(curl -sk -X GET "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients/$CLIENT_UUID/client-secret" \
    -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.value')
  echo "  📝 Client Secret: $CLIENT_SECRET"
fi

# Add scopes to mcp-server client
echo ""
echo "Adding scopes to mcp-server client..."
curl -sk -X PUT "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients/$CLIENT_UUID/default-client-scopes/$OPENID_SCOPE_UUID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" > /dev/null 2>&1
curl -sk -X PUT "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients/$CLIENT_UUID/default-client-scopes/$MCP_SERVER_SCOPE_UUID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" > /dev/null 2>&1
echo "✅ Scopes added (openid, mcp-server)"

# Add sub claim mapper
echo ""
echo "Creating sub claim mapper..."
EXISTING_SUB_MAPPER=$(curl -sk -X GET "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients/$CLIENT_UUID/protocol-mappers/models" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[] | select(.name == "sub") | .id // empty')

if [ -z "$EXISTING_SUB_MAPPER" ]; then
  curl -sk -X POST "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients/$CLIENT_UUID/protocol-mappers/models" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
      "name": "sub",
      "protocol": "openid-connect",
      "protocolMapper": "oidc-sub-mapper",
      "consentRequired": false,
      "config": {
        "id.token.claim": "true",
        "access.token.claim": "true",
        "userinfo.token.claim": "true"
      }
    }' > /dev/null
  echo "  ✅ Created sub claim mapper"
else
  echo "  ✅ sub claim mapper already exists"
fi

# Create mcp-client (public OAuth client for inspector/browser flow)
echo ""
echo "Creating mcp-client (public OAuth client)..."
MCP_CLIENT_UUID=$(curl -sk -X GET "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients?clientId=mcp-client" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[0].id // empty')

if [ -z "$MCP_CLIENT_UUID" ] || [ "$MCP_CLIENT_UUID" = "null" ]; then
  curl -sk -X POST "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
      "clientId": "mcp-client",
      "enabled": true,
      "protocol": "openid-connect",
      "publicClient": true,
      "directAccessGrantsEnabled": true,
      "standardFlowEnabled": true,
      "redirectUris": ["http://localhost:*"],
      "webOrigins": ["*"]
    }' > /dev/null

  MCP_CLIENT_UUID=$(curl -sk -X GET "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients?clientId=mcp-client" \
    -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[0].id')

  # Add scopes to mcp-client
  curl -sk -X PUT "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients/$MCP_CLIENT_UUID/default-client-scopes/$OPENID_SCOPE_UUID" \
    -H "Authorization: Bearer $ADMIN_TOKEN" > /dev/null 2>&1
  curl -sk -X PUT "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients/$MCP_CLIENT_UUID/default-client-scopes/$MCP_SERVER_SCOPE_UUID" \
    -H "Authorization: Bearer $ADMIN_TOKEN" > /dev/null 2>&1

  # Add audience mapper to include mcp-server in aud claim
  curl -sk -X POST "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients/$MCP_CLIENT_UUID/protocol-mappers/models" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
      "name": "mcp-server-audience",
      "protocol": "openid-connect",
      "protocolMapper": "oidc-audience-mapper",
      "consentRequired": false,
      "config": {
        "included.client.audience": "mcp-server",
        "id.token.claim": "true",
        "access.token.claim": "true"
      }
    }' > /dev/null 2>&1

  echo "  ✅ Created mcp-client (public OAuth client)"
else
  echo "  ✅ mcp-client already exists"
fi

# Create mcp-sts client (for token exchange)
echo ""
echo "Creating mcp-sts client (for token exchange)..."
STS_CLIENT_UUID=$(curl -sk -X GET "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients?clientId=mcp-sts" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[0].id // empty')

if [ -z "$STS_CLIENT_UUID" ] || [ "$STS_CLIENT_UUID" = "null" ]; then
  STS_CLIENT_SECRET=$(openssl rand -hex 32)

  curl -sk -X POST "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"clientId\": \"mcp-sts\",
      \"enabled\": true,
      \"protocol\": \"openid-connect\",
      \"publicClient\": false,
      \"directAccessGrantsEnabled\": true,
      \"serviceAccountsEnabled\": true,
      \"standardFlowEnabled\": false,
      \"secret\": \"$STS_CLIENT_SECRET\",
      \"attributes\": {
        \"token.exchange.grant.enabled\": \"true\"
      }
    }" > /dev/null

  STS_CLIENT_UUID=$(curl -sk -X GET "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients?clientId=mcp-sts" \
    -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[0].id')

  # Add scopes to mcp-sts
  curl -sk -X PUT "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients/$STS_CLIENT_UUID/default-client-scopes/$OPENID_SCOPE_UUID" \
    -H "Authorization: Bearer $ADMIN_TOKEN" > /dev/null 2>&1
  curl -sk -X PUT "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients/$STS_CLIENT_UUID/default-client-scopes/$MCP_SERVER_SCOPE_UUID" \
    -H "Authorization: Bearer $ADMIN_TOKEN" > /dev/null 2>&1

  echo "  ✅ Created mcp-sts client"
  echo "  📝 STS Client Secret: $STS_CLIENT_SECRET"
else
  echo "  ✅ mcp-sts client already exists"
  STS_CLIENT_SECRET=$(curl -sk -X GET "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients/$STS_CLIENT_UUID/client-secret" \
    -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.value')
  echo "  📝 STS Client Secret: $STS_CLIENT_SECRET"
fi

# Create test user
echo ""
echo "Creating test user..."
EXISTING_USER=$(curl -sk -X GET "$KEYCLOAK_URL/admin/realms/$HUB_REALM/users?username=$MCP_USERNAME&exact=true" \
  -H "Authorization: Bearer $ADMIN_TOKEN")

USER_ID=$(echo "$EXISTING_USER" | jq -r '.[0].id // empty')

if [ -z "$USER_ID" ] || [ "$USER_ID" = "null" ]; then
  curl -sk -X POST "$KEYCLOAK_URL/admin/realms/$HUB_REALM/users" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"username\": \"$MCP_USERNAME\",
      \"enabled\": true,
      \"emailVerified\": true,
      \"email\": \"$MCP_USERNAME@example.com\",
      \"firstName\": \"MCP\",
      \"lastName\": \"User\",
      \"requiredActions\": []
    }" > /dev/null

  USER_ID=$(curl -sk -X GET "$KEYCLOAK_URL/admin/realms/$HUB_REALM/users?username=$MCP_USERNAME&exact=true" \
    -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[0].id')

  # Set password
  curl -sk -X PUT "$KEYCLOAK_URL/admin/realms/$HUB_REALM/users/$USER_ID/reset-password" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"type\": \"password\",
      \"value\": \"$MCP_PASSWORD\",
      \"temporary\": false
    }" > /dev/null

  echo "  ✅ Created user: $MCP_USERNAME / $MCP_PASSWORD"
else
  echo "  ✅ User already exists: $MCP_USERNAME"
fi

# Save configuration
echo ""
echo "Saving configuration..."
mkdir -p .keycloak-config
cat > .keycloak-config/hub-config.env <<EOF
KEYCLOAK_URL="$KEYCLOAK_URL"
HUB_REALM="$HUB_REALM"
CLIENT_ID="$CLIENT_ID"
CLIENT_SECRET="$CLIENT_SECRET"
STS_CLIENT_ID="mcp-sts"
STS_CLIENT_SECRET="$STS_CLIENT_SECRET"
MCP_USERNAME="$MCP_USERNAME"
MCP_PASSWORD="$MCP_PASSWORD"
ADMIN_USER="$ADMIN_USER"
ADMIN_PASSWORD="$ADMIN_PASSWORD"
EOF

echo "✅ Configuration saved to .keycloak-config/hub-config.env"

# Step 12: Configure same-realm token exchange permissions (mcp-sts → mcp-server)
echo ""
echo "Step 12: Configuring same-realm token exchange permissions..."

# Enable management permissions on mcp-server client
curl -sk -X PUT "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients/$CLIENT_UUID/management/permissions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"enabled": true}' > /dev/null

# Get the token-exchange permission ID
MCP_SERVER_PERMS=$(curl -sk -X GET "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients/$CLIENT_UUID/management/permissions" \
  -H "Authorization: Bearer $ADMIN_TOKEN")
TOKEN_EXCHANGE_PERM_ID=$(echo "$MCP_SERVER_PERMS" | jq -r '.scopePermissions."token-exchange"')

# Get realm-management client ID
REALM_MGMT_ID=$(curl -sk -X GET "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients?clientId=realm-management" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[0].id')

# Create client policy for mcp-sts
POLICY_RESPONSE=$(curl -sk -X POST "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients/$REALM_MGMT_ID/authz/resource-server/policy/client" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"type\": \"client\",
    \"logic\": \"POSITIVE\",
    \"decisionStrategy\": \"UNANIMOUS\",
    \"name\": \"allow-mcp-sts-to-exchange-to-mcp-server\",
    \"description\": \"Allow mcp-sts client to perform token exchange to mcp-server audience\",
    \"clients\": [\"$STS_CLIENT_UUID\"]
  }" 2>/dev/null)

STS_POLICY_ID=$(echo "$POLICY_RESPONSE" | jq -r '.id // empty')

if [ -z "$STS_POLICY_ID" ] || [ "$STS_POLICY_ID" = "null" ]; then
  # Policy might already exist, try to find it
  ALL_POLICIES=$(curl -sk -X GET "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients/$REALM_MGMT_ID/authz/resource-server/policy?type=client" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
  STS_POLICY_ID=$(echo "$ALL_POLICIES" | jq -r '.[] | select(.name == "allow-mcp-sts-to-exchange-to-mcp-server") | .id')
fi

# Link policy to token-exchange permission
CURRENT_PERM=$(curl -sk -X GET "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients/$REALM_MGMT_ID/authz/resource-server/permission/$TOKEN_EXCHANGE_PERM_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN")

UPDATED_PERM=$(echo "$CURRENT_PERM" | jq --arg policy_id "$STS_POLICY_ID" '. + {policies: [$policy_id]}')

curl -sk -X PUT "$KEYCLOAK_URL/admin/realms/$HUB_REALM/clients/$REALM_MGMT_ID/authz/resource-server/permission/$TOKEN_EXCHANGE_PERM_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "$UPDATED_PERM" > /dev/null

echo "  ✅ Same-realm token exchange configured (mcp-sts → mcp-server)"

# Step 13: Create RBAC for mcp user on hub cluster
echo ""
echo "Step 13: Creating RBAC for mcp user on hub cluster..."

oc apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: mcp-cluster-admin
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: User
  name: $MCP_USERNAME
EOF

echo "  ✅ RBAC created for mcp user"

# Step 14: Fix Keycloak CA trust for cross-realm token exchange
echo ""
echo "Step 14: Fixing Keycloak CA trust for cross-realm token exchange..."

# Extract OpenShift router CA
ROUTER_CA=""
if ROUTER_CA=$(oc get secret router-ca -n openshift-ingress-operator -o jsonpath='{.data.tls\.crt}' 2>/dev/null | base64 -d); then
    if [ -n "$ROUTER_CA" ]; then
        echo "  ✅ Found router CA in openshift-ingress-operator/router-ca"
    fi
fi

if [ -z "$ROUTER_CA" ]; then
    if ROUTER_CA=$(oc get secret router-certs-default -n openshift-ingress -o jsonpath='{.data.tls\.crt}' 2>/dev/null | base64 -d); then
        if [ -n "$ROUTER_CA" ]; then
            echo "  ✅ Found router CA in openshift-ingress/router-certs-default"
        fi
    fi
fi

if [ -z "$ROUTER_CA" ]; then
    echo "  ⚠️  Could not find router CA certificate, cross-realm token exchange may fail"
else
    # Create ConfigMap
    TEMP_CA=$(mktemp)
    echo "$ROUTER_CA" > "$TEMP_CA"

    oc create configmap router-ca -n keycloak \
        --from-file=router-ca.crt="$TEMP_CA" \
        --dry-run=client -o yaml | oc apply -f -

    rm -f "$TEMP_CA"

    echo "  ✅ ConfigMap router-ca created in keycloak namespace"

    # Check if Keycloak deployment needs patching
    CURRENT_TRUSTSTORE=$(oc get deployment keycloak -n keycloak -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="KC_TRUSTSTORE_PATHS")].value}' 2>/dev/null || echo "")

    if [ "$CURRENT_TRUSTSTORE" = "/ca-certs/router-ca.crt" ]; then
        echo "  ✅ Keycloak already configured with CA trust"
    else
        # Patch Keycloak deployment
        PATCH_JSON=$(cat <<'EOF'
{
  "spec": {
    "template": {
      "spec": {
        "containers": [
          {
            "name": "keycloak",
            "env": [
              {
                "name": "KC_TRUSTSTORE_PATHS",
                "value": "/ca-certs/router-ca.crt"
              }
            ],
            "volumeMounts": [
              {
                "name": "router-ca",
                "mountPath": "/ca-certs",
                "readOnly": true
              }
            ]
          }
        ],
        "volumes": [
          {
            "name": "router-ca",
            "configMap": {
              "name": "router-ca"
            }
          }
        ]
      }
    }
  }
}
EOF
)

        oc patch deployment keycloak -n keycloak --type=strategic --patch "$PATCH_JSON"
        echo "  ✅ Keycloak deployment patched with CA trust"
        echo "  ⏳ Waiting for Keycloak to restart..."

        oc rollout status deployment/keycloak -n keycloak --timeout=5m
        echo "  ✅ Keycloak restarted with CA trust"
    fi
fi

echo ""
echo "=========================================="
echo "✅ Hub Keycloak Setup Complete!"
echo "=========================================="
echo ""
echo "Configuration Summary:"
echo "  Keycloak URL: $KEYCLOAK_URL"
echo "  Hub Realm: $KEYCLOAK_URL/realms/$HUB_REALM"
echo ""
echo "  Clients created:"
echo "    - mcp-server (confidential): $CLIENT_SECRET"
echo "    - mcp-client (public OAuth): for browser/inspector flow"
echo "    - mcp-sts (STS): $STS_CLIENT_SECRET"
echo ""
echo "  Test User: $MCP_USERNAME / $MCP_PASSWORD"
echo "  Admin: $ADMIN_USER / $ADMIN_PASSWORD"
echo ""
echo "  V1 Features: token-exchange:v1,admin-fine-grained-authz:v1"
echo "  openid Scope: ✅ Configured on all clients"
echo "  sub Claim Mapper: ✅ Configured"
echo "  Token Exchange: ✅ Enabled"
echo "  Same-Realm Exchange: ✅ Configured (mcp-sts → mcp-server)"
echo ""
echo "Next Steps:"
echo "  1. Wait for cluster-bot to be ready"
echo "  2. Register cluster-bot with:"
echo "     CLUSTER_NAME=cluster-bot MANAGED_KUBECONFIG=/path/to/kubeconfig \\"
echo "       ./hack/acm/acm-register-managed-cluster.sh"
echo ""
echo "Test authentication:"
echo "  curl -sk -X POST \"$KEYCLOAK_URL/realms/$HUB_REALM/protocol/openid-connect/token\" \\"
echo "    -d \"grant_type=password\" -d \"client_id=$CLIENT_ID\" \\"
echo "    -d \"client_secret=$CLIENT_SECRET\" -d \"username=$MCP_USERNAME\" \\"
echo "    -d \"password=$MCP_PASSWORD\" -d \"scope=openid $CLIENT_ID\""
echo ""
