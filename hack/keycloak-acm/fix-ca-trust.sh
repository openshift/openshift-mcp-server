#!/bin/bash
set -euo pipefail

# Fix Keycloak CA Trust for Same-Instance Cross-Realm Token Exchange
#
# This script configures Keycloak to trust the OpenShift router CA certificate,
# enabling JWKS signature validation for cross-realm token exchange.
#
# Based on: SINGLE_KEYCLOAK_SUCCESS.md

echo "==========================================="
echo "Fixing Keycloak CA Trust"
echo "==========================================="
echo ""
echo "This will configure Keycloak to trust the OpenShift router CA certificate"
echo "for same-instance cross-realm JWKS validation."
echo ""

# Check if running on OpenShift
if ! kubectl get route -n keycloak keycloak >/dev/null 2>&1; then
    echo "❌ Error: Not running on OpenShift (no route found)"
    echo "This script is designed for OpenShift clusters with OpenShift routes"
    exit 1
fi

echo "Step 1: Extracting OpenShift router CA certificate..."

# Try different sources for the router CA
ROUTER_CA=""

# Method 1: Try router-ca from openshift-ingress-operator namespace
if ROUTER_CA=$(kubectl get secret router-ca -n openshift-ingress-operator -o jsonpath='{.data.tls\.crt}' 2>/dev/null | base64 -d); then
    if [ -n "$ROUTER_CA" ]; then
        echo "  ✅ Found router CA in openshift-ingress-operator/router-ca"
    fi
fi

# Method 2: Try router-certs-default from openshift-ingress namespace
if [ -z "$ROUTER_CA" ]; then
    if ROUTER_CA=$(kubectl get secret router-certs-default -n openshift-ingress -o jsonpath='{.data.tls\.crt}' 2>/dev/null | base64 -d); then
        if [ -n "$ROUTER_CA" ]; then
            echo "  ✅ Found router CA in openshift-ingress/router-certs-default"
        fi
    fi
fi

# Method 3: Extract from ingress controller
if [ -z "$ROUTER_CA" ]; then
    INGRESS_CA=$(kubectl get ingresscontroller default -n openshift-ingress-operator -o jsonpath='{.spec.defaultCertificate.name}' 2>/dev/null)
    if [ -n "$INGRESS_CA" ]; then
        if ROUTER_CA=$(kubectl get secret "$INGRESS_CA" -n openshift-ingress -o jsonpath='{.data.tls\.crt}' 2>/dev/null | base64 -d); then
            if [ -n "$ROUTER_CA" ]; then
                echo "  ✅ Found router CA in openshift-ingress/$INGRESS_CA"
            fi
        fi
    fi
fi

# Verify we found a CA
if [ -z "$ROUTER_CA" ]; then
    echo "❌ Error: Could not find OpenShift router CA certificate"
    echo ""
    echo "Tried:"
    echo "  - secret/router-ca in openshift-ingress-operator"
    echo "  - secret/router-certs-default in openshift-ingress"
    echo "  - ingress controller default certificate"
    exit 1
fi

# Verify it's a valid certificate
if ! echo "$ROUTER_CA" | openssl x509 -noout -text >/dev/null 2>&1; then
    echo "❌ Error: Invalid CA certificate format"
    exit 1
fi

# Show certificate info
echo ""
echo "Router CA Certificate Details:"
echo "$ROUTER_CA" | openssl x509 -noout -subject -issuer -dates | sed 's/^/  /'
echo ""

echo "Step 2: Creating router-ca ConfigMap in keycloak namespace..."

# Create temporary file
TEMP_CA=$(mktemp)
echo "$ROUTER_CA" > "$TEMP_CA"

# Create or update ConfigMap
kubectl create configmap router-ca -n keycloak \
    --from-file=router-ca.crt="$TEMP_CA" \
    --dry-run=client -o yaml | kubectl apply -f -

rm -f "$TEMP_CA"

echo "  ✅ ConfigMap router-ca created/updated"
echo ""

echo "Step 3: Checking Keycloak deployment..."

