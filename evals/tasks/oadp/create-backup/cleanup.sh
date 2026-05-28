#!/usr/bin/env bash
kubectl delete backup.velero.io app-backup -n openshift-adp --ignore-not-found
kubectl delete namespace oadp-eval-app --ignore-not-found
