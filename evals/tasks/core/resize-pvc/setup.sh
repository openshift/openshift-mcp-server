#!/usr/bin/env bash
set -euo pipefail

selection=$(kubectl get storageclasses -o json --request-timeout=30s | jq -er '
  [.items[] | select(.allowVolumeExpansion == true)]
  | sort_by([
      (if (.metadata.annotations["storageclass.kubernetes.io/is-default-class"] == "true"
          or .metadata.annotations["storageclass.beta.kubernetes.io/is-default-class"] == "true") then 0 else 1 end),
      .metadata.name])
  | if length == 0 then error("No expandable StorageClass available") else .[0] end
  | [.metadata.name, (.volumeBindingMode // "Immediate")]
  | if (.[0] | type) != "string" or (.[1] | type) != "string" then error("Invalid StorageClass metadata") else @tsv end
')
IFS=$'\t' read -r storage_class binding_mode <<< "$selection"
if [[ ${#storage_class} -gt 253 || ! $storage_class =~ ^[a-z0-9]([-a-z0-9.]*[a-z0-9])?$ ]]; then
    echo "Invalid expandable StorageClass name" >&2
    exit 1
fi
case "$binding_mode" in
    Immediate|WaitForFirstConsumer) ;;
    *) echo "Unsupported StorageClass volumeBindingMode: $binding_mode" >&2; exit 1 ;;
esac
if [[ $(grep -Ec '^[[:space:]]+storage: 1Gi$' artifacts/storage-pvc.yaml) != 1 || $(grep -Ec '^[[:space:]]+storage:' artifacts/storage-pvc.yaml) != 1 ]]; then
    echo "Initial PVC fixture must request 1Gi" >&2
    exit 1
fi
if [[ $(grep -Ec '^[[:space:]]+storageClassName: EXPANDABLE_STORAGE_CLASS$' artifacts/storage-pvc.yaml) != 1 || $(grep -Ec '^[[:space:]]+storageClassName:' artifacts/storage-pvc.yaml) != 1 ]]; then
    echo "PVC fixture must contain exactly one expandable StorageClass placeholder" >&2
    exit 1
fi

rendered_pvc=$(mktemp)
trap 'rm -f "$rendered_pvc"' EXIT
sed "s/EXPANDABLE_STORAGE_CLASS/$storage_class/" artifacts/storage-pvc.yaml > "$rendered_pvc"
namespace_created=false
cleanup_failure() {
    local status=$?
    trap - ERR
    if [[ $namespace_created == true ]]; then
        if ! kubectl delete namespace resize-pv --ignore-not-found --wait=false --timeout=30s --request-timeout=30s; then
            echo "Failed to clean up task namespace resize-pv" >&2
        fi
    fi
    exit "$status"
}
trap cleanup_failure ERR

kubectl delete namespace resize-pv --ignore-not-found --timeout=120s --request-timeout=30s
kubectl create namespace resize-pv --request-timeout=30s
namespace_created=true

kubectl apply -f "$rendered_pvc" --request-timeout=30s
if [[ $binding_mode == WaitForFirstConsumer ]]; then
    kubectl apply -f artifacts/storage-pod.yaml --request-timeout=30s
fi
kubectl wait --namespace resize-pv --for=jsonpath='{.status.phase}'=Bound pvc/storage-pvc --timeout=120s --request-timeout=30s
kubectl get pvc storage-pvc --namespace resize-pv -o json --request-timeout=30s | jq -e --arg storage_class "$storage_class" '
  .status.phase == "Bound"
  and .spec.resources.requests.storage == "1Gi"
  and .spec.storageClassName == $storage_class
  and .spec.accessModes == ["ReadWriteOnce"]
' > /dev/null
if [[ $binding_mode == Immediate ]]; then
    kubectl apply -f artifacts/storage-pod.yaml --request-timeout=30s
fi
kubectl wait --namespace resize-pv --for=condition=Ready pod/storage-pod --timeout=120s --request-timeout=30s
