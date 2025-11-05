#!/usr/bin/env bash

# Install ACM operator (Subscription, OperatorGroup, etc.)

set -euo pipefail

echo "Installing ACM Operator (release 2.14)..."
oc apply -k https://github.com/redhat-cop/gitops-catalog/advanced-cluster-management/operator/overlays/release-2.14

# Wait for CSV to appear and get its name
echo "Waiting for ACM operator CSV to be ready..."
CSV_NAME=""
for i in {1..60}; do
    CSV_NAME=$(oc get csv -n open-cluster-management -o name 2>/dev/null | grep advanced-cluster-management || true)
    if [ -n "$CSV_NAME" ]; then
        echo "ACM CSV found: $CSV_NAME"
        break
    fi
    echo "  Waiting for ACM CSV to appear ($i/60)..."
    sleep 5
done

if [ -z "$CSV_NAME" ]; then
    echo "Error: ACM CSV not found after waiting"
    exit 1
fi

# Wait for CSV to be ready
echo "Waiting for CSV to reach Succeeded phase..."
oc wait --for=jsonpath='{.status.phase}'=Succeeded "$CSV_NAME" -n open-cluster-management --timeout=300s

echo "✓ ACM Operator installation complete"