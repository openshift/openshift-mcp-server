# OADP Toolset

This toolset provides tools for managing OpenShift API for Data Protection (OADP) resources including Velero backups, restores, schedules, and related CRDs.

## Overview

The OADP toolset covers all 23 CRDs shipped by OADP with 90 tools organized into these categories:

| Category | CRDs | Tools |
|----------|------|-------|
| Velero Core | Backup, Restore, Schedule | 17 |
| Storage Locations | BackupStorageLocation, VolumeSnapshotLocation | 10 |
| Velero Internal | BackupRepository, DeleteBackupRequest, DownloadRequest, PodVolumeBackup, PodVolumeRestore, ServerStatusRequest | 15 |
| Data Mover (v2alpha1) | DataUpload, DataDownload | 6 |
| OADP | DataProtectionApplication, CloudStorage, DataProtectionTest | 13 |
| Non-Admin Controller | NonAdminBackup, NonAdminRestore, NonAdminBSL, NonAdminBSLRequest, NonAdminDownloadRequest | 20 |
| VM Restore | VirtualMachineBackupsDiscovery, VirtualMachineFileRestore | 8 |

## Tools

### Backup Tools

- **oadp_backup_list** - List all Velero backups
- **oadp_backup_get** - Get backup details and status
- **oadp_backup_create** - Create a new backup
- **oadp_backup_delete** - Delete a backup
- **oadp_backup_logs** - Get backup logs and status information

**Example - Create a Backup:**
```json
{
  "name": "my-app-backup",
  "includedNamespaces": ["my-app"],
  "storageLocation": "default",
  "ttl": "720h"
}
```

### Restore Tools

- **oadp_restore_list** - List all restores
- **oadp_restore_get** - Get restore details and status
- **oadp_restore_create** - Create a restore from backup
- **oadp_restore_delete** - Delete a restore record
- **oadp_restore_logs** - Get restore logs and status information

**Example - Create a Restore:**
```json
{
  "name": "my-app-restore",
  "backupName": "my-app-backup",
  "includedNamespaces": ["my-app"]
}
```

### Schedule Tools

- **oadp_schedule_list** - List backup schedules
- **oadp_schedule_get** - Get schedule details
- **oadp_schedule_create** - Create a backup schedule
- **oadp_schedule_update** - Update schedule configuration
- **oadp_schedule_delete** - Delete a schedule
- **oadp_schedule_pause** - Pause or unpause a schedule

**Example - Create a Daily Schedule:**
```json
{
  "name": "daily-backup",
  "schedule": "0 2 * * *",
  "includedNamespaces": ["production"],
  "ttl": "720h"
}
```

### Storage Location Tools

- **oadp_backup_storage_location_list** - List BackupStorageLocations
- **oadp_backup_storage_location_get** - Get BSL details
- **oadp_backup_storage_location_create** - Create a BSL
- **oadp_backup_storage_location_update** - Update BSL configuration
- **oadp_backup_storage_location_delete** - Delete a BSL
- **oadp_volume_snapshot_location_list** - List VolumeSnapshotLocations
- **oadp_volume_snapshot_location_get** - Get VSL details
- **oadp_volume_snapshot_location_create** - Create a VSL
- **oadp_volume_snapshot_location_update** - Update VSL configuration
- **oadp_volume_snapshot_location_delete** - Delete a VSL

### DataProtectionApplication Tools

- **oadp_dpa_list** - List DPA instances
- **oadp_dpa_get** - Get DPA configuration and status
- **oadp_dpa_create** - Create a DPA
- **oadp_dpa_update** - Update DPA configuration
- **oadp_dpa_delete** - Delete a DPA

### Repository and Request Tools

- **oadp_backup_repository_list/get/delete** - Manage backup repositories
- **oadp_delete_backup_request_list/get** - View delete backup requests
- **oadp_download_request_list/get/create/delete** - Manage download requests
- **oadp_server_status_request_list/get/create/delete** - Check Velero server status

### Data Mover Tools (v2alpha1)

- **oadp_data_upload_list/get/cancel** - Manage data uploads
- **oadp_data_download_list/get/cancel** - Manage data downloads

### Pod Volume Tools

- **oadp_pod_volume_backup_list/get** - View pod volume backup status
- **oadp_pod_volume_restore_list/get** - View pod volume restore status

### OADP-Specific Tools

- **oadp_cloud_storage_list/get/create/delete** - Manage cloud storage configurations
- **oadp_data_protection_test_list/get/create/delete** - Run data protection tests

### Non-Admin Controller Tools

For multi-tenant backup scenarios:

- **oadp_non_admin_backup_list/get/create/delete** - Non-admin backup operations
- **oadp_non_admin_restore_list/get/create/delete** - Non-admin restore operations
- **oadp_non_admin_bsl_list/get/create/update/delete** - Non-admin BSL management
- **oadp_non_admin_bsl_request_list/get/approve** - BSL request approval workflow
- **oadp_non_admin_download_request_list/get/create/delete** - Non-admin downloads

### VM Restore Tools

For KubeVirt virtual machine backup/restore:

- **oadp_vm_backup_discovery_list/get/create/delete** - Discover VM backups
- **oadp_vm_file_restore_list/get/create/delete** - Restore individual VM files

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
  "namespace": "custom-oadp-namespace",
  "name": "my-backup"
}
```

## Common Use Cases

### Check OADP Health

1. List DPAs: `oadp_dpa_list`
2. Get DPA status: `oadp_dpa_get` with name
3. Check BSL availability: `oadp_backup_storage_location_list`

### Create and Monitor a Backup

1. Create backup: `oadp_backup_create`
2. Check status: `oadp_backup_get`
3. View logs: `oadp_backup_logs`

### Restore from Backup

1. List backups: `oadp_backup_list`
2. Create restore: `oadp_restore_create` with `backupName`
3. Monitor: `oadp_restore_get` and `oadp_restore_logs`

### Set Up Scheduled Backups

1. Create schedule: `oadp_schedule_create` with cron expression
2. Verify: `oadp_schedule_get`
3. Pause if needed: `oadp_schedule_pause`

### Check Data Mover Status

For CSI volume backups:

1. List uploads: `oadp_data_upload_list`
2. Check progress: `oadp_data_upload_get`
3. Cancel if stuck: `oadp_data_upload_cancel`

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

### Non-Admin Tools Return Empty

Non-Admin Controller must be enabled in the DPA:
```yaml
spec:
  nonAdmin:
    enable: true
```

### VM Tools Not Working

VM backup/restore requires OADP with KubeVirt integration enabled.

## Security Considerations

### Read-Only Tools

These tools only read data and are safe for monitoring:
- All `*_list` and `*_get` tools
- `*_logs` tools

### Destructive Tools

These tools modify cluster state and require appropriate permissions:
- `*_create` tools
- `*_update` tools
- `*_delete` tools
- `*_cancel` tools
- `oadp_restore_create` (modifies cluster resources)

Use `--read-only` or `--disable-destructive` flags to restrict access:
```bash
kubernetes-mcp-server --toolsets oadp --read-only
```
