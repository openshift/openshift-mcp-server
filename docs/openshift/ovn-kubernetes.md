# OVN-Kubernetes Toolset

## Overview

This toolset provides MCP tools for inspecting and troubleshooting an OVN-Kubernetes cluster across two layers:

- **OVN layer** — Query OVN Northbound/Southbound databases, list logical flows, and trace packets through the logical network. Executes OVN CLI commands (`ovn-nbctl`, `ovn-sbctl`, `ovn-trace`) inside `ovnkube-node` pods via `pods/exec`.
- **OVS layer** — Inspect Open vSwitch configuration, OpenFlow tables, and datapath state. Executes OVS CLI commands (`ovs-vsctl`, `ovs-ofctl`, `ovs-appctl`) inside `ovnkube-node` pods via `pods/exec`.

**OVN Layer Tools** — Query OVN Northbound/Southbound databases, list logical flows, and trace packets through the logical network:

| Tool | Description |
|------|-------------|
| `ovn_show` | OVN configuration overview via `ovn-nbctl show` / `ovn-sbctl show` |
| `ovn_get` | Query records from OVN database tables via `ovn-nbctl list` / `ovn-sbctl list` |
| `ovn_lflow_list` | List logical flows from the Southbound database via `ovn-sbctl lflow-list` |
| `ovn_trace` | Trace a packet through the OVN logical network via `ovn-trace` |

**OVS Layer Tools** — Inspect Open vSwitch configuration, OpenFlow tables, and datapath state on ovnkube-node pods:

| Tool | Description |
|------|-------------|
| `ovs_vsctl` | OVS switch configuration (bridges, ports, interfaces) via `ovs-vsctl` |
| `ovs_ofctl` | OpenFlow flow inspection via `ovs-ofctl` |
| `ovs_appctl` | OVS datapath and pipeline diagnostics via `ovs-appctl` |

All tools in this toolset are read-only and do not modify OVN or OVS state.

## Prerequisites

### Cluster Requirements

- **CNI**: OVN-Kubernetes installed and configured
- **Nodes**: At least one node running `ovnkube-node` pods

### RBAC Requirements

