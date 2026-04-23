##@ E2E Tests

E2E_NAMESPACE ?= openshift-mcp-server-e2e
MCP_LIFECYCLE_OPERATOR_VERSION ?= v0.1.0
MCP_LIFECYCLE_OPERATOR_URL ?= https://github.com/kubernetes-sigs/mcp-lifecycle-operator/releases/download/$(MCP_LIFECYCLE_OPERATOR_VERSION)/install.yaml
MCP_SERVER_IMAGE ?= quay.io/redhat-user-workloads/ocp-mcp-server-tenant/openshift-mcp-server-release-03:latest

.PHONY: e2e-install-operator
e2e-install-operator: ## Install the MCP Lifecycle Operator from upstream release
	@echo "Installing MCP Lifecycle Operator $(MCP_LIFECYCLE_OPERATOR_VERSION)..."
	oc apply -f $(MCP_LIFECYCLE_OPERATOR_URL)
	@echo "Waiting for operator deployment to be available..."
	oc wait deployment -n mcp-lifecycle-operator-system \
		-l control-plane=controller-manager \
		--for=condition=Available \
		--timeout=120s
	@echo "MCP Lifecycle Operator is ready."

.PHONY: e2e-deploy-mcp-server
e2e-deploy-mcp-server: ## Deploy the MCP server via the MCPServer CRD
	@echo "Creating e2e namespace and RBAC..."
	oc apply -f hack/e2e/namespace.yaml
	oc apply -f hack/e2e/rbac.yaml
	oc apply -f hack/e2e/config.yaml
	@echo "Deploying MCPServer CR with image: $(MCP_SERVER_IMAGE)"
	@sed 's|IMAGE_PLACEHOLDER|$(MCP_SERVER_IMAGE)|g' hack/e2e/mcpserver.yaml | oc apply -f -
	@echo "MCPServer CR applied."

.PHONY: e2e-wait-ready
e2e-wait-ready: ## Wait for MCP server deployment to become ready
	@echo "Waiting for MCP server deployment to be available..."
	@oc wait deployment/kubernetes-mcp-server \
		-n $(E2E_NAMESPACE) \
		--for=condition=Available \
		--timeout=180s || { \
		echo "Deployment not ready. Dumping debug info..."; \
		echo "--- Pod status ---"; \
		oc get pods -n $(E2E_NAMESPACE) -o wide; \
		echo "--- Pod describe ---"; \
		oc describe pods -n $(E2E_NAMESPACE); \
		echo "--- Pod logs ---"; \
		oc logs -n $(E2E_NAMESPACE) -l mcp-server=kubernetes-mcp-server --tail=100; \
		exit 1; \
	}
	@echo "Verifying Service exists..."
	oc get service/kubernetes-mcp-server -n $(E2E_NAMESPACE)
	@echo "Verifying Service has endpoints..."
	@EP=$$(oc get endpoints/kubernetes-mcp-server -n $(E2E_NAMESPACE) -o jsonpath='{.subsets[0].addresses[0].ip}' 2>/dev/null); \
	if [ -z "$$EP" ]; then \
		echo "No endpoints found for Service. Dumping debug info..."; \
		oc describe service/kubernetes-mcp-server -n $(E2E_NAMESPACE); \
		oc get endpoints/kubernetes-mcp-server -n $(E2E_NAMESPACE) -o yaml; \
		oc get pods -n $(E2E_NAMESPACE) -o wide; \
		exit 1; \
	fi
	@echo "E2E smoke test passed. MCP server is running and reachable."
