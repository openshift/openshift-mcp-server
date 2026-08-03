#!/bin/bash

##########################################################
#
# Functions for managing Kiali installs.
#
##########################################################

set -u

install_kiali_operator() {
  # if not OpenShift, install from OperatorHub.io
  # This will create a subscription with the name "my-kiali"
  if [ "${IS_OPENSHIFT}" == "false" ]; then
    ${OC} apply -f https://operatorhub.io/install/kiali.yaml
    return
  fi

  local catalog_source="${1}"

  case ${catalog_source} in
    redhat)
      local kiali_subscription_source="redhat-operators"
      local kiali_subscription_name="kiali-ossm"
      ;;
    community)
      local kiali_subscription_source="community-operators"
      local kiali_subscription_name="kiali"
      ;;
    *)
      local kiali_subscription_source="${catalog_source}"
      local kiali_subscription_name="kiali-ossm"
      ;;
  esac

  infomsg "Installing the Kiali Operator from the catalog source [${catalog_source}]"
  cat <<EOM | ${OC} apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: my-kiali
  namespace: ${OLM_OPERATORS_NAMESPACE}
spec:
  channel: stable
  installPlanApproval: Automatic
  name: ${kiali_subscription_name}
  source: ${kiali_subscription_source}
  sourceNamespace: openshift-marketplace
  config:
    env:
    - name: ALLOW_ALL_ACCESSIBLE_NAMESPACES
      value: "true"
    - name: ACCESSIBLE_NAMESPACES_LABEL
      value: ""
EOM
}

install_kiali_cr() {
  local control_plane_namespace="${1}"
  infomsg "Installing the Kiali CR after CRD has been established"
  wait_for_cluster_crd "kialis.kiali.io" "Kiali Operator" "${INSTALL_ISTIO_CRD_WAIT_SECONDS:-720}"

  if ! ${OC} get namespace ${control_plane_namespace} >& /dev/null; then
    errormsg "Control plane namespace does not exist [${control_plane_namespace}]"
    exit 1
  fi

  local kiali_auth_strategy="openshift"
  if [ "${KIALI_ANONYMOUS:-}" = "true" ]; then
    kiali_auth_strategy="anonymous"
  fi

  infomsg "Installing Kiali CR with Jaeger tracing (auth.strategy: ${kiali_auth_strategy}; set KIALI_ANONYMOUS=true for anonymous)"
  cat <<EOM | ${OC} apply -f -
apiVersion: kiali.io/v1alpha1
kind: Kiali
metadata:
  name: kiali
  namespace: ${control_plane_namespace}
spec:
  version: ${KIALI_VERSION}
  auth:
    strategy: ${kiali_auth_strategy}
  external_services:
    tracing:
      enabled: true
      provider: jaeger
      in_cluster_url: "http://tracing.${control_plane_namespace}.svc.cluster.local:16685/jaeger"
      use_grpc: true
EOM
}

install_ossmconsole_cr() {
  local ossmconsole_namespace="${1}"
  infomsg "Installing the OSSMConsole CR after CRD has been established"
  wait_for_cluster_crd "ossmconsoles.kiali.io" "Kiali Operator (OSSMConsole)" "${INSTALL_ISTIO_CRD_WAIT_SECONDS:-720}"

  if ! ${OC} get kiali --all-namespaces -o name 2>/dev/null | grep -q .; then
    errormsg "OSSMC cannot be installed because Kiali is not yet installed."
    return 1
  fi

  if ! ${OC} get namespace ${ossmconsole_namespace} >& /dev/null; then
    infomsg "Creating OSSMConsole plugin namespace: ${ossmconsole_namespace}"
    ${OC} create namespace ${ossmconsole_namespace}
  fi

  cat <<EOM | ${OC} apply -f -
apiVersion: kiali.io/v1alpha1
kind: OSSMConsole
metadata:
  name: ossmconsole
  namespace: ${ossmconsole_namespace}
spec:
  version: ${KIALI_VERSION}
EOM
}

