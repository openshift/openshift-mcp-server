#!/usr/bin/env bash
set -euo pipefail

kubectl delete namespace web --ignore-not-found
kubectl create namespace web

# UBI minimal + sleep (OpenShift restricted SCC). Declares containerPort 80 so Service targetPort 80 matches the Pod spec.
# Intentional break: only the Service selector below (points at app=api, not app=http).
cat <<'EOF' | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: http
  namespace: web
spec:
  replicas: 1
  selector:
    matchLabels:
      app: http
  template:
    metadata:
      labels:
        app: http
    spec:
      containers:
      - name: main
        image: registry.access.redhat.com/ubi9/ubi-minimal:latest
        command: ["/bin/sh", "-c", "sleep infinity"]
        ports:
        - containerPort: 80
EOF

cat <<'EOF' | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: http
  namespace: web
spec:
  ports:
  - port: 80
    targetPort: 80
  selector:
    app: api
EOF

kubectl rollout status deployment/http -n web --timeout=180s
