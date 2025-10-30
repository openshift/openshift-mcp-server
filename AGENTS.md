# Project Agents.md for Kubernetes MCP Server

This Agents.md file provides comprehensive guidance for AI assistants and coding agents (like Claude, Gemini, Cursor, and others) to work with this codebase.

This repository contains the kubernetes-mcp-server project,
a powerful Go-based Model Context Protocol (MCP) server that provides native Kubernetes and OpenShift cluster management capabilities without external dependencies.
This MCP server enables AI assistants (like Claude, Gemini, Cursor, and others) to interact with Kubernetes clusters using the Model Context Protocol (MCP).

## Project Structure and Repository layout

- Go package layout follows the standard Go conventions:
  - `cmd/kubernetes-mcp-server/` – main application entry point using Cobra CLI framework.
  - `pkg/` – libraries grouped by domain.
    - `api/` - API-related functionality, tool definitions, and toolset interfaces.
    - `config/` – configuration management.
    - `helm/` - Helm chart operations integration.
    - `http/` - HTTP server and authorization middleware.
    - `kubernetes/` - Kubernetes client management, authentication, and access control.
    - `mcp/` - Model Context Protocol (MCP) server implementation with tool registration and STDIO/HTTP support.
    - `output/` - output formatting and rendering.
    - `toolsets/` - Toolset registration and management for MCP tools.
    - `version/` - Version information management.
- `.github/` – GitHub-related configuration (Actions workflows, issue templates...).
- `docs/` – documentation files.
- `npm/` – Node packages that wraps the compiled binaries for distribution through npmjs.com.
- `python/` – Python package providing a script that downloads the correct platform binary from the GitHub releases page and runs it for distribution through pypi.org.
- `Dockerfile` - container image description file to distribute the server as a container image.
- `Makefile` – tasks for building, formatting, linting and testing.

## Feature development

Implement new functionality in the Go sources under `cmd/` and `pkg/`.
The JavaScript (`npm/`) and Python (`python/`) directories only wrap the compiled binary for distribution (npm and PyPI).
Most changes will not require touching them unless the version or packaging needs to be updated.

### Adding new MCP tools

The project uses a toolset-based architecture for organizing MCP tools:

- **Tool definitions** are created in `pkg/api/` using the `ServerTool` struct.
- **Toolsets** group related tools together (e.g., config tools, core Kubernetes tools, Helm tools).
- **Registration** happens in `pkg/toolsets/` where toolsets are registered at initialization.
- Each toolset lives in its own subdirectory under `pkg/toolsets/` (e.g., `pkg/toolsets/config/`, `pkg/toolsets/core/`, `pkg/toolsets/helm/`).

When adding a new tool:
1. Define the tool handler function that implements the tool's logic.
2. Create a `ServerTool` struct with the tool definition and handler.
3. Add the tool to an appropriate toolset (or create a new toolset if needed).
4. Register the toolset in `pkg/toolsets/` if it's a new toolset.

## Building

Use the provided Makefile targets:

```bash
# Format source and build the binary
make build

# Build for all supported platforms
make build-all-platforms
```

`make build` will run `go fmt` and `go mod tidy` before compiling.
The resulting executable is `kubernetes-mcp-server`.

## Running

