---
sidebar_position: 41
---

# Resource Name Migration Guide

<!-- SPDX-License-Identifier: CC-BY-4.0 -->

:::warning
Before running the migration script or applying the manifest, please:
1. **Review the complete manifest** on the [Migration Manifest](migration-manifest.md) page to understand what changes will be made
2. **Test in a non-production environment** first if possible
3. **Ensure you have proper backups** of your cluster configuration
4. **Verify the resource names match** your current installation (default namespace is `cnpg-system`)

This migration will delete old RBAC resources and create new ones. While the operation is designed to be safe, you should review and understand the changes before proceeding. The maintainers of this project are not responsible for any issues that may arise during migration.
:::

## Overview

Starting from version 0.8.0, the plugin-barman-cloud deployment manifests use more specific, prefixed resource names to avoid conflicts with other components deployed in the same Kubernetes cluster.

## What Changed

The following resources have been renamed to use proper prefixes:

### Cluster-scoped Resources

| Old Name | New Name |
|----------|----------|
| `metrics-auth-role` | `barman-plugin-metrics-auth-role` |
| `metrics-auth-rolebinding` | `barman-plugin-metrics-auth-rolebinding` |
| `metrics-reader` | `barman-plugin-metrics-reader` |
| `objectstore-viewer-role` | `barman-plugin-objectstore-viewer-role` |
| `objectstore-editor-role` | `barman-plugin-objectstore-editor-role` |

### Namespace-scoped Resources

| Old Name | New Name | Namespace |
|----------|----------|-----------|
| `leader-election-role` | `barman-plugin-leader-election-role` | `cnpg-system` |
| `leader-election-rolebinding` | `barman-plugin-leader-election-rolebinding` | `cnpg-system` |

## Why This Change?

Using generic names for cluster-wide resources is discouraged as they may conflict with other components deployed in the same cluster. The new names make it clear that these resources belong to the barman-cloud plugin and help avoid naming collisions.

## Migration Instructions

The migration process is straightforward and can be completed with a few kubectl commands.

:::danger Verify Resources Before Deletion
**IMPORTANT**: The old resource names are generic and could potentially belong to other components in your cluster. Before deleting, verify they belong to the barman plugin by checking their labels:

```bash
# Check if the resources have the barman plugin labels
kubectl get clusterrole metrics-auth-role -o yaml | grep -A 5 "labels:"
kubectl get clusterrole metrics-reader -o yaml | grep -A 5 "labels:"
kubectl get clusterrole objectstore-viewer-role -o yaml | grep -A 5 "labels:"
kubectl get clusterrole objectstore-editor-role -o yaml | grep -A 5 "labels:"
kubectl get clusterrolebinding metrics-auth-rolebinding -o yaml | grep -A 5 "labels:"
```

Look for labels like `app.kubernetes.io/name: plugin-barman-cloud` or references to `barmancloud.cnpg.io` in the rules. If the resources don't have these indicators, **DO NOT DELETE THEM** as they may belong to another application.

If you're unsure, you can also check what the resources manage:
```bash
kubectl get clusterrole objectstore-viewer-role -o yaml
kubectl get clusterrole objectstore-editor-role -o yaml
```

These should reference `barmancloud.cnpg.io` API groups. If they don't, they are not barman plugin resources.
:::

:::tip Dry Run First
You can add `--dry-run=client` to any `kubectl delete` command to preview what would be deleted without actually removing anything.
:::

### Step 1: Delete Old Cluster-scoped Resources

**Only proceed if you've verified these resources belong to the barman plugin (see warning above).**

```bash
# Only delete if this belongs to barman plugin (check labels first)
kubectl delete clusterrole metrics-auth-role

# Only delete if this belongs to barman plugin (check labels first)
kubectl delete clusterrole metrics-reader

# Only delete if this belongs to barman plugin (check labels first)
kubectl delete clusterrole objectstore-viewer-role

# Only delete if this belongs to barman plugin (check labels first)
kubectl delete clusterrole objectstore-editor-role

# Only delete if this belongs to barman plugin (check labels first)
kubectl delete clusterrolebinding metrics-auth-rolebinding
```

If any resource is not found, that's okay - it means it was never created or already deleted.

### Step 2: Delete Old Namespace-scoped Resources

These are less likely to conflict, but you should still verify they're in the correct namespace. Replace `cnpg-system` with your namespace if different:

```bash
# First, verify these exist in your namespace
kubectl get role leader-election-role -n cnpg-system
kubectl get rolebinding leader-election-rolebinding -n cnpg-system

# Then delete them
kubectl delete role leader-election-role -n cnpg-system
kubectl delete rolebinding leader-election-rolebinding -n cnpg-system
```

### Step 3: Apply the New RBAC Manifest

Download and apply the new manifest with the updated resource names:

```bash
kubectl apply -f https://cloudnative-pg.io/plugin-barman-cloud/migration-rbac.yaml -n cnpg-system
```

Alternatively, you can copy the complete YAML from the [Migration Manifest](migration-manifest.md) page, save it to a file, and apply it locally:

```bash
kubectl apply -f barman-rbac-new.yaml -n cnpg-system
```

:::info
The new manifest will create all RBAC resources with the `barman-plugin-` prefix. Review the [Migration Manifest](migration-manifest.md) page to see exactly what will be created.
:::

## Impact

- **Downtime:** The migration requires a brief interruption as the old resources are deleted and new ones are created. The plugin controller may need to restart.
- **Permissions:** If you have custom RBAC rules or tools that reference the old resource names, they will need to be updated.
- **External Users:** If end users have been granted the `objectstore-viewer-role` or `objectstore-editor-role`, they will need to be re-granted the new role names (`barman-plugin-objectstore-viewer-role` and `barman-plugin-objectstore-editor-role`).

## Verification

After migration, verify that the new resources are created:

```bash
# Check cluster-scoped resources
kubectl get clusterrole | grep barman
kubectl get clusterrolebinding | grep barman

# Check namespace-scoped resources
kubectl get role,rolebinding -n cnpg-system | grep barman
```

You should see the new prefixed resource names.

## Troubleshooting

### Plugin Not Starting After Migration

If the plugin fails to start after migration, check:

1. **ServiceAccount permissions:** Ensure the `plugin-barman-cloud` ServiceAccount is bound to the new roles:
   ```bash
   kubectl get clusterrolebinding barman-plugin-metrics-auth-rolebinding -o yaml
   kubectl get rolebinding barman-plugin-leader-election-rolebinding -n cnpg-system -o yaml
   ```

2. **Role references:** Verify that the rolebindings reference the correct role names:
   ```bash
   kubectl describe rolebinding barman-plugin-leader-election-rolebinding -n cnpg-system
   kubectl describe clusterrolebinding barman-plugin-metrics-auth-rolebinding
   ```

### Old Resources Still Present

If old resources weren't deleted properly, you can force delete them:

```bash
kubectl delete clusterrole metrics-auth-role --ignore-not-found
kubectl delete clusterrole metrics-reader --ignore-not-found
kubectl delete clusterrole objectstore-viewer-role --ignore-not-found
kubectl delete clusterrole objectstore-editor-role --ignore-not-found
kubectl delete clusterrolebinding metrics-auth-rolebinding --ignore-not-found
kubectl delete role leader-election-role -n cnpg-system --ignore-not-found
kubectl delete rolebinding leader-election-rolebinding -n cnpg-system --ignore-not-found
```

## Support

If you encounter issues during migration, please open an issue on the [GitHub repository](https://github.com/cloudnative-pg/plugin-barman-cloud/issues).
