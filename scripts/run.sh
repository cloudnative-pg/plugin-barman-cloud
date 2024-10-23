#!/usr/bin/env bash

set -eu

cd "$(dirname "$0")/.." || exit

if [ -f .env ]; then
    source .env
fi


MYTMPDIR="$(mktemp -d)"
trap 'rm -rf -- "$MYTMPDIR"' EXIT

current_context=$(kubectl config view --raw -o json | jq -r '."current-context"' | sed "s/kind-//")
operator_image=$(KIND_CLUSTER_NAME="$current_context" KO_DOCKER_REPO=kind.local ko build -BP ./cmd/manager)
instance_image=$(KIND_CLUSTER_NAME="$current_context" KO_DOCKER_REPO=kind.local KO_DEFAULTBASEIMAGE="ghcr.io/cloudnative-pg/postgresql:17.0" ko build -BP ./cmd/manager)

# Now we deploy the plugin inside the `cnpg-system` workspace
(
  cp -r kubernetes config "$MYTMPDIR"
  cd "$MYTMPDIR/kubernetes"
  kustomize edit set image "plugin-barman-cloud=$operator_image"
  kustomize edit set secret plugin-barman-cloud "--from-literal=SIDECAR_IMAGE=$instance_image"
  kubectl apply -k .
)
