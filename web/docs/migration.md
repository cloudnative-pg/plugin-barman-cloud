---
sidebar_position: 40
---

# Migrating from Built-in CloudNativePG Backup

<!-- SPDX-License-Identifier: CC-BY-4.0 -->

The in-tree support for Barman Cloud in CloudNativePG is **deprecated starting
from version 1.26** and will be removed in a future release.

If you're currently relying on the built-in Barman Cloud integration, you can
migrate seamlessly to the new **plugin-based architecture** using the Barman
Cloud Plugin, without data loss. Follow these steps:

- [Install the Barman Cloud Plugin](installation.mdx)
- Create an `ObjectStore` resource by translating the contents of the
  `.spec.backup.barmanObjectStore` section from your existing `Cluster`
  definition
- Modify the `Cluster` resource in a single atomic change to switch from
  in-tree backup to the plugin
- Update any `ScheduledBackup` resources to use the plugin
- Update the `externalClusters` configuration, where applicable

:::tip
For a working example, refer to [this commit](https://github.com/cloudnative-pg/cnpg-playground/commit/596f30e252896edf8f734991c3538df87630f6f7)
from the [CloudNativePG Playground project](https://github.com/cloudnative-pg/cnpg-playground),
which demonstrates a full migration.
:::

---

## Step 1: Define the `ObjectStore`

Begin by creating an `ObjectStore` resource in the same namespace as your
PostgreSQL `Cluster`.

There is a **direct mapping** between the `.spec.backup.barmanObjectStore`
section in CloudNativePG and the `.spec.configuration` field in the
`ObjectStore` CR. The conversion is mostly mechanical, with one key difference:

:::warning
In the plugin architecture, retention policies are defined as part of the `ObjectStore`.
In contrast, the in-tree implementation defined them at the `Cluster` level.
:::

If your `Cluster` used `.spec.backup.retentionPolicy`, move that configuration
to `.spec.retentionPolicy` in the `ObjectStore`.

---

### Example

Here’s an excerpt from a traditional in-tree CloudNativePG backup configuration
taken from the CloudNativePG Playground project:

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: pg-eu
spec:
  # [...]
  backup:
    barmanObjectStore:
      destinationPath: s3://backups/
      endpointURL: https://minio-eu:9443
      s3Credentials:
        accessKeyId:
          name: minio-eu
          key: ACCESS_KEY_ID
        secretAccessKey:
          name: minio-eu
          key: ACCESS_SECRET_KEY
      wal:
        compression: gzip
```

This configuration translates to the following `ObjectStore` resource for the
plugin:

```yaml
apiVersion: barmancloud.cnpg.io/v1
kind: ObjectStore
metadata:
  name: minio-eu
spec:
  configuration:
    destinationPath: s3://backups/
    endpointURL: https://minio-eu:9443
    s3Credentials:
      accessKeyId:
        name: minio-eu
        key: ACCESS_KEY_ID
      secretAccessKey:
        name: minio-eu
        key: ACCESS_SECRET_KEY
    wal:
      compression: gzip
```

As you can see, the contents of `barmanObjectStore` have been copied directly
under the `configuration` field of the `ObjectStore` resource, using the same
secret references.

## Step 2: Update the `Cluster` for plugin WAL archiving

Once the `ObjectStore` resource is in place, update the `Cluster` resource as
follows in a single atomic change:

- Remove the `.spec.backup.barmanObjectStore` section
- Remove `.spec.backup.retentionPolicy` if it was defined (as it is now in the
  `ObjectStore`)
- Remove the entire `spec.backup` section if it is now empty
- Add `barman-cloud.cloudnative-pg.io` to the `plugins` list, as described in
  [Configuring WAL archiving](usage.md#configuring-wal-archiving)

This will trigger a rolling update of the `Cluster`, switching continuous
backup from the in-tree implementation to the plugin-based approach.

### Example

The updated `pg-eu` cluster will have this configuration instead of the
previous `backup` section:

```yaml
  plugins:
  - name: barman-cloud.cloudnative-pg.io
    isWALArchiver: true
    parameters:
      barmanObjectName: minio-eu
```

---

## Step 3: Update the `ScheduledBackup`

After switching the `Cluster` to use the plugin, update your `ScheduledBackup`
resources to match.

Set the backup `method` to `plugin` and reference the plugin name via
`pluginConfiguration`, as shown in ["Performing a base backup"](usage.md#performing-a-base-backup).

### Example

Original in-tree `ScheduledBackup`:

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: ScheduledBackup
metadata:
  name: pg-eu-backup
spec:
  cluster:
    name: pg-eu
  schedule: '0 0 0 * * *'
  backupOwnerReference: self
```

Updated version using the plugin:

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: ScheduledBackup
metadata:
  name: pg-eu-backup
spec:
  cluster:
    name: pg-eu
  schedule: '0 0 0 * * *'
  backupOwnerReference: self
  method: plugin
  pluginConfiguration:
    name: barman-cloud.cloudnative-pg.io
```

---

## Step 4: Update the `externalClusters` configuration

If your `Cluster` relies on one or more external clusters that use the in-tree
Barman Cloud integration, you need to update those configurations to use the
plugin-based architecture.

When a replica cluster fetches WAL files or base backups from an external
source that used the built-in backup method, follow these steps:

1. Create a corresponding `ObjectStore` resource for the external cluster, as
   shown in [Step 1](#step-1-define-the-objectstore)
2. Update the `externalClusters` section of your replica cluster to use the
   plugin instead of the in-tree `barmanObjectStore` field

### Example

Consider the original configuration using in-tree Barman Cloud:

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: pg-us
spec:
  # [...]
  externalClusters:
  - name: pg-eu
    barmanObjectStore:
      destinationPath: s3://backups/
      endpointURL: https://minio-eu:9443
      serverName: pg-eu
      s3Credentials:
        accessKeyId:
          name: minio-eu
          key: ACCESS_KEY_ID
        secretAccessKey:
          name: minio-eu
          key: ACCESS_SECRET_KEY
      wal:
        compression: gzip
```

Create the `ObjectStore` resource for the external cluster:

```yaml
apiVersion: barmancloud.cnpg.io/v1
kind: ObjectStore
metadata:
  name: minio-eu
spec:
  configuration:
    destinationPath: s3://backups/
    endpointURL: https://minio-eu:9443
    s3Credentials:
    accessKeyId:
        name: minio-eu
        key: ACCESS_KEY_ID
    secretAccessKey:
        name: minio-eu
        key: ACCESS_SECRET_KEY
    wal:
      compression: gzip
```

Update the external cluster configuration to use the plugin:

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: pg-us
spec:
  # [...]
  externalClusters:
  - name: pg-eu
    plugin:
      name: barman-cloud.cloudnative-pg.io
      parameters:
        barmanObjectName: minio-eu
        serverName: pg-eu
```

## Step 5: Verify your metrics

When migrating from the in-core solution to the plugin-based approach, you need
to monitor a different set of metrics, as described in the
["Observability"](observability.md) section.

The table below summarizes the name changes between the old in-core metrics and
the new plugin-based ones:

| Old metric name                                  | New metric name                                                  |
| ------------------------------------------------ | ---------------------------------------------------------------- |
| `cnpg_collector_last_failed_backup_timestamp`    | `barman_cloud_cloudnative_pg_io_last_failed_backup_timestamp`    |
| `cnpg_collector_last_available_backup_timestamp` | `barman_cloud_cloudnative_pg_io_last_available_backup_timestamp` |
| `cnpg_collector_first_recoverability_point`      | `barman_cloud_cloudnative_pg_io_first_recoverability_point`      |
