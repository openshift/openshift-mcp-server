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
	go test -tags e2e -v -count=1 -timeout 20m ./test/e2e/ -run TestSmoke $(E2E_ARGS)
