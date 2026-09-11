# Project Agents.md for MCP server for Red Hat OpenShift

> **Developer Preview** — This project is in a pre-release stage, intended for developers and early adopters. It is not recommended for production use without reviewing the current limitations.

This Agents.md file provides comprehensive guidance for AI assistants and coding agents (like Claude, Gemini, Cursor, and others) to work with this codebase.

This repository contains the `openshift-mcp-server` project — a Go-based Model Context Protocol (MCP) server that provides native Red Hat OpenShift and Kubernetes cluster management capabilities without external dependencies. It is a downstream fork of [containers/kubernetes-mcp-server](https://github.com/containers/kubernetes-mcp-server) with OpenShift-specific toolsets and enterprise features.

This MCP server enables AI assistants (like Claude, Gemini, Cursor, and others) to interact with Red Hat OpenShift and Kubernetes clusters using the Model Context Protocol (MCP).

## Project Structure

Standard Go layout: `cmd/kubernetes-mcp-server/` (entry point), `pkg/` (libraries by domain).
The `npm/` and `python/` directories only wrap the compiled binary for distribution — do not add features there.
Build, test, and lint commands are in the `Makefile`.

### OpenShift-specific packages

The following packages and toolsets are downstream additions not present in the upstream project:

| Package / Toolset | Location | Description |
| :--- | :--- | :--- |
| `openshift` | `pkg/toolsets/openshift/` | OpenShift Projects, ClusterOperators, `plan_mustgather` prompt |
| `openshift/mustgather` | `pkg/toolsets/mustgather/` | Offline must-gather archive analysis |
| `cluster-diagnostics` | `pkg/toolsets/cluster-diagnostics/` | Node debug execution, cluster health |
| `kubevirt` | `pkg/toolsets/kubevirt/` | OpenShift Virtualization VM management |
| `kiali` | `pkg/toolsets/kiali/` | OpenShift Service Mesh via Kiali |
| `cni-diagnostics` | `pkg/toolsets/cni-diagnostics/` | Kernel CNI diagnostics, eBPF (pwru), tcpdump |
| `ovn-kubernetes` | `pkg/toolsets/ovnkubernetes/` | OVN/OVS network inspection (read-only) |
| `netobserv` | `pkg/toolsets/netobserv/` | NetObserv console plugin network flows |
| `netedge` | `pkg/toolsets/netedge/` | Ingress/Route/DNS/HAProxy diagnostics |
| `oadp` | `pkg/toolsets/oadp/` | OADP/Velero backup and restore diagnostics |

## Feature development

### Logging

When adding log lines, always use a contextual logger (`klogutil.FromContext(ctx)` from `pkg/klogutil`). If necessary, add a `ctx` parameter to the function, and wire the context through to where you need a logger.

`klogutil.FromContext` wraps `klog.FromContext` and injects the context into the logger's values so the OpenTelemetry log bridge can extract the active trace span for log-trace correlation. Do not use `klog.FromContext` directly in production code.

If you start a new trace span (e.g. `ctx, span := tracer.Start(ctx, "op")`), you must call `klogutil.FromContext(ctx)` again to pick up the new span — the logger captured before the span was created still carries the old (or empty) span context.

### Sensitive data and redaction

When tools return data that may contain credentials, tokens, or secrets — for example kubeconfig data, API responses with bearer tokens, or cluster resources containing `Secret` values — **always log through `mcplog.SendMCPLog()`** from `pkg/mcplog`. This automatically calls `Sanitize()` which redacts known secret patterns before the message is forwarded to the MCP client.

Never log raw credentials, tokens, or private key material directly. The `Sanitize()` function uses a best-effort denylist (Bearer/Basic auth headers, JWT tokens, AWS/GitHub/Anthropic keys, PEM blocks, DB connection strings, and common JSON/YAML field names like `"password"`, `"token"`, `"secret"`).

### Adding new MCP tools

The project uses a toolset-based architecture for organizing MCP tools:

- **Tool definitions** are created in `pkg/api/` using the `ServerTool` struct.
- **Toolsets** group related tools together (e.g., config tools, core Kubernetes tools, Helm tools).
- **Registration** happens in `pkg/toolsets/` where toolsets are registered at initialization.
- Each toolset lives in its own subdirectory under `pkg/toolsets/` (e.g., `pkg/toolsets/config/`, `pkg/toolsets/core/`, `pkg/toolsets/helm/`).

**Important:** When creating a new toolset, adding tools to an existing toolset, or modifying tool definitions, **always use the `/toolset-design` skill first**. This skill validates the design (naming, grouping, input schema, eval coverage) before implementation begins.

When adding a new tool:
1. Define the tool handler function that implements the tool's logic.
2. Create a `ServerTool` struct with the tool definition and handler.
3. Add the tool to an appropriate toolset (or create a new toolset if needed).
4. Register the toolset in `pkg/toolsets/` if it's a new toolset.
5. Run `make update-readme-tools` to regenerate toolset tables in `README.md` and `docs/configuration.md`.

When adding a new **OpenShift-specific** toolset:
- Default behaviour must be **read-only** unless explicitly enabled by the user
- Document the toolset in `docs/openshift/user-guide.md` following the existing table format
- Add an RBAC example showing the minimum required permissions
- Ensure tools that access node-level resources use ephemeral debug pods (not persistent pods)

## Building

Use the provided Makefile targets:

```bash
# Format source and build the binary
make build

# Build for all supported platforms
make build-all-platforms
```

`make build` will run `go fmt` and `go mod tidy` before compiling.
The resulting executable is `openshift-mcp-server`.

## Running

Run the server with the MCP Inspector for local testing:

```bash
make build
npx @modelcontextprotocol/inspector@latest $(pwd)/openshift-mcp-server
```

To run the server locally using npm or the binary directly:

```bash
# Using npx (Node.js package runner)
npx -y openshift-mcp-server@latest

# Using uvx (Python package runner)
uvx openshift-mcp-server@latest

# Binary execution
./openshift-mcp-server
```

This MCP server is designed to run both locally and remotely.

### Local Execution

When running locally, the server connects to a Red Hat OpenShift or Kubernetes cluster using the kubeconfig file. It reads the kubeconfig from the `--kubeconfig` flag, the `KUBECONFIG` environment variable, or defaults to `~/.kube/config`.

### Remote Execution

When running remotely, deploy the server as a container image in an OpenShift cluster. The server can run as a Deployment, StatefulSet, or any other Kubernetes resource. It automatically uses the in-cluster configuration to connect to the Kubernetes API server. The recommended deployment method is the Helm chart in `charts/`.

```bash
helm install openshift-mcp-server charts/openshift-mcp-server \
  --namespace openshift-mcp-server --create-namespace
```

## Tests

Run all Go tests with:

```bash
make test
```

The test suite relies on the `setup-envtest` tooling from `sigs.k8s.io/controller-runtime`. The first run downloads a Kubernetes `envtest` environment from the internet, so network access is required. Without it some tests will fail during setup.

Before writing or modifying tests, read [`docs/dev/testing.md`](docs/dev/testing.md) for the project's testing patterns, downstream compatibility rules, and examples.

## Linting

Static analysis is performed with `golangci-lint`:

```bash
make lint
```

The `lint` target downloads the specified `golangci-lint` version if it is not already present under `_output/tools/bin/`.

## Additional Makefile targets

Beyond the basic build, test, and lint targets, the Makefile provides additional utilities:

**Local Development:**
```bash
# Setup a complete local development environment with Minikube cluster
make local-env-setup

# Tear down the local Minikube cluster
make local-env-teardown

# Show Keycloak status and connection info (for OIDC testing)
make keycloak-status

# Tail Keycloak logs
make keycloak-logs
```

**Documentation:**
```bash
# Update README.md and docs/configuration.md with the latest toolset tables
make update-readme-tools
```

**Distribution and Publishing:**
```bash
# Copy compiled binaries to each npm package
make npm-copy-binaries

# Publish the npm packages
make npm-publish

# Publish the Python packages
make python-publish
```

Run `make help` to see all available targets with descriptions.

## Dependencies

When introducing new modules run `make tidy` so that `go.mod` and `go.sum` remain tidy.

## Coding style

- The Go version is declared in `go.mod`; CI installs whatever it requires via `go-version-file`.
- Tests are written with the standard library `testing` package.
- Build, test and lint steps are defined in the Makefile — keep them working.
- OpenShift-specific toolsets must default to read-only behaviour; document any write operations clearly.

## Documentation

The `docs/` directory contains user-facing documentation:

- `docs/openshift/user-guide.md` — **Primary reference** for the OpenShift fork: full tool tables, safety guidelines, and model evaluation results
- `docs/README.md` — Documentation index and navigation
- `docs/OPENSHIFT.md` — OpenShift toolset and `plan_mustgather` prompt reference
- `docs/configuration.md` — Complete TOML configuration reference (all `StaticConfig` options, drop-in configuration, dynamic reload)
- `docs/prompts.md` — MCP Prompts configuration guide
- `docs/logging.md` — MCP Logging guide (automatic Kubernetes error logging, secret redaction)
- `docs/OTEL.md` — OpenTelemetry observability setup
- `docs/observability/metrics.md` — Metrics toolset (Prometheus / Alertmanager via obs-mcp)
- `docs/observability/tracing.md` — Tracing toolset (Grafana Tempo via obs-mcp)
- `docs/observability/logs.md` — Logs toolset (Grafana Loki / LokiStack via obs-mcp)
- `docs/observability/otelcol.md` — OpenTelemetry Collector toolset
- `docs/kubevirt.md` — KubeVirt / OpenShift Virtualization toolset
- `docs/KIALI.md` — Kiali / OSSM toolset configuration
- `docs/OADP.md` — OADP backup and restore toolset
- `docs/NETOBSERV.md` — NetObserv network observability toolset
- `docs/openshift/ovn-kubernetes.md` — OVN-Kubernetes toolset
- `docs/openshift/cni-diagnostics.md` — CNI Diagnostics toolset
- `docs/openshift/NETEDGE.md` — NetEdge Ingress/DNS toolset
- `docs/openshift/acm_setup.md` — Advanced Cluster Management multi-cluster setup
- `docs/KEYCLOAK_OIDC_SETUP.md` — OAuth/OIDC developer setup with Keycloak

The `docs/specs/` directory contains feature specifications (living documentation for coding agents):

- `docs/specs/validation.md` — Pre-execution validation layer specification (resource existence, schema, RBAC)

### Documentation conventions

- Use **lowercase filenames** for new documentation files (e.g., `configuration.md`, `prompts.md`)
- OpenShift-specific docs go under `docs/openshift/`
- The toolsets table, tools, prompts, resources, and resource templates in `README.md` and `docs/configuration.md` are **auto-generated** — use `make update-readme-tools` to update them after modifying toolsets
- Both files use marker pairs for the generated content:
  - `<!-- AVAILABLE-TOOLSETS-START -->` / `<!-- AVAILABLE-TOOLSETS-END -->` (toolset summary table)
  - `<!-- AVAILABLE-TOOLSETS-TOOLS-START -->` / `<!-- AVAILABLE-TOOLSETS-TOOLS-END -->` (tool details)
  - `<!-- AVAILABLE-TOOLSETS-PROMPTS-START -->` / `<!-- AVAILABLE-TOOLSETS-PROMPTS-END -->` (prompt details)
  - `<!-- AVAILABLE-TOOLSETS-RESOURCES-START -->` / `<!-- AVAILABLE-TOOLSETS-RESOURCES-END -->` (resource details)
  - `<!-- AVAILABLE-TOOLSETS-RESOURCES-TEMPLATES-START -->` / `<!-- AVAILABLE-TOOLSETS-RESOURCES-TEMPLATES-END -->` (resource template details)

## Distribution Methods

The server is distributed as:

- **Native binaries** for Linux, macOS, and Windows — available in [GitHub Releases](https://github.com/openshift/openshift-mcp-server/releases)
- **Container image** — built and pushed to `ghcr.io/openshift/openshift-mcp-server`
- **npm package** — available at [npmjs.com](https://www.npmjs.com/package/openshift-mcp-server); wraps the platform-specific binary, runnable via `npx`
- **Python package** — available at [pypi.org](https://pypi.org/project/openshift-mcp-server/); runnable via `uvx` or `python -m openshift_mcp_server`
- **Helm chart** — available in `charts/` for in-cluster deployment on OpenShift
