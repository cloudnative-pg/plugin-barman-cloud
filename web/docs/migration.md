---
sidebar_position: 40
---

# Migrating from Built-in CloudNativePG Backup

<!-- SPDX-License-Identifier: CC-BY-4.0 -->

The in-tree support for Barman Cloud in CloudNativePG is **deprecated starting
from version 1.26** and will be removed in a future release.

If you're currently relying on the built-in Barman Cloud integration, you can
migrate seamlessly to the new **plugin-based architecture** using the Barman
Cloud Plugin, with **no downtime**. Follow these steps:

- [Install the Barman Cloud Plugin](installation.md)
- Create an `ObjectStore` resource by translating the contents of the
  `.spec.backup.barmanObjectStore` section from your existing `Cluster`
  definition
- Modify the `Cluster` resource in a single atomic change to switch from
  in-tree backup to the plugin
- Update any `ScheduledBackup` resources to use the plugin

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
      endpointURL: http://minio-eu:9000
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
    endpointURL: http://minio-eu:9000
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

## Step 2: Update the `Cluster`

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
