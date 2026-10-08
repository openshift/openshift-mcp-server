##@ Upstream Sync
# Sync policy: fail closed on committed merge conflicts and report changed-path
# overlaps for review. Open downstream PRs are rebased after sync lands.

UPSTREAM_REPO ?= containers/kubernetes-mcp-server
UPSTREAM_REMOTE ?= upstream
ORIGIN_REMOTE ?= origin
SYNC_BRANCH_NAME ?= upstream-sync
OWNER_REPO ?= $(shell git remote get-url $(ORIGIN_REMOTE) | sed 's|https://[^@]*@github\.com/||; s|https://github\.com/||; s|git@github\.com:||; s|\.git$$||')

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

.PHONY: sync-upstream-pr sync-upstream-pr-run
sync-upstream-pr: ## Create/update PR to sync with upstream (requires gh CLI)
	@set -eu; \
	if [ -n "$$(git status --porcelain)" ]; then \
		echo "❌ Refusing to sync from a dirty worktree."; \
		git status --short; \
		exit 1; \
	fi; \
	if ! git remote get-url "$(UPSTREAM_REMOTE)" >/dev/null 2>&1; then \
		git remote add "$(UPSTREAM_REMOTE)" "https://github.com/$(UPSTREAM_REPO).git"; \
	fi; \
	UPSTREAM_URL=$$(git remote get-url "$(UPSTREAM_REMOTE)"); \
	case "$$UPSTREAM_URL" in \
		*github.com[:/]$(UPSTREAM_REPO)|*github.com[:/]$(UPSTREAM_REPO).git) ;; \
		*) echo "❌ Remote $(UPSTREAM_REMOTE) does not point to the configured upstream repository."; exit 1 ;; \
	esac; \
	git fetch $(UPSTREAM_REMOTE) || { echo "❌ Failed to fetch $(UPSTREAM_REMOTE)."; exit 1; }; \
	git fetch $(ORIGIN_REMOTE) || { echo "❌ Failed to fetch $(ORIGIN_REMOTE)."; exit 1; }; \
	UPSTREAM_SHA=$$(git rev-parse "$(UPSTREAM_REMOTE)/main"); \
	DOWNSTREAM_SHA=$$(git rev-parse "$(ORIGIN_REMOTE)/main"); \
	BEHIND_COUNT=$$(git rev-list --count "$$DOWNSTREAM_SHA..$$UPSTREAM_SHA"); \
	if [ "$$BEHIND_COUNT" -eq 0 ]; then \
		echo "✅ $(ORIGIN_REMOTE)/main is up to date. No PR needed."; \
		exit 0; \
	fi; \
	command -v gh >/dev/null 2>&1 || { echo "❌ Error: gh CLI is required. Install from https://cli.github.com/"; exit 1; }; \
	BASE_SHA=$$(git merge-base "$$DOWNSTREAM_SHA" "$$UPSTREAM_SHA") || { echo "❌ Upstream and downstream histories have no merge base."; exit 1; }; \
	TMP_DIR=$$(mktemp -d); \
	trap 'rm -rf "$$TMP_DIR"' EXIT; \
	trap 'exit 1' HUP INT TERM; \
	git diff --name-only "$$BASE_SHA" "$$UPSTREAM_SHA" > "$$TMP_DIR/upstream.paths.unsorted"; \
	git diff --name-only "$$BASE_SHA" "$$DOWNSTREAM_SHA" > "$$TMP_DIR/downstream.paths.unsorted"; \
	LC_ALL=C sort -u "$$TMP_DIR/upstream.paths.unsorted" > "$$TMP_DIR/upstream.paths"; \
	LC_ALL=C sort -u "$$TMP_DIR/downstream.paths.unsorted" > "$$TMP_DIR/downstream.paths"; \
	comm -12 "$$TMP_DIR/upstream.paths" "$$TMP_DIR/downstream.paths" > "$$TMP_DIR/committed-overlaps"; \
	if [ -s "$$TMP_DIR/committed-overlaps" ]; then \
		echo "⚠️  Paths changed on both sides since merge-base; review the sync PR carefully:"; \
		while IFS= read -r path; do printf '  %s\n' "$$path"; done < "$$TMP_DIR/committed-overlaps"; \
	fi; \
	if ! git merge-tree --write-tree "$$DOWNSTREAM_SHA" "$$UPSTREAM_SHA" > "$$TMP_DIR/merge-tree.out" 2>&1; then \
		echo "❌ Upstream/downstream merge preflight found conflicts; no sync branch was created:"; \
		while IFS= read -r line; do printf '  %s\n' "$$line"; done < "$$TMP_DIR/merge-tree.out"; \
		exit 1; \
	fi; \
	SYNC_REMOTE_REFS=$$(git ls-remote --refs "$(ORIGIN_REMOTE)" "refs/heads/$(SYNC_BRANCH_NAME)") || { echo "❌ Could not inspect the remote sync branch."; exit 1; }; \
	SYNC_REMOTE_SHA=$$(printf '%s\n' "$$SYNC_REMOTE_REFS" | cut -f1); \
	ORIGINAL_BRANCH=$$(git symbolic-ref --short HEAD 2>/dev/null || true); \
	ORIGINAL_SHA=$$(git rev-parse HEAD); \
	if [ "$$ORIGINAL_BRANCH" = "$(SYNC_BRANCH_NAME)" ]; then echo "❌ Run sync from a branch other than $(SYNC_BRANCH_NAME)."; exit 1; fi; \
	if $(MAKE) sync-upstream-pr-run UPSTREAM_SHA="$$UPSTREAM_SHA" DOWNSTREAM_SHA="$$DOWNSTREAM_SHA" SYNC_REMOTE_SHA="$$SYNC_REMOTE_SHA"; then \
		SYNC_STATUS=0; \
	else \
		SYNC_STATUS=$$?; \
	fi; \
	CURRENT_BRANCH=$$(git rev-parse --abbrev-ref HEAD 2>/dev/null || true); \
	if [ "$$CURRENT_BRANCH" = "$(SYNC_BRANCH_NAME)" ]; then \
		if [ -n "$$ORIGINAL_BRANCH" ]; then git checkout "$$ORIGINAL_BRANCH"; else git checkout --detach "$$ORIGINAL_SHA"; fi; \
	fi; \
	exit "$$SYNC_STATUS"

