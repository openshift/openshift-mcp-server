# Network Ingress & DNS Toolset (`netedge`)

This toolset provides a ModelContextProtocol-based troubleshooting framework for OpenShift Network Ingress and DNS (NIDS). It exposes existing NIDS troubleshooting tools (e.g., router logs, DNS pods, ingress controllers, diagnostic metrics) in a structured, model-accessible way to reduce time-to-diagnosis.

It is registered into the openshift-mcp-server as the `netedge` toolset.

## Compatibility and deployment requirements

Compatibility filtering checks exact APIs, not deployment readiness. It does not
check named objects, namespaces, platform identity, operators, RBAC, or socket
availability. A visible tool can still fail when its deployment prerequisites are
absent. In particular, core Pod and ConfigMap requirements are not OpenShift
platform gates, and `exec_dns_in_pod` remains usable on generic Kubernetes.

- `get_router_config`, `get_router_info`, and `get_router_sessions` use the fixed
  `openshift-ingress` namespace and `router` container. Without an explicit `pod`,
  they select a Running pod with label
  `ingresscontroller.operator.openshift.io/deployment-ingresscontroller=default`.
  An explicit pod changes only the pod name, not the namespace; a configured
  namespace does not override this nonempty hardcoded namespace. Config reads
  `/var/lib/haproxy/conf/haproxy.config`; runtime info and sessions require
  `socat` and `/var/lib/haproxy/run/haproxy.sock` in that container.
- `get_coredns_config` reads `data.Corefile` from the fixed
  `openshift-dns/dns-default` ConfigMap. It has no namespace or name arguments and
  does not look up a generic `kube-system/coredns` deployment.
- `exec_dns_in_pod` requires explicit `namespace`, `target_server`, and
  `target_name`. It creates a temporary core Pod, runs `dig`, reads its logs, and
  deletes it. Admission, scheduling, image pull access, and runtime support for
  `registry.redhat.io/openshift4/network-tools-rhel9` are required; the image does
  not imply an OpenShift API dependency.

### Compatibility filtering maintenance

Keep each lazy exact-GVK requirement beside its handler. When an API dependency
changes, update that literal, the independent constructor/registration contracts,
and the MCP catalog negative tests together.

| Tools | Exact group/version/kind |
|---|---|
| `query_prometheus` (toolset-name prefix), `inspect_route` | `route.openshift.io/v1`, `Route` |
| `get_coredns_config` | core group `""`, `v1`, `ConfigMap` |
| `get_service_endpoints` | `discovery.k8s.io/v1`, `EndpointSlice` |
| `exec_dns_in_pod`, `get_router_config`, `get_router_info`, `get_router_sessions` | core group `""`, `v1`, `Pod` |
| `probe_dns_local`, `probe_http` | None; both local probes remain ungated |

These are eight gated tools and two ungated local probes. Any enumerated target
supporting the exact GVK is sufficient. Confirmed absence hides only dependent
tools; unknown discovery, 403, and transport errors fail open. Disabled filtering
performs no compatibility discovery. Current ACM inspection enumerates managed
targets; an empty target set hides gated tools, and error aggregation can continue
through later targets. A positive target wins over absence or errors elsewhere.
The discovery REST client has a bounded per-client timeout, not a global deadline
guarantee for aggregate inspection. No provider/watcher policy or ordinary
non-discovery HTTP behavior changes with these filters.

### Constructor compatibility

Go embedders must now call
`InitQueryPrometheus(ctx context.Context, inspector api.ClusterInspector)` instead
of the former no-argument constructor. This is a source-incompatible signature
change. `Toolset.GetTools` supplies the current context and
`ToolsetContext.Inspector`; the compatibility check stays lazy.

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

Retrieve the current router's HAProxy configuration from the cluster. Supports filtering by section type, substring filter on section headers, and line-count limiting via `tail_lines`.

**Parameters:**
- `pod` (string, optional) — Router pod name. If omitted, it automatically chooses an existing pod from the default ingress controller.
- `tail_lines` (integer, optional) — Maximum number of lines to return from the end of the config output. Default: `200`.
- `section` (string, optional) — Filter to a specific HAProxy config section type. Enum: `"global"`, `"defaults"`, `"frontend"`, `"backend"`, `"listen"`.
- `filter` (string, optional) — Substring filter applied to section headers (e.g. a route or backend name). Only sections whose header contains this string are returned.

**Example:**
```yaml
section: "backend"
filter: "my-route"
tail_lines: 100
```

#### get_router_info

Retrieve HAProxy runtime statistics (uptime, session limits, memory usage) from the router via the admin socket.

**Parameters:**
- `pod` (string, optional) — Router pod name.

#### get_router_sessions

Retrieve active sessions from the router. Supports limiting the number of sessions returned and filtering by substring (e.g. backend name or source IP).

**Parameters:**
- `pod` (string, optional) — Router pod name. If omitted, it automatically chooses an existing pod from the default ingress controller.
- `limit` (integer, optional) — Maximum number of session blocks to return. Default: `50`.
- `filter` (string, optional) — Substring filter applied to each session block. Only sessions containing this string are returned (e.g. a backend name or source IP).

**Example:**
```yaml
filter: "be_http"
limit: 10
```

---

### DNS & Services

#### get_coredns_config

Retrieve the current CoreDNS configuration (`Corefile`) from the `dns-default` ConfigMap in the cluster.

**Parameters:**
- None

#### get_service_endpoints

Return `EndpointSlice` objects for a Service to verify backend pod availability. Use `limit` and `continue` to retrieve a large list one page at a time.

**Parameters:**
- `namespace` (string, required) — Service namespace.
- `service` (string, required) — Service name.
- `limit` (integer, optional) — Maximum number of items to return. Omit or set to 0 to list everything, which fails if the response exceeds `max_backend_response_bytes`. Retry with a smaller limit when that happens. The API may return fewer items than requested.
- `continue` (string, optional) — Opaque token from the `continue` field of the previous page. Send it unchanged with the same limit, namespace, and service. Omit on the first call.

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

The netedge toolset is not enabled by default. Enable it in a TOML configuration file:

```toml
toolsets = ["core", "netedge"]
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
