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

:::note Authentication Methods
The Barman Cloud Plugin does not independently test all authentication methods
supported by `barman-cloud`. The plugin's responsibility is limited to passing
the provided credentials to `barman-cloud`, which then handles authentication
according to its own implementation. Users should refer to the
[Barman Cloud documentation](https://docs.pgbarman.org/release/latest/) to
verify that their chosen authentication method is supported and properly
configured.
:::

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

### S3 Lifecycle Policy

Barman Cloud uploads backup files to S3 but does not modify them afterward.
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

You can use S3-compatible services like **RustFS**, **Linode (Akamai) Object Storage**,
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

### Server-Side Encryption with Customer Keys (SSE-C)

Some S3-compatible providers — most notably **Hetzner Object Storage** — do
not offer bucket-managed server-side encryption (SSE-S3 / SSE-KMS) and instead
only support **Server-Side Encryption with Customer-provided keys (SSE-C)**.
With SSE-C the encryption key never leaves your control: it is supplied with
every request, and the provider uses it to encrypt and decrypt objects without
storing it.

To enable SSE-C, set the `sseCustomerKey` field in the `s3Credentials` block to
a secret reference holding a **base64-encoded 256-bit (32-byte) AES key**.

Generate the key and store it in a Kubernetes secret:

```sh
# Generate a random 256-bit key, base64-encoded
openssl rand 32 | base64 > encryption.key

kubectl create secret generic aws-sse-c \
  --from-file=key=encryption.key
```

:::warning
Keep this key safe and backed up **outside** the object store. If you lose
it, your backups and WAL files become permanently unrecoverable — the
provider cannot decrypt them for you.
:::

Reference it in your `ObjectStore` definition:

```yaml
apiVersion: barmancloud.cnpg.io/v1
kind: ObjectStore
metadata:
  name: hetzner-store
spec:
  configuration:
    destinationPath: "s3://BUCKET_NAME/path/to/folder"
    endpointURL: "https://fsn1.your-objectstorage.com"
    s3Credentials:
      accessKeyId:
        name: aws-creds
        key: ACCESS_KEY_ID
      secretAccessKey:
        name: aws-creds
        key: ACCESS_SECRET_KEY
      sseCustomerKey:
        name: aws-sse-c
        key: key
  [...]
```

The same key is applied to **every** operation — base backups, WAL archiving,
WAL restore, and data restore — so it must remain unchanged and available for
the whole lifetime of the backups it protects. `sseCustomerKey` can be combined
with any authentication method, including `inheritFromIAMRole`, but not with
the bucket-managed `encryption` setting (SSE-S3 / SSE-KMS) of the `data` and
`wal` sections: `barman-cloud` rejects `--sse-customer-key` together with
`--encryption`, so an object store that sets both fails at the first backup or
WAL archive.

:::note
SSE-C relies on the `--sse-customer-key` option introduced in Barman 3.20.0,
which the plugin sidecar image ships starting from version 0.15.0.
:::

### Using Object Storage with a Private CA

For object storage services (e.g., RustFS) that use HTTPS with certificates
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
  name: s3-store
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
- Storage Account Name + [Storage Account Access Key](https://learn.microsoft.com/en-us/azure/storage/common/storage-account-keys-manage)
- Storage Account Name + [Storage Account SAS Token](https://learn.microsoft.com/en-us/azure/storage/blobs/sas-service-create)
- [Azure AD Managed Identity](https://learn.microsoft.com/en-us/entra/identity/managed-identities-azure-resources/overview)
- [Default Azure Credentials](https://learn.microsoft.com/en-us/dotnet/api/azure.identity.defaultazurecredential?view=azure-dotnet)

### Azure AD Managed Identity

This method avoids storing credentials in Kubernetes by enabling the
usage of [Azure Managed Identities](https://learn.microsoft.com/en-us/entra/identity/managed-identities-azure-resources/overview) authentication mechanism.
This can be enabled by setting the `inheritFromAzureAD` option to `true`.
Managed Identity can be configured for the AKS Cluster by following
the [Azure documentation](https://learn.microsoft.com/en-us/azure/aks/use-managed-identity?pivots=system-assigned).

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

### Default Azure Credentials

The `useDefaultAzureCredentials` option enables the default Azure credentials
flow, which uses [`DefaultAzureCredential`](https://learn.microsoft.com/en-us/python/api/azure-identity/azure.identity.defaultazurecredential)
to automatically discover and use available credentials in the following order:

1. **Environment Variables** — `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, and `AZURE_TENANT_ID` for Service Principal authentication
2. **Managed Identity** — Uses the managed identity assigned to the pod
3. **Azure CLI** — Uses credentials from the Azure CLI if available
4. **Azure PowerShell** — Uses credentials from Azure PowerShell if available

This approach is particularly useful for getting started with development and testing; it allows
the SDK to attempt multiple authentication mechanisms seamlessly across different environments.
However, this is not recommended for production. Please refer to the
[official Azure guidance](https://learn.microsoft.com/en-us/dotnet/azure/sdk/authentication/credential-chains?tabs=dac#usage-guidance-for-defaultazurecredential)
for a comprehensive understanding of `DefaultAzureCredential`.

```yaml
apiVersion: barmancloud.cnpg.io/v1
kind: ObjectStore
metadata:
  name: azure-store
spec:
  configuration:
    destinationPath: "<destination path here>"
    azureCredentials:
      useDefaultAzureCredentials: true
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


## RustFS Object Store

[RustFS](https://rustfs.com/) is an open source, S3-compatible object store
that can run inside your Kubernetes cluster. See the
[RustFS documentation](https://docs.rustfs.com/) for deployment options; a
minimal `Deployment` with a `PersistentVolumeClaim` and a `Service` exposing
port 9000 is enough for testing purposes. RustFS serves plain HTTP unless it
finds a certificate and key under the directory pointed to by
`RUSTFS_TLS_PATH`, in which case it serves HTTPS and you must reference the
CA through `endpointCA` (see
[Using Object Storage with a Private CA](#using-object-storage-with-a-private-ca)).

RustFS reads its root credentials from the `RUSTFS_ACCESS_KEY` and
`RUSTFS_SECRET_KEY` environment variables. Store the same values in a
`Secret` for the plugin:

```sh
kubectl create secret generic s3-creds \
  --from-literal=ACCESS_KEY_ID=<rustfs access key> \
  --from-literal=ACCESS_SECRET_KEY=<rustfs secret key>
```

Finally, create the Barman `ObjectStore`, pointing `endpointURL` at the
RustFS `Service`:

```yaml
apiVersion: barmancloud.cnpg.io/v1
kind: ObjectStore
metadata:
  name: s3-store
spec:
  configuration:
    destinationPath: s3://BUCKET_NAME/
    endpointURL: http://<rustfs-service>:9000
    s3Credentials:
      accessKeyId:
        name: s3-creds
        key: ACCESS_KEY_ID
      secretAccessKey:
        name: s3-creds
        key: ACCESS_SECRET_KEY
  [...]
```

The bucket is created on first use if it does not exist.

:::important
Verify on `s3://BUCKET_NAME/` the presence of archived WAL files before
proceeding with a backup.
:::
---
