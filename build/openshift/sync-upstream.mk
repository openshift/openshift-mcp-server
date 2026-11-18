##@ Upstream Sync

UPSTREAM_REPO ?= containers/kubernetes-mcp-server
UPSTREAM_REMOTE ?= upstream
ORIGIN_REMOTE ?= origin
SYNC_BRANCH_NAME ?= upstream-sync
SYNC_BASE_BRANCH ?= main
OWNER_REPO ?=
# Downstream dependency pins reapplied after taking upstream's go.mod during
# upstream sync; see the RECONCILE rule for go.mod in hack/upstream-sync.rules
# and keep this file in sync with go.mod when a dependency moves on purpose.
# Relative on purpose: these targets run with the repository root as CWD.
GO_PINS_FILE := build/openshift/go.pins

export UPSTREAM_REPO UPSTREAM_REMOTE ORIGIN_REMOTE SYNC_BRANCH_NAME SYNC_BASE_BRANCH OWNER_REPO

.PHONY: downstream-autogenerate-content
downstream-autogenerate-content: ## Regenerate generated content for downstream syncs
	$(MAKE) tidy
	go mod vendor
	$(MAKE) update-readme-tools
	$(MAKE) test-update-snapshots

.PHONY: reconcile-go-mod
reconcile-go-mod: ## Reapply the downstream go.mod pins onto upstream's go.mod
	while IFS= read -r pin; do \
		case "$$pin" in ''|'#'*) continue ;; esac; \
		go mod edit -require="$$pin"; \
	done < $(GO_PINS_FILE)
	go mod tidy
	go mod vendor
	$(MAKE) verify-go-pins

.PHONY: verify-go-pins
verify-go-pins: ## Fail unless go.mod requires every pinned module at its pinned version
	@fails=0; \
	while IFS= read -r pin; do \
		case "$$pin" in ''|'#'*) continue ;; esac; \
		module=$${pin%@*}; version=$${pin#*@}; \
		if ! awk -v m="$$module" -v v="$$version" '/^require[[:space:]]*\(/ { in_require = 1; next } /^require[[:space:]][^)]/ { if ($$2 == m && $$3 == v) found = 1 } /^\)/ { in_require = 0; next } in_require && $$1 == m && $$2 == v { found = 1 } END { exit !found }' go.mod; then \
			printf 'go.mod does not require %s at %s (pinned in %s)\n' "$$module" "$$version" "$(GO_PINS_FILE)" >&2; \
			fails=1; \
		fi; \
	done < $(GO_PINS_FILE); \
	exit $$fails

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
