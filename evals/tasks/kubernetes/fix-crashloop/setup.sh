#!/usr/bin/env bash
kubectl delete namespace crashloop-test --ignore-not-found
# Create namespace and a deployment with an invalid command that will cause crashloop
kubectl create namespace crashloop-test
cat <<'EOF' | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
  namespace: crashloop-test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: crashloop
  template:
    metadata:
      labels:
        app: crashloop
    spec:
      containers:
      - name: main
        image: registry.access.redhat.com/ubi9/ubi-minimal:latest
        command: ["/bin/sh", "-c", "exit 1"]
EOF

for i in {1..60}; do
    rc=$(kubectl get pods -n crashloop-test -l app=crashloop -o jsonpath='{.items[0].status.containerStatuses[0].restartCount}' 2>/dev/null || echo "0")
    if [[ "${rc}" =~ ^[1-9] ]]; then
        exit 0
    fi
    sleep 1
done
echo "Setup failed: pod did not enter crash loop in time"
exit 1
