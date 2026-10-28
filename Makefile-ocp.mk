include Makefile

.PHONY: build-ocp
build-ocp: clean format
	CGO_ENABLED=1 $(GO_BUILD_ENV) go build $(COMMON_BUILD_ARGS) -tags=strictfipsruntime -mod=vendor -a -o openshift-mcp-server ./cmd/kubernetes-mcp-server

.PHONY: apply-override-snapshot
apply-override-snapshot:
	@test -n "$(SNAPSHOT_FILE)" || { echo "SNAPSHOT_FILE is required; select one explicit snapshot manifest"; exit 2; }
	kubectl apply -f "$(SNAPSHOT_FILE)"

.PHONY: apply-releaseplan
apply-releaseplan:
	@test -n "$(VERSION)" || { echo "VERSION is required (for example, 1.2)"; exit 2; }
	@test -n "$(RELEASE_PLAN)" || { echo "RELEASE_PLAN is required (for example, openshift-mcp-server-releaseplan-120-stage.yaml)"; exit 2; }
	git fetch --quiet origin main
	git show FETCH_HEAD:.konflux_release/$(VERSION)/releasePlan/$(RELEASE_PLAN) | kubectl apply -f -

.PHONY: apply-releases
apply-releases:
	@echo "Release creation is intentionally manual; use kubectl create for one selected Release manifest."
	@exit 2
