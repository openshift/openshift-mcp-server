#!/bin/bash

##########################################################
#
# This installs Istio and Kiali using the latest
# release of the Sail and Kiali operators.
#
##########################################################

set -u

# Change to the directory where this script is
SCRIPT_ROOT="$( cd "$(dirname "$0")" ; pwd -P )"
cd ${SCRIPT_ROOT}

# get function definitions
source ${SCRIPT_ROOT}/func-sm.sh
source ${SCRIPT_ROOT}/func-kiali.sh
source ${SCRIPT_ROOT}/func-addons.sh
source ${SCRIPT_ROOT}/func-log.sh

# Next-arg must be present and not look like another flag (avoids set -u blow-up on missing values).
ossm_install_require_opt_value() {
  local flag="$1"
  local val="${2:-}"
  if [ -z "${val}" ] || [ "${val#-}" != "${val}" ]; then
    errormsg "Option [${flag}] requires a value. The next argument is missing or starts with '-' (use a non-flag value)."
    exit 1
  fi
}

DEFAULT_CONTROL_PLANE_NAMESPACE="istio-system"
DEFAULT_ENABLE_KIALI="true"
DEFAULT_ENABLE_OSSMCONSOLE="true"
DEFAULT_ADDONS="prometheus grafana jaeger"
DEFAULT_OC="oc"
DEFAULT_ISTIO_VERSION="latest"
DEFAULT_KIALI_VERSION="default"
DEFAULT_CATALOG_SOURCE="redhat"

OSSM_DELETE_ISTIO_NAMESPACES="${OSSM_DELETE_ISTIO_NAMESPACES:-}"

_CMD=""
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in

    # COMMANDS

    install-operators) _CMD="install-operators" ; shift ;;
    install-istio)     _CMD="install-istio"     ; shift ;;
    delete-operators)  _CMD="delete-operators"  ; shift ;;
    delete-istio)      _CMD="delete-istio"      ; shift ;;
    status)            _CMD="status"            ; shift ;;
    kiali-ui)          _CMD="kiali-ui"          ; shift ;;

    # OPTIONS

    -dn|--delete-namespaces)        OSSM_DELETE_ISTIO_NAMESPACES="yes" ; shift ;;

    -a|--addons)                    ossm_install_require_opt_value "${key}" "${2:-}"; ADDONS="${2}"                  ; shift;shift ;;
    -c|--client)                    ossm_install_require_opt_value "${key}" "${2:-}"; OC="${2}"                      ; shift;shift ;;
    -cpn|--control-plane-namespace) ossm_install_require_opt_value "${key}" "${2:-}"; CONTROL_PLANE_NAMESPACE="${2}" ; shift;shift ;;
    -cs|--catalog-source)           ossm_install_require_opt_value "${key}" "${2:-}"; CATALOG_SOURCE="${2}"          ; shift;shift ;;
    -ek|--enable-kiali)             ossm_install_require_opt_value "${key}" "${2:-}"; ENABLE_KIALI="${2}"            ; shift;shift ;;
    -eo|--enable-ossmconsole)       ossm_install_require_opt_value "${key}" "${2:-}"; ENABLE_OSSMCONSOLE="${2}"      ; shift;shift ;;
    -iv|--istio-version)            ossm_install_require_opt_value "${key}" "${2:-}"; ISTIO_VERSION="${2}"           ; shift;shift ;;
    -kv|--kiali-version)            ossm_install_require_opt_value "${key}" "${2:-}"; KIALI_VERSION="${2}"           ; shift;shift ;;

    # HELP

    -h|--help)
      cat <<HELPMSG

$0 [option...] command

Installs Istio using the Sail and Kiali operators.

