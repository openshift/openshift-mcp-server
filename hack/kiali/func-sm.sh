#!/bin/bash

##########################################################
#
# Functions for managing Service Mesh installs.
#
##########################################################

set -u

install_servicemesh_operators() {
  # if not OpenShift, install from OperatorHub.io
  if [ "${IS_OPENSHIFT}" == "false" ]; then
    ${OC} apply -f https://operatorhub.io/install/sailoperator.yaml
    return
  fi

  local catalog_source="${1}"

  case ${catalog_source} in
    redhat)
      local servicemesh_subscription_source="redhat-operators"
      local servicemesh_subscription_name="servicemeshoperator3"
      local servicemesh_subscription_channel="stable"
      ;;
    community)
      local servicemesh_subscription_source="community-operators"
      local servicemesh_subscription_name="sailoperator"
      local servicemesh_subscription_channel="stable"
      ;;
    *)
      local servicemesh_subscription_source="${catalog_source}"
      local servicemesh_subscription_name="servicemeshoperator3"
      local servicemesh_subscription_channel="candidates"
      ;;
  esac

  infomsg "Installing the Service Mesh Operators from the catalog source [${catalog_source}]"
  cat <<EOM | ${OC} apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: my-sailoperator
  namespace: ${OLM_OPERATORS_NAMESPACE}
spec:
  channel: ${servicemesh_subscription_channel}
  installPlanApproval: Automatic
  name: ${servicemesh_subscription_name}
  source: ${servicemesh_subscription_source}
  sourceNamespace: openshift-marketplace
EOM
}

