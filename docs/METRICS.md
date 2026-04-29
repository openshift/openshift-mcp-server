# Metrics Toolset (`metrics`)

This toolset provides tools for querying Prometheus/Thanos metrics and Alertmanager alerts.
It is implemented by the [`rhobs/obs-mcp`](https://github.com/rhobs/obs-mcp) package and registered
into the openshift-mcp-server as the `metrics` toolset.

## Tools

### list_metrics

**First step for metric discovery.** Lists available metric names using a regex filter.
Always start here before writing PromQL queries to discover what metrics exist.

**Parameters:**
- `name_regex` (string, required) — Regex pattern to filter metric names (e.g., `"apiserver"`, `"container_cpu"`)

**Example:** `name_regex: "kube_pod"`

---

### execute_instant_query

Execute a PromQL instant query to get point-in-time values.

**Parameters:**
- `query` (string, required) — PromQL query string
- `time` (string, optional) — Evaluation time. Accepts RFC3339 (`2024-01-15T10:00:00Z`), Unix timestamp, or relative (`-5m`, `now`)

**Example:** `query: up{job="apiserver"}`

---

### execute_range_query

Execute a PromQL range query for time-series data.

**Parameters:**
- `query` (string, required) — PromQL query string
- `step` (string, required) — Query resolution step (e.g., `15s`, `1m`, `1h`)
- `start` (string, optional) — Start time (RFC3339, Unix, or relative like `-1h`)
- `end` (string, optional) — End time (RFC3339, Unix, or relative like `now`)
- `duration` (string, optional) — Duration to look back instead of explicit start/end (e.g., `1h`, `30m`)

**Example:**
```
query:    rate(container_cpu_usage_seconds_total[5m])
duration: 1h
step:     1m
```

---

### show_timeseries

Display range query results as an interactive timeseries chart (requires a compatible UI).
Accepts the same parameters as `execute_range_query`, plus:

- `title` (string, optional) — Chart title
- `description` (string, optional) — Chart description

---

### get_label_names

Get all label names (dimensions) available for a metric or across all metrics.

**Parameters:**
- `metric` (string, optional) — Metric name to scope results to
- `start` (string, optional) — Start time (defaults to 1 hour ago)
- `end` (string, optional) — End time (defaults to now)

---

### get_label_values

Get all unique values for a specific label.

**Parameters:**
- `label` (string, required) — Label name to get values for
- `metric` (string, optional) — Metric name to scope results to
- `start` (string, optional) — Start time
- `end` (string, optional) — End time

---

### get_series

Get time series matching a selector and preview cardinality before running expensive queries.

**Parameters:**
- `matches` (string, required) — PromQL series selector (e.g., `container_cpu_usage_seconds_total{namespace="default"}`)
- `start` (string, optional) — Start time
- `end` (string, optional) — End time

---

### get_alerts

Query alerts from Alertmanager. Requires `alertmanager_url` to be configured.

**Parameters:**
- `active` (boolean, optional) — Filter for active alerts only
- `silenced` (boolean, optional) — Filter for silenced alerts
- `inhibited` (boolean, optional) — Filter for inhibited alerts
- `unprocessed` (boolean, optional) — Filter for unprocessed alerts
- `filter` (string, optional) — Label matchers (e.g., `alertname=HighCPU,severity=critical`)
- `receiver` (string, optional) — Receiver name to filter by

---

### get_silences

Query silences from Alertmanager. Requires `alertmanager_url` to be configured.

**Parameters:**
- `filter` (string, optional) — Label matchers to filter silences

---

## Enable the Toolset

### Command line

```bash
kubernetes-mcp-server --toolsets core,metrics
```

### Configuration file (TOML)

```toml
toolsets = ["core", "metrics"]
```

### MCP client configuration

```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "npx",
      "args": ["-y", "kubernetes-mcp-server@latest", "--toolsets", "core,metrics"]
    }
  }
}
```

---

## Configuration

The toolset is configured via a `[metrics]` section in the TOML config file.

```toml
[toolset_configs.metrics]
# Where to read the bearer token from: "header" (default) or "kubeconfig".
# Set to "kubeconfig" when running locally (STDIO mode) so the token is read
# from your kubeconfig session (e.g. after `oc login`).
auth_mode = "kubeconfig"

# URL of the Prometheus or Thanos Querier endpoint.
# Required for metric/query tools. Defaults to http://localhost:9090 if unset.
# Example for OpenShift in-cluster Thanos:
prometheus_url = "https://thanos-querier.openshift-monitoring.svc.cluster.local:9091"
# Example for an external route:
# prometheus_url = "https://thanos-querier-openshift-monitoring.apps.example.com"

# URL of the Alertmanager endpoint.
# Required for get_alerts and get_silences. No default — those tools fail without it.
# Example for OpenShift in-cluster Alertmanager:
alertmanager_url = "https://alertmanager-main.openshift-monitoring.svc.cluster.local:9095"
# Example for an external route:
# alertmanager_url = "https://alertmanager-main-openshift-monitoring.apps.example.com"

# Skip TLS certificate verification.
# Set to true only for development or when using a self-signed cert you can't import.
# Default: false
insecure = false

# Query safety guardrails.
# Valid values:
#   "all"   — enable all guardrails (default)
#   "none"  — disable all guardrails
#   or a comma-separated subset:
#     "disallow-explicit-name-label" — reject queries that filter by __name__ label directly
#     "require-label-matcher"        — reject queries with no label matchers (full table scans)
#     "disallow-blanket-regex"       — reject .* regexes that would match too many series
# Default: "all"
guardrails = "none"

# Maximum number of series allowed per metric before a query is rejected.
# Guards against accidentally pulling millions of series.
# Set to 0 to disable this check.
# Default: 20000
max_metric_cardinality = 0

# Maximum number of label values allowed before a blanket regex (.*) is rejected.
# Only applies when the "disallow-blanket-regex" guardrail is active.
# Set to 0 to always disallow blanket regex regardless of cardinality.
# Default: 500
max_label_cardinality = 0

# Controls whether range queries return full data points
#	instead of summary statistics to the model.
#	Default: false (return summary statistics)
range_query_full_response = false
```

### Configuration reference

| Option | Type | Default | Description |
|---|---|---|---|
| `auth_mode` | string | `"header"` | Token source: `"header"` (from request Authorization header) or `"kubeconfig"` (from kubeconfig/REST config) |
| `prometheus_url` | string | `http://localhost:9090` | Prometheus or Thanos Querier endpoint URL |
| `alertmanager_url` | string | — | Alertmanager endpoint URL (required for alert/silence tools) |
| `insecure` | bool | `false` | Skip TLS certificate verification |
| `guardrails` | string | `"all"` | Query safety checks (`"all"`, `"none"`, or comma-separated list) |
| `max_metric_cardinality` | uint64 | `20000` | Max series per metric (0 = disabled) |
| `max_label_cardinality` | uint64 | `500` | Max label values before blanket regex is rejected (0 = always reject) |
| `range_query_full_response` | bool | `false` | Controls whether range queries return full data points |

---

## Authentication and TLS

The toolset authenticates using a bearer token. The `auth_mode` option controls where it comes from:

- `"header"` (default) — from the incoming request's Authorization header. Use when running remotely behind an OAuth proxy.
- `"kubeconfig"` — from the kubeconfig / in-cluster REST config. Use when running **locally** (STDIO mode, e.g. via `npx` or the binary). Requires a token-based session (e.g. `oc login`) — client-certificate kubeconfigs will get a 401 from Thanos Querier.

**TLS certificate resolution order:**

1. CA certificate embedded in the kubeconfig (`certificate-authority-data`)
2. Service account CA file at `/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt`
   (automatically present when running in a pod on OpenShift)
3. System certificate pool (for publicly trusted certificates)

For endpoints with self-signed or private certificates that are not covered by the above, set
`insecure = true` in the config (not recommended in production).

---

## Endpoint Discovery

`obs-mcp` does not perform automatic OpenShift route discovery. Endpoint URLs must be provided explicitly via `prometheus_url` and `alertmanager_url`.

Typical values for OpenShift:

| Endpoint | In-cluster service URL | External route URL (example) |
|---|---|---|
| Thanos Querier | `https://thanos-querier.openshift-monitoring.svc.cluster.local:9091` | `https://thanos-querier-openshift-monitoring.apps.<cluster>` |
| Alertmanager | `https://alertmanager-main.openshift-monitoring.svc.cluster.local:9095` | `https://alertmanager-main-openshift-monitoring.apps.<cluster>` |

To look up the actual routes in an OpenShift cluster:

```bash
oc get routes -n openshift-monitoring
```

---

## Prerequisites

- **Prometheus or Thanos Querier** accessible at `prometheus_url`
- **Alertmanager** accessible at `alertmanager_url` (only required for `get_alerts` / `get_silences`)
- **Bearer token** with read access to the metrics endpoints (automatically sourced from kubeconfig
  or in-cluster service account)
- For OpenShift in-cluster use, the service account needs:
  ```
  cluster-monitoring-view  (ClusterRole)
  ```

---

## Query Guardrails

Guardrails protect against runaway queries that could overload the metrics backend.
They are enabled by default (`guardrails = "all"`).

| Guardrail | Key | Effect |
|---|---|---|
| Disallow explicit name label | `disallow-explicit-name-label` | Rejects queries that filter on `__name__` directly instead of using metric name syntax |
| Require label matcher | `require-label-matcher` | Rejects bare metric name queries with no label filters (prevents full-cardinality scans) |
| Disallow blanket regex | `disallow-blanket-regex` | Rejects `.*` or equivalently broad regex matchers when the matched series count exceeds `max_label_cardinality` |

To disable a specific guardrail while keeping others:

```toml
[toolset_configs.metrics]
guardrails = "disallow-explicit-name-label,require-label-matcher"  # omit disallow-blanket-regex
```