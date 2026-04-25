##@ OpenShift/Kiali

# OSSM/Sail install scripts under hack/kiali/.
OSSM_INSTALL_SCRIPT := $(abspath $(CURDIR)/hack/kiali/install-ossm-release.sh)
# Tracing: Jaeger addon only (no Tempo in vendored OSSM scripts). Mesh Zipkin -> jaeger-collector.<cp-ns>:9411.
# Sail Istio.spec.profile (not "demo" unless you want that preset). Passed to install-ossm-release.sh / func-sm.sh.
OSSM_ISTIO_PROFILE ?= default
export OSSM_ISTIO_PROFILE
INSTALL_ISTIO_CRD_WAIT_SECONDS ?= 720
export INSTALL_ISTIO_CRD_WAIT_SECONDS

# Bookinfo: Kiali hack/istio scripts downloaded with curl into BOOKINFO_DEMO_DIR (see fetch-bookinfo-hack).
BOOKINFO_DEMO_DIR ?= $(abspath $(CURDIR)/hack/kiali/bookinfo-hack)
BOOKINFO_INSTALL_SCRIPT := $(BOOKINFO_DEMO_DIR)/install-bookinfo-demo.sh
KIALI_BOOKINFO_REF ?= master
BOOKINFO_RAW_BASE = https://raw.githubusercontent.com/kiali/kiali/$(KIALI_BOOKINFO_REF)/hack/istio
# Istio full distro for Bookinfo (avoid install-bookinfo-demo.sh picking _output/istio-addons via istio-*).
BOOKINFO_OUTPUT_DIR ?= $(abspath $(CURDIR)/_output)
BOOKINFO_ISTIO_VERSION ?= 1.28.0
BOOKINFO_ISTIO_HOME := $(BOOKINFO_OUTPUT_DIR)/istio-$(BOOKINFO_ISTIO_VERSION)
# Optional: existing Istio tree with bin/istioctl + samples/bookinfo (skips download-bookinfo-istio).
BOOKINFO_ISTIO_DIR ?=
BOOKINFO_CLIENT ?= oc
KUBERNETES_CLI ?= oc
OSSM_OPERATORS_NAMESPACE ?= openshift-operators
BOOKINFO_NAMESPACE ?= bookinfo
BOOKINFO_CP_NAMESPACE ?= istio-system
OSSM_CONSOLE_NAMESPACE ?= ossmconsole
# After install-istio: wait for this Deployment (Kiali server) to exist, then rollout completes before install-kiali-support.
KIALI_DEPLOYMENT_NAME ?= kiali
# Max seconds to poll for deployment/$(KIALI_DEPLOYMENT_NAME) to be created by the operator.
KIALI_DEPLOYMENT_WAIT_MAX ?= 600
# Passed to $(KUBERNETES_CLI) rollout status --timeout=...
KIALI_ROLLOUT_TIMEOUT ?= 600s
# Optional: extra wait for Ready pods (app.kubernetes.io/name=kiali, else app=kiali) if rollout alone is not enough.
KIALI_POD_READY_TIMEOUT ?= 300s
# Cluster-scoped Istio CR name (must match IstioRevisionTag metadata.name for stable istio.io/rev=...).
BOOKINFO_ISTIO_CR_NAME ?= default
# Injection label istio.io/rev=... — use the IstioRevisionTag name (same as Istio CR name when install_istio creates the tag).
BOOKINFO_ISTIO_REVISION ?= $(BOOKINFO_ISTIO_CR_NAME)
# Traffic generator ConfigMap route: in-cluster productpage avoids OpenShift Route TLS/503 issues.
BOOKINFO_TRAFFIC_ROUTE ?= http://productpage.$(BOOKINFO_NAMESPACE).svc.cluster.local:9080/productpage
# Final health check through Kiali API after Bookinfo is installed.
BOOKINFO_KIALI_CLUSTER_NAME ?= Kubernetes
BOOKINFO_HEALTH_WORKLOAD ?= productpage-v1
# install-bookinfo-demo.sh: extra flags only (-ail is set from detected revision in the recipe).
# -tg installs Kiali traffic generator (OpenShift routes must exist; script waits after expose).
BOOKINFO_SCRIPT_EXTRA ?= -tg
# Namespace labels after the script (istio.io/rev=... is appended after revision detection).
# Do NOT set istio-injection=enabled together with istio.io/rev — use rev + istio-discovery only.
BOOKINFO_MESH_LABELS ?= istio-discovery=enabled

