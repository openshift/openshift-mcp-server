#!/usr/bin/env bash
kubectl delete schedule.velero.io nightly-backup -n openshift-adp --ignore-not-found
