# CNI Diagnostics Toolset

The CNI Diagnostics toolset provides MCP tools for Container Network Interface (CNI) diagnostics and troubleshooting.

## Overview

This toolset includes 6 MCP tools organized into two categories:

**Kernel Tools** — Diagnose kernel-level networking on nodes via privileged debug pods:

| Tool | Description |
|------|-------------|
| `get-conntrack` | Connection tracking (conntrack) entries |
| `get-iptables` | IPv4/IPv6 packet filter rules (iptables/ip6tables) |
| `get-nft` | NFtables packet filtering and classification rules |
| `get-ip` | IP routing, interfaces, neighbours, and network namespaces (iproute2) |

**Network Tools** — Advanced packet capture and eBPF tracing:

| Tool | Description |
|------|-------------|
| `tcpdump` | Packet capture on nodes or inside pod network namespaces |
| `pwru` | eBPF-based kernel packet tracing (packet, where are you?) |

All kernel tools and node-level network captures create ephemeral privileged debug pods on the target node using the configured container images. Pod-level `tcpdump` uses `pods/exec` on an existing pod instead.

## Prerequisites

### Cluster Requirements

- **Nodes**: At least one worker node
- **Kernel Features**:
  - Connection tracking (conntrack)
  - iptables or nftables
  - eBPF support for pwru (Linux kernel 4.18+)

### RBAC Requirements

