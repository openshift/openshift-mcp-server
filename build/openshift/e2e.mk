##@ E2E Tests

E2E_NAMESPACE ?= openshift-mcp-server-e2e
MCP_LIFECYCLE_OPERATOR_VERSION ?= v0.1.0
MCP_LIFECYCLE_OPERATOR_URL ?= https://github.com/kubernetes-sigs/mcp-lifecycle-operator/releases/download/$(MCP_LIFECYCLE_OPERATOR_VERSION)/install.yaml
MCP_SERVER_IMAGE ?= quay.io/redhat-user-workloads/ocp-mcp-server-tenant/openshift-mcp-server-release-03:latest
MCP_GATEWAY_VERSION ?= 0.6.0
MCP_GATEWAY_NAMESPACE ?= mcp-system
GATEWAY_NAMESPACE ?= gateway-system

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
e2e-install-gateway: ## Install the MCP Gateway controller via OLM and deploy instance via Helm
	@echo "Installing MCP Gateway $(MCP_GATEWAY_VERSION) via OLM..."
	oc create ns $(GATEWAY_NAMESPACE) --dry-run=client -o yaml | oc apply -f -
	oc create ns $(MCP_GATEWAY_NAMESPACE) --dry-run=client -o yaml | oc apply -f -
	@echo "Applying prerequisite CRDs..."
	oc apply -f hack/e2e/gateway-prereqs.yaml
	oc apply -f hack/e2e/gateway-olm.yaml
	@echo "Waiting for CatalogSource to be ready..."
	@for i in $$(seq 1 60); do \
		STATE=$$(oc get catalogsource mcp-gateway-catalog -n openshift-marketplace \
			-o jsonpath='{.status.connectionState.lastObservedState}' 2>/dev/null); \
		if [ "$$STATE" = "READY" ]; then echo "CatalogSource is ready."; break; fi; \
		if [ $$i -eq 60 ]; then echo "Timed out waiting for CatalogSource."; exit 1; fi; \
		echo "  Waiting for CatalogSource... ($$i/60) state=$$STATE"; \
		sleep 5; \
	done
	@echo "Waiting for MCP Gateway CSV to succeed..."
	@for i in $$(seq 1 60); do \
		PHASE=$$(oc get csv -n $(MCP_GATEWAY_NAMESPACE) \
			-l operators.coreos.com/mcp-gateway.$(MCP_GATEWAY_NAMESPACE)="" \
			-o jsonpath='{.items[0].status.phase}' 2>/dev/null); \
		if [ "$$PHASE" = "Succeeded" ]; then echo "CSV succeeded."; break; fi; \
		if [ $$i -eq 60 ]; then echo "Timed out waiting for CSV. phase=$$PHASE"; exit 1; fi; \
		echo "  Waiting for CSV... ($$i/60) phase=$$PHASE"; \
		sleep 5; \
	done
	@echo "Waiting for MCP Gateway CRDs to be established..."
	oc wait crd/mcpgatewayextensions.mcp.kuadrant.io --for=condition=Established --timeout=120s
	oc wait crd/mcpserverregistrations.mcp.kuadrant.io --for=condition=Established --timeout=120s
	@echo "Installing MCP Gateway instance via Helm..."
	@MCP_GATEWAY_HOST=mcp.apps.$$(oc get dns cluster -o jsonpath='{.spec.baseDomain}'); \
	echo "MCP Gateway host: $$MCP_GATEWAY_HOST"; \
	helm upgrade -i mcp-gateway oci://ghcr.io/kuadrant/charts/mcp-gateway \
		--version $(MCP_GATEWAY_VERSION) \
		--namespace $(MCP_GATEWAY_NAMESPACE) \
		--skip-crds \
		--set controller.enabled=false \
		--set gateway.create=true \
		--set gateway.name=mcp-gateway \
		--set gateway.namespace=$(GATEWAY_NAMESPACE) \
		--set gateway.publicHost="$$MCP_GATEWAY_HOST" \
		--set gateway.internalHostPattern="*.mcp.local" \
		--set gateway.gatewayClassName=openshift-default \
		--set mcpGatewayExtension.create=true \
		--set mcpGatewayExtension.gatewayRef.name=mcp-gateway \
		--set mcpGatewayExtension.gatewayRef.namespace=$(GATEWAY_NAMESPACE) \
		--set mcpGatewayExtension.gatewayRef.sectionName=mcp
	@echo "Applying ReferenceGrant for e2e namespace..."
	oc apply -f hack/e2e/gateway.yaml
	@echo "Waiting for mcp-gateway-controller deployment..."
	oc wait deployment/mcp-gateway-controller \
		-n $(MCP_GATEWAY_NAMESPACE) \
		--for=condition=Available \
		--timeout=180s
	@echo "Waiting for mcp-gateway broker deployment to be created..."
	@for i in $$(seq 1 60); do \
		oc get deployment/mcp-gateway -n $(MCP_GATEWAY_NAMESPACE) >/dev/null 2>&1 && break; \
		if [ $$i -eq 60 ]; then echo "Broker deployment was never created."; \
			oc get mcpgatewayextension -n $(MCP_GATEWAY_NAMESPACE) -o yaml; \
			oc logs -n $(MCP_GATEWAY_NAMESPACE) deployment/mcp-gateway-controller --tail=50; \
			exit 1; fi; \
		echo "  Waiting for broker deployment... ($$i/60)"; \
		sleep 5; \
	done
	@echo "Waiting for mcp-gateway broker to become available..."
	@oc wait deployment/mcp-gateway \
		-n $(MCP_GATEWAY_NAMESPACE) \
		--for=condition=Available \
		--timeout=180s || { \
		echo "mcp-gateway broker not ready. Dumping debug info..."; \
		echo "--- Pod status ---"; \
		oc get pods -n $(MCP_GATEWAY_NAMESPACE) -o wide; \
		echo "--- Pod describe ---"; \
		oc describe pods -n $(MCP_GATEWAY_NAMESPACE) -l app.kubernetes.io/name=mcp-gateway; \
		echo "--- Pod logs ---"; \
		oc logs -n $(MCP_GATEWAY_NAMESPACE) -l app.kubernetes.io/name=mcp-gateway --tail=100; \
		echo "--- MCPGatewayExtension ---"; \
		oc get mcpgatewayextension -n $(MCP_GATEWAY_NAMESPACE) -o yaml; \
		echo "--- Events ---"; \
		oc get events -n $(MCP_GATEWAY_NAMESPACE) --sort-by='.lastTimestamp' | tail -20; \
		exit 1; \
	}
	@echo "Waiting for Gateway to be programmed..."
	@for i in $$(seq 1 30); do \
		STATUS=$$(oc get gateway mcp-gateway -n $(GATEWAY_NAMESPACE) \
			-o jsonpath='{.status.conditions[?(@.type=="Programmed")].status}' 2>/dev/null); \
		if [ "$$STATUS" = "True" ]; then echo "Gateway is programmed."; break; fi; \
		if [ $$i -eq 30 ]; then echo "Gateway not programmed."; \
			oc get gateway mcp-gateway -n $(GATEWAY_NAMESPACE) -o yaml; exit 1; fi; \
		echo "  Waiting for Gateway... ($$i/30)"; \
		sleep 5; \
	done
	@echo "Creating Route for MCP Gateway..."
	@MCP_GATEWAY_HOST=mcp.apps.$$(oc get dns cluster -o jsonpath='{.spec.baseDomain}'); \
	oc create route edge mcp-gateway \
		--service=mcp-gateway-openshift-default \
		--hostname="$$MCP_GATEWAY_HOST" \
		--port=mcp \
		-n $(GATEWAY_NAMESPACE) 2>/dev/null || true
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
	oc expose svc/kubernetes-mcp-server -n $(E2E_NAMESPACE) || true
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
	@echo "MCP server is running and reachable."

