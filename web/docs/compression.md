---
sidebar_position: 80
---

# Compression

<!-- SPDX-License-Identifier: CC-BY-4.0 -->

By default, backups and WAL files are archived **uncompressed**. However, the
Barman Cloud Plugin supports multiple compression algorithms via
`barman-cloud-backup` and `barman-cloud-wal-archive`, allowing you to optimize
for space, speed, or a balance of both.

### Supported Compression Algorithms

- `bzip2`
- `gzip`
- `lz4` (wal only)
- `snappy`
- `xz` (wal only)
- `zstd` (wal only)

Compression settings for base backups and WAL archives are configured
independently. For implementation details, refer to the corresponding API
definitions:

- [`DataBackupConfiguration`](https://pkg.go.dev/github.com/cloudnative-pg/barman-cloud/pkg/api#DataBackupConfiguration)
- [`WALBackupConfiguration`](https://pkg.go.dev/github.com/cloudnative-pg/barman-cloud/pkg/api#WalBackupConfiguration)

:::important
Compression impacts both performance and storage efficiency. Choose the right
algorithm based on your recovery time objectives (RTO), storage capacity, and
network throughput.
:::

## Compression Benchmark (on MinIO)

| Compression | Backup Time (ms) | Restore Time (ms) | Uncompressed Size (MB) | Compressed Size (MB) | Ratio |
| ----------- | ---------------- | ----------------- | ---------------------- | -------------------- | ----- |
| None        | 10,927           | 7,553             | 395                    | 395                  | 1.0:1 |
| bzip2       | 25,404           | 13,886            | 395                    | 67                   | 5.9:1 |
| gzip        | 116,281          | 3,077             | 395                    | 91                   | 4.3:1 |
| snappy      | 8,134            | 8,341             | 395                    | 166                  | 2.4:1 |
