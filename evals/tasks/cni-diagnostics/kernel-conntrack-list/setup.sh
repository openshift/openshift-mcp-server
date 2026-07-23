#!/usr/bin/env bash
set -e

# This task runs against a cluster with OVN-Kubernetes CNI
# No specific setup needed - we're just reading conntrack state
# The cluster should have at least one non-control-plane node

# Verify we have at least one non-control-plane node
if ! kubectl get nodes -l '!node-role.kubernetes.io/control-plane,!node-role.kubernetes.io/master' --no-headers | grep -q .; then
    echo "Error: No non-control-plane nodes found. This task requires a cluster with non-control-plane nodes."
    exit 1
fi

echo "Setup complete: cluster has non-control-plane nodes for conntrack testing"
exit 0
