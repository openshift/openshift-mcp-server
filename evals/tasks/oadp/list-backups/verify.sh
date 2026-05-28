#!/usr/bin/env bash
# Verify: Check that backups were listed (the agent should have called oadp_backup with action: list)
# The success of this task is validated by the LLM judge checking tool usage patterns
# This script just verifies the test backup still exists
# Note: Must use backup.velero.io to avoid conflict with backups.config.openshift.io
kubectl get backup.velero.io eval-test-backup -n openshift-adp >/dev/null 2>&1
exit $?
