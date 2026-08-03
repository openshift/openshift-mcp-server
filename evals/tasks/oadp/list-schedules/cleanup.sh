#!/usr/bin/env bash
kubectl delete schedule.velero.io eval-daily-schedule -n openshift-adp --ignore-not-found
