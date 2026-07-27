#!/usr/bin/env bash
set -e

# Verify we have at least one non-control-plane node
if ! kubectl get nodes -l '!node-role.kubernetes.io/control-plane,!node-role.kubernetes.io/master' --no-headers | grep -q .; then
    echo "Error: No non-control-plane nodes found"
    exit 1
fi

echo "Setup complete: cluster has non-control-plane nodes for iptables testing"
exit 0
