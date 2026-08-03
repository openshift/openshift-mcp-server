#!/usr/bin/env bash
set -e

if ! kubectl get nodes -l '!node-role.kubernetes.io/control-plane,!node-role.kubernetes.io/master' --no-headers | grep -q .; then
    echo "Error: No non-control-plane nodes found"
    exit 1
fi

# Get API server endpoint to verify there should be connections
API_SERVER=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')
echo "API server: $API_SERVER"
echo "Setup complete"
exit 0
