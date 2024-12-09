[![CloudNativePG](./logo/cloudnativepg.png)](https://cloudnative-pg.io/)

# Barman Cloud CNPG-i plugin

Welcome to the codebase of the [barman-cloud](https://pgbarman.org/) CNPG-i plugin.

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

This plugin configures continuous backup to Object stores on a PostgreSQL
cluster, using the [barman-cloud](https://pgbarman.org/) tool suite.

The features implemented by this plugin are:

* Data directory backup
* Data directory restore
* WAL archiving
* WAL restoring
* Point-in-time recovery
* Replica clusters

This plugin supports every object store supported by barman-cloud:

* Amazon AWS S3
* Google Cloud Storage
* Microsoft Azure Blob Storage

The implementation is currently tested with the following storage backends:

* [MinIO](https://min.io/) - compatible with AWS S3
* [Azurite](https://github.com/Azure/Azurite) - simulate most of the Azure Blob
  Storage
* [fake-gcs-server](https://github.com/fsouza/fake-gcs-server) - simulate Google Cloud

Backups taken by the in-tree object store support can be restored with this
plugin.

## Prerequisites

* [CloudNativePG](https://cloudnative-pg.io) 1.25RC1 or newer
* [cert-manager](https://cert-manager.io/)

## Installation

**IMPORTANT** The plugin should be installed in the same namespace where the
operator is **installed**, usually `cnpg-system`.

**IMPORTANT** This may be different from the namespaces the operator is
listening on.

### Step 1 - verify the prerequisites are met

Supposing that CloudNativePG is installed in the default `cnpg-system`
namespace, the current version can be verified with:

```sh
$ kubectl get deployment -n cnpg-system cnpg-controller-manager | grep ghcr.io/cloudnative-pg/cloudnative-pg
```
```output
image: ghcr.io/cloudnative-pg/cloudnative-pg:1.25.0-rc1
```

Please ensure you're using CloudNativePG 1.25-rc1 or newer.

The [cert-manager](https://cert-manager.io) installation can be verified by
using the [cmctl](https://cert-manager.io/v1.6-docs/usage/cmctl/#installation)
tool:

```sh
$ cmctl check api
```
```output
The cert-manager API is ready
```

### Step 2 - install the barman-cloud plugin

The plugin can be installed via its manifest:

<!--
TODO: replace this with the latest manifest as archived in the latest commit of main
-->

```sh
# Download the plugin-barman-cloud codebase, including its manifest
$ curl -Lo plugin-barman-cloud.tgz https://api.github.com/repos/cloudnative-pg/plugin-barman-cloud/tarball/main

# Expand it in a temporary folder (this can be deleted after the plugin is installed)
$ tar xvzf plugin-barman-cloud.tgz

# From now on, the proposed commands are supposed to be invoked from
# the repository root directory
$ cd cloudnative-pg-plugin-barman-cloud*

# Apply the manifest for the latest commit in the `main` branch
$ kubectl apply -k kubernetes/
```
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

Once the plugin is installed, the following command will wait until the plugin
is ready to be used:

```sh
$ kubectl rollout status deployment -n cnpg-system barman-cloud
```
```output
deployment "barman-cloud" successfully rolled out
```

## Usage

### The BarmanObjectStore object

A BarmanObjectStore object should be created for each object stored used by the
PostgreSQL architecture. This is an example:

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

The `objectstore.spec.configuration` API is the same api used by the [in-tree
barman-cloud
support](https://pkg.go.dev/github.com/cloudnative-pg/barman-cloud/pkg/api#BarmanObjectStoreConfiguration)
and can be used like discussed in [the relative documentation
page](https://cloudnative-pg.io/documentation/preview/backup_barmanobjectstore/).

### WAL Archiving

Once the `BarmanObjectStore` has been defined, a cluster using it to archive
WALs will reference it in the `.spec.plugins` section, like in the following
example:

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

This will enable WAL archiving and data directory backups, as discussed in the
next section.

### Backup

Once the transaction log files are being archived, the cluster is ready to be
backed up.

To request a backup, the `backup.spec.pluginConfiguration` stanza must be set
with the name of this plugin like in the following example:

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

### Restore

To recover a cluster from an object store, the user should create a new
`Cluster` resource referring to the object store containing the backup, like in
the following example:

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

**IMPORTANT** recovering a cluster like in the previous example do not enable
WAL archiving for the cluster being recovered.

The latter can be configured by combining the `.spec.plugins` section with the
`externalClusters.plugin` section, like in the following example:

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
      barmanObjectName: minio-store-bis

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

The object store that is used to archive the transaction log may be the same
object store that is being used to restore a cluster or a different one.

### Replica clusters

The previous definition can be combined to setup a distributed topology using
the `.spec.replica` section like in the following example:

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