ossm_kiali_workload_container_images() {
  local cp_ns="${1}"
  local deploy_name="${2}"
  local images=""

  if ${OC} get "deployment/${deploy_name}" -n "${cp_ns}" >/dev/null 2>&1; then
    images="$(${OC} get "deployment/${deploy_name}" -n "${cp_ns}" -o jsonpath='{range .spec.template.spec.containers[*]}{.image}{" "}{end}' 2>/dev/null || true)"
  fi

  if [ -z "${images}" ]; then
    images="$(${OC} get pods -n "${cp_ns}" -l app.kubernetes.io/name=kiali -o jsonpath='{range .items[*]}{range .spec.containers[*]}{.image}{" "}{end}{end}' 2>/dev/null || true)"
  fi

  if [ -z "${images}" ]; then
    images="$(${OC} get pods -n "${cp_ns}" -l app=kiali -o jsonpath='{range .items[*]}{range .spec.containers[*]}{.image}{" "}{end}{end}' 2>/dev/null || true)"
  fi

  if [ -z "${images}" ]; then
    return 1
  fi

  echo "${images}"
  return 0
}

ossm_kiali_image_tag() {
  local image="${1}"
  local no_digest="${image%%@*}"
  if [ "${no_digest}" = "${image}" ] && [ "${image#*:}" = "${image}" ]; then
    return 1
  fi
  local tag="${no_digest##*:}"
  if [ -z "${tag}" ]; then
    return 1
  fi
  echo "${tag}"
}

ossm_kiali_clean_semver() {
  local raw="${1#v}"
  raw="${raw%%-*}"
  raw="${raw%%+*}"
  echo "${raw}"
}

ossm_kiali_semver_ge() {
  local a
  local b
  a="$(ossm_kiali_clean_semver "${1}")"
  b="$(ossm_kiali_clean_semver "${2}")"
  local a1=0 a2=0 a3=0 b1=0 b2=0 b3=0
  IFS='.' read -r a1 a2 a3 <<<"${a}"
  IFS='.' read -r b1 b2 b3 <<<"${b}"
  a1="${a1:-0}"; a2="${a2:-0}"; a3="${a3:-0}"
  b1="${b1:-0}"; b2="${b2:-0}"; b3="${b3:-0}"
  if [ "${a1}" -gt "${b1}" ]; then return 0; fi
  if [ "${a1}" -lt "${b1}" ]; then return 1; fi
  if [ "${a2}" -gt "${b2}" ]; then return 0; fi
  if [ "${a2}" -lt "${b2}" ]; then return 1; fi
  if [ "${a3}" -ge "${b3}" ]; then return 0; fi
  return 1
}

ossm_kiali_pod_wait_label() {
  local cp_ns="${1}"
  if ${OC} get pods -n "${cp_ns}" -l app.kubernetes.io/name=kiali -o name 2>/dev/null | grep -q .; then
    echo "app.kubernetes.io/name=kiali"
    return 0
  fi
  if ${OC} get pods -n "${cp_ns}" -l app=kiali -o name 2>/dev/null | grep -q .; then
    echo "app=kiali"
    return 0
  fi
  return 1
}

