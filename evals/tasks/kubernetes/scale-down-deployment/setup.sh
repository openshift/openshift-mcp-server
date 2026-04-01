#!/usr/bin/env bash
set -euo pipefail

kubectl delete namespace scale-down-test --ignore-not-found
kubectl create namespace scale-down-test

cat <<'EOF' | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-service
  namespace: scale-down-test
spec:
  replicas: 2
  selector:
    matchLabels:
      app: web-service
  template:
    metadata:
      labels:
        app: web-service
    spec:
      containers:
      - name: app
        image: registry.access.redhat.com/ubi9/ubi-minimal:latest
        command: ["/bin/sh", "-c", "sleep infinity"]
EOF

if ! kubectl rollout status deployment/web-service -n scale-down-test --timeout=180s; then
  echo "Setup failed: deployment web-service did not become ready in time"
  kubectl get deploy,pods -n scale-down-test -o wide 2>/dev/null || true
  exit 1
fi
