# Keycloak ACM Integration for OpenShift
#
# This file contains targets for setting up Keycloak with V1 token exchange
# for ACM multi-cluster environments on OpenShift.
#
# Prerequisites:
#   - OpenShift 4.19+ or 4.20+ cluster
#   - ACM installed
#   - Cluster-admin access
#
# Initial Setup (Hub Only):
#   make keycloak-acm-setup-hub         # Deploy Keycloak and configure hub realm
#   make keycloak-acm-generate-toml     # Generate MCP server configuration
#
# Environment Variables:
#   HUB_KUBECONFIG    - Path to hub cluster kubeconfig (default: $KUBECONFIG)
#   KEYCLOAK_URL      - Keycloak URL (auto-detected from route if not set)
#   ADMIN_USER        - Keycloak admin username (default: admin)
#   ADMIN_PASSWORD    - Keycloak admin password (default: admin)

##@ Keycloak ACM Integration

.PHONY: keycloak-acm-setup-hub
keycloak-acm-setup-hub: ## Deploy Keycloak on OpenShift with V1 token exchange for ACM hub
	@echo "==========================================="
	@echo "Keycloak ACM Hub Setup"
	@echo "==========================================="
	@echo ""
	@echo "This will:"
	@echo "  1. Enable TechPreviewNoUpgrade feature gate (if needed)"
	@echo "  2. Deploy Keycloak with V1 token exchange features"
	@echo "  3. Create hub realm with mcp user and clients"
	@echo "  4. Configure same-realm token exchange"
	@echo "  5. Fix CA trust for cross-realm token exchange"
	@echo "  6. Create RBAC for mcp user"
	@echo "  7. Save configuration to .keycloak-config/"
	@echo ""
	@bash ./hack/keycloak-acm/setup-hub.sh
	@echo ""
	@echo "✅ Hub Keycloak setup complete!"
	@echo ""
	@echo "Configuration saved to: .keycloak-config/hub-config.env"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Run: make keycloak-acm-generate-toml"
	@echo "  2. Start MCP server with: ./kubernetes-mcp-server --config acm-kubeconfig.toml"

.PHONY: keycloak-acm-generate-toml
keycloak-acm-generate-toml: ## Generate _output/acm-kubeconfig.toml from saved Keycloak configuration
	@echo "==========================================="
	@echo "Generating MCP Server Configuration"
	@echo "==========================================="
	@echo ""
	@bash ./hack/keycloak-acm/generate-toml.sh
	@echo ""
	@echo "Next: Start MCP server with: ./kubernetes-mcp-server --port 8080 --config _output/acm-kubeconfig.toml"

.PHONY: keycloak-acm-register-managed-cluster
keycloak-acm-register-managed-cluster: ## Register managed cluster with ACM and configure OIDC (requires: CLUSTER_NAME, MANAGED_KUBECONFIG)
	@if [ -z "$(CLUSTER_NAME)" ]; then \
		echo "Error: CLUSTER_NAME is required"; \
		echo "Usage: make keycloak-acm-register-managed-cluster CLUSTER_NAME=my-cluster MANAGED_KUBECONFIG=/path/to/kubeconfig"; \
		exit 1; \
	fi
	@if [ -z "$(MANAGED_KUBECONFIG)" ]; then \
		echo "Error: MANAGED_KUBECONFIG is required"; \
		echo "Usage: make keycloak-acm-register-managed-cluster CLUSTER_NAME=my-cluster MANAGED_KUBECONFIG=/path/to/kubeconfig"; \
		exit 1; \
	fi
	@if [ ! -f "$(MANAGED_KUBECONFIG)" ]; then \
		echo "Error: Kubeconfig file not found: $(MANAGED_KUBECONFIG)"; \
		exit 1; \
	fi
	@echo "==========================================="
	@echo "Managed Cluster Setup: $(CLUSTER_NAME)"
	@echo "==========================================="
	@echo ""
	@echo "This will:"
	@echo "  1. Create ACM ManagedCluster resource"
	@echo "  2. Apply ACM import manifests (starts cluster-proxy agents)"
	@echo "  3. Create managed cluster realm in Keycloak"
	@echo "  4. Configure cross-realm token exchange"
	@echo "  5. Enable TechPreviewNoUpgrade on managed cluster"
	@echo "  6. Configure OIDC authentication"
	@echo "  7. Create RBAC for service-account-mcp-server"
	@echo ""
	@echo "⏳ Total time: ~25-30 minutes (rollouts happen in background)"
	@echo ""
	@HUB_KUBECONFIG="$${HUB_KUBECONFIG:-$$KUBECONFIG}" && \
	if [ -z "$$HUB_KUBECONFIG" ]; then \
		echo "Error: HUB_KUBECONFIG not set and KUBECONFIG is empty"; \
		echo "Either set KUBECONFIG to hub cluster or pass HUB_KUBECONFIG=..."; \
		exit 1; \
	fi && \
	CLUSTER_NAME="$(CLUSTER_NAME)" \
	HUB_KUBECONFIG="$$HUB_KUBECONFIG" \
	MANAGED_KUBECONFIG="$(MANAGED_KUBECONFIG)" \
	bash ./hack/keycloak-acm/register-managed-cluster.sh

