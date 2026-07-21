# LVMS Task Library

Evaluation tasks for the LVMS (Logical Volume Manager Storage) toolset. These tasks exercise storage troubleshooting workflows on OpenShift clusters using LVMS.

## Prerequisites

- LVMS operator installed on the cluster
- LVMCluster CR configured (typically in `openshift-lvm-storage` namespace)
- Available block devices on nodes for volume group creation

## Toolset Design

LVMS is a **prompts-only toolset** — it provides no dedicated tools. This design follows the toolset-design principle that CRUD operations on Kubernetes resources should use the core toolset's generic tools (`resources_list`, `resources_get`, etc.).

### What the LVMS toolset provides:

- **`lvms-troubleshoot` prompt** — Pre-fetches and presents:
  - LVMCluster status and device class configuration
  - LVMVolumeGroupNodeStatus per node
  - vg-manager pod health and logs
  - **Node-level LVM data** (vgs, lvs, pvs output via debug pods)
  - Events in the LVMS namespace
  - Domain knowledge (vg_attr/lv_attr field interpretation)

### What eval tasks use:

| Tool | Source | Purpose |
|------|--------|---------|
| `resources_list` / `resources_get` | core | Query LVMS CRDs (LVMCluster, LVMVolumeGroup, etc.) |
| `nodes_debug_exec` | cluster-diagnostics | Run LVM commands on nodes (vgs, lvs, pvs, lsblk) |
| `lvms-troubleshoot` prompt | lvms | Guided diagnostics with pre-fetched data |

## Task Categories

| Task | Difficulty | Description |
|------|------------|-------------|
| check-lvmcluster-status | Easy | Verify LVMCluster health and device class configuration |
| check-capacity | Medium | Analyze thin pool utilization and storage capacity |
| check-thin-pool-utilization | Medium | Check thin pool data% and metadata% thresholds |
| list-node-block-devices | Easy | List available block devices on nodes |
| show-volume-groups | Easy | Display volume group status |
| show-lvm-physical-volumes | Easy | Display physical volume status |
| diagnose-no-space-left | Hard | Diagnose thin pool exhaustion issues |
| diagnose-disk-not-used | Hard | Investigate why a disk isn't being used by LVMS |
| troubleshoot-stuck-pvc | Hard | Diagnose why a PVC is stuck in Pending state |
| troubleshoot-prompt | Medium | Use the lvms-troubleshoot prompt for guided diagnosis |
| forced-cleanup | Hard | Perform forced cleanup of stuck LVMS resources |

## Running Tasks

```bash
# Run all LVMS tasks
mcpchecker check evals/claude-code/eval.yaml --label-selector suite=lvms

# Run specific difficulty level
mcpchecker check evals/claude-code/eval.yaml --label-selector suite=lvms,difficulty=hard
```