.PHONY: fetch-bookinfo-hack
fetch-bookinfo-hack: ## Download Kiali hack/istio bookinfo scripts (curl; ref KIALI_BOOKINFO_REF=branch|tag|commit)
	@set -e; d='$(BOOKINFO_DEMO_DIR)'; ref='$(KIALI_BOOKINFO_REF)'; base='$(BOOKINFO_RAW_BASE)'; \
	if [ -f "$$d/.fetched-ref" ] && [ "$$(cat "$$d/.fetched-ref")" = "$$ref" ] && [ -f "$$d/install-bookinfo-demo.sh" ] && [ -f "$$d/functions.sh" ]; then \
	  echo "Bookinfo hack already present ($$ref) in $$d"; exit 0; \
	fi; \
	echo "Fetching Kiali bookinfo hack ($$ref) -> $$d"; \
	mkdir -p "$$d/kustomization" "$$d/bookinfo-traffic"; \
	for f in install-bookinfo-demo.sh functions.sh istio-gateway.yaml download-istio.sh; do \
	  echo "  curl $$f"; curl -fsSL --connect-timeout 10 --max-time 120 "$$base/$$f" -o "$$d/$$f"; \
	done; \
	chmod a+x "$$d/install-bookinfo-demo.sh" "$$d/download-istio.sh"; \
	curl -fsSL --max-time 120 "$$base/kustomization/bookinfo-ppc64le.yaml" -o "$$d/kustomization/bookinfo-ppc64le.yaml"; \
	curl -fsSL --max-time 120 "$$base/kustomization/bookinfo-s390x.yaml" -o "$$d/kustomization/bookinfo-s390x.yaml"; \
	curl -fsSL --max-time 120 "$$base/bookinfo-traffic/http-route-productpage-v1.yaml" -o "$$d/bookinfo-traffic/http-route-productpage-v1.yaml"; \
	printf '%s\n' "$$ref" > "$$d/.fetched-ref"; \
	echo "Done."

.PHONY: download-bookinfo-istio
# Trust: tarball + .sha256 come only from https://github.com/istio/istio/releases/download/<ver>/
# (no istio.io piped installer). Verify digest before tar -xzf.
download-bookinfo-istio: ## Download Istio release from GitHub (tar.gz + .sha256 verify) for Bookinfo
	@set -e; \
	if [ -n "$(BOOKINFO_ISTIO_DIR)" ]; then echo "BOOKINFO_ISTIO_DIR set to $(BOOKINFO_ISTIO_DIR); skip download"; exit 0; fi; \
	dest='$(BOOKINFO_ISTIO_HOME)'; out='$(BOOKINFO_OUTPUT_DIR)'; ver_raw='$(BOOKINFO_ISTIO_VERSION)'; ver=$${ver_raw#v}; \
	if [ -x "$$dest/bin/istioctl" ]; then echo "Istio already present at $$dest"; exit 0; fi; \
	os=$$(uname -s); uarch=$$(uname -m); \
	case "$$os:$$uarch" in \
	  Linux:x86_64) tuple=linux-amd64 ;; \
	  Linux:aarch64|Linux:arm64) tuple=linux-arm64 ;; \
	  Darwin:arm64) tuple=osx-arm64 ;; \
	  Darwin:x86_64) tuple=osx ;; \
	  *) echo "Unsupported OS/arch $$os/$$uarch for Istio release tarball; set BOOKINFO_ISTIO_DIR." >&2; exit 1 ;; \
	esac; \
	base="istio-$$ver-$$tuple"; tgz="$$base.tar.gz"; url="https://github.com/istio/istio/releases/download/$$ver/$$tgz"; \
	echo "Downloading $$url -> $$dest ..."; \
	mkdir -p "$$out"; tmp=$$(mktemp -d); trap 'rm -rf "$$tmp"' EXIT; \
	( cd "$$tmp" && curl -fSL --connect-timeout 15 --max-time 300 -o "$$tgz" "$$url" && \
	  curl -fSL --connect-timeout 15 --max-time 120 -o "$$tgz.sha256" "$$url.sha256" ); \
	if command -v sha256sum >/dev/null 2>&1; then ( cd "$$tmp" && sha256sum -c "$$tgz.sha256" ); \
	elif command -v shasum >/dev/null 2>&1; then ( cd "$$tmp" && shasum -a 256 -c "$$tgz.sha256" ); \
	else echo "Need sha256sum or shasum to verify $$tgz" >&2; exit 1; fi; \
	( cd "$$tmp" && tar -xzf "$$tgz" ); \
	rm -rf "$$dest"; mv "$$tmp/istio-$$ver" "$$dest"; \
	trap - EXIT; rm -rf "$$tmp"; \
	echo "Istio $$ver ready at $$dest"

