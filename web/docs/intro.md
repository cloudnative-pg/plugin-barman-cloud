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

## Requirements

To use the Barman Cloud Plugin, you need:

- [CloudNativePG](https://cloudnative-pg.io) version **1.26** <!-- ADD WHEN 1.27 IS OUT "or later" -->
- [cert-manager](https://cert-manager.io/) to enable TLS communication between
  the plugin and the operator

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