sync-upstream-pr-run: ## Internal: execute a preflighted sync using pinned commit SHAs
	@test -z "$$(git status --porcelain)" || { echo "❌ Refusing to reset a sync branch from a dirty worktree."; exit 1; }
	@test -n "$(UPSTREAM_SHA)" -a -n "$(DOWNSTREAM_SHA)" || { echo "❌ Sync SHAs must be provided by sync-upstream-pr."; exit 1; }
	@if git show-ref --verify --quiet "refs/heads/$(SYNC_BRANCH_NAME)"; then \
		LOCAL_SYNC_SHA=$$(git rev-parse "refs/heads/$(SYNC_BRANCH_NAME)"); \
		if [ "$$LOCAL_SYNC_SHA" != "$(SYNC_REMOTE_SHA)" ]; then \
			echo "❌ Local $(SYNC_BRANCH_NAME) differs from its inspected remote tip; refusing to reset it."; \
			exit 1; \
		fi; \
	fi
	@echo "📝 Creating sync branch from pinned downstream SHA..."
	@git checkout -B $(SYNC_BRANCH_NAME) "$(DOWNSTREAM_SHA)"
	@echo "🔀 Merging pinned upstream SHA..."
	@if ! git merge --no-ff "$(UPSTREAM_SHA)" -m "chore: merge upstream changes"; then \
		echo "❌ Merge failed after preflight; aborting without resolving conflicts automatically."; \
		git merge --abort; \
		exit 1; \
	fi
	@echo "🔧 Updating dependencies..."
	@go mod tidy && go mod vendor
	@if [ -n "$$(git status --porcelain go.mod go.sum vendor/)" ]; then \
		echo "📦 Changes detected in generated files. Committing..."; \
		git add go.mod go.sum vendor/; \
		git commit -m "chore: update dependencies and vendor"; \
	fi
	@echo "Updating toolset documentation..."
	@$(MAKE) update-readme-tools
	@if [ -n "$$(git status --porcelain README.md docs/configuration.md)" ]; then \
		echo "Changes detected in toolset documentation. Committing..."; \
		git add README.md docs/configuration.md; \
		git commit -m "chore: update toolset documentation"; \
	fi
	@echo "Updating test snapshots..."
	@$(MAKE) test-update-snapshots
	@if [ -n "$$(git status --porcelain pkg/mcp/testdata/)" ]; then \
		echo "Changes detected in test snapshots. Committing..."; \
		git add pkg/mcp/testdata; \
		git commit -m "chore: update test snapshots"; \
	fi
	@echo "📤 Pushing sync branch with lease protection..."
	@set -eu; \
	PUSHED_SHA=$$(git rev-parse HEAD); \
	git merge-base --is-ancestor "$(UPSTREAM_SHA)" "$$PUSHED_SHA"; \
	git merge-base --is-ancestor "$(DOWNSTREAM_SHA)" "$$PUSHED_SHA"; \
	git push --force-with-lease="refs/heads/$(SYNC_BRANCH_NAME):$(SYNC_REMOTE_SHA)" $(ORIGIN_REMOTE) "HEAD:refs/heads/$(SYNC_BRANCH_NAME)"; \
	echo "⏳ Waiting for GitHub to index pushed branch at $$PUSHED_SHA..."; \
	INDEXED=""; \
	for i in $$(seq 1 30); do \
		INDEXED=$$(git ls-remote --refs $(ORIGIN_REMOTE) "refs/heads/$(SYNC_BRANCH_NAME)" 2>/dev/null | awk '{print $$1}'); \
		if [ "$$INDEXED" = "$$PUSHED_SHA" ]; then \
			echo "✅ Branch indexed at $$PUSHED_SHA."; \
			break; \
		fi; \
		echo "  (attempt $$i/30: got='$$INDEXED' want='$$PUSHED_SHA', retrying in 5s...)"; \
		sleep 5; \
	done; \
	if [ "$$INDEXED" != "$$PUSHED_SHA" ]; then \
		echo "❌ Timed out waiting for GitHub to index branch after $$((30 * 5))s. Exiting."; \
		exit 1; \
	fi
	@echo "🚀 Creating or updating PR..."
	@echo "  OWNER_REPO='$(OWNER_REPO)'"; \
	CHANGELOG=$$(git log --pretty=format:"- %h %s (%an)" "$(DOWNSTREAM_SHA)..$(UPSTREAM_SHA)"); \
	BEHIND_COUNT=$$(git rev-list --count "$(DOWNSTREAM_SHA)..$(UPSTREAM_SHA)"); \
	if ! PR_NUMBER=$$(gh pr list --repo "$(OWNER_REPO)" --head $(SYNC_BRANCH_NAME) --state open --json number --jq '.[0].number'); then \
		echo "❌ Failed to list PRs (possible rate-limit or auth error). Aborting to prevent duplicate PR creation."; \
		exit 1; \
	fi; \
	if [ -n "$$PR_NUMBER" ] && [ "$$PR_NUMBER" != "null" ]; then \
		echo "  Updating existing PR #$$PR_NUMBER..."; \
		PR_BODY="### 🔄 Upstream Sync"$$'\n'$$'\n'"**Update:** $$(date)"$$'\n'$$'\n'"New changes detected from upstream:"$$'\n'"$$CHANGELOG"; \
		gh api --method PATCH "repos/$(OWNER_REPO)/pulls/$$PR_NUMBER" \
			--raw-field body="$$PR_BODY"; \
	else \
		echo "  Creating new PR..."; \
		gh pr create \
			--repo "$(OWNER_REPO)" \
			--title "NO-JIRA: sync with upstream $$(date +'%Y-%m-%d')" \
			--body "### 🔄 Upstream Sync"$$'\n'$$'\n'"This PR syncs the fork with the latest upstream changes."$$'\n'$$'\n'"**Changes:**"$$'\n'"$$CHANGELOG" \
			--base main \
			--head $(SYNC_BRANCH_NAME); \
	fi
	@CURRENT_BRANCH=$$(git rev-parse --abbrev-ref HEAD 2>/dev/null); \
	if [ -n "$$CURRENT_BRANCH" ] && [ "$$CURRENT_BRANCH" = "$(SYNC_BRANCH_NAME)" ]; then \
		git checkout - 2>/dev/null || git checkout main 2>/dev/null || true; \
	fi
	@echo "✅ Done!"
