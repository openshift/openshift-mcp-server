# Tracing Toolset (`observability/traces`)

This toolset provides tools for querying [Grafana Tempo](https://grafana.com/docs/tempo/latest/) using TraceQL and the Tempo HTTP API.
It is implemented by the [`rhobs/obs-mcp`](https://github.com/rhobs/obs-mcp) package and registered into the openshift-mcp-server as the `observability/traces` toolset.

For Prometheus and Alertmanager MCP tools, see the [metrics toolset guide](./metrics.md).
For Grafana Loki and LogQL (`observability/logs` toolset), see the [logs toolset guide](./logs.md).
For OpenTelemetry Collector configuration assistance (`observability/otelcol` toolset), see the [otelcol toolset guide](./otelcol.md).

## Workflow

1. Call **`tempo_list_instances`** first to discover `TempoStack` / `TempoMonolithic` instances, namespaces, multitenancy, and tenant names.
2. Use **`tempo_search_tags`** (and optionally **`tempo_search_tag_values`**) to learn which attributes exist before writing TraceQL.
3. Run **`tempo_search_traces`** with a TraceQL query, then **`tempo_get_trace_by_id`** when you already have a trace ID.

## Tools

### tempo_list_instances

**Discovery entry point.** Lists Tempo instances visible in the Kubernetes API (`TempoStack` and `TempoMonolithic` resources).

**Parameters:** none.

**Output:** JSON per instance includes `kind` (`TempoStack` or `TempoMonolithic`), `tempoNamespace`, `tempoName`, `multitenancy`, `tenants` (when applicable), and `status`. Use `tempoNamespace`, `tempoName`, and `tenant` as parameters on all other tools.

---

### tempo_get_trace_by_id

Fetch one trace by ID (full spans, services, attributes).

**Parameters:**
- `tempoNamespace` (string, required) — Namespace of the Tempo instance
- `tempoName` (string, required) — Instance name from `tempo_list_instances`
- `tenant` (string, optional) — Required when the instance is multi-tenant; must be one of the listed tenants
- `traceid` (string, required) — Trace ID (hex string)
- `start` (string, optional) — RFC3339 start time to narrow the search window
- `end` (string, optional) — RFC3339 end time to narrow the search window

---

### tempo_search_traces

Search traces with [TraceQL](https://grafana.com/docs/tempo/latest/traceql/).

**Parameters:**
- `tempoNamespace`, `tempoName`, `tenant` — same as above
- `query` (string, required) — TraceQL selector, e.g. `{ resource.service.name="checkout" }`. Use `{}` to sample broadly while exploring; status keywords are `status=error` (not quoted).
- `limit` (number, optional) — Max traces to return
- `start`, `end` (string, optional) — RFC3339 window; use `NOW` for current time where supported. Providing both improves coverage versus a default small recent window.
- `spss` (number, optional) — Max matching spans per trace

---

### tempo_search_tags

List tag (attribute) names grouped by scope (`resource`, `span`, `intrinsic`) to help build TraceQL filters.

**Parameters:**
- `tempoNamespace`, `tempoName`, `tenant`
- `scope` (string, optional) — `resource`, `span`, or `intrinsic`
- `query` (string, optional) — TraceQL filter limiting which traces are scanned for tags
- `start`, `end` (string, optional) — RFC3339 time window
- `limit` (number, optional) — Max tag names per scope
- `maxStaleValues` (number, optional) — Early-stop threshold for block scans (higher = more thorough, slower)

---

### tempo_search_tag_values

List known values for one fully qualified tag (e.g. `resource.service.name`).

**Parameters:**
- `tempoNamespace`, `tempoName`, `tenant`
- `tag` (string, required) — Scoped tag name, e.g. `resource.service.name`
- `query`, `start`, `end`, `limit`, `maxStaleValues` — same semantics as `tempo_search_tags`

---

## Enable the Toolset

### Command line

```bash
kubernetes-mcp-server --toolsets core,observability/traces
```

### Configuration file (TOML)

```toml
toolsets = ["core", "observability/traces"]
```

### MCP client configuration

```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "npx",
      "args": ["-y", "kubernetes-mcp-server@latest", "--toolsets", "core,observability/traces"]
    }
  }
}
```

You can enable **`observability/metrics`** and **`observability/traces`** together (same obs-mcp dependency, different toolsets):

```toml
toolsets = ["core", "observability/metrics", "observability/traces"]
```

---

## Configuration

Optional settings use a **`[toolset_configs."observability/traces"]`** section (the key is the toolset name `observability/traces`).

```toml
[toolset_configs."observability/traces"]
# Same semantics as the metrics toolset: "header" (default) or "kubeconfig".
auth_mode = "kubeconfig"

# URL of the Tempo query API endpoint.
# Optional — if unset, use TempoStack/TempoMonolithic discovery
# (tempo_list_instances + tempoNamespace/tempoName on each tool call).
# tempo_url = "https://tempo-query-frontend.observability.svc.cluster.local:3200"

# Skip TLS verification (development only). Default: false
insecure = false

# Resolve Tempo query URLs via OpenShift Routes instead of in-cluster Services.
# Default: false
use_route = false
```

### Configuration reference

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `auth_mode` | string | `"header"` | Bearer token source: `"header"` or `"kubeconfig"` |
| `tempo_url` | string | — | Tempo API endpoint URL (optional; use Tempo discovery if unset) |
| `insecure` | bool | `false` | Skip TLS certificate verification |
| `use_route` | bool | `false` | Use OpenShift `Route` resources for Tempo gateway/query URLs |

---

## Authentication and TLS

Bearer token behavior matches the [metrics toolset](./metrics.md) (**Authentication and TLS** section): `auth_mode` chooses header vs kubeconfig, and TLS uses kubeconfig CA data, OpenShift service CA when in-cluster, then the system trust store. Set `insecure = true` only when you cannot install the correct CA (not recommended in production).

---

## Instance discovery

When `tempo_url` is not set, the server lists **`TempoStack`** and **`TempoMonolithic`** objects cluster-wide and derives query-frontend or gateway base URLs from each resource. With **`use_route = true`**, each instance's URL is resolved via an OpenShift `Route`; instances whose Route cannot be resolved are silently skipped (there is no fallback to service URLs). Chosen instances are **validated** against this discovery list before any request is sent, so callers cannot point tools at arbitrary URLs.

When `tempo_url` is set, that URL is used directly and Kubernetes discovery is skipped. Tool calls still require `tempoNamespace` and `tempoName` in the input schema (they are unused for URL resolution in that case).

---

## Prerequisites

- **Tempo Operator** workloads in the cluster (`TempoStack` and/or `TempoMonolithic` CRs).
- **RBAC** on the MCP identity to **list** `TempoStack` and `TempoMonolithic` objects cluster-wide. If **`use_route`** is enabled, the server also **gets** `Route` resources in each Tempo namespace to resolve external hosts.
- **Bearer token** with permission to reach the resolved Tempo query API (same patterns as the metrics toolset).

---

## Related documentation

- [Metrics toolset guide](./metrics.md) — Prometheus and Alertmanager (`observability/metrics` toolset)
- [OTEL.md](../OTEL.md) — OpenTelemetry export from this MCP server process (not the same as querying Tempo in-cluster)
