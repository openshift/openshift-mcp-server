##@ E2E Tests

E2E_NAMESPACE ?= openshift-mcp-server-e2e
MCP_LIFECYCLE_OPERATOR_VERSION ?= v0.1.0
MCP_LIFECYCLE_OPERATOR_URL ?= https://github.com/kubernetes-sigs/mcp-lifecycle-operator/releases/download/$(MCP_LIFECYCLE_OPERATOR_VERSION)/install.yaml
MCP_SERVER_IMAGE ?= quay.io/redhat-user-workloads/ocp-mcp-server-tenant/openshift-mcp-server-release-03:latest
MCP_GATEWAY_VERSION ?= v0.6.0
MCP_GATEWAY_INSTALL_URL ?= https://github.com/Kuadrant/mcp-gateway/config/install?ref=$(MCP_GATEWAY_VERSION)

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

.PHONY: e2e-install-gateway
e2e-install-gateway: ## Install the MCP Gateway controller and broker
	@echo "Installing MCP Gateway $(MCP_GATEWAY_VERSION)..."
	oc apply -k $(MCP_GATEWAY_INSTALL_URL) || true
	@echo "Waiting for MCP Gateway CRDs to be established..."
	oc wait crd/mcpgatewayextensions.mcp.kuadrant.io --for=condition=Established --timeout=60s
	oc wait crd/mcpserverregistrations.mcp.kuadrant.io --for=condition=Established --timeout=60s
	oc apply -k $(MCP_GATEWAY_INSTALL_URL)
	@echo "Applying Gateway and ReferenceGrants..."
	oc apply -f hack/e2e/gateway.yaml
	@echo "Waiting for mcp-gateway-controller deployment to be available..."
	oc wait deployment/mcp-gateway-controller \
		-n mcp-system \
		--for=condition=Available \
		--timeout=180s
	@echo "Waiting for mcp-broker-router deployment to be available..."
	oc wait deployment/mcp-broker-router \
		-n mcp-system \
		--for=condition=Available \
		--timeout=180s
	@echo "MCP Gateway is ready."

.PHONY: e2e-deploy-mcp-server
e2e-deploy-mcp-server: ## Deploy the MCP server via the MCPServer CRD
	@echo "Creating e2e namespace and RBAC..."
	oc apply -f hack/e2e/namespace.yaml
	oc apply -f hack/e2e/rbac.yaml
	oc apply -f hack/e2e/config.yaml
	@echo "Deploying MCPServer CR with image: $(MCP_SERVER_IMAGE)"
	@sed 's|IMAGE_PLACEHOLDER|$(MCP_SERVER_IMAGE)|g' hack/e2e/mcpserver.yaml | oc apply -f -
	@echo "MCPServer CR applied."

