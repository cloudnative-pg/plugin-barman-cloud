---
sidebar_position: 41
---

# Resource Name Migration Guide

<!-- SPDX-License-Identifier: CC-BY-4.0 -->

:::warning
Before proceeding with the migration process, please:
1. **Read this guide in its entirety** to understand what changes will be made
2. **Test in a non-production environment** first if possible
3. **Ensure you have proper backups** of your cluster configuration

This migration will delete old RBAC resources only after the plugin-barman-cloud upgrade. While the operation is
designed to be safe, you should review and understand the changes before proceeding. The maintainers of this project
are not responsible for any issues that may arise during migration.

**Note:** This guide assumes you are using the default `cnpg-system` namespace.
:::

## Overview

Starting from version **0.8.0**, the plugin-barman-cloud deployment manifests use more specific, prefixed resource names
to avoid conflicts with other components deployed in the same Kubernetes cluster.

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

Using generic names for cluster-wide resources is discouraged as they may conflict with other components deployed in
the same cluster. The new names make it clear that these resources belong to the barman-cloud plugin and help avoid
naming collisions.

## Migration Instructions

This three steps migration process is straightforward and can be completed with a few kubectl commands.

### Step 1: Upgrade plugin-barman-cloud

Please refer to the [Installation](installation.mdx) section to deploy the new plugin-barman-cloud release.

### Step 2: Delete Old Cluster-scoped Resources

:::danger Verify Resources Before Deletion
**IMPORTANT**: The old resource names are generic and could potentially belong to other components in your cluster.

**Before deleting each resource, verify it belongs to the barman plugin by checking:**
- For `objectstore-*` roles: Look for `barmancloud.cnpg.io` in the API groups
- For `metrics-*` roles: Check if they reference the `plugin-barman-cloud` ServiceAccount in `cnpg-system` namespace
- For other roles: Look for labels like `app.kubernetes.io/name: plugin-barman-cloud`

If a resource doesn't have these indicators, **DO NOT DELETE IT** as it may belong to another application.

Carefully review the output of each verification command before proceeding with the `delete`.
:::

:::tip Dry Run First
You can add `--dry-run=client` to any `kubectl delete` command to preview what would be deleted without actually
removing anything.
:::

**Only proceed if you've verified these resources belong to the barman plugin (see warning above).**

For each resource below, first verify it belongs to barman, then delete it:

```bash
# 1. Check metrics-auth-rolebinding FIRST (we'll check the role after)
# Look for references to plugin-barman-cloud ServiceAccount
kubectl describe clusterrolebinding metrics-auth-rolebinding
# If it references plugin-barman-cloud ServiceAccount in cnpg-system namespace, delete it:
kubectl delete clusterrolebinding metrics-auth-rolebinding

# 2. Check metrics-auth-role
# Look for references to authentication.k8s.io and authorization.k8s.io
kubectl describe clusterrole metrics-auth-role
# Verify it's not being used by any other rolebindings:
kubectl get clusterrolebinding -o json | jq -r '.items[] | select(.roleRef.name=="metrics-auth-role") | .metadata.name'
# If the above returns nothing (role is not in use) and the role looks like the barman one, delete it (see warnings section):
kubectl delete clusterrole metrics-auth-role

# 3. Check objectstore-viewer-role
# Look for barmancloud.cnpg.io API group or app.kubernetes.io/name: plugin-barman-cloud label
kubectl describe clusterrole objectstore-viewer-role
# If it shows barmancloud.cnpg.io in API groups, delete it:
kubectl delete clusterrole objectstore-viewer-role

# 4. Check objectstore-editor-role
# Look for barmancloud.cnpg.io API group or app.kubernetes.io/name: plugin-barman-cloud label
kubectl describe clusterrole objectstore-editor-role
# If it shows barmancloud.cnpg.io in API groups, delete it:
kubectl delete clusterrole objectstore-editor-role

# 5. Check metrics-reader (MOST DANGEROUS - very generic name)
# First, check if it's being used by any rolebindings OTHER than barman's:
kubectl get clusterrolebinding -o json | jq -r '.items[] | select(.roleRef.name=="metrics-reader") | "\(.metadata.name) -> \(.subjects[0].name) in \(.subjects[0].namespace)"'
# If this shows ANY rolebindings, review them carefully. Only proceed if they're all barman-related.
# Then check the role itself:
kubectl describe clusterrole metrics-reader
# If it ONLY has nonResourceURLs: /metrics and NO other rolebindings use it, delete it:
kubectl delete clusterrole metrics-reader
```

:::warning
The `metrics-reader` role is particularly dangerous to delete blindly. Many monitoring systems use this exact name. Only delete it if:
1. You've verified it ONLY grants access to `/metrics`
2. No other rolebindings reference it (checked with the jq command above)
3. You're certain it was created by the barman plugin

If you're unsure, it's safer to leave it and let the new `barman-plugin-metrics-reader` role coexist with it.
:::

If any resource is not found during the `describe` command, that's okay - it means it was never created or already deleted. Simply skip the delete command for that resource.

### Step 3: Delete Old Namespace-scoped Resources

Delete the old namespace-scoped resources in the `cnpg-system` namespace:

```bash
# Delete the old leader-election resources
kubectl delete role leader-election-role -n cnpg-system
kubectl delete rolebinding leader-election-rolebinding -n cnpg-system
```

If any resource is not found, that's okay - it means it was never created or already deleted.

## Impact

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

## Support

If you encounter issues during migration, please open an issue on the [GitHub repository](https://github.com/cloudnative-pg/plugin-barman-cloud/issues).
