# OADP Support

The Kubernetes MCP Server supports OpenShift API for Data Protection (OADP) resources using the core toolset's generic resource tools. OADP resources (Velero backups, restores, schedules, and related CRDs) can be managed through the standard `resources_list`, `resources_get`, `resources_create`, and `resources_delete` tools.

The OADP toolset provides a troubleshooting prompt for diagnosing backup and restore issues.

## Prompt

### oadp-troubleshoot

Generate a step-by-step troubleshooting guide for diagnosing OADP backup and restore issues.
Gathers DPA status, BSL health, recent backup/restore status, Velero pod health, pod logs, and events into a single diagnostic workflow.

**Arguments:**
- `namespace` (optional): The OADP namespace (default: openshift-adp)
- `backup` (optional): Name of a specific backup to troubleshoot
- `restore` (optional): Name of a specific restore to troubleshoot

**Example Usage:**
```text
Use the oadp-troubleshoot prompt with namespace=openshift-adp and backup=my-failing-backup
```

## Enable the OADP Toolset

### Option 1: Command Line

```bash
kubernetes-mcp-server --toolsets core,config,oadp
```

### Option 2: Configuration File

```toml
toolsets = ["core", "config", "oadp"]
```

### Option 3: MCP Client Configuration

```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "npx",
      "args": ["-y", "kubernetes-mcp-server@latest", "--toolsets", "core,config,oadp"]
    }
  }
}
```

## Prerequisites

OADP support requires:

1. **OADP Operator installed** - The OADP operator must be installed on the cluster
2. **DataProtectionApplication configured** - At least one DPA must be configured
3. **Proper RBAC** - The user/service account must have permissions to access Velero CRDs

### Verify OADP Installation

```bash
# Check OADP operator
oc get csv -n openshift-adp | grep oadp

# Check DPA
oc get dpa -n openshift-adp

# Check BSL availability
oc get backupstoragelocations -n openshift-adp
```

## Common Use Cases

OADP resources are managed using the core toolset's generic resource tools. Here are common workflows:

### Check OADP Health

Use `resources_list` to check DPA and BSL status:
- List DPAs: `apiVersion: oadp.openshift.io/v1alpha1`, `kind: DataProtectionApplication`
- List BSLs: `apiVersion: velero.io/v1`, `kind: BackupStorageLocation`

### Create and Monitor a Backup

1. Create a backup using `resources_create` with `apiVersion: velero.io/v1`, `kind: Backup`
2. Monitor status using `resources_get`

### Restore from Backup

1. List available backups using `resources_list`
2. Create a restore using `resources_create` with `apiVersion: velero.io/v1`, `kind: Restore`
3. Monitor status using `resources_get`

### Set Up Scheduled Backups

Create a schedule using `resources_create` with `apiVersion: velero.io/v1`, `kind: Schedule`

## Troubleshooting

Use the `oadp-troubleshoot` prompt to automatically gather diagnostic information including DPA status, BSL health, Velero pod logs, and recent events.

### Common Issues

- **BSL "Unavailable"**: Check cloud credentials and bucket accessibility
- **Backup stuck "InProgress"**: Check Velero pod logs for errors
- **DPA not reconciled**: Verify the OADP operator pod is running
