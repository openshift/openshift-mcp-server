# Contributing to MCP server for Red Hat OpenShift

> **Developer Preview** — This project is in a pre-release stage. Contributions are welcome; please review open issues and discussions before starting large pieces of work.

We'd love to have you join the community! Whether you're reporting issues, improving documentation, fixing bugs, or developing new features, your contributions are essential to our success.

## Code of Conduct

This project follows the [Red Hat Community Participation Guidelines](https://www.redhat.com/en/blog/red-hat-community-participation-guidelines). By participating, you are expected to uphold this code.

## Reporting Issues

Before reporting an issue, check the backlog of [open issues](https://github.com/openshift/openshift-mcp-server/issues) to see if someone else has already reported it. If so, feel free to add your scenario or additional information to the discussion. Please use a thumbs-up emoji on the original report rather than adding "+1" comments.

If you find a new bug or have a feature request, please open an issue. For bug reports, include enough information to reproduce the problem — the easier it is to reproduce, the faster it will be fixed. For feature requests, describe the use case and why the feature would be valuable.

Please do not include any private or sensitive information in your issue.

### Security Issues

Security issues should **NOT** be reported via GitHub issues. Report security vulnerabilities responsibly through one of these channels:

- Follow the [security policy](SECURITY.md) in this repository
- Contact [Red Hat Product Security](https://access.redhat.com/security/team/contact/) for issues affecting Red Hat products

### Support

For production support, file a case via [access.redhat.com](https://access.redhat.com):
- **Product**: OpenShift Container Platform
- **Component**: MCP server for Red Hat OpenShift

## Contributing Code

### Prerequisites

- [Go](https://go.dev/dl/) (version specified in `go.mod`)
- Access to an OpenShift or Kubernetes cluster for integration testing (optional)

### Getting Started

1. Fork the repository on GitHub.
2. Clone your fork locally:
   ```bash
   git clone https://github.com/<your-username>/openshift-mcp-server.git
   cd openshift-mcp-server
   ```
3. Create a new branch for your changes:
   ```bash
   git checkout -b my-feature
   ```
4. Make your changes and verify them:
   ```bash
   make build
   make test
   ```
5. Commit your changes (see [Commit Messages](#commit-messages)).
6. Push to your fork and [open a Pull Request](#pull-request-guidelines).

### Building

```bash
# Clean, tidy, format, lint, and build the binary
make build
```

The `build` target runs `clean`, `tidy`, `format`, and `lint` before compiling. The resulting executable is `openshift-mcp-server`. Run `make help` to see all available Makefile targets.

### Testing

```bash
make test
```

The test suite uses `setup-envtest` from `sigs.k8s.io/controller-runtime`, which provides a lightweight Kubernetes API server and etcd binary — no real cluster is required. The first run downloads the `envtest` environment, so network access is needed.

When writing tests:

- Use `testify/suite` for organizing tests into suites.
- Test the public API only (black-box testing).
- Use real implementations instead of mocks.
- Use nested subtests with `s.Run()` and descriptive names.
- Aim for one assertion per test case.
- Cover edge cases (nil inputs, empty values, invalid formats).

See [AGENTS.md](AGENTS.md#testing-patterns-and-guidelines) for detailed testing patterns and examples.

### Adding New MCP Tools

The project uses a toolset-based architecture:

1. Define the tool handler function implementing the tool's logic.
2. Create a `ServerTool` struct with the tool definition and handler in `pkg/api/`.
3. Add the tool to an appropriate toolset in `pkg/toolsets/` (or create a new toolset if needed).
4. Register the toolset in `pkg/toolsets/` if it's a new toolset.
5. Run `make update-readme-tools` to update the auto-generated toolset tables in `README.md` and `docs/configuration.md`.

When adding OpenShift-specific toolsets (e.g., under `pkg/toolsets/openshift/`, `pkg/toolsets/kubevirt/`, etc.), ensure:
- The toolset name follows the existing naming convention (lowercase, hyphenated where needed)
- Tools that access sensitive cluster data log via `mcplog.SendMCPLog()` — which automatically redacts credentials — rather than logging raw values
- Read-only behaviour is the default; write/destructive tools must be explicitly documented

### Dependencies

When introducing new modules, run `make tidy` so that `go.mod` and `go.sum` remain clean.

## Pull Request Guidelines

No Pull Request (PR) is too small! Typos, additional comments in the code, new test cases, bug fixes, new features, more documentation — it's all welcome!

All PRs should be submitted against the `main` branch. Maintainers will take care of backporting if needed.

While bug fixes can first be identified via an issue, that is not required. It's ok to just open a PR with the fix, but include the same information you would have included in an issue, such as how to reproduce it.

For larger new features, please open an issue or discussion first so the approach can be agreed upon before you invest significant time in the implementation. PRs for new features should include background on the use cases being addressed. Where sensible, break up larger PRs into smaller stand-alone ones — smaller changes are easier to review.

All PRs should include:

- **Well-documented code changes.** A commit message should answer *why* a change was made.
- **Tests.** Ideally, they should fail without your code change applied.
- **Documentation updates** if the changes affect user-facing behaviour.

## Commit Messages

This project uses [Conventional Commits](https://www.conventionalcommits.org/). Each commit message must follow the format:

```
<type>(<optional scope>): <description>

[optional body]

[optional footer(s)]
```

Common types include:

- `feat` — a new feature
- `fix` — a bug fix
- `docs` — documentation changes (see `docs/` and `docs/specs/`)
- `test` — adding or updating tests
- `refactor` — code changes that neither fix a bug nor add a feature
- `build` — changes to the build system or dependencies
- `ci` — changes to CI configuration
- `chore` — other changes that don't modify source or test files

Additional guidelines:

- Keep the subject line concise (under 72 characters).
- Separate subject from body with a blank line.
- In the body, explain *what* and *why*, not just *how*.
- Solve one problem per commit.
- Reference related issues with `Fixes: #00000` or `Closes: #00000`.

## Sign Your Commits

All commits must include a `Signed-off-by` trailer. This certifies that you wrote the patch or otherwise have the right to pass it on as an open-source patch under the [Developer Certificate of Origin](https://developercertificate.org/):

```
Developer Certificate of Origin
Version 1.1

Copyright (C) 2004, 2006 The Linux Foundation and its contributors.

By making a contribution to this project, I certify that:

(a) The contribution was created in whole or in part by me and I
    have the right to submit it under the open source license
    indicated in the file; or

(b) The contribution is based upon previous work that, to the best
    of my knowledge, is covered under an appropriate open source
    license and I have the right under that license to submit that
    work with modifications, whether created in whole or in part
    by me, under the same open source license (unless I am
    permitted to submit under a different license), as indicated
    in the file; or

(c) The contribution was provided directly to me by some other
    person who certified (a), (b) or (c) and I have not modified
    it.

(d) I understand and agree that this project and the contribution
    are public and that a record of the contribution (including all
    personal information I submit with it, including my sign-off) is
    maintained indefinitely and may be redistributed consistent with
    this project or the open source license(s) involved.
```

Add the sign-off line to every commit message:

```
Signed-off-by: Your Name <your.email@example.com>
```

If you set your `user.name` and `user.email` git configs, you can sign your commit automatically with `git commit -s`.

## Code Review

Once a PR is submitted, a maintainer will review it. If nobody responds within two weeks, please ping a maintainer. Sometimes PRs are overlooked.

Keep an eye on the CI results. If something fails, check the logs to see if it's related to your change. If you're unsure, ask in the PR comments.

If changes are requested, amend them into the relevant commit rather than adding extra "fix" commits. Use `git commit --amend` and force-push to your branch. This keeps the git history clean.

## Communication

- [GitHub Issues](https://github.com/openshift/openshift-mcp-server/issues) — for bugs and feature requests
- [GitHub Discussions](https://github.com/openshift/openshift-mcp-server/discussions) — for questions and general discussion
- [GitHub Pull Requests](https://github.com/openshift/openshift-mcp-server/pulls) — for code contributions

## Additional Resources

- [AGENTS.md](AGENTS.md) — detailed project structure, coding style, and testing guidelines
- [docs/openshift/user-guide.md](docs/openshift/user-guide.md) — full tool reference for the OpenShift fork
- [docs/](docs/) — user-facing documentation
- [docs/specs/](docs/specs/) — feature specifications (living documentation for contributors and coding agents)
- [README.md](README.md) — project overview and setup instructions
