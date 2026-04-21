##@ OpenShift/Kiali

# OSSM/Sail install scripts vendored under hack/kiali/ (see hack/kiali/UPSTREAM.txt).
OSSM_INSTALL_SCRIPT := $(abspath $(CURDIR)/hack/kiali/install-ossm-release.sh)

OSSM_SCRIPT_FILES := \
	install-ossm-release.sh \
	func-sm.sh \
	func-kiali.sh \
	func-tempo.sh \
	func-minio.sh \
	func-addons.sh \
	func-olm.sh \
	func-log.sh

KIALI_OSSM_SCRIPT_REF ?= master
OSSM_RAW_BASE := https://raw.githubusercontent.com/kiali/kiali/$(KIALI_OSSM_SCRIPT_REF)/hack/istio/sail

.PHONY: setup-kiali-openshift
setup-kiali-openshift: ## OpenShift: install operators + Istio/Kiali (hack/kiali/install-ossm-release.sh)
	@test -f '$(OSSM_INSTALL_SCRIPT)' || { echo "Missing $(OSSM_INSTALL_SCRIPT). Run explicitly: make update-ossm-install-scripts"; exit 1; }
	@echo "==> OSSM: installing operators (Sail, Kiali, Tempo) ..."
	bash '$(OSSM_INSTALL_SCRIPT)' install-operators
	@echo "==> OSSM: installing Istio, addons, and Kiali CR ..."
	bash '$(OSSM_INSTALL_SCRIPT)' install-istio
	@echo "==> setup-kiali-openshift: done."

ifeq ($(words $(MAKEFILE_LIST)),1)
.DEFAULT_GOAL := setup-kiali-openshift
endif

.PHONY: ossm-install-operators
ossm-install-operators: ## Install only operators (same as first step of setup-kiali-openshift)
	@test -f '$(OSSM_INSTALL_SCRIPT)' || { echo "Missing $(OSSM_INSTALL_SCRIPT). Run explicitly: make update-ossm-install-scripts"; exit 1; }
	bash '$(OSSM_INSTALL_SCRIPT)' install-operators

.PHONY: ossm-install-istio
ossm-install-istio: ## Install only Istio + addons + Kiali CR (same as second step of setup-kiali-openshift)
	@test -f '$(OSSM_INSTALL_SCRIPT)' || { echo "Missing $(OSSM_INSTALL_SCRIPT). Run explicitly: make update-ossm-install-scripts"; exit 1; }
	bash '$(OSSM_INSTALL_SCRIPT)' install-istio

.PHONY: ossm-status
ossm-status: ## Show OSSM/Sail/Kiali/Tempo status via vendored script
	@test -f '$(OSSM_INSTALL_SCRIPT)' || { echo "Missing $(OSSM_INSTALL_SCRIPT). Run explicitly: make update-ossm-install-scripts"; exit 1; }
	bash '$(OSSM_INSTALL_SCRIPT)' status

.PHONY: openshift-kiali-help
openshift-kiali-help: ## List OpenShift/Kiali targets (optional; from repo root use: make help)
	@echo "OpenShift/Kiali — from repo root: make help"
	@echo ""
	@grep -E '^[a-zA-Z0-9_.-]+:.*?##' '$(abspath $(lastword $(MAKEFILE_LIST)))' | sed 's/:.*##/	/'

.PHONY: update-ossm-install-scripts
update-ossm-install-scripts: ## Re-download OSSM scripts from kiali/kiali (maintenance only; KIALI_OSSM_SCRIPT_REF=branch or tag)
	@set -e; d=$$(dirname '$(OSSM_INSTALL_SCRIPT)'); mkdir -p "$$d"; \
	echo "Downloading OSSM scripts (ref=$(KIALI_OSSM_SCRIPT_REF)) into $$d ..."; \
	for f in $(OSSM_SCRIPT_FILES); do \
		printf '  fetch %s ... ' "$$f"; \
		curl -fsSL --connect-timeout 10 --max-time 60 "$(OSSM_RAW_BASE)/$$f" -o "$$d/$$f" && echo ok || { echo fail; exit 1; }; \
	done; \
	chmod a+x "$$d"/*.sh; \
	echo "Done ($$(ls -1 "$$d"/*.sh 2>/dev/null | wc -l) shell files in $$d)."
