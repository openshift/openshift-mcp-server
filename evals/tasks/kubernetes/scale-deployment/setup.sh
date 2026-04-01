#!/usr/bin/env bash
# Create namespace and a deployment with initial replicas
kubectl delete namespace scale-test --ignore-not-found
kubectl create namespace scale-test
cat <<'EOF' | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-app
  namespace: scale-test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: web-app
  template:
    metadata:
      labels:
        app: web-app
    spec:
      containers:
      - name: app
        image: registry.access.redhat.com/ubi9/ubi-minimal:latest
        command: ["/bin/sh", "-c", "sleep infinity"]
EOF
# Wait for initial deployment to be ready
for i in {1..30}; do
    if kubectl get deployment web-app -n scale-test -o jsonpath='{.status.availableReplicas}' | grep -q "1"; then
        exit 0
    fi
    sleep 2
done

echo "Setup failed for scale-deployment"
exit 1