The README demonstrates running the server via
[`mcp-inspector`](https://modelcontextprotocol.io/docs/tools/inspector):

```bash
make build
npx @modelcontextprotocol/inspector@latest $(pwd)/kubernetes-mcp-server
```

To run the server locally, you can use `npx`, `uvx` or execute the binary directly:

```bash
# Using npx (Node.js package runner)
npx -y kubernetes-mcp-server@latest

# Using uvx (Python package runner)
uvx kubernetes-mcp-server@latest

# Binary execution
./kubernetes-mcp-server
```

This MCP server is designed to run both locally and remotely.

### Local Execution

When running locally, the server connects to a Kubernetes or OpenShift cluster using the kubeconfig file.
It reads the kubeconfig from the `--kubeconfig` flag, the `KUBECONFIG` environment variable, or defaults to `~/.kube/config`.

This means that `npx -y kubernetes-mcp-server@latest` on a workstation will talk to whatever cluster your current kubeconfig points to (e.g. a local Kind cluster).

### Remote Execution

When running remotely, the server can be deployed as a container image in a Kubernetes or OpenShift cluster.
The server can be run as a Deployment, StatefulSet, or any other Kubernetes resource that suits your needs.
The server will automatically use the in-cluster configuration to connect to the Kubernetes API server.

## Tests

Run all Go tests with:

```bash
make test
```

The test suite relies on the `setup-envtest` tooling from `sigs.k8s.io/controller-runtime`.
The first run downloads a Kubernetes `envtest` environment from the internet, so network access is required.
Without it some tests will fail during setup.

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
# Setup a complete local development environment with Kind cluster
make local-env-setup

# Tear down the local Kind cluster
make local-env-teardown

# Show Keycloak status and connection info (for OIDC testing)
make keycloak-status

# Tail Keycloak logs
make keycloak-logs

# Install required development tools (like Kind) to ./_output/bin/
make tools
```

**Distribution and Publishing:**
```bash
# Copy compiled binaries to each npm package
make npm-copy-binaries

# Publish the npm packages
make npm-publish

# Publish the Python packages
make python-publish

# Update README.md with the latest toolsets
make update-readme-tools
```

Run `make help` to see all available targets with descriptions.

## Dependencies

When introducing new modules run `make tidy` so that `go.mod` and `go.sum` remain tidy.

## Coding style

- Go modules target Go **1.24** (see `go.mod`).
- Tests are written with the standard library `testing` package.
- Build, test and lint steps are defined in the Makefile—keep them working.

## Distribution Methods

The server is distributed as a binary executable, a Docker image, an npm package, and a Python package.

- **Native binaries** for Linux, macOS, and Windows are available in the GitHub releases.
- A **container image** (Docker) is built and pushed to the `quay.io/manusa/kubernetes_mcp_server` repository.
- An **npm** package is available at [npmjs.com](https://www.npmjs.com/package/kubernetes-mcp-server).
  It wraps the platform-specific binary and provides a convenient way to run the server using `npx`.
- A **Python** package is available at [pypi.org](https://pypi.org/project/kubernetes-mcp-server/).
  It provides a script that downloads the correct platform binary from the GitHub releases page and runs it.
  It provides a convenient way to run the server using `uvx` or `python -m kubernetes_mcp_server`.

## Contributing to Red Hat OpenShift MCP Server

This is a **fork** of the official Red Hat openshift-mcp-server repository. When contributing upstream, follow these strict requirements.

### Git Remote Configuration

This fork has the following remotes configured:

```bash
fork:     https://github.com/macayaven/openshift-mcp-server.git     # Your fork
origin:   https://github.com/openshift/openshift-mcp-server.git     # Red Hat upstream
upstream: https://github.com/containers/kubernetes-mcp-server.git   # Original project
```

### Red Hat PR Requirements (from PR #63 feedback)

**CRITICAL**: Red Hat reviewers require clean, well-organized PRs. Based on PR #63 feedback:

1. **Clean Commit History - MANDATORY**
   - **Rebase before submitting**: Squash WIP commits into logical, reviewable chunks
   - **One feature per commit** (or small number of well-organized commits)
   - Reviewers WILL request rebasing if there are too many commits
   - Use `git rebase -i origin/main` to clean up history before creating PR
   - Example: 37 commits → rebased to 3-5 logical commits

2. **Commit Message Format**
   - Follow conventional commits style
   - Clear, descriptive commit messages
   - Reference issues if applicable
   - Format: `type: description` (e.g., `feat: add OpenShift AI toolset`)

3. **Code Quality Requirements**
   - Must pass `make build`
   - Must pass `make lint`
   - Must pass `make test`
   - Test with mcp-inspector: `npx @modelcontextprotocol/inspector@latest $(pwd)/kubernetes-mcp-server`

4. **PR Approval Process**
   - PRs require approval from OWNERS file maintainers
   - Organization members must verify patches with `/ok-to-test` comment
   - Need `lgtm` label for merge
   - First-time contributors receive community support

5. **Architecture Principles**
   - Go-based native implementation (NOT kubectl/helm wrappers)
   - Direct Kubernetes API interaction via client-go
   - Maintain extensive test coverage
   - Follow existing package structure (`pkg/`, `cmd/`, `internal/`)

### PR Workflow for Red Hat Upstream

```bash
# 1. Ensure you're on your feature branch with changes
git checkout 001-openshift-ai

# 2. Fetch latest from Red Hat upstream
git fetch origin

# 3. Rebase onto latest upstream main
git rebase origin/main

# 4. IMPORTANT: Clean up commit history
git rebase -i origin/main
# In the interactive rebase:
# - Squash WIP commits
# - Combine related changes
# - Keep commits logical and reviewable
# - Aim for 3-5 commits maximum per feature

# 5. Force push to your fork (this updates the PR)
git push fork 001-openshift-ai --force-with-lease

# 6. Verify build and tests pass
make build
make lint
make test

# 7. Test with mcp-inspector
npx @modelcontextprotocol/inspector@latest $(pwd)/kubernetes-mcp-server

# 8. Create PR to Red Hat upstream
# - Title: Clear, descriptive (e.g., "Add OpenShift AI toolset")
# - Description: What, why, how
# - Link to any related issues
# - Include testing notes
```

### Example: Cleaning Up Commit History

```bash
# Before: 37 commits with WIP, fixes, debugging
git log --oneline | head -40

# Interactive rebase to squash
git rebase -i origin/main

# In editor, change commits like this:
# pick abc1234 feat: add OpenShift AI client package
# squash def5678 wip: update imports
# squash ghi9012 fix: typo in client
# pick jkl3456 feat: implement DataScienceProjects tools
# squash mno7890 fix: linting errors
# pick pqr1234 feat: register OpenShift AI toolset
# squash stu5678 docs: update README

# After: 3 clean commits
# 1. feat: add OpenShift AI client package
# 2. feat: implement DataScienceProjects tools
# 3. feat: register OpenShift AI toolset
```

### Common Pitfalls to Avoid

- ❌ Submitting PRs with 30+ commits
- ❌ Commits like "wip", "fix", "debugging", "oops"
- ❌ Not rebasing before submitting
- ❌ Not running `make build` and `make test` before PR
- ❌ Wrapping kubectl/helm instead of using Go clients
- ❌ Adding features without tests

### PR Checklist Before Submitting

- [ ] Rebased onto latest `origin/main`
- [ ] Commit history cleaned up (3-5 logical commits)
- [ ] `make build` passes
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] Tested with `mcp-inspector`
- [ ] Commit messages follow conventional format
- [ ] PR description explains what, why, how
- [ ] Tests added for new functionality
- [ ] Documentation updated if needed

### Local Testing with MCP Inspector

```bash
# Build the server
make build

# Test interactively with MCP Inspector
npx @modelcontextprotocol/inspector@latest $(pwd)/kubernetes-mcp-server

# This opens a web UI where you can:
# - Browse available tools
# - Test tool invocations
# - Inspect JSON schemas
# - Verify tool responses
```

### Resources

- [Previous PR example](https://github.com/openshift/openshift-mcp-server/pull/63)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [MCP Inspector](https://modelcontextprotocol.io/docs/tools/inspector)