# Ensure Kiali server image is at least OSSM_KIALI_SUPPORT_MIN_VERSION (default v2.25).
# If below, set ALLOW_AD_HOC_KIALI_IMAGE on the operator and patch the Kiali CR to OSSM_KIALI_SUPPORT_TARGET_VERSION.
# $1 = control plane namespace (e.g. istio-system), $2 = Kiali CR name (default: kiali). Deployment is usually the same name.
ossm_install_kiali_support() {
  local cp_ns="${1:-istio-system}"
  local kiali_name="${2:-kiali}"
  local min_ver="${OSSM_KIALI_SUPPORT_MIN_VERSION:-v2.25}"
  local target_ver="${OSSM_KIALI_SUPPORT_TARGET_VERSION:-v2.25}"
  local image_name="${OSSM_KIALI_SUPPORT_IMAGE_NAME:-quay.io/kiali/kiali}"
  local op_ns="${OLM_OPERATORS_NAMESPACE:-openshift-operators}"
  local deploy_name="${OSSM_KIALI_DEPLOYMENT_NAME:-${kiali_name}}"

  if [ "${IS_OPENSHIFT}" != "true" ]; then
    errormsg "install-kiali-support is only supported on OpenShift (requires kiali-operator in ${op_ns})."
    return 1
  fi

  if ! ${OC} get "kiali/${kiali_name}" -n "${cp_ns}" >/dev/null 2>&1; then
    errormsg "No Kiali CR [${kiali_name}] in namespace [${cp_ns}]. Run install-istio (with Kiali) first."
    return 1
  fi

  local images
  if ! images="$(ossm_kiali_workload_container_images "${cp_ns}" "${deploy_name}")"; then
    errormsg "No Kiali workload found in [${cp_ns}]: no Deployment/${deploy_name} and no pods with labels app.kubernetes.io/name=kiali or app=kiali. Is Kiali still provisioning?"
    return 1
  fi

  local first_img="${images%% *}"
  local current_tag
  if ! current_tag="$(ossm_kiali_image_tag "${first_img}")"; then
    errormsg "Could not parse a version tag from image [${first_img}] (digest-based image?)."
    return 1
  fi
  if [ -z "${current_tag}" ]; then
    errormsg "Empty image tag from [${first_img}]."
    return 1
  fi

  infomsg "Detected Kiali image [${first_img}] (tag: [${current_tag}]). Required minimum: [${min_ver}]."

  if ossm_kiali_semver_ge "${current_tag}" "${min_ver}"; then
    infomsg "Kiali server version is OK: [${current_tag}] is greater than or equal to [${min_ver}]. No upgrade needed."
    return 0
  fi

  infomsg "Kiali tag [${current_tag}] is below [${min_ver}]. Enabling ad-hoc image and upgrading to [${target_ver}]..."

  if ! ${OC} set env deploy/kiali-operator -n "${op_ns}" ALLOW_AD_HOC_KIALI_IMAGE=true; then
    errormsg "Failed to set ALLOW_AD_HOC_KIALI_IMAGE on kiali-operator in [${op_ns}]."
    return 1
  fi

  local merge_patch
  merge_patch="{\"spec\":{\"deployment\":{\"image_name\":\"${image_name}\",\"image_version\":\"${target_ver}\",\"override_install_check\":true}}}"
  if ! ${OC} patch "kiali" "${kiali_name}" -n "${cp_ns}" --type merge -p "${merge_patch}"; then
    errormsg "Failed to patch Kiali [${kiali_name}] in [${cp_ns}]."
    return 1
  fi

  infomsg "Waiting for Kiali deployment and pods to become ready..."
  if ${OC} get "deployment/${deploy_name}" -n "${cp_ns}" >/dev/null 2>&1; then
    ${OC} rollout status "deployment/${deploy_name}" -n "${cp_ns}" --timeout=600s
  else
    infomsg "No deployment/${deploy_name} in [${cp_ns}]; waiting on Kiali-labeled pods..."
  fi

  local wait_label="${OSSM_KIALI_POD_WAIT_LABEL:-}"
  if [ -z "${wait_label}" ]; then
    wait_label="$(ossm_kiali_pod_wait_label "${cp_ns}" 2>/dev/null)" || wait_label=""
  fi
  if [ -n "${wait_label}" ]; then
    if ! ${OC} wait --for=condition=Ready pod -l "${wait_label}" -n "${cp_ns}" --timeout=600s; then
      errormsg "Kiali pod did not become Ready within 600s (label ${wait_label}). Check pods in [${cp_ns}]."
      return 1
    fi
  elif ${OC} get "deployment/${deploy_name}" -n "${cp_ns}" >/dev/null 2>&1; then
    if ! ${OC} wait --for=condition=Available "deployment/${deploy_name}" -n "${cp_ns}" --timeout=600s; then
      errormsg "Kiali deployment [${deploy_name}] did not become Available within 600s."
      return 1
    fi
  else
    errormsg "Could not find Kiali pods to wait on (and no deployment/${deploy_name}). Check [${cp_ns}]."
    return 1
  fi

  infomsg "Updated the Kiali server to version [${target_ver}]."
}