install_istio() {
  local control_plane_namespace="${1}"
  local istio_version="${2}"
  local istio_yaml_file="${3:-}"

  # Obtained this list of CRDs by "oc get crds -oname | grep istio.io". We can't actually do that here programatically
  # because the CRDs may not even be created yet. That's why there is a while loop in here - to wait for them to be created.
  infomsg "Waiting for CRDs to be established."
  for crd in \
     authorizationpolicies.security.istio.io \
     destinationrules.networking.istio.io \
     envoyfilters.networking.istio.io \
     gateways.networking.istio.io \
     istios.sailoperator.io \
     istiocnis.sailoperator.io \
     peerauthentications.security.istio.io \
     proxyconfigs.networking.istio.io \
     requestauthentications.security.istio.io \
     serviceentries.networking.istio.io \
     sidecars.networking.istio.io \
     telemetries.telemetry.istio.io \
     virtualservices.networking.istio.io \
     wasmplugins.extensions.istio.io \
     workloadentries.networking.istio.io \
     workloadgroups.networking.istio.io
  do
    infomsg "Expecting CRD [${crd}] to be established"
    echo -n "Waiting."
    while ! ${OC} get crd ${crd} >& /dev/null ; do echo -n '.'; sleep 1; done
    ${OC} wait --for condition=established crd/${crd}
  done

  infomsg "Expecting Service Mesh operator deployment to be created"
  echo -n "Waiting."
  while ! ${OC} get deployment -n ${OLM_OPERATORS_NAMESPACE} -o name | grep -E 'sail|servicemesh|istio' >& /dev/null ; do echo -n '.'; sleep 1; done
  echo "done."
  local servicemesh_deployment="$(${OC} get deployment -n ${OLM_OPERATORS_NAMESPACE} -o name | grep -E 'sail|servicemesh|istio')"

  infomsg "Waiting for operator deployments to start..."
  for op in ${servicemesh_deployment}
  do
    infomsg "Expecting [${op}] to be ready"
    if ! ${OC} rollout status "${op}" -n "${OLM_OPERATORS_NAMESPACE}" --timeout=300s; then
      errormsg "Timed out waiting for operator deployment [${op}] to become ready."
      exit 1
    fi
  done

  infomsg "Wait for the servicemesh operator to be Ready."
  local operator_pods
  operator_pods="$(${OC} get pod -n ${OLM_OPERATORS_NAMESPACE} -o name 2>/dev/null | grep -E 'sail|servicemesh|istio' || true)"
  if [ -z "${operator_pods}" ]; then
    errormsg "No Sail/ServiceMesh operator pods found in namespace [${OLM_OPERATORS_NAMESPACE}] (cannot oc wait on an empty list)."
    exit 1
  fi
  ${OC} wait --for condition=Ready ${operator_pods} --timeout 300s -n ${OLM_OPERATORS_NAMESPACE}
  infomsg "Servicemesh operator pod(s) Ready (done)."

  # TODO: Sail has no webhooks (yet)
  #infomsg "Wait for the servicemesh validating webhook to be created."
  #while [ "$(${OC} get validatingwebhookconfigurations -o name | grep -E 'sail|servicemesh|istio')" == "" ]; do echo -n '.'; sleep 5; done
  #infomsg "done."
  #
  #infomsg "Wait for the servicemesh mutating webhook to be created."
  #while [ "$(${OC} get mutatingwebhookconfigurations -o name | grep -E 'sail|servicemesh|istio')" == "" ]; do echo -n '.'; sleep 5; done
  #infomsg "done."

  # "latest" is not a supported version when using a released version of Sail operator.
  # We try to determine the latest version of Istio supported by examining the CRD.
  if [ "${istio_version}" == "latest" ]; then
    istio_version="$(${OC} get crd istios.sailoperator.io -o json | jq -r '.spec.versions | sort_by(.name) | .[-1].schema.openAPIV3Schema.properties.spec.properties.version.default')"
    if [ -z "${istio_version}" -o "${istio_version}" == "null" ]; then
      errormsg "Cannot determine the latest supported version of Istio. You must provide an explicit vX.Y.Z version to install via the --istio-version option"
      exit 1
    fi
    infomsg "The latest supported version of Istio is [${istio_version}]. That version will be installed."
  fi

  if ! ${OC} get namespace ${control_plane_namespace} >& /dev/null; then
    infomsg "Creating control plane namespace: ${control_plane_namespace}"
    ${OC} create namespace ${control_plane_namespace}
  fi

  # IstioCNI is required for OpenShift. When on OpenShift, ensure there is one and only one IstioCNI installed.
  # It must be named "default". It will always refer to the namespace "istio-cni".
  if [ "${IS_OPENSHIFT}" == "true" ]; then
    local istiocni_yaml_file="/tmp/istiocni-cr.yaml"
    local istiocni_name="default"
    if ! ${OC} get istiocni ${istiocni_name} >& /dev/null; then
      if ! ${OC} get namespace istio-cni >& /dev/null; then
        infomsg "Creating istio-cni namespace"
        ${OC} create namespace istio-cni
      fi
      infomsg "Installing IstioCNI CR"
      cat <<EOMCNI > ${istiocni_yaml_file}
apiVersion: sailoperator.io/v1
kind: IstioCNI
metadata:
  name: ${istiocni_name}
spec:
  version: ${istio_version}
  namespace: istio-cni
EOMCNI
      while ! ${OC} apply -f ${istiocni_yaml_file}
      do
        errormsg "WARNING: Failed to create IstioCNI CR - will retry in 5 seconds to see if the error condition clears up..."
        sleep 5
      done
      infomsg "IstioCNI has been successfully created"
    else
      infomsg "IstioCNI already exists; will not create another one"
    fi
  else
    infomsg "Not installing on OpenShift; IstioCNI CR will not be created"
  fi

  infomsg "Installing Istio CR"
  if [ "${istio_yaml_file}" == "" ]; then
    # Sail applies implicit "default" + on OpenShift adds "openshift". Avoid "demo" here —
    # extra preset only; see OSSM_ISTIO_PROFILE (Makefile default: default).
    local istio_profile="${OSSM_ISTIO_PROFILE:-default}"

    # Istio jaeger addon: Zipkin ingestion on jaeger-collector:9411 (not the tracing query UI svc).
    local zipkin_address="jaeger-collector.${control_plane_namespace}.svc.cluster.local:9411"
    infomsg "Mesh tracing Zipkin address (Jaeger collector): [${zipkin_address}]"

    local istio_yaml_file="/tmp/istio-cr.yaml"
    cat <<EOM > ${istio_yaml_file}
apiVersion: sailoperator.io/v1
kind: Istio
metadata:
  name: default
spec:
  version: ${istio_version}
  namespace: ${control_plane_namespace}
  updateStrategy:
    type: RevisionBased
  profile: ${istio_profile}
  values:
    meshConfig:
      enableTracing: true
      defaultConfig:
        tracing:
          sampling: 100.0
          zipkin:
            address: "${zipkin_address}"
EOM
  fi

  while ! ${OC} apply -f ${istio_yaml_file}
  do
    errormsg "WARNING: Failed to apply [${istio_yaml_file}] to namespace [${control_plane_namespace}] - will retry in 5 seconds to see if the error condition clears up..."
    sleep 5
  done
  infomsg "[${istio_yaml_file}] has been successfully applied to namespace [${control_plane_namespace}]."

  ensure_istio_revision_tag_default "${istio_yaml_file}"
}

