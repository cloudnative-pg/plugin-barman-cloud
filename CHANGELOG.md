# Changelog

## [0.8.0](https://github.com/cloudnative-pg/plugin-barman-cloud/compare/v0.7.0...v0.8.0) (2025-10-27)


### ⚠ BREAKING CHANGES

* **rbac:** Resource names have been prefixed to avoid cluster conflicts. All cluster-scoped and namespace-scoped resources now use the `barman-plugin-` prefix for consistency; see the [Resource Name Migration Guide](https://cloudnative-pg.io/plugin-barman-cloud/resource-name-migration/) for detailed migration instructions.

### Features

* **ip:** Assign copyright to the Linux Foundation ([#571](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/571)) ([1be34fe](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/1be34fe13e830a219d0d8d68423caf2d3c55a49b))
* **rbac:** Prefix all resource names to avoid cluster conflicts ([#593](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/593)) ([c2bfe12](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/c2bfe1217e8542c80dd2b099d8d966e725e2b280)), closes [#395](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/395)


### Bug Fixes

* **deps,security:** Update to go 1.25.2 ([#581](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/581)) ([523bd1e](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/523bd1e2b3fb1d63ad930d15d172513eb0be7dee)), closes [#580](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/580)
* **deps:** Lock file maintenance documentation dependencies ([#555](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/555)) ([fad3a65](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/fad3a65340870c9d1553018e760d72b3f3a8aa4d))
* **deps:** Lock file maintenance documentation dependencies ([#612](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/612)) ([da5acb5](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/da5acb59d892670de668835d7850e4e09183e16d))
* **deps:** Update all non-major go dependencies ([#616](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/616)) ([3a9697e](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/3a9697e69c16ca913f78278ebe0f89fa355d0726))
* **deps:** Update k8s.io/utils digest to bc988d5 ([#559](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/559)) ([36db77c](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/36db77ced4be3f77278c8e831b7fae06c7beb3cb))
* **deps:** Update module github.com/cert-manager/cert-manager to v1.19.0 ([#575](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/575)) ([484b280](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/484b28017e23fd5166c558c27c15103a586f068b))
* **deps:** Update module github.com/cert-manager/cert-manager to v1.19.1 ([#600](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/600)) ([d8f78f9](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/d8f78f90d02b081ecc4a60ccc925b998f89ced00))
* **deps:** Update module github.com/onsi/ginkgo/v2 to v2.26.0 ([#560](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/560)) ([529737f](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/529737ffa43fd4af8a9602a072f9c9eda9f3e747))
* **deps:** Update module github.com/onsi/ginkgo/v2 to v2.27.0 ([#614](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/614)) ([6700c60](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/6700c6044603712d77597c1ec46beae59220ef3b))
* **deps:** Update module google.golang.org/grpc to v1.76.0 ([#569](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/569)) ([e1bc0a1](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/e1bc0a1d4a4d2c08b69726ab04484b2d43c5adf1))
* **deps:** Update module sigs.k8s.io/controller-runtime to v0.22.2 ([#568](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/568)) ([1b5955e](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/1b5955ead9b7e56c48440abd452d348bf0ec5385))
* **deps:** Update module sigs.k8s.io/controller-runtime to v0.22.3 ([#586](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/586)) ([ea76733](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/ea7673343a2120fd9871f81688ea0bf68906444a))
* Disable management of end-of-wal file flag during backup restoration ([#604](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/604)) ([931a06a](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/931a06a407cc4885bfcd653535a81aca37ecbd0c)), closes [#603](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/603)
* **e2e:** Avoid pinpointing the PostgreSQL version ([#562](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/562)) ([5276dd1](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/5276dd17cfd3bea41918a69622c385756b0404cb))
* Set LeaderElectionReleaseOnCancel to true to enable RollingUpdates ([#615](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/615)) ([49f1096](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/49f1096cba74008f84435dcbb82e59f43e5ae112)), closes [#419](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/419)

## [0.7.0](https://github.com/cloudnative-pg/plugin-barman-cloud/compare/v0.6.0...v0.7.0) (2025-09-25)


### Features

* Introduce `logLevel` setting to control verbosity ([#536](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/536)) ([0501e18](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/0501e185bab4969064c5b92977747be30bd38e95))
* Return proper gRPC error codes for expected conditions ([#549](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/549)) ([08c3f1c](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/08c3f1c2324d79d6080fbf73f11b4fa715bec4cb))
* **spec:** Add support for additional sidecar container arguments ([#520](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/520)) ([ec352ac](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/ec352ac0fdd31656321e564bcf6a026481ec06e4))


### Bug Fixes

* Avoid panicking if serverRecoveryWindow has still not been set ([#525](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/525)) ([dfd9861](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/dfd9861a3f9296bffe084a81faa8755ddca95149)), closes [#523](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/523)
* **deps:** Lock file maintenance documentation dependencies ([#534](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/534)) ([0ad066d](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/0ad066d195b8556d9cf13ac0b585bfa6ffe01b75))
* **deps:** Update all non-major go dependencies ([#521](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/521)) ([df92fa6](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/df92fa6f3e9bfd934da4be2aba4983570f751fad))
* **deps:** Update kubernetes packages to v0.34.1 ([#530](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/530)) ([eced5ea](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/eced5ea2c6d44ec3fc09b632b42c204a5d469297))
* **deps:** Update module github.com/cloudnative-pg/cnpg-i-machinery to v0.4.1 ([#551](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/551)) ([65a0d11](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/65a0d11ec8cf1fc6e3478d49ad88d9ba9c40adf6))
* **deps:** Update module github.com/onsi/ginkgo/v2 to v2.25.1 ([#495](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/495)) ([2dc29a5](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/2dc29a5dbcc4e4a5b79cc2c796d2a451ffcd654a))
* **deps:** Update module sigs.k8s.io/controller-runtime to v0.22.1 ([#531](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/531)) ([82449d9](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/82449d9351555e3b8ee128f040bffd9799279e72))
* **logs:** Log the correct name when on ObjectStore not found error ([#540](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/540)) ([a29aa1c](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/a29aa1c91af0bc7cb4a7511c49dcc461900e9a13)), closes [#539](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/539)
* **object-cache:** Improve reliability of object cache management ([#508](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/508)) ([8c3db95](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/8c3db955efc2d23593faa0c6e410e7aa0e427ebf)), closes [#502](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/502)
* Typo in variable name ([#515](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/515)) ([3c0d8c3](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/3c0d8c3a3394d5b628d03c849be86999b2e7887f))

## [0.6.0](https://github.com/cloudnative-pg/plugin-barman-cloud/compare/v0.5.0...v0.6.0) (2025-08-21)


### Features

* Add upstream backup and recovery metrics ([#459](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/459)) ([33172b6](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/33172b6466b57e23dc0479fbb9d7af53362dba91))
* Last failed backup status field and metric ([#467](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/467)) ([551a3cd](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/551a3cde09886d88851e751ab289e04630243a7c))


### Bug Fixes

* Add cluster/finalizers update permission ([#465](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/465)) ([e0c8b64](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/e0c8b64470cc31f36b0511b80bbac6ecaa8bd283))
* Check for empty WAL archive during WAL archiving ([#458](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/458)) ([950364b](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/950364b9559c7e2079c09145f4fc23ce6a96dedc)), closes [#457](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/457)
* **ci:** Show test output on failures ([#461](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/461)) ([3a77079](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/3a770798c718ad7bb88502bf55ee1beebef17e0c))
* **deps:** Lock file maintenance documentation dependencies ([#379](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/379)) ([a0327ea](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/a0327ea574558d6c1a913e13a12bb454818900a7))
* **deps:** Lock file maintenance documentation dependencies ([#399](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/399)) ([7146c51](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/7146c51de11a5d673aef23e36e07a2b0c528d3b7))
* **deps:** Lock file maintenance documentation dependencies ([#407](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/407)) ([4d323c2](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/4d323c2d3df2bcd52c126b369922bec67db68a2c))
* **deps:** Lock file maintenance documentation dependencies ([#412](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/412)) ([7aaebb3](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/7aaebb3c25e04022fd51a99fac2eeee4c91de532))
* **deps:** Lock file maintenance documentation dependencies ([#492](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/492)) ([4ab42c4](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/4ab42c43fc3399c4411382caac9dd5f72593e885))
* **deps:** Update all non-major go dependencies ([#435](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/435)) ([6028011](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/60280118c46c2b75e044b7ba44d7bc1389a5da20))
* **deps:** Update all non-major go dependencies ([#469](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/469)) ([a7bde51](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/a7bde51c63009cc8d4cc1e499e320ed954b6818a))
* **deps:** Update k8s.io/utils digest to 0af2bda ([#487](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/487)) ([83ada2b](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/83ada2b883806ff8558cb286025f267300635ef4))
* **deps:** Update k8s.io/utils digest to 4c0f3b2 ([#392](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/392)) ([e58973c](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/e58973cd55b89c2e4615cf67c85b08627590aae1))
* **deps:** Update kubernetes packages to v0.33.2 ([#410](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/410)) ([e598fb3](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/e598fb381fff2efc0022224d633949d0bb91157a))
* **deps:** Update kubernetes packages to v0.33.3 ([#450](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/450)) ([32a5539](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/32a5539c18c8b7e4b29a682986a765176e5e9d8f))
* **deps:** Update kubernetes packages to v0.33.4 ([#481](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/481)) ([423cd5f](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/423cd5fe3db5eaa0e4b4683714205ee367614c2a))
* **deps:** Update module github.com/cert-manager/cert-manager to v1.18.1 ([#401](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/401)) ([0769a28](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/0769a28a8ea4dceeb37f8627437cca7ab202339e))
* **deps:** Update module github.com/cloudnative-pg/api to v1.26.0 ([#440](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/440)) ([68dfd0e](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/68dfd0e75e666c265b2e95d228371acce31029c3))
* **deps:** Update module github.com/cloudnative-pg/cnpg-i-machinery to v0.4.0 ([#439](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/439)) ([e98facc](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/e98faccf7274e40dd8e6db021e7335444cb484a8))
* **deps:** Update module github.com/onsi/ginkgo/v2 to v2.25.0 ([#489](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/489)) ([5b67c11](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/5b67c11cd0509cd05537d2d9b78b5368bca6f649))
* **deps:** Update module google.golang.org/grpc to v1.73.0 ([#394](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/394)) ([1365906](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/1365906204d895cac78ef93d5753d0b5f717c9ac))
* **deps:** Update module google.golang.org/grpc to v1.75.0 ([#484](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/484)) ([86496ac](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/86496ac9992b4a47238e71aa884ab8bada38f520))
* **deps:** Update module sigs.k8s.io/kustomize/api to v0.20.0 ([#431](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/431)) ([d0013df](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/d0013dfe12d0ab25767ffe8d6a919992a1bea4d1))
* **deps:** Update module sigs.k8s.io/kustomize/api to v0.20.1 ([#471](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/471)) ([fa20c09](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/fa20c09525f09b52d5c09a89c3eaa05b0c1699cc))
* **images:** Use bookworm for sidecar image ([#476](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/476)) ([b264582](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/b2645827b8cd60fd8a149019d333271f75fb0874))
* Logic to retrieve ObjectStore from cache ([#429](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/429)) ([2a75d40](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/2a75d40356e31c09cc823f1edeff0e9b217f66d5))
* **unit-tests:** Metrics collect length ([#475](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/475)) ([e40ba70](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/e40ba7065a33237b2a95913ca968a01942a0eb3b))

## [0.5.0](https://github.com/cloudnative-pg/plugin-barman-cloud/compare/v0.4.1...v0.5.0) (2025-06-03)


### Features

* **deps:** Update dependency barman to v3.14.0 ([#368](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/368)) ([3550013](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/35500130bf0fe25eb3a191bc78f4818c318acf26))


### Bug Fixes

* Remove lifecycle `Pod` `Patch` subscription ([#378](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/378)) ([40316b5](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/40316b5f2d72deac0f042ceecd271a97b369a62f))

## [0.4.1](https://github.com/cloudnative-pg/plugin-barman-cloud/compare/v0.4.0...v0.4.1) (2025-05-29)


### Bug Fixes

* **deps:** Update all non-major go dependencies ([#366](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/366)) ([1097abb](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/1097abbd1d26502a3cfc81f932bffd5bef2377a4))
* **deps:** Update kubernetes packages to v0.33.1 ([#361](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/361)) ([9d4bc45](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/9d4bc456b09b9d79c1ad58f686c8201885ffe4ce))
* **deps:** Update module google.golang.org/grpc to v1.72.1 ([#345](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/345)) ([d9fd8dd](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/d9fd8dd8681e33ec64c911eade3516a73f793ac5))
* **deps:** Update module sigs.k8s.io/controller-runtime to v0.21.0 ([#367](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/367)) ([fecc2f7](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/fecc2f7d28e5ad58c6370f0a26014908ce4caaaf))
* Do not add barman-certificates projection if not needed ([#354](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/354)) ([918823d](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/918823dbf1c78e5460f83af50bf85be6c1aefafe))
* **docs:** Replace "no downtime" with "without data loss" ([#349](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/349)) ([5e1b845](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/5e1b845caedb67cf79173af3a319d55260b21627))

## [0.4.0](https://github.com/cloudnative-pg/plugin-barman-cloud/compare/v0.3.0...v0.4.0) (2025-05-12)


### Features

* Forbid usage of `.spec.configuration.serverName` in ObjectStore ([#336](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/336)) ([3420f43](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/3420f430739ac8518c83cd3b23bf6a8e42b411f7)), closes [#334](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/334)
* Log the downloaded backup catalog before restore ([#323](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/323)) ([9db184f](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/9db184f5d4c325ed18aeb4fba6c57c28b0e3ae40)), closes [#319](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/319)
* **sidecar:** Add resource requirements and limits ([#307](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/307)) ([4bb3471](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/4bb347121d3328783ca9eceb656863cde37cb8aa)), closes [#253](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/253)
* Support snapshot recovery job ([#258](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/258)) ([e00024f](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/e00024f136996305999c0440ae9b48861828e160))
* **wal:** Parallel WAL archiving ([#262](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/262)) ([88fd3e5](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/88fd3e504f35e004fab47ca33a2e67dd40120e2c)), closes [#260](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/260) [#266](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/266)


### Bug Fixes

* [#260](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/260) ([88fd3e5](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/88fd3e504f35e004fab47ca33a2e67dd40120e2c))
* [#266](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/266) ([88fd3e5](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/88fd3e504f35e004fab47ca33a2e67dd40120e2c))
* **deps:** Update all non-major go dependencies ([#246](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/246)) ([ed1feaa](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/ed1feaaddcddfabd48a2d9a28013e7585d8babd6))
* **deps:** Update all non-major go dependencies ([#278](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/278)) ([010c9b9](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/010c9b93d4e2d06eb89ba49219f15144c98515cf))
* **deps:** Update k8s.io/utils digest to 0f33e8f ([#301](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/301)) ([ab398d7](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/ab398d7d30ebe241b2b682c42c4b129254955b24))
* **deps:** Update kubernetes packages to v0.33.0 ([#281](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/281)) ([c6f36d5](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/c6f36d57562a99175e2d3d446ca2d7e7c36b09c3))
* **deps:** Update react monorepo to v19.1.0 ([#286](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/286)) ([99f31a1](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/99f31a1e5e0313534699c49393edc6beabac60ec))
* **docs:** Fix TOC links ([#261](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/261)) ([2bb5e90](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/2bb5e90357b2defd6fdaa8ff9982e21f58bc5ecc))
* Duplicate certificate projections ([#331](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/331)) ([8c20e4f](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/8c20e4fe8578b5b18277ce2ae8ba11783b1cac84)), closes [#329](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/329)
* Role patching ([#325](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/325)) ([f484b9e](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/f484b9e748ad776f7ecec0ed83a2b2424fde2dfc)), closes [#318](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/318)

## [0.3.0](https://github.com/cloudnative-pg/plugin-barman-cloud/compare/v0.2.0...v0.3.0) (2025-03-28)


### Features

* Generate apidoc using genref ([#228](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/228)) ([74bdb9a](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/74bdb9a590f169eade4eea27caa85fc3b1809e41)), closes [#206](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/206)
* Implement evaluate lifecycle hook ([#222](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/222)) ([a7ef56b](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/a7ef56b6e7a8abfcf312f42190b5c3828f9b2a79))
* Lenient decoding of CNPG resources ([#192](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/192)) ([13e3fab](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/13e3fab2688ec6ea342ed7304680025f98e6af27))
* Retention policy ([#191](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/191)) ([fecd1e9](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/fecd1e9513ce1748a289840f735a2f23a0ce5218))
* Support custom CA certificates ([#198](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/198)) ([fcbc472](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/fcbc47209222f712178ba422020c88eef7d50c08))
* Support lz4, xz, and zstandard compressions ([#201](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/201)) ([795313f](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/795313f4aa2f4888fdf2cb711de74aaea7b045a7)), closes [#200](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/200)
* Upgrade Barman to 3.13.0 ([#209](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/209)) ([56d8cce](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/56d8cceb3b8c7a17f3dcdd2dc14b48a725aaea9f)), closes [#208](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/208)


### Bug Fixes

* Controller and sidecar containers run as non-root ([#225](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/225)) ([5788c1f](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/5788c1f72794a331e9176dabc625a5937abff010)), closes [#177](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/177)
* Custom CA support for retention policies ([#224](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/224)) ([bac7b67](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/bac7b673a2ef239dd28bd2d1eced083009ad8ba6)), closes [#220](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/220)
* **deps:** Update all non-major go dependencies ([#213](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/213)) ([a5b8649](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/a5b8649bd0eac1df6e51291ff197a6a548d0f479))
* **deps:** Update all non-major go dependencies ([#219](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/219)) ([0d4a3d3](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/0d4a3d38f77e9d51a3f627fa768673e3c4b5e650))
* **deps:** Update k8s.io/utils digest to 1f6e0b7 ([#237](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/237)) ([792679f](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/792679ff673f60deeac3293d4bfb3e5182a09bef))
* **deps:** Update kubernetes packages to v0.32.3 ([#216](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/216)) ([9d22676](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/9d22676f2a5667b516a4f496ab6188a2333e5333))
* Use a fixed golangci-lint version ([#230](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/230)) ([78fe21b](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/78fe21b24dc9366c34260babe6b049a310abe9f0))

## [0.2.0](https://github.com/cloudnative-pg/plugin-barman-cloud/compare/v0.1.0...v0.2.0) (2025-03-05)


### Features

* Release-please cleanup ([#115](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/115)) ([cd03c55](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/cd03c556ef86c429b8699961eb24e1361b5759ff)), closes [#114](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/114)
* Support additional compression methods in the sidecar image ([#158](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/158)) ([ee5fd84](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/ee5fd840924c0997f301764af32a684aa8424b22)), closes [#127](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/127)


### Bug Fixes

* **deps:** Update all non-major go dependencies ([#103](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/103)) ([55258f6](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/55258f69008d1475f65d549d47a6c87485624e28))
* **deps:** Update all non-major go dependencies ([#152](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/152)) ([e77799a](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/e77799af028ba892ed8f3261554682c1b540a7f5))
* **deps:** Update github.com/cloudnative-pg/cloudnative-pg digest to 34ab236 ([#180](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/180)) ([e9e636a](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/e9e636ada08de4a1f6db0a31e2f133e703580394))
* **deps:** Update golang.org/x/net ([#188](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/188)) ([aba0748](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/aba07487891b731b6439429c7b30da21bc260d5f))
* **deps:** Update kubernetes packages to v0.32.1 ([#147](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/147)) ([dbc5550](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/dbc5550c9c503dfb0a6206a244995cdda9d28c1d))
* **deps:** Update kubernetes packages to v0.32.2 ([#172](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/172)) ([bb9658b](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/bb9658b28c95f9b7e1f202dcf2be76bff7756960))
* **deps:** Update module github.com/cloudnative-pg/api to v1 ([#131](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/131)) ([0c8ff74](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/0c8ff7426ff15623deba0c9603ba76dece3cb6a5))
* **deps:** Update module github.com/cloudnative-pg/cnpg-i-machinery to v0.1.2 ([#182](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/182)) ([12cd519](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/12cd5195234ee17ca0b09c2448cc9dc50c614149))
* **deps:** Update module google.golang.org/grpc to v1.71.0 ([#187](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/187)) ([e1f1660](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/e1f166023f55fb02d987ac011e3580af1f9d273a))
* **deps:** Update module sigs.k8s.io/kustomize/api to v0.19.0 ([#148](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/148)) ([9ba6351](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/9ba63518f929748f4a422eaa58293c8125b7a5f1))
* **deps:** Use latest commit from CNPG 1.25 branch ([#178](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/178)) ([dfbeaf8](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/dfbeaf802ec98357fdbb92b5fcefc38a29939cfe))

## 0.1.0 (2024-12-12)


### Features

* Add `liveness` and `readiness` probe support ([#69](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/69)) ([5fd9449](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/5fd9449b27394756e0baf76b1356900850f687a6))
* Additional environment variables ([#81](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/81)) ([be40375](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/be4037529c44858278dd80e3eb32f39f3f68c5c6))
* Backup method ([#20](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/20)) ([9fa1c0b](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/9fa1c0beab4882af3f4c737d049b5bafcf7e28a6))
* Grant permissions to read secrets ([#25](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/25)) ([76383a3](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/76383a30afd3bd829f01936dc3dfc81f1d189d2d))
* Operator plugin and manifests ([#18](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/18)) ([dd6548c](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/dd6548c4a26031324975d97aee345e4e6a2e7efa))
* Separate recovery and cluster object store ([#76](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/76)) ([e30edd2](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/e30edd2318d76e10fd7af344c0e4326f1e5033ec))
* Separate recovery object store from replica source ([#83](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/83)) ([e4735a2](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/e4735a2f85724cf8493f513658783e5330c3efcf))
* Sidecar injection and loading ([#22](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/22)) ([ea6ee30](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/ea6ee30d2ea30f9e9df22002ce5f5a68fcb37ade))
* Sidecar role and rolebinding ([#23](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/23)) ([2f62d53](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/2f62d539c949f344cb5534b7ffbb90860663a106))
* Restore ([#29](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/29)) ([240077c](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/240077c77192d9572767d7ec76d02e578b94faca))
* Wal-archive and wal-restore methods ([#4](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/4)) ([1c86ff6](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/1c86ff65747b5b348fb1ed2b0e5b0594fd156116))
