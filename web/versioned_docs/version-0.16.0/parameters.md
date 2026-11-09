---
sidebar_position: 100
---

# Parameters

<!-- SPDX-License-Identifier: CC-BY-4.0 -->

The following parameters are available for the Barman Cloud Plugin:

- `barmanObjectName`: references the `ObjectStore` resource to be used by the
  plugin.
- `serverName`: Specifies the server name in the object store.
- `sidecarImage`: overrides the [sidecar container image](images.md#sidecar-container-image)
  injected into the instance pods and the recovery job pods of the cluster.
  It can also be set on a recovery or replica source in `externalClusters[]`.

:::warning
`sidecarImage` is meant for debugging and testing. Running a sidecar whose
version differs from the operator's is not recommended. Leave it unset in
production.
:::

:::important
The `serverName` parameter in the `ObjectStore` resource is retained solely for
API compatibility with the in-tree `barmanObjectStore` and must always be left empty.
When needed, use the `serverName` plugin parameter in the Cluster configuration instead.
:::
