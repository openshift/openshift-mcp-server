#!/usr/bin/env bash
# Verify: Check that the restore was created from the correct backup
# Note: Must use restore.velero.io to be explicit about the API group
if kubectl get restore.velero.io app-restore -n openshift-adp >/dev/null 2>&1; then
    BACKUP_NAME=$(kubectl get restore.velero.io app-restore -n openshift-adp -o jsonpath='{.spec.backupName}')
    if [ "$BACKUP_NAME" = "restore-source-backup" ]; then
        exit 0
    else
        echo "Restore does not reference restore-source-backup"
        exit 1
    fi
else
    echo "Restore app-restore not found"
    exit 1
fi
