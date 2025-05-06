---
sidebar_position: 20
---

# Installation

<!-- SPDX-License-Identifier: CC-BY-4.0 -->

:::important
1. The plugin **must** be installed in the same namespace as the CloudNativePG
   operator (typically `cnpg-system`).

2. Keep in mind that the operator's **listening namespaces** may differ from its
   installation namespace. Double-check this to avoid configuration issues.
:::

## Verifying the Requirements

Before installing the plugin, make sure the [requirements](intro.md#requirements) are met.

### CloudNativePG Version

Ensure you're running a version of CloudNativePG that is compatible with the
plugin. If installed in the default `cnpg-system` namespace, you can verify the
version with:

```sh
kubectl get deployment -n cnpg-system cnpg-controller-manager -o yaml \
  | grep ghcr.io/cloudnative-pg/cloudnative-pg
```

Example output:

```output
image: ghcr.io/cloudnative-pg/cloudnative-pg:1.26.0
```

The version **must be 1.26 or newer**.

### cert-manager

Use the [cmctl](https://cert-manager.io/docs/reference/cmctl/#installation)
tool to confirm that `cert-manager` is installed and available:

```sh
cmctl check api
```

Example output:

```output
The cert-manager API is ready
```

Both checks are required before proceeding with the installation.

## Installing the Barman Cloud Plugin

Install the plugin using `kubectl` by applying the manifest for the latest
release:

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

Finally, check that the deployment is up and running:

```sh
kubectl rollout status deployment \
  -n cnpg-system barman-cloud
```

Example output:

```output
deployment "barman-cloud" successfully rolled out
```

This confirms that the plugin is deployed and ready to use.
