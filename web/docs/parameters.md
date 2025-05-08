---
sidebar_position: 100
---

# Parameters

<!-- SPDX-License-Identifier: CC-BY-4.0 -->

The following parameters are available for the Barman Cloud Plugin:

- `barmanObjectName`: references the `ObjectStore` resource to be used by the
  plugin.
- `serverName`: Specifies the server name in the object store.

:::important
  The `serverName` parameter within the `ObjectStore` resource exists only for
  API compatibility with the in-tree `barmanObjectStore` and is ignored by the
  plugin. Use the `serverName` plugin parameter when needed.
