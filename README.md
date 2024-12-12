[![CloudNativePG](./logo/cloudnativepg.png)](https://cloudnative-pg.io/)

# Barman Cloud CNPG-I plugin

**Status:** EXPERIMENTAL

Welcome to the codebase of the [barman-cloud](https://pgbarman.org/) CNPG-I
plugin for [CloudNativePG](https://cloudnative-pg.io/).

## Table of contents

- [Features](#features)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Usage](#usage)
  - [WAL Archiving](#wal-archiving)
  - [Backup](#backup)
  - [Restore](#restore)
  - [Replica clusters](#replica-clusters)

## Features

This plugin enables continuous backup to object storage for a PostgreSQL
cluster using the [barman-cloud](https://pgbarman.org/) tool suite.

The features provided by this plugin are:

- Data Directory Backup
- Data Directory Restore
- WAL Archiving
- WAL Restoring
- Point-in-Time Recovery (PITR)
- Replica Clusters

This plugin is compatible with all object storage services supported by
barman-cloud, including:

- Amazon AWS S3
- Google Cloud Storage
- Microsoft Azure Blob Storage

The following storage solutions have been tested and confirmed to work with
this implementation:

- [MinIO](https://min.io/) – An S3-compatible object storage solution.
- [Azurite](https://github.com/Azure/Azurite) – A simulator for Microsoft Azure Blob Storage.
- [fake-gcs-server](https://github.com/fsouza/fake-gcs-server) – A simulator for Google Cloud Storage.

Backups created with in-tree object store support can be restored using this
plugin, ensuring compatibility and reliability across environments.

## Prerequisites

To use this plugin, ensure the following prerequisites are met:

- [**CloudNativePG**](https://cloudnative-pg.io) version **1.25** or newer.
- [**cert-manager**](https://cert-manager.io/) for enabling **TLS communication** between the plugin and the operator.

## Installation

**IMPORTANT NOTES:**

1. The plugin **must** be installed in the same namespace where the operator is
   installed (typically `cnpg-system`).

2. Be aware that the operator's **listening namespaces** may differ from its
   installation namespace. Ensure you verify this distinction to avoid
   configuration issues.

Here’s an enhanced version of your instructions for verifying the prerequisites:

### Step 1 - Verify the Prerequisites

If CloudNativePG is installed in the default `cnpg-system` namespace, verify its version using the following command:

```sh
kubectl get deployment -n cnpg-system cnpg-controller-manager \
  | grep ghcr.io/cloudnative-pg/cloudnative-pg
```

Example output:

```output
image: ghcr.io/cloudnative-pg/cloudnative-pg:1.25.0-rc1
```

Ensure that the version displayed is **1.25** or newer.

Then, use the [cmctl](https://cert-manager.io/v1.6-docs/usage/cmctl/#installation)
tool to confirm that `cert-manager` is correctly installed:

```sh
cmctl check api
```

Example output:

```output
The cert-manager API is ready
```

Both checks are necessary to proceed with the installation.

### Step 2 - Install the barman-cloud Plugin

Use `kubectl` to apply the manifest for the latest commit in the `main` branch:

```sh
kubectl apply -f \
  https://github.com/cloudnative-pg/plugin-barman-cloud/releases/download/v0.1.0/manifest.yaml
```

Example output:

```output
customresourcedefinition.apiextensions.k8s.io/objectstores.barmancloud.cnpg.io created
serviceaccount/plugin-barman-cloud created
role.rbac.authorization.k8s.io/leader-election-role created
clusterrole.rbac.authorization.k8s.io/metrics-auth-role created
clusterrole.rbac.authorization.k8s.io/metrics-reader created
clusterrole.rbac.authorization.k8s.io/objectstore-editor-role created
clusterrole.rbac.authorization.k8s.io/objectstore-viewer-role created
clusterrole.rbac.authorization.k8s.io/plugin-barman-cloud created
rolebinding.rbac.authorization.k8s.io/leader-election-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/metrics-auth-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/plugin-barman-cloud-binding created
secret/plugin-barman-cloud-8tfddg42gf created
service/barman-cloud created
deployment.apps/barman-cloud configured
certificate.cert-manager.io/barman-cloud-client created
certificate.cert-manager.io/barman-cloud-server created
issuer.cert-manager.io/selfsigned-issuer created
```

After these steps, the plugin will be successfully installed. Make sure it is
ready to use by checking the deployment status as follows:

```sh
kubectl rollout status deployment \
  -n cnpg-system barman-cloud
```

Example output:

```output
deployment "barman-cloud" successfully rolled out
```

This confirms that the plugin is deployed and operational.

## Usage

### Defining the `BarmanObjectStore`

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

### Configuring WAL Archiving

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
    parameters:
      barmanObjectName: minio-store
  storage:
    size: 1Gi
```

This configuration enables both WAL archiving and data directory backups.

### Performing a Base Backup

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

### Restoring a Cluster

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

### Configuring Replica Clusters

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