delete_kiali_operator() {
  local abort_operation="false"
  for cr in \
    $(${OC} get kiali --all-namespaces -o custom-columns=K:.kind,NS:.metadata.namespace,N:.metadata.name --no-headers | sed 's/  */:/g' ) \
    $(${OC} get ossmconsole --all-namespaces -o custom-columns=K:.kind,NS:.metadata.namespace,N:.metadata.name --no-headers | sed 's/  */:/g' )
  do
    abort_operation="true"
    local res_kind=$(echo ${cr} | cut -d: -f1)
    local res_namespace=$(echo ${cr} | cut -d: -f2)
    local res_name=$(echo ${cr} | cut -d: -f3)
    errormsg "A [${res_kind}] CR named [${res_name}] in namespace [${res_namespace}] still exists. It must be deleted first."
  done
  if [ "${abort_operation}" == "true" ]; then
    errormsg "Aborting"
    exit 1
  fi

  infomsg "Unsubscribing from the Kiali Operator"
  ${OC} delete subscription --ignore-not-found=true --namespace ${OLM_OPERATORS_NAMESPACE} my-kiali

  infomsg "Deleting OLM CSVs which uninstalled the Kiali Operator and its related resources"
  for csv in $(${OC} get csv --all-namespaces --no-headers -o custom-columns=NS:.metadata.namespace,N:.metadata.name | sed 's/  */:/g' | grep kiali-operator)
  do
    ${OC} delete csv -n $(echo -n $csv | cut -d: -f1) $(echo -n $csv | cut -d: -f2)
  done

  infomsg "Delete Kiali CRDs"
  ${OC} get crds -o name | grep '.*\.kiali\.io' | xargs -r -n 1 ${OC} delete
}

delete_kiali_cr() {
  infomsg "Deleting all Kiali CRs in the cluster"
  for cr in $(${OC} get kiali --all-namespaces -o custom-columns=NS:.metadata.namespace,N:.metadata.name --no-headers | sed 's/  */:/g' )
  do
    local res_namespace=$(echo ${cr} | cut -d: -f1)
    local res_name=$(echo ${cr} | cut -d: -f2)
    ${OC} delete -n ${res_namespace} kiali ${res_name}
  done
}

delete_ossmconsole_cr() {
  infomsg "Deleting all OSSMConsole CRs in the cluster"
  for cr in $(${OC} get ossmconsole --all-namespaces -o custom-columns=NS:.metadata.namespace,N:.metadata.name --no-headers | sed 's/  */:/g' )
  do
    local res_namespace=$(echo ${cr} | cut -d: -f1)
    local res_name=$(echo ${cr} | cut -d: -f2)
    ${OC} delete -n ${res_namespace} ossmconsole ${res_name}
  done
}

status_kiali_operator() {
  infomsg ""
  infomsg "===== KIALI OPERATOR SUBSCRIPTION"
  local sub_name="$(${OC} get subscriptions -n ${OLM_OPERATORS_NAMESPACE} -o name my-kiali 2>/dev/null)"
  if [ ! -z "${sub_name}" ]; then
    infomsg "A Subscription exists for the Kiali Operator"
    ${OC} get --namespace ${OLM_OPERATORS_NAMESPACE} ${sub_name}
    infomsg ""
    infomsg "===== KIALI OPERATOR POD"
    local op_name="$(${OC} get pod -n ${OLM_OPERATORS_NAMESPACE} -o name | grep kiali)"
    [ ! -z "${op_name}" ] && ${OC} get --namespace ${OLM_OPERATORS_NAMESPACE} ${op_name} || infomsg "There is no pod"
  else
    infomsg "There is no Subscription for the Kiali Operator"
  fi
}