The OVN-Kubernetes toolset requires Kubernetes permissions to list ovnkube-node pods and execute commands inside them:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: ovn-kubernetes-mcp-user
rules:
  # Required to discover ovnkube-node pods
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list"]
  # Required to execute OVN/OVS commands inside ovnkube-node containers
  - apiGroups: [""]
    resources: ["pods/exec"]
    verbs: ["create"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: ovn-kubernetes-mcp-user-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: ovn-kubernetes-mcp-user
subjects:
  - kind: ServiceAccount
    name: kubernetes-mcp-server
    namespace: default
```

### Security Considerations

**Important security notes:**

1. **Pod exec access**: These tools use `pods/exec` on `ovnkube-node` pods, which run with elevated privileges
2. **Data sensitivity**: Output may contain network topology, IP addresses, MAC addresses, and traffic patterns
3. **Audit logging**: Tool invocations are recorded in Kubernetes audit logs

**Recommended practices**:

- Grant permissions only to trusted users
- Scope the ClusterRoleBinding to the `openshift-ovn-kubernetes` namespace where possible
- Monitor audit logs for unexpected tool usage

## Configuration

### Enabling the Toolset

Add `ovn-kubernetes` to your toolsets configuration:

```toml
# config.toml
toolsets = ["core", "ovn-kubernetes", "cni-diagnostics"]
```

The `core` toolset is required for discovering ovnkube-node pods via `pods_list`.

For a complete OVN-Kubernetes troubleshooting toolkit, also enable the [`cni-diagnostics`](cni-diagnostics.md) toolset. It provides kernel-level networking tools (conntrack, iptables, nftables, ip) and packet capture/tracing (`tcpdump`, `pwru`) that complement the OVN and OVS layer tools in this toolset.

## Tools Reference

OVN layer tools (`ovn_show`, `ovn_get`, `ovn_lflow_list`, `ovn_trace`) execute against an `ovnkube-node` pod.

**Default behavior notes:**

- **`namespace`**: defaults to `openshift-ovn-kubernetes`
- **`name`**: required; the name of a specific `ovnkube-node` pod (discover with `pods_list`)
- **`head`**: omitted or `0` → first **100** lines when `tail` is not specified
- **`tail`**: omitted or `0` → not applied
- **`apply_tail_first`**: `false` by default; when both `head` and `tail` are set, `head` is applied first

### `ovn_show`

Display a comprehensive overview of OVN configuration from either the Northbound or Southbound database.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `namespace` | No | `openshift-ovn-kubernetes` | Kubernetes namespace of the ovnkube-node pod |
| `name` | Yes | — | Name of the ovnkube-node pod |
| `database` | Yes | — | One of: `nbdb`, `sbdb` |
| `head` | No | `100` | Return only the first N lines of output |
| `tail` | No | — (not applied) | Return only the last N lines of output |
| `apply_tail_first` | No | `false` | When both `head` and `tail` are applied, apply `tail` before `head` |

Database modes:

- `nbdb` — runs `ovn-nbctl show`: logical switches, logical routers, their ports, and connections
- `sbdb` — runs `ovn-sbctl show`: chassis information, port bindings, and their relationships

### `ovn_get`

Query records from an OVN database table with flexible filtering.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `namespace` | No | `openshift-ovn-kubernetes` | Kubernetes namespace of the ovnkube-node pod |
| `name` | Yes | — | Name of the ovnkube-node pod |
| `database` | Yes | — | One of: `nbdb`, `sbdb` |
| `table` | Yes | — | Name of the OVN database table to query |
| `record` | No | — (list all) | Record identifier (UUID or name); if omitted, lists all records |
| `columns` | No | — (all columns) | Comma-separated list of columns to display (e.g., `name,_uuid,ports`) |
| `pattern` | No | — (empty) | Regex pattern to filter results (only applies when listing all records) |
| `head` | No | `100` | Return only the first N lines of output |
| `tail` | No | — (not applied) | Return only the last N lines of output |
| `apply_tail_first` | No | `false` | When both `head` and `tail` are applied, apply `tail` before `head` |

Common Northbound tables: `Logical_Switch`, `Logical_Router`, `Logical_Switch_Port`, `Logical_Router_Port`, `ACL`, `Address_Set`, `Port_Group`, `Load_Balancer`, `NAT`

Common Southbound tables: `Chassis`, `Port_Binding`, `Datapath_Binding`, `Logical_Flow`, `MAC_Binding`, `Multicast_Group`, `SB_Global`

### `ovn_lflow_list`

List logical flows from the OVN Southbound database.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `namespace` | No | `openshift-ovn-kubernetes` | Kubernetes namespace of the ovnkube-node pod |
| `name` | Yes | — | Name of the ovnkube-node pod |
| `datapath` | No | — (all datapaths) | Datapath name or UUID to filter flows for a specific logical switch/router |
| `pattern` | No | — (empty) | Regex pattern to filter flows |
| `head` | No | `100` | Return only the first N lines of output |
| `tail` | No | — (not applied) | Return only the last N lines of output |
| `apply_tail_first` | No | `false` | When both `head` and `tail` are applied, apply `tail` before `head` |

### `ovn_trace`

Trace a packet through the OVN logical network.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `namespace` | No | `openshift-ovn-kubernetes` | Kubernetes namespace of the ovnkube-node pod |
| `name` | Yes | — | Name of the ovnkube-node pod |
| `datapath` | Yes | — | Name of the logical switch or router to start the trace |
| `microflow` | Yes | — | Microflow specification describing the packet |
| `mode` | No | `detailed` | Output verbosity: `detailed`, `summary`, or `minimal` |
| `pattern` | No | — (empty) | Regex pattern to filter trace output |
| `head` | No | `100` | Return only the first N lines of output |
| `tail` | No | — (not applied) | Return only the last N lines of output |
| `apply_tail_first` | No | `false` | When both `head` and `tail` are applied, apply `tail` before `head` |

Microflow specification examples:

- `inport=="pod1" && eth.src==00:00:00:00:00:01 && ip4.src==10.244.0.5 && ip4.dst==10.244.1.5`
- `inport=="pod1" && eth.src==00:00:00:00:00:01 && icmp && ip4.src==10.244.0.5 && ip4.dst==8.8.8.8`

---

OVS layer tools (`ovs_vsctl`, `ovs_ofctl`, `ovs_appctl`) execute against an `ovnkube-node` pod. The required `action` parameter selects the specific subcommand to run.

**Default behavior notes for OVS layer tools:**

- **`namespace`**: required; on OpenShift this is `openshift-ovn-kubernetes`
- **`name`**: required; the name of a specific `ovnkube-node` pod (discover with `pods_list`)
- **`head`**: omitted or `0` → first **100** lines when `tail` is not specified
- **`tail`**: omitted or `0` → not applied
- **`apply_tail_first`**: `false` by default; when both `head` and `tail` are set, `head` is applied first

### `ovs_vsctl`

Inspect OVS switch configuration on an `ovnkube-node` pod.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `namespace` | Yes | — | Kubernetes namespace of the ovnkube-node pod (e.g. `openshift-ovn-kubernetes`) |
| `name` | Yes | — | Name of the ovnkube-node pod |
| `action` | Yes | — | One of: `show`, `list-br`, `list-ports`, `list-ifaces` |
| `bridge` | When `action=list-ports` or `list-ifaces` | — | OVS bridge name (e.g. `br-int`) |
| `head` | No | `100` | Return only the first N lines of output (only used when `action=show`) |
| `tail` | No | `—` (not applied) | Return only the last N lines of output (only used when `action=show`) |
| `apply_tail_first` | No | `false` | When both `head` and `tail` are applied, apply `tail` before `head` (only used when `action=show`) |

Actions:

- `show` — full OVS configuration overview: bridges, ports, interfaces, controllers, versions
- `list-br` — bridge names only
- `list-ports` — ports attached to the specified bridge
- `list-ifaces` — interfaces attached to the specified bridge

### `ovs_ofctl`

Inspect OpenFlow flow tables on an OVS bridge.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `namespace` | Yes | — | Kubernetes namespace of the ovnkube-node pod |
| `name` | Yes | — | Name of the ovnkube-node pod |
| `action` | Yes | — | Currently: `dump-flows` |
| `bridge` | Yes | — | OVS bridge name (e.g. `br-int`) |
| `pattern` | No | `—` (empty) | Regex pattern to filter output lines |
| `head` | No | `100` | Return only the first N lines of output |
| `tail` | No | `—` (not applied) | Return only the last N lines of output |
| `apply_tail_first` | No | `false` | When both `head` and `tail` are applied, apply `tail` before `head` |

### `ovs_appctl`

Run OVS datapath and OpenFlow pipeline diagnostics on an `ovnkube-node` pod.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `namespace` | Yes | — | Kubernetes namespace of the ovnkube-node pod |
| `name` | Yes | — | Name of the ovnkube-node pod |
| `action` | Yes | — | One of: `dpctl/dump-conntrack`, `ofproto/trace` |
| `bridge` | When `action=ofproto/trace` | — | OVS bridge name (e.g. `br-int`) |
| `flow` | When `action=ofproto/trace` | — | Flow specification (e.g. `in_port=1,ip,nw_src=10.244.0.5,nw_dst=10.96.0.1`) |
| `additional_params` | No | `—` (empty) | Additional CLI arguments (only used when `action=dpctl/dump-conntrack`; e.g. `["zone=5"]`) |
| `pattern` | No | `—` (empty) | Regex pattern to filter output lines |
| `head` | No | `100` | Return only the first N lines of output |
| `tail` | No | `—` (not applied) | Return only the last N lines of output |
| `apply_tail_first` | No | `false` | When both `head` and `tail` are applied, apply `tail` before `head` |

Actions:

- `dpctl/dump-conntrack` — datapath connection tracking entries (protocol, source, destination, ports, state)
- `ofproto/trace` — simulate a packet through the OpenFlow pipeline for the given bridge and flow

## Usage Examples

### OVN Configuration (`ovn_show`)

**Show Northbound configuration on a specific ovnkube-node pod**:

```json
{
  "name": "ovn_show",
  "arguments": {
    "namespace": "openshift-ovn-kubernetes",
    "name": "ovnkube-node-abcde",
    "database": "nbdb"
  }
}
```

**Show Southbound configuration (chassis and port bindings)**:

```json
{
  "name": "ovn_show",
  "arguments": {
    "namespace": "openshift-ovn-kubernetes",
    "name": "ovnkube-node-abcde",
    "database": "sbdb"
  }
}
```

### Query OVN Database (`ovn_get`)

**List all logical switches**:

```json
{
  "name": "ovn_get",
  "arguments": {
    "namespace": "openshift-ovn-kubernetes",
    "name": "ovnkube-node-abcde",
    "database": "nbdb",
    "table": "Logical_Switch"
  }
}
```

**Get a specific logical router with selected columns**:

```json
{
  "name": "ovn_get",
  "arguments": {
    "namespace": "openshift-ovn-kubernetes",
    "name": "ovnkube-node-abcde",
    "database": "nbdb",
    "table": "Logical_Router",
    "record": "ovn_cluster_router",
    "columns": "name,ports,nat"
  }
}
```

**List ACLs matching a pattern**:

```json
{
  "name": "ovn_get",
  "arguments": {
    "namespace": "openshift-ovn-kubernetes",
    "name": "ovnkube-node-abcde",
    "database": "nbdb",
    "table": "ACL",
    "pattern": "action=drop"
  }
}
```

### Logical Flows (`ovn_lflow_list`)

**List all logical flows**:

```json
{
  "name": "ovn_lflow_list",
  "arguments": {
    "namespace": "openshift-ovn-kubernetes",
    "name": "ovnkube-node-abcde"
  }
}
```

**List logical flows for a specific datapath**:

```json
{
  "name": "ovn_lflow_list",
  "arguments": {
    "namespace": "openshift-ovn-kubernetes",
    "name": "ovnkube-node-abcde",
    "datapath": "node1"
  }
}
```

**Filter logical flows by pattern**:

```json
{
  "name": "ovn_lflow_list",
  "arguments": {
    "namespace": "openshift-ovn-kubernetes",
    "name": "ovnkube-node-abcde",
    "pattern": "ls_in_acl"
  }
}
```

### Packet Tracing (`ovn_trace`)

**Trace a packet through the logical network**:

```json
{
  "name": "ovn_trace",
  "arguments": {
    "namespace": "openshift-ovn-kubernetes",
    "name": "ovnkube-node-abcde",
    "datapath": "node1",
    "microflow": "inport==\"pod1\" && eth.src==00:00:00:00:00:01 && ip4.src==10.244.0.5 && ip4.dst==10.244.1.5"
  }
}
```

**Trace with summary output mode**:

```json
{
  "name": "ovn_trace",
  "arguments": {
    "namespace": "openshift-ovn-kubernetes",
    "name": "ovnkube-node-abcde",
    "datapath": "node1",
    "microflow": "inport==\"pod1\" && eth.src==00:00:00:00:00:01 && ip4.src==10.244.0.5 && ip4.dst==10.244.1.5",
    "mode": "summary"
  }
}
```

### OVS Configuration (`ovs_vsctl`)

**Show full OVS configuration on a specific ovnkube-node pod**:

```json
{
  "name": "ovs_vsctl",
  "arguments": {
    "namespace": "openshift-ovn-kubernetes",
    "name": "ovnkube-node-abcde",
    "action": "show"
  }
}
```

**List bridges**:

```json
{
  "name": "ovs_vsctl",
  "arguments": {
    "namespace": "openshift-ovn-kubernetes",
    "name": "ovnkube-node-abcde",
    "action": "list-br"
  }
}
```

**List ports attached to br-int**:

```json
{
  "name": "ovs_vsctl",
  "arguments": {
    "namespace": "openshift-ovn-kubernetes",
    "name": "ovnkube-node-abcde",
    "action": "list-ports",
    "bridge": "br-int"
  }
}
```

### OpenFlow Flows (`ovs_ofctl`)

**Dump all flows on br-int**:

```json
{
  "name": "ovs_ofctl",
  "arguments": {
    "namespace": "openshift-ovn-kubernetes",
    "name": "ovnkube-node-abcde",
    "action": "dump-flows",
    "bridge": "br-int"
  }
}
```

**Filter flows matching a specific priority**:

```json
{
  "name": "ovs_ofctl",
  "arguments": {
    "namespace": "openshift-ovn-kubernetes",
    "name": "ovnkube-node-abcde",
    "action": "dump-flows",
    "bridge": "br-int",
    "pattern": "priority=100"
  }
}
```

### Datapath and Pipeline (`ovs_appctl`)

**Dump datapath connection tracking**:

```json
{
  "name": "ovs_appctl",
  "arguments": {
    "namespace": "openshift-ovn-kubernetes",
    "name": "ovnkube-node-abcde",
    "action": "dpctl/dump-conntrack"
  }
}
```

**Trace a packet through the OpenFlow pipeline on br-int**:

```json
{
  "name": "ovs_appctl",
  "arguments": {
    "namespace": "openshift-ovn-kubernetes",
    "name": "ovnkube-node-abcde",
    "action": "ofproto/trace",
    "bridge": "br-int",
    "flow": "in_port=1,ip,nw_src=10.244.0.5,nw_dst=10.96.0.1"
  }
}
```

## Common Diagnostic Workflows

### Inspect Logical Network Topology

**Scenario**: Understand the logical network layout for a cluster.

```
# 1. Find ovnkube-node pods
pods_list: labelSelector="app=ovnkube-node"

# 2. Show Northbound logical topology (switches, routers, ports)
ovn_show: namespace="openshift-ovn-kubernetes", name="<pod>", database="nbdb"

# 3. List all logical switches
ovn_get: namespace="openshift-ovn-kubernetes", name="<pod>", database="nbdb", table="Logical_Switch"

# 4. List all logical routers
ovn_get: namespace="openshift-ovn-kubernetes", name="<pod>", database="nbdb", table="Logical_Router"
```

### Debug Pod Connectivity

**Scenario**: A pod cannot reach another pod; trace the packet through OVN.

```
# 1. Find ovnkube-node pods
pods_list: labelSelector="app=ovnkube-node"

# 2. Show Southbound chassis and port bindings to identify where pods are bound
ovn_show: namespace="openshift-ovn-kubernetes", name="<pod>", database="sbdb"

# 3. Find the port binding for the source pod
ovn_get: namespace="openshift-ovn-kubernetes", name="<pod>", database="sbdb",
         table="Port_Binding", pattern="<pod-name>"

# 4. Trace the packet through the logical network
ovn_trace: namespace="openshift-ovn-kubernetes", name="<pod>",
           datapath="<logical-switch>",
           microflow="inport==\"<port>\" && eth.src==<mac> && ip4.src=<src-ip> && ip4.dst=<dst-ip>"

# 5. Inspect logical flows for the relevant datapath
ovn_lflow_list: namespace="openshift-ovn-kubernetes", name="<pod>",
                datapath="<logical-switch>"
```

### Inspect ACLs and Security Policies

**Scenario**: Verify that network policies are correctly translated to OVN ACLs.

```
# 1. Find ovnkube-node pods
pods_list: labelSelector="app=ovnkube-node"

# 2. List all ACLs in the Northbound database
ovn_get: namespace="openshift-ovn-kubernetes", name="<pod>", database="nbdb", table="ACL"

# 3. Filter for drop rules
ovn_get: namespace="openshift-ovn-kubernetes", name="<pod>", database="nbdb",
         table="ACL", pattern="action=drop"

# 4. Check port groups associated with network policies
ovn_get: namespace="openshift-ovn-kubernetes", name="<pod>", database="nbdb", table="Port_Group"
```

### Verify OVS Configuration on a Node

**Scenario**: Confirm that OVS bridges and ports are configured as expected on a specific node.

```text
# 1. Find ovnkube-node pods
pods_list: labelSelector="app=ovnkube-node"

# 2. Show OVS configuration overview
ovs_vsctl: namespace="openshift-ovn-kubernetes", name="<pod>", action="show"

# 3. Inspect ports on br-int specifically
ovs_vsctl: namespace="openshift-ovn-kubernetes", name="<pod>", action="list-ports", bridge="br-int"
```

### Trace Packet Forwarding for a Pod

**Scenario**: A pod cannot reach a Kubernetes Service; simulate the packet path through OVS on the source node.

```text
# 1. Locate the ovnkube-node pod on the source node
pods_list: labelSelector="app=ovnkube-node"

# 2. Trace the packet through the br-int OpenFlow pipeline
ovs_appctl: namespace="openshift-ovn-kubernetes", name="<pod>", action="ofproto/trace",
            bridge="br-int",
            flow="in_port=1,ip,nw_src=<pod-ip>,nw_dst=<service-clusterip>"

# 3. Inspect the matching flows on br-int for additional context
ovs_ofctl: namespace="openshift-ovn-kubernetes", name="<pod>", action="dump-flows",
           bridge="br-int"
```

### Inspect Datapath Conntrack State

**Scenario**: Investigate stateful connections tracked by the OVS datapath on a node.

```text
# 1. Locate an ovnkube-node pod on the node of interest
pods_list: labelSelector="app=ovnkube-node"

# 2. Dump datapath conntrack entries
ovs_appctl: namespace="openshift-ovn-kubernetes", name="<pod>", action="dpctl/dump-conntrack"
```

## Troubleshooting

### Common Issues

#### "Forbidden" errors

**Symptom**: `Error: pods "..." is forbidden: User "..." cannot create resource "pods/exec"`

**Solution**: Grant the required RBAC permissions (see [RBAC Requirements](#rbac-requirements)).

#### "Container not found" errors

**Symptom**: `Error: container "nbdb" not found in pod "..."` or `Error: container "ovn-controller" not found in pod "..."`

**Solution**:

- Confirm the pod is an `ovnkube-node` pod (`kubectl get pod <name> -o jsonpath='{.spec.containers[*].name}'`)
- On some OVN-K variants the container names may differ; use `pods_list` with the `app=ovnkube-node` label selector to find the correct pod

#### "Bridge not found" errors

**Symptom**: `ovs-vsctl: no bridge named ...`

**Solution**:

- Use `ovs_vsctl` with `action=list-br` to see the actual bridges on the node
- `br-int` is the OVN integration bridge and is always present; `br-ex` exists on OpenShift but may be named differently on other distributions

#### Empty or truncated output

**Symptom**: Output appears incomplete or is cut off.

**Solution**:

- By default, output is limited to 100 lines. Use the `head` parameter with a larger value, or use `tail` to get the end of the output
- Use the `pattern` parameter to filter results and reduce output volume
- For `ovn_get`, specify `columns` to limit the data returned per record

#### "no record" errors from `ovn_get`

**Symptom**: `ovn-nbctl: no row "..." in table "..."`

**Solution**:

- Verify the table name is correct (table names are case-sensitive, e.g., `Logical_Switch` not `logical_switch`)
- Verify the record identifier (UUID or name) exists by listing all records first (omit the `record` parameter)

## Related Documentation

- [Configuration Reference](../configuration.md)
- [Core Toolset](../README.md) — for `pods_list`, `pods_exec`, and other Kubernetes primitives
- [CNI Diagnostics Toolset](cni-diagnostics.md) — kernel networking and packet capture/tracing tools to use alongside this toolset for OVN-Kubernetes troubleshooting
