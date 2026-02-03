# OADP Toolset

This toolset provides tools for managing OpenShift API for Data Protection (OADP) resources including Velero backups, restores, schedules, and related CRDs.

## Overview

The OADP toolset provides 8 consolidated tools with action-based parameters covering all core OADP functionality:

| Tool | CRDs Covered | Actions |
|------|--------------|---------|
| `oadp_backup` | Backup | list, get, create, delete, logs |
| `oadp_restore` | Restore | list, get, create, delete, logs |
| `oadp_schedule` | Schedule | list, get, create, update, delete, pause |
| `oadp_dpa` | DataProtectionApplication | list, get, create, update, delete |
| `oadp_storage_location` | BackupStorageLocation, VolumeSnapshotLocation | list, get, create, update, delete |
| `oadp_data_mover` | DataUpload, DataDownload | list, get, cancel |
| `oadp_repository` | BackupRepository | list, get, delete |
| `oadp_data_protection_test` | DataProtectionTest | list, get, create, delete |

## Tools

### oadp_backup

Manage Velero/OADP backups: list, get, create, delete, or retrieve logs.

**Parameters:**
- `action` (required): One of `list`, `get`, `create`, `delete`, `logs`
- `namespace`: Namespace containing backups (default: openshift-adp)
- `name`: Name of the backup (required for get, create, delete, logs)
- `includedNamespaces`: Namespaces to include in backup (for create)
- `excludedNamespaces`: Namespaces to exclude (for create)
- `storageLocation`: BackupStorageLocation name (for create)
- `ttl`: Backup TTL duration e.g., '720h' (for create)

**Example - List Backups:**
```json
{
  "action": "list",
  "namespace": "openshift-adp"
}
```

**Example - Create a Backup:**
```json
{
  "action": "create",
  "name": "my-app-backup",
  "includedNamespaces": ["my-app"],
  "storageLocation": "default",
  "ttl": "720h"
}
```

### oadp_restore

Manage Velero/OADP restore operations: list, get, create, delete, or retrieve logs.

**Parameters:**
- `action` (required): One of `list`, `get`, `create`, `delete`, `logs`
- `namespace`: Namespace containing restores (default: openshift-adp)
- `name`: Name of the restore (required for get, create, delete, logs)
- `backupName`: Name of the backup to restore from (required for create)
- `includedNamespaces`: Namespaces to restore (for create)
- `namespaceMapping`: Map source namespaces to target namespaces (for create)

**Example - Create a Restore:**
```json
{
  "action": "create",
  "name": "my-app-restore",
  "backupName": "my-app-backup"
}
```

### oadp_schedule

Manage Velero/OADP backup schedules: list, get, create, update, delete, or pause/unpause.

**Parameters:**
- `action` (required): One of `list`, `get`, `create`, `update`, `delete`, `pause`
- `namespace`: Namespace containing schedules (default: openshift-adp)
- `name`: Name of the schedule (required for get, create, update, delete, pause)
- `schedule`: Cron expression e.g., '0 1 * * *' (for create/update)
- `includedNamespaces`: Namespaces to include in scheduled backups (for create)
- `ttl`: Backup TTL duration (for create/update)
- `paused`: Set to true to pause, false to unpause (for pause action)

**Example - Create a Daily Schedule:**
```json
{
  "action": "create",
  "name": "daily-backup",
  "schedule": "0 2 * * *",
  "includedNamespaces": ["production"],
  "ttl": "720h"
}
```

### oadp_dpa

Manage OADP DataProtectionApplication resources: list, get, create, update, or delete.

**Parameters:**
- `action` (required): One of `list`, `get`, `create`, `update`, `delete`
- `namespace`: Namespace containing DPAs (default: openshift-adp)
- `name`: Name of the DPA (required for get, create, update, delete)
- `backupLocationProvider`: Provider for backup storage e.g., aws, azure, gcp (for create)
- `backupLocationBucket`: Bucket name for backup storage (for create)
- `enableNodeAgent`: Enable NodeAgent for file-system backups (for create/update)

### oadp_storage_location

Manage Velero storage locations (BackupStorageLocation and VolumeSnapshotLocation): list, get, create, update, or delete.

