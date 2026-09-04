---
sidebar_position: 100
---

# Parameters

<!-- SPDX-License-Identifier: CC-BY-4.0 -->

The following parameters are available for the Barman Cloud Plugin:

- `barmanObjectName`: references the `ObjectStore` resource to be used by the
  plugin.
- `serverName`: the archive namespace used under the ObjectStore
  `destinationPath` for base backups and WAL files. If omitted, it defaults to
  the Cluster name. Change this value when you need a separate archive path,
  for example during a
  [PostgreSQL major upgrade](usage.md#archive-path-separation-for-postgresql-major-upgrades).

:::important
The `serverName` parameter in the `ObjectStore` resource is retained solely for
API compatibility with the in-tree `barmanObjectStore` and must always be left empty.
When needed, use the `serverName` plugin parameter in the Cluster configuration instead.
:::
