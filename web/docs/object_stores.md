---
sidebar_position: 50
---

# Object Store Providers

<!-- SPDX-License-Identifier: CC-BY-4.0 -->

The Barman Cloud Plugin enables the storage of PostgreSQL cluster backup files
in any object storage service supported by the
[Barman Cloud infrastructure](https://docs.pgbarman.org/release/latest/).

Currently, Barman Cloud supports the following providers:

- [Amazon S3](#aws-s3)
- [Microsoft Azure Blob Storage](#azure-blob-storage)
- [Google Cloud Storage](#google-cloud-storage)

You may also use any S3- or Azure-compatible implementation of the above
services.

To configure object storage with Barman Cloud, you must define an
[`ObjectStore` object](plugin-barman-cloud.v1.md#objectstore), which
establishes the connection between your PostgreSQL cluster and the object
storage backend.

Configuration details — particularly around authentication — will vary depending on
the specific object storage provider you are using.

The following sections detail the setup for each.

---

## AWS S3

[AWS Simple Storage Service (S3)](https://aws.amazon.com/s3/) is one of the
most widely adopted object storage solutions.

The Barman Cloud plugin for CloudNativePG integrates with S3 through two
primary authentication mechanisms:

- [IAM Roles for Service Accounts (IRSA)](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) —
  recommended for clusters running on EKS
- Access keys — using `ACCESS_KEY_ID` and `ACCESS_SECRET_KEY` credentials

### Access Keys

To authenticate using access keys, you’ll need:

- `ACCESS_KEY_ID`: the public key used to authenticate to S3
- `ACCESS_SECRET_KEY`: the corresponding secret key
- `ACCESS_SESSION_TOKEN`: (optional) a temporary session token, if required

These credentials must be stored securely in a Kubernetes secret:

```sh
kubectl create secret generic aws-creds \
  --from-literal=ACCESS_KEY_ID=<access key here> \
  --from-literal=ACCESS_SECRET_KEY=<secret key here>
# --from-literal=ACCESS_SESSION_TOKEN=<session token here> # if required
```

The credentials will be encrypted at rest if your Kubernetes environment
supports it.

You can then reference the secret in your `ObjectStore` definition:

```yaml
apiVersion: barmancloud.cnpg.io/v1
kind: ObjectStore
metadata:
  name: aws-store
spec:
  configuration:
    destinationPath: "s3://BUCKET_NAME/path/to/folder"
    s3Credentials:
      accessKeyId:
        name: aws-creds
        key: ACCESS_KEY_ID
      secretAccessKey:
        name: aws-creds
        key: ACCESS_SECRET_KEY
  [...]
```

### IAM Role for Service Account (IRSA)

To use IRSA with EKS, configure the service account of the PostgreSQL cluster
with the appropriate annotation:

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  [...]
spec:
  serviceAccountTemplate:
    metadata:
      annotations:
        eks.amazonaws.com/role-arn: arn:[...]
        [...]
```

In addition, configure the `ObjectStore` to inherit permissions from the IAM role referenced in the service account:

```yaml
apiVersion: barmancloud.cnpg.io/v1
kind: ObjectStore
metadata:
  [...]
spec:
  configuration:
    s3Credentials:
      inheritFromIAMRole: true
```

### S3 Lifecycle Policy

Barman Cloud uploads backup files to S3 but does not modify or delete them afterward.
To enhance data durability and protect against accidental or malicious loss,
it's recommended to implement the following best practices:

- Enable object versioning
- Enable object locking to prevent objects from being deleted or overwritten
  for a defined period or indefinitely (this provides an additional layer of
  protection against accidental deletion and ransomware attacks)
- Set lifecycle rules to expire current versions a few days after your Barman
  retention window
- Expire non-current versions after a longer period

These strategies help you safeguard backups without requiring broad delete
permissions, ensuring both security and compliance with minimal operational
overhead.


### S3-Compatible Storage Providers

You can use S3-compatible services like **MinIO**, **Linode (Akamai) Object Storage**,
or **DigitalOcean Spaces** by specifying a custom `endpointURL`.

Example with Linode (Akamai) Object Storage (`us-east1`):

```yaml
apiVersion: barmancloud.cnpg.io/v1
kind: ObjectStore
metadata:
  name: linode-store
spec:
  configuration:
    destinationPath: "s3://BUCKET_NAME/"
    endpointURL: "https://us-east1.linodeobjects.com"
    s3Credentials:
    [...]
  [...]
```

Recent changes to the [boto3 implementation](https://github.com/boto/boto3/issues/4392)
of [Amazon S3 Data Integrity Protections](https://docs.aws.amazon.com/sdkref/latest/guide/feature-dataintegrity.html)
may lead to the `x-amz-content-sha256` error when using the Barman Cloud
Plugin.

If you encounter this issue (see [GitHub issue #393](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/393)),
you can apply the following workaround by setting specific environment
variables in the `ObjectStore` resource:

```yaml
apiVersion: barmancloud.cnpg.io/v1
kind: ObjectStore
metadata:
  name: linode-store
spec:
  instanceSidecarConfiguration:
    env:
      - name: AWS_REQUEST_CHECKSUM_CALCULATION
        value: when_required
      - name: AWS_RESPONSE_CHECKSUM_VALIDATION
        value: when_required
  [...]
```

These settings ensure that checksum calculations and validations are only
applied when explicitly required, avoiding compatibility issues with certain
S3-compatible storage providers.

Example with DigitalOcean Spaces (SFO3, path-style):

```yaml
apiVersion: barmancloud.cnpg.io/v1
kind: ObjectStore
metadata:
  name: digitalocean-store
spec:
  configuration:
    destinationPath: "s3://BUCKET_NAME/path/to/folder"
    endpointURL: "https://sfo3.digitaloceanspaces.com"
    s3Credentials:
    [...]
  [...]
```

### Using Object Storage with a Private CA

For object storage services (e.g., MinIO) that use HTTPS with certificates
signed by a private CA, set the `endpointCA` field in the `ObjectStore`
definition. Unless you already have it, create a Kubernetes `Secret` with the
CA bundle:

```sh
kubectl create secret generic my-ca-secret --from-file=ca.crt
```

Then reference it:

```yaml
apiVersion: barmancloud.cnpg.io/v1
kind: ObjectStore
metadata:
  name: minio-store
spec:
  configuration:
    endpointURL: <myEndpointURL>
    endpointCA:
      name: my-ca-secret
      key: ca.crt
  [...]
```

<!-- TODO: does this also apply to the plugin? -->
:::note
If you want `ConfigMaps` and `Secrets` to be **automatically** reloaded by
instances, you can add a label with the key `cnpg.io/reload` to the
`Secrets`/`ConfigMaps`. Otherwise, you will have to reload the instances using the
`kubectl cnpg reload` subcommand.
:::

---

## Azure Blob Storage

[Azure Blob Storage](https://azure.microsoft.com/en-us/services/storage/blobs/)
is Microsoft’s cloud-based object storage solution.

Barman Cloud supports the following authentication methods:

- [Connection String](https://learn.microsoft.com/en-us/azure/storage/common/storage-configure-connection-string)
- Storage Account Name + [Access Key](https://learn.microsoft.com/en-us/azure/storage/common/storage-account-keys-manage)
- Storage Account Name + [SAS Token](https://learn.microsoft.com/en-us/azure/storage/blobs/sas-service-create)
- [Azure AD Workload Identity](https://azure.github.io/azure-workload-identity/docs/introduction.html)

### Azure AD Workload Identity

This method avoids storing credentials in Kubernetes via the
`.spec.configuration.inheritFromAzureAD` option:

```yaml
apiVersion: barmancloud.cnpg.io/v1
kind: ObjectStore
metadata:
  name: azure-store
spec:
  configuration:
    destinationPath: "<destination path here>"
    azureCredentials:
      inheritFromAzureAD: true
  [...]
```

### Access Key, SAS Token, or Connection String

Store credentials in a Kubernetes secret:

```sh
kubectl create secret generic azure-creds \
  --from-literal=AZURE_STORAGE_ACCOUNT=<storage account name> \
  --from-literal=AZURE_STORAGE_KEY=<storage account key> \
  --from-literal=AZURE_STORAGE_SAS_TOKEN=<SAS token> \
  --from-literal=AZURE_STORAGE_CONNECTION_STRING=<connection string>
```

Then reference the required keys in your `ObjectStore`:

```yaml
apiVersion: barmancloud.cnpg.io/v1
kind: ObjectStore
metadata:
  name: azure-store
spec:
  configuration:
    destinationPath: "<destination path here>"
    azureCredentials:
      connectionString:
        name: azure-creds
        key: AZURE_CONNECTION_STRING
      storageAccount:
        name: azure-creds
        key: AZURE_STORAGE_ACCOUNT
      storageKey:
        name: azure-creds
        key: AZURE_STORAGE_KEY
      storageSasToken:
        name: azure-creds
        key: AZURE_STORAGE_SAS_TOKEN
  [...]
```

For Azure Blob, the destination path format is:

```
<http|https>://<account-name>.<service-name>.core.windows.net/<container>/<blob>
```

### Azure-Compatible Providers

If you're using a different implementation (e.g., Azurite or emulator):

```
<http|https>://<local-machine-address>:<port>/<account-name>/<container>/<blob>
```

---

## Google Cloud Storage

[Google Cloud Storage](https://cloud.google.com/storage/) is supported with two
authentication modes:

- **GKE Workload Identity** (recommended inside Google Kubernetes Engine)
- **Service Account JSON key** via the `GOOGLE_APPLICATION_CREDENTIALS` environment variable

### GKE Workload Identity

Use the [Workload Identity authentication](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity)
when running in GKE:

1. Set `googleCredentials.gkeEnvironment` to `true` in the `ObjectStore`
   resource
2. Annotate the `serviceAccountTemplate` in the `Cluster` resource with the GCP
   service account

For example, in the `ObjectStore` resource:

```yaml
apiVersion: barmancloud.cnpg.io/v1
kind: ObjectStore
metadata:
  name: google-store
spec:
  configuration:
    destinationPath: "gs://<bucket>/<folder>"
    googleCredentials:
      gkeEnvironment: true
```

And in the `Cluster` resource:

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
spec:
  serviceAccountTemplate:
    metadata:
      annotations:
        iam.gke.io/gcp-service-account: [...].iam.gserviceaccount.com
```

### Service Account JSON Key

Follow Google’s [authentication setup](https://cloud.google.com/docs/authentication/getting-started),
then:

```sh
kubectl create secret generic backup-creds --from-file=gcsCredentials=gcs_credentials_file.json
```

```yaml
apiVersion: barmancloud.cnpg.io/v1
kind: ObjectStore
metadata:
  name: google-store
spec:
  configuration:
    destinationPath: "gs://<bucket>/<folder>"
    googleCredentials:
      applicationCredentials:
        name: backup-creds
        key: gcsCredentials
  [...]
```

:::important
This authentication method generates a JSON file within the container
with all the credentials required to access your Google Cloud Storage
bucket. As a result, if someone gains access to the `Pod`, they will also have
write permissions to the bucket.
:::

---


## MinIO Gateway

MinIO Gateway can proxy requests to cloud object storage providers like S3 or GCS.
For more information, refer to [MinIO official documentation](https://docs.min.io/).

### Setup

Create MinIO access credentials:

```sh
kubectl create secret generic minio-creds \
  --from-literal=MINIO_ACCESS_KEY=<minio access key> \
  --from-literal=MINIO_SECRET_KEY=<minio secret key>
```

:::note
Cloud Object Storage credentials will be used only by MinIO Gateway in this
case.
:::

Expose MinIO Gateway via `ClusterIP`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: minio-gateway-service
spec:
  type: ClusterIP
  ports:
    - port: 9000
      targetPort: 9000
      protocol: TCP
  selector:
    app: minio
```

Here follows an excerpt of an example of deployment relaying to S3:

```yaml
apiVersion: apps/v1
kind: Deployment
[...]
spec:
  containers:
  - name: minio
    image: minio/minio:RELEASE.2020-06-03T22-13-49Z
    args: ["gateway", "s3"]
    ports:
    - containerPort: 9000
    env:
    - name: MINIO_ACCESS_KEY
      valueFrom:
        secretKeyRef:
          name: minio-creds
          key: MINIO_ACCESS_KEY
    - name: MINIO_SECRET_KEY
      valueFrom:
        secretKeyRef:
          name: minio-creds
          key: MINIO_SECRET_KEY
    - name: AWS_ACCESS_KEY_ID
      valueFrom:
        secretKeyRef:
          name: aws-creds
          key: ACCESS_KEY_ID
    - name: AWS_SECRET_ACCESS_KEY
      valueFrom:
        secretKeyRef:
          name: aws-creds
          key: ACCESS_SECRET_KEY
# Uncomment the below section if session token is required
#   - name: AWS_SESSION_TOKEN
#     valueFrom:
#       secretKeyRef:
#         name: aws-creds
#         key: ACCESS_SESSION_TOKEN
```

Proceed by configuring MinIO Gateway service as the `endpointURL` in the
`ObjectStore` definition, then choose a bucket name to replace `BUCKET_NAME`:

```yaml
apiVersion: barmancloud.cnpg.io/v1
kind: ObjectStore
metadata:
  name: minio-store
spec:
  configuration:
    destinationPath: s3://BUCKET_NAME/
    endpointURL: http://minio-gateway-service:9000
    s3Credentials:
      accessKeyId:
        name: minio-creds
        key: MINIO_ACCESS_KEY
      secretAccessKey:
        name: minio-creds
        key: MINIO_SECRET_KEY
  [...]
```

:::important
Verify on `s3://BUCKET_NAME/` the presence of archived WAL files before
proceeding with a backup.
:::

---