# Maps stable injection label istio.io/rev=<name> to the active control plane (RevisionBased).
# See: https://istio.io/latest/docs/setup/upgrade/canary/#stable-revision-labels
ensure_istio_revision_tag_default() {
  local istio_yaml_file="${1:-}"
  local istio_cr_name="default"
  if [ -n "${istio_yaml_file}" ] && [ -f "${istio_yaml_file}" ]; then
    istio_cr_name="$(${OC} get -f "${istio_yaml_file}" -o jsonpath='{.metadata.name}' 2> /dev/null || true)"
    if [ -z "${istio_cr_name}" ]; then
      istio_cr_name="default"
    fi
  fi
  if ! ${OC} get crd istiorevisiontags.sailoperator.io >& /dev/null; then
    infomsg "IstioRevisionTag CRD not found; skip stable revision tag [${istio_cr_name}]"
    return 0
  fi
  infomsg "Ensuring IstioRevisionTag [${istio_cr_name}] references Istio/${istio_cr_name} (namespaces may use istio.io/rev=${istio_cr_name})"
  local tag_yaml="/tmp/istio-revision-tag-${istio_cr_name}.yaml"
  cat <<EOM > "${tag_yaml}"
apiVersion: sailoperator.io/v1
kind: IstioRevisionTag
metadata:
  name: ${istio_cr_name}
spec:
  targetRef:
    kind: Istio
    name: ${istio_cr_name}
EOM
  while ! ${OC} apply -f "${tag_yaml}"
  do
    errormsg "WARNING: Failed to apply IstioRevisionTag [${istio_cr_name}] - retrying in 5 seconds..."
    sleep 5
  done
  infomsg "IstioRevisionTag [${istio_cr_name}] applied."
}

# Read exactly "yes" from /dev/tty when available (works when stdin is a pipe).
# Otherwise require OSSM_DELETE_CONFIRM=yes (e.g. CI or fully non-interactive).
ossm_prompt_yes_or_env_confirm() {
  local prompt="$1"
  if [ -r /dev/tty ] && [ -w /dev/tty ]; then
    local ans
    read -r -p "${prompt}" ans < /dev/tty || true
    if [ "${ans}" != "yes" ]; then
      errormsg "Deletion aborted (expected exactly 'yes')."
      exit 1
    fi
  elif [ "${OSSM_DELETE_CONFIRM:-}" != "yes" ]; then
    errormsg "No usable /dev/tty for confirmation: set OSSM_DELETE_CONFIRM=yes to proceed, or run from a terminal."
    exit 1
  fi
}

# Print matched resources, require confirmation, then delete each line with oc delete.
# Interactive: type exactly "yes". Non-interactive: set OSSM_DELETE_CONFIRM=yes.
# stdin: one full resource name per line (e.g. customresourcedefinition.apiextensions.k8s.io/foo.bar.istio.io).
ossm_confirm_and_delete_resource_lines() {
  local desc="$1"
  local lines
  lines=$(cat)
  if ! echo "${lines}" | grep -q '[^[:space:]]'; then
    infomsg "No resources matched for [${desc}]; nothing to delete."
    return 0
  fi
  infomsg "---------- Matched for [${desc}] (review before delete) ----------"
  echo "${lines}"
  infomsg "-------------------------------------------------------------------"
  ossm_prompt_yes_or_env_confirm "Type 'yes' to delete these resources: "
  while IFS= read -r res; do
    [ -z "$(echo "${res}" | tr -d '[:space:]')" ] && continue
    ${OC} delete --ignore-not-found=true "${res}"
  done <<EOF
${lines}
EOF
}

