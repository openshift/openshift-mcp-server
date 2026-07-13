# Logs Toolset (`observability/logs`)

This toolset provides tools for querying [Grafana Loki](https://grafana.com/oss/loki/) using LogQL and the Loki HTTP API.
It is implemented by the [`rhobs/obs-mcp`](https://github.com/rhobs/obs-mcp) package and registered into the openshift-mcp-server as the `observability/logs` toolset.

For Prometheus and Alertmanager MCP tools, see the [metrics toolset guide](./metrics.md).
For Grafana Tempo and TraceQL (`observability/traces` toolset), see the [tracing toolset guide](./tracing.md).
For OpenTelemetry Collector configuration assistance (`observability/otelcol` toolset), see the [otelcol toolset guide](./otelcol.md).

## Workflow

1. Call **`loki_list_instances`** first to discover `LokiStack` instances, namespaces, multitenancy, and tenant names.
2. Use **`loki_label_names`** (and optionally **`loki_label_values`**) to learn which labels exist before writing LogQL queries.
3. Run **`loki_query_range`** with a LogQL query to retrieve matching log streams and lines.

## Tools

### loki_list_instances

**Discovery entry point.** Lists LokiStack instances visible in the Kubernetes API.

**Parameters:** none.

**Output:** JSON per instance includes `lokiNamespace`, `lokiName`, `status`, and resolved `url`. Use `lokiNamespace`, `lokiName`, and `tenant` as parameters on other Loki tools.

---

### loki_label_names

List available Loki label names for a time range. Use this before writing LogQL queries to discover which labels are indexed.

**Parameters:**
- `lokiNamespace` (string, optional) — Kubernetes namespace of the LokiStack (from `loki_list_instances`)
- `lokiName` (string, optional) — Name of the LokiStack (from `loki_list_instances`)
- `tenant` (string, optional) — Loki tenant ID; for LokiStack gateway modes (e.g. openshift-network) use `network`
- `start` (string, optional) — Start time (RFC3339, Unix timestamp, `NOW`, or relative like `NOW-1h`)
- `end` (string, optional) — End time (RFC3339, Unix timestamp, `NOW`, or relative)

---

### loki_label_values

List possible values for a Loki label key. Use this to build precise label matchers in LogQL.

**Parameters:**
- `label` (string, required) — Label key to inspect (e.g. `namespace`, `pod`, `container`, `SrcK8S_Namespace`)
- `lokiNamespace`, `lokiName`, `tenant`, `start`, `end` — same as `loki_label_names`

---

### loki_query_range

Execute a Loki LogQL range query and return matching log streams and lines.

**Parameters:**
- `query` (string, required) — LogQL query string (e.g. `{namespace="default"}`)
- `lokiNamespace` (string, optional) — Kubernetes namespace of the LokiStack
- `lokiName` (string, optional) — Name of the LokiStack
- `tenant` (string, optional) — Loki tenant ID
- `duration` (string, optional) — Lookback duration from now when start/end are omitted (e.g. `5m`, `1h`). Defaults to `15m`
- `start` (string, optional) — Start time (RFC3339, Unix, `NOW`, or relative)
- `end` (string, optional) — End time (RFC3339, Unix, `NOW`, or relative)
- `limit` (number, optional) — Maximum number of log lines to return. Defaults to 100, max 1000
- `direction` (string, optional) — Search direction: `backward` (default) or `forward`

---

## Enable the Toolset

### Command line

```bash
kubernetes-mcp-server --toolsets core,observability/logs
```

### Configuration file (TOML)

```toml
toolsets = ["core", "observability/logs"]
```

### MCP client configuration

```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "npx",
      "args": ["-y", "kubernetes-mcp-server@latest", "--toolsets", "core,observability/logs"]
    }
  }
}
```

You can enable **`observability/metrics`**, **`observability/traces`**, and **`observability/logs`** together (same obs-mcp dependency, different toolsets):

```toml
toolsets = ["core", "observability/metrics", "observability/traces", "observability/logs"]
```

---

## Configuration

Optional settings use a **`[toolset_configs."observability/logs"]`** section (the key is the toolset name `observability/logs`).

```toml
[toolset_configs."observability/logs"]
# Where to read the bearer token from: "header" (default) or "kubeconfig".
# Set to "kubeconfig" when running locally (STDIO mode) so the token is read
# from your kubeconfig session (e.g. after `oc login`).
auth_mode = "kubeconfig"

# URL of the Loki API endpoint.
# Optional — if unset, use LokiStack discovery (loki_list_instances + lokiNamespace/lokiName).
# Example for a direct Loki endpoint:
# loki_url = "https://logging-loki-gateway-http.openshift-logging.svc.cluster.local:8080"
loki_url = ""

# Skip TLS certificate verification (development only). Default: false
insecure = false

# Resolve Loki query URLs via OpenShift Routes instead of in-cluster Services.
# Default: false
useRoute = false
```

### Configuration reference

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `auth_mode` | string | `"header"` | Bearer token source: `"header"` or `"kubeconfig"` |
| `loki_url` | string | — | Loki API endpoint URL (optional; use LokiStack discovery if unset) |
| `insecure` | bool | `false` | Skip TLS certificate verification |
| `useRoute` | bool | `false` | Use OpenShift `Route` resources for LokiStack gateway URLs |

---

## Authentication and TLS

Bearer token behavior matches the [metrics toolset](./metrics.md) (**Authentication and TLS** section): `auth_mode` chooses header vs kubeconfig, and TLS uses kubeconfig CA data, OpenShift service CA when in-cluster, then the system trust store. Set `insecure = true` only when you cannot install the correct CA (not recommended in production).

### Loki URL resolution

When the `observability/logs` toolset is enabled, the Loki URL is determined in this order:

1. `loki_url` in the `[toolset_configs."observability/logs"]` config section (if set)
2. `LOKI_URL` environment variable
3. Default: `http://localhost:3100` (kubeconfig mode only)

In `header` mode, you can either set `loki_url` **or** use LokiStack discovery (`loki_list_instances` + `lokiNamespace`/`lokiName` arguments on each tool call).

---

## Instance discovery

The server lists **`LokiStack`** objects cluster-wide and derives gateway base URLs from each resource. With **`useRoute = true`**, it prefers OpenShift `Route` hosts where available.

Chosen instances are **validated** against this discovery list before any request is sent, so callers cannot point tools at arbitrary URLs.

---

## Prerequisites

- **Loki Operator** workloads in the cluster (`LokiStack` CRs) or a standalone Loki endpoint.
- **RBAC** on the MCP identity to **list** `LokiStack` objects cluster-wide. If **`useRoute`** is enabled, the server also **gets** `Route` resources in each Loki namespace to resolve external hosts.
- **Bearer token** with permission to reach the resolved Loki API (same patterns as the metrics toolset).

---

## Related documentation

- [Metrics toolset guide](./metrics.md) — Prometheus and Alertmanager (`observability/metrics` toolset)
- [Tracing toolset guide](./tracing.md) — Grafana Tempo and TraceQL (`observability/traces` toolset)
- [OpenTelemetry Collector toolset guide](./otelcol.md) — Component discovery, schemas, config validation (`observability/otelcol` toolset)
- [OTEL.md](../OTEL.md) — OpenTelemetry export from this MCP server process (not the same as querying Loki in-cluster)
