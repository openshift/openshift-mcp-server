#!/bin/bash
# Apply ACM Import Manifests to Managed Cluster
# This script extracts import manifests from the hub cluster and applies them to the managed cluster
# to start the klusterlet agents required for cluster-proxy functionality.
#
# Required environment variables:
#   CLUSTER_NAME         - Name of the managed cluster (e.g., managed-cluster-one)
#   HUB_KUBECONFIG       - Path to hub cluster kubeconfig
#   MANAGED_KUBECONFIG   - Path to managed cluster kubeconfig

set -e

# Validate required environment variables
if [ -z "$CLUSTER_NAME" ]; then
  echo "Error: CLUSTER_NAME environment variable is required"
  exit 1
fi

if [ -z "$HUB_KUBECONFIG" ]; then
  echo "Error: HUB_KUBECONFIG environment variable is required"
  exit 1
fi

if [ -z "$MANAGED_KUBECONFIG" ]; then
  echo "Error: MANAGED_KUBECONFIG environment variable is required"
  exit 1
fi

echo "=========================================="
echo "Applying Import Manifests"
echo "=========================================="
echo "Cluster Name: $CLUSTER_NAME"
echo "Hub Kubeconfig: $HUB_KUBECONFIG"
echo "Managed Kubeconfig: $MANAGED_KUBECONFIG"
echo ""

# Create temporary directory for manifests
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

echo "Step 1: Extracting import manifests from hub cluster..."
kubectl --kubeconfig="$HUB_KUBECONFIG" get secret ${CLUSTER_NAME}-import -n ${CLUSTER_NAME} -o jsonpath='{.data.crds\.yaml}' | base64 -d > "$TEMP_DIR/crds.yaml"
kubectl --kubeconfig="$HUB_KUBECONFIG" get secret ${CLUSTER_NAME}-import -n ${CLUSTER_NAME} -o jsonpath='{.data.import\.yaml}' | base64 -d > "$TEMP_DIR/import.yaml"
echo "  ✅ Manifests extracted to $TEMP_DIR"
echo "     - crds.yaml: $(wc -l < $TEMP_DIR/crds.yaml) lines"
echo "     - import.yaml: $(wc -l < $TEMP_DIR/import.yaml) lines"
echo ""

echo "Step 2: Applying CRDs to managed cluster..."
kubectl --kubeconfig="$MANAGED_KUBECONFIG" apply -f "$TEMP_DIR/crds.yaml"
echo "  ✅ CRDs applied"
echo ""

echo "Step 3: Applying import resources to managed cluster..."
kubectl --kubeconfig="$MANAGED_KUBECONFIG" apply -f "$TEMP_DIR/import.yaml"
echo "  ✅ Import resources applied"
echo ""

echo "Step 4: Waiting for klusterlet agents to start (30 seconds)..."
sleep 30
echo ""

echo "Step 5: Verifying klusterlet pods..."
echo ""
echo "Klusterlet Agent Pods:"
kubectl --kubeconfig="$MANAGED_KUBECONFIG" get pods -n open-cluster-management-agent
echo ""
echo "Klusterlet Addon Pods (including cluster-proxy):"
kubectl --kubeconfig="$MANAGED_KUBECONFIG" get pods -n open-cluster-management-agent-addon
echo ""

# Check if cluster-proxy agent is running
PROXY_PODS=$(kubectl --kubeconfig="$MANAGED_KUBECONFIG" get pods -n open-cluster-management-agent-addon -l component=cluster-proxy-proxy-agent --no-headers 2>/dev/null | wc -l)
if [ "$PROXY_PODS" -gt 0 ]; then
  echo "✅ SUCCESS: Cluster-proxy agent pods are running!"
else
  echo "⚠️  WARNING: Cluster-proxy agent pods not found. They may still be starting."
  echo "   Run the following command to check again:"
  echo "   kubectl --kubeconfig=$MANAGED_KUBECONFIG get pods -n open-cluster-management-agent-addon -l component=cluster-proxy-proxy-agent"
fi
echo ""
echo "=========================================="
echo "Import manifests applied successfully"
echo "=========================================="