delete_servicemesh_operators() {
  local abort_operation="false"
  for cr in \
    $(${OC} get istio             -o custom-columns=K:.kind,N:.metadata.name --no-headers | sed 's/  */:/g' ) \
    $(${OC} get istiocni          -o custom-columns=K:.kind,N:.metadata.name --no-headers | sed 's/  */:/g' ) \
    $(${OC} get istiorevisiontags -o custom-columns=K:.kind,N:.metadata.name --no-headers | sed 's/  */:/g' )
  do
    abort_operation="true"
    local res_kind=$(echo ${cr} | cut -d: -f1)
    local res_name=$(echo ${cr} | cut -d: -f2)
    errormsg "A [${res_kind}] resource named [${res_name}] still exists. You must delete it first."
  done
  if [ "${abort_operation}" == "true" ]; then
    errormsg "Aborting"
    exit 1
  fi

  infomsg "Unsubscribing from the Sail operator"
  ${OC} delete subscription --ignore-not-found=true --namespace ${OLM_OPERATORS_NAMESPACE} my-sailoperator

  infomsg "Deleting OLM CSVs which uninstalls the operators and their related resources"
  local csv_list
  csv_list="$(${OC} get csv --all-namespaces --no-headers -o custom-columns=NS:.metadata.namespace,N:.metadata.name 2>/dev/null | sed 's/  */:/g' | grep -E ':(sailoperator|servicemeshoperator3|servicemeshoperator\.|istio-operator|istiooperator)\.' || true)"
  if echo "${csv_list}" | grep -q '[^[:space:]]'; then
    infomsg "---------- Matched ClusterServiceVersions (tight name prefix) ----------"
    echo "${csv_list}"
    infomsg "-------------------------------------------------------------------------"
    ossm_prompt_yes_or_env_confirm "Type 'yes' to delete these CSVs: "
    while IFS= read -r csv; do
      [ -z "$(echo "${csv}" | tr -d '[:space:]')" ] && continue
      ${OC} delete csv -n "$(echo -n "${csv}" | cut -d: -f1)" "$(echo -n "${csv}" | cut -d: -f2)" --ignore-not-found=true
    done <<EOF
${csv_list}
EOF
  else
    infomsg "No matching CSVs to delete."
  fi

  infomsg "Deleting any cluster-scoped resources that are getting left behind"
  local cr_list
  cr_list="$(${OC} get clusterroles -o name 2>/dev/null | grep -E 'clusterrole\.rbac\.authorization\.k8s\.io/(istio-|mesh-|.*sail.*|.*servicemesh.*)' || true)"
  echo "${cr_list}" | ossm_confirm_and_delete_resource_lines "ClusterRoles (istio-/mesh-/sail/servicemesh name prefix)"

  infomsg "Delete any resources that are getting left behind"
  local leftover_list
  leftover_list="$(${OC} get secrets -n ${OLM_OPERATORS_NAMESPACE} cacerts --no-headers -o custom-columns=K:kind,NS:.metadata.namespace,N:.metadata.name 2>/dev/null | sed 's/  */:/g' || true)"
  leftover_list="${leftover_list}
$(${OC} get configmaps --all-namespaces --no-headers -o custom-columns=K:kind,NS:.metadata.namespace,N:.metadata.name 2>/dev/null | sed 's/  */:/g' | grep -Ei ':configmap:[^:]+:.*(istio|sail|servicemesh)' || true)"
  if echo "${leftover_list}" | grep -q '[^[:space:]]'; then
    infomsg "---------- Matched secrets/configmaps (cacerts + configmap names matching istio|sail|servicemesh) ----------"
    echo "${leftover_list}"
    infomsg "----------------------------------------------------------------------------------------"
    ossm_prompt_yes_or_env_confirm "Type 'yes' to delete these secrets/configmaps: "
    while IFS= read -r r; do
      [ -z "$(echo "${r}" | tr -d '[:space:]')" ] && continue
      local res_kind
      local res_namespace
      local res_name
      res_kind=$(echo "${r}" | cut -d: -f1)
      res_namespace=$(echo "${r}" | cut -d: -f2)
      res_name=$(echo "${r}" | cut -d: -f3)
      infomsg "Deleting resource [${res_name}] of kind [${res_kind}] in namespace [${res_namespace}]"
      ${OC} delete "${res_kind}" -n "${res_namespace}" "${res_name}" --ignore-not-found=true
    done <<EOF
${leftover_list}
EOF
  else
    infomsg "No matching secrets/configmaps to delete."
  fi

  infomsg "Delete the CRDs (anchored API group suffixes only)"
  local crd_list
  crd_list="$(${OC} get crds -o name 2>/dev/null | grep -E '\.istio\.io$|\.sailoperator\.io$|\.servicemesh.*\.io$' || true)"
  echo "${crd_list}" | ossm_confirm_and_delete_resource_lines "CRDs (*.istio.io, *.sailoperator.io, *.servicemesh*.io)"
}

