# v1.12.1 Release - 07/14/2020

**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.12.1/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.12.1/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
 https://storage.googleapis.com/skaffold/releases/v1.12.1/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.12.1`

Fixes:
* Lenient parsing of `minikube docker-env` (#4421)
* Ignore comments in `minikuke docker-env` output (#4422)
* Debug supports `/bin/sh -c` and `/bin/bash -c` command-lines (#4442)
* When pulling images and authentication fails, first try anonymous pulling. (#4451)
* Propagate status error code to devloopEndEvent (#4468)

Huge thanks goes out to all of our contributors for this release:

- Brian de Alwis
- David Gageot
- Tejal Desai


# v1.12.0 Release - 06/25/2020

**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.12.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.12.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
 https://storage.googleapis.com/skaffold/releases/v1.12.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.12.0`


Highlights:
* `skaffold init` now supports Java and Python projects with Buildpacks projects
* A bunch of debug improvements to `skaffold debug`
* `skaffold render` can now render manifests from previous build results.


New Features:
* `skaffold init` supports Java and Python Buildpacks projects [#4318](https://github.com/GoogleContainerTools/skaffold/pull/4318) [#4309](https://github.com/GoogleContainerTools/skaffold/pull/4309)
* Add json output to `skaffold schema list` [#4385](https://github.com/GoogleContainerTools/skaffold/pull/4385)
* `debug` now supports buildpacks-produced images [#4375](https://github.com/GoogleContainerTools/skaffold/pull/4375)
* Skaffold deploy hydrated manifests [#4316](https://github.com/GoogleContainerTools/skaffold/pull/4316)
* Add option to render manifest from previous build result [#3567](https://github.com/GoogleContainerTools/skaffold/pull/3567)

Fixes:
* 'skaffold init' for Kustomize projects generates profiles for each overlay [#4349](https://github.com/GoogleContainerTools/skaffold/pull/4349)
* Fix proto generation [#4387](https://github.com/GoogleContainerTools/skaffold/pull/4387)
* Fix `skaffold init` for java projects. [#4379](https://github.com/GoogleContainerTools/skaffold/pull/4379)
* gracefull shutdown RPC even when build is in error [#4384](https://github.com/GoogleContainerTools/skaffold/pull/4384)
* Fix port forwarding on Windows [#4373](https://github.com/GoogleContainerTools/skaffold/pull/4373)
* Add debugHelpersRegistry property [#3945](https://github.com/GoogleContainerTools/skaffold/pull/3945)
* `debug` nodejs results can result in duplicated environment variables [#4360](https://github.com/GoogleContainerTools/skaffold/pull/4360)
* Enable file-watching for `debug` [#4089](https://github.com/GoogleContainerTools/skaffold/pull/4089)
* Fix `skaffold fix --version` [#4336](https://github.com/GoogleContainerTools/skaffold/pull/4336)
* [buildpacks] `debug` detect direct processes with `/bin/sh -c ...` [#4345](https://github.com/GoogleContainerTools/skaffold/pull/4345)
* Fix render not fully overwriting output files. [#4323](https://github.com/GoogleContainerTools/skaffold/pull/4323)
* chore: use setValues not values in helm docs example [#4334](https://github.com/GoogleContainerTools/skaffold/pull/4334)
* Fix propagation of buildpacks working directory [#4337](https://github.com/GoogleContainerTools/skaffold/pull/4337)
* Debug should report CNB_APP_DIR as working directory for buildpacks images [#4326](https://github.com/GoogleContainerTools/skaffold/pull/4326)
* Support mktemp on older Macs [#4319](https://github.com/GoogleContainerTools/skaffold/pull/4319)


Updates & Refactors:
* Refactor Add proto.ActionableErr to diag.Resource and deploy.Resource.Status [#4390](https://github.com/GoogleContainerTools/skaffold/pull/4390)
* create a constant for pushing image and use that to parse error codes [#4372](https://github.com/GoogleContainerTools/skaffold/pull/4372)
* add suggestion protos and hook up with Event API [#4364](https://github.com/GoogleContainerTools/skaffold/pull/4364)
* Extend `skaffold debug` integration tests to buildpacks [#4352](https://github.com/GoogleContainerTools/skaffold/pull/4352)
* Restore buildpacks-java integration test [#4365](https://github.com/GoogleContainerTools/skaffold/pull/4365)
* Improve the error message when a released schema is changed [#4355](https://github.com/GoogleContainerTools/skaffold/pull/4355)

Docs updates:
* Move jib sync testdata to `integration/examples` [#4367](https://github.com/GoogleContainerTools/skaffold/pull/4367)
* Fix docs and error message about pullSecretPath [#4381](https://github.com/GoogleContainerTools/skaffold/pull/4381)
* Tweaks to `debug` docs [#4369](https://github.com/GoogleContainerTools/skaffold/pull/4369)
* Print Custom Builder command [#4359](https://github.com/GoogleContainerTools/skaffold/pull/4359)
* [Docs] add an example for global config [#4341](https://github.com/GoogleContainerTools/skaffold/pull/4341)


Huge thanks goes out to all of our contributors for this release:

- Alex Lewis
- Andreas Sommer
- Appu Goundan
- Balint Pato
- Brian de Alwis
- Chanseok Oh
- Chris Ge
- David Gageot
- Gaurav
- Lennox Stevenson
- Nick Kubala
- Nils Breunese
- Stefan Büringer
- Tejal Desai
- tejal29

# v1.11.0 Release - 06/11/2020

Note: This release comes with a new config version `v2beta5`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.11.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.11.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
 https://storage.googleapis.com/skaffold/releases/v1.11.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.11.0`


Highlights:
- `skaffold render` now supports hydrating manifests from pre-existing images without building locally using the `--digest-source` flag
- Users can now provide custom annotations for Kaniko pods
- `IMAGE_REPO` and `IMAGE_TAG` runtimes values now exposed through environment variables in custom builds and helm deploys
- `skaffold render` now supports Helm projects


New Features:
* Skaffold render solely perform the manifests hydration [#4193](https://github.com/GoogleContainerTools/skaffold/pull/4193)
* Setup Github Actions for testing release binary against linux and darwin [#4300](https://github.com/GoogleContainerTools/skaffold/pull/4300)
* hook up showing survey prompt if not taken or recently prompted [#4306](https://github.com/GoogleContainerTools/skaffold/pull/4306)
* add annotations feature to kaniko pod [#4280](https://github.com/GoogleContainerTools/skaffold/pull/4280)
* Initial prototype for pod health check hook up [#4223](https://github.com/GoogleContainerTools/skaffold/pull/4223)
* Make IMAGE_REPO and IMAGE_TAG templated values in custom build and helm deploy [#4278](https://github.com/GoogleContainerTools/skaffold/pull/4278)
* add tolerations option for building image with kaniko [#4256](https://github.com/GoogleContainerTools/skaffold/pull/4256)
* [buildpacks] Custom project toml [#4265](https://github.com/GoogleContainerTools/skaffold/pull/4265)
* [buildpacks] Support trusted builders [#4273](https://github.com/GoogleContainerTools/skaffold/pull/4273)
* [buildpacks] Support buildpack version from project.toml [#4266](https://github.com/GoogleContainerTools/skaffold/pull/4266)
* [buildpacks] Initial support for project.toml [#4258](https://github.com/GoogleContainerTools/skaffold/pull/4258)
* Initial implementation of Helm Renderer [#3929](https://github.com/GoogleContainerTools/skaffold/pull/3929)
* Add user-friendly validation of builder/artifact compatibility [#4312](https://github.com/GoogleContainerTools/skaffold/pull/4312)


Fixes:
* Use `pullSecretPath` to set GOOGLE_APPLICATION_CREDENTIALS [#4147](https://github.com/GoogleContainerTools/skaffold/pull/4147)
* Use "helm version --client" to avoid connecting to cluster [#4294](https://github.com/GoogleContainerTools/skaffold/pull/4294)
* apply namespace from command first when cleaning up helm release [#4281](https://github.com/GoogleContainerTools/skaffold/pull/4281)
* move field reported to changed [#4222](https://github.com/GoogleContainerTools/skaffold/pull/4222)
* Fix support for Knative services [#4249](https://github.com/GoogleContainerTools/skaffold/pull/4249)
* Remote helm charts should not be upgraded by default [#3274](https://github.com/GoogleContainerTools/skaffold/pull/3274)
* Fix dockerfile resolution [#4260](https://github.com/GoogleContainerTools/skaffold/pull/4260)
* Add control API to pause and resume autoBuild, autoDeploy and autoSync [#4145](https://github.com/GoogleContainerTools/skaffold/pull/4145)


Updates & Refactors:
* Update GCP Buildpacks builder image references to :v1 [#4313](https://github.com/GoogleContainerTools/skaffold/pull/4313)
* upgrade to yaml.v3 [#4201](https://github.com/GoogleContainerTools/skaffold/pull/4201)
* Update jib to 2.4.0 [#4308](https://github.com/GoogleContainerTools/skaffold/pull/4308)
* Use pack’s code for reading project descriptors [#4298](https://github.com/GoogleContainerTools/skaffold/pull/4298)
* Rename `buildpack` config to `buildpacks` [#4290](https://github.com/GoogleContainerTools/skaffold/pull/4290)
* Update Bazel configuration [#4291](https://github.com/GoogleContainerTools/skaffold/pull/4291)
* Add validations to Control API for Auto Triggers [#4242](https://github.com/GoogleContainerTools/skaffold/pull/4242)
* Minor renames and change in the container status message. [#4284](https://github.com/GoogleContainerTools/skaffold/pull/4284)
* Collapse ImagePullBackOff and ErrImagePullBackOff together [#4269](https://github.com/GoogleContainerTools/skaffold/pull/4269)
* [buildpacks] Update to pack v0.11.0 [#4272](https://github.com/GoogleContainerTools/skaffold/pull/4272)
* Remove default from resource name [#4270](https://github.com/GoogleContainerTools/skaffold/pull/4270)


Docs updates:
* Render and Buildpacks support are Beta [#4275](https://github.com/GoogleContainerTools/skaffold/pull/4275)


Huge thanks goes out to all of our contributors for this release:
- Appu Goundan
- Balint Pato
- Brian de Alwis
- Chanseok Oh
- Chris Ge
- David Gageot
- David Hovey
- Gaurav
- Gwonsoo-Lee
- Hasso Mehide
- Mark Burnett
- Nick Kubala
- Tejal Desai
- Thomas Strömberg
- tete17

# v1.10.1 Hotfix Release - 05/20/2020

This is a hotfix release to address an issue with newer versions of Kustomize being broken, and to address an issue in our release process with malformed binaries.

* Revert "use kubectl's built-in kustomize when possible" [#4237](https://github.com/GoogleContainerTools/skaffold/pull/4237)
* Makefile: evaluate os/arch based on target name [#4236](https://github.com/GoogleContainerTools/skaffold/pull/4236)

**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.10.1/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.10.1/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
 https://storage.googleapis.com/skaffold/releases/v1.10.1/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.10.1`


# v1.10.0 Release - 05/19/2020

Note: This release comes with a new config version `v2beta4`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.10.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.10.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
 https://storage.googleapis.com/skaffold/releases/v1.10.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.10.0`


Highlights:
- Skaffold no longer requires a standalone Kustomize binary to be installed!
- Kustomize projects are now also supported in `skaffold init`
- This release also (re)adds support for ARM binaries :)

New Features:
* allow specifying regex in global config kubecontext [#4076](https://github.com/GoogleContainerTools/skaffold/pull/4076)
* Add MacPorts install command [#4157](https://github.com/GoogleContainerTools/skaffold/pull/4157)
* use kubectl's built-in kustomize when possible [#4183](https://github.com/GoogleContainerTools/skaffold/pull/4183)
* Logger now recognises images with default-repo [#4178](https://github.com/GoogleContainerTools/skaffold/pull/4178)
* Cross compile for linux-arm [#4151](https://github.com/GoogleContainerTools/skaffold/pull/4151)
* support Kustomize projects in `skaffold init` [#3925](https://github.com/GoogleContainerTools/skaffold/pull/3925)

Fixes:
* Exclude tmpcharts folder and helm generated .lock files from list of watched files [#4181](https://github.com/GoogleContainerTools/skaffold/pull/4181)
* Retrieve the proper kind cluster name [#4212](https://github.com/GoogleContainerTools/skaffold/pull/4212)
* Return error from 'helm get' rather than swallowing result [#4173](https://github.com/GoogleContainerTools/skaffold/pull/4173)
* Rely on LogAggregator’s Zero value [#4199](https://github.com/GoogleContainerTools/skaffold/pull/4199)
* CNB command-line should only be rewritten if changed [#4176](https://github.com/GoogleContainerTools/skaffold/pull/4176)
* A default deployer is guaranteed to be set [#4204](https://github.com/GoogleContainerTools/skaffold/pull/4204)
* Don’t store portForwardResources [#4202](https://github.com/GoogleContainerTools/skaffold/pull/4202)
* allow error from kind at image loading to propagate [#4196](https://github.com/GoogleContainerTools/skaffold/pull/4196)
* Return early like for createContainerManager() [#4190](https://github.com/GoogleContainerTools/skaffold/pull/4190)
* Don’t duplicate the definition of `portForward`, under the `profiles` section. [#4165](https://github.com/GoogleContainerTools/skaffold/pull/4165)

Updates & Refactors:
* Simpler code for changeset [#4217](https://github.com/GoogleContainerTools/skaffold/pull/4217)

* Always print `ctrl-c` message [#4214](https://github.com/GoogleContainerTools/skaffold/pull/4214)
* Small `diag` improvements [#4219](https://github.com/GoogleContainerTools/skaffold/pull/4219)
* Remove duplication around `kubectlCLI` [#4215](https://github.com/GoogleContainerTools/skaffold/pull/4215)
* Better handling of per-command default values [#4209](https://github.com/GoogleContainerTools/skaffold/pull/4209)
* Simplify code to set intents up [#4211](https://github.com/GoogleContainerTools/skaffold/pull/4211)
* Rename Values to artifactOverrides [#4169](https://github.com/GoogleContainerTools/skaffold/pull/4169)
* Show suggestions for every command [#4206](https://github.com/GoogleContainerTools/skaffold/pull/4206)
* Recognise *.gcr.io default-repo in suggestions [#4208](https://github.com/GoogleContainerTools/skaffold/pull/4208)
* Common flags: simpler code and no init() function [#4200](https://github.com/GoogleContainerTools/skaffold/pull/4200)
* Move `imagesAreLocal` logic to where it belongs [#4203](https://github.com/GoogleContainerTools/skaffold/pull/4203)
* Use a single flag for log tailing [#4189](https://github.com/GoogleContainerTools/skaffold/pull/4189)
* improve deployment waiting logic in integration tests [#4162](https://github.com/GoogleContainerTools/skaffold/pull/4162)
* Show message "Press ctrl c to exit" on forward manager start [#4113](https://github.com/GoogleContainerTools/skaffold/pull/4113)
* Make sure log tailing works with pods and deployments [#4119](https://github.com/GoogleContainerTools/skaffold/pull/4119)

Docs updates:
* Change the order of properties in the doc [#4184](https://github.com/GoogleContainerTools/skaffold/pull/4184)
* Fix typo in development guide [#4152](https://github.com/GoogleContainerTools/skaffold/pull/4152)
* Update DEVELOPMENT.md on making changes to the skaffold api [#4127](https://github.com/GoogleContainerTools/skaffold/pull/4127)


Huge thanks goes out to all of our contributors for this release:
- Balint Pato
- Brian de Alwis
- Daniel Sel
- David Gageot
- Gaurav Ghosh
- Nick Kubala
- Nils Breunese
- Tejal Desai
- Thomas Strömberg


# v1.9.1 Hotfix Release - 05/07/2020

This is a hotfix release to address an issue with tailing logs while deploying with Helm, and to avoid an issue with authentication while building with Kaniko in GCB.

* Revert "Only listen to pods for the current RunID" [#4122](https://github.com/GoogleContainerTools/skaffold/pull/4122)
* Pin to kaniko v0.20.0 [#4128](https://github.com/GoogleContainerTools/skaffold/pull/4128)

**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.9.1/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.9.1/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
 https://storage.googleapis.com/skaffold/releases/v1.9.1/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.9.1`


# v1.9.0 Release - 05/05/2020

Note: This release comes with a new config version `v2beta3`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.9.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.9.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
 https://storage.googleapis.com/skaffold/releases/v1.9.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.9.0`


Highlights:
* Skaffold should now correctly debug NodeJS applications!
* Buildpacks now support Auto Sync, and debugging is enabled
* `skaffold diagnose` takes a `--yaml-only` flag to print the effective skaffold.yaml
* Git tagger now supports prefixing
* Auto-activated profiles can now be disabled with `--profile-auto-activation`
* Port-forwarding rules are now processed in sequence
* `skaffold fix` now takes an optional target schema version
* `skaffold build` now supports `--dry-run`
* `skaffold survey` added to open our user-feedback survey
* Added several fail fast conditions so initial errors are surfaced much quicker
* Error messages are becoming much simpler - this is a WIP!

New Features:
* Add events to indicate start and end of skaffold dev iterations [#4037](https://github.com/GoogleContainerTools/skaffold/pull/4037)
* Print the effective skaffold.yaml configuration [#4048](https://github.com/GoogleContainerTools/skaffold/pull/4048)
* git tagger now supports an optional prefix [#4049](https://github.com/GoogleContainerTools/skaffold/pull/4049)
* Support `skaffold fix —version skaffold/v1` [#4016](https://github.com/GoogleContainerTools/skaffold/pull/4016)
* Add a dry-run to `skaffold build` [#4039](https://github.com/GoogleContainerTools/skaffold/pull/4039)
* Add a new survey command to show Skaffold User Survey form url. [#3733](https://github.com/GoogleContainerTools/skaffold/pull/3733)
* Add CLI option `--profile-auto-activation` to allow disabling automatic profile activation. [#4034](https://github.com/GoogleContainerTools/skaffold/pull/4034)
* skaffold render --output takes GCS file path  [#3979](https://github.com/GoogleContainerTools/skaffold/pull/3979)
* Add pod checks [#3952](https://github.com/GoogleContainerTools/skaffold/pull/3952)
* First draft for adding actionable error items Framework [#4045](https://github.com/GoogleContainerTools/skaffold/pull/4045)
* Add codes for error types and detect terminated containers [#4012](https://github.com/GoogleContainerTools/skaffold/pull/4012)
* Buildpacks support Auto sync [#4079](https://github.com/GoogleContainerTools/skaffold/pull/4079)
* Disable profiles with the command line [#4054](https://github.com/GoogleContainerTools/skaffold/pull/4054)

Fixes:
* `--dry-run=client` must replace `--dry-run=true` with kubectl >= 1.18 [#4096](https://github.com/GoogleContainerTools/skaffold/pull/4096)
* fix status check event error reporting [#4101](https://github.com/GoogleContainerTools/skaffold/pull/4101)
* Fix default-repo handling for `skaffold deploy` [#4074](https://github.com/GoogleContainerTools/skaffold/pull/4074)
* Prevent the cache from sending “Build in progress” events. [#4038](https://github.com/GoogleContainerTools/skaffold/pull/4038)
* Skip podspecs that already have a debug.cloud.google.com/config annotation [#4027](https://github.com/GoogleContainerTools/skaffold/pull/4027)
* Always use the RunId overridden with an env var [#3985](https://github.com/GoogleContainerTools/skaffold/pull/3985)
* Use Go 1.14.2 to prevent SIGILL: illegal instruction on macOS [#4009](https://github.com/GoogleContainerTools/skaffold/pull/4009)
* Gracefully shutdown RPC servers. [#4010](https://github.com/GoogleContainerTools/skaffold/pull/4010)
* When a tagger fails, use a fallback tagger [#4019](https://github.com/GoogleContainerTools/skaffold/pull/4019)
* Support --default-repo=‘’ to erase the value from global config [#3990](https://github.com/GoogleContainerTools/skaffold/pull/3990)
* Run container-structure-test on remote images [#3983](https://github.com/GoogleContainerTools/skaffold/pull/3983)
* Fix nodemon versions [#4015](https://github.com/GoogleContainerTools/skaffold/pull/4015)
* Fail when cache check should have succeeded [#3996](https://github.com/GoogleContainerTools/skaffold/pull/3996)
* Fail fast if the Dockerfile can’t be found [#3999](https://github.com/GoogleContainerTools/skaffold/pull/3999)
* [json schema] When we don’t know a field’s type, let’s leave it empty [#3964](https://github.com/GoogleContainerTools/skaffold/pull/3964)
* ResourceType is of type string [#3987](https://github.com/GoogleContainerTools/skaffold/pull/3987)
* Don’t replace existing labels [#4105](https://github.com/GoogleContainerTools/skaffold/pull/4105)

Updates & Refactors:
* Use `node` wrapper to debug NodeJS apps [#4086](https://github.com/GoogleContainerTools/skaffold/pull/4086)
* add serviceAccount and runAsUser to kaniko build (resolves #3267) [#3965](https://github.com/GoogleContainerTools/skaffold/pull/3965)
* Only listen to pods for the current RunID [#4097](https://github.com/GoogleContainerTools/skaffold/pull/4097)
* Pin the version of Ko in Custom Example [#4099](https://github.com/GoogleContainerTools/skaffold/pull/4099)
* Use NODEJS_VERSION and NODE_ENV in detection [#4021](https://github.com/GoogleContainerTools/skaffold/pull/4021)
* Change default buildpacks [#4070](https://github.com/GoogleContainerTools/skaffold/pull/4070)
* Handle port forwarding rules in sequence [#4053](https://github.com/GoogleContainerTools/skaffold/pull/4053)
* Support Google Cloud Build logging options [#4043](https://github.com/GoogleContainerTools/skaffold/pull/4043)
* Fail fast when k8s is not reachable [#4031](https://github.com/GoogleContainerTools/skaffold/pull/4031)
* Fail fast if minikube is used but not started [#4042](https://github.com/GoogleContainerTools/skaffold/pull/4042)
* Introduce v2beta3 [#4029](https://github.com/GoogleContainerTools/skaffold/pull/4029)
* Update to Helm 3 in builder image [#4020](https://github.com/GoogleContainerTools/skaffold/pull/4020)
* For upgrades, direct users to the GitHub release page [#4024](https://github.com/GoogleContainerTools/skaffold/pull/4024)
* [kaniko] Better error message when upload fails [#4023](https://github.com/GoogleContainerTools/skaffold/pull/4023)
* Initial draft for sending skaffold metrics using metadata event [#3966](https://github.com/GoogleContainerTools/skaffold/pull/3966)
* Validate generated json schema [#3976](https://github.com/GoogleContainerTools/skaffold/pull/3976)
* Changing test config invalidates the build cache [#3984](https://github.com/GoogleContainerTools/skaffold/pull/3984)
* Simplify error messages [#3997](https://github.com/GoogleContainerTools/skaffold/pull/3997)

Docs updates:
* [doc] Explain how buildArgs are used by custom builder. [#4077](https://github.com/GoogleContainerTools/skaffold/pull/4077)
* Add link-able anchors to skaffold.yaml docs [#4052](https://github.com/GoogleContainerTools/skaffold/pull/4052)
* Clarify which containers log tailing works with [#4078](https://github.com/GoogleContainerTools/skaffold/pull/4078)
* Update 2020 Roadmap [#3939](https://github.com/GoogleContainerTools/skaffold/pull/3939)
* Improve GCB docs to include a table of properties [#3989](https://github.com/GoogleContainerTools/skaffold/pull/3989)
* install docs: use "install" and "choco -y" [#3992](https://github.com/GoogleContainerTools/skaffold/pull/3992)
* Add docs for configuring helm project with skaffold [#3973](https://github.com/GoogleContainerTools/skaffold/pull/3973)


Huge thanks goes out to all of our contributors for this release:
- Balint Pato
- Brian de Alwis
- Chanseok Oh
- Chris Ge
- Daniel Sel
- David Gageot
- Marcin
- Max Goltzsche
- Michael Parker
- Nick Kubala
- Pedro de Brito
- Tejal Desai
- Thomas Strömberg
- gsquared94
- knv srinivas


# v1.8.0 Release - 04/17/2020

Note: This release comes with a new config version `v2beta2`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Fixes:
* Whitelist recursively transformable kinds. [#3833](https://github.com/GoogleContainerTools/skaffold/pull/3833)
* Update error message to use `are` since `manifests` is plural [#3930](https://github.com/GoogleContainerTools/skaffold/pull/3930)
* Correctly set namespace when checking for an existing helm release via skaffold deploy [#3914](https://github.com/GoogleContainerTools/skaffold/pull/3914)
* Disable Python detector's use of PYTHON_VERSION [#3919](https://github.com/GoogleContainerTools/skaffold/pull/3919)

Updates & Refactors:
* Upgrade Jib to 2.2.0 [#3971](https://github.com/GoogleContainerTools/skaffold/pull/3971)
* Bump kubernetes to 1.14 and all other k8 deps to 0.17.0 [#3938](https://github.com/GoogleContainerTools/skaffold/pull/3938)
* Update pack image to v0.10.0 [#3956](https://github.com/GoogleContainerTools/skaffold/pull/3956)
* Introduce v2beta2 [#3942](https://github.com/GoogleContainerTools/skaffold/pull/3942)
* Refactoring on filepath.Walk [#3885](https://github.com/GoogleContainerTools/skaffold/pull/3885)
* Simplify Debug Transformer interface and allow Apply to fail on images [#3931](https://github.com/GoogleContainerTools/skaffold/pull/3931)
* Add error codes to event api to send error codes to Skaffold Event API [#3954](https://github.com/GoogleContainerTools/skaffold/pull/3954)

Docs updates:
* Update docs to point to new 2020 roadmap [#3924](https://github.com/GoogleContainerTools/skaffold/pull/3924)
* Add Kustomize example with an image built by skaffold [#3901](https://github.com/GoogleContainerTools/skaffold/pull/3901)
* Update VS Code Go launch snippet [#3950](https://github.com/GoogleContainerTools/skaffold/pull/3950)
* Improve Debug's Go docs [#3949](https://github.com/GoogleContainerTools/skaffold/pull/3949)

Thanks goes out to all of our contributors for this release:

- Balint Pato
- Brian de Alwis
- Chanseok Oh
- David Gageot
- Max Goltzsche
- Michael Parker
- Nick Kubala
- Pedro de Brito
- Tejal Desai

# v1.7.0 Release - 04/02/2020

Highlights: 
* kustomize dependencies include environment files (#3720) [#3721](https://github.com/GoogleContainerTools/skaffold/pull/3721)
* Support globs in custom/buildpacks builder deps [#3878](https://github.com/GoogleContainerTools/skaffold/pull/3878)

Note: 
* we had to revert the ARM support as it broke our release process, we will soon submit a fixed version 

Fixes: 
* Fix GCB build failure for multi-module Jib projects [#3852](https://github.com/GoogleContainerTools/skaffold/pull/3852)
* Fix possible nil dereference [#3869](https://github.com/GoogleContainerTools/skaffold/pull/3869)
* Fix console output for internal Jib tasks/goals [#3880](https://github.com/GoogleContainerTools/skaffold/pull/3880)
* Fix go test helper [#3859](https://github.com/GoogleContainerTools/skaffold/pull/3859)

Updates & Refactors:
* Better status check [#3892](https://github.com/GoogleContainerTools/skaffold/pull/3892)
* disable jib gradle in skaffold init by default [#3906](https://github.com/GoogleContainerTools/skaffold/pull/3906)
* Use new name for the linter’s cache [#3894](https://github.com/GoogleContainerTools/skaffold/pull/3894)
* Use less memory for linting [#3888](https://github.com/GoogleContainerTools/skaffold/pull/3888)
* Simplify Kaniko error message [#3870](https://github.com/GoogleContainerTools/skaffold/pull/3870)
* Wait for the logs to be printed [#3877](https://github.com/GoogleContainerTools/skaffold/pull/3877)
* Master Keychain [#3865](https://github.com/GoogleContainerTools/skaffold/pull/3865)
* Replace errors.Wrap with %w [#3860](https://github.com/GoogleContainerTools/skaffold/pull/3860)
* Show compilation errors [#3866](https://github.com/GoogleContainerTools/skaffold/pull/3866)
* Cobra context [#3842](https://github.com/GoogleContainerTools/skaffold/pull/3842)
* Format `go test` output with Go rather than bash and jq [#3853](https://github.com/GoogleContainerTools/skaffold/pull/3853)

Design proposals: 
* Update debug-events design proposal status [#3874](https://github.com/GoogleContainerTools/skaffold/pull/3874)


Docs updates: 
* Rework debug docs [#3875](https://github.com/GoogleContainerTools/skaffold/pull/3875)
* Fix of documentation issue #3266 microservices example is broken [#3867](https://github.com/GoogleContainerTools/skaffold/pull/3867)
* [docs] [release] fix firebase-tools version [#3857](https://github.com/GoogleContainerTools/skaffold/pull/3857)
* [examples] upgrade nodejs example dependencies [#3858](https://github.com/GoogleContainerTools/skaffold/pull/3858)
* Fix doc link to local cluster info [#3856](https://github.com/GoogleContainerTools/skaffold/pull/3856)
* upgrade hugo + small fixes [#3854](https://github.com/GoogleContainerTools/skaffold/pull/3854)

Huge thanks goes out to all of our contributors for this release:

- Balint Pato
- Brian de Alwis
- David Gageot
- Dmitry Stoyanov
- Nick Kubala
- Nick Novitski
- Tad Cordle
- tejal29

# v1.6.0 Release - 03/19/2020

*Note*: This release comes with a new config version `v2beta1`. To upgrade your `skaffold.yaml`, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights: 
* Support Dockerfile.dockerignore [#3837](https://github.com/GoogleContainerTools/skaffold/pull/3837)
* Add new Auto sync option [#3382](https://github.com/GoogleContainerTools/skaffold/pull/3382)
* Cross compile for linux-arm [#3819](https://github.com/GoogleContainerTools/skaffold/pull/3819)

Fixes: 

* Fix issues in `skaffold init` [#3840](https://github.com/GoogleContainerTools/skaffold/pull/3840)
* Fix `skaffold debug` panic with nodejs [#3827](https://github.com/GoogleContainerTools/skaffold/pull/3827)
* Fix this integration test on minikube [#3807](https://github.com/GoogleContainerTools/skaffold/pull/3807)
* Fix `make quicktest` [#3820](https://github.com/GoogleContainerTools/skaffold/pull/3820)
* Fix TestWaitForPodSucceeded flake [#3818](https://github.com/GoogleContainerTools/skaffold/pull/3818)
* Fix ko sample [#3805](https://github.com/GoogleContainerTools/skaffold/pull/3805)

Updates & Refactors:
* Add dependabot config file [#3832](https://github.com/GoogleContainerTools/skaffold/pull/3832)
* Upgrade kompose to 1.21.0 [#3806](https://github.com/GoogleContainerTools/skaffold/pull/3806)
* Go 1.14 [#3775](https://github.com/GoogleContainerTools/skaffold/pull/3775)
* add flag --survey to set to set/unset disable survey prompt [#3732](https://github.com/GoogleContainerTools/skaffold/pull/3732)
* Bump schema to v2beta1 [#3809](https://github.com/GoogleContainerTools/skaffold/pull/3809)
* [Diagnostics] Add validator interface. Add resource interface and PodValidator [#3742](https://github.com/GoogleContainerTools/skaffold/pull/3742)

Docs Updates:

* Simplify code that finds the artifact's type [#3825](https://github.com/GoogleContainerTools/skaffold/pull/3825)
* Use new t.Cleanup() to simplify tests [#3815](https://github.com/GoogleContainerTools/skaffold/pull/3815)
* cleanup common flags + better -f description [#3786](https://github.com/GoogleContainerTools/skaffold/pull/3786)
* unhide status check and on by default [#3792](https://github.com/GoogleContainerTools/skaffold/pull/3792)
* Normalize capitalization for types while port forwarding [#3803](https://github.com/GoogleContainerTools/skaffold/pull/3803)
* Also clean up statik files [#3804](https://github.com/GoogleContainerTools/skaffold/pull/3804)

Huge thanks goes out to all of our contributors for this release:

- Agrin
- Appu Goundan
- Balint Pato
- Brian de Alwis
- Daniel Abdelsamed
- David Gageot
- Nick Kubala
- Tejal Desai

# v1.5.0 Release - 03/05/2020

Highlights:
* Add helm3 support to the helm deployer [#3738](https://github.com/GoogleContainerTools/skaffold/pull/3738)
* Binaries for linux-arm #2068 [#3783](https://github.com/GoogleContainerTools/skaffold/pull/3783)

New Features:
* Autogenerate k8s manifests in skaffold init [#3703](https://github.com/GoogleContainerTools/skaffold/pull/3703)
* Support go Templates in Custom Builder commands [#3754](https://github.com/GoogleContainerTools/skaffold/pull/3754)
* Wire up debug events [#3645](https://github.com/GoogleContainerTools/skaffold/pull/3645)
* Support inferred sync on Custom artifacts with a Dockerfile [#3752](https://github.com/GoogleContainerTools/skaffold/pull/3752)

Fixes:
* Fix analyze update check [#3722](https://github.com/GoogleContainerTools/skaffold/pull/3722)
* report actual copy error when syncing files to containers [#3715](https://github.com/GoogleContainerTools/skaffold/pull/3715)
* skip large files during skaffold init [#3717](https://github.com/GoogleContainerTools/skaffold/pull/3717)

Updates & Refactors:
* Upgrade Jib to 2.1.0 [#3728](https://github.com/GoogleContainerTools/skaffold/pull/3728)
* Bump pack to 0.9.0 [#3776](https://github.com/GoogleContainerTools/skaffold/pull/3776)
* Use heroku/color for our colors [#3757](https://github.com/GoogleContainerTools/skaffold/pull/3757)
* skaffold init and buildpacks: skip dependencies [#3758](https://github.com/GoogleContainerTools/skaffold/pull/3758)
* Faster make v2 [#3724](https://github.com/GoogleContainerTools/skaffold/pull/3724)
* Allow Sync for non-root containers-hotreload example [#3680](https://github.com/GoogleContainerTools/skaffold/pull/3680)
* Add profile option to RunBuilder in test helper [#3761](https://github.com/GoogleContainerTools/skaffold/pull/3761)
* helm chart packaging: improve errors, logic & testability [#3743](https://github.com/GoogleContainerTools/skaffold/pull/3743)
* Refactor helm deployer to prepare for helm3 support [#3729](https://github.com/GoogleContainerTools/skaffold/pull/3729)

Docs Updates:
* Link config management doc [#3723](https://github.com/GoogleContainerTools/skaffold/pull/3723)

Huge thanks goes out to all of our contributors for this release:

- Appu
- Balint Pato
- Brian de Alwis
- Daniel Abdelsamed
- Chanseok Oh
- David Gageot
- Idan Bidani
- Nick Kubala
- shlo
- Tejal Desai
- Thomas Strömberg


# v1.4.0 Release - 02/20/2020

*Note*: This release comes with a new config version `v2alpha4`. To upgrade your `skaffold.yaml`, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:
* Add 2020 Roadmap [#3684](https://github.com/GoogleContainerTools/skaffold/pull/3684)

Fixes: 
* Allow 'make test' to work for users who do not have jq installed [#3696](https://github.com/GoogleContainerTools/skaffold/pull/3696)
* retry pruning when skaffold could not prune local images due running containers [#3643](https://github.com/GoogleContainerTools/skaffold/pull/3643)
* Fix npe when resetting status check state [#3658](https://github.com/GoogleContainerTools/skaffold/pull/3658)
* fix nilpointer with skaffold init --skip-build [#3657](https://github.com/GoogleContainerTools/skaffold/pull/3657)
* Support kustomize "extended" patches. #2909 [#3663](https://github.com/GoogleContainerTools/skaffold/pull/3663)

Updates & Refactors:
* Faster Makefile [#3706](https://github.com/GoogleContainerTools/skaffold/pull/3706)
* Simpler code dealing with durations [#3709](https://github.com/GoogleContainerTools/skaffold/pull/3709)
* Update Jib to 2.0.0 [#3707](https://github.com/GoogleContainerTools/skaffold/pull/3707)
* Reduce default status check deadline to 2 mins [#3687](https://github.com/GoogleContainerTools/skaffold/pull/3687)
* move unused pod validator code to pkg/diag [#3704](https://github.com/GoogleContainerTools/skaffold/pull/3704)
* hidden --minikube-profile flag [#3691](https://github.com/GoogleContainerTools/skaffold/pull/3691)
* [refactor] make DoInit() a proper controller [#3682](https://github.com/GoogleContainerTools/skaffold/pull/3682)
* a hidden flag for simpler access to new init format [#3660](https://github.com/GoogleContainerTools/skaffold/pull/3660)
* Disable all colors in Buildpacks’s output when not in a terminal [#3651](https://github.com/GoogleContainerTools/skaffold/pull/3651)
* Build skaffold-builder image from a pre-pushed base [#3631](https://github.com/GoogleContainerTools/skaffold/pull/3631)
* Update pack image to v0.8.1 [#3629](https://github.com/GoogleContainerTools/skaffold/pull/3629)
* customizable jib feature minimum requirements [#3628](https://github.com/GoogleContainerTools/skaffold/pull/3628)

Docs Updates: 
* Add 2020 Roadmap [#3684](https://github.com/GoogleContainerTools/skaffold/pull/3684)

Huge thanks goes out to all of our contributors for this release:

- Appu Goundan
- Balint Pato
- Brian de Alwis
- David Gageot
- David Hovey
- Max Resnick
- Nick Kubala
- Tejal Desai
- Thomas Strömberg


# v1.3.1 Release - 01/31/2020

This is a minor release to fix skaffold image `gcr.io/k8s-skaffold/skaffold:v1.3.0` issue [#3622](https://github.com/GoogleContainerTools/skaffold/issues/3622)

No changes since [v1.3.0](#v130-release---01302020)

# v1.3.0 Release - 01/30/2020

*Note*: This release comes with a new config version `v2alpha3`. To upgrade your `skaffold.yaml`, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best it can.

Highlights:
* Enable multiple kustomizations in the kustomize deployer [#3585](https://github.com/GoogleContainerTools/skaffold/pull/3585)
* Add `--kubernetes-manifest` flag to `skaffold init` to  
   - turn off auto detection for manifests and 
   - initialize deploy stanza with given flag value.
* An empty sync config `sync: {}` will sync all files in the artifact workspace and infer destination [#3496](https://github.com/GoogleContainerTools/skaffold/pull/3496)
* Configure on cluster builds to use random postfix when creating following secrets
   - Docker config secret name via `randomDockerConfigSecret` and
   - Pull secret name via `randomPullSecret`

New Features: 
* Add `--label` flag to `skaffold render`[#3558](https://github.com/GoogleContainerTools/skaffold/pull/3558)
* Support `—-buildpack` flags on GCB [#3606](https://github.com/GoogleContainerTools/skaffold/pull/3606)
* Support specific buildpacks for buildpack artifact [#3584](https://github.com/GoogleContainerTools/skaffold/pull/3584)
* Add new config `disableValidation` to kubectl deploy config to disable validation [#3512](https://github.com/GoogleContainerTools/skaffold/pull/3512)
* Implements setting environment variable in kaniko pod #3227 [#3287](https://github.com/GoogleContainerTools/skaffold/pull/3287)
* Auto sync with Buildpacks [#3555](https://github.com/GoogleContainerTools/skaffold/pull/3555)

Fixes: 
* Encode artifact image-name and container WORKDIR in container debug info [#3564](https://github.com/GoogleContainerTools/skaffold/pull/3564)
* Better detection if user is running from terminals. [#3611](https://github.com/GoogleContainerTools/skaffold/pull/3611)
* Try use the Google hosted mirror of Maven Central [#3608](https://github.com/GoogleContainerTools/skaffold/pull/3608)
* Better output for Docker commands [#3607](https://github.com/GoogleContainerTools/skaffold/pull/3607)
* Fix nil pointer dereference when no account is set on gcloud. [#3597](https://github.com/GoogleContainerTools/skaffold/pull/3597)
* Better error reporting for unrecognized builder error [#3595](https://github.com/GoogleContainerTools/skaffold/pull/3595)
* Init command fixes
   - no error in skaffold init if pre-existing skaffold.yaml is different from target file [#3575](https://github.com/GoogleContainerTools/skaffold/pull/3575)
   - `skip-build` flag shouldn't detect builders [#3528](https://github.com/GoogleContainerTools/skaffold/pull/3528)
* Automatically handle —no-pull option on `pack`. [#3576](https://github.com/GoogleContainerTools/skaffold/pull/3576)

Updates & Refactors:
* Use the same Docker client across Skaffold [#3602](https://github.com/GoogleContainerTools/skaffold/pull/3602)
* Better k8s manifest parsing for `skaffold init` [#3531](https://github.com/GoogleContainerTools/skaffold/pull/3531)
* Update dependencies
   - golangcilint [#3534](https://github.com/GoogleContainerTools/skaffold/pull/3534)
   - cli tools [#3553](https://github.com/GoogleContainerTools/skaffold/pull/3553)
   - pack to v0.8.1 [#3593](https://github.com/GoogleContainerTools/skaffold/pull/3593)
* Add verbosity flag to go tests on travis [#3548](https://github.com/GoogleContainerTools/skaffold/pull/3548)
* Add unit test for `findRunImage` [#3560](https://github.com/GoogleContainerTools/skaffold/pull/3560)
* Simpler artifact hasher [#3591](https://github.com/GoogleContainerTools/skaffold/pull/3591)
* Build skaffold-builder image from a pre-pushed base [#3433](https://github.com/GoogleContainerTools/skaffold/pull/3433)
* A bunch of refactor to init code
  - [init refactor] cleanup on analyzers and moving things into a single package [#3538](https://github.com/GoogleContainerTools/skaffold/pull/3538)
  - [init refactor] introducing init analyzers [#3533](https://github.com/GoogleContainerTools/skaffold/pull/3533)
  - simplify init walk logic and many more.

Docs Updates: 
* Initial auto sync support design doc [#2901](https://github.com/GoogleContainerTools/skaffold/pull/2901)
* Design proposal for new Debug Events [#3122](https://github.com/GoogleContainerTools/skaffold/pull/3122)
* migrate Deployment in examples from extensions/v1beta1 to apps/v1 [#3572](https://github.com/GoogleContainerTools/skaffold/pull/3572)
* Fix invalid package comments [#3589](https://github.com/GoogleContainerTools/skaffold/pull/3589)
* Fixes the command for switching to getting-started dir after cloning  [#3574](https://github.com/GoogleContainerTools/skaffold/pull/3574)
* Add Ruby/Rack application example with hot reload [#3515](https://github.com/GoogleContainerTools/skaffold/pull/3515)

Huge thanks goes out to all of our contributors for this release:

- Andrei Balici
- Appu
- Appu Goundan
- arminbuerkle
- Balint Pato
- balopat
- Brian de Alwis
- Cornelius Weig
- David Gageot
- Dmitrii Ermakov
- Jon Johnson
- jonjohnsonjr
- Miklós Kiss
- Naoki Oketani
- Nick Kubala
- Nick Novitski
- Prashant
- Prashant Arya
- Salahutdinov Dmitry
- saschahofmann
- Syed Awais Ali
- Tejal Desai
- Zac Bergquist


# v1.2.0 Release - 01/16/2019

*Note*: This release comes with a new config version `v2alpha2`. To upgrade your `skaffold.yaml`, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best it can.
Also: Happy New Year to all our users and contributors! 

Highlights:
- The deployer section in `skaffold.yaml` now accepts multiple deployers in a single pipeline.
- ValuesFiles can be rendered with environment and build variables
- CRD support: skaffold now doesn't throw an error on CRDs [#1737](https://github.com/GoogleContainerTools/skaffold/issues/1737) is fixed!
- `skaffold render` now supports the `kustomize` deployer
- parallel local builds are now supported - just set `build.local.concurrency` to 0 (no-limit) or >2   

New Features: 
* Enable multiple deployers in `skaffold.yaml` [#3392](https://github.com/GoogleContainerTools/skaffold/pull/3392)
    - The deployer section in `skaffold.yaml` now accepts multiple deployers in a single pipeline.
    - When applying profiles, deployers in the base profile no longer get wiped when merging in the deployer from the profile.
* Add forwarding deployer muxer to enable multiple deployers [#3391](https://github.com/GoogleContainerTools/skaffold/pull/3391)
* Related to #2849: Allows ValuesFiles to be templatable [#3111](https://github.com/GoogleContainerTools/skaffold/pull/3111)
    * ValuesFiles can be rendered with environment and build variables
* Implement render for kustomize [#3110](https://github.com/GoogleContainerTools/skaffold/pull/3110)
* Support parallel local builds (defaults to sequential) [#3471](https://github.com/GoogleContainerTools/skaffold/pull/3471)
* Add --target parameter with kaniko on Google Cloud Build [#3462](https://github.com/GoogleContainerTools/skaffold/pull/3462)

Fixes: 
* fix licenses path [#3517](https://github.com/GoogleContainerTools/skaffold/pull/3517)
* Dockerfile detector will only check files containing "Dockerfile" in the name [#3499](https://github.com/GoogleContainerTools/skaffold/pull/3499)
* Exclude CRD schema from transformation, fix #1737. [#3456](https://github.com/GoogleContainerTools/skaffold/pull/3456)  
* Kaniko: Cancel log streaming when pod fails to complete [#3481](https://github.com/GoogleContainerTools/skaffold/pull/3481)
* Use unique key for jib caches [#3483](https://github.com/GoogleContainerTools/skaffold/pull/3483)
* Remove false warnings when deploying multiple releases [#3470](https://github.com/GoogleContainerTools/skaffold/pull/3470)
* Fix sync infer when COPY destination contains an env variable [#3439](https://github.com/GoogleContainerTools/skaffold/pull/3439)
* Fix `skaffold credits` [#3436](https://github.com/GoogleContainerTools/skaffold/pull/3436)
* Track changes of transitive BUILD files [#3460](https://github.com/GoogleContainerTools/skaffold/pull/3460)

Updates & Refactors
* Spelling [#3458](https://github.com/GoogleContainerTools/skaffold/pull/3458) 
* Vendor pack CLI code to build with Buildpacks [#3445](https://github.com/GoogleContainerTools/skaffold/pull/3445)
* Remove gcr.io/k8s-skaffold repository from examples 
    * [#3368](https://github.com/GoogleContainerTools/skaffold/pull/3368)
    * Remove a few more references to gcr.io/k8s-skaffold [#3513](https://github.com/GoogleContainerTools/skaffold/pull/3513)
* Allow 2020 copyright year [#3511](https://github.com/GoogleContainerTools/skaffold/pull/3511)
* This test can run on Travis, with kind [#3510](https://github.com/GoogleContainerTools/skaffold/pull/3510)
* Move default images next to where they are used [#3509](https://github.com/GoogleContainerTools/skaffold/pull/3509)
* Kind 0.7.0 [#3507](https://github.com/GoogleContainerTools/skaffold/pull/3507)
* Use origin/master as baseline for schema version check [#3501](https://github.com/GoogleContainerTools/skaffold/pull/3501)
* Use pack CLI to build on GCB [#3503](https://github.com/GoogleContainerTools/skaffold/pull/3503)
* Simplify kaniko after we removed the GCS build context [#3455](https://github.com/GoogleContainerTools/skaffold/pull/3455)
* Switch to go-licenses for credits collection [#3493](https://github.com/GoogleContainerTools/skaffold/pull/3493)
* Add missing package-lock.json files [#3494](https://github.com/GoogleContainerTools/skaffold/pull/3494)
* Build Go projects with Buildpacks [#3504](https://github.com/GoogleContainerTools/skaffold/pull/3504)
* SyncMap is a matter of artifact type, not builder [#3450](https://github.com/GoogleContainerTools/skaffold/pull/3450)
* Remove Kaniko build context. [#3480](https://github.com/GoogleContainerTools/skaffold/pull/3480)
* [buildpacks] Refactor code to simplify #3395 [#3441](https://github.com/GoogleContainerTools/skaffold/pull/3441)
* Rename jib args functions [#3478](https://github.com/GoogleContainerTools/skaffold/pull/3478)
* Add gradle/maven sync parts + restructure tests [#3474](https://github.com/GoogleContainerTools/skaffold/pull/3474)
* helm deployer: Remove duplication [#3469](https://github.com/GoogleContainerTools/skaffold/pull/3469)
* Update Bazel sample [#3435](https://github.com/GoogleContainerTools/skaffold/pull/3435)
* Use the kind that’s inside skaffold-builder [#3430](https://github.com/GoogleContainerTools/skaffold/pull/3430)
* Move man generation to hack folder [#3464](https://github.com/GoogleContainerTools/skaffold/pull/3464)
* Schema v2alpha2 [#3453](https://github.com/GoogleContainerTools/skaffold/pull/3453)
* Cache Gradle downloads and Go build cache [#3425](https://github.com/GoogleContainerTools/skaffold/pull/3425)


Docs Updates: 
* [doc] Improve documentation for concurrency settings. [#3491](https://github.com/GoogleContainerTools/skaffold/pull/3491)
* [doc] Supported builders matrix [#3492](https://github.com/GoogleContainerTools/skaffold/pull/3492)
* [doc] There’s no `gcsBucket` config anymore [#3514](https://github.com/GoogleContainerTools/skaffold/pull/3514)
* Clarify GCP service account and secret creation [#3488](https://github.com/GoogleContainerTools/skaffold/pull/3488)
* Demonstrate inferred sync [#3495](https://github.com/GoogleContainerTools/skaffold/pull/3495)
* Use ko instead of buildpacks for the custom builder [#3432](https://github.com/GoogleContainerTools/skaffold/pull/3432)
* Buildpacks node sample [#3440](https://github.com/GoogleContainerTools/skaffold/pull/3440)

Huge thanks goes out to all of our contributors for this release:

- ansky
- Appu Goundan
- Arjan Topolovec
- Armin Buerkle
- Balint Pato
- Brian de Alwis
- Cedric Kring
- Chuck Dries
- Cornelius Weig
- Cyril Diagne
- David Gageot
- David Sabatie
- Farhad Vildanov
- Hwanjin Jeong
- Idan Bidani
- Josh Soref
- Marc
- Martin Hoefling
- Max Goltzsche
- Michael Beaumont
- Naoki Oketani
- Nick Kubala
- Nicklas Wallgren
- Nick Taylor
- Peter Jausovec
- Philippe Martin
- Pradip Caulagi
- Tad Cordle
- Tejal Desai
- Warren Strange

# v1.1.0 Release - 12/20/2019

*Note*: This release comes with a new config version `v2alpha1`. To upgrade your `skaffold.yaml`, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best it can.

Highlights:
- The `--port-forward` flag has been added to `skaffold run` and `skaffold deploy`
- `skaffold init` can now recognize nodeJS projects, and default to building them with Buildpacks
- Skaffold has been upgraded to build with Go 1.13
- Skaffold's `kind` version has been bumped to `v0.6.1`
- Skaffold will now default to using `gcloud` authentication when available


New Features: 
* Add `—port-forward` to `skaffold deploy` [#3418](https://github.com/GoogleContainerTools/skaffold/pull/3418)
* Add --port-forward to skaffold run [#3263](https://github.com/GoogleContainerTools/skaffold/pull/3263)
* Skaffold init recognizes nodeJS projects built with Buildpacks [#3394](https://github.com/GoogleContainerTools/skaffold/pull/3394)
* Add env vars to kaniko specs [#3389](https://github.com/GoogleContainerTools/skaffold/pull/3389)
* Default to gcloud auth [#3282](https://github.com/GoogleContainerTools/skaffold/pull/3282)
* Apply resource labels in the deployer [#3390](https://github.com/GoogleContainerTools/skaffold/pull/3390)
* Add commands to list/print json schemas [#3355](https://github.com/GoogleContainerTools/skaffold/pull/3355)


Fixes:
* fix wait logic in TestWaitForPodSucceeded [#3414](https://github.com/GoogleContainerTools/skaffold/pull/3414)
* Support FROM “scratch” [#3379](https://github.com/GoogleContainerTools/skaffold/pull/3379)
* Fix two issues with profiles [#3278](https://github.com/GoogleContainerTools/skaffold/pull/3278)
* `debug` should replace existing ports or environment values [#3195](https://github.com/GoogleContainerTools/skaffold/pull/3195)


Updates & Refactors:
* No buffering of test output [#3420](https://github.com/GoogleContainerTools/skaffold/pull/3420)
* Simplify skaffold init code [#3406](https://github.com/GoogleContainerTools/skaffold/pull/3406)
* Setup kind and build the docker image in // [#3413](https://github.com/GoogleContainerTools/skaffold/pull/3413)
* Upgrade to Go 1.13 [#3412](https://github.com/GoogleContainerTools/skaffold/pull/3412)
* Convert git tag into proper docker tag [#3407](https://github.com/GoogleContainerTools/skaffold/pull/3407)
* Better check for valid Kubernetes manifests [#3404](https://github.com/GoogleContainerTools/skaffold/pull/3404)
* add a resourceCounter to track pods [#3016](https://github.com/GoogleContainerTools/skaffold/pull/3016)
* Use --set-string for helm image values [#3313](https://github.com/GoogleContainerTools/skaffold/pull/3313)
* Bump kind to v0.6.1 [#3357](https://github.com/GoogleContainerTools/skaffold/pull/3357)
* Improve code that chooses random port for tests [#3304](https://github.com/GoogleContainerTools/skaffold/pull/3304)
* add container spec args to to go debug [#3276](https://github.com/GoogleContainerTools/skaffold/pull/3276)
* Cache expensive Go compilation and linting [#3341](https://github.com/GoogleContainerTools/skaffold/pull/3341)
* Change SyncMap supported types check style [#3328](https://github.com/GoogleContainerTools/skaffold/pull/3328)
* Improve error output when kompose fails [#3299](https://github.com/GoogleContainerTools/skaffold/pull/3299)
* Bump default Kaniko image [#3306](https://github.com/GoogleContainerTools/skaffold/pull/3306)
* Error instead of opening interactive prompt with --force init [#3252](https://github.com/GoogleContainerTools/skaffold/pull/3252)


Docs Updates: 
* document IDE setup [#3397](https://github.com/GoogleContainerTools/skaffold/pull/3397)
* Convert Asciidoc to simpler markdown [#3365](https://github.com/GoogleContainerTools/skaffold/pull/3365)
* [doc] Add missing configuration to the git tagger [#3283](https://github.com/GoogleContainerTools/skaffold/pull/3283)
* document skaffold debug & credits [#3285](https://github.com/GoogleContainerTools/skaffold/pull/3285)


Huge thanks goes out to all of our contributors for this release:

- Appu Goundan
- Balint Pato
- Brian de Alwis
- Chuck Dries
- Cornelius Weig
- Cyril Diagne
- David Gageot
- David Sabatie
- Idan Bidani
- Martin Hoefling
- Michael Beaumont
- Naoki Oketani
- Nick Kubala
- Nick Taylor
- Nicklas Wallgren
- Peter Jausovec
- Philippe Martin
- Pradip Caulagi
- Tad Cordle
- Tejal Desai
- ansky
- balopat

# v1.0.1 Release - 11/18/2019

This is a minor release to fix auto-project selection for GCB and Kaniko #3245.

# v1.0.0 Release - 11/07/2019

🎉🎉🎉🎉🎉🎉 
After two years, we are extremely excited to announce first generally available release v1.0.0 of Skaffold!
See [Skaffold Deprecation Policy](http://skaffold.dev/docs/references/deprecation/) for details on what GA means.
See [Feature Maturity](https://skaffold.dev/docs/references/deprecation/#skaffold-features) to find out more on feature maturity.
🎉🎉🎉🎉🎉🎉 

*Note*: This release also comes with a new config version `v1`. To upgrade your `skaffold.yaml`, use `skaffold fix`. If you don't upgrade, skaffold will auto-upgrade in memory as best it can, and print a warning message.
       
Highlights: 

- Revamped the http://skaffold.dev splash page, added client testimonials, and added a ton of missing documentation, clearer maturity state and what functionality applies for what skaffold command 
- Added experimental support for Cloud Native BuildPacks
- Third party open source licenses are now surfaced in `skaffold credits` command 


New Features: 
* Adding support for Cloud Native Buildpacks [#3000](https://github.com/GoogleContainerTools/skaffold/pull/3000)
* skaffold credits to surface thirdparty licenses [#3138](https://github.com/GoogleContainerTools/skaffold/pull/3138)

Fixes: 
* Fix redundant Jib image flags generated by init [#3191](https://github.com/GoogleContainerTools/skaffold/pull/3191)
* Simplify default repo handling and fix #3088 [#3089](https://github.com/GoogleContainerTools/skaffold/pull/3089)
* Fix EnvVarMap indices when caching is enabled [#3114](https://github.com/GoogleContainerTools/skaffold/pull/3114)
* Use native zsh completion script generator [#3137](https://github.com/GoogleContainerTools/skaffold/pull/3137)
* Allow configuring `jib` plugin type [#2964](https://github.com/GoogleContainerTools/skaffold/pull/2964)
* Fix writing rendered manifests to files [#3152](https://github.com/GoogleContainerTools/skaffold/pull/3152)
* Fixed issue with tagging of insecure registries. [#3127](https://github.com/GoogleContainerTools/skaffold/pull/3127)

Updates & refactorings:
* v1beta18 -> v1 [#3174](https://github.com/GoogleContainerTools/skaffold/pull/3174)
* Prepare kubectl and helm deployers for `--kubeconfig` flag [#3108](https://github.com/GoogleContainerTools/skaffold/pull/3108)
* init --analyze should return unique image names [#3141](https://github.com/GoogleContainerTools/skaffold/pull/3141)
* Don’t need race detection/code coverage [#3140](https://github.com/GoogleContainerTools/skaffold/pull/3140)
* Prepare cli-go to accept `--kubeconfig` setting [#3107](https://github.com/GoogleContainerTools/skaffold/pull/3107)
* Delegate release notes to external tool [#3055](https://github.com/GoogleContainerTools/skaffold/pull/3055)
* [buildpacks] Run cleanup on ctrl-c [#3184](https://github.com/GoogleContainerTools/skaffold/pull/3184)
* Remove trailing \n from download url [#3201](https://github.com/GoogleContainerTools/skaffold/pull/3201)
* Use native zsh completion script generator [#3137](https://github.com/GoogleContainerTools/skaffold/pull/3137)

Docs updates: 

* permissive docs/themes dir creation [#3154](https://github.com/GoogleContainerTools/skaffold/pull/3154)
* Skaffold API docs [#3068](https://github.com/GoogleContainerTools/skaffold/pull/3068)
* Fix splash [#3147](https://github.com/GoogleContainerTools/skaffold/pull/3147)
* Skaffold image credits [#3146](https://github.com/GoogleContainerTools/skaffold/pull/3146)
* [docs] a few docs changes [#3181](https://github.com/GoogleContainerTools/skaffold/pull/3181)
* Simplify custom builder example [#3183](https://github.com/GoogleContainerTools/skaffold/pull/3183)
* Improve the k8s yaml used in buildpacks sample [#3182](https://github.com/GoogleContainerTools/skaffold/pull/3182)
* [example] update apiVersion of Deployment [#3161](https://github.com/GoogleContainerTools/skaffold/pull/3161)
* [docs] Right steps for secret creation in `generate-pipeline` flow. [#3180](https://github.com/GoogleContainerTools/skaffold/pull/3180)
* [docs] [output] meaningful message for healthcheck context exceeded. [#3177](https://github.com/GoogleContainerTools/skaffold/pull/3177)
* [docs] minimal jib gcb docs [#3179](https://github.com/GoogleContainerTools/skaffold/pull/3179)
* [docs] skaffold run docs in Continuous Delivery pipeline [#3173](https://github.com/GoogleContainerTools/skaffold/pull/3173)
* [docs] update buildpacks tutorial to custom builder [#3166](https://github.com/GoogleContainerTools/skaffold/pull/3166)
* [docs] change config version to v1 [#3175](https://github.com/GoogleContainerTools/skaffold/pull/3175)
* [docs, API] control api + swagger ui for http api [#3158](https://github.com/GoogleContainerTools/skaffold/pull/3158)
* [docs] maturity model defined by JSON [#3162](https://github.com/GoogleContainerTools/skaffold/pull/3162)
* [docs] add init docs [#3149](https://github.com/GoogleContainerTools/skaffold/pull/3149)
* [docs] Add logging docs [#3170](https://github.com/GoogleContainerTools/skaffold/pull/3170)
* [docs] Working With Skaffold [#3169](https://github.com/GoogleContainerTools/skaffold/pull/3169)
* [docs] Add docs for dev and ci/cd workflows [#3153](https://github.com/GoogleContainerTools/skaffold/pull/3153)
* [docs] docs changes for feature matrix [#3164](https://github.com/GoogleContainerTools/skaffold/pull/3164)
* [docs] remove diagnose from feature matrix. [#3167](https://github.com/GoogleContainerTools/skaffold/pull/3167)
* [docs] fix alerts [#3159](https://github.com/GoogleContainerTools/skaffold/pull/3159)
* [docs] Rework skaffold.dev splash page [#3145](https://github.com/GoogleContainerTools/skaffold/pull/3145)
* [docs] document activation of multiple profiles [#3112](https://github.com/GoogleContainerTools/skaffold/pull/3112)
* [docs] Fixes a broken link to the Profiles page [#3144](https://github.com/GoogleContainerTools/skaffold/pull/3144)
* [docs] fix install links [#3135](https://github.com/GoogleContainerTools/skaffold/pull/3135)
* [docs] Fix broken link to installation guide [#3134](https://github.com/GoogleContainerTools/skaffold/pull/3134)
* Add example to `skaffold deploy` [#3202](https://github.com/GoogleContainerTools/skaffold/pull/3202)
* [Doc] Buildpacks [#3199](https://github.com/GoogleContainerTools/skaffold/pull/3199)
* [docs] add docs for buildpacks [#3198](https://github.com/GoogleContainerTools/skaffold/pull/3198)
* [example] update apiVersion of Deployment [#3161](https://github.com/GoogleContainerTools/skaffold/pull/3161)
* [docs] move builders in to individual pages [#3193](https://github.com/GoogleContainerTools/skaffold/pull/3193)
* [docs] Cleanup docs [#3176](https://github.com/GoogleContainerTools/skaffold/pull/3176)
* [docs] quick feedback page update [#3196](https://github.com/GoogleContainerTools/skaffold/pull/3196)
* [website] unify fonts [#3197](https://github.com/GoogleContainerTools/skaffold/pull/3197)
* [docs] Add healthcheck [#3178](https://github.com/GoogleContainerTools/skaffold/pull/3178)
* [doc] `debug` does not work with buildpack builder and maybe custom builder images too [#3204](https://github.com/GoogleContainerTools/skaffold/pull/3204)

Huge thanks goes out to all of our contributors for this release:

- Balint Pato
- Brian de Alwis
- Cornelius Weig
- David Gageot
- Martin Hoefling
- Naoki Oketani
- Nick Kubala
- Nicklas Wallgren
- Peter Jausovec
- Pradip Caulagi
- Tad Cordle
- Tejal Desai
- ansky

# v0.41.0 Release - 09/26/2019

*Note*: This release also comes with a new config version `v1beta17`.
        To upgrade your `skaffold.yaml`, use `skaffold fix`. If you don't upgrade, skaffold will auto-upgrade in memory as best it can, and print a warning message.
        See [Skaffold Deprecation Policy](http://skaffold.dev/docs/references/deprecation/) for details on what beta means.

*Note*: the custom artifact builder now uses $IMAGE instead of $IMAGES, please update your scripts! $IMAGES is now deprecated (undocumented, still works, but may go away eventually)

New Features: 

* Adding ephemeralstorage and ResourceStorage for kaniko pods [#3013](https://github.com/GoogleContainerTools/skaffold/pull/3013)
* Integrate file sync events into dev command [#3009](https://github.com/GoogleContainerTools/skaffold/pull/3009)
* add event api integration for deploy health check [#3072](https://github.com/GoogleContainerTools/skaffold/pull/3072)
* New version v1beta17 [#3041](https://github.com/GoogleContainerTools/skaffold/pull/3041)

Fixes:

* Improve skaffold init file traversal [#3062](https://github.com/GoogleContainerTools/skaffold/pull/3062)
* Fix `—force=false` [#3086](https://github.com/GoogleContainerTools/skaffold/pull/3086)
* Interrupt skaffold init with ctrl-c [#3070](https://github.com/GoogleContainerTools/skaffold/pull/3070)
* display survey prompt which points to survey url [#3011](https://github.com/GoogleContainerTools/skaffold/pull/3011)
* Fix remove patch in Profiles [#3045](https://github.com/GoogleContainerTools/skaffold/pull/3045)
* Fix `skaffold deploy --tail` [#3049](https://github.com/GoogleContainerTools/skaffold/pull/3049)


Updates & Refactorings:

* Log durations instead of always printing them [#3102](https://github.com/GoogleContainerTools/skaffold/pull/3102)
* Add heuristics to speed up Jib check in skaffold init [#3120](https://github.com/GoogleContainerTools/skaffold/pull/3120)
* [Custom] [Deprecation] Use $IMAGE instead of $IMAGES  [#3084](https://github.com/GoogleContainerTools/skaffold/pull/3084)
* Remove logs before building and testing [#3105](https://github.com/GoogleContainerTools/skaffold/pull/3105)
* Align kubectl/kustomize cleanup output with deploy output [#3103](https://github.com/GoogleContainerTools/skaffold/pull/3103)
* `skaffold build` shouldn’t print the tags used in deployments [#3091](https://github.com/GoogleContainerTools/skaffold/pull/3091)
* Update a few dependencies [#3087](https://github.com/GoogleContainerTools/skaffold/pull/3087)
* Upgrade Jib to 1.7.0 [#3093](https://github.com/GoogleContainerTools/skaffold/pull/3093)
* [Custom] Clearer message when image was not built [#3085](https://github.com/GoogleContainerTools/skaffold/pull/3085)
* Warn when default or provided port not available for API Server [#3065](https://github.com/GoogleContainerTools/skaffold/pull/3065)
* [Cache] Ignore file not found [#3066](https://github.com/GoogleContainerTools/skaffold/pull/3066)
* [kaniko] Stop printing the logs on ctrl-c [#3069](https://github.com/GoogleContainerTools/skaffold/pull/3069)
* a windows build file [#3063](https://github.com/GoogleContainerTools/skaffold/pull/3063)
* Activate more linters [#3057](https://github.com/GoogleContainerTools/skaffold/pull/3057)
* Don’t print bazel slow warnings more than once. [#3059](https://github.com/GoogleContainerTools/skaffold/pull/3059)
* [Bazel] Target must end with .tar [#3058](https://github.com/GoogleContainerTools/skaffold/pull/3058)
* remove Container Was Terminated message [#3054](https://github.com/GoogleContainerTools/skaffold/pull/3054)
* Update docker and go-containerregistry [#3053](https://github.com/GoogleContainerTools/skaffold/pull/3053)
* Update dependencies and rollback to older k8s [#3052](https://github.com/GoogleContainerTools/skaffold/pull/3052)
* Use a switch instead of if [#3042](https://github.com/GoogleContainerTools/skaffold/pull/3042)
* Warn about unused configs [#3046](https://github.com/GoogleContainerTools/skaffold/pull/3046)

Docs:

* Close the bracket in documentation [#3101](https://github.com/GoogleContainerTools/skaffold/pull/3101)
* Clarify debug docs for deprecated Workload APIs [#3092](https://github.com/GoogleContainerTools/skaffold/pull/3092)
* move pr template instructions to comments [#3080](https://github.com/GoogleContainerTools/skaffold/pull/3080)
* Rename custom/buildpacks sample config [#3076](https://github.com/GoogleContainerTools/skaffold/pull/3076)
* Docs updates [#3079](https://github.com/GoogleContainerTools/skaffold/pull/3079)
* Major docs restructure [#3071](https://github.com/GoogleContainerTools/skaffold/pull/3071)
* generate docs for proto [#3067](https://github.com/GoogleContainerTools/skaffold/pull/3067)
* Make all docs have TOC on the right hand side. [#3064](https://github.com/GoogleContainerTools/skaffold/pull/3064)
* Add HaTS and Opt-In Feedback links [#2919](https://github.com/GoogleContainerTools/skaffold/pull/2919)
* getting started -> quickstart [#3030](https://github.com/GoogleContainerTools/skaffold/pull/3030)

Design proposals: 

* kube-context design proposal: add note about the implementation status [#2991](https://github.com/GoogleContainerTools/skaffold/pull/2991)


Huge thanks goes out to all of our contributors for this release:

- Amet Umerov
- Andreas Sommer
- Balint Pato
- Brian de Alwis
- Cornelius Weig
- David Gageot
- Hugo Duncan
- Jens Ulrich Hjuler Fosgerau
- Michael Beaumont
- Nick Kubala
- Philippe Martin
- Prashant
- Priya Wadhwa
- Tad Cordle
- Tejal Desai

# v0.40.0 Release - 09/26/2019

This release adds a new command, `skaffold render`, which will output templated kubernetes manifests rather than sending them through `kubectl` to deploy to your cluster. This can be used to commit final manifests to a git repo for use in GitOps workflows.

*This command has been implemented for the `kubectl` deployer only; implementations for `kustomize` and `helm` will follow in the next release.*

*Note*: This release also comes with a new config version `v1beta16`.
        To upgrade your `skaffold.yaml`, use `skaffold fix`. If you don't upgrade, skaffold will auto-upgrade in memory as best it can, and print a warning message.
        See [Skaffold Deprecation Policy](http://skaffold.dev/docs/references/deprecation/) for details on what beta means.


New Features:

* Add option to override kubecontext from `skaffold.yaml` [#2510](https://github.com/GoogleContainerTools/skaffold/pull/2510)
* Support YAML anchors in skaffold.yaml (key must start with a dot) [#2836](https://github.com/GoogleContainerTools/skaffold/pull/2836)
* Add file sync to Event and State APIs [#2978](https://github.com/GoogleContainerTools/skaffold/pull/2978)
* Implement 'skaffold render' for kubectl deployer [#2943](https://github.com/GoogleContainerTools/skaffold/pull/2943)
* Add skip tls verify option to kaniko builder [#2976](https://github.com/GoogleContainerTools/skaffold/pull/2976)
* Add PullSecretMountPath to ClusterDetails [#2975](https://github.com/GoogleContainerTools/skaffold/pull/2975)

Bugfixes:

* Fix bugs in insecure registries for kaniko [#2974](https://github.com/GoogleContainerTools/skaffold/pull/2974)
* Fix check flake by not using Github API [#3033](https://github.com/GoogleContainerTools/skaffold/pull/3033)
* Pass the context [#3014](https://github.com/GoogleContainerTools/skaffold/pull/3014)
* Push once [#2855](https://github.com/GoogleContainerTools/skaffold/pull/2855)
* Tiny typo fix for build output in skaffold deploy [#2988](https://github.com/GoogleContainerTools/skaffold/pull/2988)
* Don't assume string keys in helm charts [#2982](https://github.com/GoogleContainerTools/skaffold/pull/2982)
* Properly tag images with digest when using helm [#2956](https://github.com/GoogleContainerTools/skaffold/pull/2956)
* Reset State on Build [#2944](https://github.com/GoogleContainerTools/skaffold/pull/2944)
* reset deploy state [#2945](https://github.com/GoogleContainerTools/skaffold/pull/2945)
* Fix Flake TestPollResourceStatus/resource_stabilizes by removing sleep from test. [#2934](https://github.com/GoogleContainerTools/skaffold/pull/2934)

Updates & Refactors:

* handle StatusCheck Events implementation logic [#2929](https://github.com/GoogleContainerTools/skaffold/pull/2929)
* Custom artifact depends by default on the whole workspace [#3028](https://github.com/GoogleContainerTools/skaffold/pull/3028)
* Strip the debugging information [#3027](https://github.com/GoogleContainerTools/skaffold/pull/3027)
* Improve error messages for `deploy.kubeContext` error cases [#2993](https://github.com/GoogleContainerTools/skaffold/pull/2993)
* Bump golangci-lint to v1.20.0 [#3018](https://github.com/GoogleContainerTools/skaffold/pull/3018)
* Refactor `setDefaults` code [#2995](https://github.com/GoogleContainerTools/skaffold/pull/2995)
* Every type of artifact should be handled. [#2996](https://github.com/GoogleContainerTools/skaffold/pull/2996)
* Simpler code for GCB dependencies [#2997](https://github.com/GoogleContainerTools/skaffold/pull/2997)
* Extract code that handles graceful termination [#3005](https://github.com/GoogleContainerTools/skaffold/pull/3005)
* Download pack like the other packages [#2998](https://github.com/GoogleContainerTools/skaffold/pull/2998)
* go mod tidy [#3003](https://github.com/GoogleContainerTools/skaffold/pull/3003)
* [custom] Test error case [#3004](https://github.com/GoogleContainerTools/skaffold/pull/3004)
* v1beta16 [#2955](https://github.com/GoogleContainerTools/skaffold/pull/2955)
* report StatusCheck Events [#2929](https://github.com/GoogleContainerTools/skaffold/pull/2929)
* Add Pod Status when pod is pending by going through pending container [#2932](https://github.com/GoogleContainerTools/skaffold/pull/2932)
* rename imageList to podSelector [#2989](https://github.com/GoogleContainerTools/skaffold/pull/2989)
* Specifying artifact location i.e locally or remote [#2958](https://github.com/GoogleContainerTools/skaffold/pull/2958)
* remove duplicate status check [#2966](https://github.com/GoogleContainerTools/skaffold/pull/2966)

Docs:

* Add page about kube-context handling in docs concepts section [#2992](https://github.com/GoogleContainerTools/skaffold/pull/2992)
* Fix sample’s version [#3015](https://github.com/GoogleContainerTools/skaffold/pull/3015)
* Fix versions used in examples [#2999](https://github.com/GoogleContainerTools/skaffold/pull/2999)
* Docs Splash Page Update [#3031](https://github.com/GoogleContainerTools/skaffold/pull/3031)
* [docs] re-add exceptions in deprecation policy [#3029](https://github.com/GoogleContainerTools/skaffold/pull/3029)
* add links to docs which are present [#3026](https://github.com/GoogleContainerTools/skaffold/pull/3026)
* moving deprecation policy to skaffold.dev [#3017](https://github.com/GoogleContainerTools/skaffold/pull/3017)
* add survey link and reword community office hours [#3019](https://github.com/GoogleContainerTools/skaffold/pull/3019)
* Bump Hugo to 0.58.3 [#3001](https://github.com/GoogleContainerTools/skaffold/pull/3001)
* List all builders in doc [#3002](https://github.com/GoogleContainerTools/skaffold/pull/3002)
* Add small pr guidelines [#2977](https://github.com/GoogleContainerTools/skaffold/pull/2977)
* link validation in docs [#2984](https://github.com/GoogleContainerTools/skaffold/pull/2984)

Huge thanks goes out to all of our contributors for this release:

- Andreas Sommer
- Balint Pato
- Cornelius Weig
- David Gageot
- Hugo Duncan
- Jens Ulrich Hjuler Fosgerau
- Michael Beaumont
- Nick Kubala
- Prashant
- Priya Wadhwa
- Tejal Desai


# v0.39.0 Release - 09/26/2019

*Note*: This release comes with a new config version `v1beta15`.
        To upgrade your `skaffold.yaml`, use `skaffold fix`. If you don't upgrade, skaffold will auto-upgrade in memory as best it can, and print a warning message.
        See [Skaffold Deprecation Policy](http://skaffold.dev/docs/references/deprecation/) for details on what beta means.
        The env vars `DIGEST`, `DIGEST_HEX` and `DIGEST_ALGO` now fail if found in `envTemplate` fields. 

Highlights: 

* We now include build args in the artifact cache hash generation [#2926](https://github.com/GoogleContainerTools/skaffold/pull/2926) 
* Skaffold now passes the `--set-files` argument to the helm CLI: you can define `helm.release.setFiles` in the skaffold.yaml
* Skaffold now passes the `--build-args` arguments to kustomize: you can define `deploy.kustomize.buildArgs` in the skaffold.yaml

New Features:

* Optional pull secret for Kaniko [#2910](https://github.com/GoogleContainerTools/skaffold/pull/2910)
* Add Jib-Gradle support for Kotlin buildscripts [#2914](https://github.com/GoogleContainerTools/skaffold/pull/2914)
* Add graceful termination for custom builders [#2886](https://github.com/GoogleContainerTools/skaffold/pull/2886)
* Add docs and tutorial for buildpacks [#2879](https://github.com/GoogleContainerTools/skaffold/pull/2879)
* kustomize build args [#2871](https://github.com/GoogleContainerTools/skaffold/pull/2871)
* Add `setFiles` to `HelmDeploy.HelmRelease` skaffold config which will be add `--set-files` argument to helm CLI [#2895](https://github.com/GoogleContainerTools/skaffold/pull/2895)

Bug Fixes:

* fix flake TestGetSetFileValues [#2936](https://github.com/GoogleContainerTools/skaffold/pull/2936)
* Fix helm deployer with imageStrategy helm and fix test runner [#2887](https://github.com/GoogleContainerTools/skaffold/pull/2887)
* Include build args in cache hash generation [#2926](https://github.com/GoogleContainerTools/skaffold/pull/2926)
* Fix test flake TestPollResourceStatus [#2907](https://github.com/GoogleContainerTools/skaffold/pull/2907)
* Fix build script for doc generation. [#2884](https://github.com/GoogleContainerTools/skaffold/pull/2884)

Updates & Refactors:

* Create new v1beta15 config [#2881](https://github.com/GoogleContainerTools/skaffold/pull/2881)
* adding release comment management to all config.go [#2917](https://github.com/GoogleContainerTools/skaffold/pull/2917)
* Change final status check error message to be more concise. [#2930](https://github.com/GoogleContainerTools/skaffold/pull/2930)
* Add unimplemented 'skaffold render' command [#2942](https://github.com/GoogleContainerTools/skaffold/pull/2942)
* Bump golangci-lint to v0.19.0 [#2927](https://github.com/GoogleContainerTools/skaffold/pull/2927)
* Add pod resource with no status check implemented. [#2928](https://github.com/GoogleContainerTools/skaffold/pull/2928)
* added support for interface type in schema check [#2924](https://github.com/GoogleContainerTools/skaffold/pull/2924)
* add protos for status check [#2916](https://github.com/GoogleContainerTools/skaffold/pull/2916)
* Refactor Deployment common functions in to a  Base struct in prep to pod [#2905](https://github.com/GoogleContainerTools/skaffold/pull/2905)
* Add missing T.Helper() in testutil.Check* as required [#2913](https://github.com/GoogleContainerTools/skaffold/pull/2913)
* Removing testing version dependent skaffold config test in examples [#2890](https://github.com/GoogleContainerTools/skaffold/pull/2890)
* rename hack/versions/cmd/new/new.go to hack/versions/cmd/new/version.go [#2882](https://github.com/GoogleContainerTools/skaffold/pull/2882)
* [Refactor] Move pollDeploymentStatus to resource.Deployment.CheckStatus [#2896](https://github.com/GoogleContainerTools/skaffold/pull/2896)
* init: Add default config name [#2668](https://github.com/GoogleContainerTools/skaffold/pull/2668)
* Upgrade jib to 1.6.1 [#2891](https://github.com/GoogleContainerTools/skaffold/pull/2891)
* Print deployment status after every 0.5 seconds. [#2866](https://github.com/GoogleContainerTools/skaffold/pull/2866)
* Fail PR if it has a structural schema change in a released version [#2864](https://github.com/GoogleContainerTools/skaffold/pull/2864)

Docs:

* add better docs for recreate pods [#2937](https://github.com/GoogleContainerTools/skaffold/pull/2937)
* added release comments manually [#2931](https://github.com/GoogleContainerTools/skaffold/pull/2931)
* add github pull request template [#2894](https://github.com/GoogleContainerTools/skaffold/pull/2894)
        
        
Huge thanks goes out to all of our contributors for this release:

- Aisuko
- Andreas Sommer
- Balint Pato
- Brian de Alwis
- Cedric Kring
- Chanseok Oh
- Cornelius Weig
- David Gageot
- Dominic Werner
- Jack Davis
- Marlon Gamez
- Medya Gh
- Michael Beaumont
- Nick Kubala
- Prashant Arya
- Priya Wadhwa
- Tad Cordle
- Tejal Desai
- Willy Aguirre


# v0.38.0 Release - 09/12/2019

*Note*: This release comes with a new config version `v1beta14`.
        To upgrade your `skaffold.yaml`, use `skaffold fix`. If you don't upgrade, skaffold will auto-upgrade in memory as best it can, and print a warning message.
        See [Skaffold Deprecation Policy](http://skaffold.dev/docs/references/deprecation/) for details on what beta means.        
        The env vars `DIGEST`, `DIGEST_HEX` and `DIGEST_ALGO` won't work anymore in envTemplates.

New Features:

* Add Go container debugging support [#2306](https://github.com/GoogleContainerTools/skaffold/pull/2306)
* Note: `jibMaven` and `jibGradle` are now just simply `jib` - your old config should be upgraded automatically. [#2808](https://github.com/GoogleContainerTools/skaffold/pull/2808)
* Add Kaniko builder to GCB [#2708](https://github.com/GoogleContainerTools/skaffold/pull/2708)

Bug Fixes:

* Upgrade golangci-lint to v1.18.0 [#2853](https://github.com/GoogleContainerTools/skaffold/pull/2853)
* Always add image flag to jib builders in skaffold init [#2854](https://github.com/GoogleContainerTools/skaffold/pull/2854)
* add deploy stabilize timer [#2845](https://github.com/GoogleContainerTools/skaffold/pull/2845)
* Only activate `env: "KEY="` for empty environment variable value, clearly document pattern behavior [#2839](https://github.com/GoogleContainerTools/skaffold/pull/2839)
* Small random fixes to tests and code [#2801](https://github.com/GoogleContainerTools/skaffold/pull/2801)
* skaffold init can be interrupted when kompose is running [#2803](https://github.com/GoogleContainerTools/skaffold/pull/2803)
* Fix portforward flake [#2824](https://github.com/GoogleContainerTools/skaffold/pull/2824)
* Improve `skaffold init` behavior when tags are used in manifests [#2773](https://github.com/GoogleContainerTools/skaffold/pull/2773)
* Skip secret creation/check [#2783](https://github.com/GoogleContainerTools/skaffold/pull/2783)

Updates & Refactors:

* Print deployment status check summary when a status check is completed. [#2811](https://github.com/GoogleContainerTools/skaffold/pull/2811)
* add tests for `Status.String` method. [#2861](https://github.com/GoogleContainerTools/skaffold/pull/2861)
* Update dependencies [#2857](https://github.com/GoogleContainerTools/skaffold/pull/2857)
* Prepare to Add functionality to Replacer interface to restrict setting labels on certain kinds. [#2060](https://github.com/GoogleContainerTools/skaffold/pull/2060)
* Add Resource.Status object and remove sync.Map [#2851](https://github.com/GoogleContainerTools/skaffold/pull/2851)
* Add `Deployment` resource struct. [#2847](https://github.com/GoogleContainerTools/skaffold/pull/2847)
* refactor pollDeploymentRolloutStatus [#2846](https://github.com/GoogleContainerTools/skaffold/pull/2846)
* Improve runner [#2828](https://github.com/GoogleContainerTools/skaffold/pull/2828)
* Ignore codecov.io upload errors [#2841](https://github.com/GoogleContainerTools/skaffold/pull/2841)
* fix flake in in-cluster build [#2799](https://github.com/GoogleContainerTools/skaffold/pull/2799)
* skaffold trace -> kaniko debug [#2823](https://github.com/GoogleContainerTools/skaffold/pull/2823)
* Single way of mocking Kubernetes client/dynamic client [#2796](https://github.com/GoogleContainerTools/skaffold/pull/2796)
* Remove caching flags true from integration test [#2831](https://github.com/GoogleContainerTools/skaffold/pull/2831)
* add example for skaffold generate-pipeline [#2822](https://github.com/GoogleContainerTools/skaffold/pull/2822)
* Improve versioning [#2798](https://github.com/GoogleContainerTools/skaffold/pull/2798)
* Simplify TestBuildInCluster [#2829](https://github.com/GoogleContainerTools/skaffold/pull/2829)
* Simplify doDev() [#2815](https://github.com/GoogleContainerTools/skaffold/pull/2815)
* Remove misleading log [#2802](https://github.com/GoogleContainerTools/skaffold/pull/2802)
* Merge back release/v0.37.1 [#2800](https://github.com/GoogleContainerTools/skaffold/pull/2800)
* increasing unit test timeout to 90s [#2805](https://github.com/GoogleContainerTools/skaffold/pull/2805)
* remove unused values helm example [#2819](https://github.com/GoogleContainerTools/skaffold/pull/2819)
* Add --config-files flag for generate-pipeline command [#2766](https://github.com/GoogleContainerTools/skaffold/pull/2766)
* Update dependencies [#2818](https://github.com/GoogleContainerTools/skaffold/pull/2818)

Docs:

* [doc-style]/Sorting out the tools list follow the workflow picture. [#2838](https://github.com/GoogleContainerTools/skaffold/pull/2838)
* Design proposal for integrationtest command [#2671](https://github.com/GoogleContainerTools/skaffold/pull/2671)
* Split the concepts section into several sub-pages [#2810](https://github.com/GoogleContainerTools/skaffold/pull/2810)

Huge thanks goes out to all of our contributors for this release:

- Aisuko
- Andreas Sommer
- Balint Pato
- balopat
- Brian de Alwis
- Cedric Kring
- Chanseok Oh
- Cornelius Weig
- daddz
- David Gageot
- Jack Davis
- Marlon Gamez
- Medya Gh
- Nick Kubala
- Prashant Arya
- Tad Cordle
- Tejal Desai

# v0.37.1 Release - 09/04/2019

This is a minor release for a privacy policy update:

* add privacy notice and command to set update check false [#2774](https://github.com/GoogleContainerTools/skaffold/pull/2774)

# v0.37.0 Release - 08/29/2019

No new features in this release!

Bug Fixes:

* Use active gcloud credentials for executing cloudbuild when available [#2731](https://github.com/GoogleContainerTools/skaffold/pull/2731)
* Restore original images only if there are no remote manifests [#2746](https://github.com/GoogleContainerTools/skaffold/pull/2746)
* List manifests in the order given by the user [#2729](https://github.com/GoogleContainerTools/skaffold/pull/2729)
* Fix 'skaffold diagnose' for custom builder without dependencies [#2724](https://github.com/GoogleContainerTools/skaffold/pull/2724)
* Don't panic when dockerConfig isn't provided [#2735](https://github.com/GoogleContainerTools/skaffold/pull/2735)
* Don't set KanikoArtifact if CustomArtifact is set [#2716](https://github.com/GoogleContainerTools/skaffold/pull/2716)
* [Caching] Artifact’s config is an input to digest calculation [#2728](https://github.com/GoogleContainerTools/skaffold/pull/2728)
* Don’t fetch images that are aliases for scratch [#2720](https://github.com/GoogleContainerTools/skaffold/pull/2720)
* Implement exponential backoff for retrieving cloud build status [#2667](https://github.com/GoogleContainerTools/skaffold/pull/2667)
* Fix call to newPortForwardEntry constructor in kubectl_forwarder_test [#2703](https://github.com/GoogleContainerTools/skaffold/pull/2703)
* Add information about top level owner to port forward key [#2675](https://github.com/GoogleContainerTools/skaffold/pull/2675)
* Turn RPC State forwardedPorts into map keyed by the local port [#2659](https://github.com/GoogleContainerTools/skaffold/pull/2659)
* Show the duration of the deploy phase [#2739](https://github.com/GoogleContainerTools/skaffold/pull/2739)
* Configure jib.allowInsecureRegistries as required [#2674](https://github.com/GoogleContainerTools/skaffold/pull/2674)

Updates & Refactors:

* Pass extra env to the Docker CLI [#2737](https://github.com/GoogleContainerTools/skaffold/pull/2737)
* Improve manifest splitting. [#2727](https://github.com/GoogleContainerTools/skaffold/pull/2727)
* Bazel query should specify --output [#2712](https://github.com/GoogleContainerTools/skaffold/pull/2712)
* Print the output of failed integration tests [#2725](https://github.com/GoogleContainerTools/skaffold/pull/2725)
* We must handle every profile field type [#2726](https://github.com/GoogleContainerTools/skaffold/pull/2726)
* Fix CI scripts [#2736](https://github.com/GoogleContainerTools/skaffold/pull/2736)
* Directs "Download" button to Quickstart [#2695](https://github.com/GoogleContainerTools/skaffold/pull/2695)
* Small improvements to code coverage [#2719](https://github.com/GoogleContainerTools/skaffold/pull/2719)
* Don’t store log lines as mutable slices of bytes [#2721](https://github.com/GoogleContainerTools/skaffold/pull/2721)
* more debugging for kubectl portforward [#2707](https://github.com/GoogleContainerTools/skaffold/pull/2707)
* Remove time sensitive tests [#2655](https://github.com/GoogleContainerTools/skaffold/pull/2655)
* Log a warning and rebuild if needed when caching fails [#2685](https://github.com/GoogleContainerTools/skaffold/pull/2685)
* Improve logging warning when encountering profile field of unhandled type [#2691](https://github.com/GoogleContainerTools/skaffold/pull/2691)
* refactor: Add upgrade utility to handle all pipelines in a SkaffoldConfig [#2582](https://github.com/GoogleContainerTools/skaffold/pull/2582)
* Add struct for generate_pipeline to keep track of related data [#2686](https://github.com/GoogleContainerTools/skaffold/pull/2686)
* Add unit tests to kubectl forwarder [#2661](https://github.com/GoogleContainerTools/skaffold/pull/2661)
* separate checks + unit tests [#2676](https://github.com/GoogleContainerTools/skaffold/pull/2676)
* Add UPSTREAM_CLIENT_TYPE user agent environment variable to kaniko pod [#2723](https://github.com/GoogleContainerTools/skaffold/pull/2723)

Docs: 

* Document Docker buildArgs as templated field [#2696](https://github.com/GoogleContainerTools/skaffold/pull/2696)
* Update cache-artifacts option usage language to reflect new default [#2711](https://github.com/GoogleContainerTools/skaffold/pull/2711)
* docs: clarify that tagged images in manifests are not replaced [#2598](https://github.com/GoogleContainerTools/skaffold/pull/2598)
* fix development guide link [#2710](https://github.com/GoogleContainerTools/skaffold/pull/2710)
* Update community section of README [#2682](https://github.com/GoogleContainerTools/skaffold/pull/2682)

Huge thanks goes out to all of our contributors for this release:

- Aaron Paz
- Andreas Sommer
- Appu
- Balint Pato
- bpopovschi
- Brian de Alwis
- Cedric Kring
- Chanseok Oh
- Charles-Henri GUERIN
- Cornelius Weig
- David Gageot
- Dmitri Moore
- Filip Krakowski
- Jason McClellan
- JieJhih Jhang
- Marlon Gamez
- Matt Brown
- Medya Ghazizadeh
- Michael Beaumont
- Nick Kubala
- Prashant Arya
- Priya Wadhwa
- Russell Wolf
- Sébastien Le Gall
- Sergei Morozov
- Tad Cordle
- Tanner Bruce
- Taylor Barrella
- Tejal Desai
- Tom Dickman


# v0.36.0 Release - 08/15/2019

New Features:

* Add CLI option `--kube-context` to override the kubecontext in Skaffold [#2447](https://github.com/GoogleContainerTools/skaffold/pull/2447)
* Set artifact caching on by default [#2621](https://github.com/GoogleContainerTools/skaffold/pull/2621)
* Add flag `status-check-deadline` instead of default 10 minutes [#2591](https://github.com/GoogleContainerTools/skaffold/pull/2591)
* skaffold generate-pipeline command (experimental) [#2567](https://github.com/GoogleContainerTools/skaffold/pull/2567)

Bug Fixes:

* Pass minikube docker configuration to container-structure-test [#2597](https://github.com/GoogleContainerTools/skaffold/pull/2597)
* Use pointers for connection listeners so they can be closed properly [#2652](https://github.com/GoogleContainerTools/skaffold/pull/2652)
* Don't look up services in all namespaces. [#2651](https://github.com/GoogleContainerTools/skaffold/pull/2651)
* Add CLI flag `--config` for configuring the global config location [#2555](https://github.com/GoogleContainerTools/skaffold/pull/2555)
* Fix kaniko permissions with generate-pipeline command [#2622](https://github.com/GoogleContainerTools/skaffold/pull/2622)
* Fix remoteManifests [#2258](https://github.com/GoogleContainerTools/skaffold/pull/2258)
* docker auth: use GetAllCredentials() to use credHelpers [#2573](https://github.com/GoogleContainerTools/skaffold/pull/2573)
* Add missing digest when setting helm image tag [#2624](https://github.com/GoogleContainerTools/skaffold/pull/2624)
* Make sure we mute/unmute logs at the correct times [#2602](https://github.com/GoogleContainerTools/skaffold/pull/2602)


Updates & Refactors:

* Merge global and context-specific array settings in Skaffold config [#2590](https://github.com/GoogleContainerTools/skaffold/pull/2590)
* Add unit test for LoadOrStore  [#2649](https://github.com/GoogleContainerTools/skaffold/pull/2649)
* Add constructor for creating portForwardEntry [#2648](https://github.com/GoogleContainerTools/skaffold/pull/2648)
* Link task resources in generate-pipeline output [#2638](https://github.com/GoogleContainerTools/skaffold/pull/2638)
* Select resources by UUID label [#2609](https://github.com/GoogleContainerTools/skaffold/pull/2609)
* Collect namespaces of deployed resources. [#2640](https://github.com/GoogleContainerTools/skaffold/pull/2640)
* Add port forwarding integration test [#2623](https://github.com/GoogleContainerTools/skaffold/pull/2623)
* Fix issue with remote Kustomizations in dev mode. (#2581) [#2611](https://github.com/GoogleContainerTools/skaffold/pull/2611)
* Watch all artifact workspaces, including those outside of the working directory [#2614](https://github.com/GoogleContainerTools/skaffold/pull/2614)
* Make skaffold-generate pipeline command hidden [#2616](https://github.com/GoogleContainerTools/skaffold/pull/2616)
* refactor code used by pkg/skaffold/runner/generate_pipeline.go [#2617](https://github.com/GoogleContainerTools/skaffold/pull/2617)
* Update skaffold init --artifact to use JSON structs instead of paths [#2364](https://github.com/GoogleContainerTools/skaffold/pull/2364)
* fix travis build + docs whitespaces to trigger build [#2610](https://github.com/GoogleContainerTools/skaffold/pull/2610)
* Update .travis.yml [#2600](https://github.com/GoogleContainerTools/skaffold/pull/2600)
* build master only on travis CI [#2607](https://github.com/GoogleContainerTools/skaffold/pull/2607)

Docs: 

* Design proposal for configurable kubecontext [#2384](https://github.com/GoogleContainerTools/skaffold/pull/2384)
* Removed broken link, since the page doesn't exists anymore [#2644](https://github.com/GoogleContainerTools/skaffold/pull/2644)


Huge thanks goes out to all of our contributors for this release:

- Balint Pato
- bpopovschi
- Chanseok Oh
- Cornelius Weig
- Filip Krakowski
- Jason McClellan
- Marlon Gamez
- Matt Brown
- Nick Kubala
- Priya Wadhwa
- Tad Cordle
- Tanner Bruce
- Tejal Desai


# v0.35.0 Release - 08/02/2019

*Note for Jib users*: The Jib binding has changed and projects are now required to use
        Jib v1.4.0 or later.  Maven multi-module projects no longer require
        binding `jib:build` or `jib:dockerBuild` to the _package_ phase and should be removed.

New Features:

* Add Jib detection to skaffold init [#2276](https://github.com/GoogleContainerTools/skaffold/pull/2276)
* Add ability to pass an explicit registry value to Helm charts [#2188](https://github.com/GoogleContainerTools/skaffold/pull/2188)

Bug Fixes:

* Make sure we mute/unmute logs at the correct times [#2592](https://github.com/GoogleContainerTools/skaffold/pull/2592)
* Fix handling of whitelisted directories in dockerignore [#2589](https://github.com/GoogleContainerTools/skaffold/pull/2589)
* Cleaner kubectl `port-forward` retry logic [#2593](https://github.com/GoogleContainerTools/skaffold/pull/2593)
* Negotiate docker API version when creating minikube docker client [#2577](https://github.com/GoogleContainerTools/skaffold/pull/2577)
* Retry port forwarding when we see forwarding-related errors from kubectl [#2566](https://github.com/GoogleContainerTools/skaffold/pull/2566)

Updates & Refactors:

* Refactor: Use new `kubectl.CLI` util to shell out to `kubectl` [#2509](https://github.com/GoogleContainerTools/skaffold/pull/2509)
* Remove duplication around Go modules settings [#2580](https://github.com/GoogleContainerTools/skaffold/pull/2580)
* Faster tests [#2570](https://github.com/GoogleContainerTools/skaffold/pull/2570)
* [linters] Use vendored dependencies. Don’t download them. [#2579](https://github.com/GoogleContainerTools/skaffold/pull/2579)
* Improve Jib support on gcb [#2548](https://github.com/GoogleContainerTools/skaffold/pull/2548)
* Bring back applying labels to services deployed with helm [#2568](https://github.com/GoogleContainerTools/skaffold/pull/2568)
* Fix linter deadline [#2572](https://github.com/GoogleContainerTools/skaffold/pull/2572)
* Go Modules [#2541](https://github.com/GoogleContainerTools/skaffold/pull/2541)
* Make all embedded fields on runner private [#2565](https://github.com/GoogleContainerTools/skaffold/pull/2565)
* Simplify FakeAPIClient [#2563](https://github.com/GoogleContainerTools/skaffold/pull/2563)
* Minor changes to kubectl and kustomize deployers [#2537](https://github.com/GoogleContainerTools/skaffold/pull/2537)
* Simplify Sync code [#2564](https://github.com/GoogleContainerTools/skaffold/pull/2564)
* Starting a refactoring around RunContext and Docker local/remote Api [#2497](https://github.com/GoogleContainerTools/skaffold/pull/2497)

Huge thanks goes out to all of our contributors for this release:

- Balint Pato
- Cornelius Weig
- David Gageot
- Michael Beaumont
- Nick Kubala
- Priya Wadhwa
- Tad Cordle
- Tejal Desai



# v0.34.1 Release - 07/25/2019
This minor release addresses [#2523](https://github.com/GoogleContainerTools/skaffold/issues/2523), a
breaking issue that prevented ports for resources from being re-forwarded on redeploy.

New Features:
* Let the user specify a path and a secret name [#2539](https://github.com/GoogleContainerTools/skaffold/pull/2539)
* Add configuration option for sync inference [3/3] [#2088](https://github.com/GoogleContainerTools/skaffold/pull/2088)
* Expose control API for builds, syncs, and deploys [#2450](https://github.com/GoogleContainerTools/skaffold/pull/2450)

Bug Fixes:
* Monitor kubectl logs when port forwarding and retry on error [#2543](https://github.com/GoogleContainerTools/skaffold/pull/2543)
* Make sure logs are not intermixed [#2538](https://github.com/GoogleContainerTools/skaffold/pull/2538)

Updates & Refactors:
* Add a jibGradle sample [#2549](https://github.com/GoogleContainerTools/skaffold/pull/2549)
* Make Jib test projects more lightweight [#2544](https://github.com/GoogleContainerTools/skaffold/pull/2544)
* Add a quicktest Makefile target [#2540](https://github.com/GoogleContainerTools/skaffold/pull/2540)
* Improve Maven/Jib multimodule builds between Minikube and remote clusters [#2122](https://github.com/GoogleContainerTools/skaffold/pull/2122)
* Use test helpers [#2520](https://github.com/GoogleContainerTools/skaffold/pull/2520)
* Better message when a container is terminated [#2514](https://github.com/GoogleContainerTools/skaffold/pull/2514)
* Simpler code [#2532](https://github.com/GoogleContainerTools/skaffold/pull/2532)
* Remove unused code [#2513](https://github.com/GoogleContainerTools/skaffold/pull/2513)
* Fix linter issues [#2527](https://github.com/GoogleContainerTools/skaffold/pull/2527)
* Longer deadline for linters [#2518](https://github.com/GoogleContainerTools/skaffold/pull/2518)
* Code format [#2519](https://github.com/GoogleContainerTools/skaffold/pull/2519)
* Remove duplicate go version [#2517](https://github.com/GoogleContainerTools/skaffold/pull/2517)
* Move test.sh to hack folder [#2515](https://github.com/GoogleContainerTools/skaffold/pull/2515)
* Travis CI: integration stage -> job [#2504](https://github.com/GoogleContainerTools/skaffold/pull/2504)

Huge thanks goes out to all of our contributors for this release:

- Appu
- Balint Pato
- Brian de Alwis
- Cedric Kring
- Charles-Henri GUERIN
- Cornelius Weig
- David Gageot
- Jason McClellan
- JieJhih Jhang
- Marlon Gamez
- Medya Ghazizadeh
- Nick Kubala
- Prashant Arya
- Priya Wadhwa
- Sébastien Le Gall
- Tad Cordle
- Taylor Barrella
- Tejal Desai
- Tom Dickman


# v0.34.0 Release - 07/19/2019

*Note*: This release comes with a new config version `v1beta13`.
        To upgrade your `skaffold.yaml`, use `skaffold fix`. If you don't upgrade, skaffold will auto-upgrade in memory as best it can, and print a warning message.
        See [Skaffold Deprecation Policy](http://skaffold.dev/docs/references/deprecation/) for details on what beta means.

New Features:
* File output flag for writing built images to a specified file [#2476](https://github.com/GoogleContainerTools/skaffold/pull/2476)
* Default to notify trigger [#2482](https://github.com/GoogleContainerTools/skaffold/pull/2482)
* Support --reproducible for Kaniko build [#2453](https://github.com/GoogleContainerTools/skaffold/pull/2453)
* Add `options` command to show global flags [#2454](https://github.com/GoogleContainerTools/skaffold/pull/2454)
* Add deployment health check implementation [#2359](https://github.com/GoogleContainerTools/skaffold/pull/2359)
* Add a `metadata.name` field to skaffold.yaml [#2437](https://github.com/GoogleContainerTools/skaffold/pull/2437)
*  Update skaffold init --analyze to handle more builder types [#2327](https://github.com/GoogleContainerTools/skaffold/pull/2327)
* Support alternative Kustomization config filenames (#2422) [#2439](https://github.com/GoogleContainerTools/skaffold/pull/2439)
* Add resync/rebuild directly in monitor callback [#2438](https://github.com/GoogleContainerTools/skaffold/pull/2438)
* The user is now able to disable RPC in dev mode [#2427](https://github.com/GoogleContainerTools/skaffold/pull/2427)
* Feat(sync): skip sync on non-running pods [#2403](https://github.com/GoogleContainerTools/skaffold/pull/2403)
* Allow for remote kustomize bases [#2269](https://github.com/GoogleContainerTools/skaffold/pull/2269)
* :sparkles: Add support for regexp in profile activation kubeContext [#2065](https://github.com/GoogleContainerTools/skaffold/pull/2065)

Bug Fixes:
* Fix port forwarding in dev loop [#2477](https://github.com/GoogleContainerTools/skaffold/pull/2477)
* Propagate special error on configuration change [#2501](https://github.com/GoogleContainerTools/skaffold/pull/2501)
* Fix proto generation and testing [#2446](https://github.com/GoogleContainerTools/skaffold/pull/2446)
* Add back /v1/event_log endpoint for events [#2436](https://github.com/GoogleContainerTools/skaffold/pull/2436)
* Pruning should happen after Cleanup [#2441](https://github.com/GoogleContainerTools/skaffold/pull/2441)
* Fix script that creates a new version to make it work on osx [#2429](https://github.com/GoogleContainerTools/skaffold/pull/2429)
* Fix proto generation test [#2419](https://github.com/GoogleContainerTools/skaffold/pull/2419)
* Fix Monitor test [#2413](https://github.com/GoogleContainerTools/skaffold/pull/2413)

Updates & Refactors:
* Set statuscheck to false. [#2499](https://github.com/GoogleContainerTools/skaffold/pull/2499)
* Simpler faster find configs [#2494](https://github.com/GoogleContainerTools/skaffold/pull/2494)
* Add a few more examples to commands with -—help [#2489](https://github.com/GoogleContainerTools/skaffold/pull/2489)
* Little increase to Code coverage [#2490](https://github.com/GoogleContainerTools/skaffold/pull/2490)
* Revisited artifact caching [#2470](https://github.com/GoogleContainerTools/skaffold/pull/2470)
* Upgrade go container registry to remove spurious logs [#2487](https://github.com/GoogleContainerTools/skaffold/pull/2487)
* Add `skaffold config` examples [#2483](https://github.com/GoogleContainerTools/skaffold/pull/2483)
* Update ISSUE_TEMPLATE.md [#2486](https://github.com/GoogleContainerTools/skaffold/pull/2486)
* Upgrade to Jib 1.4.0 [#2480](https://github.com/GoogleContainerTools/skaffold/pull/2480)
* Improve the logs when there's no skaffold.yaml [#2467](https://github.com/GoogleContainerTools/skaffold/pull/2467)
* Better mock for docker.ImageID [#2461](https://github.com/GoogleContainerTools/skaffold/pull/2461)
* Upgrade go-containerregistry [#2455](https://github.com/GoogleContainerTools/skaffold/pull/2455)
* [caching] Simpler code and fixing nits [#2456](https://github.com/GoogleContainerTools/skaffold/pull/2456)
* Simplify caching [#2452](https://github.com/GoogleContainerTools/skaffold/pull/2452)
* Test Sync mode with both triggers [#2449](https://github.com/GoogleContainerTools/skaffold/pull/2449)
* Set out on the root command [#2445](https://github.com/GoogleContainerTools/skaffold/pull/2445)
* Kaniko proxy [#2283](https://github.com/GoogleContainerTools/skaffold/pull/2283)
* Freeze v1beta12 and prepare v1beta13 [#2430](https://github.com/GoogleContainerTools/skaffold/pull/2430)
* Correctly migrate sync config in profiles [#2415](https://github.com/GoogleContainerTools/skaffold/pull/2415)
* Improve skaffold help output  [#2324](https://github.com/GoogleContainerTools/skaffold/pull/2324)
* Watch namespaces for each Helm release [#2423](https://github.com/GoogleContainerTools/skaffold/pull/2423)
* Add support for kustomization resources (#2416) [#2420](https://github.com/GoogleContainerTools/skaffold/pull/2420)
* Test regexp usage in profiles activation [#2417](https://github.com/GoogleContainerTools/skaffold/pull/2417)
* Update dev guide with regards to integration tests [#2418](https://github.com/GoogleContainerTools/skaffold/pull/2418)
* Improve `skaffold help` [#2434](https://github.com/GoogleContainerTools/skaffold/pull/2434)
* Remove unused property [#2428](https://github.com/GoogleContainerTools/skaffold/pull/2428)
* Wait for parallel builds to be cancelled on error [#2424](https://github.com/GoogleContainerTools/skaffold/pull/2424)
* Better integration tests [#2406](https://github.com/GoogleContainerTools/skaffold/pull/2406)
* Faster proto generation [#2402](https://github.com/GoogleContainerTools/skaffold/pull/2402)
* Build with Go 1.12 [#2396](https://github.com/GoogleContainerTools/skaffold/pull/2396)
* Move docker code where it belongs [#2393](https://github.com/GoogleContainerTools/skaffold/pull/2393)
* Transfer control of dev loop from file watcher to dev listener [#2354](https://github.com/GoogleContainerTools/skaffold/pull/2354)
* Simplify test debug [#2399](https://github.com/GoogleContainerTools/skaffold/pull/2399)
* Fix minor warnings on doc site [#2389](https://github.com/GoogleContainerTools/skaffold/pull/2389)

Huge thanks goes out to all of our contributors for this release:
- Balint Pato
- Charles-Henri GUERIN
- Cornelius Weig
- David Gageot
- Jason McClellan
- Marlon Gamez
- Medya Ghazizadeh
- Nick Kubala
- Prashant Arya
- Priya Wadhwa
- Sébastien Le Gall
- Tad Cordle
- Taylor Barrella
- Tejal Desai

# v0.33.0 Release - 07/02/2019

*Note*: This release comes with a new config version `v1beta12`.
        To upgrade your `skaffold.yaml`, use `skaffold fix`. If you don't upgrade, skaffold will auto-upgrade in memory as best it can, and print a warning message.
        See [Skaffold Deprecation Policy](http://skaffold.dev/docs/references/deprecation/) for details on what beta means.

New Features:
* Add support for user defined port forwarding [#2336](https://github.com/GoogleContainerTools/skaffold/pull/2336)
* Redesign port forwarding [#2215](https://github.com/GoogleContainerTools/skaffold/pull/2215)
* Support buildArgs with `useDockerCLI=false` and Kaniko [#2299](https://github.com/GoogleContainerTools/skaffold/pull/2299)
* Optimized loading of Docker images into kind nodes [#2286](https://github.com/GoogleContainerTools/skaffold/pull/2286)

Bug Fixes:
* Fix schema doc [#2388](https://github.com/GoogleContainerTools/skaffold/pull/2388)
* Bazel: support sub directories [#2312](https://github.com/GoogleContainerTools/skaffold/pull/2312)
* Custom Builder: Fix bug when no deps specified [#2391](https://github.com/GoogleContainerTools/skaffold/pull/2391)
* Fix missing logs when kaniko exists immediately [#2352](https://github.com/GoogleContainerTools/skaffold/pull/2352)
* Fix support for URL manifests [#2348](https://github.com/GoogleContainerTools/skaffold/pull/2348)
* Start API server only once [#2382](https://github.com/GoogleContainerTools/skaffold/pull/2382)
* Support Cluster config with a path [#2342](https://github.com/GoogleContainerTools/skaffold/pull/2342)
* Schemas: Make sure preferredOrder is in sync with field order in Go structs [#2361](https://github.com/GoogleContainerTools/skaffold/pull/2361)
* Fix handling of multi stage builds [#2340](https://github.com/GoogleContainerTools/skaffold/pull/2340)
* Kaniko: fix host path support [#2333](https://github.com/GoogleContainerTools/skaffold/pull/2333)
* upgrade go bazel rules to in examples to fix bazel breaking release 0.18.6 [#2311](https://github.com/GoogleContainerTools/skaffold/pull/2311)

Updates & Refactors:
* Refactor skaffold init for more flexible builder detection [#2274](https://github.com/GoogleContainerTools/skaffold/pull/2274)
* Configure linter to check for unclosed http body [#2392](https://github.com/GoogleContainerTools/skaffold/pull/2392)
* Jib Builder: add more tests [#2390](https://github.com/GoogleContainerTools/skaffold/pull/2390)
* match some more jib files in owners [#2386](https://github.com/GoogleContainerTools/skaffold/pull/2386)
* Use goroutine for sync [#2378](https://github.com/GoogleContainerTools/skaffold/pull/2378)
* Increase test coverage on Jib Builder [#2383](https://github.com/GoogleContainerTools/skaffold/pull/2383)
* Reduce the amount of logs [#2375](https://github.com/GoogleContainerTools/skaffold/pull/2375)
* Add a test to Kaniko builder [#2371](https://github.com/GoogleContainerTools/skaffold/pull/2371)
* Upgrade kind to 0.4.0 [#2369](https://github.com/GoogleContainerTools/skaffold/pull/2369)
* Improve the logs [#2323](https://github.com/GoogleContainerTools/skaffold/pull/2323)
* Better error message for references that can’t be parsed [#2367](https://github.com/GoogleContainerTools/skaffold/pull/2367)
* Update toolchain and make sure versions are pinned [#2362](https://github.com/GoogleContainerTools/skaffold/pull/2362)
* Add status check flag [#2338](https://github.com/GoogleContainerTools/skaffold/pull/2338)
* Update Jib to 1.3.0 [#2363](https://github.com/GoogleContainerTools/skaffold/pull/2363)
* Validate port-forwards by attempting to bind to port [#2345](https://github.com/GoogleContainerTools/skaffold/pull/2345)
* Add portforward diagram to docs [#2353](https://github.com/GoogleContainerTools/skaffold/pull/2353)
* Restart the dev loop when the skaffold config changes [#2347](https://github.com/GoogleContainerTools/skaffold/pull/2347)
* Kustomize: pick up patchesStrategicMerge changes [#2349](https://github.com/GoogleContainerTools/skaffold/pull/2349)
* Update examples [#2343](https://github.com/GoogleContainerTools/skaffold/pull/2343)
* Move default labeller to deploy since it used in deployer.  [#2335](https://github.com/GoogleContainerTools/skaffold/pull/2335)
* Don’t always start the rpc server [#2328](https://github.com/GoogleContainerTools/skaffold/pull/2328)
* Add missing v1beta11 version [#2332](https://github.com/GoogleContainerTools/skaffold/pull/2332)
* freeze v1beta11 [#2329](https://github.com/GoogleContainerTools/skaffold/pull/2329)
* stop receiving the signals [#2257](https://github.com/GoogleContainerTools/skaffold/pull/2257)
* Improve and test the notify trigger [#2297](https://github.com/GoogleContainerTools/skaffold/pull/2297)
* Add scripts to test and generate files from proto [#2316](https://github.com/GoogleContainerTools/skaffold/pull/2316)
* Improve tests [#2309](https://github.com/GoogleContainerTools/skaffold/pull/2309)
* Refactor the SkaffoldRunner [#2307](https://github.com/GoogleContainerTools/skaffold/pull/2307)
* Add an integration test for Kaniko with a Target [#2308](https://github.com/GoogleContainerTools/skaffold/pull/2308)
* Update Kaniko [#2313](https://github.com/GoogleContainerTools/skaffold/pull/2313)
* Upgrade k8s libraries to 1.12.9 [#2310](https://github.com/GoogleContainerTools/skaffold/pull/2310)
* Jib builder should not reuse commands [#2302](https://github.com/GoogleContainerTools/skaffold/pull/2302)

Huge thanks goes out to all of our contributors for this release:

- Appu
- Balint Pato
- Brian de Alwis
- Cedric Kring
- David Gageot
- JieJhih Jhang
- Nick Kubala
- Priya Wadhwa
- Tad Cordle
- Tejal Desai
- Tom Dickman


# v0.32.0 Release - 06/20/2019

New Features:
* Add resourceType and resourceName to PortForward event [#2272](https://github.com/GoogleContainerTools/skaffold/pull/2272)
* Add custom artifact type to cluster builder [#2048](https://github.com/GoogleContainerTools/skaffold/pull/2048)
* Add Python debugging support [#2205](https://github.com/GoogleContainerTools/skaffold/pull/2205)
* Add K8sManagedBy function to labeller [#2270](https://github.com/GoogleContainerTools/skaffold/pull/2270)
* Add resources to Kaniko init container [#2260](https://github.com/GoogleContainerTools/skaffold/pull/2260)
* Implements `skaffold find-configs -d <dir>` command [#2244](https://github.com/GoogleContainerTools/skaffold/pull/2244)
* Expand values file paths prefixed with ~ [#2233](https://github.com/GoogleContainerTools/skaffold/pull/2233)
* Implement destination inference for sync of dockerfile artifacts [2/3] [#2084](https://github.com/GoogleContainerTools/skaffold/pull/2084)

Bug Fixes:
* Handle `eu.gcr.io` like `gcr.io` when replacing default image [#2300](https://github.com/GoogleContainerTools/skaffold/pull/2300)
* Fix config reload in skaffold dev [#2279](https://github.com/GoogleContainerTools/skaffold/pull/2279)
* Docker is case sensitive about networks [#2288](https://github.com/GoogleContainerTools/skaffold/pull/2288)
* cluster builder fails to detect insecure registries [#2266](https://github.com/GoogleContainerTools/skaffold/pull/2266)
* fix static linking of linux binary [#2252](https://github.com/GoogleContainerTools/skaffold/pull/2252)
* fix racy test [#2251](https://github.com/GoogleContainerTools/skaffold/pull/2251)

Updates & Refactors:
* Remove the `config out of date` warning [#2298](https://github.com/GoogleContainerTools/skaffold/pull/2298)
* Fix codecov2 [#2293](https://github.com/GoogleContainerTools/skaffold/pull/2293)
* Handle simple glob patterns when upgrading the sync patterns [#2287](https://github.com/GoogleContainerTools/skaffold/pull/2287)
* more debug codeowner-ship [#2292](https://github.com/GoogleContainerTools/skaffold/pull/2292)
* comment setup for codecov [#2291](https://github.com/GoogleContainerTools/skaffold/pull/2291)
* adding tests for cluster builder [#2275](https://github.com/GoogleContainerTools/skaffold/pull/2275)
* Update debug owners [#2285](https://github.com/GoogleContainerTools/skaffold/pull/2285)
* add test for one method in diagnose.  [#2238](https://github.com/GoogleContainerTools/skaffold/pull/2238)
* Bumping kustomize version fixes #2137 [#2265](https://github.com/GoogleContainerTools/skaffold/pull/2265)
* Upgrade golangci-lint to v1.17.1 [#2248](https://github.com/GoogleContainerTools/skaffold/pull/2248)
* ability to `make integration` only on a chosen set of integration tests [#2250](https://github.com/GoogleContainerTools/skaffold/pull/2250)
* another testcase for local builder [#2253](https://github.com/GoogleContainerTools/skaffold/pull/2253)
* Simpler Makefile [#2259](https://github.com/GoogleContainerTools/skaffold/pull/2259)
* Add `debug` codeowners [#2247](https://github.com/GoogleContainerTools/skaffold/pull/2247)
* Improve code coverage [#2242](https://github.com/GoogleContainerTools/skaffold/pull/2242)
* Mark `debug` as alpha [#2246](https://github.com/GoogleContainerTools/skaffold/pull/2246)
* Use kind to run integration tests on TravisCI [#2196](https://github.com/GoogleContainerTools/skaffold/pull/2196)
* tests for local.NewBuilder [#2240](https://github.com/GoogleContainerTools/skaffold/pull/2240)
* Add unit tests for WaitForPodSucceeded [#2239](https://github.com/GoogleContainerTools/skaffold/pull/2239)
* remove webhook from coverage report [#2236](https://github.com/GoogleContainerTools/skaffold/pull/2236)
* Dep ensure [#2230](https://github.com/GoogleContainerTools/skaffold/pull/2230)

Huge thanks goes out to all of our contributors for this release:

- Balint Pato
- balopat
- Brian de Alwis
- Cedric Kring
- Cornelius Weig
- David Gageot
- Nick Kubala
- priyawadhwa
- Priya Wadhwa
- steevee
- stephane lacoin
- Stephane Lacoin (aka nxmatic)
- Tejal Desai
- Valentin Fedoskin
- yfei1


# v0.31.0 Release - 06/06/2019

New Features:
* Add CI on windows [#2214](https://github.com/GoogleContainerTools/skaffold/pull/2214)
* Add API build trigger *alpha* [#2201](https://github.com/GoogleContainerTools/skaffold/pull/2201)
* Cross compile Skaffold, with CGO=1, using xgo [#2006](https://github.com/GoogleContainerTools/skaffold/pull/2006)
* Print logs from init containers [#2182](https://github.com/GoogleContainerTools/skaffold/pull/2182)

Bug Fixes:
* Fix master branch [#2221](https://github.com/GoogleContainerTools/skaffold/pull/2221)
* Fix error in flag refactoring for `skaffold run --tail` [#2172](https://github.com/GoogleContainerTools/skaffold/pull/2172)

Updates & refactoring:
* Increase test coverage [#2225](https://github.com/GoogleContainerTools/skaffold/pull/2225)
* Use test wrapper in more tests [#2222](https://github.com/GoogleContainerTools/skaffold/pull/2222)
* Improve documentation for kaniko [#2186](https://github.com/GoogleContainerTools/skaffold/pull/2186)
*  Use the testutils test helper [#2218](https://github.com/GoogleContainerTools/skaffold/pull/2218)
* Remove AppVeyor [#2219](https://github.com/GoogleContainerTools/skaffold/pull/2219)
* Check man page with a unit test instead of a script [#2180](https://github.com/GoogleContainerTools/skaffold/pull/2180)
* Test helper to make tests less verbose [#2193](https://github.com/GoogleContainerTools/skaffold/pull/2193)
* Refactor cmd builder [#2179](https://github.com/GoogleContainerTools/skaffold/pull/2179)
* Faster Travis CI [#2210](https://github.com/GoogleContainerTools/skaffold/pull/2210)
* Simplify schema upgrades: remove duplication [#2212](https://github.com/GoogleContainerTools/skaffold/pull/2212)
* Moar tests [#2195](https://github.com/GoogleContainerTools/skaffold/pull/2195)
* Add a test help to verify that a test panicked [#2194](https://github.com/GoogleContainerTools/skaffold/pull/2194)
* [Refactor] Move gRPC and HTTP server logic out of event package [#2199](https://github.com/GoogleContainerTools/skaffold/pull/2199)
* Update _index.md [#2192](https://github.com/GoogleContainerTools/skaffold/pull/2192)
* Multiple small improvements to unit tests [#2189](https://github.com/GoogleContainerTools/skaffold/pull/2189)
* Test tester [#2181](https://github.com/GoogleContainerTools/skaffold/pull/2181)
* Remove dead code [#2183](https://github.com/GoogleContainerTools/skaffold/pull/2183)
* Remove trailing dot. [#2178](https://github.com/GoogleContainerTools/skaffold/pull/2178)
* Remove $ from example commands [#2177](https://github.com/GoogleContainerTools/skaffold/pull/2177)
* Add sync test and refactor InParallel [#2118](https://github.com/GoogleContainerTools/skaffold/pull/2118)


Huge thanks goes out to all of our contributors for this release:

- Alexandre Ardhuin
- Balint Pato
- Brian de Alwis
- Byungjin Park
- Chanseok Oh
- Charles-Henri GUÉRIN
- Cornelius Weig
- David Gageot
- Dmitri Moore
- Etan Shaul
- Gareth Evans
- g-harel
- guille
- Ilyes Hammadi
- Iván Aponte
- Marcos Ottonello
- Martin Hoefling
- Michael FIG
- Nick Kubala
- Persevere Von
- peter
- Pierre-Yves Aillet
- Prashant Arya
- Priya Wadhwa
- Rahul Sinha
- robertrbruno
- Rory Shively
- Tad Cordle
- Taylor Barrella
- Tejal Desai
- Tigran Tch
- TJ Koblentz
- u5surf
- venkatk-25
- Xiaoxi He


# v0.30.0 Release - 05/23/2019

*Note*: This release comes with a new config version `v1beta11`.
        To upgrade your `skaffold.yaml`, use `skaffold fix`. If you don't upgrade, skaffold will auto-upgrade in memory as best it can, and print a warning message.
        See [Skaffold Deprecation Policy](http://skaffold.dev/docs/references/deprecation/) for details on what beta means.

New Features: 

* Add support for npm run-script-based launches for `skaffold debug` [#2141](https://github.com/GoogleContainerTools/skaffold/pull/2141)
* Support deploying remote helm charts [#2058](https://github.com/GoogleContainerTools/skaffold/pull/2058)
* Option to mount HostPath in each Kaniko Pod to be used as cache volume [#1690](https://github.com/GoogleContainerTools/skaffold/pull/1690)
* Additional git tagger variants (TreeSha, AbbrevTreeSha) [#1905](https://github.com/GoogleContainerTools/skaffold/pull/1905)
* Enable `skaffold debug` for kustomize [#2043](https://github.com/GoogleContainerTools/skaffold/pull/2043)
* :sparkles: Add option `--no-prune-children` [#2113](https://github.com/GoogleContainerTools/skaffold/pull/2113)
* Turn port forwarding off by default [#2115](https://github.com/GoogleContainerTools/skaffold/pull/2115)

Bug Fixes:

* Remove build dependency for helm deploy [#2121](https://github.com/GoogleContainerTools/skaffold/pull/2121)
* Check for env variables for root cmd persistent flags [#2143](https://github.com/GoogleContainerTools/skaffold/pull/2143)
* skaffold debug: log unsupported objects or versions [#2138](https://github.com/GoogleContainerTools/skaffold/pull/2138)
* Don't panic for nil pod watch object [#2112](https://github.com/GoogleContainerTools/skaffold/pull/2112)
* Fix bugs in custom builder [#2130](https://github.com/GoogleContainerTools/skaffold/pull/2130)

Updates & refactoring: 

* Freeze v1beta10 config [#2109](https://github.com/GoogleContainerTools/skaffold/pull/2109)
* Add Annotations to command and flags per phase annotation. [#2022](https://github.com/GoogleContainerTools/skaffold/pull/2022)
* Add smoke test for `skaffold diagnose` [#2157](https://github.com/GoogleContainerTools/skaffold/pull/2157)
* More tests [#2128](https://github.com/GoogleContainerTools/skaffold/pull/2128)
* Refactor the runner [#2155](https://github.com/GoogleContainerTools/skaffold/pull/2155)
* Remove some old plugin related code from event handler [#2156](https://github.com/GoogleContainerTools/skaffold/pull/2156)
* Test helper to override value for tests [#2147](https://github.com/GoogleContainerTools/skaffold/pull/2147)
* Simpler Travis configuration [#2146](https://github.com/GoogleContainerTools/skaffold/pull/2146)
* Remove duplication around cobra code. [#2145](https://github.com/GoogleContainerTools/skaffold/pull/2145)
* Bring helm integration test back [#2140](https://github.com/GoogleContainerTools/skaffold/pull/2140)
* Use testutil.NewTempDir() instead [#2149](https://github.com/GoogleContainerTools/skaffold/pull/2149)
* Simpler code [#2148](https://github.com/GoogleContainerTools/skaffold/pull/2148)
* Use more recent Golang images [#2132](https://github.com/GoogleContainerTools/skaffold/pull/2132)
* Always use the same technique to cleanup global variables in tests. [#2135](https://github.com/GoogleContainerTools/skaffold/pull/2135)
* Update jib [#2133](https://github.com/GoogleContainerTools/skaffold/pull/2133)

Docs updates:

* Fix and improve sync samples [#2131](https://github.com/GoogleContainerTools/skaffold/pull/2131)
* docs: correct header name for jump. [#2079](https://github.com/GoogleContainerTools/skaffold/pull/2079)
* added the notice about skaffold deploy [#2107](https://github.com/GoogleContainerTools/skaffold/pull/2107)
* add explanation to cloud build section docs [#2104](https://github.com/GoogleContainerTools/skaffold/pull/2104)

Design proposals: 


* [Design Proposal] Event API v2 [#1949](https://github.com/GoogleContainerTools/skaffold/pull/1949)
* [Design Proposal] Setting proxy for Kaniko Pod [#2064](https://github.com/GoogleContainerTools/skaffold/pull/2064)


Huge thanks goes out to all of our contributors for this release:

- Balint Pato
- Brian de Alwis
- Charles-Henri GUÉRIN
- Cornelius Weig
- David Gageot
- Iván Aponte
- Martin Hoefling
- Nick Kubala
- Persevere Von
- Prashant Arya
- Priya Wadhwa
- Taylor Barrella
- Tejal Desai

# v0.29.0 Release - 05/09/2019

*Note*: This release comes with a new config version `v1beta10`.
        To upgrade your `skaffold.yaml`, use `skaffold fix`. If you don't upgrade, skaffold will auto-upgrade in memory as best it can, and print a warning message.
        See [Skaffold Deprecation Policy](http://skaffold.dev/docs/references/deprecation/) for details on what beta means.
        
**Note**: `skaffold deploy` now requires images to be built first, `skaffold deploy` will not build images itself. 

Users can use `skaffold deploy` in this flow for example: 

1. `skaffold build -q > built.json`
2. ` skaffold deploy -a built.json` 

Or if users want a single command that builds and deploys, they can still run `skaffold run`. 

New Features: 

* Add command to custom artifact dependencies [#2095](https://github.com/GoogleContainerTools/skaffold/pull/2095)
* Improve syntax for artifact.sync config [1/3] [#1847](https://github.com/GoogleContainerTools/skaffold/pull/1847)
* Add dockerfile to custom dependencies [#2049](https://github.com/GoogleContainerTools/skaffold/pull/2049)
* Automatically watch helm subcharts when skipBuildDependencies is enabled [#1371](https://github.com/GoogleContainerTools/skaffold/pull/1371)
* Allow environment variables to be used in docker build argument [#1912](https://github.com/GoogleContainerTools/skaffold/pull/1912)
* Add option to configure the networking stack in docker build [#2036](https://github.com/GoogleContainerTools/skaffold/pull/2036)
* Allow --no-cache to be passed to docker [#2054](https://github.com/GoogleContainerTools/skaffold/pull/2054)
* Deploy to  consume build output [#2001](https://github.com/GoogleContainerTools/skaffold/pull/2001)
* Add k8 style managed by label to skaffold deployed pods [#2055](https://github.com/GoogleContainerTools/skaffold/pull/2055)
* Support kubectl deploy absolute manifest files [#2011](https://github.com/GoogleContainerTools/skaffold/pull/2011)
        
Bug Fixes:

* Add custom artifact for custom local builds [#1999](https://github.com/GoogleContainerTools/skaffold/pull/1999)
* Add version as unknown if version.Get().Version is empty [#2097](https://github.com/GoogleContainerTools/skaffold/pull/2097)
* Fix image release process: master -> edge, tag -> latest [#2099](https://github.com/GoogleContainerTools/skaffold/pull/2099)
* :bug: fix kubectl apply error handling [#2076](https://github.com/GoogleContainerTools/skaffold/pull/2076)
* Remove podname from port forward key [#2047](https://github.com/GoogleContainerTools/skaffold/pull/2047)
* Correctly parse env-var for multi-valued flags [#2032](https://github.com/GoogleContainerTools/skaffold/pull/2032)

Updates & refactoring: 

* Prefix Skaffold labels with 'skaffold-' [#2062](https://github.com/GoogleContainerTools/skaffold/pull/2062)
* Remove copy paste deploy_test.go [#2085](https://github.com/GoogleContainerTools/skaffold/pull/2085)
* Freeze v1beta9 config [#2035](https://github.com/GoogleContainerTools/skaffold/pull/2035)
* Add unit test for port forwarding key [#2059](https://github.com/GoogleContainerTools/skaffold/pull/2059)
* Refactor kaniko builder to cluster builder [#2037](https://github.com/GoogleContainerTools/skaffold/pull/2037)
* Attaching os standard error and out stream to the copy command [#1960](https://github.com/GoogleContainerTools/skaffold/pull/1960)

Docs updates:

* Mention kind in docs for local development [#2090](https://github.com/GoogleContainerTools/skaffold/pull/2090)
* Clarify which containers are port forwarded [#2078](https://github.com/GoogleContainerTools/skaffold/pull/2078)
* Improve nodejs example to show subdirectories sync [#2024](https://github.com/GoogleContainerTools/skaffold/pull/2024)
* Minor fix on Markdown to follow markdown rules [#2052](https://github.com/GoogleContainerTools/skaffold/pull/2052)
* Note filesync limitation for files not owned by container user [#2041](https://github.com/GoogleContainerTools/skaffold/pull/2041)

Design proposals: 

* Design proposal for sync improvements [#1844](https://github.com/GoogleContainerTools/skaffold/pull/1844)


Huge thanks goes out to all of our contributors for this release:

- Balint Pato
- Brian de Alwis
- Byungjin Park
- Charles-Henri GUÉRIN
- Cornelius Weig
- David Gageot
- Dmitri Moore
- Ilyes Hammadi
- Nick Kubala
- peter
- Pierre-Yves Aillet
- Prashant Arya
- Priya Wadhwa
- Rahul Sinha
- robertrbruno
- Tejal Desai
- Tigran Tch
- Xiaoxi He


# v0.28.0 Release - 04/25/2019

*Note*: This release comes with a new config version `v1beta9`.
        To upgrade your `skaffold.yaml`, use `skaffold fix`. If you don't upgrade, skaffold will auto-upgrade in memory as best it can, and print a warning message.
        See [Skaffold Deprecation Policy](http://skaffold.dev/docs/references/deprecation/) for details on what beta means.

New Features: 
* Git tagger variants (Tags, CommitSha, AbbrevCommitSha) [#1902](https://github.com/GoogleContainerTools/skaffold/pull/1902)
* Add `--force` command line option to run and deploy sub-commands [#1568](https://github.com/GoogleContainerTools/skaffold/pull/1568)
* Full validation for `required` and `oneOf` config fields [#1939](https://github.com/GoogleContainerTools/skaffold/pull/1939)
* Add hidden flag `--force-colors` to always print color codes [#2033](https://github.com/GoogleContainerTools/skaffold/pull/2033)

Bug Fixes:
* Rename SkaffoldPipeline to SkaffoldConfig [#2015](https://github.com/GoogleContainerTools/skaffold/pull/2015)
* Fix typo [#2013](https://github.com/GoogleContainerTools/skaffold/pull/2013)
* Include runtime dependencies for taggers in `gcr.io/k8s-skaffold/skaffold` [#1987](https://github.com/GoogleContainerTools/skaffold/pull/1987)
* fix show some option in skaffold delete #1995 [#1997](https://github.com/GoogleContainerTools/skaffold/pull/1997)
* Fix panic when upgrading configurations with patches [#1971](https://github.com/GoogleContainerTools/skaffold/pull/1971)
* Fix error message when the `skaffold.yaml` is not found [#1947](https://github.com/GoogleContainerTools/skaffold/pull/1947)
* Fix syncing for Jib [#1926](https://github.com/GoogleContainerTools/skaffold/pull/1926)

Updates & refactoring: 
* Reduce overhead of Jib builder [#1744](https://github.com/GoogleContainerTools/skaffold/pull/1744)
* Remove plugin code from config [#2016](https://github.com/GoogleContainerTools/skaffold/pull/2016)
* Update a few dependencies [#2020](https://github.com/GoogleContainerTools/skaffold/pull/2020)
* Remove some dead code [#2017](https://github.com/GoogleContainerTools/skaffold/pull/2017)
* Don’t fetch the same config twice [#2014](https://github.com/GoogleContainerTools/skaffold/pull/2014)
* Remove unused instructions from Makefile [#2012](https://github.com/GoogleContainerTools/skaffold/pull/2012)
*  Remove bazel plugin & revert back to original [#1989](https://github.com/GoogleContainerTools/skaffold/pull/1989)
*  Remove docker plugin and revert to original code structure [#1990](https://github.com/GoogleContainerTools/skaffold/pull/1990)
* Don't run GCB example on structure tests [#1984](https://github.com/GoogleContainerTools/skaffold/pull/1984)
* Use `RunOrFailOutput` instead of `RunOrFail` to see the error logs in test [#1976](https://github.com/GoogleContainerTools/skaffold/pull/1976)
* Freeze v1beta8 skaffold config [#1965](https://github.com/GoogleContainerTools/skaffold/pull/1965)
* Remove the experimental UI [#1953](https://github.com/GoogleContainerTools/skaffold/pull/1953)
* Always configure which command runs [#1956](https://github.com/GoogleContainerTools/skaffold/pull/1956)

Docs updates:

* Add more github shields [#2026](https://github.com/GoogleContainerTools/skaffold/pull/2026)
* Improve Skaffold-Jib docs for Maven multi-module projects [#1993](https://github.com/GoogleContainerTools/skaffold/pull/1993)
* Add contributing docs for making a config change [#1982](https://github.com/GoogleContainerTools/skaffold/pull/1982)
* Add start on filesync doc [#1994](https://github.com/GoogleContainerTools/skaffold/pull/1994)
* Add some documentation for container structure tests [#1959](https://github.com/GoogleContainerTools/skaffold/pull/1959)
* Add documentation for insecure registries [#1973](https://github.com/GoogleContainerTools/skaffold/pull/1973)
* Add documentation for local development setups [#1970](https://github.com/GoogleContainerTools/skaffold/pull/1970)


Huge thanks goes out to all of our contributors for this release:

- Alexandre Ardhuin
- Balint Pato
- Brian de Alwis
- Cornelius Weig
- David Gageot
- Nick Kubala
- Priya Wadhwa
- Tad Cordle
- Tejal Desai
- u5surf


# v0.27.0 Release - 04/12/2019

*Note*: This release comes with a new config version `v1beta8`.
        To upgrade your `skaffold.yaml`, use `skaffold fix`. If you don't upgrade, skaffold will auto-upgrade in memory as best it can, and print a warning message.
        See [Skaffold Deprecation Policy](http://skaffold.dev/docs/references/deprecation/) for details on what beta means.

New Features:

* Add support for pushing/pulling to insecure registries [#1870](https://github.com/GoogleContainerTools/skaffold/pull/1870)
* Minor doc updates [#1923](https://github.com/GoogleContainerTools/skaffold/pull/1923)
* Specify the resource requirements for the kaniko pod in Skaffold Config [#1683](https://github.com/GoogleContainerTools/skaffold/pull/1683)
* Validate pipeline config [#1881](https://github.com/GoogleContainerTools/skaffold/pull/1881)
* Configure Jib builds to use plain progress updates [#1895](https://github.com/GoogleContainerTools/skaffold/pull/1895)
* Remove intermediate images and containers from local builds by default [#1400](https://github.com/GoogleContainerTools/skaffold/pull/1400)

Bug Fixes:

* remove runcontext creation from gcb builder path [#1944](https://github.com/GoogleContainerTools/skaffold/pull/1944)
* Remove duplicate cache code [#1922](https://github.com/GoogleContainerTools/skaffold/pull/1922)
* fixing non-oneof inline struct handling in schemas [#1904](https://github.com/GoogleContainerTools/skaffold/pull/1904)
* Undo fmt.Fprint -> color.White.Fprint [#1903](https://github.com/GoogleContainerTools/skaffold/pull/1903)
* ctx->runCtx [#1940](https://github.com/GoogleContainerTools/skaffold/pull/1940)
* Fix flakes with rpc integration test [#1860](https://github.com/GoogleContainerTools/skaffold/pull/1860)

Updates & refactoring:

* remove inline structs from schema [#1913](https://github.com/GoogleContainerTools/skaffold/pull/1913)
* Introduce RunContext object for passing necessary context to runner constructor methods [#1885](https://github.com/GoogleContainerTools/skaffold/pull/1885)
* extracting Pipeline from SkaffoldConfig [#1899](https://github.com/GoogleContainerTools/skaffold/pull/1899)
* Freeze v1alpha7 skaffold config version [#1914](https://github.com/GoogleContainerTools/skaffold/pull/1914)
* Adding a design proposal template and README. [#1838](https://github.com/GoogleContainerTools/skaffold/pull/1838)
* Upgrade golangci-lint to v1.16.0 [#1898](https://github.com/GoogleContainerTools/skaffold/pull/1898)
* Remove container-friendly flags for Java 8 [#1894](https://github.com/GoogleContainerTools/skaffold/pull/1894)
* Improve helm examples [#1891](https://github.com/GoogleContainerTools/skaffold/pull/1891)

# v0.26.0 Release - 3/27/2019


New features:

* Add debugging support for Skaffold: `skaffold debug` [#1702](https://github.com/GoogleContainerTools/skaffold/pull/1702)
* Add portName to the PortEvent payload of the event api [#1855](https://github.com/GoogleContainerTools/skaffold/pull/1855)
* Add HTTP reverse proxy for gRPC server to expose REST API for event server [#1825](https://github.com/GoogleContainerTools/skaffold/pull/1825)
* Preserve sync subtree for '***'. [#1813](https://github.com/GoogleContainerTools/skaffold/pull/1813)
* Error if no Dockerfiles are found for skaffold init --analyze  [#1810](https://github.com/GoogleContainerTools/skaffold/pull/1810)

Fixes:

* Fix unnecessary warning in caching [#1873](https://github.com/GoogleContainerTools/skaffold/pull/1873)
* Add folders to tarballs [#1878](https://github.com/GoogleContainerTools/skaffold/pull/1878)
* Fix go routine leak [#1874](https://github.com/GoogleContainerTools/skaffold/pull/1874)
* Fix skaffold build templating output and add tests [#1841](https://github.com/GoogleContainerTools/skaffold/pull/1841)
* Don't expose ports to the outside and fix a race condition [#1850](https://github.com/GoogleContainerTools/skaffold/pull/1850)
* removing goroutine leak [#1871](https://github.com/GoogleContainerTools/skaffold/pull/1871)
* Verify patches and fail with a proper error message [#1864](https://github.com/GoogleContainerTools/skaffold/pull/1864)
* Support 1.11+ as a kubectl version [#1843](https://github.com/GoogleContainerTools/skaffold/pull/1843)

Updates & refactorings:

* Add integration testing and example for skipBuildDependencies option [#1368](https://github.com/GoogleContainerTools/skaffold/pull/1368)
* Improve Doc’s Dockerfile [#1875](https://github.com/GoogleContainerTools/skaffold/pull/1875)
* Add tests for skaffold init walk flow. [#1809](https://github.com/GoogleContainerTools/skaffold/pull/1809)
* Refactor Wait Utils Into Watchers [#1811](https://github.com/GoogleContainerTools/skaffold/pull/1811)
* Enhance hack/check-samples script [#1858](https://github.com/GoogleContainerTools/skaffold/pull/1858)
* removing unnecessary exit from plugin processes [#1848](https://github.com/GoogleContainerTools/skaffold/pull/1848)
* Improve test coverage [#1840](https://github.com/GoogleContainerTools/skaffold/pull/1840)
* Fix warning with `find` on TravisCI [#1846](https://github.com/GoogleContainerTools/skaffold/pull/1846)
* Basic unit test to go through all the cobra related code [#1835](https://github.com/GoogleContainerTools/skaffold/pull/1835)
* Compute tags in parallel [#1820](https://github.com/GoogleContainerTools/skaffold/pull/1820)
* Increase integration tests timeout to 15minutes [#1834](https://github.com/GoogleContainerTools/skaffold/pull/1834)
* Faster git tagger [#1817](https://github.com/GoogleContainerTools/skaffold/pull/1817)
* Add unit tests for kustomize [#1828](https://github.com/GoogleContainerTools/skaffold/pull/1828)
* Improve test coverage [#1827](https://github.com/GoogleContainerTools/skaffold/pull/1827)
* Remove duplication in eventing [#1829](https://github.com/GoogleContainerTools/skaffold/pull/1829)
* Simplify upgrade code [#1830](https://github.com/GoogleContainerTools/skaffold/pull/1830)
* Check that samples are both in ./examples and ./integration/examples [#1832](https://github.com/GoogleContainerTools/skaffold/pull/1832)
* Add total time for `skaffold build` [#1818](https://github.com/GoogleContainerTools/skaffold/pull/1818)
* Check cached artifacts in parallel [#1821](https://github.com/GoogleContainerTools/skaffold/pull/1821)
* Debug integration tests [#1816](https://github.com/GoogleContainerTools/skaffold/pull/1816)
* Faster doc preview [#1773](https://github.com/GoogleContainerTools/skaffold/pull/1773)

Docs updates:

* Improve manual installation instruction for windows [#1883](https://github.com/GoogleContainerTools/skaffold/pull/1883)
* more docs for profiles [#1882](https://github.com/GoogleContainerTools/skaffold/pull/1882)
* Add missing env variables in CLI reference doc [#1863](https://github.com/GoogleContainerTools/skaffold/pull/1863)
* Add React example app featuring hot module reload [#1826](https://github.com/GoogleContainerTools/skaffold/pull/1826)
* fix Markdown rendering deprecation-policy.md [#1845](https://github.com/GoogleContainerTools/skaffold/pull/1845)
* Fix Safari issue on skaffold.dev yaml reference [#1831](https://github.com/GoogleContainerTools/skaffold/pull/1831)


Huge thanks goes out to all of our contributors for this release:

- Balint Pato
- Brian de Alwis
- Cornelius Weig
- David Gageot
- Etan Shaul
- g-harel
- Michael FIG
- Nick Kubala
- peter
- Priya Wadhwa
- Tejal Desai

# v0.25.0 Release - 3/15/2019

*Note*: This release comes with a new config version `v1beta7`.
        To upgrade your `skaffold.yaml`, use `skaffold fix`. If you don't upgrade, skaffold will auto-upgrade in memory as best it can, and print a warning message.
        See [Skaffold Deprecation Policy](http://skaffold.dev/docs/references/deprecation/) for details on what beta means.


*Deprecation notice*: With this release we mark for deprecation the `flags` (KanikoArtifact.AdditionalFlags) field in kaniko; instead Kaniko's additional flags will now be represented as unique fields under `kaniko` per artifact (`KanikoArtifact` type).
This flag will be removed earliest 06/15/2019.

New features:

* Config upgrade: handle helm overrides [#1646](https://github.com/GoogleContainerTools/skaffold/pull/1646)
* Enable custom InitContainer image in LocalDir build of kaniko [#1727](https://github.com/GoogleContainerTools/skaffold/pull/1727)
* Add --analyze flag to skaffold init [#1725](https://github.com/GoogleContainerTools/skaffold/pull/1725)

Fixes:

* Initialize Artifact.Workspace to "." by default in plugin case too [#1804](https://github.com/GoogleContainerTools/skaffold/pull/1804)
* Fix race conditions and run tests with a race detector [#1801](https://github.com/GoogleContainerTools/skaffold/pull/1801)
* Support ctrl-c during tagging and cache checking [#1796](https://github.com/GoogleContainerTools/skaffold/pull/1796)
* Fix race in event logs [#1786](https://github.com/GoogleContainerTools/skaffold/pull/1786)
* Fix schema [#1785](https://github.com/GoogleContainerTools/skaffold/pull/1785)
* helm secrets integration [#1617](https://github.com/GoogleContainerTools/skaffold/pull/1617)
* Regenerating schemas for v1beta6 and v1beta7 [#1757](https://github.com/GoogleContainerTools/skaffold/pull/1757)
* Fix typo in option name for 'enable-rpc' [#1718](https://github.com/GoogleContainerTools/skaffold/pull/1718)
* Test that images that can’t be built are not pushed [#1729](https://github.com/GoogleContainerTools/skaffold/pull/1729)

Updates & refactorings:

* v1beta7 [#1751](https://github.com/GoogleContainerTools/skaffold/pull/1751)
* Refactor KanikoBuild into KanikoArtifact and Cluster [#1797](https://github.com/GoogleContainerTools/skaffold/pull/1797)
* Add logic for finding next available port for gRPC if provided one is in use [#1752](https://github.com/GoogleContainerTools/skaffold/pull/1752)
* Check for artifacts in cache in parallel [#1799](https://github.com/GoogleContainerTools/skaffold/pull/1799)
* combined output for integration tests skaffold runner [#1800](https://github.com/GoogleContainerTools/skaffold/pull/1800)
* Remove debug code [#1802](https://github.com/GoogleContainerTools/skaffold/pull/1802)
* Make integration tests shorter and more stable [#1790](https://github.com/GoogleContainerTools/skaffold/pull/1790)
* Initialize LocalCluster in docker local builder plugin [#1791](https://github.com/GoogleContainerTools/skaffold/pull/1791)
* Faster integration tests [#1789](https://github.com/GoogleContainerTools/skaffold/pull/1789)
* Fake k8s context for test [#1788](https://github.com/GoogleContainerTools/skaffold/pull/1788)
* Move bazel code into plugins directory [#1707](https://github.com/GoogleContainerTools/skaffold/pull/1707)
* Add Initializer Interface to skaffold to support other deployers in skaffold init [#1756](https://github.com/GoogleContainerTools/skaffold/pull/1756)
* Refactor caching [#1779](https://github.com/GoogleContainerTools/skaffold/pull/1779)
* Try newer versions of Go [#1775](https://github.com/GoogleContainerTools/skaffold/pull/1775)
* Add back tracking of forwarded ports to avoid race condition [#1780](https://github.com/GoogleContainerTools/skaffold/pull/1780)
* Refactor local builder docker code into plugins directory [#1717](https://github.com/GoogleContainerTools/skaffold/pull/1717)
* Improve `make test` [#1776](https://github.com/GoogleContainerTools/skaffold/pull/1776)
* Upgrade the linter [#1777](https://github.com/GoogleContainerTools/skaffold/pull/1777)
* Simplify port choosing logic [#1747](https://github.com/GoogleContainerTools/skaffold/pull/1747)
* Remove duplication integration tests [#1760](https://github.com/GoogleContainerTools/skaffold/pull/1760)
* Upgrade Jib to 1.0.2 [#1772](https://github.com/GoogleContainerTools/skaffold/pull/1772)
* added some extra logging for test failures for easier feedback [#1763](https://github.com/GoogleContainerTools/skaffold/pull/1763)
* Improve caching [#1755](https://github.com/GoogleContainerTools/skaffold/pull/1755)
* Fix bug in jib in GCB [#1754](https://github.com/GoogleContainerTools/skaffold/pull/1754)
* Only get images list once for caching [#1758](https://github.com/GoogleContainerTools/skaffold/pull/1758)
* Simplify integration tests [#1750](https://github.com/GoogleContainerTools/skaffold/pull/1750)
* Nicer output [#1745](https://github.com/GoogleContainerTools/skaffold/pull/1745)
* Upgrade Kaniko to 0.9.0 [#1736](https://github.com/GoogleContainerTools/skaffold/pull/1736)
* Improve artifact caching [#1741](https://github.com/GoogleContainerTools/skaffold/pull/1741)
* Only go through images once for artifact caching [#1743](https://github.com/GoogleContainerTools/skaffold/pull/1743)
* Try to use the local docker to get the image config [#1735](https://github.com/GoogleContainerTools/skaffold/pull/1735)
* Update go-containerregistry [#1730](https://github.com/GoogleContainerTools/skaffold/pull/1730)
* Improve `skaffold init` performance by not walking hidden dirs. [#1724](https://github.com/GoogleContainerTools/skaffold/pull/1724)

Docs updates:
* added subcommands to the cli reference [#1793](https://github.com/GoogleContainerTools/skaffold/pull/1793)
* Add instructions to DEVELOPMENT.md for installing tools [#1764](https://github.com/GoogleContainerTools/skaffold/pull/1764)
* adding more logs for webhook [#1782](https://github.com/GoogleContainerTools/skaffold/pull/1782)
* Don’t break pages that reference `annotated-skaffold.yaml` [#1770](https://github.com/GoogleContainerTools/skaffold/pull/1770)
* Fix regression in sync [#1722](https://github.com/GoogleContainerTools/skaffold/pull/1722)
* Bail out on docker build error [#1723](https://github.com/GoogleContainerTools/skaffold/pull/1723)
* Updated Install section [#1716](https://github.com/GoogleContainerTools/skaffold/pull/1716)


Huge thanks goes out to all of our contributors for this release:

- Balint Pato
- Chanseok Oh
- Cornelius Weig
- David Gageot
- Michael FIG
- Nick Kubala
- Priya Wadhwa
- Rory Shively
- Tejal Desai
- balopat
- guille
- priyawadhwa
- venkatk-25


# v0.24.0 Release - 3/1/2019

*Note*: This release comes with a new config version `v1beta6`.
To upgrade your `skaffold.yaml`, use `skaffold fix`. If you don't upgrade, skaffold will auto-upgrade in memory as best it can, and print a warning message.
See [Skaffold Deprecation Policy](http://skaffold.dev/docs/references/deprecation/) for details on what beta means.

New Features:
* Add gRPC based event API [#1574](https://github.com/GoogleContainerTools/skaffold/pull/1574)
* Add artifact cache to track artifacts for faster restart  [#1632](https://github.com/GoogleContainerTools/skaffold/pull/1632)
* Helm flags for Global, Install and Upgrade helm commands [#1673](https://github.com/GoogleContainerTools/skaffold/pull/1673)
* v1beta6 [#1674](https://github.com/GoogleContainerTools/skaffold/pull/1674)
* Diagnose skaffold.yaml [#1686](https://github.com/GoogleContainerTools/skaffold/pull/1686)
* Added local execution environment to docker builder plugin [#1656](https://github.com/GoogleContainerTools/skaffold/pull/1656)
* Added bazel in local execution environment [#1662](https://github.com/GoogleContainerTools/skaffold/pull/1662)

Fixes:
* Fix bug in sync [#1709](https://github.com/GoogleContainerTools/skaffold/pull/1709)
* Fix schemas [#1701](https://github.com/GoogleContainerTools/skaffold/pull/1701)
* Fix default-repo handling for images with non-alphabetic characters [#1697](https://github.com/GoogleContainerTools/skaffold/pull/1697)
* Fix gke connection for Integration tests [#1699](https://github.com/GoogleContainerTools/skaffold/pull/1699)
* Handle pointers in profile overlay [#1693](https://github.com/GoogleContainerTools/skaffold/pull/1693)

Updates & refactorings:
* Build before [#1694](https://github.com/GoogleContainerTools/skaffold/pull/1694)
* completion: add wrapper code to transform bash to zsh completion [#1685](https://github.com/GoogleContainerTools/skaffold/pull/1685)
* Add a test for changing tests with a profile [#1687](https://github.com/GoogleContainerTools/skaffold/pull/1687)
* Add example of a Jib-Maven multi-module project [#1676](https://github.com/GoogleContainerTools/skaffold/pull/1676)
* Restructure integration tests [#1678](https://github.com/GoogleContainerTools/skaffold/pull/1678)
* added logging to skaffold dev integration tests [#1684](https://github.com/GoogleContainerTools/skaffold/pull/1684)
* added default-repo to getting started [#1672](https://github.com/GoogleContainerTools/skaffold/pull/1672)
* Make the hot-reload example more exemplary [#1680](https://github.com/GoogleContainerTools/skaffold/pull/1680)


Docs updates:
* Generate skaffold references [#1675](https://github.com/GoogleContainerTools/skaffold/pull/1675)
* Update HUGO [#1679](https://github.com/GoogleContainerTools/skaffold/pull/1679)
* first cut at jib doc [#1661](https://github.com/GoogleContainerTools/skaffold/pull/1661)
* Generate annotated-skaffold.yaml [#1659](https://github.com/GoogleContainerTools/skaffold/pull/1659)
* Improve documentation [#1713](https://github.com/GoogleContainerTools/skaffold/pull/1713)
* Improve docs [#1682](https://github.com/GoogleContainerTools/skaffold/pull/1682)

Huge thanks goes out to all of our contributors for this release:

- Balint Pato
- Brian de Alwis
- Cornelius Weig
- David Gageot
- Jonas Eckerström
- Nick Kubala
- Priya Wadhwa
- Tjerk Wolterink


# v0.23.0 Release - 2/14/2019

*Note*: This release comes with a new config version `v1beta5`.
        To upgrade your `skaffold.yaml`, use `skaffold fix`. If you don't upgrade, skaffold will auto-upgrade in memory as best it can, and print a warning message.
        See [Skaffold Deprecation Policy](http://skaffold.dev/docs/references/deprecation/) for details on what beta means.

*Deprecation notice*: With this release we mark for deprecation the following env variables in the `envTemplate` tagger:
- `DIGEST`
- `DIGEST_ALGO`
- `DIGEST_HEX`
Currently these variables resolve to `_DEPRECATED_<envvar>_`, and the new tagging mechanism adds a digest to the image name thus it shouldn't break existing configurations.
This backward compatibility behavior will be removed earliest 05/14/2019.

New features:
* Builder plugin for docker in GCB [#1577](https://github.com/GoogleContainerTools/skaffold/pull/1577)
* Add custom build arguments in jib artifacts [#1609](https://github.com/GoogleContainerTools/skaffold/pull/1609)
* Generate json schema [#1644](https://github.com/GoogleContainerTools/skaffold/pull/1644)
* Add --color option [#1618](https://github.com/GoogleContainerTools/skaffold/pull/1618)
* v1beta5 [#1610](https://github.com/GoogleContainerTools/skaffold/pull/1610)
* Experimental UI mode for `skaffold dev` [#1533](https://github.com/GoogleContainerTools/skaffold/pull/1533)
* Upgrade to Kaniko v0.8.0 [#1603](https://github.com/GoogleContainerTools/skaffold/pull/1603)
* New tagging mechanism [#1482](https://github.com/GoogleContainerTools/skaffold/pull/1482)
* Add --build-image option to build command [#1591](https://github.com/GoogleContainerTools/skaffold/pull/1591)
* Allow user to specify custom kaniko image [#1588](https://github.com/GoogleContainerTools/skaffold/pull/1588)
* Better profiles [#1541](https://github.com/GoogleContainerTools/skaffold/pull/1541)

Fixes:
* Don't push all tags when sha256 builds just one [#1634](https://github.com/GoogleContainerTools/skaffold/pull/1634)
* Handle env commands with multiple variable definitions (#1625) [#1626](https://github.com/GoogleContainerTools/skaffold/pull/1626)
* Rollback Docker dependencies filtering based on target [#1620](https://github.com/GoogleContainerTools/skaffold/pull/1620)
* Fix sub directory support with Kaniko and GCB [#1613](https://github.com/GoogleContainerTools/skaffold/pull/1613)
* Fix regression from port forwarding [#1616](https://github.com/GoogleContainerTools/skaffold/pull/1616)
* Check for new skaffold version when skaffold.yaml parsing fails [#1587](https://github.com/GoogleContainerTools/skaffold/pull/1587)
* Propagate --skip-tests to builders [#1598](https://github.com/GoogleContainerTools/skaffold/pull/1598)
* fix docs build [#1607](https://github.com/GoogleContainerTools/skaffold/pull/1607)
* Ignore cache-from pull errors [#1604](https://github.com/GoogleContainerTools/skaffold/pull/1604)
* `[kubectl]` apply labels by patching yaml [#1489](https://github.com/GoogleContainerTools/skaffold/pull/1489)

Updates & refactorings:
* Optimize sync [#1641](https://github.com/GoogleContainerTools/skaffold/pull/1641)
* kubectl deployer: warn when pattern matches no file [#1647](https://github.com/GoogleContainerTools/skaffold/pull/1647)
* Add integration tests for taggers [#1635](https://github.com/GoogleContainerTools/skaffold/pull/1635)
* Adding a few tests for `skaffold build` [#1628](https://github.com/GoogleContainerTools/skaffold/pull/1628)
* adding scripts for preparing new config version [#1584](https://github.com/GoogleContainerTools/skaffold/pull/1584)
* Remove Tagger from Builder interface [#1601](https://github.com/GoogleContainerTools/skaffold/pull/1601)
* copyright 2019 [#1606](https://github.com/GoogleContainerTools/skaffold/pull/1606)
* Remove unused constants [#1602](https://github.com/GoogleContainerTools/skaffold/pull/1602)
* Remove stopped containers in make targets [#1590](https://github.com/GoogleContainerTools/skaffold/pull/1590)
* Add missing tests for build/sequence.go [#1575](https://github.com/GoogleContainerTools/skaffold/pull/1575)
* Extract yaml used in documentation into files [#1593](https://github.com/GoogleContainerTools/skaffold/pull/1593)

Docs updates:
* Improve comments and schema [#1652](https://github.com/GoogleContainerTools/skaffold/pull/1652)
* Add `required` tags [#1642](https://github.com/GoogleContainerTools/skaffold/pull/1642)
* Add more comments to the Config structs [#1630](https://github.com/GoogleContainerTools/skaffold/pull/1630)
* Add short docs about automatic port-forwarding [#1637](https://github.com/GoogleContainerTools/skaffold/pull/1637)
* Improve documentation [#1599](https://github.com/GoogleContainerTools/skaffold/pull/1599)
* Fix DEVELOPMENT.md fragment [#1576](https://github.com/GoogleContainerTools/skaffold/pull/1576)
* Improve the Skaffold.dev documentation [#1579](https://github.com/GoogleContainerTools/skaffold/pull/1579)

Huge thanks goes out to all of our contributors for this release:

- Balint Pato
- Brian de Alwis
- Cornelius Weig
- David Gageot
- Michael Beaumont
- Michael FIG
- Nick Kubala
- Priya Wadhwa
- Shuhei Kitagawa


# v0.22.0 Release - 1/31/2019

*Note*: This release comes with a new config version `v1beta4`.
        To upgrade your `skaffold.yaml`, use `skaffold fix`. If you don't upgrade, skaffold will auto-upgrade in memory as best it can, and print a warning message.
        See [Skaffold Deprecation Policy](http://skaffold.dev/docs/references/deprecation/) for details on what beta means.

New features:
* Introduce configuration option to configure image pushing per kube-context [#1355](https://github.com/GoogleContainerTools/skaffold/pull/1355)
* Better support for docker build with a target [#1497](https://github.com/GoogleContainerTools/skaffold/pull/1497)
* Reintroduce the fsNotify trigger [#1562](https://github.com/GoogleContainerTools/skaffold/pull/1562)
* Add zsh completion [#1531](https://github.com/GoogleContainerTools/skaffold/pull/1531)
* `#296` Support remote helm chart repositories [#1254](https://github.com/GoogleContainerTools/skaffold/pull/1254)

Fixes:
* Fix bug in port forwarding [#1529](https://github.com/GoogleContainerTools/skaffold/pull/1529)
* Fix doc for Kustomize deploy: path option [#1527](https://github.com/GoogleContainerTools/skaffold/pull/1527)
* Fix broken links in Getting Started [#1523](https://github.com/GoogleContainerTools/skaffold/pull/1523)
* Use configured namespace for pod watcher. [#1473](https://github.com/GoogleContainerTools/skaffold/pull/1473)
* Pass DOCKER* env variables for jib to connect to minikube [#1505](https://github.com/GoogleContainerTools/skaffold/pull/1505)

Updates & Refactorings:
* Upgrade to jib 1.0.0 [#1512](https://github.com/GoogleContainerTools/skaffold/pull/1512)
* Don’t use local Docker to push Bazel images [#1493](https://github.com/GoogleContainerTools/skaffold/pull/1493)
* Use kubectl to read the manifests [#1451](https://github.com/GoogleContainerTools/skaffold/pull/1451)
* Simplify integration tests [#1539](https://github.com/GoogleContainerTools/skaffold/pull/1539)
* Fix master branch [#1569](https://github.com/GoogleContainerTools/skaffold/pull/1569)
* Add missing tests for watch/triggers [#1557](https://github.com/GoogleContainerTools/skaffold/pull/1557)
* Improve triggers [#1561](https://github.com/GoogleContainerTools/skaffold/pull/1561)
* Add tests for labels package [#1534](https://github.com/GoogleContainerTools/skaffold/pull/1534)

Docs updates:
* Fix skaffold.dev indexing on Google [#1547](https://github.com/GoogleContainerTools/skaffold/pull/1547)
* 2019 roadmap [#1530](https://github.com/GoogleContainerTools/skaffold/pull/1530)
* Should be v1beta3 [#1515](https://github.com/GoogleContainerTools/skaffold/pull/1515)
* Renaming the CoC for GitHub [#1518](https://github.com/GoogleContainerTools/skaffold/pull/1518)
* Add Priya as a Codeowner [#1544](https://github.com/GoogleContainerTools/skaffold/pull/1544)
* Add Priya as a maintainer [#1542](https://github.com/GoogleContainerTools/skaffold/pull/1542)
* Note JVM flags specific to Java 8 in examples/jib [#1563](https://github.com/GoogleContainerTools/skaffold/pull/1563)

Huge thanks goes out to all of our contributors for this release:

- Balint Pato
- Brian de Alwis
- Cornelius Weig
- David Gageot
- Koen De Keyser
- Labesse Kévin
- Michael FIG
- Nick Kubala
- Priya Wadhwa
- Shuhei Kitagawa
- czhc


# v0.21.1 Release - 1/22/2019

New Features:
* Add a log when bazel deps take a long time [#1498](https://github.com/GoogleContainerTools/skaffold/pull/1498)
* Pre-pull cache-from images [#1495](https://github.com/GoogleContainerTools/skaffold/pull/1495)
* Pass bazel args to `bazel info bazel-bin` [#1487](https://github.com/GoogleContainerTools/skaffold/pull/1487)
* Support secretGenerators with kustomize  [#1488](https://github.com/GoogleContainerTools/skaffold/pull/1488)


Fixes:
* Fix coloured output when building in // [#1501](https://github.com/GoogleContainerTools/skaffold/pull/1501)
* Fix onbuild analysis [#1491](https://github.com/GoogleContainerTools/skaffold/pull/1491)
* Fix Broken link to references/config in documentation [#1486](https://github.com/GoogleContainerTools/skaffold/pull/1486)


Updates & refactorings:
* Add error for non Docker artifacts built with Kaniko [#1494](https://github.com/GoogleContainerTools/skaffold/pull/1494)
* Update bazel example [#1492](https://github.com/GoogleContainerTools/skaffold/pull/1492)
* Revert "Merge pull request #1439 from ltouati/fsnotify" [#1508](https://github.com/GoogleContainerTools/skaffold/pull/1508)
* Don’t log if nothing is copied or deleted [#1504](https://github.com/GoogleContainerTools/skaffold/pull/1504)
* Add more integration tests [#1502](https://github.com/GoogleContainerTools/skaffold/pull/1502)
* Remove file committed by error [#1500](https://github.com/GoogleContainerTools/skaffold/pull/1500)


Docs updates:
* Update doc around local development [#1446](https://github.com/GoogleContainerTools/skaffold/pull/1446)
* [doc] Fix default value for manifests [#1485](https://github.com/GoogleContainerTools/skaffold/pull/1485)

Huge thanks goes out to all of our contributors for this release:

- David Gageot
- Nick Kubala
- Priya Wadhwa
- Shane Lee


# v0.21.0 Release - 1/17/2019

*Note*: This release comes with a new config version `v1beta3`.
        To upgrade your `skaffold.yaml`, use `skaffold fix`. If you don't upgrade, skaffold will auto-upgrade in memory as best it can, and print a warning message.
        See [Skaffold Deprecation Policy](http://skaffold.dev/docs/references/deprecation/) for details on what beta means.

New Features:
* Add support for urls in deploy.kubectl.manifests [#1408](https://github.com/GoogleContainerTools/skaffold/pull/1408)
* Add some tests for Sync [#1406](https://github.com/GoogleContainerTools/skaffold/pull/1406)
* Get digest on push and imageID on build [#1428](https://github.com/GoogleContainerTools/skaffold/pull/1428)
* Implement a notification based watcher [#1439](https://github.com/GoogleContainerTools/skaffold/pull/1439)
* Add k8s version check to kustomize deployer [#1449](https://github.com/GoogleContainerTools/skaffold/pull/1449)
* Support new K8s context name in Docker Desktop [#1448](https://github.com/GoogleContainerTools/skaffold/pull/1448)
* Upload sources for any kind of artifact [#1477](https://github.com/GoogleContainerTools/skaffold/pull/1477)
* feat(docker creds) can mount docker config into kaniko pod [#1466](https://github.com/GoogleContainerTools/skaffold/pull/1466)
* Support Jib on Google Cloud Build [#1478](https://github.com/GoogleContainerTools/skaffold/pull/1478)


Fixes:
* fix search URL for skaffold.dev + github edit link [#1417](https://github.com/GoogleContainerTools/skaffold/pull/1417)
* Print error messages when containers can’t be started [#1415](https://github.com/GoogleContainerTools/skaffold/pull/1415)
* Script should be executable [#1423](https://github.com/GoogleContainerTools/skaffold/pull/1423)
* Fix port-forwarding not being triggered. [#1433](https://github.com/GoogleContainerTools/skaffold/pull/1433)
* Fix localDir context for Kaniko on Windows [#1438](https://github.com/GoogleContainerTools/skaffold/pull/1438)
* Remove spurious warning [#1442](https://github.com/GoogleContainerTools/skaffold/pull/1442)
* Test what was actually deployed [#1462](https://github.com/GoogleContainerTools/skaffold/pull/1462)
* Fix jib tagging [#1475](https://github.com/GoogleContainerTools/skaffold/pull/1475)


Updates & refactorings:
* Move trigger related code to the watcher [#1422](https://github.com/GoogleContainerTools/skaffold/pull/1422)
* Simplify fake docker api [#1424](https://github.com/GoogleContainerTools/skaffold/pull/1424)
* Small improvements gcb [#1425](https://github.com/GoogleContainerTools/skaffold/pull/1425)
* Small improvements to kaniko builder [#1426](https://github.com/GoogleContainerTools/skaffold/pull/1426)
* Update golangci lint [#1430](https://github.com/GoogleContainerTools/skaffold/pull/1430)
* Refactor docker api [#1429](https://github.com/GoogleContainerTools/skaffold/pull/1429)
* Use latest release of Jib [#1440](https://github.com/GoogleContainerTools/skaffold/pull/1440)
* Refactor FakeCmd [#1456](https://github.com/GoogleContainerTools/skaffold/pull/1456)
* Use cmd.Run() indirection [#1457](https://github.com/GoogleContainerTools/skaffold/pull/1457)
* Clear error message for unsupported artifact on GCB [#1453](https://github.com/GoogleContainerTools/skaffold/pull/1453)
* Improve port-forwarding [#1452](https://github.com/GoogleContainerTools/skaffold/pull/1452)
* Minor changes to kaniko builder [#1461](https://github.com/GoogleContainerTools/skaffold/pull/1461)
* Show duplication in jib code [#1454](https://github.com/GoogleContainerTools/skaffold/pull/1454)
* Remove some duplication in Jib builder [#1465](https://github.com/GoogleContainerTools/skaffold/pull/1465)
* Use Maven wrapper for Jib example easier start. [#1471](https://github.com/GoogleContainerTools/skaffold/pull/1471)
* Simplify docker.AddTag() [#1464](https://github.com/GoogleContainerTools/skaffold/pull/1464)
* Embed labelling into Deployers [#1463](https://github.com/GoogleContainerTools/skaffold/pull/1463)
* Refactor port forwarding [#1474](https://github.com/GoogleContainerTools/skaffold/pull/1474)


Docs updates:
* CLI reference docs automation [#1418](https://github.com/GoogleContainerTools/skaffold/pull/1418)
* installation link to readme [#1437](https://github.com/GoogleContainerTools/skaffold/pull/1437)
* docs: typo + add setValueTemplates usecase [#1450](https://github.com/GoogleContainerTools/
* fix(docs) updated references for imageName to be image [#1468](https://github.com/GoogleContainerTools/skaffold/pull/1468)
* More fixes to the builders doc [#1469](https://github.com/GoogleContainerTools/skaffold/pull/1469)
* fix: correct spelling of Kaninko to Kaniko [#1472](https://github.com/GoogleContainerTools/skaffold/pull/1472)

Huge thank you for this release towards our contributors:

- Balint Pato
- Bruno Miguel Custodio
- Cedric Kring
- David Gageot
- Gareth Evans
- George Oakling
- Ivan Portyankin
- Lionel Touati
- Matt Rickard
- Matti Paksula
- Nick Kubala
- Priya Wadhwa


# v0.20.0 Release - 12/21/2018

*Note*: This release comes with a new config version `v1beta2`.
        To upgrade your `skaffold.yaml`, use `skaffold fix`. If you don't upgrade, skaffold will auto-upgrade in memory as best it can, and print a warning message.
        See [Skaffold Deprecation Policy](http://skaffold.dev/docs/references/deprecation/) for details on what beta means.

New Features:

* Add additional flags to kaniko builder [#1387](https://github.com/GoogleContainerTools/skaffold/pull/1387)

Fixes:

* Omit empty strings in jib sections of the config [#1399](https://github.com/GoogleContainerTools/skaffold/pull/1399)
* Don’t panic if image field is not of type string [#1386](https://github.com/GoogleContainerTools/skaffold/pull/1386)
* Fix Windows to Linux file sync by always converting path separators to *nix style [#1351](https://github.com/GoogleContainerTools/skaffold/pull/1351)
* Support labeling with hardcoded namespace [#1359](https://github.com/GoogleContainerTools/skaffold/pull/1359)
* Image name are case sensitive [#1342](https://github.com/GoogleContainerTools/skaffold/pull/1342)
* Print logs for containers that are not ready [#1344](https://github.com/GoogleContainerTools/skaffold/pull/1344)
* Cleanup only if something was actually deployed [#1343](https://github.com/GoogleContainerTools/skaffold/pull/1343)
* Don’t assume bazel-bin is symlinked in workspace [#1340](https://github.com/GoogleContainerTools/skaffold/pull/1340)


Updates & refactorings:

* Cleanup tagger tests [#1375](https://github.com/GoogleContainerTools/skaffold/pull/1375)
* Local builders return a digest [#1374](https://github.com/GoogleContainerTools/skaffold/pull/1374)
* Remove kqueue tag [#1402](https://github.com/GoogleContainerTools/skaffold/pull/1402)
* Improve runner unit tests [#1398](https://github.com/GoogleContainerTools/skaffold/pull/1398)
* Create directory before kubectl cp [#1390](https://github.com/GoogleContainerTools/skaffold/pull/1390)
* Add missing fake k8s context [#1384](https://github.com/GoogleContainerTools/skaffold/pull/1384)
* Improve schema upgrade tests [#1383](https://github.com/GoogleContainerTools/skaffold/pull/1383)
* Update kaniko image to latest version [#1381](https://github.com/GoogleContainerTools/skaffold/pull/1381)
* Introduce config version v1beta2 [#1376](https://github.com/GoogleContainerTools/skaffold/pull/1376)
* Tag image by digest [#1367](https://github.com/GoogleContainerTools/skaffold/pull/1367)
* Pass tag options by value [#1372](https://github.com/GoogleContainerTools/skaffold/pull/1372)
* Extract push/no-push logic into builder [#1366](https://github.com/GoogleContainerTools/skaffold/pull/1366)
* keeping integration test only examples under integration tests [#1362](https://github.com/GoogleContainerTools/skaffold/pull/1362)
* Display usage tips to the user [#1361](https://github.com/GoogleContainerTools/skaffold/pull/1361)
* Handle errors in release walking [#1356](https://github.com/GoogleContainerTools/skaffold/pull/1356)

Docs updates:

* new skaffold site [#1338](https://github.com/GoogleContainerTools/skaffold/pull/1338)

Utilities:

* If webhook deployment fails, upload logs [#1348](https://github.com/GoogleContainerTools/skaffold/pull/1348)

Huge thank you for this release towards our contributors:

- Balint Pato
- David Gageot
- Gareth Evans
- Matt Rickard
- Nick Kubala
- Priya Wadhwa
- Travis Cline
- Valery Vitko


# v0.19.0 Release - 11/29/2018

*Note*: This release comes with a new config version `v1beta1`.
        To upgrade your `skaffold.yaml`, use `skaffold fix`. If you don't upgrade, skaffold will auto-upgrade in memory as best it can, and print a warning message.
        See [Skaffold Deprecation Policy](http://skaffold.dev/docs/references/deprecation/) for details on what beta means.


New features:

* Run tests in skaffold build, add `skip-tests` flag to skip tests [#1326](https://github.com/GoogleContainerTools/skaffold/pull/1326)
* Allow ** glob pattern in sync parameters [#1266](https://github.com/GoogleContainerTools/skaffold/pull/1266)
* Add caching to kaniko builder [#1287](https://github.com/GoogleContainerTools/skaffold/pull/1287)
* Support slashes in file sync glob patterns on windows [#1280](https://github.com/GoogleContainerTools/skaffold/pull/1280)
* Add --compose-file option to skaffold init [#1282](https://github.com/GoogleContainerTools/skaffold/pull/1282)
* Automatically fix old configs by default [#1259](https://github.com/GoogleContainerTools/skaffold/pull/1259)
* adding skaffold version to the docker push user agent [#1260](https://github.com/GoogleContainerTools/skaffold/pull/1260)

Fixes:

* Fix node security issue [#1323](https://github.com/GoogleContainerTools/skaffold/pull/1323)
* Allow passing arguments to bazel build [#1289](https://github.com/GoogleContainerTools/skaffold/pull/1289)
* Get tmp Directory from os env in kaniko local context storing [#1285](https://github.com/GoogleContainerTools/skaffold/pull/1285)


Updates & refactorings:

* Apply default values upgraded configurations [#1332](https://github.com/GoogleContainerTools/skaffold/pull/1332)
* Remove duplication between run and deploy [#1331](https://github.com/GoogleContainerTools/skaffold/pull/1331)
* Remove pointer to runtime.Object interface [#1329](https://github.com/GoogleContainerTools/skaffold/pull/1329)
* Shorter logs [#1335](https://github.com/GoogleContainerTools/skaffold/pull/1335)
* Update deps [#1333](https://github.com/GoogleContainerTools/skaffold/pull/1333)
* dep ensure && dep prune [#1297](https://github.com/GoogleContainerTools/skaffold/pull/1297)
* Should support v1alpha5 [#1314](https://github.com/GoogleContainerTools/skaffold/pull/1314)
* Improve kubernetes.Logger [#1309](https://github.com/GoogleContainerTools/skaffold/pull/1309)
* introduce v1beta1 config [#1305](https://github.com/GoogleContainerTools/skaffold/pull/1305)
* Simpler Runner [#1304](https://github.com/GoogleContainerTools/skaffold/pull/1304)
* Don’t run tests if nothing was built [#1302](https://github.com/GoogleContainerTools/skaffold/pull/1302)
* Simplify the Runner's tests [#1303](https://github.com/GoogleContainerTools/skaffold/pull/1303)
* removing the artifacts from appveyor [#1300](https://github.com/GoogleContainerTools/skaffold/pull/1300)
* The multi-deployer feature is not working. Remove it [#1291](https://github.com/GoogleContainerTools/skaffold/pull/1291)

Breaking changes:

* Remove ACR builder [#1308](https://github.com/GoogleContainerTools/skaffold/pull/1308)
* Remove `quiet` command line flag [#1292](https://github.com/GoogleContainerTools/skaffold/pull/1292)

Docs updates:

* Clarify what manifest paths are relative to when specifying in skaffold yaml [#1336](https://github.com/GoogleContainerTools/skaffold/pull/1336)
* adding deprecation policy and document component stability [#1324](https://github.com/GoogleContainerTools/skaffold/pull/1324)
* Add missing fields to annotated-skaffold.yaml [#1310](https://github.com/GoogleContainerTools/skaffold/pull/1310)
* brew install skaffold [#1290](https://github.com/GoogleContainerTools/skaffold/pull/1290)
* Lists indented in the installation section (minor fix) [#1298](https://github.com/GoogleContainerTools/skaffold/pull/1298)
* Make usage messages look like the others. [#1267](https://github.com/GoogleContainerTools/skaffold/pull/1267)

Utilities:

* [docs-webhook] remove docs-modifications label from issue instead of deleting the label [#1306](https://github.com/GoogleContainerTools/skaffold/pull/1306)
* [docs-webhook] hugo extended version + nodejs  [#1295](https://github.com/GoogleContainerTools/skaffold/pull/1295)
* [docs-webhook] Release latest version of docs controller image [#1293](https://github.com/GoogleContainerTools/skaffold/pull/1293)
* [docs-webhook] upgrading hugo + unpinning webhook image [#1288](https://github.com/GoogleContainerTools/skaffold/pull/1288)
* [lint] Golangci lint upgrade [#1281](https://github.com/GoogleContainerTools/skaffold/pull/1281)
* [compilation] Support system's LDFLAGS, make compilation reproducible [#1270](https://github.com/GoogleContainerTools/skaffold/pull/1270)

Huge thank you for this release towards our contributors:
- Balint Pato
- Cedric Vidal
- David Gageot
- Igor Zibarev
- Ihor Dvoretskyi
- Jamie Lennox
- Maxim Baz
- Nick Kubala
- Pascal Ehlert
- Priya Wadhwa
- Venkatesh


# v0.18.0 Release - 11/08/2018

Bug Fixes:

* Don't lose test configuration when running skaffold fix [#1251](https://github.com/GoogleContainerTools/skaffold/pull/1251)
* Fix jib errors on ctrl-c [#1248](https://github.com/GoogleContainerTools/skaffold/pull/1248)
* Fix sync [#1253](https://github.com/GoogleContainerTools/skaffold/pull/1253)
* Update examples and release notes to use v1alpha5 [#1244](https://github.com/GoogleContainerTools/skaffold/pull/1244)
* Set Kind on `skaffold init` [#1237](https://github.com/GoogleContainerTools/skaffold/pull/1237)
* Do not print the manifest on to stdout when doing a deploy by kustomize [#1234](https://github.com/GoogleContainerTools/skaffold/pull/1234)
* Fixed panic if skaffold.yaml is empty (#1216) [#1221](https://github.com/GoogleContainerTools/skaffold/pull/1221)
* Suppress fatal error reporting when ^C skaffold with jib [#1228](https://github.com/GoogleContainerTools/skaffold/pull/1228)
* portforward for resources with hardcoded namespace [#1223](https://github.com/GoogleContainerTools/skaffold/pull/1223)

Updates:

* Output config version in skaffold version [#1252](https://github.com/GoogleContainerTools/skaffold/pull/1252)
* Port forward multiple ports [#1250](https://github.com/GoogleContainerTools/skaffold/pull/1250)
* Improve errors [#1255](https://github.com/GoogleContainerTools/skaffold/pull/1255)
* Move structure tests out of getting-started example [#1220](https://github.com/GoogleContainerTools/skaffold/pull/1220)
* changes related to our docs review flow:
  * Add github pkg to webhook [#1230](https://github.com/GoogleContainerTools/skaffold/pull/1230)
  * Allow webhook to create a deployment [#1227](https://github.com/GoogleContainerTools/skaffold/pull/1227)
  * Add hugo and git to webhook image [#1226](https://github.com/GoogleContainerTools/skaffold/pull/1226)
  * Add support for creating a service from webhook [#1213](https://github.com/GoogleContainerTools/skaffold/pull/1213)

Huge thank you for this release towards our contributors:
- Balint Pato
- Brian de Alwis
- David Gageot
- Matt Rickard
- Nick Kubala
- Priya Wadhwa
- RaviTezu
- varunkashyap

# v0.17.0 Release - 10/26/2018

Note: This release comes with a config change, use `skaffold fix` to upgrade your config to `v1alpha5`.
We 'skipped' `v1alpha4` due to an accidental merge: see [#1235](https://github.com/GoogleContainerTools/skaffold/issues/1235#issuecomment-436429009)

New Features:

* Add support for setting default-repo in global config [#1057](https://github.com/GoogleContainerTools/skaffold/pull/1057)
* Add support for building Maven multimodule projects [#1152](https://github.com/GoogleContainerTools/skaffold/pull/1152)
* Azure Container Registry runner [#1107](https://github.com/GoogleContainerTools/skaffold/pull/1107)

Bug fixes:

* Improve Kaniko builder [#1168](https://github.com/GoogleContainerTools/skaffold/pull/1168)
* Use os.SameFile() to check for mvnw working-dir echo bug [#1167](https://github.com/GoogleContainerTools/skaffold/pull/1167)
* Fix kaniko default behavior [#1139](https://github.com/GoogleContainerTools/skaffold/pull/1139)

Updates:

* Change SkaffoldOption Labeller to not include a comma in the label value [#1169](https://github.com/GoogleContainerTools/skaffold/pull/1169)
* Remove annoying log [#1163](https://github.com/GoogleContainerTools/skaffold/pull/1163)
* Prepare next version of the config file [#1146](https://github.com/GoogleContainerTools/skaffold/pull/1146)
* Improve error handling for `completion` command [#1206](https://github.com/GoogleContainerTools/skaffold/pull/1206)
* Jib sample [#1147](https://github.com/GoogleContainerTools/skaffold/pull/1147)
* Node.js example with dependency handling and hot-reload [#1148](https://github.com/GoogleContainerTools/skaffold/pull/1148)

Huge thank you for this release towards our contributors:
- Balint Pato
- Brian de Alwis
- Cedric Kring
- David Gageot
- Geert-Johan Riemer
- Martino Fornasa
- Matt Rickard
- Nick Kubala
- Priya Wadhwa
- foo0x29a
- varunkashyap


# v0.16.0 Release - 10/11/2018

New Features:
* Add a `skaffold diagnose` command [#1109](https://github.com/GoogleContainerTools/skaffold/pull/1109)
* Add localdir buildcontext to kaniko builder [#983](https://github.com/GoogleContainerTools/skaffold/pull/983)
* Add --label flag to specify custom labels for deployments [#1098](https://github.com/GoogleContainerTools/skaffold/pull/1098)
* Add support for building projects using jib [#1073](https://github.com/GoogleContainerTools/skaffold/pull/1073)

Bug Fixes:
* Fix jib artifacts in skaffold diagnose [#1141](https://github.com/GoogleContainerTools/skaffold/pull/1141)
* Fix master [#1133](https://github.com/GoogleContainerTools/skaffold/pull/1133)
* Delete and redeploy object upon error 'field is immutable' [#940](https://github.com/GoogleContainerTools/skaffold/pull/940)
* Fix `skaffold fix` [#1123](https://github.com/GoogleContainerTools/skaffold/pull/1123)
* Lists files recursively in jib.getDependencies and other fixes. [#1097](https://github.com/GoogleContainerTools/skaffold/pull/1097)
* Merge error? [#1095](https://github.com/GoogleContainerTools/skaffold/pull/1095)
* Fix missing parenthesis [#1089](https://github.com/GoogleContainerTools/skaffold/pull/1089)

Updates:
* Move sync code to pkg/skaffold/sync/kubectl [#1138](https://github.com/GoogleContainerTools/skaffold/pull/1138)
* Add a test to check version upgrades [#1103](https://github.com/GoogleContainerTools/skaffold/pull/1103)
* Add a way to unset global config values [#1086](https://github.com/GoogleContainerTools/skaffold/pull/1086)
* Handles Jib build directly to registry when push=true. [#1132](https://github.com/GoogleContainerTools/skaffold/pull/1132)
* Simplify Jib code [#1130](https://github.com/GoogleContainerTools/skaffold/pull/1130)
* Trim the dockerfile a bit: [#1128](https://github.com/GoogleContainerTools/skaffold/pull/1128)
* Pass context when listing dependencies [#1108](https://github.com/GoogleContainerTools/skaffold/pull/1108)
* Remove fully qualified jib path for maven goals [#1129](https://github.com/GoogleContainerTools/skaffold/pull/1129)
* Merge master into jib_skaffold feature branch. [#1117](https://github.com/GoogleContainerTools/skaffold/pull/1117)
* Merge Jib feature-branch [#1063](https://github.com/GoogleContainerTools/skaffold/pull/1063)
* Improves jib.getDependencies. [#1125](https://github.com/GoogleContainerTools/skaffold/pull/1125)
* skipPush -> push [#1114](https://github.com/GoogleContainerTools/skaffold/pull/1114)
* Support for dot files in dockerignore [#1122](https://github.com/GoogleContainerTools/skaffold/pull/1122)
* remove project level skaffold.yaml [#1119](https://github.com/GoogleContainerTools/skaffold/pull/1119)
* Merge master into jib_skaffold feature branch [#1116](https://github.com/GoogleContainerTools/skaffold/pull/1116)
* Unify Jib command wrapper usage [#1105](https://github.com/GoogleContainerTools/skaffold/pull/1105)
* Update labels when deploying to namespace other than default [#1115](https://github.com/GoogleContainerTools/skaffold/pull/1115)
* Improve sync [#1102](https://github.com/GoogleContainerTools/skaffold/pull/1102)
* Rename SkaffoldConfig to SkaffoldPipeline [#1087](https://github.com/GoogleContainerTools/skaffold/pull/1087)
* Kaniko improvements [#1101](https://github.com/GoogleContainerTools/skaffold/pull/1101)
* File Sync for skaffold dev [#1039](https://github.com/GoogleContainerTools/skaffold/pull/1039)
* Implement a manual trigger for watch mode [#1085](https://github.com/GoogleContainerTools/skaffold/pull/1085)
* Skaffold init asks user to write skaffold.yaml [#1091](https://github.com/GoogleContainerTools/skaffold/pull/1091)
* Couple of improvements to the test phase [#1080](https://github.com/GoogleContainerTools/skaffold/pull/1080)
* Merges branch 'master' into jib_skaffold. [#1088](https://github.com/GoogleContainerTools/skaffold/pull/1088)
* Implements jib.GetDependenciesMaven/GetDependenciesGradle. [#1058](https://github.com/GoogleContainerTools/skaffold/pull/1058)
* Add test runner [#1013](https://github.com/GoogleContainerTools/skaffold/pull/1013)
* Simplify schema versioning [#1067](https://github.com/GoogleContainerTools/skaffold/pull/1067)
* Changelog changes for v0.15.1 [#1075](https://github.com/GoogleContainerTools/skaffold/pull/1075)
* Minor logging improvements [#1142](https://github.com/GoogleContainerTools/skaffold/pull/1142)


# v0.15.1 Release - 10/02/2018

This is a minor release to address an inconsistency in the `skaffold fix` upgrade:

* Transform values files in profiles to v1alpha3 [#1070](https://github.com/GoogleContainerTools/skaffold/pull/1070)


# v0.15.0 Release - 9/27/2018

New Features:
* Added kustomize to deploy types [#1027](https://github.com/GoogleContainerTools/skaffold/pull/1027)
* Basic support for watching Kustomize dependencies [#1015](https://github.com/GoogleContainerTools/skaffold/pull/1015)
* Basic support for using kubectl and helm together [#586](https://github.com/GoogleContainerTools/skaffold/pull/586)
* Add support for multiple helm values files [#985](https://github.com/GoogleContainerTools/skaffold/pull/985)
* Add v1alpha3 Config [#982](https://github.com/GoogleContainerTools/skaffold/pull/982)

Bug Fixes:
* annotated.yaml: fix gcb timeout format [#1040](https://github.com/GoogleContainerTools/skaffold/pull/1040)
* Catch a 409 when creating a bucket and continue. [#1044](https://github.com/GoogleContainerTools/skaffold/pull/1044)
* Fix typo [#1045](https://github.com/GoogleContainerTools/skaffold/pull/1045)
* Fix issues with build args replacement [#1028](https://github.com/GoogleContainerTools/skaffold/pull/1028)
* prevent watcher failure if helm valuesFilePath not set [#930](https://github.com/GoogleContainerTools/skaffold/pull/930)
* Correctly parse build tags that contain port numbers [#1001](https://github.com/GoogleContainerTools/skaffold/pull/1001)
* FIX kubectl should only redeploy updated manifests [#1014](https://github.com/GoogleContainerTools/skaffold/pull/1014)
* Fix race conditions in TestWatch [#987](https://github.com/GoogleContainerTools/skaffold/pull/987)

Updates:
* Simpler merged PR collection for release notes [#1054](https://github.com/GoogleContainerTools/skaffold/pull/1054)
* Improve kustomize deployer [#1036](https://github.com/GoogleContainerTools/skaffold/pull/1036)
* kustomizePath is a folder that defaults to . [#1030](https://github.com/GoogleContainerTools/skaffold/pull/1030)
* Discard output in tests [#1021](https://github.com/GoogleContainerTools/skaffold/pull/1021)
* Add a test for `kubectl should only redeploy updated manifests` [#1022](https://github.com/GoogleContainerTools/skaffold/pull/1022)
* Examples versioning [#1019](https://github.com/GoogleContainerTools/skaffold/pull/1019)
* add nkubala to MAINTAINERS [#993](https://github.com/GoogleContainerTools/skaffold/pull/993)
* Debounce rapid file changes [#1005](https://github.com/GoogleContainerTools/skaffold/pull/1005)
* Print kubectl client version [#991](https://github.com/GoogleContainerTools/skaffold/pull/991)
* Auto configure authentication helper for gcr.io [#989](https://github.com/GoogleContainerTools/skaffold/pull/989)
* Tweak the Dockerfile. [#1007](https://github.com/GoogleContainerTools/skaffold/pull/1007)
* Skip kaniko-related test when running locally [#990](https://github.com/GoogleContainerTools/skaffold/pull/990)
* Extract code from GCB [#986](https://github.com/GoogleContainerTools/skaffold/pull/986)


# v0.14.0 Release - 9/13/2018

New Features:
* Allow `skaffold dev —watch image` [#925](https://github.com/GoogleContainerTools/skaffold/pull/925)
* Port forward pods automatically during `skaffold dev` [#945](https://github.com/GoogleContainerTools/skaffold/pull/945)
* Add skaffold 'init' [#919](https://github.com/GoogleContainerTools/skaffold/pull/919)

Bug Fixes:
* Get namespace for updating objects from build artifact [#951](https://github.com/GoogleContainerTools/skaffold/pull/951)
* Remove service labeling temporarily [#965](https://github.com/GoogleContainerTools/skaffold/pull/965)
* Don't prefix pod names when port forwarding [#976](https://github.com/GoogleContainerTools/skaffold/pull/976)

Updates:
* Don’t compute onbuild triggers for images that are stage names [#938](https://github.com/GoogleContainerTools/skaffold/pull/938)
* Don't unmute logs if an error happened [#928](https://github.com/GoogleContainerTools/skaffold/pull/928)
* Exclude helm dependency chart packages from watched files [#932](https://github.com/GoogleContainerTools/skaffold/pull/932)
* Pass --recreate-pods to helm by default in dev mode [#946](https://github.com/GoogleContainerTools/skaffold/pull/946)
* Default to kubectl deploy [#956](https://github.com/GoogleContainerTools/skaffold/pull/956)
* Simplify helm tests [#957](https://github.com/GoogleContainerTools/skaffold/pull/957)
* Pull 'cache-from' images on Google Cloud Build [#958](https://github.com/GoogleContainerTools/skaffold/pull/958)
* update check respected quiet flag [#964](https://github.com/GoogleContainerTools/skaffold/pull/964)
* Fix typo in portforwarder [#975](https://github.com/GoogleContainerTools/skaffold/pull/975)


# v0.13.0 Release - 8/16/2018

New Features:
* Add --tail flag to stream logs with skaffold run [#914](https://github.com/GoogleContainerTools/skaffold/pull/914)
* Add DEVELOPMENT.md [#901](https://github.com/GoogleContainerTools/skaffold/pull/901)

Bug Fixes:
* fixes `skaffold version` in the released docker image [#933](https://github.com/GoogleContainerTools/skaffold/pull/933)

Updates:
* as a base for future features - global skaffold config [#896](https://github.com/GoogleContainerTools/skaffold/pull/896)
* Remove duplication in kustomize deployer [#900](https://github.com/GoogleContainerTools/skaffold/pull/900)
* update readme with documentation links [#908](https://github.com/GoogleContainerTools/skaffold/pull/908)
* Fix a typo in "annotated-skaffold.yaml" [#907](https://github.com/GoogleContainerTools/skaffold/pull/907)
* Decouple visiting manifests and replacing images [#909](https://github.com/GoogleContainerTools/skaffold/pull/909)
* Add a simple test for Watcher [#898](https://github.com/GoogleContainerTools/skaffold/pull/898)
* Add test for signal handling [#917](https://github.com/GoogleContainerTools/skaffold/pull/917)
* Add the --target flag as a parameter to the docker builder. [#894](https://github.com/GoogleContainerTools/skaffold/pull/894)
* Misc improvements [#911](https://github.com/GoogleContainerTools/skaffold/pull/911)
* Add --tail flag to stream logs with skaffold run [#914](https://github.com/GoogleContainerTools/skaffold/pull/914)
* Extract code to tail logs [#924](https://github.com/GoogleContainerTools/skaffold/pull/924)
* Improve logs [#918](https://github.com/GoogleContainerTools/skaffold/pull/918)
* Add yamltags [#388](https://github.com/GoogleContainerTools/skaffold/pull/388)
* adding wrapper script for release note generation  [#935](https://github.com/GoogleContainerTools/skaffold/pull/935)
* detete -> delete [#941](https://github.com/GoogleContainerTools/skaffold/pull/941)


# v0.12.0 Release - 8/16/2018
New Features:
* Update check [#866](https://github.com/GoogleContainerTools/skaffold/pull/866)
* Simpler and faster git tagger [#846](https://github.com/GoogleContainerTools/skaffold/pull/846)
* Support setting namespace for every deployer [#852](https://github.com/GoogleContainerTools/skaffold/pull/852)
* Improve Cloud Build builder [#874](https://github.com/GoogleContainerTools/skaffold/pull/874)
* Improve file change tracking [#888](https://github.com/GoogleContainerTools/skaffold/pull/888)


Bug Fixes:
* Run Kaniko builds in parallel [#876](https://github.com/GoogleContainerTools/skaffold/pull/876)
* Do not run kubectl if nothing has changed [#877](https://github.com/GoogleContainerTools/skaffold/pull/877)
* fix version in released docker image [#878](https://github.com/GoogleContainerTools/skaffold/pull/878)
* Fix integration tests [#881](https://github.com/GoogleContainerTools/skaffold/pull/881)

Updates:
* Run Kaniko builds in parallel [#876](https://github.com/GoogleContainerTools/skaffold/pull/876)
* Watch mode 4th edition [#833](https://github.com/GoogleContainerTools/skaffold/pull/833)
* add bazel to skaffold docker image, add integration test for bazel [#879](https://github.com/GoogleContainerTools/skaffold/pull/879)
* Add missing filename to error message [#880](https://github.com/GoogleContainerTools/skaffold/pull/880)
* Fix minor lint errors surfaced by the 'misspell' and 'unparam' lint modules [#883](https://github.com/GoogleContainerTools/skaffold/pull/883)
* Update golangci-lint to v1.9.3 and enable misspell+unparam modules [#884](https://github.com/GoogleContainerTools/skaffold/pull/884)
* add codecov to travis and repo [#885](https://github.com/GoogleContainerTools/skaffold/pull/885)
* Add test helper to handle actions on tmp dirs [#893](https://github.com/GoogleContainerTools/skaffold/pull/893)
* Use reflection to overlay profile onto config [#872](https://github.com/GoogleContainerTools/skaffold/pull/872)


# v0.11.0 Release - 8/02/2018
New Features:
* Pass buildArgs to Kaniko [#822](https://github.com/GoogleContainerTools/skaffold/pull/822)
* Add pop of color to terminal output with a color formatter [#857](https://github.com/GoogleContainerTools/skaffold/pull/857)

Bug Fixes:
* Substitute build args from config into parsed Dockerfile before processing deps [#828](https://github.com/GoogleContainerTools/skaffold/pull/828)
* Fix color.Fprintln bug [#861](https://github.com/GoogleContainerTools/skaffold/pull/861)
* Issue #836: Use releaseName to get release info. [#855](https://github.com/GoogleContainerTools/skaffold/pull/855)
* Switch to gcr for the kaniko builder example. [#845](https://github.com/GoogleContainerTools/skaffold/pull/845)

Updates:
* boilerplate.sh: fail if python script not found; run from any dir [#827](https://github.com/GoogleContainerTools/skaffold/pull/827)
* Revert to default grace period [#815](https://github.com/GoogleContainerTools/skaffold/pull/815)
* Skip the deployment if no manifests are defined [#832](https://github.com/GoogleContainerTools/skaffold/pull/832)
* Slightly faster git tagger [#839](https://github.com/GoogleContainerTools/skaffold/pull/839)
* Don’t tag the same images twice [#842](https://github.com/GoogleContainerTools/skaffold/pull/842)
* Faster code to get image digest [#838](https://github.com/GoogleContainerTools/skaffold/pull/838)
* Simpler code to print Kaniko logs [#831](https://github.com/GoogleContainerTools/skaffold/pull/831)
* Simpler sha256 tagger code [#847](https://github.com/GoogleContainerTools/skaffold/pull/847)
* Move builders to sub packages [#830](https://github.com/GoogleContainerTools/skaffold/pull/830)
* Shell out docker build [#840](https://github.com/GoogleContainerTools/skaffold/pull/840)
* Don’t redeploy twice the same manifest in a dev loop [#843](https://github.com/GoogleContainerTools/skaffold/pull/843)
* Remove `skaffold docker` commands [#853](https://github.com/GoogleContainerTools/skaffold/pull/853)
* Find docker deps 10x faster [#837](https://github.com/GoogleContainerTools/skaffold/pull/837)
* Simplify docker related code. [#854](https://github.com/GoogleContainerTools/skaffold/pull/854)
* add support for helm image convention vs fqn setting [#826](https://github.com/GoogleContainerTools/skaffold/pull/826)
* Update dep to v0.5.0 [#862](https://github.com/GoogleContainerTools/skaffold/pull/862)


# v0.10.0 Release - 7/13/2018
New Features:
* kustomize: use custom path in deploy deps [#766](https://github.com/GoogleContainerTools/skaffold/pull/766)
* helm: add deploy dependency paths [#765](https://github.com/GoogleContainerTools/skaffold/pull/765)
* Use digest when the git repo has no commit [#794](https://github.com/GoogleContainerTools/skaffold/pull/794)
* GCB now builds artifacts in // [#805](https://github.com/GoogleContainerTools/skaffold/pull/805)
* Default kubectl manifests to `k8s/*.yaml` [#810](https://github.com/GoogleContainerTools/skaffold/pull/810)
* Support disk size and machine type for GCB [#808](https://github.com/GoogleContainerTools/skaffold/pull/808)
* Support additional flags for kubectl commands [#807](https://github.com/GoogleContainerTools/skaffold/pull/807)
* Try to guess GCB projectID from the image name [#809](https://github.com/GoogleContainerTools/skaffold/pull/809)

Bug Fixes:
* kustomize: cleanup custom kustomize path [#781](https://github.com/GoogleContainerTools/skaffold/pull/781)
* corrected region typo [#792](https://github.com/GoogleContainerTools/skaffold/pull/792)
* Fixed a small typo in docs [#797](https://github.com/GoogleContainerTools/skaffold/pull/797)
* Small code changes [#796](https://github.com/GoogleContainerTools/skaffold/pull/796)

Updates:
* docs: alphabetize readme peoples [#764](https://github.com/GoogleContainerTools/skaffold/pull/764)
* makefile: redirection for checksums [#768](https://github.com/GoogleContainerTools/skaffold/pull/768)
* brew: remove version from formula [#763](https://github.com/GoogleContainerTools/skaffold/pull/763)
* Add the logo [#774](https://github.com/GoogleContainerTools/skaffold/pull/774)
* ci: also push latest skaffold image on commit [#773](https://github.com/GoogleContainerTools/skaffold/pull/773)
* tests: pin golangci-lint version to v1.8.1 [#780](https://github.com/GoogleContainerTools/skaffold/pull/780)
* Remove dead code [#784](https://github.com/GoogleContainerTools/skaffold/pull/784)
* Improve GCR docs [#795](https://github.com/GoogleContainerTools/skaffold/pull/795)
* Extract code to build a single artifact locally [#798](https://github.com/GoogleContainerTools/skaffold/pull/798)
* Use dynamic client for labels [#782](https://github.com/GoogleContainerTools/skaffold/pull/782)
* Update Kaniko to v0.2.0 [#803](https://github.com/GoogleContainerTools/skaffold/pull/803)
* Upgrade k8s dependency to 1.11.0 [#804](https://github.com/GoogleContainerTools/skaffold/pull/804)
* Fix missing logs [#786](https://github.com/GoogleContainerTools/skaffold/pull/786)
* calculate version from git [#814](https://github.com/GoogleContainerTools/skaffold/pull/814)
* logs: use namespace flag when streaming pods [#819](https://github.com/GoogleContainerTools/skaffold/pull/819)

# v0.9.0 Release - 6/28/2018
New Features:
* Print the image name that's being built [#732](https://github.com/GoogleContainerTools/skaffold/pull/732)
* Publish windows binaries on AppVeyor [#738](https://github.com/GoogleContainerTools/skaffold/pull/738)
* Add labeling for profiles [#736](https://github.com/GoogleContainerTools/skaffold/pull/736)
* Improve Git tagger [#714](https://github.com/GoogleContainerTools/skaffold/pull/714)
* Support docker build --cache-from [#737](https://github.com/GoogleContainerTools/skaffold/pull/737)
* Add custom kustomization path [#749](https://github.com/GoogleContainerTools/skaffold/pull/749)
* Use tags only in case of perfect match [#755](https://github.com/GoogleContainerTools/skaffold/pull/755)

Bug Fixes:
* fixed a bug in dirtyTag which may leave extra whitespaces in changedPath [#721](https://github.com/GoogleContainerTools/skaffold/pull/721)
* Remove duplication in code handling labels [#723](https://github.com/GoogleContainerTools/skaffold/pull/723)
* Fix: Links for D4M Edge and D4W Edge were swapped [#735](https://github.com/GoogleContainerTools/skaffold/pull/735)
* Fix bug where dirty submodules broke hash generation [#711](https://github.com/GoogleContainerTools/skaffold/pull/711)
* Remove warning for an image that’s built and used by fqn [#713](https://github.com/GoogleContainerTools/skaffold/pull/713)
* Don’t always fail if some COPY patterns don't match any file [#744](https://github.com/GoogleContainerTools/skaffold/pull/744)
* Fix dev loop [#758](https://github.com/GoogleContainerTools/skaffold/pull/758)
* Fix kaniko defaults [#756](https://github.com/GoogleContainerTools/skaffold/pull/756)
* Don’t complain when object is not found during cleanup [#759](https://github.com/GoogleContainerTools/skaffold/pull/759)

Updates:
* Deployers should only rely on their specific config [#739](https://github.com/GoogleContainerTools/skaffold/pull/739)
* Builders should only rely on their specific config [#740](https://github.com/GoogleContainerTools/skaffold/pull/740)
* e2e test for helm deployments. [#743](https://github.com/GoogleContainerTools/skaffold/pull/743)
* New code to watch file changes [#620](https://github.com/GoogleContainerTools/skaffold/pull/620)
* docs: add info about published artifacts [#751](https://github.com/GoogleContainerTools/skaffold/pull/751)

# v0.8.0 Release - 06/21/2018

New Features
* cloudbuild: publish skaffold images on commit and tag [#655](https://github.com/GoogleContainerTools/skaffold/pull/655)
* Asciidocs and refdocs tooling [#648](https://github.com/GoogleContainerTools/skaffold/pull/648)
* Add support for skaffold.yml as a default config file fixes #225 [#665](https://github.com/GoogleContainerTools/skaffold/pull/665)
* adds helper script for release notes [#662](https://github.com/GoogleContainerTools/skaffold/pull/662)
* docs: add weekly meeting snippet [#675](https://github.com/GoogleContainerTools/skaffold/pull/675)
* Add labels to all k8s objects deployed by skaffold [#644](https://github.com/GoogleContainerTools/skaffold/pull/644)
* Implement packaging for helm deployment [#682](https://github.com/GoogleContainerTools/skaffold/pull/682)
* mv tagPolicy:env example [#697](https://github.com/GoogleContainerTools/skaffold/pull/697)
* windows: add appveyor [#702](https://github.com/GoogleContainerTools/skaffold/pull/702)
* add WSL support [#694](https://github.com/GoogleContainerTools/skaffold/pull/694)
* Add labels from options [#716](https://github.com/GoogleContainerTools/skaffold/pull/716)
* Add tests for helm deployment with `packaged' option [#696](https://github.com/GoogleContainerTools/skaffold/pull/696)
* Fix issue #404 - Allow to use bazel subtarget [#689](https://github.com/GoogleContainerTools/skaffold/pull/689)
* fix: allow environment variables to be used in helm values [#707](https://github.com/GoogleContainerTools/skaffold/pull/707)
* Improve Kaniko code and ns handling [#722](https://github.com/GoogleContainerTools/skaffold/pull/722)
* Support wildcards in Dockerfiles [#712](https://github.com/GoogleContainerTools/skaffold/pull/712)

Bug Fixes

* make: fix release path [#650](https://github.com/GoogleContainerTools/skaffold/pull/650)
* Fixing the licence [#652](https://github.com/GoogleContainerTools/skaffold/pull/652)
* typo fix [#660](https://github.com/GoogleContainerTools/skaffold/pull/660)
* Ignore missing authConfigs during docker build [#664](https://github.com/GoogleContainerTools/skaffold/pull/664)
* lint fixes [#669](https://github.com/GoogleContainerTools/skaffold/pull/669)
* Fix hack/dep.sh on travisCI [#680](https://github.com/GoogleContainerTools/skaffold/pull/680)
* Use git binary or fallback to go-git [#639](https://github.com/GoogleContainerTools/skaffold/pull/639)
* Fix git detection [#683](https://github.com/GoogleContainerTools/skaffold/pull/683)
* remove extraneous space [#688](https://github.com/GoogleContainerTools/skaffold/pull/688)
* Create and apply patch when adding labels to API objects [#687](https://github.com/GoogleContainerTools/skaffold/pull/687)
* Fix issue with 100% CPU usage in logs.go. [#704](https://github.com/GoogleContainerTools/skaffold/pull/704)

Updates

* Remove fsnotify [#646](https://github.com/GoogleContainerTools/skaffold/pull/646)
* Update go-containerregistry [#651](https://github.com/GoogleContainerTools/skaffold/pull/651)
* cloudbuild: increase timeout to 20m [#658](https://github.com/GoogleContainerTools/skaffold/pull/658)
* Update docker libraries [#676](https://github.com/GoogleContainerTools/skaffold/pull/676)
* Update apimachinery and client-go to kubernetes-1.11.0-beta2 [#684](https://github.com/GoogleContainerTools/skaffold/pull/684)
* Update release_notes.sh [#710](https://github.com/GoogleContainerTools/skaffold/pull/710)
* Remove unused imports [#724](https://github.com/GoogleContainerTools/skaffold/pull/724)


# v0.7.0 Release - 06/07/2018


New Features

* cmd: add skaffold deploy [#624](https://github.com/GoogleContainerTools/skaffold/pull/624)
* Remove no-manifest code. [#640](https://github.com/GoogleContainerTools/skaffold/pull/640)
* Add an mtime file watcher. [#549](https://github.com/GoogleContainerTools/skaffold/pull/549)
* Add functionality to toggle the `--wait` flag on helm install/upgrade [#633](https://github.com/GoogleContainerTools/skaffold/pull/633)
* Add kustomize deployer [#641](https://github.com/GoogleContainerTools/skaffold/pull/641)
* Add datetime tagger tagpolicy [#621](https://github.com/GoogleContainerTools/skaffold/pull/621)
* Helm: add option to generate override values.yaml based on data passed into skaffold.yaml [#632](https://github.com/GoogleContainerTools/skaffold/pull/632)
* add `--output` and `--quiet` to `skaffold build` [#606](https://github.com/GoogleContainerTools/skaffold/pull/606)
* Add the ability to express the release name as a template [#602](https://github.com/GoogleContainerTools/skaffold/pull/602)
* Simpler code that logs containers [#612](https://github.com/GoogleContainerTools/skaffold/pull/612)

Bug Fixes

* Fix image parsing in skaffold deploy [#638](https://github.com/GoogleContainerTools/skaffold/pull/638)
* Fix flaky test [#594](https://github.com/GoogleContainerTools/skaffold/pull/594)
* fix: allow an environment variable to default the deploy namespace [#497](https://github.com/GoogleContainerTools/skaffold/pull/497)
* Add BUILD and WORKSPACE files to dependencies [#636](https://github.com/GoogleContainerTools/skaffold/pull/636)
* Misc fixes to dev mode [#589](https://github.com/GoogleContainerTools/skaffold/pull/589)


Updates

* Quick Start GKE Doc - reference change for k8s-pod deployment [#615](https://github.com/GoogleContainerTools/skaffold/pull/615)
* kaniko: pin image version to v0.1.0 [#592](https://github.com/GoogleContainerTools/skaffold/pull/592)
* Refactor the envTemplate code to make it reusable [#601](https://github.com/GoogleContainerTools/skaffold/pull/601)
* Simplify runner test [#609](https://github.com/GoogleContainerTools/skaffold/pull/609)
* Move kubernetes client creation to kubernetes package [#608](https://github.com/GoogleContainerTools/skaffold/pull/608)
* Remove unused field. [#616](https://github.com/GoogleContainerTools/skaffold/pull/616)
* Remove annoying testdata folder [#614](https://github.com/GoogleContainerTools/skaffold/pull/614)
* Dockerfile should always be sent to daemon [#605](https://github.com/GoogleContainerTools/skaffold/pull/605)
* Simplify code that resolves dependencies [#610](https://github.com/GoogleContainerTools/skaffold/pull/610)
* Switch boilerplate to The Skaffold Authors. [#626](https://github.com/GoogleContainerTools/skaffold/pull/626)
* Improve runner code [#645](https://github.com/GoogleContainerTools/skaffold/pull/645)
* Simplify helm_test [#607](https://github.com/GoogleContainerTools/skaffold/pull/607)
* Replace gometalinter with GolangCI-Lint [#619](https://github.com/GoogleContainerTools/skaffold/pull/619)
* Update go-git to v4.4.0 [#634](https://github.com/GoogleContainerTools/skaffold/pull/634)
* Remove afero [#613](https://github.com/GoogleContainerTools/skaffold/pull/613)


https://github.com/GoogleContainerTools/skaffold/compare/v0.6.1...v0.7.0

# v0.6.1 Release - 5/24/2018
New Features
* Stricter YAML parsing [#570](https://github.com/GoogleContainerTools/skaffold/pull/570)
* Show helm's output and errors [#576](https://github.com/GoogleContainerTools/skaffold/pull/576)
* Support ~ in secret path for Kaniko [#455](https://github.com/GoogleContainerTools/skaffold/pull/455)
* `skaffold dev` now stops on non-build errors [#540](https://github.com/GoogleContainerTools/skaffold/pull/540)

Bug Fixes
* GCB Build fixed [#579](https://github.com/GoogleContainerTools/skaffold/pull/579)
* Show errors of kubectl and helm commands [#560](https://github.com/GoogleContainerTools/skaffold/pull/560)
* Can now run `skaffold build` without a kubernetes cluster [#540](https://github.com/GoogleContainerTools/skaffold/pull/540)

Updates
* Updated google/go-containerregistry [#571](https://github.com/GoogleContainerTools/skaffold/pull/571)
* Added a user agent to GCB calls [#582](https://github.com/GoogleContainerTools/skaffold/pull/582)
* Simplified runner code [#540](https://github.com/GoogleContainerTools/skaffold/pull/540)
* Silenced usage text on errors [#449](https://github.com/GoogleContainerTools/skaffold/pull/449)
* Skipped fully qualified names when replacing image names [#566](https://github.com/GoogleContainerTools/skaffold/pull/566)
* Improved docker dependencies code [#466](https://github.com/GoogleContainerTools/skaffold/pull/466)

https://github.com/GoogleContainerTools/skaffold/compare/v0.6.0...v0.6.1

# v0.6.0 Release - 5/16/2018
New Features
* Improve the `docker load` output in Bazel build [#475](https://github.com/GoogleContainerTools/skaffold/pull/475)
* `envTemplate` now supports `DIGEST_ALGO` and `DIGEST_HEX` variables [#495](https://github.com/GoogleContainerTools/skaffold/pull/495)
* Perform cleanup on `SIGPIPE` signal [#515](https://github.com/GoogleContainerTools/skaffold/pull/515)
* Learnt a `skaffold build` command [#476](https://github.com/GoogleContainerTools/skaffold/pull/476), [#553](https://github.com/GoogleContainerTools/skaffold/pull/553)
* Git tagger should use tags over commits [#552](https://github.com/GoogleContainerTools/skaffold/pull/552)

Bug Fixes
* Fixed the microservices example [#451](https://github.com/GoogleContainerTools/skaffold/pull/451)
* Don't fail if `~/.docker/config.json` doesn't exist [#454](https://github.com/GoogleContainerTools/skaffold/pull/454)
* Fix the Git Tagger name [#473](https://github.com/GoogleContainerTools/skaffold/pull/473)
* Git Tagger now handles deleted files without failing [#471](https://github.com/GoogleContainerTools/skaffold/pull/471)
* Add files to the context tarball with Unix separators [#489](https://github.com/GoogleContainerTools/skaffold/pull/489)
* Fix and improve `annotated-skaffold.yaml` [#467](https://github.com/GoogleContainerTools/skaffold/pull/467), [#520](https://github.com/GoogleContainerTools/skaffold/pull/520), [#536](https://github.com/GoogleContainerTools/skaffold/pull/536), [#542](https://github.com/GoogleContainerTools/skaffold/pull/542)
* Handle private docker registries with explicit port numbers [#525](https://github.com/GoogleContainerTools/skaffold/pull/525)
* Ignore empty manifests [#538](https://github.com/GoogleContainerTools/skaffold/pull/538)
* Default values are set after a profile is applied [#533](https://github.com/GoogleContainerTools/skaffold/pull/533)
* Remove warning when building images [#548](https://github.com/GoogleContainerTools/skaffold/pull/548)
* Some logs where not printed [#513](https://github.com/GoogleContainerTools/skaffold/pull/513)

Updates
* Improvements to the documentation [#452](https://github.com/GoogleContainerTools/skaffold/pull/452), [#453](https://github.com/GoogleContainerTools/skaffold/pull/453), [#556](https://github.com/GoogleContainerTools/skaffold/pull/556)
* Improve `kubectl` and `helm` commands output [#534](https://github.com/GoogleContainerTools/skaffold/pull/534)
* Code improvements [#485](https://github.com/GoogleContainerTools/skaffold/pull/485), [#537](https://github.com/GoogleContainerTools/skaffold/pull/537), [#544](https://github.com/GoogleContainerTools/skaffold/pull/544), [#545](https://github.com/GoogleContainerTools/skaffold/pull/545)
* Improved Git Issue template [#532](https://github.com/GoogleContainerTools/skaffold/pull/532)

https://github.com/GoogleContainerTools/skaffold/compare/v0.5.0...v0.6.0

# v0.5.0 Release - 4/23/2018
New Features
* Added kaniko builder
* Added support for "remote-manifests" in kubectl deployer
* `skaffold dev` now performs a cleanup of deployed resources on exit
* `skaffold dev` redeploys when deploy dependencies are changed (only kubectl deployer currently)

Bug Fixes
* GCB builder now uses tags correctly
* Supports multi-stage dockerfiles with onbuild commands
* Better error messages
* Fixed tagger working directory

Updates
* Switched from containers/image to google/go-containerregistry
* Integration tests now run in separate namespaces

# v0.4.0 Release - 4/12/2018
New Features
* Added `skaffold fix` command to migrate configs from v1alpha1 to v1alpha2
* Added `skaffold completion` command to output bash completion for skaffold subcommands
* Warns when an image is built but not used
* Artifacts can now be built with bazel
* Environment variable template tagger
* Support multiple document YAML files
* Helm deployer now accepts extra set values

Bug Fixes
* Logs use relative time instead of host time, which fixes issues with clock sync on local clusters
* Removed duplicate error
* Docker build args passed to Google Container Builder
* Fixed unreliable file detection when using IntelliJ or other IDEs
* Better handling of default values
* Fixed issue with some logs being displayed twice
* Fixed .dockerignore support

Updates
* Updated go-git package
* Refactored watch package

# v0.3.0 Release - 3/29/2018
New Features
* Logs are now colored by image deployment, different container instances will get different colors in `skaffold dev`
* Better and less error prone logging
* All logs are shown for a pod with deployed containers
* Helm deployer now supports chart versions
* Helm deployer now supports custom values file path
* Logs are now muted during the build and deploy cycle
* Integration tests added
* Conditional rebuilding of changed artifacts
* Builds only triggered after a quiet period
* Duration of build and deploy is now logged


Bug Fixes
* .dockerignore now works if context is parent directory
* removed duplicate logs
* private registry authentication issues are fixed
* no logs are missed

Breaking config changes
* tagPolicy is now a struct instead of a string, to convert to the new config format

```
tagPolicy: gitCommit
```
becomes
```
tagPolicy:
    gitCommit: {}
```

* kubectl deployer no longer needs templated keys in manifests. Simply just make sure the artifacts in your skaffold.yaml correspond to the images in your kubernetes manifests and they will be automatically updated.

# v0.2.0 Release - 3/9/2018

New Features
* Added "skip-push" optimization for Docker for Desktop Kubernetes Clusters
* Examples should now be ran from their own directory
* Fixed kubernetes context for build and deploy
* Added options for GCR auth
* Set default log level to warn
* Change git commit to use short ID instead
* Helm deployer now accepts namespace and values file
* Local builder now accepts docker build-args
* Added --tag flag for skaffold run
* Cache image configs by name
* Kubectl Generate a basic manifest if none provided

Bug fixes
* Dockerfile parsing for remote ADD file works correctly now
* Closed image config file descriptor

# v0.1.0 Release - 3/2/2018

* Added `skaffold run` command
* Added `skaffold dev` command
* Added `skaffold version` command
* Added `skaffold docker deps` command to parse dockerfile dependencies
* Added `skaffold docker context` command to generate minimal docker context tar
* Added Builders: Local Docker, Google Cloud Builder
* Added Deployers: Kubectl, Helm
* Filesystem watcher (kqueue)
* Log streaming of deploy resources
* Minikube optimizations
* Dockerfile introspection
* Added initial skaffold docker image with dependencies
* Globbing filepath config fields
* Added skaffold config
* Added initial integration test
