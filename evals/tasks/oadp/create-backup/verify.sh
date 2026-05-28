#!/usr/bin/env bash
# Verify: Check that the backup was created
# Note: Must use backup.velero.io to avoid conflict with backups.config.openshift.io
if kubectl get backup.velero.io app-backup -n openshift-adp >/dev/null 2>&1; then
    # Check that it targets the correct namespace
    INCLUDED_NS=$(kubectl get backup.velero.io app-backup -n openshift-adp -o jsonpath='{.spec.includedNamespaces[0]}')
    if [ "$INCLUDED_NS" = "oadp-eval-app" ]; then
        exit 0
    else
        echo "Backup does not include oadp-eval-app namespace"
        exit 1
    fi
else
    echo "Backup app-backup not found"
    exit 1
fi
