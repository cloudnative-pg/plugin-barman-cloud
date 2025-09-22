---
sidebar_position: 1
sidebar_label: "Introduction"
---

# Barman Cloud Plugin

<!-- SPDX-License-Identifier: CC-BY-4.0 -->

The **Barman Cloud Plugin** for [CloudNativePG](https://cloudnative-pg.io/)
enables online continuous physical backups of PostgreSQL clusters to object storage
using the `barman-cloud` suite from the [Barman](https://docs.pgbarman.org/release/latest/)
project.

:::important
If you plan to migrate your existing CloudNativePG cluster to the new
plugin-based approach using the Barman Cloud Plugin, see
["Migrating from Built-in CloudNativePG Backup"](migration.md)
for detailed instructions.
:::

## Requirements

Before using the Barman Cloud Plugin, ensure that the following components are
installed and properly configured:

- [CloudNativePG](https://cloudnative-pg.io) version 1.26 or later

  - We strongly recommend version 1.27.0 or later, which includes improved
    error handling and status reporting for the plugin.
  - If you are running an earlier release, refer to the
    [upgrade guide](https://cloudnative-pg.io/documentation/current/installation_upgrade).

- [cert-manager](https://cert-manager.io/)

  - The recommended way to enable secure TLS communication between the plugin
    and the operator.
  - Alternatively, you can provide your own certificate bundles. See the
    [CloudNativePG documentation on TLS configuration](https://cloudnative-pg.io/documentation/current/cnpg_i/#configuring-tls-certificates).

- [`kubectl-cnpg`](https://cloudnative-pg.io/documentation/current/kubectl-plugin/)
  plugin (optional but recommended)

  - Simplifies debugging and monitoring with additional status and inspection
    commands.
  - Multiple installation options are available in the
    [installation guide](https://cloudnative-pg.io/documentation/current/kubectl-plugin/#install).

## Key Features

This plugin provides the following capabilities:

- Physical online backup of the data directory
- Physical restore of the data directory
- Write-Ahead Log (WAL) archiving
- WAL restore
- Full cluster recovery
- Point-in-Time Recovery (PITR)
- Seamless integration with replica clusters for bootstrap and WAL restore from archive

:::important
The Barman Cloud Plugin is designed to **replace the in-tree object storage support**
previously provided via the `.spec.backup.barmanObjectStore` section in the
`Cluster` resource.
Backups created using the in-tree approach are fully supported and compatible
with this plugin.
:::

## Supported Object Storage Providers

The plugin works with all storage backends supported by `barman-cloud`, including:

- **Amazon S3**
- **Google Cloud Storage**
- **Microsoft Azure Blob Storage**

In addition, the following S3-compatible and simulator solutions have been
tested and verified:

- [MinIO](https://min.io/) – An S3-compatible storage solution
- [Azurite](https://github.com/Azure/Azurite) – A simulator for Azure Blob Storage
- [fake-gcs-server](https://github.com/fsouza/fake-gcs-server) – A simulator for Google Cloud Storage

:::tip
For more details, refer to [Object Store Providers](object_stores.md).
:::
