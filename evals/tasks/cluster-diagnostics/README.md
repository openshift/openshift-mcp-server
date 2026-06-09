# Cluster Diagnostics Task Stack

Cluster diagnostics MCP tasks live here. Each subdirectory is a self-contained scenario that exercises the `cluster-diagnostics` toolset—primarily `nodes_debug_exec` for privileged, ephemeral node-level debugging.

## Prerequisites

- Kubernetes cluster with at least one control-plane node (Kind is used in CI as `mcp-eval-cluster`)
- `kubectl` configured for that cluster
- MCP server built (`make build`) and reachable at `http://localhost:8080/mcp` (see [`evals/mcp-config.yaml`](../../mcp-config.yaml))
- MCP server running with the `cluster-diagnostics` toolset enabled (for example, `TOOLSETS=core,config,cluster-diagnostics`)
- `read_only = false` and `disable_destructive = false` in server configuration (`nodes_debug_exec` is hidden when either is enabled)
- ServiceAccount permissions: create/delete pods, exec into pods, schedule on nodes, run privileged pods
- [mcpchecker](https://github.com/mcpchecker/mcpchecker) installed (`make mcpchecker`)

For **Claude Code** evals ([`evals/claude-code/eval.yaml`](../../claude-code/eval.yaml)), also install the ACP agent adapter:

```bash
npm install -g @agentclientprotocol/claude-agent-acp
```

Set Anthropic credentials (for example, `ANTHROPIC_API_KEY`) so `claude-agent-acp` can run.

## Why These Tasks Require `nodes_debug_exec`

Prompts are written so the answer is **not** available from standard Kubernetes API reads alone (for example, `resources_list` on Node objects):

| Task | Why API tools are insufficient |
|------|--------------------------------|
| **check-kubelet-active** | Kubelet **systemd service state** on the host (for example, `active`) is not exposed in Node `status` |
| **list-host-etc** | A directory listing of the node **host filesystem** `/etc` requires executing on the node, not listing cluster resources |

Task-level `assertions.toolsUsed` in each `task.yaml` verifies that `nodes_debug_exec` was called.

## Prompt Design

Prompts describe **what to find**, not **how to find it**. They do not name MCP tools, container images, or implementation details such as `/host` mounts or `chroot`. The agent must discover and use the appropriate cluster diagnostics tooling on its own.

## Cleanup

Each task ships with **`cleanup.sh`**, which deletes ephemeral debug pods labeled `app.kubernetes.io/component=node-debug`. The script is idempotent: it exits successfully when no matching pods exist.

Response checks use `verify.contains` in the task YAML (for example, `"active"` or `"hosts"` in the agent reply).

## Registration

Tasks are discovered automatically when `metadata.labels.suite` is `cluster-diagnostics`. They are registered in:

- [`evals/openai-agent/eval.yaml`](../../openai-agent/eval.yaml)
- [`evals/claude-code/eval.yaml`](../../claude-code/eval.yaml)
- [`evals/gemini-agent/eval.yaml`](../../gemini-agent/eval.yaml)

The eval-level taskSet uses `toolPattern: ".*"` with `minToolCalls: 1` and `maxToolCalls: 20`. Individual tasks pin `nodes_debug_exec` in task-level `assertions.toolsUsed`.

## CI

These tasks are included in the eval configuration files above. The mcpchecker workflow does not yet expose a dedicated `cluster-diagnostics` suite selector; run them locally with the label selector below.

## Running Locally

1. **Configure the server** — `cluster-diagnostics` is not in the default upstream toolsets; enable it explicitly. Because `nodes_debug_exec` is destructive, also disable read-only mode. Example drop-in at `dev/config/mcp-configs/cluster-diagnostics.toml`:

   ```toml
   read_only = false
   disable_destructive = false
   toolsets = ["core", "config", "cluster-diagnostics"]
   ```

   Point the server at your cluster with `kubeconfig = "/path/to/kubeconfig"` in the same file, or set `KUBECONFIG` when starting the server.

2. **Start the MCP server**:

   ```bash
   make run-server TOOLSETS=core,config,cluster-diagnostics
   ```

3. **Run evals** (Claude Code agent example):

   ```bash
   make run-evals \
     EVAL_CONFIG=evals/claude-code/eval.yaml \
     EVAL_LABEL_SELECTOR=suite=cluster-diagnostics

   # Single task:
   make run-evals \
     EVAL_CONFIG=evals/claude-code/eval.yaml \
     EVAL_LABEL_SELECTOR=suite=cluster-diagnostics \
     EVAL_TASK_FILTER=check-kubelet-active
   ```

4. **Stop the server** when finished:

   ```bash
   make stop-server
   ```

## Adding a New Task

1. Create a new subdirectory (for example, `my-scenario/`) with `task.yaml` and optional `setup.sh` and `cleanup.sh`.
2. Set `metadata.labels.suite: cluster-diagnostics` so the task is picked up by the cluster-diagnostics taskSet.
3. Choose a prompt whose answer requires host-level execution—not something already available from Node resources or other read-only MCP tools.
4. Write a goal-only prompt; pin `nodes_debug_exec` in task-level `assertions.toolsUsed`.
5. Provide `cleanup.sh` to remove ephemeral debug pods after each run.

## Tasks

| Task | Difficulty | Prompt | Verification |
|------|------------|--------|--------------|
| **check-kubelet-active** | easy | Check whether the kubelet service is active on the control-plane node and report the result. | Response contains `active` |
| **list-host-etc** | medium | On the control-plane node, list the contents of `/etc` and report what you find. | Response contains `hosts` |
