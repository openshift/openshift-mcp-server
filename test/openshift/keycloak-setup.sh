#!/bin/bash
#
# Installs cert-manager, deploys Keycloak with the project's realm, and
# configures the OCP API server for OIDC authentication.
#
# Designed to run inside a CI test pod with oc in PATH and KUBECONFIG
# pointing at an IPI-provisioned OCP cluster.
set -euo pipefail

CERT_MANAGER_VERSION="${CERT_MANAGER_VERSION:-v1.16.2}"

echo "=== Keycloak CI Setup ==="

# --- Version gate --------------------------------------------------------
CLUSTER_VERSION=$(oc get clusterversion version -o jsonpath='{.status.desired.version}')
OCP_MAJOR=$(echo "$CLUSTER_VERSION" | cut -d. -f1)
OCP_MINOR=$(echo "$CLUSTER_VERSION" | cut -d. -f2)
if [[ "$OCP_MAJOR" -eq 4 && "$OCP_MINOR" -lt 20 ]]; then
    echo "ExternalOIDC requires OCP 4.20+, got $CLUSTER_VERSION — skipping Keycloak setup"
    exit 0
fi
echo "Cluster version: $CLUSTER_VERSION (ExternalOIDC supported)"

# --- cert-manager --------------------------------------------------------
echo "Installing cert-manager ${CERT_MANAGER_VERSION}..."
oc apply -f "https://github.com/cert-manager/cert-manager/releases/download/${CERT_MANAGER_VERSION}/cert-manager.yaml"
echo "Waiting for cert-manager deployments..."
oc wait --namespace cert-manager --for=condition=available deployment/cert-manager --timeout=120s
oc wait --namespace cert-manager --for=condition=available deployment/cert-manager-cainjector --timeout=120s
oc wait --namespace cert-manager --for=condition=available deployment/cert-manager-webhook --timeout=120s
echo "cert-manager ready"

# --- Self-signed CA issuer chain -----------------------------------------
echo "Creating self-signed CA issuer chain..."
oc apply -f dev/config/cert-manager/selfsigned-issuer.yaml
# Wait for the CA certificate to be issued
oc wait --for=condition=ready certificate/selfsigned-ca -n cert-manager --timeout=60s
echo "CA issuer chain ready"

# --- STS assertion keypair -----------------------------------------------
echo "Generating STS assertion keypair..."
dev/config/keycloak/gen-sts-assertion-keypair.sh

# --- Keycloak deployment -------------------------------------------------
echo "Rendering realm import with STS assertion cert..."
STS_CERT_DER=$(grep -v -- '-----' test/e2e/testdata/generated/sts-assertion.crt | tr -d '\n')
sed "s|@@STS_ASSERTION_CERT_DER@@|${STS_CERT_DER}|" \
    dev/config/keycloak/realm-import.yaml > /tmp/realm-import.rendered.yaml

echo "Deploying Keycloak..."
oc create namespace keycloak --dry-run=client -o yaml | oc apply -f -
oc apply -f /tmp/realm-import.rendered.yaml
oc apply -f dev/config/keycloak/deployment.yaml

echo "Waiting for Keycloak TLS certificate..."
oc wait --for=condition=ready certificate/keycloak-tls -n keycloak --timeout=120s

echo "Restarting Keycloak to pick up the realm import..."
oc rollout restart deployment/keycloak -n keycloak

echo "Waiting for Keycloak to be ready..."
oc rollout status deployment/keycloak -n keycloak --timeout=300s
echo "Keycloak ready"

# --- OCP API server OIDC configuration -----------------------------------
echo "Extracting cert-manager CA..."
oc get secret selfsigned-ca-secret -n cert-manager \
    -o jsonpath='{.data.ca\.crt}' | base64 -d > /tmp/ca.crt

echo "Creating CA ConfigMap in openshift-config..."
oc create configmap keycloak-oidc-ca \
    --from-file=ca-bundle.crt=/tmp/ca.crt \
    -n openshift-config \
    --dry-run=client -o yaml | oc apply -f -

echo "Patching Authentication CR for OIDC..."
oc patch authentication.config/cluster --type=merge -p '{
  "spec": {
    "type": "OIDC",
    "webhookTokenAuthenticator": null,
    "oidcProviders": [{
      "name": "keycloak",
      "issuer": {
        "issuerURL": "https://keycloak.keycloak.svc:8443/realms/openshift",
        "audiences": ["openshift"],
        "issuerCertificateAuthority": {"name": "keycloak-oidc-ca"}
      },
      "claimMappings": {
        "username": {"claim": "preferred_username", "prefixPolicy": "NoPrefix"},
        "groups": {"claim": "groups", "prefix": ""}
      },
      "oidcClients": []
    }]
  }
}'

echo "Waiting for kube-apiserver to start rolling out..."
if ! oc wait co/kube-apiserver --for=condition=Progressing --timeout=120s; then
    echo "API server did not start progressing — checking status"
    oc get co/kube-apiserver
    exit 1
fi

echo "Waiting for kube-apiserver rollout to complete (this may take several minutes)..."
if ! oc wait co/kube-apiserver --for=condition=Progressing=false --timeout=900s; then
    echo "API server rollout timed out"
    oc get co/kube-apiserver
    oc get po -n openshift-kube-apiserver -L revision -l apiserver
    exit 1
fi

echo "Verifying cluster operators are healthy..."
if oc get co kube-apiserver authentication --no-headers | grep -v "True  *False  *False"; then
    echo "WARNING: some cluster operators not in expected state"
    oc get co kube-apiserver authentication
fi
echo "API server OIDC configuration complete"

# --- RBAC for OIDC users -------------------------------------------------
echo "Applying OIDC RBAC bindings..."
oc apply -f test/openshift/keycloak-rbac.yaml

echo "=== Keycloak CI Setup Complete ==="
