#!/usr/bin/env bash

# Install MultiCluster Engine (MCE) - required for ACM
# This script installs the MCE operator and creates a MultiClusterEngine instance

set -euo pipefail

echo "Installing MultiCluster Engine (MCE)..."

# Create namespace
echo "Creating multicluster-engine namespace..."
oc create namespace multicluster-engine --dry-run=client -o yaml | oc apply -f -

# Create OperatorGroup
echo "Creating MCE OperatorGroup..."
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: multicluster-engine
  namespace: multicluster-engine
spec:
  targetNamespaces:
  - multicluster-engine
EOF

# Create Subscription
echo "Creating MCE Subscription..."
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: multicluster-engine
  namespace: multicluster-engine
spec:
  channel: stable-2.9
  name: multicluster-engine
  source: redhat-operators
  sourceNamespace: openshift-marketplace
EOF

# Wait for CSV to appear and get its name
echo "Waiting for MCE operator CSV to be ready..."
CSV_NAME=""
for i in {1..60}; do
    CSV_NAME=$(oc get csv -n multicluster-engine -o name 2>/dev/null | grep multicluster-engine || true)
    if [ -n "$CSV_NAME" ]; then
        echo "MCE CSV found: $CSV_NAME"
        break
    fi
    echo "  Waiting for MCE CSV to appear ($i/60)..."
    sleep 5
done

if [ -z "$CSV_NAME" ]; then
    echo "Error: MCE CSV not found after waiting"
    exit 1
fi

# Wait for CSV to be ready
echo "Waiting for CSV to reach Succeeded phase..."
oc wait --for=jsonpath='{.status.phase}'=Succeeded "$CSV_NAME" -n multicluster-engine --timeout=300s

# Create MultiClusterEngine instance
echo "Creating MultiClusterEngine instance..."
cat <<EOF | oc apply -f -
apiVersion: multicluster.openshift.io/v1
kind: MultiClusterEngine
metadata:
  name: multiclusterengine
spec: {}
EOF

# Wait for ManagedCluster CRD
echo "Waiting for ManagedCluster CRD to be available..."
for i in {1..120}; do
    if oc get crd managedclusters.cluster.open-cluster-management.io >/dev/null 2>&1; then
        echo "✅ ManagedCluster CRD is now available!"
        break
    fi
    echo "  Waiting for ManagedCluster CRD ($i/120)..."
    sleep 5
done

echo "✓ MCE installation complete"