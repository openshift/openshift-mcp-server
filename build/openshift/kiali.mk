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
BOOKINFO_NAMESPACE ?= bookinfo
BOOKINFO_CP_NAMESPACE ?= istio-system
# Cluster-scoped Istio CR name (must match IstioRevisionTag metadata.name for stable istio.io/rev=...).
BOOKINFO_ISTIO_CR_NAME ?= default
# Injection label istio.io/rev=... — use the IstioRevisionTag name (same as Istio CR name when install_istio creates the tag).
BOOKINFO_ISTIO_REVISION ?= $(BOOKINFO_ISTIO_CR_NAME)
# Traffic generator ConfigMap route: in-cluster productpage avoids OpenShift Route TLS/503 issues.
BOOKINFO_TRAFFIC_ROUTE ?= http://productpage.$(BOOKINFO_NAMESPACE).svc.cluster.local:9080/productpage
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
	bash '$(OSSM_INSTALL_SCRIPT)' -c '$(BOOKINFO_CLIENT)' install-operators
	@echo "==> OSSM: installing Istio, addons, and Kiali CR ..."
	bash '$(OSSM_INSTALL_SCRIPT)' -c '$(BOOKINFO_CLIENT)' -cpn '$(BOOKINFO_CP_NAMESPACE)' install-istio
	@$(MAKE) -s install-bookinfo-openshift
	@echo "==> Bookinfo: OpenShift routes (productpage / gateways):"
	@'$(BOOKINFO_CLIENT)' get route -n '$(BOOKINFO_NAMESPACE)' 2>/dev/null || true
	@echo "==> setup-kiali-openshift: done."

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
	  -c '$(BOOKINFO_CLIENT)' -n '$(BOOKINFO_NAMESPACE)' -in '$(BOOKINFO_CP_NAMESPACE)' -wt 5m -id "$$istio_id" \
	  -ail "istio.io/rev=$$rev" $(BOOKINFO_SCRIPT_EXTRA); \
	echo "==> Bookinfo: namespace labels for Sail sidecar injection ($(BOOKINFO_MESH_LABELS) istio.io/rev=$$rev)"; \
	'$(BOOKINFO_CLIENT)' label namespace '$(BOOKINFO_NAMESPACE)' istio-injection- 2>/dev/null || true; \
	'$(BOOKINFO_CLIENT)' label namespace '$(BOOKINFO_NAMESPACE)' $(BOOKINFO_MESH_LABELS) istio.io/rev="$$rev" --overwrite; \
	echo "==> Bookinfo: rollout restart so pods join the mesh"; \
	'$(BOOKINFO_CLIENT)' rollout restart deployment --all -n '$(BOOKINFO_NAMESPACE)' 2>/dev/null || true; \
	'$(BOOKINFO_CLIENT)' rollout restart statefulset --all -n '$(BOOKINFO_NAMESPACE)' 2>/dev/null || true; \
	tg_route='$(BOOKINFO_TRAFFIC_ROUTE)'; \
	if '$(BOOKINFO_CLIENT)' get configmap traffic-generator-config -n '$(BOOKINFO_NAMESPACE)' -o name >/dev/null 2>&1; then \
	  patch=$$(printf '%s' '[{"op":"replace","path":"/data/route","value":"'"$$tg_route"'"}]'); \
	  '$(BOOKINFO_CLIENT)' patch configmap traffic-generator-config -n '$(BOOKINFO_NAMESPACE)' --type=json -p "$$patch"; \
	  '$(BOOKINFO_CLIENT)' delete pod -n '$(BOOKINFO_NAMESPACE)' -l kiali-test=traffic-generator --ignore-not-found=true --wait=false 2>/dev/null || true; \
	  echo "==> Bookinfo: traffic generator route -> $$tg_route"; \
	fi

.PHONY: ossm-install-operators
ossm-install-operators: ## Install only operators (same as first step of setup-kiali-openshift)
	@test -f '$(OSSM_INSTALL_SCRIPT)' || { echo "Missing $(OSSM_INSTALL_SCRIPT). Expected vendored scripts under hack/kiali/ in this repo."; exit 1; }
	bash '$(OSSM_INSTALL_SCRIPT)' -c '$(BOOKINFO_CLIENT)' -cpn '$(BOOKINFO_CP_NAMESPACE)' install-operators

.PHONY: ossm-install-istio
ossm-install-istio: ## Install only Istio + addons + Kiali CR (same as second step of setup-kiali-openshift)
	@test -f '$(OSSM_INSTALL_SCRIPT)' || { echo "Missing $(OSSM_INSTALL_SCRIPT). Expected vendored scripts under hack/kiali/ in this repo."; exit 1; }
	bash '$(OSSM_INSTALL_SCRIPT)' -c '$(BOOKINFO_CLIENT)' -cpn '$(BOOKINFO_CP_NAMESPACE)' install-istio

.PHONY: ossm-status
ossm-status: ## Show OSSM/Sail/Kiali status via vendored script
	@test -f '$(OSSM_INSTALL_SCRIPT)' || { echo "Missing $(OSSM_INSTALL_SCRIPT). Expected vendored scripts under hack/kiali/ in this repo."; exit 1; }
	bash '$(OSSM_INSTALL_SCRIPT)' status

.PHONY: openshift-kiali-help
openshift-kiali-help: ## List OpenShift/Kiali targets (optional; from repo root use: make help)
	@echo "OpenShift/Kiali — from repo root: make help"
	@echo ""
	@grep -E '^[a-zA-Z0-9_.-]+:.*?##' '$(abspath $(lastword $(MAKEFILE_LIST)))' | sed 's/:.*##/	/'
