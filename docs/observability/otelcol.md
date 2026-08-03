# OpenTelemetry Collector Toolset (`observability/otelcol`)

This toolset provides tools for OpenTelemetry Collector configuration assistance: listing
components, fetching JSON schemas, validating component configs, and listing supported versions.
It is implemented by the [`rhobs/obs-mcp`](https://github.com/rhobs/obs-mcp) package and
registered into the openshift-mcp-server as the `observability/otelcol` toolset.

Component schemas are embedded in the binary (via `redhat-opentelemetry-collector`); no
running Collector instance or cluster endpoint is required.

For Prometheus and Alertmanager MCP tools, see the [metrics toolset guide](./metrics.md).
For Grafana Loki and LogQL (`observability/logs` toolset), see the [logs toolset guide](./logs.md).
For Grafana Tempo and TraceQL, see the [tracing toolset guide](./tracing.md).

## Workflow

1. Call **`otelcol_get_versions`** to see supported Collector versions and the latest default.
2. Call **`otelcol_list_components`** for the target version to discover receivers, processors,
   exporters, extensions, and connectors — do not guess component names.
3. Call **`otelcol_get_component_schema`** to inspect configuration fields before writing YAML.
4. Call **`otelcol_validate_config`** to validate a component config against its JSON schema.

If the user specifies a version, use it consistently across all tool calls.

## Tools

### otelcol_get_versions

**Discovery entry point.** Lists available OpenTelemetry Collector versions and identifies the latest.

**Parameters:** none.

**Output:** JSON includes `versions` (array) and `latest_version` (string).

---

### otelcol_list_components

List available Collector components for a given version.

**Parameters:**
- `version` (string, optional) — Collector version (e.g., `v0.144.0` or `0.144.0`). Defaults to latest.

**Output:** JSON includes `version` and component name lists grouped by type (`receivers`, `processors`, `exporters`, `extensions`, `connectors`).

---

### otelcol_get_component_schema

Get the JSON schema for a component's configuration options.

**Parameters:**
- `component_type` (string, required) — One of: `receiver`, `processor`, `exporter`, `extension`, `connector`
- `component_name` (string, required) — Component name from `otelcol_list_components` (e.g., `otlp`, `batch`, `debug`)
- `version` (string, optional) — Collector version. Defaults to latest.

**Output:** JSON includes `type`, `name`, `version`, and `schema` (object).

---

### otelcol_validate_config

Validate a component configuration against its JSON schema.

**Parameters:**
- `component_type` (string, required) — One of: `receiver`, `processor`, `exporter`, `extension`, `connector`
- `component_name` (string, required) — Component name from `otelcol_list_components`
- `config` (string, required) — Configuration as YAML or JSON string
- `format` (string, optional) — `yaml` (default) or `json`
- `version` (string, optional) — Collector version. Defaults to latest.

**Output:** JSON includes `valid` (boolean), `version`, and `errors` (array when invalid).

---

## Enable the Toolset

### Command line

```bash
kubernetes-mcp-server --toolsets core,observability/otelcol
```

### Configuration file (TOML)

```toml
toolsets = ["core", "observability/otelcol"]
```

### MCP client configuration

```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "npx",
      "args": ["-y", "kubernetes-mcp-server@latest", "--toolsets", "core,observability/otelcol"]
    }
  }
}
```

You can enable **`observability/metrics`**, **`observability/traces`**, and **`observability/otelcol`** together (same obs-mcp dependency, different toolsets):

```toml
toolsets = ["core", "observability/metrics", "observability/traces", "observability/otelcol"]
```

---

## Configuration

The toolset works out of the box with embedded schemas. Advanced deployments may use a
**`[toolset_configs."observability/otelcol"]`** section; the only configurable field is `SchemaFS`, which is
normally set programmatically and not required for standard use.

No Prometheus, Tempo, or Collector endpoint URLs are needed.

---

## Prerequisites

- **None for schema tools** — schemas ship inside the MCP server binary.
- **No cluster RBAC** — unlike `observability/metrics` and `observability/traces`, this toolset does not call the Kubernetes API.

---

## Related documentation

- [Metrics toolset guide](./metrics.md) — Prometheus and Alertmanager (`observability/metrics` toolset)
- [Logs toolset guide](./logs.md) — Grafana Loki and LogQL (`observability/logs` toolset)
- [Tracing toolset guide](./tracing.md) — Grafana Tempo and TraceQL (`observability/traces` toolset)
- [OTEL.md](../OTEL.md) — OpenTelemetry export from this MCP server process (not the same as Collector config assistance)
