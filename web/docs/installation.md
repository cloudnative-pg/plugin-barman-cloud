---
sidebar_position: 4
---

# Installation

**IMPORTANT NOTES:**

1. The plugin **must** be installed in the same namespace where the operator is
   installed (typically `cnpg-system`).

2. Be aware that the operator's **listening namespaces** may differ from its
   installation namespace. Ensure you verify this distinction to avoid
   configuration issues.

Here’s an enhanced version of your instructions for verifying the prerequisites:

## Step 1 - Verify the Prerequisites

If CloudNativePG is installed in the default `cnpg-system` namespace, verify its version using the following command:

```sh
kubectl get deployment -n cnpg-system cnpg-controller-manager -o yaml \
  | grep ghcr.io/cloudnative-pg/cloudnative-pg
```

Example output:

```output
image: ghcr.io/cloudnative-pg/cloudnative-pg:1.26.0
```

Ensure that the version displayed is **1.26** or newer.

Then, use the [cmctl](https://cert-manager.io/docs/reference/cmctl/#installation)
tool to confirm that `cert-manager` is correctly installed:

```sh
cmctl check api
```

Example output:

```output
The cert-manager API is ready
```

Both checks are necessary to proceed with the installation.

## Step 2 - Install the barman-cloud Plugin

Use `kubectl` to apply the manifest for the latest commit in the `main` branch:

<!-- x-release-please-start-version -->
```sh
kubectl apply -f \
  https://github.com/cloudnative-pg/plugin-barman-cloud/releases/download/v0.3.0/manifest.yaml
```
<!-- x-release-please-end -->

Example output:

```output
customresourcedefinition.apiextensions.k8s.io/objectstores.barmancloud.cnpg.io created
serviceaccount/plugin-barman-cloud created
role.rbac.authorization.k8s.io/leader-election-role created
clusterrole.rbac.authorization.k8s.io/metrics-auth-role created
clusterrole.rbac.authorization.k8s.io/metrics-reader created
clusterrole.rbac.authorization.k8s.io/objectstore-editor-role created
clusterrole.rbac.authorization.k8s.io/objectstore-viewer-role created
clusterrole.rbac.authorization.k8s.io/plugin-barman-cloud created
rolebinding.rbac.authorization.k8s.io/leader-election-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/metrics-auth-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/plugin-barman-cloud-binding created
secret/plugin-barman-cloud-8tfddg42gf created
service/barman-cloud created
deployment.apps/barman-cloud configured
certificate.cert-manager.io/barman-cloud-client created
certificate.cert-manager.io/barman-cloud-server created
issuer.cert-manager.io/selfsigned-issuer created
```

After these steps, the plugin will be successfully installed. Make sure it is
ready to use by checking the deployment status as follows:

```sh
kubectl rollout status deployment \
  -n cnpg-system barman-cloud
```

Example output:

```output
deployment "barman-cloud" successfully rolled out
```

This confirms that the plugin is deployed and operational.
