#!/usr/bin/env bash
# Verify: Check that the backup exists (agent should have called oadp_backup with action: get)
# Note: Must use backup.velero.io to avoid conflict with backups.config.openshift.io
kubectl get backup.velero.io status-check-backup -n openshift-adp >/dev/null 2>&1
exit $?
