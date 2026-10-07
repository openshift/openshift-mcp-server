# OpenShift Service Mesh (OSSM) Integration

The OpenShift MCP Server exposes OpenShift Service Mesh (OSSM) tools (backed by Kiali) so AI assistants can query mesh topology, health, metrics, distributed traces, and manage Istio configurations.

## Enable the OSSM Toolset

Enable the OSSM tools via the server TOML configuration file.

Config (TOML):

```toml
toolsets = ["core", "ossm"]

[toolset_configs.kiali]
url = "https://kiali.example" # Endpoint/route to reach the Kiali console
# insecure = true  # optional: allow insecure TLS (not recommended in production)
# certificate_authority = "/path/to/ca.crt"  # File path to CA certificate
# When url is https and insecure is false, certificate_authority is required.
```

When the `ossm` toolset is enabled, the toolset configuration is required via `[toolset_configs.kiali]`. If missing or invalid, the server will refuse to start.

## How Authentication Works

- The server uses your existing Kubernetes/OpenShift credentials (from kubeconfig or in-cluster ServiceAccount) to set a bearer token for Kiali/OSSM calls.
- If you pass an HTTP Authorization header to the MCP HTTP endpoint, that is not required for OSSM; OSSM calls use the server's configured token.

## Multi-Cluster Support

OSSM can manage multiple clusters within a service mesh. Most OSSM tools accept an optional `meshCluster` parameter to target a specific mesh cluster. When omitted, OSSM defaults to its home cluster (where Kiali/OSSM is deployed).

Use `ossm_list_mesh_clusters` to discover available mesh cluster names before calling other tools. The `name` field from that response is the only valid value for `meshCluster`.

OSSM tools are not cluster-aware via the MCP server's Kubernetes multi-cluster mechanism: the server does not inject a `context` parameter on them. Use `meshCluster` to select mesh scope. Core Kubernetes tools still use `context` when multi-cluster is enabled.

## Available Tools

| Tool | Description |
| :--- | :--- |
| `ossm_get_mesh_traffic_graph` | Returns service-to-service traffic topology, dependencies, and network metrics (throughput, response time, mTLS) for specified namespaces |
| `ossm_get_mesh_status` | Retrieves high-level health, topology, and environment details of the Istio service mesh |
| `ossm_get_resource_details` | Fetches lists or detailed info for mesh resources (applications, workloads, services) |
| `ossm_get_metrics` | Returns a compact JSON summary of Istio metrics (latency quantiles, traffic trends, throughput) for a given resource |
| `ossm_list_traces` | Lists distributed traces for a service in a namespace |
| `ossm_get_trace_details` | Fetches a single distributed trace by trace ID and returns its call hierarchy |
| `ossm_get_pod_performance` | Returns a human-readable summary comparing Pod CPU/memory usage to requests and limits |
| `ossm_get_logs` | Gets logs of a Kubernetes Pod or workload in a namespace with optional severity filtering |
| `ossm_list_mesh_clusters` | Returns the list of Istio mesh clusters accessible by Kiali/OSSM |
| `ossm_manage_istio_config` | Creates, patches, or deletes Istio, Gateway API, and Inference API configuration objects |
| `ossm_manage_istio_config_read` | Lists or gets Istio, Gateway API, and Inference API configuration objects (read-only) |

## Troubleshooting

- **Missing OSSM configuration when `ossm` toolset is enabled** → Set `[toolset_configs.kiali].url` in the config TOML.
- **Invalid URL** → Ensure `[toolset_configs.kiali].url` is a valid `http(s)://host` URL.
- **TLS certificate validation**:
  - If `[toolset_configs.kiali].url` uses HTTPS and `[toolset_configs.kiali].insecure` is `false`, you must set `[toolset_configs.kiali].certificate_authority` with the path to the CA certificate file. Relative paths are resolved relative to the directory containing the config file.
  - For non-production environments you can set `[toolset_configs.kiali].insecure = true` to skip certificate verification.
