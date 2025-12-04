#!/usr/bin/env bash

# Check ACM installation status

set -euo pipefail

echo "=========================================="
echo "ACM Installation Status"
echo "=========================================="
echo ""

echo "Namespaces:"
oc get namespaces | grep -E "(open-cluster-management|multicluster-engine)" || echo "No ACM namespaces found"
echo ""

echo "Operators:"
oc get csv -n open-cluster-management 2>/dev/null || echo "No operators found in open-cluster-management namespace"
echo ""

echo "MultiClusterHub:"
oc get multiclusterhub -n open-cluster-management -o wide 2>/dev/null || echo "No MultiClusterHub found"
echo ""

echo "ACM Pods:"
oc get pods -n open-cluster-management 2>/dev/null || echo "No pods found in open-cluster-management namespace"
echo ""

echo "ManagedClusters:"
oc get managedclusters 2>/dev/null || echo "No ManagedClusters found (this is normal for fresh install)"