The CNI Diagnostics toolset requires Kubernetes permissions to list nodes, create debug pods, and execute commands:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: cni-diagnostics-mcp-user
rules:
  # Required to verify nodes exist before creating debug pods
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["get", "list"]
  # Required to create and manage privileged debug pods on nodes
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["create", "get", "list", "delete"]
  # Required to execute commands in debug pods (and pod-level tcpdump)
  - apiGroups: [""]
    resources: ["pods/exec"]
    verbs: ["create"]
  # Required for debug pod log access
  - apiGroups: [""]
    resources: ["pods/log"]
    verbs: ["get"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: cni-diagnostics-mcp-user-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cni-diagnostics-mcp-user
subjects:
  - kind: ServiceAccount
    name: kubernetes-mcp-server
    namespace: default
```

### Security Considerations

**Important security notes:**

1. **Privileged access**: These tools create privileged debug pods with host network access
2. **Data sensitivity**: Output may contain network topology, IP addresses, and traffic patterns
3. **Resource impact**: Packet capture tools can impact node performance if used excessively
4. **Audit logging**: Tool invocations are recorded in Kubernetes audit logs

**Recommended practices**:

- Grant permissions only to trusted users
- Use namespace-scoped RoleBindings when possible
- Monitor audit logs for unexpected tool usage
- Configure `read_only = true` in server config to prevent destructive operations elsewhere in the server

## Configuration

### Enabling the Toolset

Add `cni-diagnostics` to your toolsets configuration:

```toml
# config.toml
toolsets = ["core", "config", "cni-diagnostics"]
```

When troubleshooting OVN-Kubernetes clusters, enable this toolset together with [`ovn-kubernetes`](ovn-kubernetes.md) so you have both CNI/kernel diagnostics and OVN/OVS layer tools:

```toml
# config.toml
toolsets = ["core", "ovn-kubernetes", "cni-diagnostics"]
```

### Container Image Configuration

Customize the container images used for debug pods:

```toml
[toolset_configs.cni-diagnostics]
# Container image for kernel tools (conntrack, iptables, nft, ip)
# Default: nicolaka/netshoot:v0.16
kernel_debug_image = "nicolaka/netshoot:v0.16"

# Container image for tcpdump (node-level captures)
# Default: nicolaka/netshoot:v0.16
tcpdump_image = "nicolaka/netshoot:v0.16"

# Container image for pwru (eBPF packet tracing)
# Default: docker.io/cilium/pwru:v1.0.10
pwru_image = "docker.io/cilium/pwru:v1.0.10"
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `kernel_debug_image` | string | `nicolaka/netshoot:v0.16` | Image for kernel tools (must include conntrack, iptables/ip6tables, nft, and ip utilities) |
| `tcpdump_image` | string | `nicolaka/netshoot:v0.16` | Image for node-level tcpdump captures (must include tcpdump) |
| `pwru_image` | string | `docker.io/cilium/pwru:v1.0.10` | Image for pwru (must include pwru and eBPF support) |

When the toolset config section is omitted entirely, handlers fall back to the same default images at runtime.

## Tools Reference

Kernel tools (`get-conntrack`, `get-iptables`, `get-nft`, `get-ip`) create ephemeral debug pods on the target node. Network tools use debug pods for node-level operations (`tcpdump` with `target_type=node`, `pwru`) or `pods/exec` for pod-level `tcpdump` (`target_type=pod`).

**Default behavior notes:**

- **Kernel tools / `pwru` debug pod namespace** (`namespace` on kernel tools, `node_pod_namespace` on `pwru`): omitted or empty → `default`
- **`tcpdump` `namespace`**: required when `target_type=pod`; when `target_type=node`, omitted or empty → `default` (namespace for the node debug pod, not the capture target)
- **`head`**: omitted or `0` → first **100** lines (kernel tools only; `get-nft` applies the same limit at runtime)
- **`tail`**: omitted or `0` → not applied
- **`timeout_seconds`**: omitted or `0` → server default timeout (node debug operations fall back to **60 seconds** per `nodes_debug_exec` call when the MCP request context has no deadline). When set (up to **300**), that timeout applies to the **entire tool call**

### `get-conntrack`

Inspect the connection tracking table on a node.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `node` | Yes | — | Target node name |
| `namespace` | No | `default` | Namespace for the debug pod |
| `command` | No | `-L` | Conntrack operation: `-L`/`--dump`, `-C`/`--count`, or `-S`/`--stats` |
| `filter_parameters` | No | `—` (empty) | conntrack filter flags (e.g. `-s 10.244.1.10 -p tcp --dport 443`) |
| `head` | No | `100` | Return only the first N lines of output (omitted or `0` → 100) |
| `tail` | No | `—` (not applied) | Return only the last N lines of output (only applied when set to a positive value) |
| `apply_tail_first` | No | `false` | When both `head` and `tail` are applied, apply `tail` before `head` |
| `timeout_seconds` | No | `—` | When omitted or `0`: server default timeout (see [default behavior notes](#tools-reference)). When set: timeout for the entire tool call (maximum 300) |

If the configured image lacks the `conntrack` binary, only `-L`/`--dump` is supported and output is read from `/proc/net/nf_conntrack`.

### `get-iptables`

List iptables or ip6tables rules on a node.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `node` | Yes | — | Target node name |
| `namespace` | No | `default` | Namespace for the debug pod |
| `table` | No | `filter` | Table name: `filter`, `nat`, `mangle`, `raw`, or `security` |
| `command` | No | `-L` | `-L`/`--list` or `-S`/`--list-rules` |
| `filter_parameters` | No | `—` (empty) | Additional iptables flags (e.g. `-v -n`, `-6` for IPv6) or a chain name (e.g. `FORWARD`) passed after `-L` |
| `head` | No | `100` | Return only the first N lines of output (omitted or `0` → 100) |
| `tail` | No | `—` (not applied) | Return only the last N lines of output (only applied when set to a positive value) |
| `apply_tail_first` | No | `false` | When both `head` and `tail` are applied, apply `tail` before `head` |
| `timeout_seconds` | No | `—` | When omitted or `0`: server default timeout (see [default behavior notes](#tools-reference)). When set: timeout for the entire tool call (maximum 300) |

Use `-6` or `--ipv6` in `filter_parameters` to query ip6tables instead of iptables. When `filter_parameters` is omitted, all chains in the selected `table` are listed.

### `get-nft`

List nftables rules on a node.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `node` | Yes | — | Target node name |
| `namespace` | No | `default` | Namespace for the debug pod |
| `command` | Yes | — | One of: `list ruleset`, `list tables`, `list chains`, `list sets`, `list maps`, `list flowtables` |
| `address_families` | No | `—` (empty) | Address family filter: `ip`, `ip6`, `inet`, `arp`, `bridge`, or `netdev` |
| `head` | No | `100` | Return only the first N lines of output (omitted or `0` → 100 at runtime) |
| `tail` | No | `—` (not applied) | Return only the last N lines of output (only applied when set to a positive value) |
| `apply_tail_first` | No | `false` | When both `head` and `tail` are applied, apply `tail` before `head` |
| `timeout_seconds` | No | `—` | When omitted or `0`: server default timeout (see [default behavior notes](#tools-reference)). When set: timeout for the entire tool call (maximum 300) |

### `get-ip`

Run iproute2 `ip` subcommands on a node.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `node` | Yes | — | Target node name |
| `namespace` | No | `default` | Namespace for the debug pod |
| `command` | Yes | — | One of: `address show`, `link show`, `neighbour show`, `netns show`, `route show`, `rule show`, `vrf show`, `xfrm state list`, `xfrm policy list` |
| `options` | No | `—` (empty) | ip command options (e.g. `-4`, `-6`, `-d`, `-n <netns>`) |
| `filter_parameters` | No | `—` (empty) | Additional filter arguments passed to the subcommand |
| `head` | No | `100` | Return only the first N lines of output (omitted or `0` → 100) |
| `tail` | No | `—` (not applied) | Return only the last N lines of output (only applied when set to a positive value) |
| `apply_tail_first` | No | `false` | When both `head` and `tail` are applied, apply `tail` before `head` |
| `timeout_seconds` | No | `—` | When omitted or `0`: server default timeout (see [default behavior notes](#tools-reference)). When set: timeout for the entire tool call (maximum 300) |

### `tcpdump`

Capture network packets on a node or inside a pod. Node-level captures create an ephemeral debug pod; pod-level captures use `pods/exec` on an existing pod.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `target_type` | Yes | — | `node` or `pod` |
| `name` | Yes | — | Name of the target node (when `target_type=node`) or pod (when `target_type=pod`) |
| `namespace` | When `target_type=pod` | `default` (node captures only) | Namespace of the target pod, or namespace for the node debug pod when `target_type=node` |
| `container_name` | No | (pod default container) | Container within the pod (pod captures only) |
| `interface` | No | `—` (tcpdump default) | Network interface name or `any` |
| `packet_count` | No | `100` | Packets to capture (omitted or `0` → 100; maximum 1000) |
| `bpf_filter` | No | `—` (empty) | BPF filter expression |
| `snaplen` | No | `96` | Snapshot length in bytes (omitted or `0` → 96; maximum 1500) |
| `timeout_seconds` | No | `—` | When omitted or `0`: server default timeout (see [default behavior notes](#tools-reference)). When set: timeout for the entire tool call (maximum 300) |

When stderr is present, output is labeled with `-- stdout --` and `-- stderr --` sections.

### `pwru`

Trace packets through the kernel networking stack using eBPF on a node.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `node_name` | Yes | — | Node to run pwru on |
| `node_pod_namespace` | No | `default` | Namespace for the debug pod |
| `bpf_filter` | No | `—` (empty) | BPF filter expression |
| `output_limit_lines` | No | `100` | Maximum trace events to capture (omitted or `0` → 100; maximum 1000) |
| `timeout_seconds` | No | `—` | When omitted or `0`: server default timeout (see [default behavior notes](#tools-reference)). When set: timeout for the entire tool call (maximum 300) |

When stderr is present, output is labeled with `-- stdout --` and `-- stderr --` sections (same as `tcpdump`).

## Usage Examples

### Connection Tracking (`get-conntrack`)

**List active connections**:

```json
{
  "name": "get-conntrack",
  "arguments": {
    "node": "worker-0",
    "command": "-L"
  }
}
```

**Filter connections to the API server**:

```json
{
  "name": "get-conntrack",
  "arguments": {
    "node": "worker-0",
    "command": "-L",
    "filter_parameters": "-p tcp --dport 6443",
    "head": 50
  }
}
```

**Show connection tracking statistics**:

```json
{
  "name": "get-conntrack",
  "arguments": {
    "node": "worker-0",
    "command": "-S"
  }
}
```

### IPtables Rules (`get-iptables`)

**List NAT rules**:

```json
{
  "name": "get-iptables",
  "arguments": {
    "node": "worker-0",
    "command": "-L",
    "table": "nat"
  }
}
```

**List IPv6 filter rules with verbose output**:

```json
{
  "name": "get-iptables",
  "arguments": {
    "node": "worker-0",
    "command": "-L",
    "table": "filter",
    "filter_parameters": "-6 -v -n"
  }
}
```

### NFtables (`get-nft`)

**List all rulesets**:

```json
{
  "name": "get-nft",
  "arguments": {
    "node": "worker-0",
    "command": "list ruleset"
  }
}
```

**List tables for the inet family**:

```json
{
  "name": "get-nft",
  "arguments": {
    "node": "worker-0",
    "command": "list tables",
    "address_families": "inet"
  }
}
```

### IP Routing (`get-ip`)

**Show all routing tables**:

```json
{
  "name": "get-ip",
  "arguments": {
    "node": "worker-0",
    "command": "route show",
    "filter_parameters": "table all"
  }
}
```

**Show network interfaces**:

```json
{
  "name": "get-ip",
  "arguments": {
    "node": "worker-0",
    "command": "link show"
  }
}
```

**Show IPv4 addresses**:

```json
{
  "name": "get-ip",
  "arguments": {
    "node": "worker-0",
    "command": "address show",
    "options": "-4"
  }
}
```

### Packet Capture (`tcpdump`)

**Capture TCP traffic on a node**:

```json
{
  "name": "tcpdump",
  "arguments": {
    "target_type": "node",
    "name": "worker-0",
    "packet_count": 50,
    "bpf_filter": "tcp and port 8080"
  }
}
```

**Capture in a pod**:

```json
{
  "name": "tcpdump",
  "arguments": {
    "target_type": "pod",
    "name": "my-app-pod",
    "namespace": "default",
    "packet_count": 100,
    "bpf_filter": "host 10.96.0.1"
  }
}
```

### eBPF Packet Tracing (`pwru`)

**Trace packets through the kernel**:

```json
{
  "name": "pwru",
  "arguments": {
    "node_name": "worker-0",
    "bpf_filter": "host 10.244.0.5 and tcp and dst port 443",
    "output_limit_lines": 100
  }
}
```

## Common Diagnostic Workflows

### Diagnose Pod Egress

**Scenario**: A pod cannot reach a destination outside the cluster; inspect node routing, connection state, host filtering, and on-the-wire traffic without assuming how egress or SNAT is implemented

```text
# 1. Review node routes toward the destination
get-ip: node="worker-0", command="route show", filter_parameters="table all"

# 2. Check connection tracking for flows from the pod to the destination
get-conntrack: node="worker-0", command="-L", filter_parameters="-s <pod-ip> -d <destination-ip>"

# 3. Review host packet filtering that may affect egress
get-nft: node="worker-0", command="list ruleset", address_families="inet"

# 4. Capture traffic on the node toward the destination
tcpdump: target_type="node", name="worker-0", bpf_filter="host <pod-ip> and host <destination-ip>", packet_count=50
```

### Diagnose Inter-Pod Connectivity

**Scenario**: Traffic between two pods is failing; inspect routing, connection state, the kernel packet path, and host filtering without assuming how pod networking is implemented

```text
# 1. Review node routes on the source node
get-ip: node="worker-0", command="route show", filter_parameters="table all"

# 2. Check connection tracking for flows between the pods
get-conntrack: node="worker-0", command="-L", filter_parameters="-s <source-pod-ip> -d <dest-pod-ip>"

# 3. Trace the packet path through the kernel networking stack
pwru: node_name="worker-0", bpf_filter="host <source-pod-ip> and host <dest-pod-ip>", output_limit_lines=200

# 4. Review host packet filtering if the trace stops at a filter hook
get-nft: node="worker-0", command="list ruleset", address_families="inet"
```

### Trace Dropped Pod Traffic

**Scenario**: A pod on a node cannot reach a destination (another pod, a node, or an external host); trace where packets are dropped in the kernel networking stack without assuming how services or NAT are implemented

```text
# 1. Review node routes toward the destination
get-ip: node="worker-0", command="route show", filter_parameters="table all"

# 2. Check connection tracking for flows from the pod to the destination
get-conntrack: node="worker-0", command="-L", filter_parameters="-s <pod-ip> -d <destination-ip>"

# 3. Trace the packet path through the kernel networking stack
pwru: node_name="worker-0", bpf_filter="host <pod-ip> and host <destination-ip>", output_limit_lines=200

# 4. Review host packet filtering if the trace stops at a filter hook
get-nft: node="worker-0", command="list ruleset", address_families="inet"
```

## Troubleshooting

### Common Issues

#### "Forbidden" errors

**Symptom**: `Error: pods is forbidden: User "..." cannot create resource "pods"`

**Solution**: Grant the required RBAC permissions (see [RBAC Requirements](#rbac-requirements))

#### "Node not found" errors

**Symptom**: `Error: node worker-0 does not exist`

**Solution**:

- Verify node name: `kubectl get nodes`
- Ensure the node is Ready
- Check node labels if using node selectors

#### "Image pull failed" errors

**Symptom**: `Failed to pull image: ... not found`

**Solution**:

- Verify image name in `[toolset_configs.cni-diagnostics]`
- Check image registry access from nodes
- Use `imagePullSecrets` if needed

#### "Debug pod stuck in Pending"

**Symptom**: Debug pod never reaches Running state

**Solution**:

- Check node capacity: `kubectl describe node <node-name>`
- Verify pod security policies allow privileged pods
- Check for taints/tolerations
- Review pod events: `kubectl describe pod <debug-pod>`

#### "conntrack: command not found"

**Symptom**: Error running conntrack commands

**Solution**:

- Use an image that includes the conntrack utility
- The default `nicolaka/netshoot:v0.16` image includes common networking tools
- Override with `kernel_debug_image` in config if needed

#### "tcpdump: permission denied"

**Symptom**: Cannot capture packets

**Solution**:

- Ensure the debug pod has `CAP_NET_RAW`
- Verify the node allows privileged pods
- Check pod security context and SCCs (OpenShift)

### Debug Tips

1. **Check debug pod logs**:
   ```bash
   kubectl logs -n <namespace> <debug-pod-name>
   ```

2. **Verify image has required utilities**:
   ```bash
   kubectl run test --rm -it --image=nicolaka/netshoot:v0.16 -- which conntrack
   ```

3. **Test RBAC permissions**:
   ```bash
   kubectl auth can-i create pods --as=system:serviceaccount:<namespace>:<sa-name>
   kubectl auth can-i create pods/exec --as=system:serviceaccount:<namespace>:<sa-name>
   ```

4. **Check cluster audit logs** for Forbidden events, pod creation, and exec events

## Limitations

- **Kernel tools** require ephemeral debug pod creation (adds latency)
- **pwru** requires eBPF support (Linux 4.18+) and runs only on nodes
- **Concurrent captures** may impact node performance
- **Debug pods** are ephemeral and cleaned up after use

## Performance Considerations

- Packet capture can impact node CPU if used heavily
- eBPF tracing (`pwru`) has lower overhead than `tcpdump`
- Use **BPF filters** to reduce captured packet volume
- Set **packet_count** and **output_limit_lines** to the minimum needed
- Use **head**/**tail** on kernel tools to limit output size
- Set **timeout_seconds** when the entire tool call may exceed the server default timeout (maximum 300)

## Best Practices

1. Use specific BPF filters to capture only relevant traffic
2. Limit packet counts and trace line counts to the minimum needed
3. Monitor resource usage when capturing on production nodes
4. Use `read_only = true` in server config when possible
5. Test diagnostic workflows in non-production clusters first

## Related Documentation

- [Configuration Reference](../configuration.md)
- [Core Toolset](../core.md) — for `pods_exec`, `resources_*`
- [OVN-Kubernetes Toolset](ovn-kubernetes.md) — enable alongside this toolset for complete OVN-Kubernetes troubleshooting

