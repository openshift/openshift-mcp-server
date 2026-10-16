##@ Upstream Sync

UPSTREAM_REPO ?= containers/kubernetes-mcp-server
UPSTREAM_REMOTE ?= upstream
ORIGIN_REMOTE ?= origin
SYNC_BRANCH_NAME ?= upstream-sync
SYNC_BASE_BRANCH ?= main
OWNER_REPO ?=

export UPSTREAM_REPO UPSTREAM_REMOTE ORIGIN_REMOTE SYNC_BRANCH_NAME SYNC_BASE_BRANCH OWNER_REPO

.PHONY: downstream-autogenerate-content
downstream-autogenerate-content: ## Regenerate generated content for downstream syncs
	$(MAKE) tidy
	go mod vendor
	$(MAKE) update-readme-tools
	$(MAKE) test-update-snapshots

.PHONY: sync-upstream-check
sync-upstream-check: ## Inspect upstream sync plan without publishing
	@bash hack/sync-upstream.sh plan

.PHONY: sync-upstream-pr
sync-upstream-pr: ## Create or update the upstream sync PR (requires gh CLI)
	@bash hack/sync-upstream.sh publish
