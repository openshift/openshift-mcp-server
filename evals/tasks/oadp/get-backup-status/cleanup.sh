#!/usr/bin/env bash
kubectl delete backup.velero.io status-check-backup -n openshift-adp --ignore-not-found
