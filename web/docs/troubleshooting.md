---
sidebar_position: 50
---

# Troubleshooting

<!-- SPDX-License-Identifier: CC-BY-4.0 -->

This guide helps you diagnose and resolve common issues with the Barman Cloud Plugin.

## Before You Begin

### Recommended Upgrades

:::important
**CloudNativePG 1.27.0** offers significantly improved error and status reporting for plugins. If you're experiencing issues, we strongly recommend upgrading to version 1.27.0 or later for better diagnostics.

- **Upgrade CloudNativePG**: Follow the [official upgrade guide](https://cloudnative-pg.io/documentation/current/installation_upgrade)
- **Update kubectl-cnpg plugin**: Install or update the kubectl plugin for better debugging capabilities. See the [kubectl plugin documentation](https://cloudnative-pg.io/documentation/current/kubectl-plugin/)
:::

### Viewing Logs

To effectively troubleshoot issues, you need to check logs from multiple sources:

:::note
The following commands assume you've installed the CloudNativePG operator in the default `cnpg-system` namespace. If you've installed it in a different namespace, adjust the commands accordingly.
:::

```bash
# View operator logs (contains plugin interaction logs)
# Assumes operator is installed in the default cnpg-system namespace
kubectl logs -n cnpg-system deployment/cnpg-controller-manager -f

# View sidecar container logs (barman-cloud operations)
kubectl logs -n <namespace> <cluster-pod-name> -c plugin-barman-cloud -f

# View plugin manager logs
kubectl logs -n cnpg-system deployment/barman-cloud -f

# View all containers in a pod
kubectl logs -n <namespace> <cluster-pod-name> --all-containers=true

# View previous container logs (if container restarted)
kubectl logs -n <namespace> <cluster-pod-name> -c plugin-barman-cloud --previous
```

## Common Issues

### Plugin Installation Issues

#### Plugin pods not starting

**Symptoms:**
- Plugin pods are in `CrashLoopBackOff` or `Error` state
- Plugin deployment is not ready

**Possible causes and solutions:**

1. **Certificate issues**
   ```bash
   # Check if cert-manager is installed and running
   kubectl get pods -n cert-manager
   
   # Check if the plugin certificate is created
   kubectl get certificates -n cnpg-system
   ```
   
   If cert-manager is not installed, install it first:
   ```bash
   kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml
   ```

2. **Image pull errors**
   ```bash
   # Check pod events for image pull errors
   kubectl describe pod -n cnpg-system -l app.kubernetes.io/name=barman-cloud
   ```
   
   Verify the image exists and you have proper credentials if using a private registry.

3. **Resource constraints**
   ```bash
   # Check node resources
   kubectl top nodes
   kubectl describe nodes
   ```
   
   Ensure your cluster has sufficient CPU and memory resources.

### Backup Failures

#### Quick Backup Troubleshooting Checklist

When a backup fails, follow these steps in order:

1. **Check backup status**: `kubectl get backups.postgresql.cnpg.io -n <namespace>`
2. **Get error details and target pod**: 
   ```bash
   kubectl describe backups.postgresql.cnpg.io -n <namespace> <backup-name>
   # Or extract just the target pod name
   kubectl get backups.postgresql.cnpg.io -n <namespace> <backup-name> -o jsonpath='{.status.instanceID.podName}'
   ```
3. **Check the specific target pod's sidecar logs**:
   ```bash
   TARGET_POD=$(kubectl get backups.postgresql.cnpg.io -n <namespace> <backup-name> -o jsonpath='{.status.instanceID.podName}')
   kubectl logs -n <namespace> $TARGET_POD -c plugin-barman-cloud --tail=100 | grep -E "ERROR|FATAL|panic"
   ```
4. **Check cluster events**: `kubectl get events -n <namespace> --field-selector involvedObject.name=<cluster-name> --sort-by='.lastTimestamp'`
5. **Verify plugin is running**: `kubectl get pods -n cnpg-system -l app.kubernetes.io/name=barman-cloud`
6. **Check operator logs**: `kubectl logs -n cnpg-system deployment/cnpg-controller-manager --tail=100 | grep -i "backup\|plugin"`
7. **Check plugin manager logs**: `kubectl logs -n cnpg-system deployment/barman-cloud --tail=100`

#### Backup job fails immediately

**Symptoms:**
- Backup pods terminate with error
- No backup files appear in object storage
- Backup shows `failed` phase with various error messages

**Common failure modes and solutions:**

1. **"requested plugin is not available" errors**
   ```
   ERROR: requested plugin is not available: barman
   ERROR: requested plugin is not available: barman-cloud
   ERROR: requested plugin is not available: barman-cloud.cloudnative-pg.io
   ```
   
   **Cause:** The plugin name in the Cluster configuration doesn't match the deployed plugin or the plugin isn't registered
   
   **Solution:** 
   
   a. **Check plugin registration status**:
   ```bash
   # If you have kubectl-cnpg plugin installed (v1.27.0+)
   kubectl cnpg status -n <namespace> <cluster-name>
   ```
   
   Look for the "Plugins status" section:
   ```
   Plugins status
   Name                            Version  Status  Reported Operator Capabilities
   ----                            -------  ------  ------------------------------
   barman-cloud.cloudnative-pg.io  0.6.0    N/A     Reconciler Hooks, Lifecycle Service
   ```
   
   :::tip
   If the Plugins status section is missing:
   - Install or update kubectl-cnpg plugin to the latest version
   - Ensure CloudNativePG operator is v1.27.0 or later
   :::
   
   b. **Verify correct plugin name in Cluster spec**:
   ```yaml
   apiVersion: postgresql.cnpg.io/v1
   kind: Cluster
   spec:
     plugins:
       - name: barman-cloud.cloudnative-pg.io
         parameters:
           barmanObjectStore: <your-objectstore-name>
   ```
   
   c. **Check plugin deployment is running**:
   ```bash
   kubectl get deployment -n cnpg-system barman-cloud
   ```

2. **"rpc error: code = Unknown desc = panic caught: assignment to entry in nil map" errors**
   
   **Cause:** Configuration issue, often a typo or missing required field in the ObjectStore configuration
   
   **Solution:** 
   - Check the sidecar container logs for detailed error messages:
     ```bash
     kubectl logs -n <namespace> <cluster-pod> -c plugin-barman-cloud
     ```
   - Verify your ObjectStore configuration has all required fields
   - Common issues include:
     - Missing or incorrect secret references
     - Typos in configuration parameters
     - Missing required environment variables in secrets

**General debugging steps:**

1. **Check backup status and identify the target instance**
   ```bash
   # List all backups and their status
   kubectl get backups.postgresql.cnpg.io -n <namespace>
   
   # Using kubectl-cnpg plugin (if installed)
   kubectl cnpg backup list -n <namespace>
   
   # Get detailed backup information including error messages and target instance
   kubectl describe backups.postgresql.cnpg.io -n <namespace> <backup-name>
   
   # Extract the target pod name from a failed backup
   kubectl get backups.postgresql.cnpg.io -n <namespace> <backup-name> -o jsonpath='{.status.instanceID.podName}'
   
   # Or get more details including the target pod, method, phase and error
   kubectl get backups.postgresql.cnpg.io -n <namespace> <backup-name> -o jsonpath='Pod: {.status.instanceID.podName}{"\n"}Method: {.status.method}{"\n"}Phase: {.status.phase}{"\n"}Error: {.status.error}{"\n"}'
   
   # Check the cluster status for backup-related information
   kubectl cnpg status <cluster-name> -n <namespace> --verbose
   ```

2. **Check sidecar logs on the backup target pod**
   ```bash
   # First, identify which pod was the backup target (from step 1)
   TARGET_POD=$(kubectl get backups.postgresql.cnpg.io -n <namespace> <backup-name> -o jsonpath='{.status.instanceID.podName}')
   echo "Backup target pod: $TARGET_POD"
   
   # Check the sidecar logs on the specific target pod
   kubectl logs -n <namespace> $TARGET_POD -c plugin-barman-cloud --tail=100
   
   # Follow the logs in real-time to see ongoing issues
   kubectl logs -n <namespace> $TARGET_POD -c plugin-barman-cloud -f
   
   # Check for specific errors in the target pod around the backup time
   kubectl logs -n <namespace> $TARGET_POD -c plugin-barman-cloud --since=10m | grep -E "ERROR|FATAL|panic|failed"
   
   # Alternative: List all cluster pods and their roles
   kubectl get pods -n <namespace> -l cnpg.io/cluster=<cluster-name> \
     -o custom-columns=NAME:.metadata.name,ROLE:.metadata.labels.cnpg\\.io/instanceRole,INSTANCE:.metadata.labels.cnpg\\.io/instanceName
   
   # Check sidecar logs on ALL cluster pods for any errors (if target is unclear)
   for pod in $(kubectl get pods -n <namespace> -l cnpg.io/cluster=<cluster-name> -o name); do
     echo "=== Checking $pod ==="
     kubectl logs -n <namespace> $pod -c plugin-barman-cloud --tail=20 | grep -i error || echo "No errors found"
   done
   ```

3. **Check events for backup-related issues**
   ```bash
   # Check events for the cluster
   kubectl get events -n <namespace> --field-selector involvedObject.name=<cluster-name>
   
   # Check events for failed backups
   kubectl get events -n <namespace> --field-selector involvedObject.kind=Backup
   
   # Get all recent events in the namespace
   kubectl get events -n <namespace> --sort-by='.lastTimestamp' | tail -20
   ```

4. **Verify ObjectStore configuration**
   ```bash
   # Check the ObjectStore resource
   kubectl get objectstores.barmancloud.cnpg.io -n <namespace> <objectstore-name> -o yaml
   
   # Verify the secret exists and has correct keys
   kubectl get secret -n <namespace> <secret-name> -o yaml
   ```

5. **Common error messages and solutions:**

   - **"AccessDenied" or "403 Forbidden"**: Check cloud credentials and bucket permissions
   - **"NoSuchBucket"**: Verify the bucket exists and the endpoint URL is correct
   - **"Connection timeout"**: Check network connectivity and firewall rules
   - **"SSL certificate problem"**: For self-signed certificates, check CA bundle configuration

#### Backup performance issues

**Symptoms:**
- Backups take extremely long
- Backups timeout

**Plugin-specific considerations:**

1. **Check ObjectStore parallelism settings**
   - Adjust `maxParallel` in ObjectStore configuration
   - Monitor sidecar container resource usage during backups

2. **Verify plugin resource allocation**
   - Check if the sidecar container has sufficient CPU/memory
   - Review plugin container logs for resource-related warnings

:::tip
For Barman-specific features like compression, encryption, and performance tuning, refer to the [Barman documentation](https://docs.pgbarman.org/latest/).
:::

### WAL Archiving Issues

#### WAL archiving through plugin stops working

**Symptoms:**
- WAL files accumulating on primary
- Cluster warnings about WAL archiving
- Plugin sidecar logs show WAL archive errors

**Plugin-specific debugging:**

1. **Check plugin sidecar logs for WAL archiving errors**
   ```bash
   # Check recent WAL archive operations in sidecar
   kubectl logs -n <namespace> <primary-pod> -c plugin-barman-cloud --tail=50 | grep -i wal
   ```

2. **Verify the plugin is handling archive_command**
   ```bash
   # The archive_command should be routing through the plugin
   kubectl exec -n <namespace> <primary-pod> -c postgres -- psql -U postgres -c "SHOW archive_command;"
   ```

3. **Check ObjectStore configuration for WAL settings**
   - Ensure ObjectStore has proper WAL retention settings
   - Verify credentials have permissions for WAL operations

### Restore Issues

#### Restore fails during recovery

**Symptoms:**
- New cluster stuck in recovery mode
- Plugin sidecar shows restore errors
- PostgreSQL won't start

**Plugin-specific debugging:**

1. **Check plugin sidecar logs during restore**
   ```bash
   # Check the sidecar logs on the recovering cluster pods
   kubectl logs -n <namespace> <cluster-pod-name> -c plugin-barman-cloud --tail=100
   
   # Look for restore-related errors
   kubectl logs -n <namespace> <cluster-pod-name> -c plugin-barman-cloud | grep -E "restore|recovery|ERROR"
   ```

2. **Verify plugin can access backups**
   ```bash
   # Check if ObjectStore is properly configured for restore
   kubectl get objectstores.barmancloud.cnpg.io -n <namespace> <objectstore-name> -o yaml
   
   # Check PostgreSQL recovery logs
   kubectl logs -n <namespace> <cluster-pod> -c postgres | grep -i recovery
   ```

:::tip
For detailed Barman restore operations and troubleshooting, refer to the [Barman documentation](https://docs.pgbarman.org/latest/barman-cloud-restore.html).
:::

#### Point-in-time recovery (PITR) configuration issues

**Symptoms:**
- PITR target time not reached
- Plugin sidecar shows WAL access errors
- Recovery stops before target

**Plugin-specific configuration:**

1. **Verify plugin configuration for PITR**
   ```yaml
   apiVersion: postgresql.cnpg.io/v1
   kind: Cluster
   spec:
     plugins:
       - name: barman-cloud.cloudnative-pg.io
         parameters:
           barmanObjectStore: <objectstore-name>
     bootstrap:
       recovery:
         recoveryTarget:
           targetTime: "2024-01-15 10:30:00"
           targetTimezone: "UTC"
   ```

2. **Check plugin sidecar for WAL access**
   ```bash
   # Check sidecar logs during recovery for WAL-related errors
   kubectl logs -n <namespace> <cluster-pod> -c plugin-barman-cloud | grep -i wal
   ```

:::note
For detailed PITR configuration and WAL management, see the [Barman PITR documentation](https://docs.pgbarman.org/latest/).
:::

### Plugin Configuration Issues

#### Plugin cannot connect to object storage

**Symptoms:**
- Plugin sidecar logs show connection errors
- Backups fail with authentication or network errors
- ObjectStore resource shows errors

**Plugin-specific solutions:**

1. **Verify ObjectStore CRD configuration**
   ```bash
   # Check ObjectStore resource status
   kubectl get objectstores.barmancloud.cnpg.io -n <namespace> <objectstore-name> -o yaml
   
   # Verify the secret exists and has correct keys for your provider
   kubectl get secret -n <namespace> <secret-name> -o jsonpath='{.data}' | jq 'keys'
   ```

2. **Check plugin sidecar connectivity**
   ```bash
   # Check sidecar logs for connection errors
   kubectl logs -n <namespace> <cluster-pod> -c plugin-barman-cloud | grep -E "connection|timeout|SSL|certificate"
   ```

3. **Provider-specific configuration**
   - See [Object Store Configuration](object_stores.md) for provider-specific settings
   - Ensure `endpointURL` and `s3UsePathStyle` match your storage type
   - Verify network policies allow egress to your storage provider

## Diagnostic Commands

### Using kubectl-cnpg plugin

The kubectl-cnpg plugin provides enhanced debugging capabilities. Make sure you have it installed and updated:

```bash
# Install or update kubectl-cnpg plugin
kubectl krew install cnpg
# Or download directly from: https://github.com/cloudnative-pg/cloudnative-pg/releases

# Check plugin status (requires CNPG 1.27.0+)
kubectl cnpg status <cluster-name> -n <namespace>

# View cluster status in detail
kubectl cnpg status <cluster-name> -n <namespace> --verbose

# Check backup status
kubectl cnpg backup list -n <namespace>

# View plugin capabilities
kubectl cnpg plugin list -n <namespace>
```

## Getting Help

If you continue to experience issues:

1. **Check the documentation**
   - Review the [Installation Guide](installation.mdx)
   - Check [Object Store Configuration](object_stores.md) for provider-specific settings
   - Review [Usage Examples](usage.md) for correct configuration patterns

2. **Gather diagnostic information**
   ```bash
   # Create a diagnostic bundle (⚠️sanitize these before sharing!)
   kubectl get objectstores.barmancloud.cnpg.io -A -o yaml > /tmp/objectstores.yaml
   kubectl get clusters.postgresql.cnpg.io -A -o yaml > /tmp/clusters.yaml
   kubectl logs -n cnpg-system deployment/barman-cloud --tail=1000 > /tmp/plugin.log
   ```

3. **Community support**
   - CloudNativePG Slack: [#cloudnativepg-users](https://cloud-native.slack.com/messages/cloudnativepg-users)
   - GitHub Issues: [plugin-barman-cloud](https://github.com/cloudnative-pg/plugin-barman-cloud/issues)

4. **When reporting issues, include:**
   - CloudNativePG version
   - Barman Cloud Plugin version
   - Kubernetes version
   - Cloud provider and region
   - Relevant configuration (⚠️sanitize/redact sensitive information)
   - Error messages and logs
   - Steps to reproduce

## Known Issues and Limitations

### Current Known Issues

1. **WAL overwrite protection**: Unlike the in-tree Barman archiver, the plugin doesn't prevent WAL overwrites when multiple clusters share the same name and object store path ([#263](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/263))
2. **Migration compatibility**: After migrating from in-tree backup to the plugin, the `kubectl cnpg backup` command syntax has changed ([#353](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/353)):
   ```bash
   # Old command (in-tree, no longer works after migration)
   kubectl cnpg backup -n <namespace> <cluster-name> --method=barmanObjectStore
   
   # New command (plugin-based)
   kubectl cnpg backup -n <namespace> <cluster-name> --method=plugin --plugin-name=barman-cloud.cloudnative-pg.io
   ```

### Plugin Limitations

1. **Installation method**: Currently only supports manifest and Kustomize installation ([#351](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/351) - Helm chart requested)
2. **Sidecar resource sharing**: The plugin sidecar container shares pod resources with PostgreSQL
3. **Plugin restart behavior**: Restarting the sidecar container requires restarting the entire PostgreSQL pod

### Compatibility Matrix

| Plugin Version | CloudNativePG Version | Kubernetes Version | Notes |
|---------------|----------------------|-------------------|--------|
| 0.6.x         | 1.26.x, **1.27.x** (recommended) | 1.28+ | CNPG 1.27.0+ provides enhanced plugin status reporting |
| 0.5.x         | 1.25.x, 1.26.x      | 1.27+             | Limited plugin diagnostics |

:::tip
Always check the [Release Notes](https://github.com/cloudnative-pg/plugin-barman-cloud/releases) for version-specific known issues and fixes.
:::