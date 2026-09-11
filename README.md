# MCP server for Red Hat OpenShift

[![GitHub License](https://img.shields.io/github/license/openshift/openshift-mcp-server)](https://github.com/openshift/openshift-mcp-server/blob/main/LICENSE)
[![Build](https://github.com/openshift/openshift-mcp-server/actions/workflows/build.yaml/badge.svg)](https://github.com/openshift/openshift-mcp-server/actions/workflows/build.yaml)

> **Developer Preview** — The MCP server for Red Hat OpenShift is in a pre-release stage, intended for developers and early adopters to test and provide feedback. It is not recommended for production use due to potential limitations and instability.

[✨ Features](#features) | [🚀 Getting Started](#getting-started) | [⚙️ Configuration](#configuration) | [🛠️ Tools](#tools-and-functionalities) | [🔒 Safety and Best Practices](#safety-and-best-practices) | [📚 Documentation](#documentation) | [🧑‍💻 Development](#development)

---

## ✨ Features <a id="features"></a>

The **MCP server for Red Hat OpenShift** is a downstream fork of [containers/kubernetes-mcp-server](https://github.com/containers/kubernetes-mcp-server), enhanced with OpenShift-specific enterprise capabilities for cluster operations.

It is a **native Go-based [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server** — not an `oc` or `kubectl` wrapper — that communicates directly with the Kubernetes/OpenShift API server.

### Enterprise features (OpenShift-specific)

1. OAuth & OIDC integration for Token Exchange support with Keycloak
2. Modular Toolset Architecture with Read-Only defaults
3. Full Observability stack with OpenTelemetry (Prometheus/Thanos, Loki, Tempo, OTel Collector)
4. Advanced Cluster Management (ACM) integration and MCP gateway support

### OpenShift toolsets (added in this fork)

| Toolset | Description |
| :--- | :--- |
| `openshift` | OpenShift-specific tools: Projects, ClusterOperators, and the `plan_mustgather` prompt |
| `openshift/mustgather` | Analyze OpenShift must-gather archives offline without a live cluster connection |
| `cluster-diagnostics` | Node debug execution, cluster health checks, resource diagnostics |
| `kubevirt` | OpenShift Virtualization: create, manage, troubleshoot VirtualMachines |
| `kiali` | OpenShift Service Mesh topology and observability via Kiali |
| `cni-diagnostics` | Kernel-level CNI diagnostics: conntrack, iptables, nftables, ip, tcpdump, eBPF (pwru) |
| `ovn-kubernetes` | OVN/OVS network inspection: logical flows, packet trace, OVN-NB/SB databases (read-only) |
| `netobserv` | Network observability via the NetObserv console plugin API (flows, metrics, CSV export) |
| `netedge` | Ingress and DNS diagnostics: HAProxy router config, Route inspection, CoreDNS, probes |
| `oadp` | OpenShift API for Data Protection (OADP): Velero backup and restore troubleshooting |

### Toolsets inherited from upstream

| Toolset | Description |
| :--- | :--- |
| `core` | Pods, Generic Resources, Events, Namespaces, Nodes — enabled by default, read-only |
| `config` | View and manage kubeconfig / in-cluster configuration — enabled by default |
| `helm` | Helm chart install, list, uninstall |
| `tekton` | Tekton Pipeline and Task run management and log retrieval |
| `kcp` | KCP workspace and multi-tenancy management |
| `observability/metrics` | Prometheus/Thanos instant and range queries, Alertmanager alerts and silences |
| `observability/logs` | Grafana Loki / LokiStack log queries (LogQL) |
| `observability/traces` | Grafana Tempo trace search and retrieval (TraceQL) |
| `observability/otelcol` | OpenTelemetry Collector configuration schema validation and component documentation |

---

## 🚀 Getting Started <a id="getting-started"></a>

### Requirements

- Access to a **Red Hat OpenShift** or Kubernetes cluster
- `kubeconfig` file (typically `~/.kube/config`) or in-cluster service account
- Node.js 14+ if using `npx`; no external dependencies for native binaries

### Installation

**npm (Recommended)**

```bash
npx -y openshift-mcp-server@latest \
  --toolsets core,openshift,cluster-diagnostics,observability/metrics,observability/logs
```

**Native Binary**

Download from [GitHub Releases](https://github.com/openshift/openshift-mcp-server/releases/latest):

```bash
wget https://github.com/openshift/openshift-mcp-server/releases/latest/download/openshift-mcp-server-linux-x86_64
chmod +x openshift-mcp-server-linux-x86_64
./openshift-mcp-server-linux-x86_64 --toolsets core,openshift
```

**Container / Podman**

```bash
podman run -v ~/.kube/config:/kubeconfig:ro \
  -e KUBECONFIG=/kubeconfig \
  ghcr.io/openshift/openshift-mcp-server:latest
```

**Helm (in-cluster deployment)**

```bash
helm install openshift-mcp-server charts/openshift-mcp-server \
  --namespace openshift-mcp-server --create-namespace
```

### Client Configuration

**Cursor IDE** — edit `~/.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "openshift-mcp-server": {
      "command": "npx",
      "args": [
        "-y", "openshift-mcp-server@latest",
        "--toolsets", "core,openshift,cluster-diagnostics,kubevirt,observability/metrics,observability/logs,oadp"
      ],
      "env": {
        "KUBECONFIG": "~/.kube/config"
      }
    }
  }
}
```

**Claude Desktop** — edit `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "openshift": {
      "command": "npx",
      "args": ["-y", "openshift-mcp-server@latest"],
      "env": {
        "KUBECONFIG": "~/.kube/config"
      }
    }
  }
}
```

---

## ⚙️ Configuration <a id="configuration"></a>

### CLI Options

| Option | Description |
| :--- | :--- |
| `--toolsets` | Comma-separated list of toolsets to enable |
| `--kubeconfig` | Path to kubeconfig file (default: `~/.kube/config`) |
| `--read-only` | Disable all write operations (create, update, delete) |
| `--disable-destructive` | Disable delete and update operations; allow create |
| `--disable-multi-cluster` | Restrict to current kubeconfig context only |
| `--cluster-provider` | One of: `kubeconfig`, `in-cluster`, `kcp`, `disabled` |
| `--log-level` | Logging verbosity 0–9 (kubectl-style) |
| `--port` | Enable Streamable HTTP mode on specified port |
| `--stateless` | Disable tool/prompt change notifications (use for load-balanced deployments) |
| `--config` | Path to TOML configuration file |

### TOML Configuration File

```toml
# Enable toolsets
toolsets = ["core", "openshift", "cluster-diagnostics", "kubevirt", "observability/metrics", "observability/logs", "oadp"]

# Restrict specific Kubernetes resources from being accessed
[[denied_resources]]
group = ""
version = "v1"
kind = "Secret"

[[denied_resources]]
group = "rbac.authorization.k8s.io"
version = "v1"
kind = "ClusterRoleBinding"

# Filter specific tools
# enabled_tools = ["pods_list", "pods_get", "pods_log"]
# disabled_tools = ["resources_delete", "pods_delete"]

# Kiali / OSSM configuration
[toolset_configs.kiali]
url = "https://kiali.apps.mycluster.example.com"

# NetObserv configuration (optional; defaults to in-cluster service URL on OpenShift)
[toolset_configs.netobserv]
# url = "https://netobserv-plugin.netobserv.svc.cluster.local:9001"
```

See [Configuration Reference](docs/configuration.md) for full details including OIDC/Keycloak, ACM multi-cluster, TLS, and `tool_overrides`.

---

## 🛠️ Tools and Functionalities <a id="tools-and-functionalities"></a>

By default the MCP server for Red Hat OpenShift enables only `core` and `config` toolsets in read-only mode. To enable additional toolsets such as `kubevirt`, `kiali`, or `oadp`, update your `config.toml`:

```toml
toolsets = ["core", "openshift", "kubevirt"]
```

### Core

#### Pods

| Tool | Description |
| :--- | :--- |
| `pods_list` | List all pods in the cluster from all namespaces with optional label and field selectors |
| `pods_list_in_namespace` | List all pods in a specified namespace with optional label and field selectors |
| `pods_get` | Get a specific pod by name in the current or provided namespace |
| `pods_delete` | Delete a pod by name in the current or provided namespace |
| `pods_top` | List resource consumption (CPU and memory) for pods via the Metrics Server |
| `pods_exec` | Execute a command in a pod |
| `pods_log` | Get the logs of a pod with options for container selection, tail lines, and previous container logs |
| `pods_run` | Run a pod in a specified namespace with a container image and optional name and port exposure |

#### Generic Resources

| Tool | Description |
| :--- | :--- |
| `resources_list` | List Kubernetes resources by apiVersion and kind with optional namespace and selectors |
| `resources_get` | Get a specific resource by apiVersion, kind, name, and optional namespace |
| `resources_create_or_update` | Create or update a resource from a YAML or JSON representation (not enabled by default) |
| `resources_delete` | Delete a resource by apiVersion, kind, name, and optional namespace (not enabled by default) |
| `resources_scale` | Get or update the scale of a resource (e.g., Deployment, StatefulSet) (not enabled by default) |

#### Nodes

| Tool | Description |
| :--- | :--- |
| `nodes_log` | Get logs from a Kubernetes node through the API proxy to the kubelet |
| `nodes_stats_summary` | Get detailed resource usage statistics from a node via the kubelet Summary API |
| `nodes_top` | List resource consumption (CPU and memory) for nodes via the Metrics Server |
| `nodes_debug_exec` | Run commands on a node using a privileged ephemeral debug pod |

#### Namespaces and Events

| Tool | Description |
| :--- | :--- |
| `namespaces_list` | List all Kubernetes namespaces in the current cluster |
| `projects_list` | List all OpenShift projects in the current cluster (OpenShift-only) |
| `events_list` | List Kubernetes events (warnings, errors, state changes) for debugging and troubleshooting |

### OpenShift

#### Prompts

| Prompt | Description |
| :--- | :--- |
| `plan_mustgather` | Generate YAML manifests for collecting a must-gather archive from an OpenShift cluster |

The `plan_mustgather` prompt supports: custom node/node-selector targeting, audit log collection, custom timeout and `--since` filtering, multi-image operator-specific gathering, and host-network access for advanced gather scripts.

### Must-Gather Offline Analysis (`openshift/mustgather`)

Analyze OpenShift must-gather archives without a live cluster connection. Supports inspecting resources, events, pod logs, node state, etcd, and monitoring data from the archive.

### KubeVirt / OpenShift Virtualization

| Tool | Description |
| :--- | :--- |
| `vm_create` | Create a VirtualMachine with automatic resolution of instance types, preferences, and container disk images |
| `vm_lifecycle` | Start, stop, or restart a VirtualMachine |
| `vm_clone` | Clone an existing VirtualMachine using the KubeVirt Clone API |

| Prompt | Description |
| :--- | :--- |
| `vm-troubleshoot` | Step-by-step troubleshooting for VirtualMachine issues: storage, cloud-init, nodeSelector, migration, crashloops |

See [KubeVirt documentation](docs/kubevirt.md) for supported OS images, instance type hints, and networking (Multus NADs).

### Kiali / OSSM

| Tool | Description |
| :--- | :--- |
| `kiali_mesh_graph` | Namespace topology with health summary for apps, workloads, and services in the mesh |
| `kiali_get_resource_details` | List or get Kubernetes resources (services, workloads) within the service mesh |
| `kiali_get_metrics` | Metrics for a specific resource with configurable duration, step, rate interval, direction |
| `kiali_get_traces` | Distributed traces for a resource, or trace details by trace ID |
| `kiali_workload_logs` | Workload pod logs with automatic pod and container discovery |
| `kiali_manage_istio_config` | Create, patch, or delete Istio configuration objects (Gateways, VirtualServices, DestinationRules) |
| `kiali_manage_istio_config_read` | List or get Istio configuration objects in read-only mode |
| `kiali_list_mesh_clusters` | List clusters managed within the Istio mesh |

### CNI Diagnostics

| Tool | Description |
| :--- | :--- |
| `get-conntrack` | Connection tracking (conntrack) entries from a node |
| `get-iptables` | IPv4/IPv6 packet filter rules (iptables/ip6tables) |
| `get-nft` | NFtables packet filtering and classification rules |
| `get-ip` | IP routing, interfaces, neighbours, and network namespaces (iproute2) |
| `tcpdump` | Packet capture on nodes or inside pod network namespaces |
| `pwru` | eBPF-based kernel packet tracing (`packet, where are you?`) — requires Linux kernel 4.18+ |

### OVN-Kubernetes

All tools in this toolset are read-only.

| Tool | Description |
| :--- | :--- |
| `ovn_show` | OVN configuration overview via `ovn-nbctl show` / `ovn-sbctl show` |
| `ovn_get` | Query records from OVN Northbound or Southbound database tables |
| `ovn_lflow_list` | List logical flows from the OVN Southbound database |
| `ovn_trace` | Trace a packet through the OVN logical network |
| `ovs_vsctl` | OVS switch configuration: bridges, ports, interfaces |
| `ovs_ofctl` | OpenFlow table inspection |
| `ovs_appctl` | OVS datapath and pipeline diagnostics |

### NetObserv

| Tool | Description |
| :--- | :--- |
| `netobserv_list_flows` | Flow records from Loki via the NetObserv plugin API |
| `netobserv_get_flow_metrics` | Aggregated network flow metrics |
| `netobserv_export_flows` | Export network flows as CSV |

### NetEdge (Ingress & DNS)

| Tool | Description |
| :--- | :--- |
| `netedge_query_prometheus` | Specialized diagnostic queries for `ingress`, `dns`, or `operators` components |
| `inspect_route` | Inspect an OpenShift Route configuration and status (TLS fields redacted) |
| `get_router_config` | Retrieve HAProxy router configuration with optional section and substring filtering |
| `get_coredns_config` | Retrieve the CoreDNS configuration (Corefile) from the `openshift-dns` namespace |

### OADP / Velero

| Prompt | Description |
| :--- | :--- |
| `oadp-troubleshoot` | Diagnose backup and restore issues: DPA status, BSL health, Velero pod health, pod logs, events |

OADP resources (backups, restores, schedules, CRDs) are also accessible via the `core` toolset generic resource tools.

### Observability

Observability query tools are provided by [rhobs/obs-mcp](https://github.com/rhobs/obs-mcp) and registered as sub-toolsets. Enable them explicitly.

| Tool | Toolset | Description |
| :--- | :--- | :--- |
| `list_metrics` | `observability/metrics` | List available metric names using a regex filter (start here before writing PromQL) |
| `execute_instant_query` | `observability/metrics` | Execute a PromQL instant query against Prometheus/Thanos Querier |
| `execute_range_query` | `observability/metrics` | Execute a PromQL range query returning time-series data |
| `get_alerts` | `observability/metrics` | Query alerts from Alertmanager with state filtering |
| `get_silences` | `observability/metrics` | Query silences from Alertmanager with label matcher filtering |
| `loki_list_instances` | `observability/logs` | Discover LokiStack instances, namespaces, and tenant names |
| `loki_label_names` | `observability/logs` | List available Loki label names for a time range |
| `loki_query_range` | `observability/logs` | Execute a LogQL range query and return matching log streams and lines |
| `tempo_search_traces` | `observability/traces` | Search distributed traces in Tempo using TraceQL |

See the [metrics](docs/observability/metrics.md), [logs](docs/observability/logs.md), [tracing](docs/observability/tracing.md), and [otelcol](docs/observability/otelcol.md) guides for full details.

---

## 🔒 Safety and Best Practices <a id="safety-and-best-practices"></a>

### Read-Only and Access Control

By default, the server ships with `core` and `config` toolsets in **read-only mode**. No write operations (create, update, delete) are permitted unless explicitly enabled.

```toml
# Production-safe: read-only
read_only = true

# Allow writes but prevent delete/update
disable_destructive = true
```

### Human-in-the-Loop (HITL)

It is strongly recommended that users review the agent's proposed actions in the MCP client before applying them to the cluster. For "write" actions, configure a human approval mechanism in your MCP client interface.

- Manually verify suggested resource versions (avoid deprecated APIs)
- Use read-only mode when auditing or investigating

### Access Revocation

To revoke or limit access:

1. **Remove specific tools** via `config.toml`:
   ```toml
   disabled_tools = ["resources_delete", "pods_delete"]
   ```
2. **Deny specific resource types**:
   ```toml
   [[denied_resources]]
   group = ""
   version = "v1"
   kind = "Secret"
   ```
3. **Uninstall the server**: `helm uninstall openshift-mcp-server`
4. **Revoke RBAC**: Delete the user's RoleBinding/ClusterRoleBinding
5. **Remove MCP gateway registration**: `oc delete mcpsr <name>`

### MCP Gateway (Recommended for Production)

Route all traffic through the [MCP gateway](https://docs.kuadrant.io/1.4.x/mcp-gateway/docs/guides/register-mcp-servers/) to apply security guardrails, authorization, and audit controls.

### Data Ownership and Telemetry

The MCP server for Red Hat OpenShift does not store any cluster information or state. If telemetry is opted in, the following aggregate-only metrics are collected:

- `Cluster:k8s_mcp_tool_calls:sum` — total tool invocations
- `Cluster:k8s_mcp_tool_errors:sum` — total failed tool invocations
- `Cluster:k8s_mcp_http_requests:sum` — total HTTP requests to the server

These metrics do not capture the content of requests, errors, or cluster data — only aggregate counts.

### Sensitive Data Redaction

The server automatically redacts tokens, keys, passwords, and credentials from MCP logging output.

---

## 📚 Documentation <a id="documentation"></a>

| Guide | Description |
| :--- | :--- |
| [User Guide](docs/openshift/user-guide.md) | Full tool tables, safety guidelines, and model evaluation results |
| [OpenShift Toolset](docs/OPENSHIFT.md) | `plan_mustgather` prompt reference |
| [KubeVirt / OpenShift Virtualization](docs/kubevirt.md) | VM creation, lifecycle, troubleshooting |
| [OVN-Kubernetes Toolset](docs/openshift/ovn-kubernetes.md) | OVN/OVS network inspection tools |
| [CNI Diagnostics Toolset](docs/openshift/cni-diagnostics.md) | Kernel networking and eBPF tools |
| [NetObserv Integration](docs/NETOBSERV.md) | Network flow observability |
| [NetEdge Toolset](docs/openshift/NETEDGE.md) | Ingress, Routes, DNS diagnostics |
| [OSSM / Kiali](docs/KIALI.md) | Service Mesh topology and observability |
| [OADP Support](docs/OADP.md) | Velero backup and restore |
| [ACM Setup](docs/openshift/acm_setup.md) | Advanced Cluster Management multi-cluster |
| [ACM with Keycloak](docs/openshift/acm_keycloak_setup.md) | OIDC authentication for ACM |
| [Keycloak OIDC Setup](docs/KEYCLOAK_OIDC_SETUP.md) | Token Exchange with Keycloak |
| [Entra ID Setup](docs/ENTRA_ID_SETUP.md) | Token Exchange with Microsoft Entra ID |
| [Observability: Metrics](docs/observability/metrics.md) | Prometheus/Thanos/Alertmanager tools |
| [Observability: Logs](docs/observability/logs.md) | Loki / LokiStack (LogQL) |
| [Observability: Traces](docs/observability/tracing.md) | Tempo (TraceQL) |
| [Observability: OTel Collector](docs/observability/otelcol.md) | OpenTelemetry Collector config |
| [Configuration Reference](docs/configuration.md) | Full TOML config: RBAC, OIDC, denied resources, tool overrides |

---

## 🧑‍💻 Development <a id="development"></a>

### Prerequisites

- [Go](https://go.dev/dl/) (version specified in `go.mod`)

### Build

```bash
make build
```

### Test

```bash
make test
```

The test suite uses `setup-envtest` from `sigs.k8s.io/controller-runtime`. No real cluster is required; the first run downloads the envtest binaries.

### Run with MCP Inspector

```bash
make build
npx @modelcontextprotocol/inspector@latest $(pwd)/openshift-mcp-server
```

### Local Dev Environment

```bash
make local-env-setup    # Starts a local Minikube cluster with required components
make local-env-teardown
```

See [CONTRIBUTING.md](CONTRIBUTING.md) and [AGENTS.md](AGENTS.md) for contribution guidelines, toolset design patterns, and testing conventions.

---

## Troubleshooting and Support

For technical issues, file a bug via [access.redhat.com](https://access.redhat.com):
- **Product**: OpenShift Container Platform
- **Component**: MCP server for Red Hat OpenShift

For feedback: https://forms.gle/QZji8fcFMTaV8bv26

---

## Related Projects

- [containers/kubernetes-mcp-server](https://github.com/containers/kubernetes-mcp-server) — Upstream project this fork extends
- [rhobs/obs-mcp](https://github.com/rhobs/obs-mcp) — Observability MCP toolsets (Prometheus, Loki, Tempo)
- [netobserv-network-observability/netobserv-operator](https://github.com/netobserv-network-observability/netobserv-operator) — NetObserv operator
- [kuadrant/mcp-gateway](https://docs.kuadrant.io/1.4.x/mcp-gateway/) — MCP gateway for authorization and security guardrails

---

**Repository**: [openshift/openshift-mcp-server](https://github.com/openshift/openshift-mcp-server)
**Issues**: [GitHub Issues](https://github.com/openshift/openshift-mcp-server/issues)
**License**: [Apache 2.0](LICENSE)
