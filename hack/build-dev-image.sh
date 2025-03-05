#!/usr/bin/env sh

# This script builds the images of the barman cloud plugin, to be used
# to quickly test images in a development environment.
#
# After each run, the built images will have these names:
#
# - `plugin-barman-cloud:dev`
# - `plugin-barman-cloud-sidecar:dev`

set -eu

docker build -t plugin-barman-cloud:dev --file containers/Dockerfile.plugin .
docker build -t plugin-barman-cloud-sidecar:dev --file containers/Dockerfile.sidecar .