.PHONY: setup-kiali-openshift
setup-kiali-openshift: ## OpenShift: OSSM/Sail + Istio/Kiali + Bookinfo (Kiali hack script + OpenShift Routes)
	@test -f '$(OSSM_INSTALL_SCRIPT)' || { echo "Missing $(OSSM_INSTALL_SCRIPT). Expected vendored scripts under hack/kiali/ in this repo."; exit 1; }
	@echo "==> OSSM: installing operators (Sail, Kiali) ..."
	bash '$(OSSM_INSTALL_SCRIPT)' -c '$(KUBERNETES_CLI)' install-operators
	@echo "==> OSSM: installing Istio, addons, and Kiali CR ..."
	bash '$(OSSM_INSTALL_SCRIPT)' -c '$(KUBERNETES_CLI)' -cpn '$(BOOKINFO_CP_NAMESPACE)' install-istio
	@echo "==> Kiali: waiting for server workload (deployment/$(KIALI_DEPLOYMENT_NAME), then ready) ..."
	@set -e; \
	ns='$(BOOKINFO_CP_NAMESPACE)'; d='$(KIALI_DEPLOYMENT_NAME)'; m='$(KIALI_DEPLOYMENT_WAIT_MAX)'; t=0; \
	while ! '$(KUBERNETES_CLI)' get "deployment/$$d" -n "$$ns" -o name >/dev/null 2>&1; do \
	  if [ "$$t" -ge "$$m" ]; then echo "Timeout ($${m}s) waiting for deployment/$$d in namespace $$ns (Kiali operator still reconciling?)." >&2; exit 1; fi; \
	  echo " ... waiting for deployment/$$d ($$t/$$m s)"; \
	  sleep 5; t=$$((t+5)); \
	done; \
	echo "==> Kiali: rollout status (pods become ready) ..."; \
	'$(KUBERNETES_CLI)' rollout status "deployment/$$d" -n "$$ns" --timeout='$(KIALI_ROLLOUT_TIMEOUT)'; \
	if '$(KUBERNETES_CLI)' get pod -n "$$ns" -l 'app.kubernetes.io/name=kiali' -o name >/dev/null 2>&1; then \
	  '$(KUBERNETES_CLI)' wait --for=condition=Ready pod -l 'app.kubernetes.io/name=kiali' -n "$$ns" --timeout='$(KIALI_POD_READY_TIMEOUT)'; \
	elif '$(KUBERNETES_CLI)' get pod -n "$$ns" -l 'app=kiali' -o name >/dev/null 2>&1; then \
	  '$(KUBERNETES_CLI)' wait --for=condition=Ready pod -l 'app=kiali' -n "$$ns" --timeout='$(KIALI_POD_READY_TIMEOUT)'; \
	else \
	  echo " (no pod with app.kubernetes.io/name=kiali or app=kiali; assuming deployment readiness is enough)"; \
	fi
	@echo "==> Kiali: checking version ..."
	bash '$(OSSM_INSTALL_SCRIPT)' -c '$(KUBERNETES_CLI)' install-kiali-support
	@$(MAKE) -s install-bookinfo-openshift
	@$(MAKE) -s validate-bookinfo-kiali-health
	@echo "==> Bookinfo: OpenShift routes (productpage / gateways):"
	@'$(KUBERNETES_CLI)' get route -n '$(BOOKINFO_NAMESPACE)' 2>/dev/null || true
	@echo "==> setup-kiali-openshift: done."

OSSM_DELETE_NAMESPACES ?= yes

