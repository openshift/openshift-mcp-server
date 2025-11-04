#!/usr/bin/env bash

# Uninstall ACM (reverse order: instance first, then operator)

set -euo pipefail

echo "Uninstalling ACM Instance..."
oc delete multiclusterhub multiclusterhub -n open-cluster-management 2>/dev/null || true

echo "Waiting for MultiClusterHub to be deleted..."
oc wait --for=delete multiclusterhub/multiclusterhub -n open-cluster-management --timeout=300s 2>/dev/null || true

echo "Uninstalling ACM Operator..."
oc delete -k https://github.com/redhat-cop/gitops-catalog/advanced-cluster-management/operator/overlays/release-2.14 2>/dev/null || true

echo "Cleaning up namespaces..."
oc delete namespace open-cluster-management --timeout=300s 2>/dev/null || true

echo "✓ ACM uninstallation complete"