---
sidebar_position: 90
---

# Miscellaneous

<!-- SPDX-License-Identifier: CC-BY-4.0 -->

## Backup Object Tagging

You can attach key-value metadata tags to backup artifacts—such as base
backups, WAL files, and history files—via the `.spec.configuration` section of
the `ObjectStore` resource.

- `tags`: applied to base backups and WAL files
- `historyTags`: applied to history files only

### Example

```yaml
apiVersion: barmancloud.cnpg.io/v1
kind: ObjectStore
metadata:
  name: my-store
spec:
  configuration:
    [...]
    tags:
      backupRetentionPolicy: "expire"
    historyTags:
      backupRetentionPolicy: "keep"
  [...]
```

## Extra Options for Backup and WAL Archiving

You can pass additional command-line arguments to `barman-cloud-backup` and
`barman-cloud-wal-archive` using the `additionalCommandArgs` field in the
`ObjectStore` configuration.

- `.spec.configuration.data.additionalCommandArgs`: for `barman-cloud-backup`
- `.spec.configuration.wal.archiveAdditionalCommandArgs`: for `barman-cloud-wal-archive`

Each field accepts a list of string arguments. If an argument is already
configured elsewhere in the plugin, the duplicate will be ignored.

### Example: Extra Backup Options

```yaml
kind: ObjectStore
metadata:
  name: my-store
spec:
  configuration:
    data:
      additionalCommandArgs:
        - "--min-chunk-size=5MB"
        - "--read-timeout=60"
```

### Example: Extra WAL Archive Options

```yaml
kind: ObjectStore
metadata:
  name: my-store
spec:
  configuration:
    wal:
      archiveAdditionalCommandArgs:
        - "--max-concurrency=1"
        - "--read-timeout=60"
```

For a complete list of supported options, refer to the
[official Barman Cloud documentation](https://docs.pgbarman.org/release/latest/).