.PHONY: clean-kiali-openshift
clean-kiali-openshift: ## OpenShift: remove Bookinfo + Istio/Kiali + operators installed by setup-kiali-openshift
	@test -f '$(OSSM_INSTALL_SCRIPT)' || { echo "Missing $(OSSM_INSTALL_SCRIPT). Expected vendored scripts under hack/kiali/ in this repo."; exit 1; }
	@set -e; \
	cli='$(KUBERNETES_CLI)'; ns='$(BOOKINFO_NAMESPACE)'; \
	echo "==> Step 1/4 - Cleaning mesh resources (Istio/Kiali/addons CRs) ..."; \
	OSSM_DELETE_CONFIRM=yes bash '$(OSSM_INSTALL_SCRIPT)' -c '$(KUBERNETES_CLI)' -cpn '$(BOOKINFO_CP_NAMESPACE)' delete-istio; \
	echo "==> Step 2/4 - Cleaning namespaces (Bookinfo + OSSM Console, then control-plane when enabled) ..."; \
	if [ "$$(basename -- "$$cli")" = "oc" ]; then \
	  $$cli delete project "$$ns" --ignore-not-found=true || true; \
	  $$cli delete project '$(OSSM_CONSOLE_NAMESPACE)' --ignore-not-found=true || true; \
	fi; \
	$$cli delete namespace "$$ns" --ignore-not-found=true || true; \
	$$cli wait --for=delete "namespace/$$ns" --timeout=180s >/dev/null 2>&1 || true; \
	$$cli delete namespace '$(OSSM_CONSOLE_NAMESPACE)' --ignore-not-found=true || true; \
	$$cli wait --for=delete "namespace/$(OSSM_CONSOLE_NAMESPACE)" --timeout=180s >/dev/null 2>&1 || true; \
	if [ "$(OSSM_DELETE_NAMESPACES)" = "yes" ]; then \
	  for doomed_ns in '$(BOOKINFO_CP_NAMESPACE)' istio-cni; do \
	    $$cli delete namespace "$$doomed_ns" --ignore-not-found=true || true; \
	    $$cli wait --for=delete "namespace/$$doomed_ns" --timeout=180s >/dev/null 2>&1 || true; \
	  done; \
	fi; \
	echo "==> Step 3/4 - Cleaning Sail/Kiali operators ..."; \
	OSSM_DELETE_CONFIRM=yes bash '$(OSSM_INSTALL_SCRIPT)' -c '$(KUBERNETES_CLI)' -cpn '$(BOOKINFO_CP_NAMESPACE)' delete-operators; \
	echo "==> Step 4/4 - Final residual cleanup (subscriptions/CSVs/CRDs/routes/SCC) ..."; \
	opns='$(OSSM_OPERATORS_NAMESPACE)'; cpns='$(BOOKINFO_CP_NAMESPACE)'; \
	$$cli delete subscription --ignore-not-found=true -n "$$opns" my-kiali my-sailoperator; \
	csvs="$$( $$cli get csv --all-namespaces --no-headers -o custom-columns=NS:.metadata.namespace,N:.metadata.name 2>/dev/null | awk '$$2 ~ /(kiali-operator|sailoperator|servicemeshoperator3|servicemeshoperator\.|istio-operator|istiooperator)/ {print $$1 ":" $$2}' )"; \
	if [ -n "$$csvs" ]; then \
	  echo "$$csvs" | while IFS=: read -r csv_ns csv_name; do $$cli delete csv -n "$$csv_ns" "$$csv_name" --ignore-not-found=true; done; \
	fi; \
	crds="$$( $$cli get crds -o name 2>/dev/null | awk '$$0 ~ /\.istio\.io$$|\.sailoperator\.io$$|\.servicemesh.*\.io$$/ {print $$0}' )"; \
	if [ -n "$$crds" ]; then echo "$$crds" | while IFS= read -r crd; do $$cli delete "$$crd" --ignore-not-found=true; done; fi; \
	$$cli -n "$$cpns" delete route --ignore-not-found=true kiali istio-ingressgateway; \
	$$cli delete scc istio-addons-scc --ignore-not-found=true 2>/dev/null || true; \
	echo "==> clean-kiali-openshift: done."

.PHONY: setup-kiali-openshift-clean
setup-kiali-openshift-clean: clean-kiali-openshift ## Alias for clean-kiali-openshift

ifeq ($(words $(MAKEFILE_LIST)),1)
.DEFAULT_GOAL := setup-kiali-openshift
endif

