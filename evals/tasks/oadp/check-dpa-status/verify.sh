#!/usr/bin/env bash
# Verify: Check that DPA CRD is accessible (agent should have called oadp_dpa with action: list or get)
kubectl get crd dataprotectionapplications.oadp.openshift.io >/dev/null 2>&1
exit $?
