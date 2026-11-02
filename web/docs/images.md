---
sidebar_position: 99
---

# Container Images

<!-- SPDX-License-Identifier: CC-BY-4.0 -->

The Barman Cloud Plugin is distributed using two container images:

- One for deploying the plugin components
- One for the sidecar that runs alongside each PostgreSQL instance in a
  CloudNativePG `Cluster` using the plugin

## Plugin Container Image

The plugin image contains the logic required to operate the Barman Cloud Plugin
within your Kubernetes environment with CloudNativePG. It is published on the
GitHub Container Registry at `ghcr.io/cloudnative-pg/plugin-barman-cloud`.

This image is built from the
[`Dockerfile.plugin`](https://github.com/cloudnative-pg/plugin-barman-cloud/blob/main/containers/Dockerfile.plugin)
in the plugin repository.

## Sidecar Container Image

The sidecar image is used within each PostgreSQL pod in the cluster. It
includes the latest supported version of Barman Cloud and is responsible for
performing WAL archiving and backups on behalf of CloudNativePG.

It is available at `ghcr.io/cloudnative-pg/plugin-barman-cloud-sidecar` and is
built from the
[`Dockerfile.sidecar`](https://github.com/cloudnative-pg/plugin-barman-cloud/blob/main/containers/Dockerfile.sidecar).

These sidecar images are designed to work seamlessly with the
[`minimal` PostgreSQL container images](https://github.com/cloudnative-pg/postgres-containers?tab=readme-ov-file#minimal-images)
maintained by the CloudNativePG Community.

The image can be overridden per cluster with the
[`sidecarImage` parameter](parameters.md).

:::warning
This override is meant for debugging and testing. Running a sidecar whose
version differs from the operator's is not recommended. Leave it unset in
production.
:::
