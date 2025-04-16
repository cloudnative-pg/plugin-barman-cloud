# API Reference

## Packages
- [barmancloud.cnpg.io/v1](#barmancloudcnpgiov1)


## barmancloud.cnpg.io/v1

Package v1 contains API Schema definitions for the barmancloud v1 API group

### Resource Types
- [ObjectStore](#objectstore)



#### InstanceSidecarConfiguration



InstanceSidecarConfiguration defines the configuration for the sidecar that runs in the instance pods.



_Appears in:_
- [ObjectStoreSpec](#objectstorespec)

| Field | Description | Required | Default | Validation |
| --- | --- | --- | --- | --- |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#envvar-v1-core) array_ | The environment to be explicitly passed to the sidecar |  |  |  |
| `retentionPolicyIntervalSeconds` _integer_ | The retentionCheckInterval defines the frequency at which the<br />system checks and enforces retention policies. |  | 1800 |  |


#### ObjectStore



ObjectStore is the Schema for the objectstores API.





| Field | Description | Required | Default | Validation |
| --- | --- | --- | --- | --- |
| `apiVersion` _string_ | `barmancloud.cnpg.io/v1` | True | | |
| `kind` _string_ | `ObjectStore` | True | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. | True |  |  |
| `spec` _[ObjectStoreSpec](#objectstorespec)_ | Specification of the desired behavior of the ObjectStore.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status | True |  |  |
| `status` _[ObjectStoreStatus](#objectstorestatus)_ | Most recently observed status of the ObjectStore. This data may not be up to<br />date. Populated by the system. Read-only.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status |  |  |  |


#### ObjectStoreSpec



ObjectStoreSpec defines the desired state of ObjectStore.



_Appears in:_
- [ObjectStore](#objectstore)

| Field | Description | Required | Default | Validation |
| --- | --- | --- | --- | --- |
| `configuration` _[BarmanObjectStoreConfiguration](https://pkg.go.dev/github.com/cloudnative-pg/barman-cloud/pkg/api#BarmanObjectStoreConfiguration)_ | The configuration for the barman-cloud tool suite | True |  |  |
| `retentionPolicy` _string_ | RetentionPolicy is the retention policy to be used for backups<br />and WALs (i.e. '60d'). The retention policy is expressed in the form<br />of `XXu` where `XX` is a positive integer and `u` is in `[dwm]` -<br />days, weeks, months. |  |  | Pattern: `^[1-9][0-9]*[dwm]$` <br /> |
| `instanceSidecarConfiguration` _[InstanceSidecarConfiguration](#instancesidecarconfiguration)_ | The configuration for the sidecar that runs in the instance pods |  |  |  |


#### ObjectStoreStatus



ObjectStoreStatus defines the observed state of ObjectStore.



_Appears in:_
- [ObjectStore](#objectstore)

| Field | Description | Required | Default | Validation |
| --- | --- | --- | --- | --- |
| `serverRecoveryWindow` _object (keys:string, values:[RecoveryWindow](#recoverywindow))_ | ServerRecoveryWindow maps each server to its recovery window | True |  |  |


#### RecoveryWindow



RecoveryWindow represents the time span between the first
recoverability point and the last successful backup of a PostgreSQL
server, defining the period during which data can be restored.



_Appears in:_
- [ObjectStoreStatus](#objectstorestatus)

| Field | Description | Required | Default | Validation |
| --- | --- | --- | --- | --- |
| `firstRecoverabilityPoint` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#time-v1-meta)_ | The first recoverability point in a PostgreSQL server refers to<br />the earliest point in time to which the database can be<br />restored. | True |  |  |
| `lastSuccussfulBackupTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#time-v1-meta)_ | The last successful backup time | True |  |  |