.PHONY: e2e-register-mcp-server
e2e-register-mcp-server: ## Register the MCP server with the MCP Gateway
	@echo "Applying HTTPRoute and MCPServerRegistration..."
	oc apply -f hack/e2e/mcpserver-registration.yaml
	@echo "Verifying MCPServerRegistration exists..."
	oc get mcpserverregistration/kubernetes-mcp-server -n $(E2E_NAMESPACE)
	@echo "MCPServerRegistration applied."

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
	@echo "Creating Route for direct MCP server access..."
	oc expose svc/kubernetes-mcp-server -n $(E2E_NAMESPACE)
	@echo "Sending direct MCP initialize request via Route..."
	@ROUTE_HOST=$$(oc get route/kubernetes-mcp-server -n $(E2E_NAMESPACE) -o jsonpath='{.spec.host}'); \
	if [ -z "$$ROUTE_HOST" ]; then \
		echo "Route has no host assigned."; \
		oc get route/kubernetes-mcp-server -n $(E2E_NAMESPACE) -o yaml; \
		exit 1; \
	fi; \
	echo "Route host: $$ROUTE_HOST"; \
	HTTP_CODE=$$(curl -s -o /tmp/mcp-direct.json -w '%{http_code}' --max-time 10 \
		-X POST "http://$$ROUTE_HOST/mcp" \
		-H "Content-Type: application/json" \
		-d '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"e2e-direct-test","version":"1.0.0"}},"id":1}'); \
	echo "Direct HTTP status: $$HTTP_CODE"; \
	cat /tmp/mcp-direct.json 2>/dev/null; echo; \
	if [ "$$HTTP_CODE" != "200" ]; then \
		echo "Direct MCP request failed with HTTP $$HTTP_CODE."; \
		oc logs -n $(E2E_NAMESPACE) -l mcp-server=kubernetes-mcp-server --tail=50; \
		exit 1; \
	fi; \
	if ! grep -q '"result"' /tmp/mcp-direct.json; then \
		echo "Direct response missing expected 'result' field."; \
		cat /tmp/mcp-direct.json; \
		exit 1; \
	fi
	@echo "Verifying MCPServerRegistration exists..."
	oc get mcpserverregistration/kubernetes-mcp-server -n $(E2E_NAMESPACE)
	@echo "E2E smoke test passed. MCP server is running and reachable via MCP Gateway."

.PHONY: e2e-smoke-test
e2e-smoke-test: ## Smoke test the MCP server through the MCP Gateway
	@echo "Retrieving Gateway address..."
	@GW_HOST=""; \
	for i in $$(seq 1 30); do \
		GW_HOST=$$(oc get gateway mcp-gateway -n gateway-system -o jsonpath='{.status.addresses[0].value}' 2>/dev/null); \
		if [ -n "$$GW_HOST" ]; then break; fi; \
		echo "Waiting for Gateway address... ($$i/30)"; \
		sleep 5; \
	done; \
	if [ -z "$$GW_HOST" ]; then \
		echo "Gateway did not receive an address."; \
		oc get gateway mcp-gateway -n gateway-system -o yaml; \
		exit 1; \
	fi; \
	echo "Gateway address: $$GW_HOST"; \
	echo "Sending MCP initialize request through Gateway..."; \
	HTTP_CODE=$$(curl -s -o /tmp/mcp-response.json -w '%{http_code}' --max-time 10 \
		-X POST "http://$$GW_HOST/mcp" \
		-H "Content-Type: application/json" \
		-d '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"e2e-smoke-test","version":"1.0.0"}},"id":1}'); \
	echo "HTTP status: $$HTTP_CODE"; \
	cat /tmp/mcp-response.json 2>/dev/null; echo; \
	if [ "$$HTTP_CODE" != "200" ]; then \
		echo "Expected HTTP 200, got $$HTTP_CODE. Dumping debug info..."; \
		echo "--- Gateway status ---"; \
		oc get gateway mcp-gateway -n gateway-system -o yaml; \
		echo "--- HTTPRoute status ---"; \
		oc get httproute kubernetes-mcp-server -n $(E2E_NAMESPACE) -o yaml; \
		echo "--- MCPServerRegistration ---"; \
		oc get mcpserverregistration kubernetes-mcp-server -n $(E2E_NAMESPACE) -o yaml; \
		echo "--- Broker logs ---"; \
		oc logs -n mcp-system -l app=mcp-broker-router --tail=50; \
		exit 1; \
	fi; \
	if ! grep -q '"result"' /tmp/mcp-response.json; then \
		echo "Response missing expected 'result' field."; \
		cat /tmp/mcp-response.json; \
		exit 1; \
	fi; \
	echo "Smoke test passed. MCP server responded through the MCP Gateway."

.PHONY: e2e-setup
e2e-setup: e2e-install-operator e2e-install-gateway e2e-deploy-mcp-server e2e-register-mcp-server e2e-wait-ready ## Install all components and wait for readiness

.PHONY: e2e-test
e2e-test: e2e-setup e2e-smoke-test ## Run the full e2e test (setup + smoke tests)
