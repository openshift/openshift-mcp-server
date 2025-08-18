include Makefile

.PHONY: build-ocp
build-ocp: clean format
	CGO_ENABLED=1 $(GO_BUILD_ENV) go build $(COMMON_BUILD_ARGS) -tags=strictfipsruntime -mod=vendor -a -o openshift-mcp-server ./cmd/kubernetes-mcp-server
