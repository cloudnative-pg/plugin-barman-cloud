---
sidebar_position: 55
---

# Observability

<!-- SPDX-License-Identifier: CC-BY-4.0 -->

The Barman Cloud Plugin exposes the following metrics through the native
Prometheus exporter of the instance manager:

- `barman_cloud_cloudnative_pg_io_last_failed_backup_timestamp`:
  the UNIX timestamp of the most recent failed backup.

- `barman_cloud_cloudnative_pg_io_last_available_backup_timestamp`:
  the UNIX timestamp of the most recent successfully available backup.

- `barman_cloud_cloudnative_pg_io_first_recoverability_point`:
  the UNIX timestamp representing the earliest point in time from which the
  cluster can be recovered.

These metrics supersede the previously available in-core metrics that used the
`cnpg_collector` prefix. The new metrics are exposed under the
`barman_cloud_cloudnative_pg_io` prefix instead.