Valid options:

  -a|--addons <addon names>
      A space-separated list of addon names that will be installed in the control plane namespace.
      This is only used with the "install-istio" command.
      The list of supported addon names are: prometheus, jaeger, grafana, loki
      Default: ${DEFAULT_ADDONS}

  -c|--client <path to k8s client>
      A filename or path to the 'oc' or 'kubectl' client.
      OpenShift is detected when the cluster exposes the route.openshift.io API (not from the binary name).
      Default: ${DEFAULT_OC}

  -cpn|--control-plane-namespace <name>
      The name of the control plane namespace if Istio is to be installed.
      This is only used with the "install-istio" command.
      Default: ${DEFAULT_CONTROL_PLANE_NAMESPACE}

  -dn|--delete-namespaces
      Only with delete-istio: after removing CRs, delete the control plane and CNI namespaces
      inferred from Istio/IstioCNI (destructive). Omitted by default so shared namespaces are not removed.
      If the list includes the namespace from --control-plane-namespace, you must confirm (type yes), or set OSSM_DELETE_CONFIRM=yes when not on a TTY.

  -cs|--catalog-source <redhat|community>
      The name of the OpenShift catalog source where the operators will come from. You can choose
      to install the operators from the RedHat product catalog or the Community catalog.
      Valid values are "redhat" and "community".
      This is only used with the "install-operators" command and when using an OpenShift cluster.
      Default: ${DEFAULT_CATALOG_SOURCE}

  -ek|--enable-kiali <true|false>
      If true, and you elect to install-operators, the Kiali operator is installed
      with the rest of the Service Mesh operators.
      If true, and you elect to install-istio, a Kiali CR and optionally an OSSMConsole CR
      will be created (see --enable-ossmconsole).
      This is ignored when deleting operators (i.e. regardless of this setting, all
      operators are deleted, Kiali operator included).
      This is ignored when deleting Istio (i.e. regardless of this setting, all
      Kiali CRs are deleted).
      Default: ${DEFAULT_ENABLE_KIALI}

  -eo|--enable-ossmconsole <true|false>
      If true, and you elect to enable Kiali (--enable-kiali) this will install OSSMC also.
      This is ignored if you are installing on a non-OpenShift cluster.
      This is only used with the "install-istio" command.
      Default: ${DEFAULT_ENABLE_OSSMCONSOLE}

  -iv|--istio-version
      The version of Istio control plane that will be installed.
      This is only used with the "install-istio" command.
      Default: ${DEFAULT_ISTIO_VERSION}

  Environment OSSM_ISTIO_PROFILE
      Sail Istio CR spec.profile (e.g. default, demo, empty). Default: default.

  Stable revision label (istio.io/rev=default)
      After install-istio, an IstioRevisionTag named like the Istio CR (default: "default")
      references the Istio CR; Sail keeps it pointed at the active IstioRevision.
      Workloads can use istio.io/rev=default instead of default-v1-28-x.

  -kv|--kiali-version
      The version of the Kiali Server and OSSM Console plugin that will be installed.
      This is only used with "install-istio" command and only if Kiali is to be installed.
      Default: ${DEFAULT_KIALI_VERSION}

The command must be one of:

  * install-operators: Install the latest version of the Sail operator and (if --enable-kiali is "true") the Kiali operator.
  * install-istio: Install Istio control plane (you must first have installed the operators). Also installs the configured addons.
  * delete-operators: Delete the Sail and Kiali operators (you must first delete all Istio control planes and Kiali CRs manually).
  * delete-istio: Uninstalls Istio control plane, Kiali, and addons. Namespaces are only deleted if -dn/--delete-namespaces is set (or OSSM_DELETE_ISTIO_NAMESPACES=yes).
  * status: Provides details about resources that have been installed (not including the addons).
  * kiali-ui: Pops up a browser tab pointing to the Kiali UI.

HELPMSG
      exit 0
      ;;
    *)
      errormsg "Unknown argument [$key]. Aborting."
      exit 1
      ;;
  esac
done

# Setup user-defined environment

# Istio.spec.profile (Sail): e.g. default, demo, empty, openshift-ambient — not the Istio CR name.
OSSM_ISTIO_PROFILE="${OSSM_ISTIO_PROFILE:-default}"

CONTROL_PLANE_NAMESPACE="${CONTROL_PLANE_NAMESPACE:-${DEFAULT_CONTROL_PLANE_NAMESPACE}}"
ENABLE_KIALI="${ENABLE_KIALI:-${DEFAULT_ENABLE_KIALI}}"
ENABLE_OSSMCONSOLE="${ENABLE_OSSMCONSOLE:-${DEFAULT_ENABLE_OSSMCONSOLE}}"
ADDONS="${ADDONS:-${DEFAULT_ADDONS}}"
OC="${OC:-${DEFAULT_OC}}"
ISTIO_VERSION="${ISTIO_VERSION:-${DEFAULT_ISTIO_VERSION}}"
KIALI_VERSION="${KIALI_VERSION:-${DEFAULT_KIALI_VERSION}}"
CATALOG_SOURCE="${CATALOG_SOURCE:-${DEFAULT_CATALOG_SOURCE}}"

infomsg "OSSM_ISTIO_PROFILE=$OSSM_ISTIO_PROFILE"
infomsg "CONTROL_PLANE_NAMESPACE=$CONTROL_PLANE_NAMESPACE"
infomsg "ENABLE_KIALI=$ENABLE_KIALI"
infomsg "ENABLE_OSSMCONSOLE=$ENABLE_OSSMCONSOLE"
infomsg "ADDONS=$ADDONS"
infomsg "OC=$OC"
infomsg "ISTIO_VERSION=$ISTIO_VERSION"
infomsg "KIALI_VERSION=$KIALI_VERSION"
infomsg "CATALOG_SOURCE=$CATALOG_SOURCE"

