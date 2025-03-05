# Changelog

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
