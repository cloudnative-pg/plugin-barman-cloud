#!/usr/bin/env bash

set -eu

cd "$(dirname "$0")/.." || exit

kubectl delete clusters --all
kubectl delete backups --all
kubectl exec -ti mc -- mc rm -r --force minio/backups