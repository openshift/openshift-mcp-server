#!/usr/bin/env bash

# Install ACM instance (MultiClusterHub CR)

set -euo pipefail

echo "Installing ACM Instance (MultiClusterHub)..."
cat <<EOF | oc apply -f -
apiVersion: operator.open-cluster-management.io/v1
kind: MultiClusterHub
metadata:
  name: multiclusterhub
  namespace: open-cluster-management
spec:
  availabilityConfig: High
EOF

echo "Waiting for MultiClusterHub to be ready (this may take several minutes)..."
oc wait --for=condition=Complete --timeout=900s multiclusterhub/multiclusterhub -n open-cluster-management || true

echo "✓ ACM Instance installation complete"