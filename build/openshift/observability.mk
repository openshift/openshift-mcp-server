##@ Observability toolsets (OpenShift)

# Export a short-lived prometheus-k8s SA token when unset. CI kubeconfigs often
# use client certs that the monitoring OAuth proxy rejects; Routes need a bearer.
define observability_e2e_with_token
	@if [ -z "$${PROMETHEUS_K8S_TOKEN}" ]; then \
		echo "exporting PROMETHEUS_K8S_TOKEN via oc/kubectl create token prometheus-k8s -n openshift-monitoring"; \
		export PROMETHEUS_K8S_TOKEN="$$(oc create token prometheus-k8s -n openshift-monitoring --duration=20m 2>/dev/null \
			|| kubectl create token prometheus-k8s -n openshift-monitoring --duration=20m)"; \
	fi; \
	PROMETHEUS_K8S_TOKEN="$${PROMETHEUS_K8S_TOKEN}" $(1)
endef

.PHONY: test-observability-e2e-metrics
test-observability-e2e-metrics: ## Metrics + route discovery only (platform monitoring; no Tempo/Loki setup)
	$(call observability_e2e_with_token,go test -tags observability_e2e -count=1 -timeout=10m -v \
		-run 'TestMetricsE2E|TestRouteDiscovery' ./pkg/observability/tests/)

.PHONY: test-observability-e2e
test-observability-e2e: ## Full observability e2e (routes, metrics, tempo/loki/otelcol)
	$(call observability_e2e_with_token,go test -tags observability_e2e -count=1 -timeout=30m -v ./pkg/observability/tests/)

.PHONY: setup-observability-e2e
setup-observability-e2e: ## Install Tempo/Loki stacks for observability e2e
	./hack/e2e/observability/setup-openshift.sh tempo loki

.PHONY: clean-observability-e2e
clean-observability-e2e: ## Tear down Tempo/Loki stacks for observability e2e
	./hack/e2e/observability/setup-openshift.sh clean