**Parameters:**
- `action` (required): One of `list`, `get`, `create`, `update`, `delete`
- `type` (required): Storage location type: `bsl` (BackupStorageLocation) or `vsl` (VolumeSnapshotLocation)
- `namespace`: Namespace containing storage locations (default: openshift-adp)
- `name`: Name of the storage location (required for get, create, update, delete)
- `provider`: Storage provider e.g., aws, azure, gcp (for create)
- `bucket`: Bucket name for object storage (for BSL create)
- `region`: Region for the storage (for create/update)

**Example - List Backup Storage Locations:**
```json
{
  "action": "list",
  "type": "bsl"
}
```

### oadp_data_mover

Manage Velero data mover resources (DataUpload and DataDownload for CSI snapshots): list, get, or cancel.

**Parameters:**
- `action` (required): One of `list`, `get`, `cancel`
- `type` (required): Resource type: `upload` (DataUpload) or `download` (DataDownload)
- `namespace`: Namespace containing resources (default: openshift-adp)
- `name`: Name of the resource (required for get, cancel)
- `labelSelector`: Label selector to filter resources (for list)

### oadp_repository

Manage Velero BackupRepository resources (connections to backup storage): list, get, or delete.

**Parameters:**
- `action` (required): One of `list`, `get`, `delete`
- `namespace`: Namespace containing repositories (default: openshift-adp)
- `name`: Name of the repository (required for get, delete)

### oadp_data_protection_test

Manage OADP DataProtectionTest resources for validating storage connectivity: list, get, create, or delete.

**Parameters:**
- `action` (required): One of `list`, `get`, `create`, `delete`
- `namespace`: Namespace containing resources (default: openshift-adp)
- `name`: Name of the test (required for get, create, delete)
- `backupLocationName`: Name of the BackupStorageLocation to test (for create)
- `uploadTestFileSize`: Size of test file for upload speed test e.g., '100MB' (for create)

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

The OADP tools require:

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

## Default Namespace

All OADP tools default to the `openshift-adp` namespace. Override with the `namespace` parameter:

```json
{
  "action": "list",
  "namespace": "custom-oadp-namespace"
}
```

## Common Use Cases

### Check OADP Health

```json
{"action": "list"}  // oadp_dpa
{"action": "get", "name": "velero-sample"}  // oadp_dpa
{"action": "list", "type": "bsl"}  // oadp_storage_location
```

### Create and Monitor a Backup

```json
{"action": "create", "name": "my-backup", "includedNamespaces": ["my-app"]}  // oadp_backup
{"action": "get", "name": "my-backup"}  // oadp_backup
{"action": "logs", "name": "my-backup"}  // oadp_backup
```

### Restore from Backup

```json
{"action": "list"}  // oadp_backup
{"action": "create", "name": "my-restore", "backupName": "my-backup"}  // oadp_restore
{"action": "get", "name": "my-restore"}  // oadp_restore
```

### Set Up Scheduled Backups

```json
{"action": "create", "name": "daily", "schedule": "0 2 * * *", "includedNamespaces": ["prod"]}  // oadp_schedule
{"action": "get", "name": "daily"}  // oadp_schedule
{"action": "pause", "name": "daily", "paused": true}  // oadp_schedule
```

### Check Data Mover Status

For CSI volume backups:

```json
{"action": "list", "type": "upload"}  // oadp_data_mover
{"action": "get", "type": "upload", "name": "my-upload"}  // oadp_data_mover
{"action": "cancel", "type": "upload", "name": "my-upload"}  // oadp_data_mover
```

## Troubleshooting

### "DPA not found" Error

Ensure OADP is installed and configured:
```bash
oc get dpa -n openshift-adp
```

### Backup Stuck in "InProgress"

Check Velero pod logs:
```bash
oc logs -n openshift-adp -l app.kubernetes.io/name=velero
```

### BSL Shows "Unavailable"

Verify cloud credentials:
```bash
oc get secret -n openshift-adp cloud-credentials
oc get bsl -n openshift-adp -o yaml
```

## Security Considerations

All tools use the `action` parameter pattern, making it clear what operation will be performed:

- **Read-only actions**: `list`, `get`, `logs`
- **Mutating actions**: `create`, `update`, `delete`, `cancel`, `pause`

Use `--read-only` or `--disable-destructive` flags to restrict access:
```bash
kubernetes-mcp-server --toolsets oadp --read-only
```
