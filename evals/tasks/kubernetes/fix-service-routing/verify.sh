#!/usr/bin/env bash
set -euo pipefail

# No kubectl run / wget (PodSecurity restricted + workload may not listen on :80).

if ! kubectl wait --for=condition=ready pod -l app=http -n web --timeout=180s 2>/dev/null; then
    echo "Failed: no Ready pod with label app=http in namespace web"
    kubectl get pods -n web -o wide 2>/dev/null || true
    exit 1
fi

# jsonpath {.} over addresses[] is empty on many kubectl builds; go-template can also be empty
# depending on client wiring. Parse JSON so we match what `kubectl get -o wide` shows.
addrs=""
if command -v python3 >/dev/null 2>&1; then
  addrs=$(kubectl get endpointslice.discovery.k8s.io -n web -l kubernetes.io/service-name=http -o json 2>/dev/null | python3 -c "
import json, sys
try:
    d = json.load(sys.stdin)
    for it in d.get('items', []):
        for ep in it.get('endpoints') or []:
            for a in ep.get('addresses') or []:
                print(a, end=' ')
except Exception:
    pass
" 2>/dev/null || true)
fi
if [[ -z "${addrs// }" ]]; then
  addrs=$(kubectl get endpointslice.discovery.k8s.io -n web -l kubernetes.io/service-name=http \
    -o go-template='{{range .items}}{{range .endpoints}}{{range .addresses}}{{.}} {{end}}{{end}}{{end}}' 2>/dev/null || true)
fi
if [[ -z "${addrs// }" ]]; then
    echo "Failed: EndpointSlice for service http has no endpoint addresses"
    kubectl get endpointslice -n web -l kubernetes.io/service-name=http -o wide 2>/dev/null || true
    exit 1
fi

exit 0
