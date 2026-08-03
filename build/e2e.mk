##@ E2E Tests

UV = $(shell pwd)/_output/tools/bin/uv
UV_VERSION = 0.7.15
E2E_IMAGE ?= localhost/kubernetes-mcp-server:e2e
E2E_DIR = test/e2e

KUBECTL = $(shell pwd)/_output/tools/bin/kubectl
KUBECTL_VERSION ?= v1.36.2

HELM = $(shell pwd)/_output/tools/bin/helm
HELM_VERSION ?= v3.21.2

.PHONY: uv
uv:
	@[ -f $(UV) ] || { \
		set -e ;\
		echo "Installing uv $(UV_VERSION) to $$(dirname $(UV))..." ;\
		mkdir -p $$(dirname $(UV)) ;\
		curl -LsSf https://astral.sh/uv/$(UV_VERSION)/install.sh | env UV_INSTALL_DIR=$$(dirname $(UV)) INSTALLER_NO_MODIFY_PATH=1 sh ;\
	}

# Download and install kubectl if not already installed
.PHONY: kubectl
kubectl:
	@[ -f $(KUBECTL) ] || { \
		set -e ;\
		echo "Installing kubectl $(KUBECTL_VERSION) to $(KUBECTL)..." ;\
		mkdir -p $$(dirname $(KUBECTL)) ;\
		OS=$$(uname -s | tr '[:upper:]' '[:lower:]') ;\
		ARCH=$$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') ;\
		curl -L https://dl.k8s.io/release/$(KUBECTL_VERSION)/bin/$${OS}/$${ARCH}/kubectl -o $(KUBECTL) ;\
		chmod +x $(KUBECTL) ;\
	}

# Download and install helm if not already installed
.PHONY: helm
helm:
	@[ -f $(HELM) ] || { \
		set -e ;\
		echo "Installing helm $(HELM_VERSION) to $(HELM)..." ;\
		mkdir -p $$(dirname $(HELM)) ;\
		TMPDIR=$$(mktemp -d) ;\
		OS=$$(uname -s | tr '[:upper:]' '[:lower:]') ;\
		ARCH=$$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') ;\
		curl -L https://get.helm.sh/helm-$(HELM_VERSION)-$${OS}-$${ARCH}.tar.gz | tar xz -C $$TMPDIR ;\
		mv $$TMPDIR/$${OS}-$${ARCH}/helm $(HELM) ;\
		rm -rf $$TMPDIR ;\
	}

.PHONY: e2e-image
e2e-image: ## Build the e2e container image and load it into the Kind cluster
	$(CONTAINER_ENGINE) build -t $(E2E_IMAGE) .
	@if [ "$(shell uname -s)" != "Darwin" ] && echo "$(CONTAINER_ENGINE)" | grep -q "podman"; then \
		export KIND_EXPERIMENTAL_PROVIDER=podman; \
	fi; \
	$(KIND) load docker-image $(E2E_IMAGE) --name $(KIND_CLUSTER_NAME)

.PHONY: e2e-full-setup
e2e-full-setup: kind-create-cluster kuadrant-setup e2e-image ## Create Kind cluster, install Kuadrant MCP Gateway, and build e2e image

.PHONY: e2e-test
e2e-test: uv kubectl helm ## Run all e2e tests (auto-skips tests whose infrastructure is missing)
	KUBECTL_BIN=$(KUBECTL) HELM_BIN=$(HELM) MCP_SERVER_IMAGE=$(E2E_IMAGE) $(UV) run --directory $(E2E_DIR) --locked pytest -v $(PYTEST_ARGS)
