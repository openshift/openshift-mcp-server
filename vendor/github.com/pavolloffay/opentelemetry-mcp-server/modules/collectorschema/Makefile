OCB_VERSION ?= 0.138.0
SCHEMA_OUTPUT_DIR ?= ../schemas/$(OCB_VERSION)

# Default target - runs both schema generation and changelog processing
.PHONY: all
all: generate-schemas changelogs

.PHONY: install-ocb
install-ocb:
	@mkdir -p .bin
	GOBIN=$(PWD)/.bin go install go.opentelemetry.io/collector/cmd/builder@v$(OCB_VERSION)

.PHONY: build-collector
build-collector: install-ocb
	./.bin/builder --config manifest-$(OCB_VERSION).yaml --skip-compilation
	@# Add collectorschema dependency (OCB regenerates go.mod, removing our additions)
	@cd build && \
		echo 'require github.com/pavolloffay/opentelemetry-mcp-server/modules/collectorschema v0.0.0' >> go.mod && \
		echo 'replace github.com/pavolloffay/opentelemetry-mcp-server/modules/collectorschema => ../' >> go.mod && \
		go mod tidy

# Schema output directory (can be overridden)
SCHEMA_OUTPUT_DIR ?= ../schemas/$(OCB_VERSION)

.PHONY: generate-schemas
generate-schemas:
	@echo "Generating JSON schemas for all OpenTelemetry collector components..."
#	OCB_VERSION=0.135.0 make build-collector
#	cd build && go mod vendor && SCHEMA_OUTPUT_DIR=../schemas/0.135.0 go test -run TestGenerateAllSchemas -v
#	OCB_VERSION=0.136.0 make build-collector
#	cd build && go mod vendor && SCHEMA_OUTPUT_DIR=../schemas/0.136.0 go test -run TestGenerateAllSchemas -v
#	OCB_VERSION=0.137.0 make build-collector
#	cd build && go mod vendor && SCHEMA_OUTPUT_DIR=../schemas/0.137.0 go test -run TestGenerateAllSchemas -v
#	OCB_VERSION=0.138.0 make build-collector
#	cd build && go mod vendor && SCHEMA_OUTPUT_DIR=../schemas/0.138.0 go test -run TestGenerateAllSchemas -v
#	OCB_VERSION=0.139.0 make build-collector
#	cd build && go mod vendor && SCHEMA_OUTPUT_DIR=../schemas/0.139.0 go test -run TestGenerateAllSchemas -v
#	OCB_VERSION=0.140.0 make build-collector
#	cd build && go mod vendor && SCHEMA_OUTPUT_DIR=../schemas/0.140.0 go test -run TestGenerateAllSchemas -v
#	OCB_VERSION=0.141.0 make build-collector
#	cd build && go mod vendor && SCHEMA_OUTPUT_DIR=../schemas/0.141.0 go test -run TestGenerateAllSchemas -v
#	OCB_VERSION=0.142.0 make build-collector
#	cd build && go mod vendor && SCHEMA_OUTPUT_DIR=../schemas/0.142.0 go test -run TestGenerateAllSchemas -v
#	OCB_VERSION=0.143.0 make build-collector
#	cd build && go mod vendor && SCHEMA_OUTPUT_DIR=../schemas/0.143.0 go test -run TestGenerateAllSchemas -v
#	OCB_VERSION=0.144.0 make build-collector
#	cd build && go mod vendor && SCHEMA_OUTPUT_DIR=../schemas/0.144.0 go test -run TestGenerateAllSchemas -v
#	OCB_VERSION=0.145.0 make build-collector
#	cd build && go mod vendor && SCHEMA_OUTPUT_DIR=../schemas/0.145.0 go test -run TestGenerateAllSchemas -v
#	OCB_VERSION=0.146.0 make build-collector
#	cd build && go mod vendor && SCHEMA_OUTPUT_DIR=../schemas/0.146.0 go test -run TestGenerateAllSchemas -v
#	OCB_VERSION=0.147.0 make build-collector
#	cd build && go mod vendor && SCHEMA_OUTPUT_DIR=../schemas/0.147.0 go test -run TestGenerateAllSchemas -v
#	OCB_VERSION=0.148.0 make build-collector
#	cd build && go mod vendor && SCHEMA_OUTPUT_DIR=../schemas/0.148.0 go test -run TestGenerateAllSchemas -v
#	OCB_VERSION=0.149.0 make build-collector
#	cd build && go mod vendor && SCHEMA_OUTPUT_DIR=../schemas/0.149.0 go test -run TestGenerateAllSchemas -v
	OCB_VERSION=0.150.0 make build-collector
	cd build && go mod vendor && SCHEMA_OUTPUT_DIR=../schemas/0.150.0 go test -run TestGenerateAllSchemas -v

.PHONY: changelogs
changelogs:
	@echo "Downloading OpenTelemetry CHANGELOG files..."
	@echo "Downloading opentelemetry-collector CHANGELOG.md..."
	mkdir -p tmp
	@curl -sSL -o tmp/opentelemetry-collector-CHANGELOG.md \
		https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector/main/CHANGELOG.md
	@echo "Downloading opentelemetry-collector-contrib CHANGELOG.md..."
	@curl -sSL -o tmp/opentelemetry-collector-CHANGELOG-contrib.md \
		https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector-contrib/main/CHANGELOG.md
	@echo "CHANGELOG files downloaded to schemas/ directory"
	@echo "Parsing changelog files and extracting version-specific content..."
	@./scripts/parse_changelogs.sh
	@echo "Version-specific changelog files generated"

.PHONY: clean-schemas
clean-schemas:
	rm -rf build/$(SCHEMA_OUTPUT_DIR)

.PHONY: test
test:
	@echo "Running tests in root package..."
	go test ./...
	@echo "Running tests in build package..."
	cd build && go test ./...

.PHONY: clean
clean: clean-schemas
	rm -rf _build .bin build/schema-generator
	rm -f ../schemas/opentelemetry-collector-CHANGELOG.md ../schemas/opentelemetry-collector-contrib-CHANGELOG.md

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all (default)               - Generate JSON schemas and process changelogs"
	@echo "  install-ocb                 - Install OpenTelemetry Collector Builder to ./.bin"
	@echo "                                Override version with: make OCB_VERSION=v0.110.0 install-ocb"
	@echo "  build-collector             - Build OpenTelemetry collector using manifest.yaml"
	@echo "  build-schema-generator      - Build standalone schema generator tool"
	@echo "  generate-schemas            - Generate JSON schemas using go test"
	@echo "                                Override output dir with: make SCHEMA_OUTPUT_DIR=my-schemas generate-schemas"
	@echo "  generate-schemas-standalone - Generate JSON schemas using standalone tool"
	@echo "  changelogs                  - Download CHANGELOG.md files and extract version-specific content"
	@echo "  test                        - Run tests in all packages"
	@echo "  clean-schemas               - Remove generated schema files"
	@echo "  clean                       - Remove build artifacts and local binaries"
	@echo "  help                        - Show this help message"