status_kiali_cr() {
  infomsg ""
  infomsg "===== Kiali CRs"
  if ${OC} get kiali --all-namespaces -o name 2>/dev/null | grep -q .; then
    infomsg "One or more Kiali CRs exist in the cluster"
    ${OC} get kiali --all-namespaces
    infomsg ""
    for cr in \
      $(${OC} get kiali --all-namespaces -o custom-columns=NS:.metadata.namespace,N:.metadata.name --no-headers | sed 's/  */:/g' )
    do
      local res_namespace=$(echo ${cr} | cut -d: -f1)
      local res_name=$(echo ${cr} | cut -d: -f2)
      infomsg "Kiali [${res_name}] namespace [${res_namespace}]:"
      ${OC} get pods --namespace ${res_namespace} -l app.kubernetes.io/name=kiali
      infomsg ""
      infomsg "Kiali Web Console can be accessed here: "
      if [ "${IS_OPENSHIFT}" == "true" ]; then
        ${OC} get route -n ${res_namespace} -l app.kubernetes.io/name=kiali -o jsonpath='https://{..spec.host}{"\n"}'
      else
        infomsg "Cannot determine where the UI is on non-OpenShift clusters."
      fi
    done
  else
    infomsg "There are no Kiali CRs in the cluster"
  fi
}

status_ossmconsole_cr() {
  infomsg ""
  infomsg "===== OSSMConsole CRs"
  if ${OC} get ossmconsole --all-namespaces -o name 2>/dev/null | grep -q .; then
    infomsg "One or more OSSMConsole CRs exist in the cluster"
    ${OC} get ossmconsole --all-namespaces
    infomsg ""
    for cr in \
      $(${OC} get ossmconsole --all-namespaces -o custom-columns=NS:.metadata.namespace,N:.metadata.name --no-headers | sed 's/  */:/g' )
    do
      local res_namespace=$(echo ${cr} | cut -d: -f1)
      local res_name=$(echo ${cr} | cut -d: -f2)
      infomsg "OSSMConsole [${res_name}] namespace [${res_namespace}]:"
      ${OC} get pods --namespace ${res_namespace} -l app.kubernetes.io/name=ossmconsole
      infomsg ""
    done
  else
    infomsg "There are no OSSMConsole CRs in the cluster"
  fi
}

# Wait until OLM has installed an operator that registers this CRD (install-istio runs right after install-operators).
# $1 = crd name, $2 = human description, $3 = max seconds (default 720)
wait_for_cluster_crd() {
  local crd_name="${1}"
  local human="${2:-${crd_name}}"
  local max_wait="${3:-720}"
  local waited=0
  infomsg "Waiting for CRD [${crd_name}] (${human})..."
  while ! ${OC} get crd "${crd_name}" >& /dev/null; do
    if [ "${waited}" -ge "${max_wait}" ]; then
      errormsg "Timeout after ${max_wait}s waiting for CRD [${crd_name}] (${human}). Check the operator Subscription in ${OLM_OPERATORS_NAMESPACE} and catalog source."
      exit 1
    fi
    echo -n "."
    sleep 5
    waited=$((waited + 5))
  done
  echo ""
  if ! ${OC} wait --for condition=established "crd/${crd_name}" --timeout=3m; then
    errormsg "CRD [${crd_name}] did not become Established within 3m."
    exit 1
  fi
  infomsg "CRD [${crd_name}] is ready"
}
