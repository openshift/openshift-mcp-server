##@ Upstream Sync

UPSTREAM_REPO ?= containers/kubernetes-mcp-server
UPSTREAM_REMOTE ?= upstream
ORIGIN_REMOTE ?= origin
SYNC_BRANCH_NAME ?= upstream-sync

.PHONY: sync-upstream-check
sync-upstream-check: ## Check if fork is behind upstream (dry-run)
	@echo "🔍 Checking sync status with upstream..."
	@git remote add $(UPSTREAM_REMOTE) "https://github.com/$(UPSTREAM_REPO).git" 2>/dev/null || true
	@git fetch $(UPSTREAM_REMOTE)
	@git fetch $(ORIGIN_REMOTE)
	@BEHIND_COUNT=$$(git rev-list --count $(ORIGIN_REMOTE)/main..$(UPSTREAM_REMOTE)/main); \
	if [ "$$BEHIND_COUNT" -eq "0" ]; then \
		echo "✅ $(ORIGIN_REMOTE)/main is up to date with $(UPSTREAM_REMOTE)/main."; \
	else \
		echo "⚠️  $(ORIGIN_REMOTE)/main is behind $(UPSTREAM_REMOTE)/main by $$BEHIND_COUNT commits"; \
		echo ""; \
		echo "Changelog:"; \
		git log --pretty=format:"  - %h %s (%an)" $(ORIGIN_REMOTE)/main..$(UPSTREAM_REMOTE)/main; \
		echo ""; \
		echo ""; \
		echo "Run 'make sync-upstream-pr' to create a PR"; \
	fi

.PHONY: sync-upstream-pr
sync-upstream-pr: ## Create/update PR to sync with upstream (requires gh CLI)
	@echo "🔍 Checking sync status with upstream..."
	@command -v gh >/dev/null 2>&1 || { echo "❌ Error: gh CLI is required. Install from https://cli.github.com/"; exit 1; }
	@git remote add $(UPSTREAM_REMOTE) "https://github.com/$(UPSTREAM_REPO).git" 2>/dev/null || true
	@git fetch $(UPSTREAM_REMOTE)
	@git fetch $(ORIGIN_REMOTE)
	@BEHIND_COUNT=$$(git rev-list --count $(ORIGIN_REMOTE)/main..$(UPSTREAM_REMOTE)/main); \
	if [ "$$BEHIND_COUNT" -eq "0" ]; then \
		echo "✅ $(ORIGIN_REMOTE)/main is up to date. No PR needed."; \
		exit 0; \
	fi
	@echo "📝 Creating sync branch..."
	@CURRENT_BRANCH=$$(git rev-parse --abbrev-ref HEAD); \
	BEHIND_COUNT=$$(git rev-list --count $(ORIGIN_REMOTE)/main..$(UPSTREAM_REMOTE)/main); \
	echo "  Behind by $$BEHIND_COUNT commits"; \
	git checkout -B $(SYNC_BRANCH_NAME) $(UPSTREAM_REMOTE)/main
	@echo "📤 Pushing sync branch..."
	@git push -f $(ORIGIN_REMOTE) $(SYNC_BRANCH_NAME)
	@echo "🚀 Creating or updating PR..."
	@CHANGELOG=$$(git log --pretty=format:"- %h %s (%an)" $(ORIGIN_REMOTE)/main..$(UPSTREAM_REMOTE)/main); \
	BEHIND_COUNT=$$(git rev-list --count $(ORIGIN_REMOTE)/main..$(UPSTREAM_REMOTE)/main); \
	if gh pr list --head $(SYNC_BRANCH_NAME) --state open | grep -q "$(SYNC_BRANCH_NAME)"; then \
		echo "  Updating existing PR..."; \
		gh pr edit $(SYNC_BRANCH_NAME) --body "### 🔄 Upstream Sync"$$'\n'$$'\n'"**Update:** $$(date)"$$'\n'$$'\n'"New changes detected from upstream:"$$'\n'"$$CHANGELOG"; \
	else \
		echo "  Creating new PR..."; \
		gh pr create \
			--title "chore: sync with upstream $$(date +'%Y-%m-%d') - $$BEHIND_COUNT new commits" \
			--body "### 🔄 Upstream Sync"$$'\n'$$'\n'"This PR syncs the fork with the latest upstream changes."$$'\n'$$'\n'"**Changes:**"$$'\n'"$$CHANGELOG" \
			--base main \
			--head $(SYNC_BRANCH_NAME); \
	fi
	@CURRENT_BRANCH=$$(git rev-parse --abbrev-ref HEAD 2>/dev/null); \
	if [ -n "$$CURRENT_BRANCH" ] && [ "$$CURRENT_BRANCH" = "$(SYNC_BRANCH_NAME)" ]; then \
		git checkout - 2>/dev/null || git checkout main 2>/dev/null || true; \
	fi
	@echo "✅ Done!"
