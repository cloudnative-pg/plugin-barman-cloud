---
sidebar_position: 5
---

# Usage

## Defining the `BarmanObjectStore`

A `BarmanObjectStore` object should be created for each object store used in
your PostgreSQL architecture. Below is an example configuration for using
MinIO:

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

The `.spec.configuration` API follows the same schema as the
[in-tree barman-cloud support](https://pkg.go.dev/github.com/cloudnative-pg/barman-cloud/pkg/api#BarmanObjectStoreConfiguration).
Refer to [the CloudNativePG documentation](https://cloudnative-pg.io/documentation/preview/backup_barmanobjectstore/)
for detailed usage.

## Configuring WAL Archiving

Once the `BarmanObjectStore` is defined, you can configure a PostgreSQL cluster
to archive WALs by referencing the store in the `.spec.plugins` section, as
shown below:

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

Once WAL archiving is enabled, the cluster is ready for backups. To create a
backup, configure the `backup.spec.pluginConfiguration` section to specify this
plugin:

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: Backup
metadata:
  name: backup-example
spec:
  method: plugin
  cluster:
    name: cluster-example
  pluginConfiguration:
    name: barman-cloud.cloudnative-pg.io
```

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

**NOTE:** The above configuration does **not** enable WAL archiving for the
restored cluster.

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
