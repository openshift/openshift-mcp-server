#!/usr/bin/env bash
kubectl delete restore.velero.io app-restore -n openshift-adp --ignore-not-found
kubectl delete backup.velero.io restore-source-backup -n openshift-adp --ignore-not-found
