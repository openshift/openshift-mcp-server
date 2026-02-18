#!/usr/bin/env bash
kubectl delete backup.velero.io eval-test-backup -n openshift-adp --ignore-not-found