delete_istio() {
  infomsg "Deleting all Istio and IstioCNI CRs (if they exist) which uninstalls all the Service Mesh components"
  local doomed_namespaces=""
  for cr in \
    $(${OC} get istio             -o custom-columns=K:.kind,N:.metadata.name,NS:.spec.namespace --no-headers | sed 's/  */:/g' ) \
    $(${OC} get istiocni          -o custom-columns=K:.kind,N:.metadata.name,NS:.spec.namespace --no-headers | sed 's/  */:/g' ) \
    $(${OC} get istiorevisiontags -o custom-columns=K:.kind,N:.metadata.name,NS:.spec.namespace --no-headers | sed 's/  */:/g' )
  do
    local res_kind=$(echo ${cr} | cut -d: -f1)
    local res_name=$(echo ${cr} | cut -d: -f2)
    local doomed_ns=$(echo ${cr} | cut -d: -f3)
    ${OC} delete ${res_kind} ${res_name}
    doomed_namespaces="$(echo ${doomed_ns} ${doomed_namespaces} | tr ' ' '\n' | sort -u)"
  done

  infomsg "Deleting the control plane and CNI namespaces"
  for ns in ${doomed_namespaces}
  do
    ${OC} delete namespace ${ns}
  done
}

status_servicemesh_operators() {
  infomsg ""
  infomsg "===== SERVICEMESH OPERATOR SUBSCRIPTION"
  local sub_name="$(${OC} get subscriptions -n ${OLM_OPERATORS_NAMESPACE} -o name my-sailoperator 2>/dev/null)"
  if [ ! -z "${sub_name}" ]; then
    ${OC} get --namespace ${OLM_OPERATORS_NAMESPACE} ${sub_name}
    infomsg ""
    infomsg "===== SERVICEMESH OPERATOR PODS"
    local all_pods="$(${OC} get pods -n ${OLM_OPERATORS_NAMESPACE} -o name | grep -E 'sail|servicemesh|istio')"
    [ ! -z "${all_pods}" ] && ${OC} get --namespace ${OLM_OPERATORS_NAMESPACE} ${all_pods} || infomsg "There are no pods"
  else
    infomsg "There are no Subscriptions for the Service Mesh Operators"
  fi
}

status_istio() {
  infomsg ""
  infomsg "===== Istio CRs"
  if [ "$(${OC} get istio 2> /dev/null | wc -l)" -gt "0" ] ; then
    infomsg "One or more Istio CRs exist in the cluster"
    ${OC} get istio
    infomsg ""
    for cr in \
      $(${OC} get istio -o custom-columns=NS:.spec.namespace,N:.metadata.name --no-headers | sed 's/  */:/g' )
    do
      local res_namespace=$(echo ${cr} | cut -d: -f1)
      local res_name=$(echo ${cr} | cut -d: -f2)
      infomsg "Istio [${res_name}] control plane namespace [${res_namespace}]:"
      ${OC} get pods -n ${res_namespace}
    done
  else
    infomsg "There are no Istio CRs in the cluster"
  fi

  infomsg ""
  infomsg "===== IstioCNI CRs"
  if [ "$(${OC} get istiocni 2> /dev/null | wc -l)" -gt "0" ] ; then
    infomsg "One or more Istio CNI CRs exist in the cluster"
    ${OC} get istiocni
    infomsg ""
    for cr in \
      $(${OC} get istiocni -o custom-columns=NS:.spec.namespace,N:.metadata.name --no-headers | sed 's/  */:/g' )
    do
      local res_namespace=$(echo ${cr} | cut -d: -f1)
      local res_name=$(echo ${cr} | cut -d: -f2)
      infomsg "IstioCNI [${res_name}], CNI namespace [${res_namespace}]:"
      ${OC} get pods -n ${res_namespace}
    done
  else
    infomsg "There are no IstioCNI CRs in the cluster"
  fi
}
