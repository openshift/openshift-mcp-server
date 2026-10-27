##@ Downstream PatternFly Apps

.PHONY: patternfly-assets patternfly-assets-check
# Check the committed assets before the shared target builds and tests them.
# Browser tests must not regenerate stale bundles and silently hide drift.
browser-test: patternfly-assets-check

patternfly-assets: ## Bundle the downstream PatternFly application assets
	npm ci --prefix pkg/mcpapps/openshift/frontend
	npm run build --prefix pkg/mcpapps/openshift/frontend

patternfly-assets-check: ## Check that the committed PatternFly bundles match their sources
	npm ci --prefix pkg/mcpapps/openshift/frontend
	npm run check --prefix pkg/mcpapps/openshift/frontend