.PHONY: keycloak-acm-apply-import-manifests
keycloak-acm-apply-import-manifests: ## [Optional] Re-apply ACM import manifests to managed cluster (already included in registration)
	@if [ -z "$(CLUSTER_NAME)" ]; then \
		echo "Error: CLUSTER_NAME is required"; \
		echo "Usage: make keycloak-acm-apply-import-manifests CLUSTER_NAME=my-cluster MANAGED_KUBECONFIG=/path/to/kubeconfig"; \
		exit 1; \
	fi
	@if [ -z "$(MANAGED_KUBECONFIG)" ]; then \
		echo "Error: MANAGED_KUBECONFIG is required"; \
		echo "Usage: make keycloak-acm-apply-import-manifests CLUSTER_NAME=my-cluster MANAGED_KUBECONFIG=/path/to/kubeconfig"; \
		exit 1; \
	fi
	@if [ ! -f "$(MANAGED_KUBECONFIG)" ]; then \
		echo "Error: Kubeconfig file not found: $(MANAGED_KUBECONFIG)"; \
		exit 1; \
	fi
	@HUB_KUBECONFIG="$${HUB_KUBECONFIG:-$$KUBECONFIG}" && \
	if [ -z "$$HUB_KUBECONFIG" ]; then \
		echo "Error: HUB_KUBECONFIG not set and KUBECONFIG is empty"; \
		echo "Either set KUBECONFIG to hub cluster or pass HUB_KUBECONFIG=..."; \
		exit 1; \
	fi && \
	CLUSTER_NAME="$(CLUSTER_NAME)" \
	HUB_KUBECONFIG="$$HUB_KUBECONFIG" \
	MANAGED_KUBECONFIG="$(MANAGED_KUBECONFIG)" \
	bash ./hack/keycloak-acm/apply-import-manifests.sh

.PHONY: keycloak-acm-status
keycloak-acm-status: ## Show Keycloak ACM configuration status
	@echo "==========================================="
	@echo "Keycloak ACM Configuration Status"
	@echo "==========================================="
	@echo ""
	@if [ -f .keycloak-config/hub-config.env ]; then \
		echo "✅ Hub configuration found:"; \
		echo ""; \
		source .keycloak-config/hub-config.env && \
		echo "  Keycloak URL: $$KEYCLOAK_URL"; \
		echo "  Hub Realm:    $$HUB_REALM"; \
		echo "  MCP User:     $$MCP_USERNAME"; \
		echo ""; \
		kubectl get pods -n keycloak -l app=keycloak 2>/dev/null && echo "" || echo "  ⚠️  Keycloak pod not found"; \
		kubectl get route keycloak -n keycloak -o jsonpath='{.spec.host}' 2>/dev/null && echo "" || echo "  ⚠️  Keycloak route not found"; \
	else \
		echo "❌ Hub configuration not found"; \
		echo "   Run: make keycloak-acm-setup-hub"; \
	fi
	@echo ""
	@if [ -f acm-kubeconfig.toml ]; then \
		echo "✅ MCP configuration found: acm-kubeconfig.toml"; \
		echo ""; \
		echo "Configured clusters:"; \
		grep '^\[cluster_provider_configs.acm-kubeconfig.clusters' acm-kubeconfig.toml | \
			sed 's/\[cluster_provider_configs.acm-kubeconfig.clusters."\(.*\)"\]/  - \1/'; \
	else \
		echo "❌ MCP configuration not found"; \
		echo "   Run: make keycloak-acm-generate-toml"; \
	fi
	@echo ""
