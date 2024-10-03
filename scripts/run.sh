#!/usr/bin/env bash

set -eu

cd "$(dirname "$0")/.." || exit

if [ -f .env ]; then
    source .env
fi

current_context=$(kubectl config view --raw -o json | jq -r '."current-context"' | sed "s/kind-//")
operator_image=$(KIND_CLUSTER_NAME="$current_context" KO_DOCKER_REPO=kind.local ko build -BP ./cmd/operator)
instance_image=$(KIND_CLUSTER_NAME="$current_context" KO_DOCKER_REPO=kind.local KO_DEFAULTBASEIMAGE="ghcr.io/cloudnative-pg/postgresql:17.0" ko build -BP ./cmd/instance)

(
  cd kubernetes;
  kustomize edit set image "plugin-barman-cloud=$operator_image"
  kustomize edit set secret plugin-barman-cloud "--from-literal=SIDECAR_IMAGE=$instance_image"
)

# Now we deploy the plugin inside the `cnpg-system` workspace
kubectl apply -k kubernetes/
