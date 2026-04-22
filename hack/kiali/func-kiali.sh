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
  if [ "$(${OC} get kiali --all-namespaces 2> /dev/null | wc -l)" -gt "0" ] ; then
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
  if [ "$(${OC} get ossmconsole --all-namespaces 2> /dev/null | wc -l)" -gt "0" ] ; then
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
