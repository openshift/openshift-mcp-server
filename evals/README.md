# Kubernetes MCP Server Evals

This directory contains the [mcpchecker](https://github.com/mcpchecker/mcpchecker)
evaluations for the **Kubernetes MCP server**. The same tasks are run against the
same MCP server using different AI agents (Claude Code, an OpenAI-compatible agent,
and others), so agents can be compared on an identical benchmark.

## Structure

```
evals/
├── README.md                       # This file
├── mcp-config.yaml                 # Shared MCP server connection (http://localhost:8080/mcp)
├── core-eval-testing/              # Agent configs and per-suite eval configs (see below)
├── results/                        # Committed eval results (baselines)
└── tasks/                          # Shared task library, grouped by suite (see tasks/README.md)
    └── <suite>/                    # core, config, helm, kiali, kubevirt, tekton, netobserv, observability
        └── <task-name>/
            ├── <task-name>.yaml    # Task definition (prompt, verify, labels). kubevirt/tekton use task.yaml
            ├── setup.sh            # Optional pre-task cluster setup
            ├── verify.sh           # Optional post-task verification
            ├── cleanup.sh          # Optional resource cleanup
            └── artifacts/          # Optional K8s manifests
```

Tasks are grouped into suites via a `suite: <name>` label. The available suites are
`core`, `config`, `helm`, `kiali`, `kubevirt`, `tekton`, `netobserv`, and `observability`.

A second, orthogonal `readonly: "true"` label marks tasks that are safe to run
against a server started with `read_only = true`. It cuts across the `suite:` label
rather than replacing it — see [Read-only suite](#read-only-suite-core-readonly)
below.

## Prerequisites

- A Kubernetes cluster (kind, minikube, or any cluster) and `kubectl` configured for it
- The Kubernetes MCP server running at `http://localhost:8080/mcp` (see `make run-server`)
- `mcpchecker` and the standalone `agent`: `make mcpchecker` installs both under
  `_output/tools/bin/`. The
  `make run-evals` target calls that path directly; to run the bare `mcpchecker …`
  commands shown below by hand, add `_output/tools/bin` to your `PATH`.
  Installation requires Bash, `curl`, `unzip`, `mktemp`, `mkdir`, `sleep`, and
  either `sha256sum` or `shasum`. Its destination filesystem must support atomic
  directory creation for the per-destination installer lock. The current Prow
  `rhel-9-golang-1.26-openshift-5.0` build root provides these prerequisites;
  revalidate them when the CI build root changes.
- For the **`acp-anthropic`** agent only:
  - The `claude` CLI (Claude Code) installed, on your `PATH`, and **signed in**.
    The agent runs through your existing Claude Code session, so your Claude
    subscription (or an `ANTHROPIC_API_KEY`) covers the agent and no separate key
    is needed. Run `claude` once interactively to log in if you have not already.
  - The `claude-agent-acp` adapter, which bridges mcpchecker to that `claude` CLI,
    installed with `make claude-agent-acp` (installs locally under
    `_output/tools/node_modules/`, so it needs Node.js/npm). `make run-evals`
    prepends `_output/tools/node_modules/.bin` to `PATH` automatically.

## Quickstart: run one suite locally with Claude Code (ACP)

This runs the `kubevirt` suite end to end with the `acp-anthropic` agent on a
Sonnet model, with **no OpenAI key required**. With this agent both the agent and
the LLM judge run on your Claude subscription, so every suite is keyless (see
[Eval configs](#eval-configs) below).

```bash
# 1. Build the server and install the eval tooling
make build mcpchecker claude-agent-acp

# 2. Create a cluster and point KUBECONFIG at it.
#    KUBECONFIG must be EXPORTED so both the kubernetes extension's setup steps
#    and each task's verify.sh kubectl target the same cluster.
make minikube-create-cluster MINIKUBE_PROFILE=mcp-eval-cluster
export KUBECONFIG="$PWD/_output/kubeconfig"

# 3. Install KubeVirt (only needed for the kubevirt suite).
#    This also auto-deploys the common-instancetypes (u1.small, u1.medium, ...)
#    via virt-operator, so instancetype tasks work without a separate step.
make kubevirt-install

# 4. Start the MCP server with the toolsets the suite needs
make run-server TOOLSETS=core,config,kubevirt

# 5. Run the suite. AGENT selects the agent, SUITE selects the tasks, MODEL sets
#    ANTHROPIC_MODEL for ACP agents. Add EVAL_TASK_FILTER=<name> to run one task.
make run-evals SUITE=kubevirt AGENT=acp-anthropic MODEL=sonnet

# 6. Tear down
make stop-server
make minikube-delete-cluster MINIKUBE_PROFILE=mcp-eval-cluster
```

> A bare `make minikube-create-cluster` is enough for kubevirt and most toolset suites.
> `make local-env-setup` also installs cert-manager and Keycloak, which toolset
> evals do not need (and which pull from registries that are not always reachable).

The make-target invocation in step 5 is equivalent to the underlying command:

```bash
KUBECONFIG="$PWD/_output/kubeconfig" ANTHROPIC_MODEL=sonnet mcpchecker check \
  evals/core-eval-testing/acp-anthropic/eval-kubevirt.yaml \
  --label-selector suite=kubevirt --output json
```

### Model selection

The model for the `acp-anthropic` agent is chosen with the `ANTHROPIC_MODEL`
environment variable (for example `ANTHROPIC_MODEL=sonnet`), which
`claude-agent-acp` reads as its first-priority model source. It **forces** that
model for the run, overriding whatever default your Claude Code session would
otherwise use, and it runs on your existing Claude subscription (no API key
needed). `make run-evals` sets it for you from the `MODEL` variable, so
`make run-evals ... MODEL=sonnet` is how you pin the eval to Sonnet. The same
variable also drives the judge (see [Eval configs](#eval-configs)), so agent and
judge share the model.

The `builtin-*` agents instead take their model from their `agent.yaml` plus the
`MODEL_BASE_URL` and `MODEL_KEY` environment variables.

## Running with a builtin agent

```bash
# Set your model credentials
export MODEL_BASE_URL='https://your-api-endpoint.com/v1'
export MODEL_KEY='your-api-key'

# Run the core suite with builtin-openai (this is what CI runs)
make run-evals SUITE=core AGENT=builtin-openai
# equivalently:
mcpchecker check evals/core-eval-testing/builtin-openai/eval-core.yaml --label-selector suite=core

# Or use a different builtin agent:
make run-evals SUITE=core AGENT=builtin-anthropic
make run-evals SUITE=core AGENT=builtin-google
```

Different models may pick different tools (`pods_*` or `resources_*`) for the same
task. The assertions accept either, for example `toolPattern: "(pods_.*|resources_.*)"`.

## Eval configs

`make run-evals` resolves the eval config in priority order:

1. **Per-suite override** (`evals/tasks/<suite>/<agent>/eval.yaml`) — takes
   precedence when present. No suite currently ships one; this is a hook for
   dropping in a one-off local override.
2. **Core-eval-testing** (`evals/core-eval-testing/<agent>/eval-<suite>.yaml`) —
   the canonical per-suite config for each agent, and what every suite resolves to
   today. **CI** runs `builtin-openai` here. Available agents: `builtin-openai`,
   `builtin-anthropic`, `builtin-google`, `acp-anthropic`, `acp-google` (not all
   have eval configs yet).

Each eval config's `llmJudge` references the same `agent.yaml` used by the agent
itself, so the judge reuses the agent's model and credentials — no separate judge
setup is needed.

A task is "judge-backed" when its verify phase has an `llmJudge` step —
**including the legacy `verify: contains:` short form, which is judge-evaluated
(semantic), not a literal string match.** Judge-backed task counts per suite:
`helm` 0, `core` 4, `kubevirt` 4, `tekton` 5, `config` 3, `kiali` 18. Filter
any config to one suite with `--label-selector suite=<name>`.

## Agent configuration

Agent configs live in `evals/core-eval-testing/<agent>/agent.yaml`. There are two
kinds:

**ACP agents** run through an external CLI adapter:

- **`acp-anthropic`** — uses `claude-agent-acp` (Claude Code via ACP, keyless)
- **`acp-google`** — uses `gemini --experimental-acp`

**Builtin agents** use mcpchecker's built-in LLM agent:

- **`builtin-openai`** — `openai:gpt-6-luna` (what CI runs)
- **`builtin-anthropic`** — `anthropic:claude-sonnet-4-6`
- **`builtin-google`** — `google:gemini-3.1-pro-preview`

## Filtering tasks by suite

The `--label-selector` flag (or the `SUITE` make variable) chooses which tasks run.

```bash
# core tasks (builtin-openai is the default AGENT)
make run-evals SUITE=core

# kiali tasks (needs Istio + Kiali installed; see `make setup-kiali`)
make run-evals SUITE=kiali AGENT=acp-anthropic MODEL=sonnet

# netobserv tasks (needs mock plugin or real NetObserv; see tasks/netobserv/README.md)
make run-evals SUITE=netobserv AGENT=builtin-openai
```

If you omit `--label-selector`, the eval config's own `labelSelector` settings in
each `taskSets` entry determine which tasks run.

Note: with `AGENT=acp-anthropic` every suite is keyless (agent and judge both run
on your Claude subscription). With `builtin-*` agents the judge-backed tasks need
the provider's API key (see [Eval configs](#eval-configs)).

## Read-only suite (`core-readonly`)

Proves that a model using only the tools exposed by a server started with
`read_only=true` can still complete real read-only diagnostic workflows (log
inspection, event filtering, metrics). It reuses the read-only-compatible
tasks already in `core`/`config` via the `readonly: "true"` label (see
[Filtering tasks by suite](#filtering-tasks-by-suite) above) instead of
duplicating them into a new suite.

As a suite-level safety net, every task also asserts `toolsNotUsed` for
`resources_create_or_update|resources_delete|resources_scale|pods_delete|pods_exec|pods_run`
(see `evals/core-eval-testing/builtin-openai/eval-core-readonly.yaml`). `pods_exec`
is included even though it isn't a CRUD write tool because it's annotated
`DestructiveHint: true` in `pkg/toolsets/core/pods.go` -- depending on the pod's
entrypoint, an executed command can kill the pod.

Note this suite intentionally does **not** cover write-blocking itself:
whether `read_only=true` actually rejects a write attempt (as opposed to just
hiding write tools from the tool list) is a deterministic property of the
server, not of model behavior, so it's covered by
`TestReadOnlyBlocksWriteToolInvocation` in `pkg/mcp/mcp_tools_test.go` instead
of an eval task — see [`docs/dev/testing.md`](../docs/dev/testing.md). Using a
live LLM to verify a property that doesn't depend on the model would just add
cost and non-determinism for no extra assurance.

The MCP server **must** be started in read-only mode for this suite — its
tasks assume write tools are unavailable. With `SUITE=core-readonly`,
`run-server` defaults `READ_ONLY` to `true` and adds `read_only = true` as the
last drop-in of its generated config dir, so it overrides the `read_only = false`
in `dev/config/mcp-configs`. The suite does not check the server's mode itself,
so against a server without read-only mode it would still pass without proving
anything. Likewise, `EVAL_LABEL_SELECTOR` defaults to `readonly=true` instead of
`suite=$(SUITE)`:

```bash
make run-server SUITE=core-readonly TOOLSETS=core,config
make run-evals SUITE=core-readonly
```

`READ_ONLY=true|false` can still be passed explicitly to override the default.

## Versions

`make mcpchecker` installs the official **v0.0.22 checker and standalone agent**
release binaries, verifying each archive against the twelve committed hashes in
`build/mcpchecker-v0.0.22.sha256` before extracting only its expected member.
Both components support Linux, macOS and Windows on amd64/arm64; other
architectures fail explicitly rather than building from source or skipping.
Prow and local Make consumers use this installer without compiling the dependency
with the server's Go toolchain. The current `builtin-*` configurations instantiate
their LLM agent **in-process**; installing the standalone `agent` does not mean
those configurations launch it. Its absolute `agent-path` output and tools/bin
PATH entry remain available for external ACP consumers. Existing agent/model and
judge options are unchanged.
The installer serializes concurrent updates to one destination and reuses the
cache only when its metadata names the requested version, platform, and committed
archive checksum and the binary still matches the digest stored in that metadata.
This detects stale metadata and binary-only modification; because the metadata is
stored beside the binary and is writable by the same user, it does not protect
against hostile modification of both files. Published metadata is explicitly
mode `0644` rather than inheriting a restrictive umask; the binary remains mode
`0755`. The release download uses a 10-second connection timeout and a
300-second total timeout so the installer cannot hold the lock indefinitely.
Lock acquisition is separately bounded and the lock is removed on normal exits.
An uncatchable termination can
leave a lock directory behind; a later installer fails explicitly after its
timeout so the caller can check for active installers before removing the lock.
Only v0.0.22 is supported by the evaluation composites and both-component Make
target. The original six-row `build/mcpchecker-v0.0.21.sha256` manifest and
five-argument installer API remain for explicit **checker-only** compatibility:

```bash
bash build/install-mcpchecker.sh v0.0.21 linux amd64 \
  "$PWD/_output/tools/bin/mcpchecker" "$PWD/build/mcpchecker-v0.0.21.sha256"
```

The optional sixth argument selects `mcpchecker` (default) or `agent`; v0.0.21
has no approved agent row and cannot silently fetch one. Unknown versions or
version/component combinations fail closed. A writable cache plus its colocated
metadata is not an adversarial trust boundary. The committed manifest in reviewed
source, not a freshly downloaded checksum, is the runtime archive trust anchor.
The hashes are publisher-channel integrity checks, not a signature or attestation.

The two evaluation composites use fixed source-root helpers and pinned third-party
actions, not a mutable nested setup action. Setup installs both components into a
fresh owned action-instance directory without executing either downloaded binary.

`make claude-agent-acp` likewise installs `@agentclientprotocol/claude-agent-acp@latest`;
pin it with `make claude-agent-acp CLAUDE_AGENT_ACP_VERSION=<version>` for local
reproducibility. CI never installs the adapter (it runs `builtin-openai`), so this
knob is purely local.

## Compatibility filtering maintenance

Existing netedge tools use lazy exact-GVK compatibility requirements supplied by
the current `api.ClusterInspector` API. Keep each requirement beside its handler;
when a handler's API dependency changes, update that literal, the independent
constructor/registration contracts and the full catalog negative tests together.

| Tools | Exact group/version/kind |
|---|---|
| `query_prometheus` (toolset-name prefix), `inspect_route` | `route.openshift.io/v1`, `Route` |
| `get_coredns_config` | core group `""`, `v1`, `ConfigMap` |
| `get_service_endpoints` | `discovery.k8s.io/v1`, `EndpointSlice` |
| `exec_dns_in_pod`, `get_router_config`, `get_router_info`, `get_router_sessions` | core group `""`, `v1`, `Pod` |
| `probe_dns_local`, `probe_http` | None; both local probes remain ungated |

These are eight requirements and two ungated tools, not object, namespace, verb,
RBAC, connectivity or operator-configuration checks. Any enumerated target with
the exact GVK is enough. Confirmed absence hides only dependent tools; unknown
discovery/403/transport errors fail open. Disabled filtering performs no such
queries. Current ACM inspection enumerates managed targets; an empty target set
hides gated tools, and error aggregation can continue through later targets.
The discovery REST client has a bounded per-client timeout, but this is **not a
global deadline guarantee** for aggregate inspection. No provider/watcher policy
or ordinary non-discovery HTTP behavior changes here.

## Workflow trust and deployment status

The workflow separates a secret-free, read-only candidate build from protected
evaluation. **C** is the whole candidate commit. **H** is a separately reviewed,
immutable harness commit checked out from the fixed public repository
`redhat-chai-bot/openshift_openshift-mcp-server`. Only the compiled candidate
server crosses into the harness; candidate Makefiles, scripts, configs, tasks,
actions and caches do not. Trusted `make -C trusted -o build run-server` uses H's
configuration without rebuilding the server. Read-only API identity/review/SHA
checks establish eligibility and reject drift; they do **not** make code safe or
constitute credential-release approval. Before deployment and each approval, an
independent maintainer must audit that administrative bypass of the protected
environment is disabled. The REST environment response does not expose that
setting: automated eligibility checks do not verify it. This external audited
deployment prerequisite is mandatory, alongside protected approval of exact C/H.

The `mcpchecker-reviewed` protected environment requires independent human
approval of **exact C and H, including the whole candidate as privileged code
execution**. Candidate and evaluator share an OS account: this is **not a
sandbox**, and removing model variables from the server's environment cannot
prove it cannot inspect sibling files/processes. Do not approve or run if that
risk is unacceptable; there is no automatic safe-fork fallback. Model/judge
credentials are confined to the trusted evaluation step, not build, setup,
server, diff, reporter or scheduled writer.

**A harness pin is not approval for credentialed execution.** The older published
workflow pinned H `efd3cf05f607ab7804c4a16b543d916ebfc51a74`, followed by H2
`62b465f982d92ca72128eb52a819925ac9ef26cd`. The published C3 source
`601554bed0375083b88d62d115d3ed0cde324fed` pins immutable H3
`4942815c1af74894f58640938c12c4f84623ec69`. H3 and C3 completed independent
review and local repository gates. Those historical local checks do not
substitute for the integrated tree's own validation, deployment, or maintainer
risk approval. The published H4
`b918eee433d300093cc73b72f05d9a8e55a2a88e` is a signed history-preserving merge
of published C3 and main `432c1f3cd76df74b0bdb9ca010643abbac996219`. H4 completed
independent integration review and full precommit/postcommit local repository
gates. Published C4 `4452cb14c4fb6767ee4c02ccbae7f38ab3db8e7e` pins H4;
H4 and C4 completed independent review and local mandatory gates before publication.
The published H5 `aa2cde6a978111cd162b79082084175bf5b0e820` is a
signed child of published C4 containing the reviewed bootstrap, observability TLS,
and app-message security corrections. H5 completed independent review and full
local mandatory gates. Published C5
`ec4b1c715e7cbbfbd568a99d3c27798952cb3550` pins that actual immutable H5;
C5 also completed independent review and local mandatory gates before publication.
The subsequent Prow jobs failed during cloning/merging with newer main, before
tests or a new security scan executed. The local integration with main
`87f6e7c0f60b122de54e5becbf57314d59f7c3dc` is not yet committed or validated
as a new harness. The workflow retains H5 until a reviewed, validated H6 exists
and a separately reviewed candidate update pins its actual SHA. No new external
SECURITY clearance has been established by these local checks.
Neither local validation nor pinning deploys or activates credentialed workflows.
No branch, artifact or label substitutes for the reviewed immutable harness.

### Operator-only Kiali multicluster

The upstream Kiali multicluster local Make targets, configurations, task links,
and operator instructions remain available for explicitly operator-run use; see
[`core-eval-testing/README.md`](core-eval-testing/README.md) and `build/kiali.mk`.
Automatic protected evaluation, reporter mapping, and scheduled baseline branches
for Kiali multicluster are deferred. They are not part of the trusted supported
suite allowlist. The local setup defaults to mutable Kiali `master`/`dev` inputs
and transitive external scripts; pinning this repository alone does not make
those inputs immutable. Any future automatic integration requires separately
reviewed immutable dependencies and lifecycle/consumer checks. Keeping these
operator paths does not authorize their execution, credentialed runs, or workflow
activation.

A maintainer must separately deploy the reviewed workflow/report code to the
trusted default branch, configure the protected environment and scoped secrets,
and approve exact C/H before any credentialed run. A feature-branch change alone
does not deploy `workflow_run` code. The existing downstream repository guard
remains unchanged and dormant. Publication, guard activation, secret/environment
configuration and live workflow/model/cluster runs are **not authorized by local
implementation or offline tests**; none is claimed to have happened.

The evaluation runner requires Linux child-subreaper support, Node.js and Python 3
(standard library only) on PATH. The protected workflow uses GitHub-hosted Ubuntu;
generic macOS/Windows callers are rejected, not claimed supported. A capture owner
supervises the checker, diagnostic drainer and inherited-pipe descendants through
completion. Its fixed 1800-second deadline includes EOF collection even after the
checker parent exits. Expiry always fails capture, sends TERM to the owned checker
group, then escalates after 30 seconds and allows five seconds for EOF draining
before killing a stuck drainer. It reaps owned children, with an explicit failure
if another five seconds of post-KILL reaping cannot complete.
Actual checker/drainer statuses remain separate from capture expiry. Unsupported
subreaper setup and incomplete cleanup fail explicitly; this is not a sandbox or
a guarantee against processes escaping their owned group or kernel-stuck tasks.
Diagnostics are drained to EOF and bounded to 1 MiB after redaction, with an
explicit truncation marker. Complete summary JSON has a separate 64 MiB bound;
it is parsed before publishing a bounded excerpt. Result acceptance stays 64 MiB.
These are offline-checked wrapper contracts, not native multi-OS or model-runtime
attestations.

The evaluation collects diagnostics with `fail-on-error=false`, but its final
gate requires successful check, fresh valid nonempty results, successful summary
and verification, and successful applicable diff. A missing baseline is an
explicit optional absence; a corrupt present baseline fails. Task/assertion
thresholds remain **0.8**, not 100% task success; task assertions, maxToolCalls,
security defaults and CI Go versions are unchanged. Generic composite thresholds
still default to 1.0. Reporter artifacts are bounded typed DATA, isolated per
suite and API-bound to source run/attempt/current reviewed PR head; raw task/diff
text is not rendered into comments. Trusted run-name metadata records the source
trigger run, with no artifact-selected PR fallback. Scheduled baseline updates
remain schedule-only and may retain valid partial failed-suite results, never
manufacture a pass or consume a PR candidate's results.

The three RBAC task contracts/production policy and cross-repository Prow
extension kubeconfig staging remain deferred. Main's tools/bin PATH and
`MCP_EVAL_KUBECONFIG`/`KUBECONFIG` precedence are retained, not a claim that all
Prow or Google-agent failures are fixed. Offline fixtures do not validate live
GitHub approvals, model behavior or cluster semantics.

## Istio/Kiali bootstrap

The trusted Kiali bootstrap retains Istio **1.31.0**, Kiali **v2.33**, the demo
profile, tracing transform, sample addons and readiness waits. It downloads a
fixed official Istio archive over HTTPS with 10-second connection/300-second
total timeouts, checks `build/istio-1.31.0.sha256` before listing/extracting, and
publishes only regular `bin/istioctl` and the three required addon YAML files.
It does not execute a downloaded installer script or reuse an unchecked binary.
Linux amd64/arm64/armv7 and macOS amd64/arm64 map to five approved archives;
unsupported platforms fail closed. Local bootstrap prerequisites include Bash,
curl, tar/gzip, sed, grep, mktemp, core file utilities and sha256sum or shasum.

Both Bookinfo inputs use `openshift-service-mesh/istio` commit
`b6fb5348dd66280245e3d98d8d93f416ece453c3`; the Sail ingress sample uses
`istio-ecosystem/sail-operator` commit
`0c7171a9fdc4762366a78e9a0e3f226ff6e6bbd3`. There is no main/master fallback
if a pinned path disappears. Fixed source pointers and archive hashes do not
attest every runtime image or eliminate inherited publisher/runner trust; this
is not a hermetic or sandboxed supply chain. Public archive bytes and pinned
YAML can be inspected as data without running istioctl or contacting a cluster.

## Expected results

A successful run reports tasks passed, assertions passed (the expected tools were
called), and verification passed (the cluster ends in the expected state). Results
are written to `mcpchecker-<eval-name>-out.json`.