.PHONY: install-bookinfo-openshift
install-bookinfo-openshift: fetch-bookinfo-hack download-bookinfo-istio ## Install Bookinfo via Kiali script (always -id to avoid wrong _output/istio-* match)
	@test -f '$(BOOKINFO_INSTALL_SCRIPT)' || { echo "Missing $(BOOKINFO_INSTALL_SCRIPT) after fetch-bookinfo-hack."; exit 1; }
	@set -e; \
	cr='$(BOOKINFO_ISTIO_CR_NAME)'; \
	rev='$(BOOKINFO_ISTIO_REVISION)'; \
	[ -n "$$rev" ] || { echo "Bookinfo: BOOKINFO_ISTIO_REVISION is empty."; exit 1; }; \
	echo "==> Bookinfo: using istio.io/rev=$$rev (IstioRevisionTag / Istio CR name $$cr)"; \
	istio_home='$(BOOKINFO_ISTIO_HOME)'; \
	istio_id='$(BOOKINFO_ISTIO_DIR)'; \
	if [ -z "$$istio_id" ]; then istio_id="$$istio_home"; fi; \
	OUTPUT_DIR='$(BOOKINFO_OUTPUT_DIR)' bash '$(BOOKINFO_INSTALL_SCRIPT)' \
	  -c '$(KUBERNETES_CLI)' -n '$(BOOKINFO_NAMESPACE)' -in '$(BOOKINFO_CP_NAMESPACE)' -wt 5m -id "$$istio_id" \
	  -ail "istio.io/rev=$$rev" $(BOOKINFO_SCRIPT_EXTRA); \
	echo "==> Bookinfo: namespace labels for Sail sidecar injection ($(BOOKINFO_MESH_LABELS) istio.io/rev=$$rev)"; \
	'$(KUBERNETES_CLI)' label namespace '$(BOOKINFO_NAMESPACE)' istio-injection- 2>/dev/null || true; \
	'$(KUBERNETES_CLI)' label namespace '$(BOOKINFO_NAMESPACE)' $(BOOKINFO_MESH_LABELS) istio.io/rev="$$rev" --overwrite; \
	echo "==> Bookinfo: rollout restart so pods join the mesh"; \
	'$(KUBERNETES_CLI)' rollout restart deployment --all -n '$(BOOKINFO_NAMESPACE)' 2>/dev/null || true; \
	'$(KUBERNETES_CLI)' rollout restart statefulset --all -n '$(BOOKINFO_NAMESPACE)' 2>/dev/null || true; \
	tg_route='$(BOOKINFO_TRAFFIC_ROUTE)'; \
	if '$(KUBERNETES_CLI)' get configmap traffic-generator-config -n '$(BOOKINFO_NAMESPACE)' -o name >/dev/null 2>&1; then \
	  patch=$$(printf '%s' '[{"op":"replace","path":"/data/route","value":"'"$$tg_route"'"}]'); \
	  '$(KUBERNETES_CLI)' patch configmap traffic-generator-config -n '$(BOOKINFO_NAMESPACE)' --type=json -p "$$patch"; \
	  '$(KUBERNETES_CLI)' delete pod -n '$(BOOKINFO_NAMESPACE)' -l kiali-test=traffic-generator --ignore-not-found=true --wait=false 2>/dev/null || true; \
	  echo "==> Bookinfo: traffic generator route -> $$tg_route"; \
	fi

