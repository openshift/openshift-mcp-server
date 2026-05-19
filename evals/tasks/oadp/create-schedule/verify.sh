#!/usr/bin/env bash
# Verify: Check that the schedule was created with correct cron expression
if kubectl get schedule.velero.io nightly-backup -n openshift-adp >/dev/null 2>&1; then
    CRON=$(kubectl get schedule.velero.io nightly-backup -n openshift-adp -o jsonpath='{.spec.schedule}')
    if [ "$CRON" = "0 3 * * *" ]; then
        exit 0
    else
        echo "Schedule has incorrect cron expression: $CRON"
        exit 1
    fi
else
    echo "Schedule nightly-backup not found"
    exit 1
fi
