# Website

This website is built using [Docusaurus](https://docusaurus.io/), a modern
static website generator.

### Requirements

- Docker
- [Yarn](https://yarnpkg.com/)
- [Dagger](https://dagger.io/)
- [Task](https://taskfile.dev/)

### Installation

```shell
$ yarn
```

### Local Development

```shell
$ yarn start
```

This command starts a local development server and opens up a browser window.
Most changes are reflected live without having to restart the server.

### Build

```shell
$ yarn build
```

This command generates static content into the `build` directory and can be
served using any static contents hosting service.

### Test the build

```shell
$ yarn serve
```

By default, this will load your site at http://localhost:3000/.

### Spellchecking

From the top directory:

```shell
task spellcheck
```

### Versioning

Docusaurus allows versioning of the documentation to maintain separate sets of
documentation for different software versions.

To create a new documentation version:

```shell
$ yarn docusaurus docs:version X.Y.Z
```