.PHONY: e2e-smoke-test
e2e-smoke-test: ## Smoke test the MCP server through the MCP Gateway
	@echo "Waiting for MCPServerRegistration to be ready (broker polls every 60s)..."
	@for i in $$(seq 1 24); do \
		READY=$$(oc get mcpserverregistration/kubernetes-mcp-server -n $(E2E_NAMESPACE) \
			-o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null); \
		if [ "$$READY" = "True" ]; then echo "MCPServerRegistration is ready."; break; fi; \
		if [ $$i -eq 24 ]; then echo "MCPServerRegistration not ready."; \
			oc get mcpserverregistration -n $(E2E_NAMESPACE) -o yaml; \
			oc logs -n $(MCP_GATEWAY_NAMESPACE) deployment/mcp-gateway --tail=30; exit 1; fi; \
		echo "  Waiting for MCPServerRegistration... ($$i/24)"; \
		sleep 5; \
	done
	@echo "Retrieving Gateway Route host..."
	@ROUTE_HOST=$$(oc get route/mcp-gateway -n $(GATEWAY_NAMESPACE) -o jsonpath='{.spec.host}' 2>/dev/null); \
	if [ -z "$$ROUTE_HOST" ]; then \
		echo "Gateway Route has no host. Dumping debug info..."; \
		oc get routes -n $(GATEWAY_NAMESPACE) -o wide; \
		oc get svc -n $(GATEWAY_NAMESPACE) -o wide; \
		exit 1; \
	fi; \
	echo "Gateway Route host: $$ROUTE_HOST"; \
	echo "Sending MCP initialize request through Gateway..."; \
	HTTP_CODE=$$(curl -s -o /tmp/mcp-response.json -w '%{http_code}' --max-time 10 \
		-X POST "https://$$ROUTE_HOST/mcp" \
		-H "Content-Type: application/json" \
		-k \
		-d '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"e2e-smoke-test","version":"1.0.0"}},"id":1}'); \
	echo "HTTP status: $$HTTP_CODE"; \
	cat /tmp/mcp-response.json 2>/dev/null; echo; \
	if [ "$$HTTP_CODE" != "200" ]; then \
		echo "Expected HTTP 200, got $$HTTP_CODE. Dumping debug info..."; \
		echo "--- Gateway status ---"; \
		oc get gateway mcp-gateway -n $(GATEWAY_NAMESPACE) -o yaml; \
		echo "--- HTTPRoute status ---"; \
		oc get httproute kubernetes-mcp-server -n $(E2E_NAMESPACE) -o yaml; \
		echo "--- MCPServerRegistration ---"; \
		oc get mcpserverregistration kubernetes-mcp-server -n $(E2E_NAMESPACE) -o yaml; \
		echo "--- MCPGatewayExtension ---"; \
		oc get mcpgatewayextension -n $(MCP_GATEWAY_NAMESPACE) -o yaml; \
		echo "--- Broker logs ---"; \
		oc logs -n $(MCP_GATEWAY_NAMESPACE) -l app.kubernetes.io/name=mcp-gateway --tail=50; \
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
