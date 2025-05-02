---
sidebar_position: 60
---

# Retention Policies

<!-- SPDX-License-Identifier: CC-BY-4.0 -->

The Barman Cloud Plugin supports **automated cleanup of obsolete backups** via
retention policies, configured in the `.spec.retentionPolicy` field of the
`ObjectStore` resource.

:::note
This feature uses the `barman-cloud-backup-delete` command with the
`--retention-policy "RECOVERY WINDOW OF {{ value }} {{ unit }}"` syntax.
:::

#### Example: 30-Day Retention Policy

```yaml
apiVersion: barmancloud.cnpg.io/v1
kind: ObjectStore
metadata:
  name: my-store
spec:
  [...]
  retentionPolicy: "30d"
````

:::note
A **recovery window retention policy** ensures the cluster can be restored to
any point in time between the calculated *Point of Recoverability* (PoR) and
the latest WAL archive. The PoR is defined as `current time - recovery window`.
The **first valid backup** is the most recent backup completed before the PoR.
Backups older than that are marked as *obsolete* and deleted after the next
backup completes.
:::

