---
sidebar_position: 10
---

# Main Concepts

<!-- SPDX-License-Identifier: CC-BY-4.0 -->

:::important
Before proceeding, make sure to review the following sections of the
CloudNativePG documentation:

- [**Backup**](https://cloudnative-pg.io/documentation/current/backup/)
- [**WAL Archiving**](https://cloudnative-pg.io/documentation/current/wal_archiving/)
- [**Recovery**](https://cloudnative-pg.io/documentation/current/recovery/)
:::

The **Barman Cloud Plugin** enables **hot (online) backups** of PostgreSQL
clusters in CloudNativePG through [`barman-cloud`](https://pgbarman.org),
supporting continuous physical backups and WAL archiving to an **object
store**—without interrupting write operations.

It also supports both **full recovery** and **Point-in-Time Recovery (PITR)**
of a PostgreSQL cluster.

## The Object Store

At the core is the [`ObjectStore` custom resource (CRD)](plugin-barman-cloud.v1.md#objectstorespec),
which acts as the interface between the PostgreSQL cluster and the target
object storage system. It allows you to configure:

- **Authentication and bucket location** via the `.spec.configuration` section
- **WAL archiving** settings—such as compression type, parallelism, and
  server-side encryption—under `.spec.configuration.wal`
- **Base backup options**—with similar settings for compression, concurrency,
  and encryption—under `.spec.configuration.data`
- **Retention policies** to manage the life-cycle of archived WALs and backups
  via `.spec.configuration.retentionPolicy`

WAL files are archived in the `wals` directory, while base backups are stored
as **tarballs** in the `base` directory, following the
[Barman Cloud convention](https://docs.pgbarman.org/cloud/latest/usage/#object-store-layout).

:::tip
For details, refer to the
[API reference for the `ObjectStore` resource](plugin-barman-cloud.v1.md#objectstorespec).
:::

## Integration with a CloudNativePG Cluster

CloudNativePG can delegate continuous backup and recovery responsibilities to
the **Barman Cloud Plugin** by configuring the `.spec.plugins` section of a
`Cluster` resource. This setup requires a corresponding `ObjectStore` resource
to be defined.

:::important
While it is technically possible to reuse the same `ObjectStore` for multiple
`Cluster` resources within the same namespace, it is strongly recommended to
dedicate one object store per PostgreSQL cluster to ensure data isolation and
operational clarity.
:::

The following example demonstrates how to configure a CloudNativePG cluster
named `cluster-example` to use a previously defined `ObjectStore` (also named
`cluster-example`) in the same namespace. Setting `isWALArchiver: true` enables
WAL archiving through the plugin:

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: cluster-example
spec:
  # Other cluster settings...
  plugins:
    - name: barman-cloud.cloudnative-pg.io
      isWALArchiver: true
      parameters:
        barmanObjectName: cluster-example
```

## Backup of a Postgres Cluster

Once the object store is defined and the `Cluster` is configured to use the
Barman Cloud Plugin, **WAL archiving is activated immediately** on the
PostgreSQL primary.

Physical base backups are seamlessly managed by CloudNativePG using the
`Backup` and `ScheduledBackup` resources, respectively for
[on-demand](https://cloudnative-pg.io/documentation/current/backup/#on-demand-backups)
and
[scheduled](https://cloudnative-pg.io/documentation/current/backup/#scheduled-backups)
backups.

To use the Barman Cloud Plugin, you must set the `method` to `plugin` and
configure the `pluginConfiguration` section as shown:

```yaml
[...]
spec:
  method: plugin
  pluginConfiguration:
    name: barman-cloud.cloudnative-pg.io
  [...]
```

With this configuration, CloudNativePG supports:

- Backups from both **primary** and **standby** instances
- Backups from **designated primaries** in a distributed topology using
  [replica clusters](https://cloudnative-pg.io/documentation/current/replica_cluster/)

:::tip
For details on how to back up from a standby, refer to the official documentation:
[Backup from a standby](https://cloudnative-pg.io/documentation/current/backup/#backup-from-a-standby).
:::

:::important
Both backup and WAL archiving operations are executed by sidecar containers
running in the same pod as the PostgreSQL `Cluster` primary instance—except
when backups are taken from a standby, in which case the sidecar runs alongside
the standby pod.
The sidecar containers use a [dedicated container image](images.md) that
includes only the supported version of Barman Cloud.
:::

## Recovery of a Postgres Cluster

In PostgreSQL, *recovery* refers to the process of starting a database instance
from an existing backup.  The Barman Cloud Plugin integrates with CloudNativePG
to support both **full recovery** and **Point-in-Time Recovery (PITR)** from an
object store.

Recovery in this context is *not in-place*: it bootstraps a brand-new
PostgreSQL cluster from a backup and replays the necessary WAL files to reach
the desired recovery target.

To perform a recovery, define an *external cluster* that references the
appropriate `ObjectStore`, and use it as the source in the `bootstrap` section
of the target cluster:

```yaml
[...]
spec:
  [...]
  bootstrap:
    recovery:
      source: source
  externalClusters:
  - name: source
    plugin:
      name: barman-cloud.cloudnative-pg.io
      parameters:
        barmanObjectName: cluster-example
        serverName: cluster-example
  [...]
```

The critical element here is the `externalClusters` section of the `Cluster`
resource, where the `plugin` stanza instructs CloudNativePG to use the Barman
Cloud Plugin to access the object store for recovery.

This same mechanism can be used for a variety of scenarios enabled by the
CloudNativePG API, including:

* **Full cluster recovery** from the latest backup
* **Point-in-Time Recovery (PITR)**
* Bootstrapping **replica clusters** in a distributed topology

:::tip
For complete instructions and advanced use cases, refer to the official
[Recovery documentation](https://cloudnative-pg.io/documentation/current/recovery/).
:::
