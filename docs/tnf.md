# TNF (Two-Node Fencing) Support

The `tnf` toolset extends the OpenShift MCP server with diagnostics for Two-Node Fencing clusters. These are 2-node bare metal OpenShift clusters that use pacemaker and STONITH for fencing to prevent split-brain scenarios.

## Tools

### tnf_check_fencing_config

Validates fencing configuration and readiness for a TNF cluster. Checks cluster topology (platform, node count, TNF profile), critical operator health (etcd, machine-api, baremetal), Machine/Node/BareMetalHost correlation, BMC addresses and credential secrets, FenceAgentsRemediation templates and active remediations, and NodeHealthCheck resources. Returns a diagnostic summary identifying configuration issues that could prevent fencing from functioning correctly.

**Arguments:**
- `namespace` (optional): Namespace containing BareMetalHost resources (default: searches all namespaces)

### tnf_check_stonith_status

Creates a temporary privileged debug pod on a control-plane node to run `pcs` diagnostic commands. Returns pacemaker cluster state, STONITH device configuration, quorum status, and recent fencing history. The debug pod is automatically cleaned up after execution.

**Arguments:**
- `node` (optional): Name of the node to run diagnostics on (default: auto-detects the first control-plane node)
- `namespace` (optional): Namespace to create the temporary debug pod in (default: `default`)
- `timeout_seconds` (optional): Maximum time in seconds to wait for the diagnostic commands to complete (default: 120)

## Prompt

### tnf-troubleshoot

Runs a full TNF fencing diagnostic workflow that collects cluster topology, node health, critical operator status, BareMetalHost/BMC health, pacemaker/STONITH status, and remediation operator status into a single structured report. Guides the LLM through analysis using the domain knowledge resource to assess split-brain risk and recommend actions.

**Arguments:**
- `node` (optional): Preferred node to run STONITH diagnostics from

**Invoke in Claude Code:**
```text
/mcp__kubernetes-mcp-server__tnf-troubleshoot
```

## Resource

### tnf://domain-knowledge/fencing

Static reference material covering:
- How two-node fencing works (corosync, pacemaker, STONITH)
- Two-node quorum rules and `wait_for_all` behavior
- Split-brain risk assessment matrix
- Common issues and recovery procedures (STONITH disabled, BMC unreachable, node won't rejoin, quorum lost, fence race)

## Enable the TNF Toolset

### Option 1: Command Line

```bash
kubernetes-mcp-server --toolsets core,config,tnf
```

### Option 2: Configuration File

```toml
toolsets = ["core", "config", "tnf"]
```

### Option 3: MCP Client Configuration

```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "kubernetes-mcp-server",
      "args": ["--toolsets", "core,config,tnf"]
    }
  }
}
```

## Prerequisites

TNF support requires:

1. **Two-node bare metal OpenShift cluster** with pacemaker/STONITH fencing configured
2. **BareMetalHost CRDs** installed (standard on bare metal clusters via the baremetal-operator)
3. **Proper RBAC** for reading Nodes, BareMetalHosts, Secrets, and creating debug pods

### Verify TNF Setup

```bash
# Check for 2-node bare metal cluster
oc get nodes
oc get infrastructure cluster -o jsonpath='{.status.platform}'

# Check BareMetalHosts exist
oc get baremetalhosts -A

# Check pacemaker is running
oc debug node/<node-name> -- chroot /host pcs status
```

## What It Diagnoses

| Area | What's Checked |
|------|---------------|
| Cluster topology | Platform, node count, TNF profile detection |
| Node health | Kubernetes Ready status, conditions |
| BareMetalHost | Provisioning state, power status, BMC address |
| BMC credentials | Secret existence, username/password keys present |
| Pacemaker | Cluster name, online/offline/standby nodes |
| STONITH devices | Configuration, agent type, target node, started status |
| Quorum | Quorate status, vote count, two-node flags |
| Fencing history | Recent fencing events, success/failure |
| Remediation operators | FenceAgentsRemediation and NodeHealthCheck CRD status |

## Limitations

- **API must be reachable**: All tools depend on the Kubernetes API. When the API is unreachable (e.g. during a fencing event that takes down the API server), the tools detect the failure and return an out-of-band recovery guide with BMC access and manual diagnostic procedures instead of raw cluster data. The `tnf-troubleshoot` prompt integrates this fallback automatically.
- **Read-only diagnostics**: The tools do not trigger, stop, or modify fencing operations.
- **Debug pod privileges**: The STONITH check requires creating a privileged debug pod with host access (same as `oc debug node/<name>`).
