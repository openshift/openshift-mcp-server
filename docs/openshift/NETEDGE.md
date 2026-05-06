# Network Ingress & DNS Toolset (`netedge`)

This toolset provides a ModelContextProtocol-based troubleshooting framework for OpenShift Network Ingress and DNS (NIDS). It exposes existing NIDS troubleshooting tools (e.g., router logs, DNS pods, ingress controllers, diagnostic metrics) in a structured, model-accessible way to reduce time-to-diagnosis.

It is registered into the openshift-mcp-server as the `netedge` toolset.

## Tools

The NetEdge toolset is divided into four main categories: Diagnostics, Ingress & Routes, DNS & Services, and Connectivity Probes.

### Diagnostics

#### netedge_query_prometheus

Executes specialized diagnostic queries for specific NetEdge components. This is the recommended starting point for investigating high-level health issues.

**Parameters:**
- `diagnostic_target` (string, required) — Run specialized diagnostics for a specific component. Enum: `"ingress"`, `"dns"`, `"operators"`.

**Example:** `diagnostic_target: "ingress"`

---

### Ingress & Routes

#### inspect_route

Inspect an OpenShift Route to view its full configuration and status. Sensitive TLS fields (like private keys and certificates) are automatically redacted for safety.

**Parameters:**
- `namespace` (string, required) — Route namespace.
- `route` (string, required) — Route name.

**Example:**
```yaml
namespace: "openshift-console"
route: "console"
```

#### get_router_config

Retrieve the current router's HAProxy configuration from the cluster. This reads the live configuration directly from the router pod.

**Parameters:**
- `pod` (string, optional) — Router pod name. If omitted, it automatically chooses an existing pod from the default ingress controller.

#### get_router_info

Retrieve HAProxy runtime statistics (uptime, session limits, memory usage) from the router via the admin socket.

**Parameters:**
- `pod` (string, optional) — Router pod name.

#### get_router_sessions

Retrieve all active sessions from the router. Useful for deep traffic analysis.

**Parameters:**
- `pod` (string, optional) — Router pod name.

---

### DNS & Services

#### get_coredns_config

Retrieve the current CoreDNS configuration (`Corefile`) from the `dns-default` ConfigMap in the cluster.

**Parameters:**
- None

#### get_service_endpoints

Return `EndpointSlice` objects for a Service to verify backend pod availability.

**Parameters:**
- `namespace` (string, required) — Service namespace.
- `service` (string, required) — Service name.

#### exec_dns_in_pod

Spin up a temporary, ephemeral pod in the cluster to execute a DNS lookup using `dig`. This verifies internal cluster networking and the DNS resolution path from a pod's perspective. The pod is automatically cleaned up after execution.

**Parameters:**
- `namespace` (string, required) — Namespace to run the ephemeral pod in.
- `target_server` (string, required) — DNS server IP to query (e.g., `"172.30.0.10"`).
- `target_name` (string, required) — DNS name to query (e.g., `"kubernetes.default.svc.cluster.local"`).
- `record_type` (string, optional) — DNS record type (A, AAAA, CNAME, etc.). Defaults to `"A"`.

**Example:**
```yaml
namespace: "default"
target_server: "172.30.0.10"
target_name: "my-service.default.svc.cluster.local"
```

---

### Connectivity Probes

#### probe_dns_local

Run a DNS query using local libraries on the MCP server host to verify connectivity and resolution from the external perspective.

**Parameters:**
- `server` (string, required) — DNS server IP (e.g., `"8.8.8.8"`, `"10.0.0.10"`).
- `name` (string, required) — FQDN to query.
- `type` (string, optional) — Record type. Defaults to `"A"`.

#### probe_http

Send an HTTP(S) request from the MCP server host to verify reachability and inspect the response status code and headers.

**Parameters:**
- `url` (string, required) — The URL to probe (e.g., `"https://example.com/path"`).
- `method` (string, optional) — HTTP method to use. Defaults to `"GET"`.
- `timeout_seconds` (integer, optional) — Request timeout in seconds. Defaults to `5`.

---

## Enable the Toolset

### Command line

```bash
kubernetes-mcp-server --toolsets core,netedge
```

### Configuration file (TOML)

```toml
toolsets = ["core", "netedge"]
```

### MCP client configuration

```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "npx",
      "args": ["-y", "kubernetes-mcp-server@latest", "--toolsets", "core,netedge"]
    }
  }
}
```

---

## Configuration

The toolset can be configured via a `[toolset_configs.netedge]` section in the TOML config file.

```toml
[toolset_configs.netedge]
# Where to read the bearer token from: "header" (default) or "kubeconfig".
# Set to "kubeconfig" when running locally (STDIO mode) so the token is read
# from your kubeconfig session (e.g. after `oc login`).
auth_mode = "kubeconfig"

# Allow insecure TLS connections for diagnostic probes.
# Default: false
insecure = false
```

---

## Authentication and TLS

The toolset authenticates using the OpenShift cluster's credentials. The `auth_mode` option controls where the token is sourced:

- `"header"` (default) — from the incoming request's Authorization header. Use when running remotely behind an OAuth proxy.
- `"kubeconfig"` — from the kubeconfig / in-cluster REST config. Use when running **locally** (STDIO mode).

Diagnostic metrics (via `netedge_query_prometheus`) interact with the in-cluster Thanos Querier and resolve TLS certificates using the system CA or the OpenShift service account CA.

---

## Offline Analysis (Must-Gather)

The NetEdge toolset is designed to support both live-cluster and offline analysis modes. Currently, tools like `get_router_config`, `get_router_info`, and `exec_dns_in_pod` require a live cluster connection because they rely on `exec` commands or dynamic resource creation.

However, an upcoming integration will enable "Virtual Cluster" environments. By pointing the MCP server to a local `must-gather` archive or a remote Prow CI job URL, the agent will be able to debug past failures without live access.

*Note: The exact configuration mechanism for offline mode (e.g., `cluster="file:///path/to/must-gather"`) is under active development and will integrate with the `omc` (OpenShift Must-Gather Client).*

---

## Safety Guardrails

The NetEdge toolset incorporates several safety mechanisms to prevent disruption to the cluster:

1. **TLS Redaction:** The `inspect_route` tool automatically redacts sensitive fields like private keys and certificates from the returned payload.
2. **Ephemeral Resources:** The `exec_dns_in_pod` tool creates a temporary pod to run network diagnostics. This pod is tightly scoped with minimal CPU/Memory limits, drops all capabilities, and is guaranteed to be deleted after the timeout (120 seconds), even if the context is cancelled.
3. **Read-Only Scope:** Router inspection tools (`get_router_config`, `get_router_info`, `get_router_sessions`) access the HAProxy admin socket exclusively in read-only modes to gather diagnostics without mutating state.
