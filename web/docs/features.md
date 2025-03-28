---
sidebar_position: 2
---

# Features

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

