##@ Observability toolsets (OpenShift)

define observability_e2e_with_token
	@if [ -z "$${PROMETHEUS_K8S_TOKEN}" ]; then \
		echo "exporting PROMETHEUS_K8S_TOKEN via oc/kubectl create token prometheus-k8s -n openshift-monitoring"; \
		export PROMETHEUS_K8S_TOKEN="$$(oc create token prometheus-k8s -n openshift-monitoring --duration=45m 2>/dev/null \
			|| kubectl create token prometheus-k8s -n openshift-monitoring --duration=45m)"; \
	fi; \
	PROMETHEUS_K8S_TOKEN="$${PROMETHEUS_K8S_TOKEN}" $(1)
endef

.PHONY: test-observability-e2e-routes
test-observability-e2e-routes: ## OpenShift-only Route discovery + CMO metric
	$(call observability_e2e_with_token,go test -tags observability_e2e -count=1 -timeout=10m -v \
		-run 'TestRouteDiscovery|TestMetricsE2E' ./pkg/observability/tests/)

.PHONY: setup-observability-e2e
setup-observability-e2e: ## Install Tempo/Loki stacks for TestObs*
	./hack/e2e/observability/setup-openshift.sh tempo loki

.PHONY: clean-observability-e2e
clean-observability-e2e: ## Tear down Tempo/Loki stacks
	./hack/e2e/observability/setup-openshift.sh clean
