##@ Korrel8r Evals

KORREL8R_NS := openshift-cluster-observability-operator
KORREL8R_ROUTE := korrel8r-mcp-eval
KORREL8R_CONFIG_DIR := _output/korrel8r-eval-configs
KORREL8R_OC = $(if $(MCP_EVAL_KUBECONFIG),KUBECONFIG="$(MCP_EVAL_KUBECONFIG)" )oc -n $(KORREL8R_NS)

.PHONY: expose-korrel8r
expose-korrel8r: ## Create a temporary HTTPS Route and config for local Korrel8r evals
	@mkdir -p _output
	@test ! -e _output/korrel8r-eval-route.uid || { echo "Korrel8r eval Route already tracked; run make stop-korrel8r first" >&2; exit 1; }
	@$(KORREL8R_OC) create route reencrypt $(KORREL8R_ROUTE) --service=korrel8r --port=web -o jsonpath='{.metadata.uid}' > _output/korrel8r-eval-route.uid \
		|| { rm -f _output/korrel8r-eval-route.uid; exit 1; }
	@set -e; trap '$(MAKE) stop-korrel8r' EXIT; \
	$(KORREL8R_OC) wait --for=jsonpath='{.status.ingress[0].host}' route/$(KORREL8R_ROUTE) --timeout=60s; \
	HOST=$$($(KORREL8R_OC) get route/$(KORREL8R_ROUTE) -o jsonpath='{.status.ingress[0].host}'); \
	URL="https://$$HOST"; \
	for i in $$(seq 1 30); do \
		STATUS=$$(curl -q -sS --max-time 3 -o /dev/null -w '%{http_code}' "$$URL/api/v1alpha1/domains" 2>/dev/null) || STATUS=""; \
		case "$$STATUS" in 200|401|403) break ;; esac; \
		[ $$i -lt 30 ] || { echo "Korrel8r Route unreachable at $$URL (check ingress TLS/CA)" >&2; exit 1; }; \
		sleep 1; \
	done; \
	mkdir -p $(KORREL8R_CONFIG_DIR); \
	cp dev/config/mcp-configs/*.toml $(KORREL8R_CONFIG_DIR)/; \
	printf '[toolset_configs."observability/korrel8r"]\nurl = "%s"\n' "$$URL" > $(KORREL8R_CONFIG_DIR)/zy-korrel8r-route.toml; \
	trap - EXIT; echo "Korrel8r eval Route ready at $$URL"

.PHONY: run-korrel8r-evals
run-korrel8r-evals: build ## Run Korrel8r evals using a temporary HTTPS Route
	@set -e; \
	trap '$(MAKE) stop-server stop-korrel8r' EXIT; \
	$(MAKE) expose-korrel8r; \
	$(MAKE) run-server TOOLSETS=core,observability/korrel8r MCP_CONFIG_DIR=$(KORREL8R_CONFIG_DIR); \
	$(MAKE) run-evals SUITE=korrel8r

.PHONY: stop-korrel8r
stop-korrel8r: ## Delete the temporary Korrel8r eval Route
	@if [ -f _output/korrel8r-eval-route.uid ]; then \
		CREATED=$$(cat _output/korrel8r-eval-route.uid); \
		CURRENT=$$($(KORREL8R_OC) get route/$(KORREL8R_ROUTE) -o jsonpath='{.metadata.uid}' 2>/dev/null) || true; \
		if [ -n "$$CREATED" ] && [ "$$CREATED" = "$$CURRENT" ]; then \
			$(KORREL8R_OC) delete route/$(KORREL8R_ROUTE) || exit 1; \
		fi; \
		rm -f _output/korrel8r-eval-route.uid; \
	fi
	@rm -rf $(KORREL8R_CONFIG_DIR)
