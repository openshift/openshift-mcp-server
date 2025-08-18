include Makefile

.PHONY: build-ocp
build-ocp: clean format
	$(GO_BUILD_ENV) go build $(COMMON_BUILD_ARGS) $(GOFLAGS) -o $(BINARY_NAME) ./cmd/kubernetes-mcp-server
