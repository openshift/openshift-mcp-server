#!/usr/bin/env bash
# Verify: Check that the test schedule exists
kubectl get schedule.velero.io eval-daily-schedule -n openshift-adp >/dev/null 2>&1
exit $?
