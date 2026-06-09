#!/usr/bin/env bash

# Import a managed cluster into ACM
# Usage: ./import-cluster.sh <cluster-name> <kubeconfig-path>

set -euo pipefail

CLUSTER_NAME="${1:-}"
MANAGED_KUBECONFIG="${2:-}"

# Validate inputs
if [ -z "$CLUSTER_NAME" ]; then
    echo "Error: CLUSTER_NAME is required"
    echo "Usage: $0 <cluster-name> <kubeconfig-path>"
    exit 1
fi

if [ -z "$MANAGED_KUBECONFIG" ]; then
    echo "Error: MANAGED_KUBECONFIG is required"
    echo "Usage: $0 <cluster-name> <kubeconfig-path>"
    exit 1
fi

if [ ! -f "$MANAGED_KUBECONFIG" ]; then
    echo "Error: Kubeconfig file not found: $MANAGED_KUBECONFIG"
    exit 1
fi

echo "==========================================="
echo "Importing cluster: $CLUSTER_NAME"
echo "==========================================="

# Step 1: Create ManagedCluster resource
echo "Step 1: Creating ManagedCluster resource on hub..."
cat <<EOF | oc apply -f -
apiVersion: cluster.open-cluster-management.io/v1
kind: ManagedCluster
metadata:
  name: $CLUSTER_NAME
  labels:
    cloud: auto-detect
    vendor: auto-detect
spec:
  hubAcceptsClient: true
EOF

# Step 2: Wait for import secret
echo "Step 2: Waiting for import secret to be created..."
for i in {1..60}; do
    if oc get secret -n "$CLUSTER_NAME" "$CLUSTER_NAME-import" 2>/dev/null; then
        echo "✅ Import secret created!"
        break
    fi
    echo "  Waiting for import secret ($i/60)..."
    sleep 2
done

# Step 3: Extract import manifests
echo "Step 3: Extracting import manifests..."
mkdir -p _output/acm-import
oc get secret -n "$CLUSTER_NAME" "$CLUSTER_NAME-import" -o jsonpath='{.data.crds\.yaml}' | base64 -d > "_output/acm-import/${CLUSTER_NAME}-crds.yaml"
oc get secret -n "$CLUSTER_NAME" "$CLUSTER_NAME-import" -o jsonpath='{.data.import\.yaml}' | base64 -d > "_output/acm-import/${CLUSTER_NAME}-import.yaml"
echo "Import manifests saved to _output/acm-import/"

# Step 4: Apply CRDs to managed cluster
echo "Step 4: Applying CRDs to managed cluster..."
KUBECONFIG="$MANAGED_KUBECONFIG" oc apply -f "_output/acm-import/${CLUSTER_NAME}-crds.yaml"
echo "  Waiting for CRDs to be established..."
sleep 5

# Step 5: Apply import manifest
echo "Step 5: Applying import manifest to managed cluster..."
KUBECONFIG="$MANAGED_KUBECONFIG" oc apply -f "_output/acm-import/${CLUSTER_NAME}-import.yaml"

# Step 6: Wait for klusterlet to be ready
echo "Step 6: Waiting for klusterlet to be ready..."
for i in {1..120}; do
    if oc get managedcluster "$CLUSTER_NAME" -o jsonpath='{.status.conditions[?(@.type=="ManagedClusterConditionAvailable")].status}' 2>/dev/null | grep -q "True"; then
        echo "✅ Cluster $CLUSTER_NAME is now available!"
        break
    fi
    echo "  Waiting for cluster to become available ($i/120)..."
    sleep 5
done

echo "==========================================="
echo "✓ Cluster import complete!"
echo "==========================================="
oc get managedcluster "$CLUSTER_NAME"