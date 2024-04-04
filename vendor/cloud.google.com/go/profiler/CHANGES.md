# Changes

## [0.4.0](https://github.com/googleapis/google-cloud-go/compare/profiler/v0.3.1...profiler/v0.4.0) (2023-10-18)


### Features

* **profiler:** Support configurable debug logging destination ([#8104](https://github.com/googleapis/google-cloud-go/issues/8104)) ([fc3d840](https://github.com/googleapis/google-cloud-go/commit/fc3d84058b8932152408bc3ee0a5584dfe0b0c19))
* **profiler:** Update all direct dependencies ([b340d03](https://github.com/googleapis/google-cloud-go/commit/b340d030f2b52a4ce48846ce63984b28583abde6))


### Bug Fixes

* **profiler:** Migrate to protobuf-go v2 ([#8730](https://github.com/googleapis/google-cloud-go/issues/8730)) ([deeb583](https://github.com/googleapis/google-cloud-go/commit/deeb58308cbbb033e46d478b4dc8766c6689e71e)), refs [#8585](https://github.com/googleapis/google-cloud-go/issues/8585)
* **profiler:** REST query UpdateMask bug ([df52820](https://github.com/googleapis/google-cloud-go/commit/df52820b0e7721954809a8aa8700b93c5662dc9b))
* **profiler:** Update golang.org/x/net to v0.17.0 ([174da47](https://github.com/googleapis/google-cloud-go/commit/174da47254fefb12921bbfc65b7829a453af6f5d))
* **profiler:** Update grpc to v1.55.0 ([1147ce0](https://github.com/googleapis/google-cloud-go/commit/1147ce02a990276ca4f8ab7a1ab65c14da4450ef))

## [0.3.1](https://github.com/googleapis/google-cloud-go/compare/profiler/v0.3.0...profiler/v0.3.1) (2022-12-02)


### Bug Fixes

* **profiler:** downgrade some dependencies ([7540152](https://github.com/googleapis/google-cloud-go/commit/754015236d5af7c82a75da218b71a87b9ead6eb5))

## [0.3.0](https://github.com/googleapis/google-cloud-go/compare/profiler/v0.2.0...profiler/v0.3.0) (2022-05-19)


### Bug Fixes

* **profiler:** relax service name regexp to allow service names starting with numbers to be used ([#5994](https://github.com/googleapis/google-cloud-go/issues/5994)) ([a1d8ac9](https://github.com/googleapis/google-cloud-go/commit/a1d8ac99b714d7df4923acbb794dbe04ce748013))


### Miscellaneous Chores

* **profiler:** use 0.3.0 as release ([#6030](https://github.com/googleapis/google-cloud-go/issues/6030)) ([7a90829](https://github.com/googleapis/google-cloud-go/commit/7a90829b62843a2cd38e6c1dfac35c137d33a40c))

## [0.2.0](https://github.com/googleapis/google-cloud-go/compare/profiler/v0.1.2...profiler/v0.2.0) (2022-02-14)


### Features

* **profiler:** add better version metadata to calls ([d1ad921](https://github.com/googleapis/google-cloud-go/commit/d1ad921d0322e7ce728ca9d255a3cf0437d26add))

### [0.1.2](https://www.github.com/googleapis/google-cloud-go/compare/profiler/v0.1.1...profiler/v0.1.2) (2022-01-04)


### Bug Fixes

* **profiler:** refine regular expression for parsing backoff duration in E2E tests ([#5229](https://www.github.com/googleapis/google-cloud-go/issues/5229)) ([4438aeb](https://www.github.com/googleapis/google-cloud-go/commit/4438aebca2ec01d4dbf22287aa651937a381e043))
* **profiler:** remove certificate expiration workaround ([#5222](https://www.github.com/googleapis/google-cloud-go/issues/5222)) ([2da36c9](https://www.github.com/googleapis/google-cloud-go/commit/2da36c95f44d5f88fd93cd949ab78823cea74fe7))

### [0.1.1](https://www.github.com/googleapis/google-cloud-go/compare/profiler/v0.1.0...profiler/v0.1.1) (2021-10-11)


### Bug Fixes

* **profiler:** workaround certificate expiration issue in integration tests ([#4955](https://www.github.com/googleapis/google-cloud-go/issues/4955)) ([de9e465](https://www.github.com/googleapis/google-cloud-go/commit/de9e465bea8cd0580c45e87d2cbc2b610615b363))

## v0.1.0

This is the first tag to carve out profiler as its own module. See
[Add a module to a multi-module repository](https://github.com/golang/go/wiki/Modules#is-it-possible-to-add-a-module-to-a-multi-module-repository).
