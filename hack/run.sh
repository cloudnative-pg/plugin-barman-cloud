#!/usr/bin/env bash
##
## Copyright © contributors to CloudNativePG, established as
## CloudNativePG a Series of LF Projects, LLC.
##
## Licensed under the Apache License, Version 2.0 (the "License");
## you may not use this file except in compliance with the License.
## You may obtain a copy of the License at
##
##     http://www.apache.org/licenses/LICENSE-2.0
##
## Unless required by applicable law or agreed to in writing, software
## distributed under the License is distributed on an "AS IS" BASIS,
## WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
## See the License for the specific language governing permissions and
## limitations under the License.
##
## SPDX-License-Identifier: Apache-2.0
##

set -eu

cd "$(dirname "$0")/.." || exit

if [ -f .env ]; then
    source .env
fi


MYTMPDIR="$(mktemp -d)"
trap 'rm -rf -- "$MYTMPDIR"' EXIT

current_context=$(kubectl config view --raw -o json | jq -r '."current-context"' | sed "s/kind-//")
operator_image=$(KIND_CLUSTER_NAME="$current_context" KO_DOCKER_REPO=kind.local ko build -BP ./cmd/manager)
instance_image=$(KIND_CLUSTER_NAME="$current_context" KO_DOCKER_REPO=kind.local KO_DEFAULTBASEIMAGE="ghcr.io/cloudnative-pg/postgresql:18.0" ko build -BP ./cmd/manager)

# Now we deploy the plugin inside the `cnpg-system` workspace
(
  cp -r kubernetes config "$MYTMPDIR"
  cd "$MYTMPDIR/kubernetes"
  kustomize edit set image "plugin-barman-cloud=$operator_image"
  kustomize edit set secret plugin-barman-cloud "--from-literal=SIDECAR_IMAGE=$instance_image"
  kubectl apply -k .
)
