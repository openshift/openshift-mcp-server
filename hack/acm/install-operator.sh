#!/usr/bin/env bash

# Install ACM operator (Subscription, OperatorGroup, etc.)

set -euo pipefail

echo "Installing ACM Operator (release 2.14)..."
oc apply -k https://github.com/redhat-cop/gitops-catalog/advanced-cluster-management/operator/overlays/release-2.14

# Wait for CSV to appear
echo "Waiting for ACM operator CSV to be ready..."
for i in {1..60}; do
    if oc get csv -n open-cluster-management -o name 2>/dev/null | grep -q advanced-cluster-management; then
        echo "ACM CSV found, waiting for Succeeded phase..."
        break
    fi
    echo "  Waiting for ACM CSV to appear ($i/60)..."
    sleep 5
done

# Wait for CSV to be ready
oc wait --for=jsonpath='{.status.phase}'=Succeeded csv -l operators.coreos.com/advanced-cluster-management.open-cluster-management -n open-cluster-management --timeout=300s

echo "✓ ACM Operator installation complete"