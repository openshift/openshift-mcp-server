##@ OpenShift CI E2E Tests

.PHONY: e2e-ci-setup
e2e-ci-setup: helm kubectl ## Download tools needed for e2e tests (cluster is provided by ci-operator)

.PHONY: e2e-ci-test
e2e-ci-test: ## Run e2e tests against a CI-provisioned cluster
	KUBECONFIG=$(KUBECONFIG) \
	KUBECTL_PATH=$(KUBECTL) \
	HELM_PATH=$(HELM) \
	MCP_SERVER_IMAGE=$(MCP_SERVER_IMAGE) \
	CHART_PATH=$(shell pwd)/charts/kubernetes-mcp-server \
	go test -tags e2e -v -count=1 -timeout 20m ./test/e2e/ -skip '^TestObs[A-Z]' $(E2E_ARGS)

# Same auth_mode as Kind (header default). PROMETHEUS_K8S_TOKEN is forwarded by
# TestObs MCP clients. OBS_* URLs require the upstream e2e helper env support.
.PHONY: e2e-ci-test-obs
e2e-ci-test-obs: ## Run TestObs* against OpenShift monitoring + installed Tempo/Loki
	@if [ -z "$${PROMETHEUS_K8S_TOKEN}" ]; then \
		echo "exporting PROMETHEUS_K8S_TOKEN via oc/kubectl create token prometheus-k8s -n openshift-monitoring"; \
		export PROMETHEUS_K8S_TOKEN="$$(oc create token prometheus-k8s -n openshift-monitoring --duration=45m 2>/dev/null \
			|| kubectl create token prometheus-k8s -n openshift-monitoring --duration=45m)"; \
	fi; \
	KUBECONFIG=$(KUBECONFIG) \
	KUBECTL_PATH=$(KUBECTL) \
	HELM_PATH=$(HELM) \
	MCP_SERVER_IMAGE=$(MCP_SERVER_IMAGE) \
	CHART_PATH=$(shell pwd)/charts/kubernetes-mcp-server \
	OBS_PROMETHEUS_URL="$${OBS_PROMETHEUS_URL:-https://prometheus-k8s.openshift-monitoring.svc:9091}" \
	OBS_ALERTMANAGER_URL="$${OBS_ALERTMANAGER_URL:-https://alertmanager-main.openshift-monitoring.svc:9094}" \
	PROMETHEUS_K8S_TOKEN="$${PROMETHEUS_K8S_TOKEN}" \
	go test -tags e2e -v -count=1 -timeout 30m ./test/e2e/ -run '^TestObs[A-Z]' $(E2E_ARGS)
