#!/usr/bin/env bash
set -euo pipefail

kubectl delete pods -A -l app.kubernetes.io/component=node-debug --ignore-not-found --wait=true --timeout=120s
kubectl delete namespace tcpdump-test --ignore-not-found