.PHONY: validate-bookinfo-kiali-health
validate-bookinfo-kiali-health: ## Wait until Bookinfo workload is Healthy according to Kiali API
	@set -e; \
	client='$(KUBERNETES_CLI)'; \
	cpns='$(BOOKINFO_CP_NAMESPACE)'; \
	ns='$(BOOKINFO_NAMESPACE)'; \
	wl='$(BOOKINFO_HEALTH_WORKLOAD)'; \
	cluster_name='$(BOOKINFO_KIALI_CLUSTER_NAME)'; \
	max_wait="$${BOOKINFO_HEALTH_WAIT_SECONDS:-300}"; \
	retry="$${BOOKINFO_HEALTH_RETRY_SECONDS:-5}"; \
	elapsed=0; \
	echo "==> Kiali health check: waiting for $$ns/$$wl to be Healthy (cluster=$$cluster_name)"; \
	kiali_host="$$( $$client -n "$$cpns" get route kiali -o jsonpath='{.spec.host}' 2>/dev/null || true )"; \
	if [ -z "$$kiali_host" ]; then \
	  echo "Kiali route not found in namespace $$cpns"; \
	  exit 1; \
	fi; \
	kiali_token="$$( $$client whoami -t 2>/dev/null || true )"; \
	if [ -z "$$kiali_token" ]; then \
	  echo "Cannot obtain token from $$client whoami -t (required for Kiali API auth)."; \
	  exit 1; \
	fi; \
	api_path="/api/clusters/workloads?health=true&istioResources=true&namespaces=$$ns&clusterName=$$cluster_name"; \
	api_url_kiali_prefix="https://$$kiali_host/kiali$$api_path"; \
	api_url_root_prefix="https://$$kiali_host$$api_path"; \
	echo "==> Kiali route: https://$$kiali_host"; \
	echo "==> Kiali API check (try #1): $$api_url_kiali_prefix"; \
	echo "==> Kiali API check (try #2): $$api_url_root_prefix"; \
	while true; do \
	  checked_url="$$api_url_kiali_prefix"; \
	  response="$$(curl -ksS --max-time 20 -H "Authorization: Bearer $$kiali_token" "$$checked_url" 2>/dev/null || true)"; \
	  status=""; \
	  if command -v jq >/dev/null 2>&1; then \
	    status="$$(printf '%s' "$$response" | jq -r --arg ns "$$ns" --arg wl "$$wl" '.workloads[]? | select(.namespace == $$ns and .name == $$wl) | .health.status.status' 2>/dev/null | head -n 1)"; \
	  else \
	    if echo "$$response" | tr -d '\n\r' | grep -Eq "\"name\":\"$$wl\".*\"namespace\":\"$$ns\".*\"status\":\\{\"status\":\"Healthy\""; then \
	      status="Healthy"; \
	    fi; \
	  fi; \
	  if [ -z "$$status" ]; then \
	    checked_url="$$api_url_root_prefix"; \
	    response="$$(curl -ksS --max-time 20 -H "Authorization: Bearer $$kiali_token" "$$checked_url" 2>/dev/null || true)"; \
	    if command -v jq >/dev/null 2>&1; then \
	      status="$$(printf '%s' "$$response" | jq -r --arg ns "$$ns" --arg wl "$$wl" '.workloads[]? | select(.namespace == $$ns and .name == $$wl) | .health.status.status' 2>/dev/null | head -n 1)"; \
	    else \
	      if echo "$$response" | tr -d '\n\r' | grep -Eq "\"name\":\"$$wl\".*\"namespace\":\"$$ns\".*\"status\":\\{\"status\":\"Healthy\""; then \
	        status="Healthy"; \
	      fi; \
	    fi; \
	  fi; \
	  if [ "$$status" = "Healthy" ]; then \
	    echo "==> Kiali health check OK: $$ns/$$wl is Healthy (url=$$checked_url)"; \
	    break; \
	  fi; \
	  if [ "$$elapsed" -ge "$$max_wait" ]; then \
	    echo "Timed out after $$max_wait seconds waiting for $$ns/$$wl to be Healthy via Kiali API."; \
	    echo "Last observed status: [$${status:-<missing>}]"; \
	    echo "Last checked URL: $$checked_url"; \
	    echo "Queried path: $$api_path"; \
	    exit 1; \
	  fi; \
	  echo -n "."; \
	  sleep "$$retry"; \
	  elapsed=$$((elapsed + retry)); \
	done; \
	echo ""

.PHONY: ossm-install-operators
ossm-install-operators: ## Install only operators (same as first step of setup-kiali-openshift)
	@test -f '$(OSSM_INSTALL_SCRIPT)' || { echo "Missing $(OSSM_INSTALL_SCRIPT). Expected vendored scripts under hack/kiali/ in this repo."; exit 1; }
	bash '$(OSSM_INSTALL_SCRIPT)' -c '$(KUBERNETES_CLI)' -cpn '$(BOOKINFO_CP_NAMESPACE)' install-operators

.PHONY: ossm-install-istio
ossm-install-istio: ## Install only Istio + addons + Kiali CR (same as second step of setup-kiali-openshift)
	@test -f '$(OSSM_INSTALL_SCRIPT)' || { echo "Missing $(OSSM_INSTALL_SCRIPT). Expected vendored scripts under hack/kiali/ in this repo."; exit 1; }
	bash '$(OSSM_INSTALL_SCRIPT)' -c '$(KUBERNETES_CLI)' -cpn '$(BOOKINFO_CP_NAMESPACE)' install-istio

.PHONY: ossm-status
ossm-status: ## Show OSSM/Sail/Kiali status via vendored script
	@test -f '$(OSSM_INSTALL_SCRIPT)' || { echo "Missing $(OSSM_INSTALL_SCRIPT). Expected vendored scripts under hack/kiali/ in this repo."; exit 1; }
	bash '$(OSSM_INSTALL_SCRIPT)' status

.PHONY: openshift-kiali-help
openshift-kiali-help: ## List OpenShift/Kiali targets (optional; from repo root use: make help)
	@echo "OpenShift/Kiali — from repo root: make help"
	@echo ""
	@grep -E '^[a-zA-Z0-9_.-]+:.*?##' '$(abspath $(lastword $(MAKEFILE_LIST)))' | sed 's/:.*##/	/'