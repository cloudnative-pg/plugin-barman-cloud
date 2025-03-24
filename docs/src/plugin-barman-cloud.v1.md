# API Reference

<p>Package v1 contains API Schema definitions for the barmancloud v1 API group</p>


## Resource Types


- [ObjectStore](#barmancloud-cnpg-io-v1-ObjectStore)

## ObjectStore     {#barmancloud-cnpg-io-v1-ObjectStore}



<p>ObjectStore is the Schema for the objectstores API.</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
<tr><td><code>apiVersion</code> <B>[Required]</B><br/>string</td><td><code>barmancloud.cnpg.io/v1</code></td></tr>
<tr><td><code>kind</code> <B>[Required]</B><br/>string</td><td><code>ObjectStore</code></td></tr>
<tr><td><code>metadata</code> <B>[Required]</B><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta"><i>meta/v1.ObjectMeta</i></a>
</td>
<td>
   <span class="text-muted">No description provided.</span>Refer to the Kubernetes API documentation for the fields of the <code>metadata</code> field.</td>
</tr>
<tr><td><code>spec</code> <B>[Required]</B><br/>
<a href="#barmancloud-cnpg-io-v1-ObjectStoreSpec"><i>ObjectStoreSpec</i></a>
</td>
<td>
   <p>Specification of the desired behavior of the ObjectStore.
More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status</p>
</td>
</tr>
<tr><td><code>status</code><br/>
<a href="#barmancloud-cnpg-io-v1-ObjectStoreStatus"><i>ObjectStoreStatus</i></a>
</td>
<td>
   <p>Most recently observed status of the ObjectStore. This data may not be up to
date. Populated by the system. Read-only.
More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status</p>
</td>
</tr>
</tbody>
</table>

## InstanceSidecarConfiguration     {#barmancloud-cnpg-io-v1-InstanceSidecarConfiguration}


**Appears in:**

- [ObjectStoreSpec](#barmancloud-cnpg-io-v1-ObjectStoreSpec)


<p>InstanceSidecarConfiguration defines the configuration for the sidecar that runs in the instance pods.</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
<tr><td><code>env</code><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#envvar-v1-core"><i>[]core/v1.EnvVar</i></a>
</td>
<td>
   <p>The environment to be explicitly passed to the sidecar</p>
</td>
</tr>
<tr><td><code>retentionPolicyIntervalSeconds</code><br/>
<i>int</i>
</td>
<td>
   <p>The retentionCheckInterval defines the frequency at which the
system checks and enforces retention policies.</p>
</td>
</tr>
</tbody>
</table>

## ObjectStoreSpec     {#barmancloud-cnpg-io-v1-ObjectStoreSpec}


**Appears in:**

- [ObjectStore](#barmancloud-cnpg-io-v1-ObjectStore)


<p>ObjectStoreSpec defines the desired state of ObjectStore.</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
<tr><td><code>configuration</code> <B>[Required]</B><br/>
<a href="https://pkg.go.dev/github.com/cloudnative-pg/barman-cloud/pkg/api/#BarmanObjectStoreConfiguration"><i>github.com/cloudnative-pg/barman-cloud/pkg/api.BarmanObjectStoreConfiguration</i></a>
</td>
<td>
   <p>The configuration for the barman-cloud tool suite</p>
</td>
</tr>
<tr><td><code>retentionPolicy</code><br/>
<i>string</i>
</td>
<td>
   <p>RetentionPolicy is the retention policy to be used for backups
and WALs (i.e. '60d'). The retention policy is expressed in the form
of <code>XXu</code> where <code>XX</code> is a positive integer and <code>u</code> is in <code>[dwm]</code> -
days, weeks, months.</p>
</td>
</tr>
<tr><td><code>instanceSidecarConfiguration</code><br/>
<a href="#barmancloud-cnpg-io-v1-InstanceSidecarConfiguration"><i>InstanceSidecarConfiguration</i></a>
</td>
<td>
   <p>The configuration for the sidecar that runs in the instance pods</p>
</td>
</tr>
</tbody>
</table>

## ObjectStoreStatus     {#barmancloud-cnpg-io-v1-ObjectStoreStatus}


**Appears in:**

- [ObjectStore](#barmancloud-cnpg-io-v1-ObjectStore)


<p>ObjectStoreStatus defines the observed state of ObjectStore.</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
<tr><td><code>serverRecoveryWindow</code> <B>[Required]</B><br/>
<a href="#barmancloud-cnpg-io-v1-RecoveryWindow"><i>map[string]RecoveryWindow</i></a>
</td>
<td>
   <p>ServerRecoveryWindow maps each server to its recovery window</p>
</td>
</tr>
</tbody>
</table>

## RecoveryWindow     {#barmancloud-cnpg-io-v1-RecoveryWindow}


**Appears in:**

- [ObjectStoreStatus](#barmancloud-cnpg-io-v1-ObjectStoreStatus)


<p>RecoveryWindow represents the time span between the first
recoverability point and the last successful backup of a PostgreSQL
server, defining the period during which data can be restored.</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
<tr><td><code>firstRecoverabilityPoint</code> <B>[Required]</B><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta"><i>meta/v1.Time</i></a>
</td>
<td>
   <p>The first recoverability point in a PostgreSQL server refers to
the earliest point in time to which the database can be
restored.</p>
</td>
</tr>
<tr><td><code>lastSuccussfulBackupTime</code> <B>[Required]</B><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta"><i>meta/v1.Time</i></a>
</td>
<td>
   <p>The last successful backup time</p>
</td>
</tr>
</tbody>
</table>