# Open URL in the default browser when possible; otherwise print instructions (SSH, headless, unknown OS).
ossm_open_url_in_browser() {
  local url="$1"
  if [ -z "${url}" ]; then
    errormsg "No URL to open."
    return 1
  fi
  infomsg "Kiali UI URL: ${url}"
  if [ -n "${SSH_CONNECTION:-}" ] || [ -n "${SSH_CLIENT:-}" ]; then
    infomsg "SSH session detected; not opening a browser automatically. Copy the URL above into a browser on your machine."
    return 0
  fi
  local uname_s
  uname_s="$(uname -s 2>/dev/null || true)"
  case "${uname_s}" in
    Darwin)
      if command -v open >/dev/null 2>&1; then
        open "${url}" && return 0
      fi
      ;;
    Linux)
      if { [ -n "${DISPLAY:-}" ] || [ -n "${WAYLAND_DISPLAY:-}" ]; } && command -v xdg-open >/dev/null 2>&1; then
        xdg-open "${url}" >/dev/null 2>&1 && return 0
      fi
      if [ -n "${WSL_DISTRO_NAME:-}" ] && command -v wslview >/dev/null 2>&1; then
        wslview "${url}" >/dev/null 2>&1 && return 0
      fi
      ;;
    CYGWIN*|MSYS*|MINGW*)
      if command -v cmd.exe >/dev/null 2>&1; then
        cmd.exe /c start "" "${url}" >/dev/null 2>&1 && return 0
      fi
      ;;
  esac
  case "${OSTYPE:-}" in
    msys*|cygwin*|mingw*)
      if command -v cmd.exe >/dev/null 2>&1; then
        cmd.exe /c start "" "${url}" >/dev/null 2>&1 && return 0
      fi
      ;;
  esac
  if [ "${OS:-}" = "Windows_NT" ] && command -v cmd.exe >/dev/null 2>&1; then
    cmd.exe /c start "" "${url}" >/dev/null 2>&1 && return 0
  fi
  infomsg "Could not launch a graphical browser (no suitable opener). Open the URL above manually."
}

# Post-install validation: ensure Kiali resources are ready.
ossm_validate_kiali_ready() {
  local max_wait="${OSSM_POST_INSTALL_WAIT_SECONDS:-300}"
  local retry_interval="${OSSM_POST_INSTALL_RETRY_SECONDS:-5}"
  local waited=0

  if [ "${ENABLE_KIALI}" != "true" ]; then
    infomsg "Skipping post-install validation because ENABLE_KIALI is not true."
    return 0
  fi

  infomsg "Post-install validation: waiting for Kiali pod(s) to become ready."
  waited=0
  while true; do
    local kiali_pods
    kiali_pods="$(${OC} -n "${CONTROL_PLANE_NAMESPACE}" get pods -l app.kubernetes.io/name=kiali -o name 2>/dev/null || true)"
    if echo "${kiali_pods}" | grep -q '[^[:space:]]'; then
      if ${OC} -n "${CONTROL_PLANE_NAMESPACE}" wait --for=condition=Ready ${kiali_pods} --timeout="${retry_interval}s" >/dev/null 2>&1; then
        break
      fi
    fi
    if [ "${waited}" -ge "${max_wait}" ]; then
      errormsg "Kiali pod(s) are not ready after ${max_wait}s in namespace [${CONTROL_PLANE_NAMESPACE}]."
      exit 1
    fi
    echo -n "."
    sleep "${retry_interval}"
    waited=$((waited + retry_interval))
  done
  echo ""

  infomsg "Post-install validation: waiting for Kiali service endpoints."
  waited=0
  while true; do
    if ${OC} -n "${CONTROL_PLANE_NAMESPACE}" get endpoints kiali -o jsonpath='{.subsets[0].addresses[0].ip}' 2>/dev/null | grep -q .; then
      break
    fi
    if [ "${waited}" -ge "${max_wait}" ]; then
      errormsg "Kiali service has no ready endpoints after ${max_wait}s in namespace [${CONTROL_PLANE_NAMESPACE}]."
      exit 1
    fi
    echo -n "."
    sleep "${retry_interval}"
    waited=$((waited + retry_interval))
  done
  echo ""
  infomsg "Kiali validation succeeded."
}

# Check the type of cluster we are talking to.
# * OpenShift: cluster exposes route.openshift.io API (not inferred from client binary name).
# * If OpenShift, make sure we are logged in (whoami).
# * If the API probe fails but the client is oc, still require whoami so a logged-out OpenShift session fails clearly.
# * Define the namespace where the operators are expected to run based on cluster type.

if ! which ${OC} >& /dev/null; then
  errormsg "The client is not valid [${OC}]. Use --client to specify a valid path to 'oc' or 'kubectl'."
  exit 1
fi