if ! kubectl get deployment keycloak -n keycloak >/dev/null 2>&1; then
    echo "❌ Error: Keycloak deployment not found in keycloak namespace"
    exit 1
fi

echo "  ✅ Keycloak deployment found"
echo ""

echo "Step 4: Patching Keycloak deployment with KC_TRUSTSTORE_PATHS..."

# Check if already patched
CURRENT_TRUSTSTORE=$(kubectl get deployment keycloak -n keycloak -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="KC_TRUSTSTORE_PATHS")].value}' 2>/dev/null || echo "")

if [ "$CURRENT_TRUSTSTORE" = "/ca-certs/router-ca.crt" ]; then
    echo "  ℹ️  KC_TRUSTSTORE_PATHS already configured"

    # Check if volume mount exists
    VOLUME_MOUNT=$(kubectl get deployment keycloak -n keycloak -o jsonpath='{.spec.template.spec.containers[0].volumeMounts[?(@.name=="router-ca")].mountPath}' 2>/dev/null || echo "")

    if [ -n "$VOLUME_MOUNT" ]; then
        echo "  ✅ Volume mount already configured"
        echo ""
        echo "==========================================="
        echo "✅ Keycloak CA Trust Already Configured!"
        echo "==========================================="
        exit 0
    fi
fi

# Create patch JSON
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

# Apply strategic merge patch
kubectl patch deployment keycloak -n keycloak --type=strategic --patch "$PATCH_JSON"

echo "  ✅ Deployment patched"
echo ""

echo "Step 5: Waiting for Keycloak to restart..."

# Wait for rollout
if kubectl rollout status deployment/keycloak -n keycloak --timeout=5m; then
    echo "  ✅ Keycloak rollout complete"
else
    echo "  ⚠️  Rollout taking longer than expected"
    echo "  Check status with: kubectl rollout status deployment/keycloak -n keycloak"
fi

echo ""
echo "Step 6: Verifying Keycloak is ready..."

# Wait for pod to be ready
for i in {1..30}; do
    if kubectl get pods -n keycloak -l app=keycloak -o jsonpath='{.items[0].status.conditions[?(@.type=="Ready")].status}' 2>/dev/null | grep -q "True"; then
        echo "  ✅ Keycloak pod is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "  ⚠️  Keycloak pod not ready after 5 minutes"
        echo "  Check logs with: make keycloak-logs"
        exit 1
    fi
    sleep 10
done

echo ""

# Get Keycloak URL
KEYCLOAK_URL="https://$(kubectl get route keycloak -n keycloak -o jsonpath='{.spec.host}')"

# Check Keycloak health
if curl -sk "$KEYCLOAK_URL/health/ready" | grep -q '"status":"UP"'; then
    echo "  ✅ Keycloak health check passed"
else
    echo "  ⚠️  Keycloak health check did not return UP"
    echo "  URL: $KEYCLOAK_URL/health/ready"
fi

echo ""
echo "==========================================="
echo "✅ Keycloak CA Trust Fixed!"
echo "==========================================="
echo ""
echo "What this enables:"
echo "  ✅ Cross-realm JWKS signature validation"
echo "  ✅ validateSignature=true in IDP configuration"
echo "  ✅ Proper TLS trust for same-instance token exchange"
echo ""
echo "Keycloak Configuration:"
echo "  KC_TRUSTSTORE_PATHS: /ca-certs/router-ca.crt"
echo "  ConfigMap: router-ca (keycloak namespace)"
echo "  Router CA: Imported into JVM truststore"
echo ""
echo "Next steps:"
if [ -f ".keycloak-config/hub-config.env" ]; then
    echo "  ✅ Hub realm already configured"
    echo "  → Register managed clusters: make keycloak-acm-register-managed-declarative CLUSTER_NAME=... MANAGED_KUBECONFIG=..."
else
    echo "  1. Setup hub realm: make keycloak-acm-setup-hub-declarative"
    echo "  2. Register managed clusters: make keycloak-acm-register-managed-declarative CLUSTER_NAME=... MANAGED_KUBECONFIG=..."
fi
echo ""
