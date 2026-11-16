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

# Prow supplies the exact PR base SHA; locally, compare against the fetched
# downstream main branch. Override PR_BASE_REF for other target branches.
PR_BASE_REF ?= $(if $(PULL_BASE_SHA),$(PULL_BASE_SHA),$(ORIGIN_REMOTE)/main)
PR_UPSTREAM_REF ?= $(UPSTREAM_REMOTE)/main

.PHONY: check-downstream-files
check-downstream-files: ## Reject downstream PR edits to upstream-owned files
	@set -eu; \
	if ! git remote get-url "$(UPSTREAM_REMOTE)" >/dev/null 2>&1; then \
		git remote add "$(UPSTREAM_REMOTE)" "https://github.com/$(UPSTREAM_REPO).git"; \
	fi; \
	UPSTREAM_URL=$$(git remote get-url "$(UPSTREAM_REMOTE)"); \
	case "$$UPSTREAM_URL" in \
		*github.com[:/]$(UPSTREAM_REPO)|*github.com[:/]$(UPSTREAM_REPO).git) ;; \
		*) echo "Upstream remote $(UPSTREAM_REMOTE) does not point to $(UPSTREAM_REPO)."; exit 1 ;; \
	esac; \
	git fetch "$(UPSTREAM_REMOTE)" || { echo "Failed to fetch $(UPSTREAM_REMOTE)."; exit 1; }; \
	git fetch "$(ORIGIN_REMOTE)" || { echo "Failed to fetch $(ORIGIN_REMOTE)."; exit 1; }; \
	go run ./internal/tools/check-downstream-files -base "$(PR_BASE_REF)" -upstream "$(PR_UPSTREAM_REF)"

.PHONY: sync-upstream-check
sync-upstream-check: ## Inspect upstream sync plan without publishing
	@bash hack/sync-upstream.sh plan

.PHONY: sync-upstream-pr
sync-upstream-pr: ## Create or update the upstream sync PR (requires gh CLI)
	@bash hack/sync-upstream.sh publish

.PHONY: check-upstream-sync-rules
check-upstream-sync-rules: ## Validate upstream-sync rules and fixture expectations
	@bash hack/sync-upstream.sh validate
	@bash hack/test-upstream-sync-rules.sh
