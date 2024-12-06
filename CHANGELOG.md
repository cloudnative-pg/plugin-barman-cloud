# Changelog

## 0.1.0 (2024-12-06)


### Features

* add `liveness` and `readiness` probe support ([#69](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/69)) ([5fd9449](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/5fd9449b27394756e0baf76b1356900850f687a6))
* additional environment variables ([#81](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/81)) ([be40375](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/be4037529c44858278dd80e3eb32f39f3f68c5c6))
* grant permissions to read secrets ([#25](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/25)) ([76383a3](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/76383a30afd3bd829f01936dc3dfc81f1d189d2d))
* operator plugin and manifests ([#18](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/18)) ([dd6548c](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/dd6548c4a26031324975d97aee345e4e6a2e7efa))
* separate recovery and cluster object store ([#76](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/76)) ([e30edd2](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/e30edd2318d76e10fd7af344c0e4326f1e5033ec))
* sidecar injection and loading ([#22](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/22)) ([ea6ee30](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/ea6ee30d2ea30f9e9df22002ce5f5a68fcb37ade))
* sidecar role and rolebinding ([#23](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/23)) ([2f62d53](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/2f62d539c949f344cb5534b7ffbb90860663a106))
* **spike:** backup method ([#20](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/20)) ([9fa1c0b](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/9fa1c0beab4882af3f4c737d049b5bafcf7e28a6))
* **spike:** restore ([#29](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/29)) ([240077c](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/240077c77192d9572767d7ec76d02e578b94faca))
* **spike:** wal-archive and wal-restore methods ([#4](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/4)) ([1c86ff6](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/1c86ff65747b5b348fb1ed2b0e5b0594fd156116))


### Bug Fixes

* avoid injecting the plugin environment into the PG container ([#62](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/62)) ([9c77e3d](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/9c77e3de9f05a56c567c9fa6b0f75ca55a05ddf8))
* **deps:** update all non-major go dependencies ([#15](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/15)) ([3289d91](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/3289d91db4f924bad2f7f6dc8901f4544616233e))
* **deps:** update all non-major go dependencies ([#9](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/9)) ([435986b](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/435986b7a1e7bf9e5d4d1c018c37fd6e28f2aaa7))
* **deps:** update kubernetes packages to v0.31.1 ([#10](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/10)) ([76486c2](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/76486c28637fa10be3b8b5f260d5b626ac142ca4))
* **deps:** update kubernetes packages to v0.31.3 ([#64](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/64)) ([c639af1](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/c639af1295123c12d462d52b769ac0c973c22c93))
* **deps:** update module github.com/cert-manager/cert-manager to v1.16.2 [security] ([#63](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/63)) ([53d2c09](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/53d2c0999313b1447d873b27b1f87e1dd93c6e6a))
* **deps:** update module k8s.io/client-go to v0.31.1 ([#16](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/16)) ([cbefe26](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/cbefe26440203e88f8d60335b64f32b01249ba0d))
* **deps:** update module sigs.k8s.io/controller-runtime to v0.19.2 ([#67](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/67)) ([74d4f5d](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/74d4f5d1902ed557375adff3e775b35dd662d2fc))
* **deps:** update module sigs.k8s.io/kustomize/api to v0.18.0 ([#51](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/51)) ([b2d3032](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/b2d303205499ccca426fe2b72964eeefa6556fdd))
* ensure restore configuration points to manager `wal-restore` ([#68](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/68)) ([afd4603](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/afd4603023ce0f245687856eb05d9a30875b8bac))
* exit code 0 on clean shutdown ([#70](https://github.com/cloudnative-pg/plugin-barman-cloud/issues/70)) ([9d8fa07](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/9d8fa079fec6b82c5aef6397b4b6318fbe9ebb0b))
* obsolete deepcopy ([1e6c69b](https://github.com/cloudnative-pg/plugin-barman-cloud/commit/1e6c69bac022914732fbaabb5bae0969893f5049))
