---
sidebar_position: 30
---

# Using the Barman Cloud Plugin

<!-- SPDX-License-Identifier: CC-BY-4.0 -->

After [installing the plugin](installation.mdx) in the same namespace as the
CloudNativePG operator, enabling your PostgreSQL cluster to use the Barman
Cloud Plugin involves just a few steps:

- Defining the object store containing your WAL archive and base backups, using
  your preferred [provider](object_stores.md)
- Instructing the Postgres cluster to use the Barman Cloud Plugin

From that moment, you’ll be able to issue on-demand backups or define a backup
schedule, as well as rely on the object store for recovery operations.

The rest of this page details each step, using MinIO as object store provider.

## Defining the `ObjectStore`

An `ObjectStore` resource must be created for each object store used in your
PostgreSQL architecture. Here's an example configuration using MinIO:

```yaml
apiVersion: barmancloud.cnpg.io/v1
kind: ObjectStore
metadata:
  name: minio-store
spec:
  configuration:
    destinationPath: s3://backups/
    endpointURL: http://minio:9000
    s3Credentials:
      accessKeyId:
        name: minio
        key: ACCESS_KEY_ID
      secretAccessKey:
        name: minio
        key: ACCESS_SECRET_KEY
    wal:
      compression: gzip
```

The `.spec.configuration` schema follows the same format as the
[in-tree barman-cloud support](https://pkg.go.dev/github.com/cloudnative-pg/barman-cloud/pkg/api#BarmanObjectStoreConfiguration).
Refer to [the CloudNativePG documentation](https://cloudnative-pg.io/documentation/preview/backup_barmanobjectstore/)
for additional details.

:::important
The `serverName` parameter in the `ObjectStore` resource is retained solely for
API compatibility with the in-tree `barmanObjectStore` and must always be left empty.
When needed, use the `serverName` plugin parameter in the Cluster configuration instead.
:::

## Configuring WAL Archiving

Once the `ObjectStore` is defined, you can configure your PostgreSQL cluster
to archive WALs by referencing the store in the `.spec.plugins` section:

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: cluster-example
spec:
  instances: 3
  imagePullPolicy: Always
  plugins:
  - name: barman-cloud.cloudnative-pg.io
    isWALArchiver: true
    parameters:
      barmanObjectName: minio-store
  storage:
    size: 1Gi
```

This configuration enables both WAL archiving and data directory backups.

## Performing a Base Backup

Once WAL archiving is enabled, the cluster is ready for backups. Backups can be
created either declaratively (with YAML manifests) or imperatively (with the
`cnpg` plugin).

### Declarative approach (YAML manifest)

Create a backup resource by applying a YAML manifest:

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: Backup
metadata:
  name: backup-example
spec:
  cluster:
    name: cluster-example
  method: plugin
  pluginConfiguration:
    name: barman-cloud.cloudnative-pg.io
```

### Imperative approach (using the `cnpg` plugin)

The quickest way to trigger an on-demand backup is with the `cnpg` plugin:

```bash
kubectl cnpg backup -n <namespace> <cluster-name> \
  --method=plugin \
  --plugin-name=barman-cloud.cloudnative-pg.io
```

:::note Migration from in-tree backups
If you are migrating from the in-tree backup system, note the change in syntax:

```bash
# Old command (in-tree backup)
kubectl cnpg backup -n <namespace> <cluster-name> --method=barmanObjectStore

# New command (plugin-based backup)
kubectl cnpg backup -n <namespace> <cluster-name> \
  --method=plugin \
  --plugin-name=barman-cloud.cloudnative-pg.io
```
:::

## Restoring a Cluster

To restore a cluster from an object store, create a new `Cluster` resource that
references the store containing the backup. Below is an example configuration:

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: cluster-restore
spec:
  instances: 3
  imagePullPolicy: IfNotPresent
  bootstrap:
    recovery:
      source: source
  externalClusters:
  - name: source
    plugin:
      name: barman-cloud.cloudnative-pg.io
      parameters:
        barmanObjectName: minio-store
        serverName: cluster-example
  storage:
    size: 1Gi
```

:::important
The above configuration does **not** enable WAL archiving for the restored cluster.
:::

To enable WAL archiving for the restored cluster, include the `.spec.plugins`
section alongside the `externalClusters.plugin` section, as shown below:

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: cluster-restore
spec:
  instances: 3
  imagePullPolicy: IfNotPresent
  bootstrap:
    recovery:
      source: source
  plugins:
  - name: barman-cloud.cloudnative-pg.io
    isWALArchiver: true
    parameters:
      # Backup Object Store (push, read-write)
      barmanObjectName: minio-store-bis
  externalClusters:
  - name: source
    plugin:
      name: barman-cloud.cloudnative-pg.io
      parameters:
        # Recovery Object Store (pull, read-only)
        barmanObjectName: minio-store
        serverName: cluster-example
  storage:
    size: 1Gi
```

The same object store may be used for both transaction log archiving and
restoring a cluster, or you can configure separate stores for these purposes.

## Configuring Replica Clusters

You can set up a distributed topology by combining the previously defined
configurations with the `.spec.replica` section. Below is an example of how to
define a replica cluster:

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: cluster-dc-a
spec:
  instances: 3
  primaryUpdateStrategy: unsupervised

  storage:
    storageClass: csi-hostpath-sc
    size: 1Gi

  plugins:
  - name: barman-cloud.cloudnative-pg.io
    isWALArchiver: true
    parameters:
      barmanObjectName: minio-store-a

  replica:
    self: cluster-dc-a
    primary: cluster-dc-a
    source: cluster-dc-b

  externalClusters:
  - name: cluster-dc-a
    plugin:
      name: barman-cloud.cloudnative-pg.io
      parameters:
        barmanObjectName: minio-store-a

  - name: cluster-dc-b
    plugin:
      name: barman-cloud.cloudnative-pg.io
      parameters:
        barmanObjectName: minio-store-b
```

## Configuring the plugin instance sidecar

The Barman Cloud Plugin runs as a sidecar container next to each PostgreSQL
instance pod. It manages backup, WAL archiving, and restore processes.

Configuration comes from multiple `ObjectStore` resources:

1. The one referenced in the
   `.spec.plugins` section of the `Cluster`. This is the
   object store used for WAL archiving and base backups.
2. The one referenced in the external cluster
   used in the `.spec.replica.source` section of the `Cluster`. This is
   used by the log-shipping designated primary to get the WAL files.
3. The one referenced in the
   `.spec.bootstrap.recovery.source` section of the `Cluster`. Used by
   the initial recovery job to create the cluster from an existing backup.

You can fine-tune sidecar behavior in the `.spec.instanceSidecarConfiguration`
of your ObjectStore. These settings apply to all PostgreSQL instances that use
this object store. Any updates take effect at the next `Cluster` reconciliation,
and could generate a rollout of the `Cluster`.

```yaml
apiVersion: barmancloud.cnpg.io/v1
kind: ObjectStore
metadata:
  name: minio-store
spec:
  configuration:
  # [...]
  instanceSidecarConfiguration:
    retentionPolicyIntervalSeconds: 1800
    resources:
      requests:
        memory: "XXX"
        cpu: "YYY"
      limits:
        memory: "XXX"
        cpu: "YYY"
```

:::note
If more than one `ObjectStore` applies, the `instanceSidecarConfiguration` of
the one set in `.spec.plugins` has priority.
:::
