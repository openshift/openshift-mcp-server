#!/usr/bin/env bash
# Initialize namespace and deployment with the old image
kubectl delete namespace rollout-test --ignore-not-found
kubectl create namespace rollout-test
cat <<'EOF' | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-app
  namespace: rollout-test
spec:
  replicas: 3
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
        image: registry.access.redhat.com/ubi8/ubi-minimal:latest
        command: ["/bin/sh", "-c", "sleep infinity"]
EOF

# Wait until all replicas are available
TIMEOUT="120s"
if kubectl wait deployment/web-app -n rollout-test --for=condition=Available=True --timeout=$TIMEOUT; then
  echo "Setup succeeded for rolling-update-deployment"
  exit 0
else
  echo "Setup failed for rolling-update-deployment. Initial deployment did not become ready in time"
  exit 1
fi