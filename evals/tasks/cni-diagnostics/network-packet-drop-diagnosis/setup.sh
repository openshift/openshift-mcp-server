#!/usr/bin/env bash
set -e

if ! kubectl get nodes -l '!node-role.kubernetes.io/control-plane,!node-role.kubernetes.io/master' --no-headers | grep -q .; then
    echo "Error: No non-control-plane nodes found"
    exit 1
fi

kubectl delete namespace packet-drop-test --ignore-not-found
kubectl create namespace packet-drop-test

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: backend-pod
  namespace: packet-drop-test
  labels:
    app: backend
spec:
  securityContext:
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
  containers:
  - name: nginx
    image: quay.io/nginx/nginx-unprivileged:1.29
    ports:
    - containerPort: 8080
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
          - ALL
      runAsNonRoot: true
      seccompProfile:
        type: RuntimeDefault
---
apiVersion: v1
kind: Service
metadata:
  name: backend-svc
  namespace: packet-drop-test
spec:
  selector:
    app: backend
  ports:
  - port: 8080
    targetPort: 8080
---
apiVersion: v1
kind: Pod
metadata:
  name: client-pod
  namespace: packet-drop-test
spec:
  securityContext:
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
  containers:
  - name: busybox
    image: quay.io/quay/busybox:latest
    command: ["sleep", "3600"]
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
          - ALL
      runAsNonRoot: true
      seccompProfile:
        type: RuntimeDefault
EOF

kubectl wait --for=condition=Ready pod/backend-pod -n packet-drop-test --timeout=60s
kubectl wait --for=condition=Ready pod/client-pod -n packet-drop-test --timeout=60s

SERVICE_IP=$(kubectl get svc backend-svc -n packet-drop-test -o jsonpath='{.spec.clusterIP}')
CLIENT_IP=$(kubectl get pod client-pod -n packet-drop-test -o jsonpath='{.status.podIP}')
echo "Backend service IP: $SERVICE_IP"
echo "Client pod IP: $CLIENT_IP"

echo "Setup complete: client and backend pods created for service connectivity troubleshooting"
exit 0
