# Must-Gather Toolset (`openshift/mustgather`)

This toolset analyzes OpenShift [must-gather](https://docs.openshift.com/container-platform/latest/support/gathering-cluster-data.html) archives **offline**, without a live cluster connection. It exposes the resources, events, pod logs, node diagnostics, etcd, and monitoring data captured in an archive in a structured, model-accessible way to reduce time-to-diagnosis when triaging past failures.

It is registered into the openshift-mcp-server as the `openshift/mustgather` toolset.

> **Start here:** Call `mustgather_list` first to discover the available archives and their `archive_id`. Every other `mustgather_*` tool requires that `archive_id` (format: `mg-XXXXYYYYYYYY`, e.g. `mg-384226d712f0`).

## Tools

The must-gather toolset is divided into six main categories: Discovery, Resources & Events, Pod Logs, Node Diagnostics, etcd, and Monitoring.

### Discovery

#### mustgather_list

List the must-gather archives discovered under the configured directories. Returns each archive's `archive_id`, which must be passed to the other `mustgather_*` tools.

**Parameters:**
- None

---

### Resources & Events

#### mustgather_resources_list

List Kubernetes resources from the must-gather archive with optional filtering by namespace, labels, and fields.

**Parameters:**
- `archive_id` (string, required) — Must-gather archive ID as returned by `mustgather_list`.
- `kind` (string, required) — Resource kind (e.g., `Pod`, `Deployment`, `Service`).
- `apiVersion` (string, optional) — API version. Default: `v1`.
- `namespace` (string, optional) — Filter by namespace.
- `labelSelector` (string, optional) — Label selector (e.g., `app=nginx,tier=frontend`).
- `fieldSelector` (string, optional) — Field selector (e.g., `metadata.name=foo`).
- `limit` (integer, optional) — Maximum number of resources to return (`0` for all).

#### mustgather_events_list

List Kubernetes events from the must-gather archive with optional filtering by type, namespace, resource, and reason.

**Parameters:**
- `archive_id` (string, required) — Must-gather archive ID as returned by `mustgather_list`.
- `type` (string, optional) — Event type filter: `all`, `Warning`, `Normal`.
- `namespace` (string, optional) — Filter by namespace.
- `resource` (string, optional) — Filter by involved resource name (partial match).
- `reason` (string, optional) — Filter by event reason (partial match).
- `limit` (integer, optional) — Maximum number of events to return. Default: `100`.

#### mustgather_events_by_resource

Get all events related to a specific Kubernetes resource from the must-gather archive.

**Parameters:**
- `archive_id` (string, required) — Must-gather archive ID as returned by `mustgather_list`.
- `name` (string, required) — Resource name.
- `kind` (string, optional) — Resource kind (narrows search).
- `namespace` (string, optional) — Resource namespace.

#### mustgather_events_by_time

List Kubernetes events from the must-gather archive within a specific time range, sorted chronologically.

**Parameters:**
- `archive_id` (string, required) — Must-gather archive ID as returned by `mustgather_list`.
- `since` (string, required) — Start time in RFC3339 format (e.g. `2026-01-15T10:00:00Z`).
- `until` (string, optional) — End time in RFC3339 format (e.g. `2026-01-15T12:00:00Z`).
- `type` (string, optional) — Event type filter: `all`, `Warning`, `Normal`.
- `namespace` (string, optional) — Filter by namespace.
- `limit` (integer, optional) — Maximum number of events to return. Default: `200`.

---

### Pod Logs

#### mustgather_pod_logs_get

Get container logs for a specific pod from the must-gather archive. Returns current or previous logs.

**Parameters:**
- `archive_id` (string, required) — Must-gather archive ID as returned by `mustgather_list`.
- `namespace` (string, required) — Pod namespace.
- `pod` (string, required) — Pod name.
- `container` (string, optional) — Container name (uses first container if not specified).
- `previous` (boolean, optional) — Get previous container logs (from crash/restart).
- `tail` (integer, optional) — Number of lines from end of logs (`0` for all).

#### mustgather_pod_logs_grep

Filter pod container logs by a search string. Returns only matching lines from the must-gather archive.

**Parameters:**
- `archive_id` (string, required) — Must-gather archive ID as returned by `mustgather_list`.
- `namespace` (string, required) — Pod namespace.
- `pod` (string, required) — Pod name.
- `filter` (string, required) — String to search for in log lines.
- `container` (string, optional) — Container name (uses first container if not specified).
- `caseInsensitive` (boolean, optional) — Perform case-insensitive search. Default: `false`.
- `previous` (boolean, optional) — Search previous container logs (from crash/restart).
- `tail` (integer, optional) — Maximum number of matching lines to return (`0` for all).

#### mustgather_pod_logs_by_time

Get pod container logs within a specific time range. Each log line is expected to have an RFC3339Nano timestamp prefix (from `kubectl logs --timestamps`).

**Parameters:**
- `archive_id` (string, required) — Must-gather archive ID as returned by `mustgather_list`.
- `namespace` (string, required) — Pod namespace.
- `pod` (string, required) — Pod name.
- `since` (string, required) — Start time in RFC3339 format (e.g. `2026-01-15T10:00:00Z`).
- `until` (string, optional) — End time in RFC3339 format (e.g. `2026-01-15T12:00:00Z`).
- `container` (string, optional) — Container name (uses first container if not specified).
- `previous` (boolean, optional) — Search previous container logs (from crash/restart).
- `limit` (integer, optional) — Maximum number of lines to return. Default: `500`.

---

### Node Diagnostics

#### mustgather_node_diagnostics_get

Get comprehensive diagnostic information for a specific node including kubelet logs, system info, CPU/IRQ affinities, and hardware details.

**Parameters:**
- `archive_id` (string, required) — Must-gather archive ID as returned by `mustgather_list`.
- `node` (string, required) — Node name.
- `include` (string, optional) — Comma-separated diagnostics to include: `kubelet,sysinfo,cpu,irq,pods,podresources,lscpu,lspci,dmesg,cmdline`. Default: all.
- `kubeletTail` (integer, optional) — Number of lines from end of kubelet log (`0` for all). Default: `100`.

#### mustgather_node_kubelet_logs

Get kubelet logs for a specific node (decompressed from `.gz` file).

**Parameters:**
- `archive_id` (string, required) — Must-gather archive ID as returned by `mustgather_list`.
- `node` (string, required) — Node name.
- `tail` (integer, optional) — Number of lines from end (`0` for all).

#### mustgather_node_kubelet_logs_grep

Filter kubelet logs for a specific node by a search string. Returns only matching lines.

**Parameters:**
- `archive_id` (string, required) — Must-gather archive ID as returned by `mustgather_list`.
- `node` (string, required) — Node name.
- `filter` (string, required) — String to search for in log lines.
- `caseInsensitive` (boolean, optional) — Perform case-insensitive search. Default: `false`.
- `tail` (integer, optional) — Maximum number of matching lines to return (`0` for all).

---

### etcd

#### mustgather_etcd_health

Get ETCD cluster health status including endpoint health and active alarms from the must-gather archive.

**Parameters:**
- `archive_id` (string, required) — Must-gather archive ID as returned by `mustgather_list`.

#### mustgather_etcd_object_count

Get ETCD object counts by resource type from the must-gather archive.

**Parameters:**
- `archive_id` (string, required) — Must-gather archive ID as returned by `mustgather_list`.
- `limit` (integer, optional) — Maximum number of resource types to show, sorted by count descending. Default: `50`.

---

### Monitoring

#### mustgather_monitoring_prometheus_status

Get Prometheus TSDB and runtime status from the must-gather archive.

**Parameters:**
- `archive_id` (string, required) — Must-gather archive ID as returned by `mustgather_list`.
- `replica` (string, optional) — Prometheus replica (`0`, `1`, or `all`). Default: `all`.

#### mustgather_monitoring_prometheus_targets

Get Prometheus scrape targets and their health status from the must-gather archive.

**Parameters:**
- `archive_id` (string, required) — Must-gather archive ID as returned by `mustgather_list`.
- `health` (string, optional) — Filter by health status: `up`, `down`, `unknown`. Default: all.
- `replica` (string, optional) — Prometheus replica (`0`, `1`, or `all`). Default: `0`.

#### mustgather_monitoring_prometheus_tsdb

Get detailed Prometheus TSDB statistics including top metrics by series count and label cardinality.

**Parameters:**
- `archive_id` (string, required) — Must-gather archive ID as returned by `mustgather_list`.
- `limit` (integer, optional) — Number of top entries to show per category. Default: `10`.
- `replica` (string, optional) — Prometheus replica (`0`, `1`, or `all`). Default: `0`.

#### mustgather_monitoring_prometheus_alerts

Get active Prometheus alerts from the must-gather archive.

**Parameters:**
- `archive_id` (string, required) — Must-gather archive ID as returned by `mustgather_list`.
- `state` (string, optional) — Filter by alert state: `firing`, `pending`. Default: all.

#### mustgather_monitoring_prometheus_rules

Get Prometheus alerting and recording rules from the must-gather archive.

**Parameters:**
- `archive_id` (string, required) — Must-gather archive ID as returned by `mustgather_list`.
- `type` (string, optional) — Filter by rule type: `alerting`, `recording`. Default: all.

---

## Prompts

#### plan_mustgather

Plan for collecting a must-gather archive from an OpenShift cluster. Must-gather is a tool for collecting cluster data related to debugging and troubleshooting like logs, Kubernetes resources, etc.

**Arguments (all optional):** `node_name`, `node_selector`, `source_dir`, `namespace`, `gather_command`, `timeout`, `since`, `host_network`, `keep_resources`, `all_component_images`, `images`.

---

## Enable the Toolset

The must-gather toolset is not enabled by default. Enable it in a TOML configuration file:

```toml
toolsets = ["core", "openshift/mustgather"]
```

```bash
kubernetes-mcp-server --config /path/to/config.toml
```

### MCP client configuration

```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "npx",
      "args": ["-y", "kubernetes-mcp-server@latest", "--config", "/path/to/config.toml"]
    }
  }
}
```

---

## Configuration

The toolset is configured via a `[toolset_configs."openshift/mustgather"]` section in the TOML config file. Point it at one or more directories that hold your must-gather archives.

```toml
[toolset_configs."openshift/mustgather"]
# Directories scanned for must-gather archives. Each entry may be a directory
# containing one or more archives as sub-directories, or a directory that is
# itself an archive.
mustgather_dirs = ["/var/data/must-gather", "/home/user/downloads/must-gather.local.123"]

# Cap the number of lines a log tool keeps when a caller requests tailing.
# When unset (0), defaults to 1000.
tail_limit = 1000

# Cap the aggregate size, in bytes, of assembled log output returned by a
# single tool call. When unset (0), defaults to 1 MiB (~200K tokens).
max_output_size = 1048576
```

| Option | Type | Description |
|--------|------|-------------|
| `mustgather_dirs` | string array | Directories the toolset scans for must-gather archives. Each entry may be a directory containing archives as sub-directories, or a directory that is itself an archive. Archives are addressed by the stable `archive_id` returned by `mustgather_list`. |
| `tail_limit` | integer | Caps the number of lines a log tool keeps when a caller requests tailing. A tool-provided `tail` larger than this is capped to this value. Default: `1000`. |
| `max_output_size` | integer | Caps the aggregate size, in bytes, of assembled log output returned by a single tool call. Default: `1048576` (1 MiB). |

Archives are cached per parsed configuration. A server reload (`SIGHUP`) that yields a new configuration starts from a clean cache instead of carrying stale entries.

---

## Offline Analysis

Unlike live-cluster toolsets, `openshift/mustgather` requires **no cluster connection**. All data is read from the on-disk archives configured in `mustgather_dirs`, making it suitable for debugging past failures, CI job artifacts, or archives shared by others.