if ${OC} api-resources --api-group=route.openshift.io -o name 2>/dev/null | grep -q .
then
  IS_OPENSHIFT="true"
  OLM_OPERATORS_NAMESPACE="openshift-operators"
else
  IS_OPENSHIFT="false"
  OLM_OPERATORS_NAMESPACE="operators"
fi

if [ "${IS_OPENSHIFT}" = "true" ]; then
  if ! ${OC} whoami >& /dev/null; then
    errormsg "You are not logged into the OpenShift cluster. Use '${OC} login' to log into a cluster and then retry."
    exit 1
  fi
elif [ "$(basename -- "${OC}")" = "oc" ]; then
  if ! ${OC} whoami >& /dev/null; then
    errormsg "You are not logged into the OpenShift cluster. Use '${OC} login' to log into a cluster and then retry."
    exit 1
  fi
fi

if [ "${IS_OPENSHIFT}" == "true" -a "${CATALOG_SOURCE}" != "redhat" -a "${CATALOG_SOURCE}" != "community" ]; then
  errormsg "The OpenShift catalog source must be one of 'redhat' or 'community' but was [${CATALOG_SOURCE}]"
  exit 1
fi

# Process the command
if [ "${_CMD}" == "install-operators" ]; then

  if [ "${ENABLE_KIALI}" == "true" ]; then
    install_kiali_operator "${CATALOG_SOURCE}"
  fi
  install_servicemesh_operators "${CATALOG_SOURCE}"

elif [ "${_CMD}" == "install-istio" ]; then

  if [ "${ENABLE_KIALI}" == "true" ]; then
    wait_for_cluster_crd "kialis.kiali.io" "Kiali Operator" "${INSTALL_ISTIO_CRD_WAIT_SECONDS:-720}"
  fi

  wait_for_cluster_crd "istios.sailoperator.io" "Sail / Service Mesh operator" "${INSTALL_ISTIO_CRD_WAIT_SECONDS:-720}"

  install_istio "${CONTROL_PLANE_NAMESPACE}" "${ISTIO_VERSION}"

  if [ -n "${ADDONS}" ]; then
    infomsg "Installing addons: ${ADDONS}"
    for addon in ${ADDONS}; do
      if ! install_addon "${addon}"; then
        errormsg "Addon [${addon}] install failed. Aborting."
        exit 1
      fi
    done
  else
    infomsg "No addons will be installed"
  fi

  if [ "${ENABLE_KIALI}" == "true" ]; then
    install_kiali_cr "${CONTROL_PLANE_NAMESPACE}"
    if [ "${IS_OPENSHIFT}" == "true" ]; then
      if [ "${ENABLE_OSSMCONSOLE}" == "true" ]; then
        install_ossmconsole_cr "ossmconsole"
      fi
    fi
  fi

  ossm_validate_kiali_ready

elif [ "${_CMD}" == "delete-operators" ]; then

  delete_kiali_operator
  delete_servicemesh_operators

elif [ "${_CMD}" == "delete-istio" ]; then

  delete_ossmconsole_cr
  delete_kiali_cr
  delete_istio
  delete_all_addons

elif [ "${_CMD}" == "status" ]; then

  status_servicemesh_operators
  status_kiali_operator
  status_istio
  status_kiali_cr
  status_ossmconsole_cr

elif [ "${_CMD}" == "kiali-ui" ]; then

  if [ "${IS_OPENSHIFT}" == "true" ]; then
    kiali_host="$(${OC} -n "${CONTROL_PLANE_NAMESPACE}" get route kiali -o jsonpath='{.spec.host}' 2>/dev/null)"
    if [ -z "${kiali_host}" ]; then
      errormsg "Kiali Route not found in namespace [${CONTROL_PLANE_NAMESPACE}]. Is the Kiali CR ready?"
      exit 1
    fi
    kiali_url="http://${kiali_host}"
  else
    kiali_lb_hostname="$(${OC} -n "${CONTROL_PLANE_NAMESPACE}" get svc kiali -o jsonpath='{.status.loadBalancer.ingress[0].hostname}' 2>/dev/null)"
    kiali_lb_ip="$(${OC} -n "${CONTROL_PLANE_NAMESPACE}" get svc kiali -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)"
    if [ -n "${kiali_lb_hostname}" ]; then
      kiali_lb_host_or_ip="${kiali_lb_hostname}"
    elif [ -n "${kiali_lb_ip}" ]; then
      kiali_lb_host_or_ip="${kiali_lb_ip}"
    else
      errormsg "Kiali Service LoadBalancer ingress is not yet assigned in [${CONTROL_PLANE_NAMESPACE}]."
      exit 1
    fi
    kiali_url="http://${kiali_lb_host_or_ip}:20001"
  fi

  ossm_open_url_in_browser "${kiali_url}"

else
  errormsg "Missing or unknown command. See --help for usage."
  exit 1
fi
