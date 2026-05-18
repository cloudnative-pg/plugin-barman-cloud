[![CloudNativePG](./logo/cloudnativepg.png)](https://cloudnative-pg.io/)

# Barman Cloud CNPG-I plugin for CloudNativePG

The Barman Cloud CNPG-I plugin extends CloudNativePG with backup and restore integration powered by the Barman Cloud ecosystem.

## Table of Contents

- [What this plugin does](#what-this-plugin-does)
- [Highlights](#highlights)
- [Quick Start](#quick-start)
- [Repository Layout](#repository-layout)
- [Documentation](#documentation)
- [Community](#community)
- [Sponsors and trademarks](#sponsors-and-trademarks)

## What this plugin does

This repository contains the operator, instance, restore, and healthcheck commands that support Barman Cloud backup workflows for CloudNativePG. It also includes Kubernetes manifests, example configurations, end-to-end tests, and the documentation site.

## Highlights

- Backup and restore integration for CloudNativePG environments
- Operator and instance command implementations under `cmd/` and `internal/`
- Example manifests under `hack/examples/`
- Installable Kubernetes manifests generated from `config/`
- Project documentation site under `web/`

## Quick Start

### Read the full docs

Start with the official documentation:

- <https://cloudnative-pg.io/plugin-barman-cloud>

### Explore example manifests

This repository ships example YAML files such as:

- `hack/examples/backup-example.yaml`
- `hack/examples/cluster-example.yaml`
- `hack/examples/cluster-restore.yaml`
- `hack/examples/minio-store.yaml`

### Run the main local workflows

```bash
make build
make test
make build-installer
```

## Repository Layout

- `cmd/`: entrypoints for manager commands
- `internal/`: plugin internals, controllers, command implementations, and shared logic
- `config/`: CRDs, RBAC, default deployment, samples, and install manifests
- `hack/examples/`: example backup, restore, and object store manifests
- `test/`: end-to-end coverage
- `web/`: Docusaurus documentation site

## Documentation

- Product docs: <https://cloudnative-pg.io/plugin-barman-cloud>
- Governance: [GOVERNANCE.md](GOVERNANCE.md)
- Website source: [web/](web/)

## Community

The Barman Cloud CNPG-I plugin is part of the [CloudNativePG project](https://github.com/cloudnative-pg) and follows the same community-driven [governance](GOVERNANCE.md) model under the [CNCF](https://cncf.io).

## Sponsors and trademarks

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://github.com/cncf/artwork/blob/main/other/cncf/horizontal/white/cncf-white.svg?raw=true">
    <source media="(prefers-color-scheme: light)" srcset="https://github.com/cncf/artwork/blob/main/other/cncf/horizontal/color/cncf-color.svg?raw=true">
    <img src="https://github.com/cncf/artwork/blob/main/other/cncf/horizontal/color/cncf-color.svg?raw=true" alt="CNCF logo" width="50%" />
  </picture>
</p>

<p align="center">
CloudNativePG was originally built and sponsored by <a href="https://www.enterprisedb.com">EDB</a>.
</p>

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/cloudnative-pg/.github/main/logo/edb_landscape_color_white.svg">
    <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/cloudnative-pg/.github/main/logo/edb_landscape_color_grey.svg">
    <img src="https://raw.githubusercontent.com/cloudnative-pg/.github/main/logo/edb_landscape_color_grey.svg" alt="EDB logo" width="25%" />
  </picture>
</p>

<p align="center">
<a href="https://www.postgresql.org/about/policies/trademarks/">Postgres, PostgreSQL, and the Slonik Logo</a>
are trademarks or registered trademarks of the PostgreSQL Community Association of Canada, and used with their permission.
</p>
