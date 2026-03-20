---
sidebar_position: 90
---

# Troubleshooting

<!-- SPDX-License-Identifier: CC-BY-4.0 -->

This guide helps you diagnose and resolve common issues with the Barman Cloud
plugin.

:::important
We are continuously improving the integration between CloudNativePG and the
Barman Cloud plugin as it moves toward greater stability and maturity. For this
reason, we recommend using the latest available version of both components.
See the [*Requirements* section](intro.md#requirements) for details.
:::

:::note
The following commands assume you installed the CloudNativePG operator in
the default `cnpg-system` namespace. If you installed it in a different
namespace, adjust the commands accordingly.
:::

## Viewing Logs

To troubleshoot effectively, you’ll often need to review logs from multiple
sources:

```sh
# View operator logs (includes plugin interaction logs)
kubectl logs -n cnpg-system deployment/cnpg-controller-manager -f

# View plugin manager logs
kubectl logs -n cnpg-system deployment/barman-cloud -f

# View sidecar container logs (Barman Cloud operations)
kubectl logs -n <namespace> <cluster-pod-name> -c plugin-barman-cloud -f

# View all containers in a pod
kubectl logs -n <namespace> <cluster-pod-name> --all-containers=true

# View previous container logs (if container restarted)
kubectl logs -n <namespace> <cluster-pod-name> -c plugin-barman-cloud --previous
```

## Common Issues

### Plugin Installation Issues

#### Plugin pods not starting

**Symptoms:**

- Plugin pods stuck in `CrashLoopBackOff` or `Error`
- Plugin deployment not ready

**Possible causes and solutions:**

1. **Certificate issues**

   ```sh
   # Check if cert-manager is installed and running
   kubectl get pods -n cert-manager

   # Check if the plugin certificate is created
   kubectl get certificates -n cnpg-system
   ```

   If cert-manager is not installed, install it first:

   ```sh
   # Note: other installation methods for cert-manager are available
   kubectl apply -f \
     https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml
   ```

   If you are using your own certificates without cert-manager, you will need
   to verify the entire certificate chain yourself.


2. **Image pull errors**

   ```sh
   # Check pod events for image pull errors
   kubectl describe pod -n cnpg-system -l app=barman-cloud
   ```

   Verify the image exists and you have proper credentials if using a private
   registry.


3. **Resource constraints**

   ```sh
   # Check node resources
   kubectl top nodes
   kubectl describe nodes
   ```

   Make sure your cluster has sufficient CPU and memory resources.

### Backup Failures

#### Quick Backup Troubleshooting Checklist

When a backup fails, follow these steps in order:

1. **Check backup status**:

   ```sh
   kubectl get backups.postgresql.cnpg.io -n <namespace>
   ```
2. **Get error details and target pod**:

   ```sh
   kubectl describe backups.postgresql.cnpg.io \
     -n <namespace> <backup-name>

   kubectl get backups.postgresql.cnpg.io \
     -n <namespace> <backup-name> \
     -o jsonpath='{.status.instanceID.podName}'
   ```
3. **Check the target pod’s sidecar logs**:

   ```sh
   TARGET_POD=$(kubectl get backups.postgresql.cnpg.io \
     -n <namespace> <backup-name> \
     -o jsonpath='{.status.instanceID.podName}')

   kubectl logs \
     -n <namespace> $TARGET_POD -c plugin-barman-cloud \
     --tail=100 | grep -E "ERROR|FATAL|panic"
   ```
4. **Check cluster events**:

   ```sh
   kubectl get events -n <namespace> \
     --field-selector involvedObject.name=<cluster-name> \
     --sort-by='.lastTimestamp'
   ```
5. **Verify plugin is running**:

   ```sh
   kubectl get pods \
     -n cnpg-system -l app=barman-cloud
   ```
6. **Check operator logs**:

   ```sh
   kubectl logs \
     -n cnpg-system deployment/cnpg-controller-manager \
     --tail=100 | grep -i "backup\|plugin"
   ```
7. **Check plugin manager logs**:

   ```sh
   kubectl logs \
     -n cnpg-system deployment/barman-cloud --tail=100
   ```

#### Backup job fails immediately

**Symptoms:**

- Backup pods terminate with error
- No backup files appear in object storage
- Backup shows `failed` phase with various error messages

**Common failure modes and solutions:**

1. **"requested plugin is not available" errors**

   ```
   requested plugin is not available: barman
   requested plugin is not available: barman-cloud
   requested plugin is not available: barman-cloud.cloudnative-pg.io
   ```
   
   **Cause:** The plugin name in the Cluster configuration doesn’t match the
   deployed plugin, or the plugin isn’t registered.
   
   **Solution:** 
   
   a. **Check plugin registration:**

   ```sh
   # If you have the `cnpg` plugin installed (v1.27.0+)
   kubectl cnpg status -n <namespace> <cluster-name>
   ```
   
   Look for the "Plugins status" section:
   ```
   Plugins status
   Name                            Version  Status  Reported Operator Capabilities
   ----                            -------  ------  ------------------------------
   barman-cloud.cloudnative-pg.io  0.6.0    N/A     Reconciler Hooks, Lifecycle Service
   ```
   
   b. **Verify plugin name in `Cluster` spec**:

   ```yaml
   apiVersion: postgresql.cnpg.io/v1
   kind: Cluster
   spec:
     plugins:
       - name: barman-cloud.cloudnative-pg.io
         parameters:
           barmanObjectName: <your-objectstore-name>
   ```
   
   c. **Check plugin deployment is running**:

   ```sh
   kubectl get deployment -n cnpg-system barman-cloud
   ```

2. **"rpc error: code = Unknown desc = panic caught: assignment to entry in nil map" errors**
   
   **Cause:** Misconfiguration in the `ObjectStore` (e.g., typo or missing field).
   
   **Solution:** 

   - Review sidecar logs for details
   - Verify `ObjectStore` configuration and secrets
   - Common issues include:
     - Missing or incorrect secret references
     - Typos in configuration parameters
     - Missing required environment variables in secrets

#### Backup performance issues

**Symptoms:**

- Backups take extremely long
- Backups timeout

**Plugin-specific considerations:**

1. **Check `ObjectStore` parallelism settings**
   - Adjust `maxParallel` in `ObjectStore` configuration
   - Monitor sidecar container resource usage during backups

2. **Verify plugin resource allocation**
   - Check if the sidecar container has sufficient CPU/memory
   - Review plugin container logs for resource-related warnings

:::tip
For Barman-specific features like compression, encryption, and performance
tuning, refer to the [Barman documentation](https://docs.pgbarman.org/latest/).
:::

### WAL Archiving Issues

#### WAL archiving stops

**Symptoms:**

- WAL files accumulate on the primary
- Cluster shows WAL archiving warnings
- Sidecar logs show WAL errors

**Debugging steps:**

1. **Check plugin sidecar logs for WAL archiving errors**
   ```sh
   # Check recent WAL archive operations in sidecar
   kubectl logs -n <namespace> <primary-pod> -c plugin-barman-cloud \
     --tail=50 | grep -i wal
   ```

2. **Check ObjectStore configuration for WAL settings**
   - Ensure ObjectStore has proper WAL retention settings
   - Verify credentials have permissions for WAL operations

### Restore Issues

#### Restore fails during recovery

**Symptoms:**

- New cluster stuck in recovery
- Plugin sidecar shows restore errors
- PostgreSQL won’t start

**Debugging steps:**

1. **Check plugin sidecar logs during restore**

   ```sh
   # Check the sidecar logs on the recovering cluster pods
   kubectl logs -n <namespace> <cluster-pod-name> \
     -c plugin-barman-cloud --tail=100
   
   # Look for restore-related errors
   kubectl logs -n <namespace> <cluster-pod-name> \
     -c plugin-barman-cloud | grep -E "restore|recovery|ERROR"
   ```

2. **Verify plugin can access backups**

   ```sh
   # Check if `ObjectStore` is properly configured for restore
   kubectl get objectstores.barmancloud.cnpg.io \
     -n <namespace> <objectstore-name> -o yaml
   
   # Check PostgreSQL recovery logs
   kubectl logs -n <namespace> <cluster-pod> \
     -c postgres | grep -i recovery
   ```

:::tip
For detailed Barman restore operations and troubleshooting, refer to the
[Barman documentation](https://docs.pgbarman.org/latest/barman-cloud-restore.html).
:::

#### Point-in-time recovery (PITR) configuration issues

**Symptoms:**

- PITR doesn’t reach target time
- WAL access errors
- Recovery halts early

**Debugging steps:**

1. **Verify PITR configuration in the `Cluster` spec**

   ```yaml
   apiVersion: postgresql.cnpg.io/v1
   kind: Cluster
   metadata:
     name: <cluster-restore-name>
   spec:
     storage:
       size: 1Gi

     bootstrap:
       recovery:
         source: origin
         recoveryTarget:
           targetTime: "2024-01-15T10:30:00Z"

     externalClusters:
       - name: origin
         plugin:
           enabled: true
           name: barman-cloud.cloudnative-pg.io
           parameters:
             barmanObjectName: <object-store-name>
             serverName: <source-cluster-name>
   ```

2. **Check sidecar logs for WAL-related errors**

   ```sh
   kubectl logs -n <namespace> <cluster-pod> \
     -c plugin-barman-cloud | grep -i wal
   ```

:::note
Timestamps without an explicit timezone suffix
(e.g., `2024-01-15 10:30:00`) are interpreted as UTC.
:::

:::warning
Always specify an explicit timezone in your timestamp to avoid ambiguity.
For example, use `2024-01-15T10:30:00Z` or `2024-01-15T10:30:00+02:00`
instead of `2024-01-15 10:30:00`.
:::

:::note
For detailed PITR configuration and WAL management, see the
[Barman PITR documentation](https://docs.pgbarman.org/latest/).
:::

### Plugin Configuration Issues

#### Plugin cannot connect to object storage

**Symptoms:**

- Sidecar logs show connection errors
- Backups fail with authentication or network errors
- `ObjectStore` resource reports errors

**Solution:**

1. **Verify `ObjectStore` CRD configuration and secrets**

   ```sh
   # Check ObjectStore resource status
   kubectl get objectstores.barmancloud.cnpg.io \
     -n <namespace> <objectstore-name> -o yaml
   
   # Verify the secret exists and has correct keys for your provider
   kubectl get secret -n <namespace> <secret-name> \
     -o jsonpath='{.data}' | jq 'keys'
   ```

2. **Check sidecar logs for connectivity issues**
   ```sh
   kubectl logs -n <namespace> <cluster-pod> \
     -c plugin-barman-cloud | grep -E "connect|timeout|SSL|cert"
   ```

3. **Adjust provider-specific settings (endpoint, path style, etc.)**
   - See [Object Store Configuration](object_stores.md) for provider-specific settings
   - Ensure `endpointURL` match your storage type
   - Verify network policies allow egress to your storage provider

## Diagnostic Commands

### Using the `cnpg` plugin for `kubectl`

The `cnpg` plugin for `kubectl` provides extended debugging capabilities.
Keep it updated:

```sh
# Install or update the `cnpg` plugin
kubectl krew install cnpg
# Or using an alternative method: https://cloudnative-pg.io/documentation/current/kubectl-plugin/#install

# Check plugin status (requires CNPG 1.27.0+)
kubectl cnpg status <cluster-name> -n <namespace>

# View cluster status in detail
kubectl cnpg status <cluster-name> -n <namespace> --verbose
```

## Getting Help

If problems persist:

1. **Check the documentation**

   - [Installation Guide](installation.mdx)
   - [Object Store Configuration](object_stores.md) (for provider-specific settings)
   - [Usage Examples](usage.md)


2. **Gather diagnostic information**

   ```sh
   # Create a diagnostic bundle (⚠️ sanitize these before sharing!)
   kubectl get objectstores.barmancloud.cnpg.io -A -o yaml > /tmp/objectstores.yaml
   kubectl get clusters.postgresql.cnpg.io -A -o yaml > /tmp/clusters.yaml
   kubectl logs -n cnpg-system deployment/barman-cloud --tail=1000 > /tmp/plugin.log
   ```


3. **Community support**

   - CloudNativePG Slack: [#cloudnativepg-users](https://cloud-native.slack.com/messages/cloudnativepg-users)
   - GitHub Issues: [plugin-barman-cloud](https://github.com/cloudnative-pg/plugin-barman-cloud/issues)


4. **Include when reporting**

   - CloudNativePG version
   - Plugin version
   - Kubernetes version
   - Cloud provider and region
   - Relevant configuration (⚠️ sanitize/redact sensitive information)
   - Error messages and logs
   - Steps to reproduce

## Known Issues and Limitations

### Current Known Issues

1. **Migration compatibility**: After migrating from in-tree backup to the
   plugin, the `kubectl cnpg backup` command syntax has changed
   ([#353](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/353)):

   ```sh
   # Old command (in-tree, no longer works after migration)
   kubectl cnpg backup -n <namespace> <cluster-name> \
     --method=barmanObjectStore
   
   # New command (plugin-based)
   kubectl cnpg backup -n <namespace> <cluster-name> \
     --method=plugin --plugin-name=barman-cloud.cloudnative-pg.io
   ```

### Plugin Limitations

1. **Installation method**: Currently only supports manifest and Kustomize
   installation ([#351](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/351) -
   Helm chart requested)

2. **Sidecar resource sharing**: The plugin sidecar container shares pod
   resources with PostgreSQL

3. **Plugin restart behavior**: Restarting the sidecar container requires
   restarting the entire PostgreSQL pod

## Recap of General Debugging Steps

### Check Backup Status and Identify the Target Instance

```sh
# List all backups and their status
kubectl get backups.postgresql.cnpg.io -n <namespace>

# Get detailed backup information including error messages and target instance
kubectl describe backups.postgresql.cnpg.io \
  -n <namespace> <backup-name>

# Extract the target pod name from a failed backup
kubectl get backups.postgresql.cnpg.io \
  -n <namespace> <backup-name> \
  -o jsonpath='{.status.instanceID.podName}'

# Get more details including the target pod, method, phase, and error
kubectl get backups.postgresql.cnpg.io \
  -n <namespace> <backup-name> \
  -o jsonpath='Pod: {.status.instanceID.podName}{"\n"}Method: {.status.method}{"\n"}Phase: {.status.phase}{"\n"}Error: {.status.error}{"\n"}'

# Check the cluster status for backup-related information
kubectl cnpg status <cluster-name> -n <namespace> --verbose
```

### Check Sidecar Logs on the Backup Target Pod

```sh
# Identify which pod was the backup target (from the previous step)
TARGET_POD=$(kubectl get backups.postgresql.cnpg.io \
  -n <namespace> <backup-name> \
  -o jsonpath='{.status.instanceID.podName}')
echo "Backup target pod: $TARGET_POD"

# Check the sidecar logs on the specific target pod
kubectl logs -n <namespace> $TARGET_POD \
  -c plugin-barman-cloud --tail=100

# Follow the logs in real time
kubectl logs -n <namespace> $TARGET_POD \
  -c plugin-barman-cloud -f

# Check for specific errors in the target pod around the backup time
kubectl logs -n <namespace> $TARGET_POD \
  -c plugin-barman-cloud --since=10m | grep -E "ERROR|FATAL|panic|failed"

# Alternative: List all cluster pods and their roles
kubectl get pods -n <namespace> -l cnpg.io/cluster=<cluster-name> \
  -o custom-columns=NAME:.metadata.name,ROLE:.metadata.labels.cnpg\\.io/instanceRole,INSTANCE:.metadata.labels.cnpg\\.io/instanceName

# Check sidecar logs on ALL cluster pods (if the target is unclear)
for pod in $(kubectl get pods -n <namespace> -l cnpg.io/cluster=<cluster-name> -o name); do
  echo "=== Checking $pod ==="
  kubectl logs -n <namespace> $pod -c plugin-barman-cloud \
    --tail=20 | grep -i error || echo "No errors found"
done
```

### Check Events for Backup-Related Issues

```sh
# Check events for the cluster
kubectl get events -n <namespace> \
  --field-selector involvedObject.name=<cluster-name>

# Check events for failed backups
kubectl get events -n <namespace> \
  --field-selector involvedObject.kind=Backup

# Get all recent events in the namespace
kubectl get events -n <namespace> --sort-by='.lastTimestamp' | tail -20
```

### Verify `ObjectStore` Configuration

```sh
# Check the ObjectStore resource
kubectl get objectstores.barmancloud.cnpg.io \
  -n <namespace> <objectstore-name> -o yaml

# Verify the secret exists and has the correct keys
kubectl get secret -n <namespace> <secret-name> -o yaml
# Alternatively
kubectl get secret -n <namespace> <secret-name> -o jsonpath='{.data}' | jq 'keys'
```

### Common Error Messages and Solutions

* **"AccessDenied" or "403 Forbidden"** — Check cloud credentials and bucket permissions.
* **"NoSuchBucket"** — Verify the bucket exists and the endpoint URL is correct.
* **"Connection timeout"** — Check network connectivity and firewall rules.
* **"SSL certificate problem"** — For self-signed certificates, verify the CA bundle configuration.

