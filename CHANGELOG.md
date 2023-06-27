# v2.6.0 Release - 06/27/2023
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.6.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.6.0/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.6.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.6.0/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.6.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.6.0`

Note: This release comes with a new config version, `v4beta6`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:

New Features and Additions:
* feat: implement skaffold delete for docker deployer using docker labels [#8885](https://github.com/GoogleContainerTools/skaffold/pull/8885)
* feat: new verify timeout config feature [#8801](https://github.com/GoogleContainerTools/skaffold/pull/8801)
* feat: support tempalte parameterization for helm [#8911](https://github.com/GoogleContainerTools/skaffold/pull/8911)
* feat: logic to only emit the tags related with verify on `skaffold verify` [#8851](https://github.com/GoogleContainerTools/skaffold/pull/8851)

Fixes:
* fix: Go default template doesn't work for tagging [#8881](https://github.com/GoogleContainerTools/skaffold/pull/8881)
* fix: Clean up dev images except the last [#8897](https://github.com/GoogleContainerTools/skaffold/pull/8897)
* fix: add docker to the LTS container images [#8905](https://github.com/GoogleContainerTools/skaffold/pull/8905)
* fix: #8870 manifest kustomize paths using env var with absolute path [#8877](https://github.com/GoogleContainerTools/skaffold/pull/8877)
* fix: condition to not update helm deployer hook patches, is not needed [#8862](https://github.com/GoogleContainerTools/skaffold/pull/8862)
* fix: logic to interrupt a k8sjob logs as soon as it fails [#8847](https://github.com/GoogleContainerTools/skaffold/pull/8847)
* Always Pass skaffold binary in post-render to add labels for status-check [#8826](https://github.com/GoogleContainerTools/skaffold/pull/8826)
* fix: conditionally drain docker logs on stop to avoid docker deployer to stay in infinite loop [#8838](https://github.com/GoogleContainerTools/skaffold/pull/8838)
* fix(ko): Ko builder push vs load behavior [#8845](https://github.com/GoogleContainerTools/skaffold/pull/8845)
* fix: Replace Kustomize field `patches` in examples [#8757](https://github.com/GoogleContainerTools/skaffold/pull/8757)

Updates and Refactors:
* chore: port apply-setter krm function over to skaffold [#8902](https://github.com/GoogleContainerTools/skaffold/pull/8902)
* chore: upgrade go version [#8895](https://github.com/GoogleContainerTools/skaffold/pull/8895)
* chore: bump github/codeql-action from 2.20.0 to 2.20.1 [#8903](https://github.com/GoogleContainerTools/skaffold/pull/8903)
* chore: bump github/codeql-action from 1.0.26 to 2.20.0 [#8888](https://github.com/GoogleContainerTools/skaffold/pull/8888)
* chore: bump schema version to v4beta6 [#8849](https://github.com/GoogleContainerTools/skaffold/pull/8849)
* chore: bump peter-evans/create-or-update-comment from 3.0.1 to 3.0.2 [#8865](https://github.com/GoogleContainerTools/skaffold/pull/8865)
* chore: add lock to jib cache lookup [#8850](https://github.com/GoogleContainerTools/skaffold/pull/8850)
* chore: update release build script to support internal scanning [#8834](https://github.com/GoogleContainerTools/skaffold/pull/8834)

Docs, Test, and Release Updates:
* test: increase progressDeadlineSeconds timeout for TestRunUnstableChecked [#8833](https://github.com/GoogleContainerTools/skaffold/pull/8833)

Huge thanks goes out to all of our contributors for this release:

- dependabot[bot]
- ericzzzzzzz
- Halvard Skogsrud
- Michael Plump
- rajesh
- Renzo Rojas
- Ryan Ohnemus

# v2.5.0 Release - 05/25/2023
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.5.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.5.0/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.5.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.5.0/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.5.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.5.0`

New Features and Additions:
* feat: new k8s task and exec env for custom actions [#8755](https://github.com/GoogleContainerTools/skaffold/pull/8755)

Updates and Refactors:
* chore: update various container deps. [#8810](https://github.com/GoogleContainerTools/skaffold/pull/8810)
* test: disable failing buildpacks tests [#8812](https://github.com/GoogleContainerTools/skaffold/pull/8812)
* chore: add script to update lts dependencies [#8773](https://github.com/GoogleContainerTools/skaffold/pull/8773)
* chore: update go deps [#8789](https://github.com/GoogleContainerTools/skaffold/pull/8789)
* chore: update Dockerfile bin deps [#8774](https://github.com/GoogleContainerTools/skaffold/pull/8774)
* chore: bump github.com/cloudflare/circl from 1.1.0 to 1.3.3 [#8771](https://github.com/GoogleContainerTools/skaffold/pull/8771)
* chore: bump github.com/docker/distribution from 2.8.1+incompatible to 2.8.2+incompatible [#8772](https://github.com/GoogleContainerTools/skaffold/pull/8772)
* chore: Upload deps lisences [#8747](https://github.com/GoogleContainerTools/skaffold/pull/8747)
* chore: updated examples/ to updated schema version [#8748](https://github.com/GoogleContainerTools/skaffold/pull/8748)
* chore: bump flask from 2.3.1 to 2.3.2 in /integration/examples [#8734](https://github.com/GoogleContainerTools/skaffold/pull/8734)
* chore: bump flask from 2.3.1 to 2.3.2 in /examples [#8735](https://github.com/GoogleContainerTools/skaffold/pull/8735)
* chore: bump flask from 1.0 to 2.2.5 in /examples/hot-reload/python [#8744](https://github.com/GoogleContainerTools/skaffold/pull/8744)
* chore: bump peter-evans/create-or-update-comment from 3.0.0 to 3.0.1 [#8736](https://github.com/GoogleContainerTools/skaffold/pull/8736)
* chore: bump flask from 1.0 to 2.2.5 in /integration/examples/hot-reload/python [#8738](https://github.com/GoogleContainerTools/skaffold/pull/8738)
* chore: bump github.com/sigstore/rekor from 1.0.1 to 1.1.1 [#8741](https://github.com/GoogleContainerTools/skaffold/pull/8741)

Docs, Test, and Release Updates:
* docs: doc page for Custom Actions and skaffold exec [#8809](https://github.com/GoogleContainerTools/skaffold/pull/8809)
* docs: fix reference to dateTime tagger [#8813](https://github.com/GoogleContainerTools/skaffold/pull/8813)
* docs: update skaffold.yaml page to show latest schema version [#8808](https://github.com/GoogleContainerTools/skaffold/pull/8808)
* docs: add `overrides` and `jobManifestPath` to verify docs [#8762](https://github.com/GoogleContainerTools/skaffold/pull/8762)
* fix: resolve issue where hack/release.sh wouldn't mark schema as released [#8752](https://github.com/GoogleContainerTools/skaffold/pull/8752)
* fix: resolve issues with hack/new-version.sh so it works w/ no manual changes necessary [#8750](https://github.com/GoogleContainerTools/skaffold/pull/8750)
* fix: scanning filter not working properly due to version sorting [#8727](https://github.com/GoogleContainerTools/skaffold/pull/8727)

Huge thanks goes out to all of our contributors for this release:

- Aaron Prindle
- Benjamin Petersen
- dependabot[bot]
- ericzzzzzzz
- Maggie Neterval
- Michael Plump
- Renzo Rojas

# v2.4.1 Release - 05/10/2023
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.4.1/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.4.1/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.4.1/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.4.1/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.4.1/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.4.1`

Fixes:
* fix: discard standout from helm dep command to not have corrupted data in output yaml file (#8756)

# v2.4.0 Release - 05/03/2023
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.4.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.4.0/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.4.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.4.0/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.4.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.4.0`

Note: This release comes with a new config version, `v4beta5`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

New Features and Additions:
feat: add custom actions execution modes to `inspect executionModes list` (#8697)
feat: add new 'skaffold inspect executionModes' command (#8651)
feat: add support for deployment cancellation and error surface when admission webhook blocks pod skaffold is waiting on (#8624)
feat: add template support for `chartPath` (#8645)
feat: add wait group in docker and k8s job logger to avoid race condition (#8695)
feat: better Job support by allowing skaffold to re-apply Jobs by removing child pod label transform (#8659)
feat: change inspect executionMode list to list all actions exec mode per default (#8719)
feat: custom actions interfaces and actions runner implementation (#8563)
feat: extend `inspect jobManifestPaths [list|modify]` to include custom actions info (#8703)
feat: extend schema to support customActions stanza (#8616)
feat: helm dependency build on render (#8486)
feat: logic to create a new ActionsRunner (#8681)
feat: new docker exec env and task (#8662)
feat: new exec command to execute a custom action (#8696)
feat: read firelog API key from embedded file (#8646)
feat: set firelog API key (#8617)
feat: standarize renders to inject namespace only if --namespace or render specific config is set (#8561)
feat: support set value file for render (#8647)

Fixes:
fix: add curl to skaffold docker image (#8669)
fix: add the LOG_STREAM_RUN_GCLOUD_NOT_FOUND code in the proto file and run of ./hack/generate-proto.sh (#8644)
fix: change util pkg import to use ParseEnvVariablesFromFile (#8700)
fix: create new docker network only when the --docker-network flag is not set (#8649)
fix: delete does not working properly (#8702)
fix: deploy to multiple namespaces (#8623)
fix: fix issue where verify would panic if a jobManifestPath with no spec.template.spec.metadata.labels existed (#8618)
fix: new `remove` method in docker client to use in custom actions and don't get the error related with prune (#8710)
fix: remove printing errors from IsKubernetesManifest and change doApply to not use this method but ParseKubernetesObjects to retrieve the error (#8559)
fix: resolve issue where skaffold logger could hang indefinitely if k8s job pod wasn't created (#8717)
fix: update skaffold verify to respect deploy default namespace field (#8660)
fix: use active gcp account (#8584)
fix: missing Sessionable ID in exported metrics (#8737)

Updates and Refactors:
chore: add v2.0.8 release to CHANGELOG.md (#8685)	
chore: add v2.3.1 release to CHANGELOG.md (#8663)	
chore: bump examples/ schema versions (#8607)	
chore: bump flask from 2.2.3 to 2.3.1 in /examples (#8707)	
chore: bump flask from 2.2.3 to 2.3.1 in /integration/examples (#8706)	
chore: bump github.com/docker/docker (#8636)	
chore: bump github.com/opencontainers/runc from 1.1.4 to 1.1.5 (#8602)	
chore: bump google.golang.org/protobuf from 1.29.0 to 1.29.1 (#8705)	
chore: bump image deps (#8612)	
chore: bump ossf/scorecard-action from 2.1.2 to 2.1.3 (#8614)	
chore: bump peter-evans/create-or-update-comment from 2.1.1 to 3.0.0 (#8639)	
chore: bump xt0rted/pull-request-comment-branch from 1.4.0 to 2.0.0 (#8613)	
chore: increase vulns scanning window (#8723)	
chore: remove kaniko NoPush field from skaffold schemas as it does not work currently (#8591)	
chore: restore firelog exporter  (#8555) (#8599)	
chore: update go version and related deps to enhance security (#8704)	
chore: Update ko builder to use ko v0.13.0 (#8699)	
chore: upgrade docker and make integration-in-docker to use docker dependencies from pr (#8596)	
chore: vendor deps (#8725)


Docs, Test, and Release Updates:
ci: Use Go 1.20 in GitHub Actions workflows (#8691)
docs: fix typo in render page (#8638)
docs: minor edits to Cloud Build docs (#8571)
docs: Tutorial: Go coverage profiles for e2e tests (#8558)
docs: update helm renderer docs to use correct manifests vs deploy syntax (#8667)
docs: update templating.md to reflect chartPath addition + minor field fixes (#8661)
docs: update verify docs to reflect k8s job support (#8601)
test: comment last line of expected log due to issue #8728 (#8729)

Huge thanks goes out to all of our contributors for this release:

- Aaron Prindle
- Anis Khan
- dependabot[bot]
- Dominik Siebel
- ericzzzzzzz
- Gaurav
- Halvard Skogsrud
- Maggie Neterval
- Renzo Rojas
- Vishnu Bharathi

# v2.0.8 Release - 4/17/2023
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.8/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.8/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.8/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.8/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.0.8/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.0.8`

Fixes/Chores:
* chore: chore: bump helm to v3.10.2 [#8684](https://github.com/GoogleContainerTools/skaffold/pull/8684)

# v2.3.1 Release - 4/11/2023
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.3.1/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.3.1/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.3.1/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.3.1/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.3.1/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.3.1`

Fixes:
* fix: update skaffold verify to respect deploy default namespace field (#8660)
* fix: fix issue where verify would panic if a jobManifestPath with no spec.template.spec.metadata.labels existed (#8618)
* fix: deploy to multiple namespaces (#8623)

Features:
* feat: better Job support by allowing skaffold to re-apply Jobs by removing child pod label transform (#8659)
* feat: add support for deployment cancellation and error surface when admission webhook blocks pod skaffold is waiting on (#8624)
* feat: add new 'skaffold inspect executionModes' command (#8651)

Chores:
* chore: upgrade docker and make integration-in-docker to use docker dependencies from pr (#8596)
* chore: bump image deps (#8612)

# v2.3.0 Release - 03/28/2023
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.3.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.3.0/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.3.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.3.0/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.3.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.3.0`

Note: This release comes with a new config version, `v4beta4`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:
* `skaffold verify` functionality fixed to properly support environment variables and has updated support allowing users to now runs verify tests as Kubernetes Jobs (in additional to the previously supported local container option)
* When using the `tolerateFailuresUntilDeadline` or `--tolerate-failures-until-deadline` flag, now Skaffold will also tolerate specific cluster connection issues until deadline (in additonal to the previous supported k8s status issues).

New Features and Additions:
* feat: add 'skaffold inspect jobManifestPath' and 'skaffold transform-schema jobManifestPath' commands [#8575](https://github.com/GoogleContainerTools/skaffold/pull/8575)
* feat: add k8s Job support to verify and status check [#8415](https://github.com/GoogleContainerTools/skaffold/pull/8415)
* feat: Whitelist strimzi.io CRDs [#8491](https://github.com/GoogleContainerTools/skaffold/pull/8491)

Fixes:
* fix: add upgrade logic to inject a kubectl deployer when an old kustomize deployer is detected [#8457](https://github.com/GoogleContainerTools/skaffold/pull/8457)
* fix: can't use ctrl-c to terminate building with kaniko at uploading build context stage [#8516](https://github.com/GoogleContainerTools/skaffold/pull/8516)
* fix: correctly rewrite /deploy/kubectl/manifests in patch upgrades [#8585](https://github.com/GoogleContainerTools/skaffold/pull/8585)
* fix: improve verify command s.t. os envs not passed through to container envs and instead add a flag for this purpose [#8557](https://github.com/GoogleContainerTools/skaffold/pull/8557)
* fix: make it so tolerateFailuresUntilDeadline also handles kubectl failures (vs. just parsing kubectl resource status values). [#8549](https://github.com/GoogleContainerTools/skaffold/pull/8549)
* fix: move verify schema changes to v4beta4 and remove it from already released v4beta3 [#8514](https://github.com/GoogleContainerTools/skaffold/pull/8514)

Updates and Refactors:
* chore: "revert upgrade docker version in skaffold image (#8583)" [#8590](https://github.com/GoogleContainerTools/skaffold/pull/8590)
* chore: Add output flag for diagnose [#8546](https://github.com/GoogleContainerTools/skaffold/pull/8546)
* chore: change verify schema from v1.Container to subset of direct primitive types [#8577](https://github.com/GoogleContainerTools/skaffold/pull/8577)
* chore: enhance vuln monitor [#8570](https://github.com/GoogleContainerTools/skaffold/pull/8570)
* chore: fix transformer share config-map issues [#8582](https://github.com/GoogleContainerTools/skaffold/pull/8582)
* chore: update cloudbuild config to publish distroless-skaffold image to artifact registry for vulns scanning [#8524](https://github.com/GoogleContainerTools/skaffold/pull/8524)
* chore: upgrade docker version in skaffold image [#8583](https://github.com/GoogleContainerTools/skaffold/pull/8583)
* chore: upgrade go version in skaffold image [#8540](https://github.com/GoogleContainerTools/skaffold/pull/8540)
* dep: replace dockerignore.ReadAll withgithub.com/moby/buildkit/frontend/dockerfile/dockerignore.ReadAll [#8488](https://github.com/GoogleContainerTools/skaffold/pull/8488)

Docs, Test, and Release Updates:
* docs: add tooltip with yaml path in Skaffold yaml reference page [#8477](https://github.com/GoogleContainerTools/skaffold/pull/8477)


Huge thanks goes out to all of our contributors for this release:

- Aaron Prindle
- catusax
- Dan Williams
- dependabot[bot]
- ericzzzzzzz
- Gaurav
- Mike Roberts
- Patryk Małek
- Renzo Rojas

# v2.2.0 Release - 03/06/2023
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.2.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.2.0/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.2.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.2.0/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.2.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.2.0`

Note: This release comes with a new config version, `v4beta3`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

New Features and Additions:
* feat: support deploying to multiple clusters [#8459](https://github.com/GoogleContainerTools/skaffold/pull/8459)
* feat: support parameterizations for all renders [#8365](https://github.com/GoogleContainerTools/skaffold/pull/8365)
* feat: define `skaffold.env` file for loading environment variables [#8395](https://github.com/GoogleContainerTools/skaffold/pull/8395)
* feat: add timeout for copying build context on kaniko [#8329](https://github.com/GoogleContainerTools/skaffold/pull/8329)
* feat: add support for remote docker over ssh [#8349](https://github.com/GoogleContainerTools/skaffold/pull/8349)

Fixes:
* fix: retag multiarch images [#8493](https://github.com/GoogleContainerTools/skaffold/pull/8493)
* fix: skaffold render namespace regression in v2 [#8482](https://github.com/GoogleContainerTools/skaffold/pull/8482)
* fix: resolve issue in which skaffold + helm did not properly handle ':' chars in go templating [#8464](https://github.com/GoogleContainerTools/skaffold/pull/8464)
* fix: expand home directory for docker secrets [#8476](https://github.com/GoogleContainerTools/skaffold/pull/8476)
* fix: output cause of kubernetes manifest parsing error vs generic message [#8463](https://github.com/GoogleContainerTools/skaffold/pull/8463)
* Add missing space [#8450](https://github.com/GoogleContainerTools/skaffold/pull/8450)
* fix: not inject `metadata.namespace` in manifests rendered with kustomize [#8409](https://github.com/GoogleContainerTools/skaffold/pull/8409)
* fix: change log level to Info so `skaffold render --output=render.yaml` produces same output as `skaffold render &> render.yaml` [#8341](https://github.com/GoogleContainerTools/skaffold/pull/8341)
* fix: reverse order of deployers during cleanup (#7284) (backport v1) [#7927](https://github.com/GoogleContainerTools/skaffold/pull/7927)
* fix: prevent long startup time [#8376](https://github.com/GoogleContainerTools/skaffold/pull/8376)
* fix: resolve issue where verify validation did not properly validate uniqueness across all modules [#8373](https://github.com/GoogleContainerTools/skaffold/pull/8373)
* fix:  envTemplate is not working with command template and output with… [#8393](https://github.com/GoogleContainerTools/skaffold/pull/8393)
* fix: resolve issue where verify incorrectly failed when run with multiple modules where any module didn't have verify test cases [#8369](https://github.com/GoogleContainerTools/skaffold/pull/8369)
* fix: resolve issue where kubectl Flags.Apply namespace flag usage would fail [#8351](https://github.com/GoogleContainerTools/skaffold/pull/8351)

Updates and Refactors:
* chore: Security hotfixes for v2.0.6 branch [#8480](https://github.com/GoogleContainerTools/skaffold/pull/8480)
* chore: remove validation for kpt version [#8425](https://github.com/GoogleContainerTools/skaffold/pull/8425)
* chore: bump golang.org/x/net from 0.0.0-20220909164309-bea034e7d591 to 0.7.0 in /integration/examples/grpc-e2e-tests/service [#8478](https://github.com/GoogleContainerTools/skaffold/pull/8478)
* chore: bump golang.org/x/net from 0.0.0-20220909164309-bea034e7d591 to 0.7.0 in /examples/grpc-e2e-tests/service [#8473](https://github.com/GoogleContainerTools/skaffold/pull/8473)
* chore: bump golang.org/x/crypto from 0.0.0-20210921155107-089bfa567519 to 0.1.0 in /hack/tools [#8469](https://github.com/GoogleContainerTools/skaffold/pull/8469)
* chore: upgrade kpt to support parameterization [#8470](https://github.com/GoogleContainerTools/skaffold/pull/8470)
* chore: remove unnecessary code comments [#8350](https://github.com/GoogleContainerTools/skaffold/pull/8350)
* chore: delete unused helm fields and methods from v1 -> v2 migration [#8461](https://github.com/GoogleContainerTools/skaffold/pull/8461)
* chore: bump golang.org/x/text from 0.3.7 to 0.3.8 in /examples/grpc-e2e-tests/service [#8468](https://github.com/GoogleContainerTools/skaffold/pull/8468)
* chore: bump golang.org/x/text from 0.3.7 to 0.3.8 in /integration/examples/grpc-e2e-tests/cloud-spanner-bootstrap [#8467](https://github.com/GoogleContainerTools/skaffold/pull/8467)
* chore(deps): bump actions/upload-artifact from 3.1.1 to 3.1.2 [#8304](https://github.com/GoogleContainerTools/skaffold/pull/8304)
* chore: bump golang.org/x/text from 0.3.7 to 0.3.8 in /examples/grpc-e2e-tests/cloud-spanner-bootstrap [#8466](https://github.com/GoogleContainerTools/skaffold/pull/8466)
* chore: bump golang.org/x/text from 0.3.7 to 0.3.8 in /integration/examples/grpc-e2e-tests/service [#8465](https://github.com/GoogleContainerTools/skaffold/pull/8465)
* chore: Update skaffold base image [#8460](https://github.com/GoogleContainerTools/skaffold/pull/8460)
* chore: change skaffold base image [#8433](https://github.com/GoogleContainerTools/skaffold/pull/8433)
* chore: add krm functions to allowList [#8445](https://github.com/GoogleContainerTools/skaffold/pull/8445)
* chore(deps): bump github.com/containerd/containerd from 1.6.15 to 1.6.18 [#8444](https://github.com/GoogleContainerTools/skaffold/pull/8444)
* chore(deps): bump flask from 2.2.2 to 2.2.3 in /integration/examples [#8442](https://github.com/GoogleContainerTools/skaffold/pull/8442)
* chore(deps): bump flask from 2.2.2 to 2.2.3 in /examples [#8443](https://github.com/GoogleContainerTools/skaffold/pull/8443)
* chore: upgrade dependencies [#8431](https://github.com/GoogleContainerTools/skaffold/pull/8431)
* chore: upgrade go in dockerfile [#8420](https://github.com/GoogleContainerTools/skaffold/pull/8420)
* chore: bump pack version used in skaffold pack image [#8428](https://github.com/GoogleContainerTools/skaffold/pull/8428)
* chore: bump schema version to v4beta3 [#8421](https://github.com/GoogleContainerTools/skaffold/pull/8421)
* refactor: replace 4d63.com/tz with time/tzdata [#8408](https://github.com/GoogleContainerTools/skaffold/pull/8408)
* chore(deps): bump peter-evans/create-or-update-comment from 2.1.0 to 2.1.1 [#8404](https://github.com/GoogleContainerTools/skaffold/pull/8404)
* chore(deps): bump rack from 2.1.4.1 to 2.1.4.2 in /examples/ruby/backend [#8332](https://github.com/GoogleContainerTools/skaffold/pull/8332)

Docs, Test, and Release Updates:
* docs: add `minikube tunnel` command in the tutorial [#8490](https://github.com/GoogleContainerTools/skaffold/pull/8490)
* chore: update CHANGELOG.md with 3 patch releases [#8487](https://github.com/GoogleContainerTools/skaffold/pull/8487)
* Docs: Update _index.md to use more appropriate grammar [#8484](https://github.com/GoogleContainerTools/skaffold/pull/8484)
* chore: release/v1.39.6 [#8479](https://github.com/GoogleContainerTools/skaffold/pull/8479)
* chore: add docs to explain keep-running-on-failure [#8446](https://github.com/GoogleContainerTools/skaffold/pull/8446)
* docs: follow-ups to builders page refactor [#8449](https://github.com/GoogleContainerTools/skaffold/pull/8449)
* bump golang.org/x/net from 0.6.0 to 0.7.0 [#8451](https://github.com/GoogleContainerTools/skaffold/pull/8451)
* docs: add Python 3.11 not currently supported but coming soon info [#8435](https://github.com/GoogleContainerTools/skaffold/pull/8435)
* docs: restructure builders docs [#8426](https://github.com/GoogleContainerTools/skaffold/pull/8426)
* docs: update upgrade.md to reflect helm hooks support change [#8419](https://github.com/GoogleContainerTools/skaffold/pull/8419)
* docs: fix issue where debug.md link was not rendered properly [#8412](https://github.com/GoogleContainerTools/skaffold/pull/8412)
* docs: add IMAGE_DIGEST_* as well to *.tag docs examples [#8402](https://github.com/GoogleContainerTools/skaffold/pull/8402)
* docs: add more detail to Cloud Run deployer page [#8381](https://github.com/GoogleContainerTools/skaffold/pull/8381)
* docs: Upgrade hugo and docsy versions to enable collapsible nav [#8398](https://github.com/GoogleContainerTools/skaffold/pull/8398)
* chore: bump examples to v4beta2 after v2.1.0 release [#8355](https://github.com/GoogleContainerTools/skaffold/pull/8355)
* test: integration test for helm render with OCI repo [#8352](https://github.com/GoogleContainerTools/skaffold/pull/8352)


Huge thanks goes out to all of our contributors for this release:

- Aaron Prindle
- Alex
- David Peleg
- Gaurav
- Hironori Yamamoto
- Maggie Neterval
- Nick Phillips
- Patryk Małek
- Renzo Rojas
- Stephen Johnston
- Thomas Griseau
- catusax
- ericzzzzzzz

# v2.0.6 Release - 3/02/2022
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.6/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.6/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.6/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.6/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.0.6/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.0.6`

## What's Changed
* fix: skaffold render namespace regression in v2 [#8482](https://github.com/GoogleContainerTools/skaffold/pull/8482)
* chore: Security hotfixes for v2.0.6 branch [#8480](https://github.com/GoogleContainerTools/skaffold/pull/8480)

# v1.39.6 Release - 3/01/2022
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.39.6/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.39.6/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.39.6/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.39.6/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v1.39.6/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.39.6`

Note: This is a security patch release.

## What's Changed
* chore: release/v1.39.6 by @ericzzzzzzz in https://github.com/GoogleContainerTools/skaffold/pull/8479

# v1.37.3 Release - 03/01/2023
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.37.3/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.37.3/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.37.3/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.37.3/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v1.37.3/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.37.3`

Note: This is a security patch release.

# v2.1.0 Release - 01/20/2023
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.1.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.1.0/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.1.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.1.0/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.1.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.1.0`

Note: This release comes with a new config version, `v4beta2`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:

New Features and Additions:
* feat: add ingore-path kaniko flag support [#8340](https://github.com/GoogleContainerTools/skaffold/pull/8340)
* feat: add keep-running-on-failure-implementation [#8270](https://github.com/GoogleContainerTools/skaffold/pull/8270)
* feat: add new inspect namespaces list command to skaffold [#8309](https://github.com/GoogleContainerTools/skaffold/pull/8309)
* feat: allow specifying debug runtime in `skaffold.yaml` for artifact [#8295](https://github.com/GoogleContainerTools/skaffold/pull/8295)
* feat: change components installed in docker images to include alpha and beta [#8314](https://github.com/GoogleContainerTools/skaffold/pull/8314)
* feat: get image digest from container logs for kaniko builder [#8264](https://github.com/GoogleContainerTools/skaffold/pull/8264)
* feat: support-external-cmd-call-in-template [#8296](https://github.com/GoogleContainerTools/skaffold/pull/8296)

Fixes:
* fix: add proper artifactOverrides->setValueTemplates conversion when upgrading from v2beta29 [#8335](https://github.com/GoogleContainerTools/skaffold/pull/8335)
* fix: backport, divide stdout and stderr from helm to not create corrupted outputs [#8333](https://github.com/GoogleContainerTools/skaffold/pull/8333)
* fix: resolve issue where skaffold always added namespace to rendered manifests [#8312](https://github.com/GoogleContainerTools/skaffold/pull/8312)
* fix: resolve issue where skaffold would panic when StatusCheck was not set [#8135](https://github.com/GoogleContainerTools/skaffold/pull/8135)
* fix: use new URL format for Google Cloud Build log [#8323](https://github.com/GoogleContainerTools/skaffold/pull/8323)
* fix: use release namespace in render when specified [#8259](https://github.com/GoogleContainerTools/skaffold/pull/8259)
* fix: write maximum of 200 metrics per session [#8294](https://github.com/GoogleContainerTools/skaffold/pull/8294)
* fix: handle StatefulSets with an OnDelete update strategy [#8292](https://github.com/GoogleContainerTools/skaffold/pull/8292)

Updates and Refactors:
* chore: make iterative status check default to true [#8212](https://github.com/GoogleContainerTools/skaffold/pull/8212)
* chore(deps): bump ossf/scorecard-action from 2.1.1 to 2.1.2 [#8278](https://github.com/GoogleContainerTools/skaffold/pull/8278)
* chore: update skaffold image deps based on lts policy [#8347](https://github.com/GoogleContainerTools/skaffold/pull/8347)

Docs, Test, and Release Updates:
* chore: unskip TestFix* integration tests [#8334](https://github.com/GoogleContainerTools/skaffold/pull/8334)
* docs: add status check documentation for new tolerateFailuresUntilDeadline config field [#8337](https://github.com/GoogleContainerTools/skaffold/pull/8337)
* docs: remove `log tailing` from note of unsupported features for Cloud Run [#8344](https://github.com/GoogleContainerTools/skaffold/pull/8344)
* doc: Updating installation link for Cloud Code in VSCode [#8326](https://github.com/GoogleContainerTools/skaffold/pull/8326)
* docs: remove duplicate maturity entry for Cloud Run Deployer [#8280](https://github.com/GoogleContainerTools/skaffold/pull/8280)
* docs: update cloudrun docs to include log streaming and Job support [#8338](https://github.com/GoogleContainerTools/skaffold/pull/8338)
* docs: update docs for new `runtimeType` field [#8298](https://github.com/GoogleContainerTools/skaffold/pull/8298)
* docs: Update Quickstart and Tutorials pages with new Skaffold onboarding walkthrough [#8274](https://github.com/GoogleContainerTools/skaffold/pull/8274)


Huge thanks goes out to all of our contributors for this release:

- Aaron Prindle
- Benjamin Kaplan
- dependabot[bot]
- Eng Zer Jun
- ericzzzzzzz
- Gaurav
- GregCKrause
- hampus77
- Jeremy Glover
- June Rhodes
- Laurent Grangeau
- Maggie Neterval
- Oleksandr Simonov
- qwerjkl112
- Renzo Rojas
- Riccardo Carlesso
- Romin Irani
- Seth Rylan Gainey
- Suzuki Shota
- TAKAHASHI Shuuji
- Uzlopak

# v2.0.5 Release - 1/20/2023
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.5/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.5/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.5/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.5/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.0.5/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.0.5`

Fixes:
* fix: use release namespace in render when specified [#8259](https://github.com/GoogleContainerTools/skaffold/pull/8259)
* fix: add proper artifactOverrides->setValueTemplates conversion when upgrading from v2beta29 [#8335](https://github.com/GoogleContainerTools/skaffold/pull/8335)

# v1.39.5 Release - 1/20/2022
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.39.5/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.39.5/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.39.5/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.39.5/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v1.39.5/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.39.5`

Fixes:
* fix: backport, divide stdout and stderr from helm to not create corrupted outputs [#8333](https://github.com/GoogleContainerTools/skaffold/pull/8333)

# v2.0.4 Release - 12/21/2022
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.4/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.4/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.4/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.4/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.0.4/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.0.4`

Fixes:
* fix: resolve issue where skaffold would not add digest when using helm in v1 -> v2 migration case [#8269](https://github.com/GoogleContainerTools/skaffold/pull/8269)
* fix: remove kubecontext check warning from kubectl deploy [#8256](https://github.com/GoogleContainerTools/skaffold/pull/8256)
* fix: expand namespace with env variables [#8222](https://github.com/GoogleContainerTools/skaffold/pull/8222)
* fix: properly wire deploy.kubectl.defaultNamespace field to be set in SKAFFOLD_NAMESPACES [#8129](https://github.com/GoogleContainerTools/skaffold/pull/8129)
* fix: new condition to create hydrate-dir only if a kpt renderer or deployer [#8117](https://github.com/GoogleContainerTools/skaffold/pull/8117)
* fix: correct issue where skaffold setTemplateValues env vars were in some cases empty [#8261](https://github.com/GoogleContainerTools/skaffold/pull/8261)

# v2.0.3 Release - 12/01/2022
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.3/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.3/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.3/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.3/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.0.3/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.0.3`

Fixes:
* fix: support alternative env var naming using support env variable + artifact-name (vs env variable + index) [#8175](https://github.com/GoogleContainerTools/skaffold/pull/8175)

# v2.0.2 Release - 11/15/2022
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.2/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.2/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.2/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.2/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.0.2/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.0.2`

Note: This release comes with a new config version, `v4beta1`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

New Features and Additions:
* feat: add --tolerate-failures-until-deadline flag and deploy.tolerateFailuresUntilDeadline config for improved ci/cd usage [#8047](https://github.com/GoogleContainerTools/skaffold/pull/8047)
* feat: Add an example e2e test environment / e2e tests for GRPC service. [#7932](https://github.com/GoogleContainerTools/skaffold/pull/7932)
* feat: add skipTests to ignore helm test directory on manifest generation [#8011](https://github.com/GoogleContainerTools/skaffold/pull/8011)
* feat: add support for deploying Cloud Run Jobs. [#7915](https://github.com/GoogleContainerTools/skaffold/pull/7915)
* feat: context copy retry. If we add a for loop and execute the kubectlcli.Run method for copying context to kaniko pod , this makes more reliable and less prune to network failures [#7887](https://github.com/GoogleContainerTools/skaffold/pull/7887)
* feat: new cloudbuild slim [#8004](https://github.com/GoogleContainerTools/skaffold/pull/8004)
* feat: new dockerfiles and cloudbuild config for slim image [#7996](https://github.com/GoogleContainerTools/skaffold/pull/7996)
* feat: completion for the Fish shell [#8097](https://github.com/GoogleContainerTools/skaffold/pull/8097)

Fixes:
* fix: add description to StatusCheck TaskEvent [#8017](https://github.com/GoogleContainerTools/skaffold/pull/8017)
* fix: Avoid nil context error [#8038](https://github.com/GoogleContainerTools/skaffold/pull/8038)
* fix: cleanup not called when using helm deployer alone [#8040](https://github.com/GoogleContainerTools/skaffold/pull/8040)
* fix: container image push to local cluster [#8007](https://github.com/GoogleContainerTools/skaffold/pull/8007)
* fix: log duplication when using dependencies [#8042](https://github.com/GoogleContainerTools/skaffold/pull/8042)
* fix: not print error message if it is empty [#8005](https://github.com/GoogleContainerTools/skaffold/pull/8005)
* fix: preserve configs order when creating renderers and deployers [#8028](https://github.com/GoogleContainerTools/skaffold/pull/8028)
* fix: properly add RemoteManifests support to skaffold v2 [#8036](https://github.com/GoogleContainerTools/skaffold/pull/8036)
* fix: use std lib signal handling [#8046](https://github.com/GoogleContainerTools/skaffold/pull/8046)
* fix(sec): upgrade runc version v1.0.2 -> v1.1.2 [#8050](https://github.com/GoogleContainerTools/skaffold/pull/8050)
* fix: add unique tag to test image to avoid collisions with other tests [#8087](https://github.com/GoogleContainerTools/skaffold/pull/8087)
* fix: correct issues with current upgrade logic for artifactOverrides with helm imageStrategy [#8066](https://github.com/GoogleContainerTools/skaffold/pull/8066)
* fix: override protocols argument pass to helm post-renderer [#8083](https://github.com/GoogleContainerTools/skaffold/pull/8083)
* fix: resolve issue where skaffold filter command did not properly configure the filter allow & deny lists [#8085](https://github.com/GoogleContainerTools/skaffold/pull/8085)

Updates and Refactors:
* chore: add v1.39.3 release CHANGELOG.md entry [#7991](https://github.com/GoogleContainerTools/skaffold/pull/7991)
* chore: bump skaffold schema version to v4beta1 [#8034](https://github.com/GoogleContainerTools/skaffold/pull/8034)
* chore: make syncstore generic [#8000](https://github.com/GoogleContainerTools/skaffold/pull/8000)
* chore: update examples/getting-started to go 1.19 [#8043](https://github.com/GoogleContainerTools/skaffold/pull/8043)
* chore: update workflow files [#8001](https://github.com/GoogleContainerTools/skaffold/pull/8001)
* chore: upgrade jib plugin versions to 3.3.1 [#8003](https://github.com/GoogleContainerTools/skaffold/pull/8003)
* chore(deps): bump some .github/workflows deps [#8051](https://github.com/GoogleContainerTools/skaffold/pull/8051)
* chore(doc): note that filesync works for debug [#8044](https://github.com/GoogleContainerTools/skaffold/pull/8044)
* chore: reduce gcp integration test time [#8080](https://github.com/GoogleContainerTools/skaffold/pull/8080)
* chore: remove uncessary server bins [#8092](https://github.com/GoogleContainerTools/skaffold/pull/8092)
* refactor: use exclude directives instead of replace directives for pinning [#8056](https://github.com/GoogleContainerTools/skaffold/pull/8056)

Docs, Test, and Release Updates:
* docs: Linked to the Google Cloud Solutions Template [#8054](https://github.com/GoogleContainerTools/skaffold/pull/8054)
* docs: migrate v2 docs -> skaffold.dev and v1 docs -> skaffold-v1.web.app [#7966](https://github.com/GoogleContainerTools/skaffold/pull/7966)
* docs: update skaffold examples and documentation to properly reflect v1 ->v2 artifactOverrides changes - 2nd attempt [#8019](https://github.com/GoogleContainerTools/skaffold/pull/8019)
* docs: update skaffold examples and documentation to properly reflect v1 ->v2 artifactOverrides changes [#8013](https://github.com/GoogleContainerTools/skaffold/pull/8013)
* docs: update skaffold.gliffy diagram with updated information for skaffold v2 [#8035](https://github.com/GoogleContainerTools/skaffold/pull/8035)
* docs: add detailed information on how to use helm rendering with v2.X.Y as well as how post-renderer usage works [#8093](https://github.com/GoogleContainerTools/skaffold/pull/8093)

Huge thanks goes out to all of our contributors for this release:

- Aaron Prindle
- agarwalnit
- b4nks
- Benjamin Kaplan
- Brian de Alwis
- Bryan C. Mills
- dependabot[bot]
- Emily Wang
- ericzzzzzzz
- Gaurav
- Imre Nagi
- Jeremy Tymes
- Julian Lawrence
- Justin Santa Barbara
- Maggie Neterval
- Michele Sorcinelli
- Renzo Rojas
- Santiago Nuñez-Cacho
- Sergei Kononov
- Sergei Morozov
- Steven Powell
- techchickk
- Tomás Mota

# v1.39.4 Release - 11/11/2022
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.39.4/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.39.4/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.39.4/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.39.4/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v1.39.4/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.39.4`

Fixes:
* fix: resolve issue where skaffold apply didn't respect config - statusCheck:false (#8077)

Updates and Refactors:
* chore: update skaffold deps image dependencies (#8067)

# v2.0.1 Release - 10/27/2022
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.1/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.1/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.1/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.1/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.0.1/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.0.1`

Fixes:
* fix: stderr of "helm template" not printed [#7986](https://github.com/GoogleContainerTools/skaffold/pull/7986)
* fix: revert "fix: no longer pass os env vars through to verify as thiss can cause unexpected issues with PATH, GOPATH, etc (#7949)[#7998](https://github.com/GoogleContainerTools/skaffold/pull/7998)

Updates and Refactors:
* chore: bump examples/* to apiVersion: skaffold/v3 [#7970](https://github.com/GoogleContainerTools/skaffold/pull/7970)
* chore: bump version deps [#7969](https://github.com/GoogleContainerTools/skaffold/pull/7969)
* chore: set v3 schema as the first schema shown on v2 doc site [#7982](https://github.com/GoogleContainerTools/skaffold/pull/7982)
* chore(deps): bump actions/upload-artifact from 3.1.0 to 3.1.1 [#7973](https://github.com/GoogleContainerTools/skaffold/pull/7973)
* chore(deps): bump peter-evans/create-or-update-comment from 2.0.1 to 2.1.0 [#7974](https://github.com/GoogleContainerTools/skaffold/pull/7974)

Docs, Test, and Release Updates:
* docs: Add a copy of the buildpacks node tutorial without a skaffold.yaml [#7968](https://github.com/GoogleContainerTools/skaffold/pull/7968)
* docs: Fix helm command [#7987](https://github.com/GoogleContainerTools/skaffold/pull/7987)

# v1.39.3 Release - 10/26/2022
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.39.3/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.39.3/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.39.3/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.39.3/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v1.39.3/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.39.3`

Fixes:
* fix: backport Bazel build context fixes to v1 (#7866)
* fix: update integration tests

Updates and Refactors:
* chore: update v1 branch to go1.19.1

# v2.0.0 Release - 10/20/2022
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.0/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.0/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.0.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.0.0`

Note: This release comes with a new config version, `v3`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Skaffold v2.0.0 is officially released today 🎊!  Users can try it out by using the updated Skaffold [installation guide](https://skaffold-v2.web.app/docs/install/).  A critical requirement for Skaffold v2 is that all of your existing Skaffold configurations continue to just work with Skaffold v2.  This means that for the vast majority of users, the new features below should be immediately accessible requiring no additional configuration changes.  For additional information on what has changed from Skaffold v1 to v2 and what specific options might require some manual tweaking, please refer to the upgrade guide.

Highlights & New Features:
* 💻 Support for deploying to ARM, X86 or Multi-Arch K8s clusters from your x86 or ARM machine 
* 👟 New Cloud Run Deployer brings the power of Skaffold to Google Clouds serverless container runtime 
* 📜 Skaffold render phase has been split from deploy phase providing increased granularity of control for GitOps workflows 
* 🚦New Skaffold verify phase enables improved testing capabilities making Skaffold even better as a CI/CD tool
* ⚙️ Tighter integration with kpt lets you more dynamically manage large amounts of configuration and keep it in sync 

Note: the below PRs only reflect the changes from v2.0.0-beta3 to the v2.0.0 GA release.  To see all of the PRs related to the v1.39.2 -> v2.0.0 transition see the Release Notes for  [v2.0.0-beta1](https://github.com/GoogleContainerTools/skaffold/releases/tag/v2.0.0-beta1),  [v2.0.0-beta2](https://github.com/GoogleContainerTools/skaffold/releases/tag/v2.0.0-beta2), and  [v2.0.0-beta3](https://github.com/GoogleContainerTools/skaffold/releases/tag/v2.0.0-beta3)

Fixes:
* fix: add inputDigest tagger alias to customTemplate tagger [#7867](https://github.com/GoogleContainerTools/skaffold/pull/7867)
* fix: allow `init` without `render` and `deploy` [#7936](https://github.com/GoogleContainerTools/skaffold/pull/7936)
* fix: Cloud Code log viewer shows render stage correctly [#7937](https://github.com/GoogleContainerTools/skaffold/pull/7937)
* fix: deploy integration test image builds need to be multi-platform since we're testing against 3 types of k8s cluster [#7882](https://github.com/GoogleContainerTools/skaffold/pull/7882)
* fix: ensure project descriptor in Pack Build [#7884](https://github.com/GoogleContainerTools/skaffold/pull/7884)
* fix: images being pushed with same tags overwriting each other [#7900](https://github.com/GoogleContainerTools/skaffold/pull/7900)
* fix: kpt render with subdirectories [#7909](https://github.com/GoogleContainerTools/skaffold/pull/7909)
* fix: no longer pass os env vars through to verify as this can cause unexpected issues with PATH, GOPATH, etc. [#7949](https://github.com/GoogleContainerTools/skaffold/pull/7949)
* fix: resolve issue where helm render did not respect --namespace flag [#7907](https://github.com/GoogleContainerTools/skaffold/pull/7907)
* fix: reverse order of deployers during cleanup (#7284) [#7925](https://github.com/GoogleContainerTools/skaffold/pull/7925)
* fix: use relref shortcode to fix links [#7890](https://github.com/GoogleContainerTools/skaffold/pull/7890)

Updates and Refactors:
* chore: Add an example for helm chart with multiple images [#7874](https://github.com/GoogleContainerTools/skaffold/pull/7874)
* chore: add generic `util.Ptr` function [#7961](https://github.com/GoogleContainerTools/skaffold/pull/7961)
* chore: change survey prompt frequency [#7912](https://github.com/GoogleContainerTools/skaffold/pull/7912)
* chore: create skaffold/v3 schema [#7960](https://github.com/GoogleContainerTools/skaffold/pull/7960)
* chore: enable platform flag in render command  [#7885](https://github.com/GoogleContainerTools/skaffold/pull/7885)
* chore: move deploy error codes to render error codes [#7893](https://github.com/GoogleContainerTools/skaffold/pull/7893)
* chore: update cross and multi-platform build maturity [#7928](https://github.com/GoogleContainerTools/skaffold/pull/7928)
* chore: update Google API client libraries [#7903](https://github.com/GoogleContainerTools/skaffold/pull/7903)
* chore: update project to use go 1.19.1  [#7871](https://github.com/GoogleContainerTools/skaffold/pull/7871)
* chore: update skaffold Dockerfiles to pull from approved GCS buckets vs open internet [#7921](https://github.com/GoogleContainerTools/skaffold/pull/7921)
* chore: update skaffold v2 docs to point reference v2.0.0 and not v2.0.0-beta* [#7948](https://github.com/GoogleContainerTools/skaffold/pull/7948)
* chore: update skaffold verify to GA maturity [#7950](https://github.com/GoogleContainerTools/skaffold/pull/7950)
* chore: update v1 branch to go1.19.1 [#7943](https://github.com/GoogleContainerTools/skaffold/pull/7943)
* chore: use cmp.Diff to check for differences [#7896](https://github.com/GoogleContainerTools/skaffold/pull/7896)
* chore(deps): bump ossf/scorecard-action from 2.0.3 to 2.0.4 [#7898](https://github.com/GoogleContainerTools/skaffold/pull/7898)
* chore(deps): bump peter-evans/create-or-update-comment from 2.0.0 to 2.0.1 [#7946](https://github.com/GoogleContainerTools/skaffold/pull/7946)
* chore(examples): update examples to Spring Boot 2.7.4 and SnakeYAML 1.32 [#7895](https://github.com/GoogleContainerTools/skaffold/pull/7895)


Docs, Test, and Release Updates:
* docs: Add banner to v1 and v2 docs indicating that v1 is archived [#7920](https://github.com/GoogleContainerTools/skaffold/pull/7920)
* docs: add cloud deploy info to skaffold ci-cd docs [#7880](https://github.com/GoogleContainerTools/skaffold/pull/7880)
* docs: Add examples to the info sidebar [#7906](https://github.com/GoogleContainerTools/skaffold/pull/7906)
* docs: Add new inputDigest tagger alias to customTemplate tagger [#7939](https://github.com/GoogleContainerTools/skaffold/pull/7939)
* docs: Add version menu label [#7917](https://github.com/GoogleContainerTools/skaffold/pull/7917)
* docs: Beta3 docs [#7879](https://github.com/GoogleContainerTools/skaffold/pull/7879)
* docs: Fix v1 and v2 docs links [#7931](https://github.com/GoogleContainerTools/skaffold/pull/7931)
* docs: Update _index.md [#7951](https://github.com/GoogleContainerTools/skaffold/pull/7951)
* docs: update skaffold v2 intro text to have correct expanded scope [#7953](https://github.com/GoogleContainerTools/skaffold/pull/7953)
* docs: update v1 and v2 docs to properly reflect recent tagger changes [#7952](https://github.com/GoogleContainerTools/skaffold/pull/7952)
* docs(ko): Design proposal: Hot reloading in dev [#7888](https://github.com/GoogleContainerTools/skaffold/pull/7888)
* test: Add multi platform integration tests [#7852](https://github.com/GoogleContainerTools/skaffold/pull/7852)

# v2.0.0-beta3 Release - 09/21/2022
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.0-beta3/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.0-beta3/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.0-beta3/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.0-beta3/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.0.0-beta3/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.0.0-beta3`

Note: This release comes with a new config version, `v3alpha1`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Skaffold’s v2 beta release `v2.0.0-beta3` is out today!  To try it out, you can go to our [v2.0.0-beta3 installation guide](https://skaffold-v2.web.app/docs/install/) or use the instructions above.  For information on migrating from skaffold v1.X.Y to skaffold v2.0.0-beta3 see our [upgrade guide here](https://skaffold-v2.web.app/docs/upgrading-to-v2/).  TLDR; there should not be any changes required for most use cases, see the [upgrade guide here](https://skaffold-v2.web.app/docs/upgrading-to-v2/) for the full details.  

Highlights:
See Skaffold's [v2.0.0-beta2 Release Notes](https://github.com/GoogleContainerTools/skaffold/releases/tag/v2.0.0-beta2) "Highlights" section for additional information on the large feature additions and changes Skaffold V2 brings.  Below will be the incremental highlights from v2.0.0-beta2 to v2.0.0-beta3
* Added `serviceAccount` field to Skaffold's `cloudbuild` configuration
* Fixed issues with using `manifests.hooks` with multiple skaffold modules
* Added additional documentation regarding Skaffold's new `manifests.transform` and `manifests.validate` fields in our kpt documentation
* Added remote image support for `skaffold verify`

New Features and Additions:
* add: added service account to cloud build config [#7843](https://github.com/GoogleContainerTools/skaffold/pull/7843)

Fixes:
* fix: backport Bazel build context fixes to v1 [#7866](https://github.com/GoogleContainerTools/skaffold/pull/7866)
* fix: remove RenderConfig and fix manifests issues with multi-config [#7877](https://github.com/GoogleContainerTools/skaffold/pull/7877)
* fix: adjust set namespace logic [#7841](https://github.com/GoogleContainerTools/skaffold/pull/7841)
* fix: remove Cloud Run location validation for non-remote commands [#7847](https://github.com/GoogleContainerTools/skaffold/pull/7847)
* fix: add remote image support to verify [#7835](https://github.com/GoogleContainerTools/skaffold/pull/7835)

Updates and Refactors:
* chore: skip buildpacks integration test if being run on arm cluster [#7869](https://github.com/GoogleContainerTools/skaffold/pull/7869)
* chore: update Ingress manifests [#7861](https://github.com/GoogleContainerTools/skaffold/pull/7861)
* chore(deps): bump ossf/scorecard-action from 2.0.0 to 2.0.3 [#7858](https://github.com/GoogleContainerTools/skaffold/pull/7858)
* chore: remove unnecessary rerender from devloop, [#7845](https://github.com/GoogleContainerTools/skaffold/pull/7845)
* chore: change `jib.from.image` to multi-arch image [#7849](https://github.com/GoogleContainerTools/skaffold/pull/7849)
* chore(deps): bump ossf/scorecard-action from 1.1.2 to 2.0.0 [#7853](https://github.com/GoogleContainerTools/skaffold/pull/7853)
* chore: adding status check phase with v2 taskEvent [#7846](https://github.com/GoogleContainerTools/skaffold/pull/7846)
* chore: transform installation of go-licenses from go-get to go install [#7794](https://github.com/GoogleContainerTools/skaffold/pull/7794)
* chore: upgrade jib versions to 3.3.0 [#7831](https://github.com/GoogleContainerTools/skaffold/pull/7831)
* chore: add render lifecycle hook highlight to changelog [#7829](https://github.com/GoogleContainerTools/skaffold/pull/7829

Docs, Test, and Release Updates:
* docs: fix small typo in v2.0.0-beta2 release changelog [#7832](https://github.com/GoogleContainerTools/skaffold/pull/7832))
* docs: update v2 docs to have up to date v2.0.0-beta2 links [#7828](https://github.com/GoogleContainerTools/skaffold/pull/7828)
* docs: Fix incorrect default set for project for jib [#7857](https://github.com/GoogleContainerTools/skaffold/pull/7857)
* docs: add kpt manifest.transform|validate functionality examples [#7870](https://github.com/GoogleContainerTools/skaffold/pull/7870)

Huge thanks goes out to all of our contributors for this release:

- Aaron Prindle
- Benjamin Kaplan
- Boran Seref
- dependabot[bot]
- ericzzzzzzz
- Gaurav
- Mridula
- Oladapo Ajala
- Renzo Rojas
- Tejal Desai


# v2.0.0-beta2 Release - 08/30/2022
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.0-beta2/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.0-beta2/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.0-beta2/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.0-beta2/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.0.0-beta2/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.0.0-beta2`

Note: This release comes with a new config version, `v3alpha1`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Skaffold’s v2 beta release `v2.0.0-beta2` is out today!  To try it out, you can go to our [v2.0.0-beta2 installation guide](https://skaffold-v2.web.app/docs/install/) or use the instructions above.  For information on migrating from skaffold v1.X.Y to skaffold v2.0.0-beta2 see our [upgrade guide here](https://skaffold-v2.web.app/docs/upgrading-to-v2/).  TLDR; there should not be any changes required for most use cases, see the [upgrade guide here](https://skaffold-v2.web.app/docs/upgrading-to-v2/) for the full details.  

Highlights:

See Skaffold's [v2.0.0-beta1 Release Notes](https://github.com/GoogleContainerTools/skaffold/releases/tag/v2.0.0-beta1) "Highlights" section for additional information on the large feature additions and changes Skaffold V2 brings.  Below will be the incremental highlights from v2.0.0-beta1 to v2.0.0-beta2
* cross-platform AND multi-platform support for Skaffold is now feature complete!  See the [cross-platform and multi-platform docs](https://skaffold-v2-latest.web.app/docs/pipeline-stages/builders/#cross-platform-and-multi-platform-build-support/) and the [Managing ARM Workloads docs](https://skaffold-v2.web.app/docs/workflows/handling-platforms/) for the full information on the feature.  See also the [sample skaffold.yaml & app](https://github.com/GoogleContainerTools/skaffold/blob/main/integration/examples/cross-platform-builds/skaffold.yaml) here.
* skaffold render phase now supports lifecycle hooks
* skaffold verify now supports profiles
* skaffold init functionality for helm applications has increased support for Skaffold V2 - apiVersion:skaffold/v3alpha1


New Features and Additions:
* feat: add profile support to skaffold verify [#7807](https://github.com/GoogleContainerTools/skaffold/pull/7807)
* feat: upgrade profiles when running skaffold fix [#7800](https://github.com/GoogleContainerTools/skaffold/pull/7800)
* feat: add hooks support to render schema and phase [#7785](https://github.com/GoogleContainerTools/skaffold/pull/7785)

Fixes:
* fix: helm template warnings breaking yaml parsing [#7825](https://github.com/GoogleContainerTools/skaffold/pull/7825)
* fix: Remove the flag from deploy command [#7823](https://github.com/GoogleContainerTools/skaffold/pull/7823)
* fix: Fill IMAGE_TAG,etc on Docker builds [#7788](https://github.com/GoogleContainerTools/skaffold/pull/7788)
* chore: Add integration test for build cross platform images [#7818](https://github.com/GoogleContainerTools/skaffold/pull/7818)
* feat: add toleration for GKE ARM nodes taint [#7789](https://github.com/GoogleContainerTools/skaffold/pull/7789)
* fix: add auth to image pull [#7814](https://github.com/GoogleContainerTools/skaffold/pull/7814)
* fix: check platform values for known OS and Arch, and fail fast [#7817](https://github.com/GoogleContainerTools/skaffold/pull/7817)
* fix: add host platform metric to platform type events [#7821](https://github.com/GoogleContainerTools/skaffold/pull/7821)
* fix: make helm renderer to use manifest.ManifestList  [#7795](https://github.com/GoogleContainerTools/skaffold/pull/7795)
* fix: add node unschedulable to retriable errors [#7798](https://github.com/GoogleContainerTools/skaffold/pull/7798)
* fix: remove required tag from cloud run region [#7784](https://github.com/GoogleContainerTools/skaffold/pull/7784)
* fix: added apply to --rpc-port and --rpc-http-port definedons [#7799](https://github.com/GoogleContainerTools/skaffold/pull/7799)

Updates and Refactors:
* chore: raise error when there is a multiplatform build without container registry [#7786](https://github.com/GoogleContainerTools/skaffold/pull/7786)
* chore: safe type assersions [#7770](https://github.com/GoogleContainerTools/skaffold/pull/7770)
* chore: add-render-event-in-runner [#7781](https://github.com/GoogleContainerTools/skaffold/pull/7781)
* refactor: remove v1 runner interfaces and simplify code  [#7724](https://github.com/GoogleContainerTools/skaffold/pull/7724)

Docs, Test, and Release Updates:
fix: improve multi-arch docs [#7767](https://github.com/GoogleContainerTools/skaffold/pull/7767)

Huge thanks goes out to all of our contributors for this release:

- 조태혁
- Aaron Prindle
- Benjamin Kaplan
- dependabot[bot]
- dhodun
- ericzzzzzzz
- Gaurav
- Halvard Skogsrud
- Karolína Lišková
- Oladapo Ajala
- Pablo Caderno
- Renzo Rojas
- Tejal Desai
# v2.0.0-beta1 Release - 08/03/2022
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.0-beta1/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.0-beta1/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.0-beta1/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.0-beta1/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.0.0-beta1/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.0.0-beta1`

Note: This release comes with a new config version, `v3alpha1`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Skaffold’s v2 beta release `v2.0.0-beta1` is out today!  To try it out, you can go to our [v2.0.0-beta1 installation guide](https://skaffold-v2.web.app/docs/install/) or use the instructions above.  For information on migrating from skaffold v1.X.Y to skaffold v2.0.0-beta1 see our [upgrade guide here](https://skaffold-v2.web.app/docs/upgrading-to-v2/).  TLDR; there should not be any changes required for most use cases, see the [upgrade guide here](https://skaffold-v2.web.app/docs/upgrading-to-v2/) for the full details.  


Highlights:

In Skaffold v2 what was previously the `deploy` phase of Skaffold is now split into a new `render` phase and `deploy` phase.  This clear boundary of separation between render and deploy phases allowed the team to simplify our code and CLI allowing us to clean up previously confusing or redundant flags like `skaffold deploy --render-only`, `skaffold deploy --skip-render`. 
This release comes with a new schema version `v3alpha1`. This schema introduced a new `manifests` section which declares all resources an application deploys e.g helm charts, kubernetes yaml, kustomize directories and kpt configuration. This decoupling of manifests declaration from the deploy section allows manifests to be used across deploy tools e.g.
* you can configure kpt deployer to render and apply kubernetes yaml, helm charts or
* you can configure the kubectl deployer to apply helm charts and helm to render the charts. 
See the [upgrade guide here](https://skaffold-v2.web.app/docs/upgrading-to-v2/) for more information.


New Features and Additions:
* More advanced and configurable `render` functionality.  See the render docs [here](https://skaffold-v2.web.app/docs/pipeline-stages/renderers/) for more details.
    * Skaffold v2 supports rendering, validating and transforming manifests with kpt v1.0.0-beta. See the kpt renderer docs [here](https://skaffold-v2.web.app/docs/pipeline-stages/renderers/kpt/) for more details.
    * Please take a look at the [kpt v0.39.0 to kpt v1.0.0 migration guide](https://kpt.dev/installation/migration) for more information
* Better rendering support for applications with helm charts. See the helm renderer docs [here](https://skaffold-v2.web.app/docs/pipeline-stages/renderers/helm/) for more details.
    * skaffold under the hood uses `helm template` to render helm charts now
* New `skaffold verify` command which allows for Skaffold to run test containers against skaffold deployments.  See the verify docs [here](https://skaffold-v2.web.app/docs/pipeline-stages/verify/) for full set of details and examples. With this users can use off the shelf (`alpine`, etc.) or skaffold built test containers and run them in a pipeline that skaffold watches and streams logs from. Supports customer requested CI/CD functionality for a Skaffold native way of supporting:
    * health checks (deployment success, readiness)
    * integration/smoke/load tests
    * monitoring checks(metrics and alerting)
* `Cloud Run` deployer support (in alpha stage).  Currently supports deploying applications, port-forwarding and eventing but not log streaming or `debug` support which will be added soon. See the docs [here](https://skaffold-v2.web.app/docs/pipeline-stages/deployers/cloudrun/) for more information

Docs, Test, and Release Updates:
* For the skaffold v2 beta period, there will be a temporary v2 docs site hosted at [https://skaffold-v2.web.app/](https://skaffold-v2.web.app/) which explains all changes, new features, and potential regressions.S

This release marks the culmination of lots of hard work across multiple teams, organizations, and external contributors.  Special thanks to @yuwenma and @bksaplan for all of their work in getting important v2 functionality across the finish line!  Huge thanks goes out to all of our contributors for this release:

- Aaron Prindle
- Aaron Son
- Abhinav Nair
- Alessandro Ros
- Alex
- beast
- betaboon
- Brian de Alwis
- bskaplan
- CuriousCorrelation
- dependabot[bot]
- ericzzzzzzz
- Flávio Roberto Santos
- GagarinX
- Gaurav
- Halvard Skogsrud
- Ikumi Nakamura
- Javier Cañadillas
- Kourtney
- Lukas
- Marlon Gamez
- Naofumi MURATA
- neilnaveen
- Nelson Chen
- Pablo Caderno
- piotrostr
- Renzo Rojas
- Sergei Morozov
- Shabir Mohamed Abdul Samadh
- Shuhei Kitagawa
- Suzuki Shota
- Tejal Desai
- Tomás Mota
- Tuan Anh Pham
- Yuwen Ma

# v1.39.1 Release - 06/28/2022
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.39.1/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.39.1/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.39.1/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.39.1/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v1.39.1/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.39.1`

Note: This release is patch to v1.39.0 which fixes a helm debug issue for client using Helm 3.1 and above; and fixes an index out of range error seen in some cases.

Fixes:
* Revert Helm 3.0 and Helm 3.1 Deployer changes [#7582](https://github.com/GoogleContainerTools/skaffold/issues/7582)
* fix: index out of range error [#7593](https://github.com/GoogleContainerTools/skaffold/pull/7593)

# v1.39.0 Release - 06/23/2022
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.39.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.39.0/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.39.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.39.0/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v1.39.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.39.0`

Note: This release comes with a new config version, `v2beta29`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:

New Features and Additions:
* feat: add `hostPlatform` and `targetPlatforms` to `v1` and `v2` skaffold events.  [#7559](https://github.com/GoogleContainerTools/skaffold/pull/7559)
* feat: Add privateWorkerPool and location configuration for gcb [#7440](https://github.com/GoogleContainerTools/skaffold/pull/7440)

* Fixes:
* fix: add panic fix and recovery logic to reflection for yaml line number info [#7577](https://github.com/GoogleContainerTools/skaffold/pull/7577)
* fix: add default value for status-check flag when no value is specified [7278](https://github.com/GoogleContainerTools/skaffold/pull/7278)
* fix: fix kubectl result formatting for debug logs [#7293](https://github.com/GoogleContainerTools/skaffold/pull/7293)
* fix: change error to warning for build platform [#7402](https://github.com/GoogleContainerTools/skaffold/pull/7402)

Docs, Test, and Release Updates:
* doc: remove image build from the render command decription [#7569](https://github.com/GoogleContainerTools/skaffold/pull/7569)

Huge thanks goes out to all of our contributors for this release:
- Aaron Prindle
- Brian de Alwis
- Gaurav
- Marlon Gamez
- Renzo Rojas
- Tejal Desai
- ericzzzzzzz
- neilnaveen


# v1.37.2 Release - 04/29/2022
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.37.2/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.37.2/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.37.2/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.37.2/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v1.37.2/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.37.2`

Fixes:
* fix: cloud build cannot find private workerpool issue [#7356](https://github.com/GoogleContainerTools/skaffold/pull/7356)
* fix: properly update 'label' field for helm + render w/ -l flag [#7349](https://github.com/GoogleContainerTools/skaffold/pull/7349)

# v1.38.0 Release - 04/06/2022
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.38.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.38.0/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.38.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.38.0/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v1.38.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.38.0`

Fixes:
* fix: fix bazel-out path for `rules_docker-0.23.0` [#7251](https://github.com/GoogleContainerTools/skaffold/pull/7251)
* fix: fix skaffold label setter to work properly for cnrm resources [#7243](https://github.com/GoogleContainerTools/skaffold/pull/7243)
* fix: change GCB backoff check to use error code instead of checking the error [#7213](https://github.com/GoogleContainerTools/skaffold/pull/7213)
* fix(examples/typescript): use tsc-watch --noClear to keep previous logs [#7227](https://github.com/GoogleContainerTools/skaffold/pull/7227)
* fix: Unmarshaling nested arrays of objects [#7217](https://github.com/GoogleContainerTools/skaffold/pull/7217)
* fix: add missing Argo CronWorkflow to transforms [#7205](https://github.com/GoogleContainerTools/skaffold/pull/7205)

Updates and Refactors:
* chore: remove maven-wrapper.jar and regenerate Maven wrappers [#7220](https://github.com/GoogleContainerTools/skaffold/pull/7220)
* chore(ko): Upgrade `ko` dependency to v0.11.2 [#7224](https://github.com/GoogleContainerTools/skaffold/pull/7224)

Docs, Test, and Release Updates:
* ci: pass `IT_PARTITION` variable when running `make integration-in-docker` [#7226](https://github.com/GoogleContainerTools/skaffold/pull/7226)
* docs: add skaffold.yaml in `docs/content/en/samples` for pipeline-stages [#7077](https://github.com/GoogleContainerTools/skaffold/pull/7077)
* docs: document command-line restrictions for Go and Python [#7260](https://github.com/GoogleContainerTools/skaffold/pull/7260)
* chore: update go version in actions that weren't specifying go 1.17 [#7263](https://github.com/GoogleContainerTools/skaffold/pull/7263)
* chore: update Jib plugin versions to 3.2.1 in examples [#7256](https://github.com/GoogleContainerTools/skaffold/pull/7256)

Huge thanks goes out to all of our contributors for this release:

- Aaron Prindle
- Brian de Alwis
- Elena Felder
- Halvard Skogsrud
- Kevin Hanselman
- Marlon Gamez
- Mohammad Sadegh Salimi
- Sam Gomena
- Sasha Morrissey
- barp

# v1.37.1 Release - 03/30/2022
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.37.1/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.37.1/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.37.1/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.37.1/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v1.37.1/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.37.1`

Fixes:
* fix: fix skaffold label setter to work properly for cnrm resources [#7221](https://github.com/GoogleContainerTools/skaffold/pull/7221)

# v1.37.0 Release - 03/16/2022
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.37.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.37.0/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.37.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.37.0/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v1.37.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.37.0`

Note: This release comes with a new config version, `v2beta28`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:
* New Resource selector to allow or deny transforming specific resources in deploy stage [#7056](https://github.com/GoogleContainerTools/skaffold/pull/7056)
* Cross-platform support for kaniko builder [#7142](https://github.com/GoogleContainerTools/skaffold/pull/7142)
* Cross-platform support for GCB build env [#7134](https://github.com/GoogleContainerTools/skaffold/pull/7134)
* Cross-platform support for jib builder [#7110](https://github.com/GoogleContainerTools/skaffold/pull/7110)

New Features and Additions:
* feat: add basic metrics for resource selector usage [#7192](https://github.com/GoogleContainerTools/skaffold/pull/7192)
* feat: Allow [Confluent-for-Kubernetes](https://docs.confluent.io/operator/current/overview.html?_ga=2.137619952.1165154847.1647462431-157638067.1647462431&_gac=1.194052959.1647462431.CjwKCAjwlcaRBhBYEiwAK341jVVmaAbOOVWBcuYdjPp3PtwfEbiz48zXgui9hhn5stfT-G9JULvLuBoCZOIQAvD_BwE) types to be transformed [#7179](https://github.com/GoogleContainerTools/skaffold/pull/7179)
* feat: add metrics related to cross-platform build [#7172](https://github.com/GoogleContainerTools/skaffold/pull/7172)
* feat: add better support and messaging around using helm with skaffold apply [#7149](https://github.com/GoogleContainerTools/skaffold/pull/7149)
* feat(v2): Add kubectl renderer [#7118](https://github.com/GoogleContainerTools/skaffold/pull/7118)
* feat(lsp): add validation checking to lsp [#7097](https://github.com/GoogleContainerTools/skaffold/pull/7097)
* feat: add argo workflows kinds to transformable allow list [#7102](https://github.com/GoogleContainerTools/skaffold/pull/7102)

Fixes:
* fix: ignore concurrency if not specified [#7182](https://github.com/GoogleContainerTools/skaffold/pull/7182)
* fix(v2): TestDockerDebug failing #7170 [#7189](https://github.com/GoogleContainerTools/skaffold/pull/7189)
* fix(v2): TestInspectBuildEnv failing [#7170](https://github.com/GoogleContainerTools/skaffold/pull/7170)
* fix: `make install` works for Mac M1 [#7159](https://github.com/GoogleContainerTools/skaffold/pull/7159)
* fix: choose cli default-repo over config file [#7144](https://github.com/GoogleContainerTools/skaffold/pull/7144)
* fix: warn instead of fail for multi-arch builds [#7145](https://github.com/GoogleContainerTools/skaffold/pull/7145)
* fix: specifying platforms in ko builder [#7135](https://github.com/GoogleContainerTools/skaffold/pull/7135)
* fix: Typo in warning message [#7138](https://github.com/GoogleContainerTools/skaffold/pull/7138)
* fix: `make preview-docs` should run for Mac M1 [#7136](https://github.com/GoogleContainerTools/skaffold/pull/7136)
* fix: correctly handle excluded profiles [#7107](https://github.com/GoogleContainerTools/skaffold/pull/7107)
* fix:  skaffold's assumption for image tag when building via buildkit and custom output [#7120](https://github.com/GoogleContainerTools/skaffold/pull/7120)
* fix: always reset cached repo if sync is true [#7069](https://github.com/GoogleContainerTools/skaffold/pull/7069)
* fix: parsing alternative config filename `skaffold.yml` by supporting absolute paths in `config.ReadConfiguration` [#7112](https://github.com/GoogleContainerTools/skaffold/pull/7112)
* fix: correctly set the default value of `StatusCheck` to nil [#7089](https://github.com/GoogleContainerTools/skaffold/pull/7089)

Updates and Refactors:
* refactor: Use new logrus.Logger instead of default [#7193](https://github.com/GoogleContainerTools/skaffold/pull/7193)
* chore(deps): bump github.com/containerd/containerd from 1.5.8 to 1.5.9 [#7151](https://github.com/GoogleContainerTools/skaffold/pull/7151)
* chore(deps): bump actions/checkout from 2 to 3 [#7150](https://github.com/GoogleContainerTools/skaffold/pull/7150)
* chore: upgrade to helm 3.8.0 for experimental oci support [#7147](https://github.com/GoogleContainerTools/skaffold/pull/7147)
* chore(deps): bump github.com/docker/distribution from 2.7.1+incompatible to 2.8.0+incompatible [#7105](https://github.com/GoogleContainerTools/skaffold/pull/7105)
* chore(deps): bump puma from 4.3.9 to 4.3.11 in ruby example [#7117](https://github.com/GoogleContainerTools/skaffold/pull/7117)
* chore(deps): bump flask version from 2.0.2 to 2.0.3 in buildpacks-python example [#7116](https://github.com/GoogleContainerTools/skaffold/pull/7116)
* refactor(v2): remove pointers from render code.  [#7109](https://github.com/GoogleContainerTools/skaffold/pull/7109)
* refactor(v2): remove yaml v2 dependency and use skaffold pkg/yaml instead [#7094](https://github.com/GoogleContainerTools/skaffold/pull/7094)

Docs, Test, and Release Updates:
* site: support nested tabs [#7195](https://github.com/GoogleContainerTools/skaffold/pull/7195)
* docs: fix skaffold resource selector docs [#7187](https://github.com/GoogleContainerTools/skaffold/pull/7187)
* docs: fix bad link to docker deployer page [#7188](https://github.com/GoogleContainerTools/skaffold/pull/7188)
* docs: Add Docker port-forwarding note [#7176](https://github.com/GoogleContainerTools/skaffold/pull/7176)
* docs: add docs for new skaffold resourceSelector configuration [#7174](https://github.com/GoogleContainerTools/skaffold/pull/7174)
* chore: cleanup default namespace deployment [#7148](https://github.com/GoogleContainerTools/skaffold/pull/7148)
* docs: cross platform build support in GCB [#7140](https://github.com/GoogleContainerTools/skaffold/pull/7140)
* docs: make noCache option more clear [#7141](https://github.com/GoogleContainerTools/skaffold/pull/7141)
* docs: Remove references to XXenableJibInit and XXenableBuildpacksInit from init docs [#7108](https://github.com/GoogleContainerTools/skaffold/pull/7108)
* docs: refresh DEVELOPMENT.md [#7129](https://github.com/GoogleContainerTools/skaffold/pull/7129)
* docs: make explicit that user needs to change PROJECT_ID [#7068](https://github.com/GoogleContainerTools/skaffold/pull/7068)
* chore: Update ROADMAP.md [#7115](https://github.com/GoogleContainerTools/skaffold/pull/7115)
* docs: add section on deactivating profiles [#7100](https://github.com/GoogleContainerTools/skaffold/pull/7100)

Huge thanks goes out to all of our contributors for this release:

- Aaron Prindle
- Adam Jensen
- Andreas Sommer
- Brandon High
- Chris Ge
- Christopher Bartz
- Dan Williams
- Gaurav
- Halvard Skogsrud
- Marlon Gamez
- Michael Mohamed
- Riccardo Carlesso
- Savas Ersin
- Steven Powell
- Tejal Desai
- Tomás Mota
- elnoro

# v1.36.0 Release - 02/08/2022
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.36.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.36.0/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.36.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.36.0/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v1.36.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.36.0`

Note: This release comes with a new config version, `v2beta27`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

New Features and Additions:
* feat(kustomize): Template `paths` [#7049](https://github.com/GoogleContainerTools/skaffold/pull/7049)
* feat(lint): added lib for getting yaml line #s for a given schema struct, sub-element and sub-fields.  supports patches & profiles [#6955](https://github.com/GoogleContainerTools/skaffold/pull/6955)
* feat(lint): add end line and col support to lint [#6926](https://github.com/GoogleContainerTools/skaffold/pull/6926)
* feat(logs): allow user config for parsing json application logs when using kubernetes deployers [#7074](https://github.com/GoogleContainerTools/skaffold/pull/7074)
* feat(tag): validate generated/provided tag [#7042](https://github.com/GoogleContainerTools/skaffold/pull/7042)
* feat(ko): Enable templating of `labels` and `env` [#6944](https://github.com/GoogleContainerTools/skaffold/pull/6944)
* feat: add 'branches' git tag variant [#7006](https://github.com/GoogleContainerTools/skaffold/pull/7006)
* feat: fetch default-repo from local cluster [#6985](https://github.com/GoogleContainerTools/skaffold/pull/6985)
* feat: allow specifying hook command directory [#6982](https://github.com/GoogleContainerTools/skaffold/pull/6982)
* feat: allow caching from built artifact on local Docker build [#6904](https://github.com/GoogleContainerTools/skaffold/pull/6904)
* feat: add multi-level repo support to arbitrary repositories beyond just GCR and AR [#6915](https://github.com/GoogleContainerTools/skaffold/pull/6915)
* feat: ignore vim *.swp files in hot-reload example [#6895](https://github.com/GoogleContainerTools/skaffold/pull/6895)

Fixes:
* fix(ko): Debug port forwarding for `ko://` images [#7009](https://github.com/GoogleContainerTools/skaffold/pull/7009)
* fix(ko): Default repo for pushing `ko://` images [#7010](https://github.com/GoogleContainerTools/skaffold/pull/7010)
* fix(ko): Do not print image name to stdout [#6928](https://github.com/GoogleContainerTools/skaffold/pull/6928)
* fix(ko): Do not add `-trimpath` when debugging [#6874](https://github.com/GoogleContainerTools/skaffold/pull/6874)
* fix(labelling): hotfix for issues related to deploy erroring due to nested yaml field overriding in StatefulSet w/ PVC [#7011](https://github.com/GoogleContainerTools/skaffold/pull/7011)
* fix: Allow apply of manifests without file name restrictions (#6871) [#6914](https://github.com/GoogleContainerTools/skaffold/pull/6914)
* fix: Helm: uninstall: Release not loaded #5414 [#7045](https://github.com/GoogleContainerTools/skaffold/pull/7045)
* fix: include registry for `docker login` and `gcloud auth configure-docker` [#7037](https://github.com/GoogleContainerTools/skaffold/pull/7037)
* fix: allow quotes on base image tags in Dockerfiles [#7027](https://github.com/GoogleContainerTools/skaffold/pull/7027)
* fix: integration test skaffold config [#7025](https://github.com/GoogleContainerTools/skaffold/pull/7025)
* fix: unblock cloud build edge builds [#7014](https://github.com/GoogleContainerTools/skaffold/pull/7014)
* fix: limit number of simultaneous taggers to GOMAXPROCS [#6976](https://github.com/GoogleContainerTools/skaffold/pull/6976)
* fix: correct $SKIP_TEST env for custom builder [#6984](https://github.com/GoogleContainerTools/skaffold/pull/6984)
* fix: bugs for skaffold.yaml files read in via stdin [#6970](https://github.com/GoogleContainerTools/skaffold/pull/6970)
* fix: modify error handling of SyncerMux#Sync [#6934](https://github.com/GoogleContainerTools/skaffold/pull/6934)
* fix: distinguish docker build errors from streaming errors [#6910](https://github.com/GoogleContainerTools/skaffold/pull/6910)
* fix: don't send error in skaffoldLogEvent if error is nil [#6945](https://github.com/GoogleContainerTools/skaffold/pull/6945)
* fix: correct English wording in error message [#6890](https://github.com/GoogleContainerTools/skaffold/pull/6890)
* fix: apply cmd should run `kubectl create --dry-run` to get the `ManifestList` [#6875](https://github.com/GoogleContainerTools/skaffold/pull/6875)
* fix: kubectl typo in kubectl errors [#6938](https://github.com/GoogleContainerTools/skaffold/pull/6938)
* fix: docker build for artifact with cliFlags should use docker CLI [#6930](https://github.com/GoogleContainerTools/skaffold/pull/6930)
* fix: report build failures inline for failed concurrent builds [#6911](https://github.com/GoogleContainerTools/skaffold/pull/6911)
* fix: use go 1.17.x in verify-examples github action [#7088](https://github.com/GoogleContainerTools/skaffold/pull/7088)
* fix: dont update last-prompted if check-update is false [#7067](https://github.com/GoogleContainerTools/skaffold/pull/7067)
* ci: fix lts image build [#7000](https://github.com/GoogleContainerTools/skaffold/pull/7000)
* ci: ensure build_deps:latest and build_deps:latest-lts are pushed on change [#7038](https://github.com/GoogleContainerTools/skaffold/pull/7038)
* ci: use consistent go build paths caching [#7044](https://github.com/GoogleContainerTools/skaffold/pull/7044)

Updates and Refactors:
* chore(deps): bump containerd and opencontainers/images-spec [#6876](https://github.com/GoogleContainerTools/skaffold/pull/6876)
* chore: update go.mod go directive and go mod tidy [#7061](https://github.com/GoogleContainerTools/skaffold/pull/7061)
* chore: install gke-gcloud-auth-plugin in skaffold images [#7060](https://github.com/GoogleContainerTools/skaffold/pull/7060)
* chore: update Jib versions to 3.2.0 in examples [#7040](https://github.com/GoogleContainerTools/skaffold/pull/7040)
* chore: log kaniko errors that happen during the tar phase [#6901](https://github.com/GoogleContainerTools/skaffold/pull/6901)
* refactor(configlocations): make yamlinfos field private [#7005](https://github.com/GoogleContainerTools/skaffold/pull/7005)
* refactor(lint): remove unused regexp linter [#6923](https://github.com/GoogleContainerTools/skaffold/pull/6923)
* refactor: remove vestiges of parsing release info text from helm deployment [#6913](https://github.com/GoogleContainerTools/skaffold/pull/6913)
* feat(deps): Update Skaffold images to Go 1.17.3 [#7028](https://github.com/GoogleContainerTools/skaffold/pull/7028)
* feat: update to buildpacks/pack v0.23.0 [#6979](https://github.com/GoogleContainerTools/skaffold/pull/6979)
* feat: monitor OS vulnerability in Skaffold LTS images. [#6964](https://github.com/GoogleContainerTools/skaffold/pull/6964)
* fix: add godoc to Fs var in config.go [#7004](https://github.com/GoogleContainerTools/skaffold/pull/7004)
* ci: remove cross compilation step of cloud build job [#6956](https://github.com/GoogleContainerTools/skaffold/pull/6956)
* ci: add kokoro scripts for releases [#6906](https://github.com/GoogleContainerTools/skaffold/pull/6906)

Docs and Testing:
* docs(debug): add why breakpoints may fail and clean up manual configuration [#6893](https://github.com/GoogleContainerTools/skaffold/pull/6893)
* docs: add some detail to local clusters and minikube docker [#6993](https://github.com/GoogleContainerTools/skaffold/pull/6993)
* docs: make healthcheck page top level doc [#6898](https://github.com/GoogleContainerTools/skaffold/pull/6898)
* docs: link Cloud Code `Run on Kubernetes` guide in `skaffold dev` guide page [#6897](https://github.com/GoogleContainerTools/skaffold/pull/6897)
* docs: fix typo in file name [#6887](https://github.com/GoogleContainerTools/skaffold/pull/6887)
* docs: reorder `skaffold debug` docs [#6894](https://github.com/GoogleContainerTools/skaffold/pull/6894)
* docs: small typo "Skaffold knows to build" [#6967](https://github.com/GoogleContainerTools/skaffold/pull/6967)
* docs: separate quickstart guides [#6869](https://github.com/GoogleContainerTools/skaffold/pull/6869)
* docs: link cloud code support for updating skaffold.yaml files [#6866](https://github.com/GoogleContainerTools/skaffold/pull/6866)
* docs: add details on tryImportMissing [#6880](https://github.com/GoogleContainerTools/skaffold/pull/6880)
* docs: address newcomer confusion when joining k8s slack channel [#6881](https://github.com/GoogleContainerTools/skaffold/pull/6881)
* docs: link Cloud Code how-to guides in docsite [#6879](https://github.com/GoogleContainerTools/skaffold/pull/6879)
* test: modules integration tests [#6899](https://github.com/GoogleContainerTools/skaffold/pull/6899)
* test: fix failing windows UT [#7091](https://github.com/GoogleContainerTools/skaffold/pull/7091)

Huge thanks goes out to all of our contributors for this release:

- Andrés Torres
- Artem Kuznetsov
- Brian de Alwis
- Chanseok Oh
- Chris Ge
- David Xia
- Gaurav
- Halvard Skogsrud
- Joel Pearson
- Khris Richardson
- Lukas
- Marlon Gamez
- Nick Kubala
- Patrick Lundquist
- Savas Ersin
- Seth Nickell
- Tejal Desai
- Yuta Nishimori
- Zbigniew Mandziejewicz
- smaftoul

# v1.35.2 Release - 01/13/2022
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.35.2/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.35.2/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.35.2/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.35.2/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v1.35.2/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.35.2`

This is patch release to get in changes from https://github.com/GoogleContainerTools/skaffold/pull/7011, which fixes an issue with skaffold deployments and StatefulSets.

Huge thanks goes out to all of our contributors for this release:

- Aaron Prindle

# v1.35.1 Release - 11/18/2021
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.35.1/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.35.1/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.35.1/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.35.1/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v1.35.1/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.35.1`

This is patch release to fix two issue
* fix(ko): Do not add `-trimpath` when debugging [#6874](https://github.com/GoogleContainerTools/skaffold/pull/6874)
* fix: apply cmd should run `kubectl create --dry-run` to get the `ManifestList` 

Huge thanks goes out to all of our contributors for this release:

- Gaurav
- Halvard Skogsrud

# v1.35.0 Release - 11/16/2021
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.35.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.35.0/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.35.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.35.0/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v1.35.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.35.0`

Note: This release comes with a new config version, `v2beta26`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:
* [alpha] Skaffold now natively supports  `ko` builder for golang projects. Please try it out and let us [know](https://skaffold.dev/docs/pipeline-stages/builders/ko/)
* Skaffold now performs status-check for stateful sets [#6828](https://github.com/GoogleContainerTools/skaffold/pull/6828)

New Features and Additions:
* feat: add lts image, cloud build triggers [#6844](https://github.com/GoogleContainerTools/skaffold/pull/6844)
* feat: introduce --output option for "fix" cmd [#6849](https://github.com/GoogleContainerTools/skaffold/pull/6849)
* feat: add pullParent support for docker builds [#6825](https://github.com/GoogleContainerTools/skaffold/pull/6825)
* feat: add k8s manifest support to skaffold lint and one sample rule [#6795](https://github.com/GoogleContainerTools/skaffold/pull/6795)
* feat: write skaffold logs from current run to file [#6803](https://github.com/GoogleContainerTools/skaffold/pull/6803)
* feat: add dockerfile support to skaffold lint and top 2 dockerfile rules [#6793](https://github.com/GoogleContainerTools/skaffold/pull/6793)
* feat: Enable ko builder (alpha) in schema [#6811](https://github.com/GoogleContainerTools/skaffold/pull/6811)
* feat(ko): Add ko builder to local artifact builder [#6785](https://github.com/GoogleContainerTools/skaffold/pull/6785)
* feat(ko): Enable the ko builder in the API [#6820](https://github.com/GoogleContainerTools/skaffold/pull/6820)
* feat: add support for Kaniko flag --cache-copy-layers [#6703](https://github.com/GoogleContainerTools/skaffold/pull/6703)
* feat: set kpt inventory configs for render and deploy  [#6712](https://github.com/GoogleContainerTools/skaffold/pull/6712)
* feat: add dry run option to skaffold delete [#6655](https://github.com/GoogleContainerTools/skaffold/pull/6655)
* feat: status check for config-connector [#6766](https://github.com/GoogleContainerTools/skaffold/pull/6766)
* feat: enable render in `skaffold run` v2. [#6761](https://github.com/GoogleContainerTools/skaffold/pull/6761)
* feat: Add Labels to Metadata [#6782](https://github.com/GoogleContainerTools/skaffold/pull/6782)


Fixes:
* fix: interface conversion error for pod event [#6863](https://github.com/GoogleContainerTools/skaffold/pull/6863)
* fix: add diagnostic severity info to skaffold lint rules [#6862](https://github.com/GoogleContainerTools/skaffold/pull/6862)
* fix: Add skaffold internal error and return that instead of user cancelled [#6846](https://github.com/GoogleContainerTools/skaffold/pull/6846)
* fix: make kcc status-check less aggressive [#6841](https://github.com/GoogleContainerTools/skaffold/pull/6841)
* fix(log): Send Go std `log` to `logrus`, and output `ggcr` logs [#6815](https://github.com/GoogleContainerTools/skaffold/pull/6815)
* fix: fix nil pointer issue for skaff lint when encountering skaffold.yaml with no k8s manifests [#6832](https://github.com/GoogleContainerTools/skaffold/pull/6832)
* fix: fix multi-module issue for skaffold lint dockerfile support [#6831](https://github.com/GoogleContainerTools/skaffold/pull/6831)
* fix: `deploy --skip-render` not applying skaffold labels, causes status check to not work [#6838](https://github.com/GoogleContainerTools/skaffold/pull/6838)
* fix: update windows ci description to be correct [#6830](https://github.com/GoogleContainerTools/skaffold/pull/6830)
* fix: fix skaff lint field selector to work more broadly [#6834](https://github.com/GoogleContainerTools/skaffold/pull/6834)
* fix: Fix build pipeline to always build dependencies. [#6823](https://github.com/GoogleContainerTools/skaffold/pull/6823)
* fix(sync): more descriptive error for custom build inferred sync misconfiguration [#6778](https://github.com/GoogleContainerTools/skaffold/pull/6778)
* fix(ko): Fall back to build configs in `.ko.yaml` [#6821](https://github.com/GoogleContainerTools/skaffold/pull/6821)
* fix: propagate-profiles flag missing from `skaffold inspect` command [#6818](https://github.com/GoogleContainerTools/skaffold/pull/6818)
* fix: `skaffold inspect` commands should have non-zero exit-code on error [#6807](https://github.com/GoogleContainerTools/skaffold/pull/6807)
* fix(ko): Fix ko build config path matching [#6797](https://github.com/GoogleContainerTools/skaffold/pull/6797)
* fix(helm): handle templated namespaces consistently [#6767](https://github.com/GoogleContainerTools/skaffold/pull/6767)
* fix: Quotes in dockerfiles env vars break copy dependency checks [#6796](https://github.com/GoogleContainerTools/skaffold/pull/6796)
* fix(find-configs): log skaffold.yaml parsing errors at debug [#6748](https://github.com/GoogleContainerTools/skaffold/pull/6748)

Updates and Refactors:
* refactor: group/alphabetize skaffold options [#6853](https://github.com/GoogleContainerTools/skaffold/pull/6853)
* chore: upgrade k3d to latest bugfix-version [#6781](https://github.com/GoogleContainerTools/skaffold/pull/6781)
* chore: make test env check output what was found [#6744](https://github.com/GoogleContainerTools/skaffold/pull/6744)
* chore(deps): bump puma from 4.3.8 to 4.3.9 in /examples/ruby/backend [#6771](https://github.com/GoogleContainerTools/skaffold/pull/6771)
* chore: add script to improve QOL when doing release [#6774](https://github.com/GoogleContainerTools/skaffold/pull/6774)
* chore(deps): update to kompose 1.26 [#6865](https://github.com/GoogleContainerTools/skaffold/pull/6865)
* refactor: organize event v2 functions [#6802](https://github.com/GoogleContainerTools/skaffold/pull/6802)

Docs, Test, and Release Updates:
* docs: link to Cloud Code in github README [#6864](https://github.com/GoogleContainerTools/skaffold/pull/6864)
* docs(debug): Improve Go debugging documentation [#6852](https://github.com/GoogleContainerTools/skaffold/pull/6852)
* docs(ko): Improve ko docs for existing ko users [#6826](https://github.com/GoogleContainerTools/skaffold/pull/6826)
* docs: Move Docker deployer to beta [#6850](https://github.com/GoogleContainerTools/skaffold/pull/6850)
* doc: add scoop-extras installation details [#6847](https://github.com/GoogleContainerTools/skaffold/pull/6847)
* docs(ko): Shorter example values in config schema [#6837](https://github.com/GoogleContainerTools/skaffold/pull/6837)
* docs(ko): Update debug docs for ko images [#6833](https://github.com/GoogleContainerTools/skaffold/pull/6833)
* docs(ko): Templating in `flags` and `ldflags` [#6798](https://github.com/GoogleContainerTools/skaffold/pull/6798)
* docs(ko): Document the ko builder [#6792](https://github.com/GoogleContainerTools/skaffold/pull/6792)
* doc: add `minikube start` to the quickstart documentation [#6783](https://github.com/GoogleContainerTools/skaffold/pull/6783)
* docs: skaffold apply supports status check [#6779](https://github.com/GoogleContainerTools/skaffold/pull/6779)
  ing static port usage for relevant deployed resources [#6776](https://github.com/GoogleContainerTools/skaffold/pull/6776)
* docs: add release stage plan to ko builder design doc [#6764](https://github.com/GoogleContainerTools/skaffold/pull/6764)
* docs: Clarify custom local dependencies example [#6827](https://github.com/GoogleContainerTools/skaffold/pull/6827)
* test(ko): Simple integration test for ko builder [#6788](https://github.com/GoogleContainerTools/skaffold/pull/6788)
* test: add integration test for config connector status check [#6839](https://github.com/GoogleContainerTools/skaffold/pull/6839)
* test: fix integration test for stateful-sets [#6829](https://github.com/GoogleContainerTools/skaffold/pull/6829)
* test: update modules testcases [#6813](https://github.com/GoogleContainerTools/skaffold/pull/6813)
* ci: add cancel-workflow-action functionality to all github workflows [#6755](https://github.com/GoogleContainerTools/skaffold/pull/6755)


Huge thanks goes out to all of our contributors for this release:

- Aaron Prindle
- Ahmet Alp Balkan
- Brian de Alwis
- Conor A. Callahan
- Erik Godding Boye
- Gaurav
- Halvard Skogsrud
- Jeremy Lewi
- Marlon Gamez
- Max Brauer
- Nick Kubala
- Pablo Caderno
- Rouan van der Ende
- Tejal Desai
- jrcast

# v1.34.0 Release - 10/26/2021
**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.34.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.34.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v1.34.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.34.0`

Note: This release comes with a new config version, `v2beta25`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:
* `skaffold deploy`, `skaffold render` and `skaffold test` now support the `--images` flag for providing a list of images to run against.
* The `--images` flag now supports `NAME=TAG` pairs.

New Features and Additions:
* feat: trap SIGUSR1 to dump current stacktrace [#6751](https://github.com/GoogleContainerTools/skaffold/pull/6751)
* feat: skaffold lint core logic, CLI command+flags, and MVP skaffold.yaml support [#6715](https://github.com/GoogleContainerTools/skaffold/pull/6715)

Fixes:
* fix: fix non-determinism in skaffoldyamls_test.go [#6757](https://github.com/GoogleContainerTools/skaffold/pull/6757)
* fix: use new stringslice lib [#6752](https://github.com/GoogleContainerTools/skaffold/pull/6752)
* fix: correctly rewrite debug container entrypoint and bind host port [#6682](https://github.com/GoogleContainerTools/skaffold/pull/6682)
* Fix new version generation [#6616](https://github.com/GoogleContainerTools/skaffold/pull/6616)
* fix: panic caused by multiple channel closes [#6714](https://github.com/GoogleContainerTools/skaffold/pull/6714)
* fix: sanity check kpt deployer versions [#6711](https://github.com/GoogleContainerTools/skaffold/pull/6711)

Updates and Refactors:
* refactor: move some of `pkg/skaffold/util` into into packages [#6731](https://github.com/GoogleContainerTools/skaffold/pull/6731)

Docs, Test, and Release Updates:
* doc: update CI-CD article to decouple `render` and `deploy` use from GitOps [#6750](https://github.com/GoogleContainerTools/skaffold/pull/6750)
* chore: update image dependencies [#6736](https://github.com/GoogleContainerTools/skaffold/pull/6736)
* chore: use ad-hoc signing on darwin to avoid network popups [#6738](https://github.com/GoogleContainerTools/skaffold/pull/6738)
* chore: add cluster type to the instrumentation meter [#6734](https://github.com/GoogleContainerTools/skaffold/pull/6734)
* chore: Update globstar syntax in examples dependencies [#6614](https://github.com/GoogleContainerTools/skaffold/pull/6614)
* docs: add Cloud Code install instructions [#6716](https://github.com/GoogleContainerTools/skaffold/pull/6716)

Huge thanks goes out to all of our contributors for this release:

- Aaron Prindle
- Brian de Alwis
- Conor A. Callahan
- Dave Dorbin
- Gaurav
- Halvard Skogsrud
- Marlon Gamez
- Mike Verbanic
- Nick Kubala
- Tejal Desai
- Yuwen Ma

# v1.33.0 Release - 10/07/2021
**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.33.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.33.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v1.33.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.33.0`

Note: This release comes with a new config version, `v2beta24`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:
* Skaffold healthcheck now monitors standalone pods.

New Features and Additions:
* feat: status-check for standalone pods [#6697](https://github.com/GoogleContainerTools/skaffold/pull/6697)
* feat: prototype control api devloop intent [#6636](https://github.com/GoogleContainerTools/skaffold/pull/6636)
* feat: use cloud build location service to create builds in workerpool across regions [#6666](https://github.com/GoogleContainerTools/skaffold/pull/6666)
* feat: Add distinct error codes for GCB failures [#6664](https://github.com/GoogleContainerTools/skaffold/pull/6664)
* feat: add preliminary support for Config Connector service KRM [#6645](https://github.com/GoogleContainerTools/skaffold/pull/6645)
* feat:(build/docker): support env as secret source [#6632](https://github.com/GoogleContainerTools/skaffold/pull/6632)
* Update to grpc-gateway v2 [#6567](https://github.com/GoogleContainerTools/skaffold/pull/6567)

Fixes:
* fix: Kubectl port fwd returns on context canceled [#6700](https://github.com/GoogleContainerTools/skaffold/pull/6700)
* fix: Sanitize image names when default repo unset [#6678](https://github.com/GoogleContainerTools/skaffold/pull/6678)
* fix: ensure run-id is added to resources in skaffold apply [#6674](https://github.com/GoogleContainerTools/skaffold/pull/6674)
* fix: correct ko package imports and logging function call [#6673](https://github.com/GoogleContainerTools/skaffold/pull/6673)
* fix: Use gcloud spec pool instead of deprecated WorkerPool [#6658](https://github.com/GoogleContainerTools/skaffold/pull/6658)
* fix: fix unit test for skaffold inspect tests list [#6656](https://github.com/GoogleContainerTools/skaffold/pull/6656)
* fix: Wait for context cancel in k8s pod watcher [#6643](https://github.com/GoogleContainerTools/skaffold/pull/6643)
* Create port map on container config if empty [#6621](https://github.com/GoogleContainerTools/skaffold/pull/6621)
* fix: remove "managed-by" fixing helm conflict (fixes #6421) [#6618](https://github.com/GoogleContainerTools/skaffold/pull/6618)
* Check for nil when retrieving docker support mounts [#6620](https://github.com/GoogleContainerTools/skaffold/pull/6620)
* fix: make `useBuildkit` field nullable across all config versions [#6612](https://github.com/GoogleContainerTools/skaffold/pull/6612)

Updates and Refactors:
* Support globstar in dependencies for file watching [#6605](https://github.com/GoogleContainerTools/skaffold/pull/6605)
* chore: log errors when retrieving build logs [#6663](https://github.com/GoogleContainerTools/skaffold/pull/6663)
* chore(deps): bump flask from 2.0.1 to 2.0.2 in /integration/examples [#6677](https://github.com/GoogleContainerTools/skaffold/pull/6677)
* chore(deps): bump flask from 2.0.1 to 2.0.2 in /examples [#6676](https://github.com/GoogleContainerTools/skaffold/pull/6676)
* introduce schema v2beta24 [#6628](https://github.com/GoogleContainerTools/skaffold/pull/6628)
* Update pack image to v0.21.1 [#6630](https://github.com/GoogleContainerTools/skaffold/pull/6630)
* Add image label support to ko builder [#6597](https://github.com/GoogleContainerTools/skaffold/pull/6597)
* Clean up deps [#6611](https://github.com/GoogleContainerTools/skaffold/pull/6611)

Docs, Test, and Release Updates:
* Add initial docs for Docker deployer [#6613](https://github.com/GoogleContainerTools/skaffold/pull/6613)
* doc: document Helm deployer's IMAGE_NAME<N>, IMAGE_TAG<N>, IMAGE_DIGEST<N> [#6649](https://github.com/GoogleContainerTools/skaffold/pull/6649)
* docs: update skaffold development guide to include information about commit messages [#6670](https://github.com/GoogleContainerTools/skaffold/pull/6670)
* Validate changed examples with "local" builder [#6133](https://github.com/GoogleContainerTools/skaffold/pull/6133)
* fix: properly generate enums for config schemas when running `make generate-schemas` [#6651](https://github.com/GoogleContainerTools/skaffold/pull/6651)
* refactor: Rename a couple of scripts for consistency [#6625](https://github.com/GoogleContainerTools/skaffold/pull/6625)
* chore: update skaffold Q4 planning board [#6631](https://github.com/GoogleContainerTools/skaffold/pull/6631)
* Remove `hack/release-notes` binary and update script to remove after running [#6610](https://github.com/GoogleContainerTools/skaffold/pull/6610)
* Drop codecov threshold to 30% [#6608](https://github.com/GoogleContainerTools/skaffold/pull/6608)
* build: 30 min timeout for integration tests [#6684](https://github.com/GoogleContainerTools/skaffold/pull/6684)

Huge thanks goes out to all of our contributors for this release:

- Aaron Prindle
- Ahmet Alp Balkan
- Brian de Alwis
- Gaurav
- Glenn Pratt
- Halvard Skogsrud
- Marlon Gamez
- Nick Kubala
- Seth Nickell
- Tejal Desai

# v1.32.0 Release - 09/15/2021
**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.32.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.32.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v1.32.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.32.0`

Note: This release comes with a new config version, `v2beta23`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:
* **Skaffold now supports a brand new deploy type - `docker`, enabling iterative application development without Kubernetes by creating containers directly in the local Docker daemon!**
  - _This is an experimental **alpha** feature - we expect to find issues as it is put through the paces. Please be patient as we work to refine the experience, and be sure to report any issues or improvement ideas!_
* The Skaffold HTTP/gRPC APIs are now disabled by default for `dev` and `run`, and the `--enable-rpc` flag is now deprecated. The APIs will now only be activated if the `--rpc-port` or `--rpc-http-port` flags are set, and will only bind to the provided ports (rather than defaulting to `50051` or `50052`). Please see the documentation for more details.

New Features and Additions:
* New config option deploy.transformableAllowList [#6452](https://github.com/GoogleContainerTools/skaffold/pull/6452)

Fixes:
* fix: add podforwaders correctly for multimodule projects [#6606](https://github.com/GoogleContainerTools/skaffold/pull/6606)
* fix: treat pod deletion as if its containers were terminated [#6587](https://github.com/GoogleContainerTools/skaffold/pull/6587)
* Updating load images to use only local images [#6582](https://github.com/GoogleContainerTools/skaffold/pull/6582)
* fix: iterative-status-check only for deployers defining hooks [#6574](https://github.com/GoogleContainerTools/skaffold/pull/6574)

Updates and Refactors:
* Extend `--artifact` API schema to allow for manifest generation [#6599](https://github.com/GoogleContainerTools/skaffold/pull/6599)
* api: Strict port number handling and --enable-rpc deprecation [#6459](https://github.com/GoogleContainerTools/skaffold/pull/6459)
* add testType field to skaffold inspect tests output [#6561](https://github.com/GoogleContainerTools/skaffold/pull/6561)
* Persist EventV2 logs when `--event-log-file` exists [#6581](https://github.com/GoogleContainerTools/skaffold/pull/6581)
* Ko builder: Add Env, Flags, and Ldflags config [#6546](https://github.com/GoogleContainerTools/skaffold/pull/6546)
* Detect ko images for debugging, by image author [#6569](https://github.com/GoogleContainerTools/skaffold/pull/6569)
* Detect ko images for debugging, by envvar [#6563](https://github.com/GoogleContainerTools/skaffold/pull/6563)
* Make inputDigest hash calculation independent of workspace path [#6522](https://github.com/GoogleContainerTools/skaffold/pull/6522)

Docs, Test, and Release Updates:
* Make TestHelmDeploy hermetic [#6590](https://github.com/GoogleContainerTools/skaffold/pull/6590)
* Fix wrong link on the custom-tests example, fixes #6591 [#6592](https://github.com/GoogleContainerTools/skaffold/pull/6592)
* Updates skaffold.yaml example in helm section [#6585](https://github.com/GoogleContainerTools/skaffold/pull/6585)
* docs: mention hooks don't run on cached artifacts [#6577](https://github.com/GoogleContainerTools/skaffold/pull/6577)
* Run CodeQL analysis nightly and cache the go modules cache [#6548](https://github.com/GoogleContainerTools/skaffold/pull/6548)
* Fix issue 6513 - CI/CD page readability [#6555](https://github.com/GoogleContainerTools/skaffold/pull/6555)
* docs: use dropdown menu for selecting version [#6559](https://github.com/GoogleContainerTools/skaffold/pull/6559)

Huge thanks goes out to all of our contributors for this release:

- Aaron Prindle
- Ahmet Alp Balkan
- Brian de Alwis
- Gaurav
- Gerson Sosa
- Halvard Skogsrud
- Ke Zhu
- Marlon Gamez
- Mattias Öhrn
- Mike Verbanic
- Nick Kubala
- Tejal Desai
- Yuki Ito

# v1.31.0 Release - 09/01/2021
**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.31.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.31.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v1.31.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.31.0`

Note: This release comes with a new config version, `v2beta22`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:
* hooks: implement for `helm` and `kustomize` deployers [#6454](https://github.com/GoogleContainerTools/skaffold/pull/6454)
* Add `--load-images` flag to deploy [#6428](https://github.com/GoogleContainerTools/skaffold/pull/6428)
* Add skip_test env for custom build script [#6455](https://github.com/GoogleContainerTools/skaffold/pull/6455)
* Add core ko builder implementation [#6054](https://github.com/GoogleContainerTools/skaffold/pull/6054)

Fixes:
* Use `ResourceStatusCheckEvent[Updated/Completed]` where appropriate [#6550](https://github.com/GoogleContainerTools/skaffold/pull/6550)
* Fix buildpacks builder output to go through Event API [#6530](https://github.com/GoogleContainerTools/skaffold/pull/6530)
* fix: helm resource labeller checking wrong kubectl context [#6547](https://github.com/GoogleContainerTools/skaffold/pull/6547)
* fix: custom.GetDependencies should execute Command in workspace [#6321](https://github.com/GoogleContainerTools/skaffold/pull/6321)
* Add newlines to `SkaffoldLogEvent`s that come from `logrus` [#6506](https://github.com/GoogleContainerTools/skaffold/pull/6506)
* Fix skaffold inspect tests flags and help message [#6493](https://github.com/GoogleContainerTools/skaffold/pull/6493)
* fix: do not sync remote configs for specific commands [#6453](https://github.com/GoogleContainerTools/skaffold/pull/6453)
* Fix integration/'s randomPort() to return an unallocated port [#6474](https://github.com/GoogleContainerTools/skaffold/pull/6474)
* fix: TestDebug/buildpacks by pinning go version [#6463](https://github.com/GoogleContainerTools/skaffold/pull/6463)

Updates and Refactors:
* Send success for pods for a deployment when its rolled out successfully [#6534](https://github.com/GoogleContainerTools/skaffold/pull/6534)
* Add option to wait for connection to `/v2/events` on gRPC/HTTP server [#6545](https://github.com/GoogleContainerTools/skaffold/pull/6545)
* Plumb `context.Context` down into functions in pkg/skaffold/sync [#6535](https://github.com/GoogleContainerTools/skaffold/pull/6535)
* Add `Dir` config field to ko builder [#6496](https://github.com/GoogleContainerTools/skaffold/pull/6496)
* Skip tests if no tests defined in pipelines [#6527](https://github.com/GoogleContainerTools/skaffold/pull/6527)
* Use default values rather than return error in logrus hook [#6523](https://github.com/GoogleContainerTools/skaffold/pull/6523)
* Send error logs through the event API with the correct task/subtask [#6516](https://github.com/GoogleContainerTools/skaffold/pull/6516)
* Abstract k8s container representation from debug transformers [#6335](https://github.com/GoogleContainerTools/skaffold/pull/6335)
* Wrap tester in `SkaffoldRunner` to improve `SkaffoldLogEvent` labelling [#6469](https://github.com/GoogleContainerTools/skaffold/pull/6469)
* Plumb `context.Context` down into functions that use `util/cmd` functions [#6468](https://github.com/GoogleContainerTools/skaffold/pull/6468)
* log: use context.TODO for not yet plumbed ctx [#6462](https://github.com/GoogleContainerTools/skaffold/pull/6462)
* Implement Target support for ko builder [#6447](https://github.com/GoogleContainerTools/skaffold/pull/6447)
* Various small UX improvements [#6426](https://github.com/GoogleContainerTools/skaffold/pull/6426)
* Refine ko builder behavior for main packages [#6437](https://github.com/GoogleContainerTools/skaffold/pull/6437)

Docs, Test, and Release Updates:
* docs: Add skaffold.yaml to navbar [#6553](https://github.com/GoogleContainerTools/skaffold/pull/6553)
* docs: fix duplicate subpage listing [#6540](https://github.com/GoogleContainerTools/skaffold/pull/6540)
* [docs fixit] update skaffold fix text to have full description of command [#6536](https://github.com/GoogleContainerTools/skaffold/pull/6536)
* docs: update `test` phase references [#6539](https://github.com/GoogleContainerTools/skaffold/pull/6539)
* docs: update `dev` documentation for `test` phase; [#6538](https://github.com/GoogleContainerTools/skaffold/pull/6538)
* docs: rename section "Working with Skaffold" to "Guides" [#6537](https://github.com/GoogleContainerTools/skaffold/pull/6537)
* Add `skaffold apply` to CLI docs header [#6509](https://github.com/GoogleContainerTools/skaffold/pull/6509)
* docs: Add "Open in Cloud Shell" link to examples [#6514](https://github.com/GoogleContainerTools/skaffold/pull/6514)
* Document multi-stage dockerfile limitations of filesync [#6526](https://github.com/GoogleContainerTools/skaffold/pull/6526)
* docs: explain minikube detection [#6512](https://github.com/GoogleContainerTools/skaffold/pull/6512)
* docs: better wording explaining "skaffold init" [#6502](https://github.com/GoogleContainerTools/skaffold/pull/6502)
* docs: dev.md improvements [#6503](https://github.com/GoogleContainerTools/skaffold/pull/6503)
* Updated Cloud Code links to link to debug page [#6505](https://github.com/GoogleContainerTools/skaffold/pull/6505)
* document `make quicktest` [#6458](https://github.com/GoogleContainerTools/skaffold/pull/6458)
* Change wording describing how profiles work [#6445](https://github.com/GoogleContainerTools/skaffold/pull/6445)
* remove note on vendor/ usage from DEVELOPMENT.md [#6422](https://github.com/GoogleContainerTools/skaffold/pull/6422)

Huge thanks goes out to all of our contributors for this release:

- Aaron Prindle
- Ahmet Alp Balkan
- Brian de Alwis
- Chanseok Oh
- David Zumbrunnen
- Gaurav
- Glenn Pratt
- Halvard Skogsrud
- Henry Bell
- Ke Zhu
- Kourtney
- Marlon Gamez
- Mike Verbanic
- Nick Kubala
- Pradeep Kumar
- Tejal Desai
- Yanshu
- kelsk

# v1.30.0 Release - 08/11/2021
**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.30.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.30.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v1.30.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.30.0`

Note: This release comes with a new config version, `v2beta21`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:
* Support deploy hooks for Skaffold lifecycle hooks. Read more [here](https://skaffold.dev/docs/pipeline-stages/lifecycle-hooks/#before-deploy-and-after-deploy-only-for-kubectl-deployer-1) [#6376](https://github.com/GoogleContainerTools/skaffold/pull/6376)
* Add support for Kaniko flag --image-fs-extract-retry [#6380](https://github.com/GoogleContainerTools/skaffold/pull/6380)
* Support passing additional CLI flags to docker build [#6343](https://github.com/GoogleContainerTools/skaffold/pull/6343)


Fixes:
* remove single `kubeContext` validation [#6394](https://github.com/GoogleContainerTools/skaffold/pull/6394)
* Set exit code 127 for Skaffold CLI validation errors [#6401](https://github.com/GoogleContainerTools/skaffold/pull/6401)
* Fix status check showing unhealthy pods from previous dev iteration [#6370](https://github.com/GoogleContainerTools/skaffold/pull/6370)
* Ignore leading 'v' when parsing helm versions [#6362](https://github.com/GoogleContainerTools/skaffold/pull/6362)
* fix: make `client.Client` kubernetesContext aware [#6368](https://github.com/GoogleContainerTools/skaffold/pull/6368)
* ResourceForwarder should wait for its forwards [#6332](https://github.com/GoogleContainerTools/skaffold/pull/6332)
* hooks: prepend pod/container name to container hooks log lines [#6337](https://github.com/GoogleContainerTools/skaffold/pull/6337)
* Fix issue with displaying survey prompts when we shouldn't [#6354](https://github.com/GoogleContainerTools/skaffold/pull/6354)
* fix: remote manifest image substitution [#6342](https://github.com/GoogleContainerTools/skaffold/pull/6342)
* fix build --push=false for missing kubeconfig [#6366](https://github.com/GoogleContainerTools/skaffold/pull/6366)
* Ensure Cleanup is called if Deploy creates resources but fails [#6345](https://github.com/GoogleContainerTools/skaffold/pull/6345)

Updates and Refactors:
* Add functionality to support patterns in `--user` flag [#6402](https://github.com/GoogleContainerTools/skaffold/pull/6402)
* Improvements to upcoming Event Api v2 [#6399](https://github.com/GoogleContainerTools/skaffold/pull/6399), [#6407](https://github.com/GoogleContainerTools/skaffold/pull/6407), [#6395](https://github.com/GoogleContainerTools/skaffold/pull/6395)
* Change parsing templated image warning to debug info [#6398](https://github.com/GoogleContainerTools/skaffold/pull/6398)

Docs, Test, and Release Updates:
* hooks: update deploy docs [#6386](https://github.com/GoogleContainerTools/skaffold/pull/6386)
* [design proposal] Add config option 'deploy.config.transformableAllowList' [#6236](https://github.com/GoogleContainerTools/skaffold/pull/6236)
* GitLab (with capital 'L') [#6384](https://github.com/GoogleContainerTools/skaffold/pull/6384)
* Validate generated schemas in generator script [#6385](https://github.com/GoogleContainerTools/skaffold/pull/6385)
* fix: schema gen for `ContainerHook` [#6372](https://github.com/GoogleContainerTools/skaffold/pull/6372)
* Add trace-level port-allocation logs [#6293](https://github.com/GoogleContainerTools/skaffold/pull/6293)
* Bump cloud.google.com/go/storage from 1.10.0 to 1.16.0 [#6324](https://github.com/GoogleContainerTools/skaffold/pull/6324)
* Mark GCP Buildpacks builder as trusted in the examples [#6284](https://github.com/GoogleContainerTools/skaffold/pull/6284)


Huge thanks goes out to all of our contributors for this release:

- Ahmet Alp Balkan
- Brian de Alwis
- David Zumbrunnen
- Gaurav
- Ke Zhu
- Marlon Gamez
- Mike Verbanic
- Nick Kubala
- Pradeep Kumar
- Tejal Desai
- Yanshu
- dependabot[bot]

# v1.29.0 Release - 07/30/2021
Note: This release comes with a new config version, `v2beta20`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:
* Introducing Skaffold lifecycle hooks [#6330](https://github.com/GoogleContainerTools/skaffold/pull/6330)
* Rename Skaffold's `master` branch to `main` [#6263](https://github.com/GoogleContainerTools/skaffold/pull/6263)
* Add state to event handler to track `logrus` output events [#6272](https://github.com/GoogleContainerTools/skaffold/pull/6272)
* fix: kubeContext override via flag [#6331](https://github.com/GoogleContainerTools/skaffold/pull/6331)
* Added developer journey tutorial [#6201](https://github.com/GoogleContainerTools/skaffold/pull/6201)

Fixes:
* Fix tail false but still log issue [#6299](https://github.com/GoogleContainerTools/skaffold/pull/6299)
* skip manifest validation for default `kubectl deployer` [#6294](https://github.com/GoogleContainerTools/skaffold/pull/6294)
* Fix switched container/pod names in app log event [#6215](https://github.com/GoogleContainerTools/skaffold/pull/6215)
* fix remote branch lookup [#6269](https://github.com/GoogleContainerTools/skaffold/pull/6269)
* [cherry-pick] Fix Workdir error when --filename flag is used.  [#6247](https://github.com/GoogleContainerTools/skaffold/pull/6247)
* Make skaffold reproducible [#6238](https://github.com/GoogleContainerTools/skaffold/pull/6238)

Updates and Refactors:
* Give `logrus.Hook` implementation information about task and subtask [#6313](https://github.com/GoogleContainerTools/skaffold/pull/6313)
* Add `logrus.Logger` return type on `WithEventContext()` [#6309](https://github.com/GoogleContainerTools/skaffold/pull/6309)
* Add `logrus` hook for sending `SkaffoldLogEvent`s [#6250](https://github.com/GoogleContainerTools/skaffold/pull/6250)

* Prefix port forward links with `http://` [#6295](https://github.com/GoogleContainerTools/skaffold/pull/6295)
* Set output event context in cache check, tag generation, status check, port forward [#6234](https://github.com/GoogleContainerTools/skaffold/pull/6234)


Docs, Test, and Release Updates:
* Update deps and restore k3d tests [#6280](https://github.com/GoogleContainerTools/skaffold/pull/6280)
* Remove log tail test for nodejs example [#6275](https://github.com/GoogleContainerTools/skaffold/pull/6275)
* `git fetch` origin/main before `make checks` [#6274](https://github.com/GoogleContainerTools/skaffold/pull/6274)


Huge thanks goes out to all of our contributors for this release:

- Aaron Prindle
- Benjamin P. Jung
- Brian de Alwis
- Chris Willis
- dependabot[bot]
- elnoro
- Gaurav
- jelle van der Waa
- Marlon Gamez
- Nick Kubala
- Tejal Desai
- Yanshu
- Yuwen Ma


# v1.28.0 Release - 07/14/2021
Note: This release comes with a new config version, `v2beta19`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:
* Skaffold healthchecks can now be run per deployer using the `--iterative-status-check=true` flag (fixes [5774](https://github.com/GoogleContainerTools/skaffold/issues/5774)). See docs [here](https://skaffold.dev/docs/workflows/ci-cd/#waiting-for-skaffold-deployments-using-healthcheck).
* If you use `helm` with `skaffold` you might see a new survey asking for feedback on an upcoming redesign of that integration. 

New Features:
* Allow iterative status checks [#6115](https://github.com/GoogleContainerTools/skaffold/pull/6115)
* Add survey config and framework to show feature surveys to skaffold users. [#6185](https://github.com/GoogleContainerTools/skaffold/pull/6185)

Fixes:
* Make completion work again [#6138](https://github.com/GoogleContainerTools/skaffold/pull/6138)
* Propagate kaniko environment to GCB [#6181](https://github.com/GoogleContainerTools/skaffold/pull/6181)
* Fix couldn't start notify trigger in multi-config projects [#6114](https://github.com/GoogleContainerTools/skaffold/pull/6114)
* Fetch namespaces at time of sync [#6135](https://github.com/GoogleContainerTools/skaffold/pull/6135)
* Replace missing template values with empty string [#6136](https://github.com/GoogleContainerTools/skaffold/pull/6136)
* Fix survey active logic [#6194](https://github.com/GoogleContainerTools/skaffold/pull/6194)
* Don't update survey prompt if survey prompt is not shown to stdout [#6192](https://github.com/GoogleContainerTools/skaffold/pull/6192)
* change ptypes call to timestamppb to fix linters [#6164](https://github.com/GoogleContainerTools/skaffold/pull/6164)

Updates and Refactors:
* Update Skaffold dependencies [#6155](https://github.com/GoogleContainerTools/skaffold/pull/6155)
* Simplify `--timestamps` output [#6146](https://github.com/GoogleContainerTools/skaffold/pull/6146)
* Update Jib build plugin versions after 3.1.2 release [#6168](https://github.com/GoogleContainerTools/skaffold/pull/6168)
* Update feature maturities [#6202](https://github.com/GoogleContainerTools/skaffold/pull/6202)
* Add logic to show user survey in DisplaySurveyPrompt flow. [#6196](https://github.com/GoogleContainerTools/skaffold/pull/6196)
* refactor: Read globalConfig instead of kubecontext config for survey config [#6191](https://github.com/GoogleContainerTools/skaffold/pull/6191)
* Add information about workspace and dockerfile to artifact metadata [#6111](https://github.com/GoogleContainerTools/skaffold/pull/6111)
* Added template expansion for helm chart version (#5709) [#6157](https://github.com/GoogleContainerTools/skaffold/pull/6157)
* add set command for survey ids [#6197](https://github.com/GoogleContainerTools/skaffold/pull/6197)
* Bump schema version to v2beta19 [#6116](https://github.com/GoogleContainerTools/skaffold/pull/6116)

Docs, Test, and Release Updates:
* Create SECURITY.md [#6140](https://github.com/GoogleContainerTools/skaffold/pull/6140)
* Update Jib docs with some advanced usage examples [#6169](https://github.com/GoogleContainerTools/skaffold/pull/6169)
* Disable k3d integration tests [#6171](https://github.com/GoogleContainerTools/skaffold/pull/6171)
* Check release workflow [#6188](https://github.com/GoogleContainerTools/skaffold/pull/6188)
* design proposal to show user survey other than Hats [#6186](https://github.com/GoogleContainerTools/skaffold/pull/6186)
* Doc tweaks [#6176](https://github.com/GoogleContainerTools/skaffold/pull/6176)
* working cloud profiler export for skaffold [#6066](https://github.com/GoogleContainerTools/skaffold/pull/6066)
* Set specific permissions for workflows [#6139](https://github.com/GoogleContainerTools/skaffold/pull/6139)

Huge thanks goes out to all of our contributors for this release:

- Aaron Prindle
- Brian de Alwis
- Chanseok Oh
- Gaurav
- Hidenori Sugiyama
- Marlon Gamez
- Nick Kubala
- Pablo Caderno
- Tejal Desai
- Yuwen Ma

# v1.27.0 Release - 06/29/2021
Note: This release comes with a new config version, `v2beta18`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:
* Skaffold CLI respects `--kube-context` & `--kubeconfig` command line flags and uses it instead of active kubernetes context.
* Status-Check now runs per deployer sequentially. For `skaffold.yaml` with multiple deployers, the next deploy will start after previous deployed resources stabilize. Docs coming soon! 

New Features:
* Configure nodes for running cluster builds (e.g. kaniko) by using the node selector config option `cluster.nodeSelector`. [#6083](https://github.com/GoogleContainerTools/skaffold/pull/6083)
* Better defaults for GCB project when using Artifact Registry images [#6093](https://github.com/GoogleContainerTools/skaffold/pull/6093)
* Skaffold init now supports Jib and Buildpacks artifacts by default [#6063](https://github.com/GoogleContainerTools/skaffold/pull/6063)
* Structured tests configuration supports custom parameters [#6055](https://github.com/GoogleContainerTools/skaffold/pull/6055)

Fixes:
* log metrics upload failure and write to file instead. [#6108](https://github.com/GoogleContainerTools/skaffold/pull/6108)
* Skaffold Render now validates manifests [#6043](https://github.com/GoogleContainerTools/skaffold/pull/6043)
* Port-forwarding improvements for multi-config projects [#6090](https://github.com/GoogleContainerTools/skaffold/pull/6090)
* Fix helm deploy error when configuring helm arg list and skaffold overrides[#6080](https://github.com/GoogleContainerTools/skaffold/pull/6080)
* Use non alpine image and protoc 3.17.3 in proto generation [#6073](https://github.com/GoogleContainerTools/skaffold/pull/6073)
* Fix setting `kubeContext` in skaffold [#6024](https://github.com/GoogleContainerTools/skaffold/pull/6024)
* Use StdEncoding for git hash directory name [#6071](https://github.com/GoogleContainerTools/skaffold/pull/6071)
* fix status-check to return success only on exact success criteria match [#6010](https://github.com/GoogleContainerTools/skaffold/pull/6010)
* fix: gcb api throttling retry backoff not implemented correctly [#6023](https://github.com/GoogleContainerTools/skaffold/pull/6023)
* Ensure events are serialized [#6064](https://github.com/GoogleContainerTools/skaffold/pull/6064)

Updates and Refactors:
* add source file and module to config parsing error description [#6087](https://github.com/GoogleContainerTools/skaffold/pull/6087)
* Refactor to move podSelector, Syncer, StatusCheck, Debugger, Port-forwarder under Deployer [#6076](https://github.com/GoogleContainerTools/skaffold/pull/6076), 
  [#6053](https://github.com/GoogleContainerTools/skaffold/pull/6053), [#6026](https://github.com/GoogleContainerTools/skaffold/pull/6026),
  [#6021](https://github.com/GoogleContainerTools/skaffold/pull/6021),
  [#6044](https://github.com/GoogleContainerTools/skaffold/pull/6044)
* fix v3alpha version [#6084](https://github.com/GoogleContainerTools/skaffold/pull/6084),
* [v2] Update v2 with new UX [#6086](https://github.com/GoogleContainerTools/skaffold/pull/6086)
* Update to github.com/gogo/protobuf v1.3.2 (GO-2021-0053) [#6022](https://github.com/GoogleContainerTools/skaffold/pull/6022)

Docs, Test, and Release Updates:
* Document Helm image reference strategies [#6017](https://github.com/GoogleContainerTools/skaffold/pull/6017)
* Optimize k8s-skaffold/skaffold image [#6106](https://github.com/GoogleContainerTools/skaffold/pull/6106)
* Fix typo in executed file name [#6105](https://github.com/GoogleContainerTools/skaffold/pull/6105)
* Escape parentheses in shJoin [#6101](https://github.com/GoogleContainerTools/skaffold/pull/6101)
* Fix instructions to add actionable error codes. [#6094](https://github.com/GoogleContainerTools/skaffold/pull/6094)
* Updates to ko builder design proposal to add implementation approach [#6046](https://github.com/GoogleContainerTools/skaffold/pull/6046)
* fix invalid config version links in DEVELOPMENT.md [#6058](https://github.com/GoogleContainerTools/skaffold/pull/6058)


Huge thanks goes out to all of our contributors for this release:

- Aaron Prindle
- Brian de Alwis
- Chanseok Oh
- Daniel Petró
- Gaurav
- Halvard Skogsrud
- Jack
- Kaan Karakaya
- Marlon Gamez
- Mridula
- Nick Kubala
- Tejal Desai
- Yuwen Ma

# v1.26.0 Release - 06/08/2021

Highlights:

New Features:
* Emit status check subtask events for V2 API [#5961](https://github.com/GoogleContainerTools/skaffold/pull/5961)
* Buildpacks builder supports mounting read/write volumes (experimental) [#5972](https://github.com/GoogleContainerTools/skaffold/pull/5972)

Fixes:
* Fix and cleanup Kpt fn integration [#5886](https://github.com/GoogleContainerTools/skaffold/pull/5886)
* Avoid adding image digest twice to tag on render [#5958](https://github.com/GoogleContainerTools/skaffold/pull/5958)
* have BuildSubtaskEvent use InProgress status [#5963](https://github.com/GoogleContainerTools/skaffold/pull/5963)
* Ignore first user cancelled and get actual error as final error [#5941](https://github.com/GoogleContainerTools/skaffold/pull/5941)
* Fix up missed remote -> remotePath changes [#5920](https://github.com/GoogleContainerTools/skaffold/pull/5920)
* Add missing flags to `skaffold test` [#5912](https://github.com/GoogleContainerTools/skaffold/pull/5912)

Updates and Refactors:
* make sure SkaffoldLogEvent types go through correct endpoint [#5964](https://github.com/GoogleContainerTools/skaffold/pull/5964)
* update hack/generate-kind-config.sh to handle multiple mirrors [#5977](https://github.com/GoogleContainerTools/skaffold/pull/5977)
* [v3]  Add validator in render v2. [#5942](https://github.com/GoogleContainerTools/skaffold/pull/5942)
* [v3] Add the Kptfile struct to render. [#5940](https://github.com/GoogleContainerTools/skaffold/pull/5940)
* setup /v2/skaffoldLogs endpoint [#5951](https://github.com/GoogleContainerTools/skaffold/pull/5951)
* Refactor to use new SkaffoldWriter type [#5894](https://github.com/GoogleContainerTools/skaffold/pull/5894)
* Show more detailed error when unknown Project [#5939](https://github.com/GoogleContainerTools/skaffold/pull/5939)
* Add event logger type and function to set event context for writer [#5937](https://github.com/GoogleContainerTools/skaffold/pull/5937)
* Remove unsupported `docker.secret.dst` field [#5927](https://github.com/GoogleContainerTools/skaffold/pull/5927)
* Add step field for `BuildSubtaskEvent` to represent the different parts of a build for an artifact [#5915](https://github.com/GoogleContainerTools/skaffold/pull/5915)
* Pass kubeconfig to `kpt live` [#5906](https://github.com/GoogleContainerTools/skaffold/pull/5906)
* Use Helm chart version in render [#5922](https://github.com/GoogleContainerTools/skaffold/pull/5922)
* Add pointer for .NET debugging for Rider [#5924](https://github.com/GoogleContainerTools/skaffold/pull/5924)
* skaffold trace wrapping of critical functions & skaffold trace exporters via SKAFFOLD_TRACE env var [#5854](https://github.com/GoogleContainerTools/skaffold/pull/5854)
* Ensure tag stripping logic can optionally accept digests [#5919](https://github.com/GoogleContainerTools/skaffold/pull/5919)
* Update metadata event emission to happen every devloop and update build metadata [#5918](https://github.com/GoogleContainerTools/skaffold/pull/5918)
* Add additional detail text field for task protos [#5929](https://github.com/GoogleContainerTools/skaffold/pull/5929)
* Add distinct error codes for docker no space error and better suggestion [#5938](https://github.com/GoogleContainerTools/skaffold/pull/5938)
* Add support for Port forwarding with resourceName with Templated Fields [#5934](https://github.com/GoogleContainerTools/skaffold/pull/5934)
* Pause debug pod watchers before next iteration deploy [#5932](https://github.com/GoogleContainerTools/skaffold/pull/5932)

Docs, Test, and Release Updates:
* Add integration tests for `skaffold inspect build-env` commands [#5973](https://github.com/GoogleContainerTools/skaffold/pull/5973)
* Add/fix remoteChart tests [#5921](https://github.com/GoogleContainerTools/skaffold/pull/5921)
* Container Structure Test page should use `skaffold test` [#5911](https://github.com/GoogleContainerTools/skaffold/pull/5911)
* Improve documentation of docker buildArgs (#5871) [#5901](https://github.com/GoogleContainerTools/skaffold/pull/5901)
* Document `inputDigest` tagger, and move `sha256` tagger to end [#5948](https://github.com/GoogleContainerTools/skaffold/pull/5948)

Huge thanks goes out to all of our contributors for this release:

- Aaron Prindle
- Appu
- Brian de Alwis
- Gaurav
- Marlon Gamez
- Mattias Öhrn
- Nick Kubala
- Piotr Wielgolaski
- Rob Thorne
- Tejal Desai
- Yuwen Ma

# v1.25.0 Release - 05/25/2021

Highlights:
* Debug support for pydevd, new --protocols debug flag [#5759](https://github.com/GoogleContainerTools/skaffold/pull/5759)

New Features:
* Allow caching from previously built artifacts on GCB [#5903](https://github.com/GoogleContainerTools/skaffold/pull/5903)

Fixes:
* fix: setting default deployer definition [#5861](https://github.com/GoogleContainerTools/skaffold/pull/5861)
* Schemas should emit type=object for object [#5876](https://github.com/GoogleContainerTools/skaffold/pull/5876)
* Failing to delete source archive should not fail GCB builds [#5891](https://github.com/GoogleContainerTools/skaffold/pull/5891)
* Fix `skaffold diagnose` to work when skaffold.yaml is outside of source root dir [#5900](https://github.com/GoogleContainerTools/skaffold/pull/5900)

Updates and Refactors:
* [V3] add renderer basic struct. [#5793](https://github.com/GoogleContainerTools/skaffold/pull/5793)
* [v3] Add render generator [#5865](https://github.com/GoogleContainerTools/skaffold/pull/5865)
* Move application logs to their own endpoint for API V2 [#5868](https://github.com/GoogleContainerTools/skaffold/pull/5868)
* Update go-containerregistry to 0.5.1 [#5881](https://github.com/GoogleContainerTools/skaffold/pull/5881)
* Update pack to 0.18.1 [#5882](https://github.com/GoogleContainerTools/skaffold/pull/5882)
* Add .NET .csproj detection to init for buildpacks [#5883](https://github.com/GoogleContainerTools/skaffold/pull/5883)
* Refactor metrics prompt functions and change `color` package name [#5890](https://github.com/GoogleContainerTools/skaffold/pull/5890)

Docs, Test, and Release Updates:
* Fix yaml reference rendering for object-type examples [#5872](https://github.com/GoogleContainerTools/skaffold/pull/5872)
* Update _index.md [#5902](https://github.com/GoogleContainerTools/skaffold/pull/5902)
* Rework `debug` docs and add small section on troubleshooting [#5905](https://github.com/GoogleContainerTools/skaffold/pull/5905)

Huge thanks goes out to all of our contributors for this release:

- Asdrubal
- Brian de Alwis
- Gaurav
- Marlon Gamez
- Matthew Michihara
- Tejal Desai
- Yuwen Ma

# v1.24.1 Release - 05/17/2021

**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.24.1/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.24.1/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v1.24.1/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.24.1`

Note: This is a patch release for fixing a regression introduced in v1.24.0 (see [#5840](https://github.com/GoogleContainerTools/skaffold/issues/5840)).

Fixes:
* Fix 5840 [#5858](https://github.com/GoogleContainerTools/skaffold/pull/5858)
* fix skaffold test to use minikube docker-env docker context [#5815](https://github.com/GoogleContainerTools/skaffold/pull/5815)

Updates and Refactors:
* propagate profiles across imported configs by default; disable using `propagate-profiles` flag [#5846](https://github.com/GoogleContainerTools/skaffold/pull/5846)
* add explicit error code `UNKNOWN_API_VERSION` [#5848](https://github.com/GoogleContainerTools/skaffold/pull/5848)
* Expose --event-log-file to render, apply, and test [#5828](https://github.com/GoogleContainerTools/skaffold/pull/5828)
* Bump flask from 1.1.2 to 2.0.0 in /integration/examples [#5822](https://github.com/GoogleContainerTools/skaffold/pull/5822)
* Bump flask from 1.1.2 to 2.0.0 in /examples [#5821](https://github.com/GoogleContainerTools/skaffold/pull/5821)
* Add kpt v1.0.0-alpha.2 to Skaffold image [#5825](https://github.com/GoogleContainerTools/skaffold/pull/5825)
* Avoid aliasing in image configuration [#5804](https://github.com/GoogleContainerTools/skaffold/pull/5804)
* Add support for Port forwarding with namespaces with Templated Fields [#5808](https://github.com/GoogleContainerTools/skaffold/pull/5808)

Huge thanks goes out to all of our contributors for this release:

- Aaron Prindle
- Brian de Alwis
- Gaurav
- Itai Schwartz
- Marlon Gamez
- Nick Kubala
- Yuwen Ma

# v1.24.0 Release - 05/11/2021

**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.24.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.24.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v1.24.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.24.0`

Note: This release comes with a new config version, `v2beta16`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:

New Features:
* Support templated release names in helm render [#5751](https://github.com/GoogleContainerTools/skaffold/pull/5751)
* Add StatusCheck field to skaffold.yaml (#4904) [#5706](https://github.com/GoogleContainerTools/skaffold/pull/5706)
* Add `skaffold inspect` command [#5765](https://github.com/GoogleContainerTools/skaffold/pull/5765)
* bring back error coloring [#5718](https://github.com/GoogleContainerTools/skaffold/pull/5718)
* add `skaffold inspect profiles` command [#5778](https://github.com/GoogleContainerTools/skaffold/pull/5778)
* Emit TaskEvent protos for Test phase [#5786](https://github.com/GoogleContainerTools/skaffold/pull/5786)
* add `skaffold inspect build-env` command [#5792](https://github.com/GoogleContainerTools/skaffold/pull/5792)
* Default digest source to 'remote' in render [#5578](https://github.com/GoogleContainerTools/skaffold/pull/5578)
* Don't add skaffold labels by default in render, and deprecate the --add-skaffold-labels flag [#5653](https://github.com/GoogleContainerTools/skaffold/pull/5653)
* Emit deploy subtask events for API V2 [#5783](https://github.com/GoogleContainerTools/skaffold/pull/5783)
* Allow specifying whether to make file paths absolute when parsing configs. [#5805](https://github.com/GoogleContainerTools/skaffold/pull/5805)
* add emission of TaskEvents for Test phase [#5814](https://github.com/GoogleContainerTools/skaffold/pull/5814)
* Add emission of TestSubtaskEvents [#5816](https://github.com/GoogleContainerTools/skaffold/pull/5816)


Fixes:
* Protect errors.allErrors with mutex [#5753](https://github.com/GoogleContainerTools/skaffold/pull/5753)
* Fix tarring of build context for artifacts with source dependencies [#5750](https://github.com/GoogleContainerTools/skaffold/pull/5750)
* skaffold diagnose to not initialize runconfig for yaml only flag [#5762](https://github.com/GoogleContainerTools/skaffold/pull/5762)
* Ensure working JVM before enabling Jib actions to avoid hangs [#5725](https://github.com/GoogleContainerTools/skaffold/pull/5725)
* Fix setting helm `--setFiles` for Windows [#5648](https://github.com/GoogleContainerTools/skaffold/pull/5648)
* Resolve all filepaths to absolute in 'skaffold diagnose' [#5791](https://github.com/GoogleContainerTools/skaffold/pull/5791)
* Default to empty secret path for Kaniko to use Workload Identity credentials [#5730](https://github.com/GoogleContainerTools/skaffold/pull/5730)
* Use default deployer in 'skaffold apply' [#5776](https://github.com/GoogleContainerTools/skaffold/pull/5776)


Updates and Refactors:
* [V3] (Part 1) Refactor schema "latest" to "latest/v1" [#5728](https://github.com/GoogleContainerTools/skaffold/pull/5728)
* Bump several build dependencies [#5747](https://github.com/GoogleContainerTools/skaffold/pull/5747)
* Consolidate tag stripping logic from Kubernetes logger [#5740](https://github.com/GoogleContainerTools/skaffold/pull/5740)
* [V3] (Part 2) Add new schema to latest/v2 [#5729](https://github.com/GoogleContainerTools/skaffold/pull/5729)
* [Refactor] Move kubernetes log code to pkg/skaffold/kubernetes/logger [#5761](https://github.com/GoogleContainerTools/skaffold/pull/5761)
* update otel libs from v0.13.0 -> v0.20.0 [#5757](https://github.com/GoogleContainerTools/skaffold/pull/5757)
* move profile verification higher up the stack [#5779](https://github.com/GoogleContainerTools/skaffold/pull/5779)
* [V3] (part 1) Change runner/v3 to runner/v2. Update v3 flag to v2 [#5780](https://github.com/GoogleContainerTools/skaffold/pull/5780)
* [v3] (part 2) Move the v1 runner components to `pkg/skaffold/runner/v1`. [#5781](https://github.com/GoogleContainerTools/skaffold/pull/5781)
* [Code style] Fix snake case import package "latest_v1" to "latestV1" [#5799](https://github.com/GoogleContainerTools/skaffold/pull/5799)
* Embed Logger inside Deployer [#5809](https://github.com/GoogleContainerTools/skaffold/pull/5809)


Docs, Test, and Release Updates:
* Update `hack/new_version.sh` script and generate v2beta16 [#5748](https://github.com/GoogleContainerTools/skaffold/pull/5748)
* Use GCR registry mirror in Travis for Linux-based platforms [#5735](https://github.com/GoogleContainerTools/skaffold/pull/5735)
* Update api.md [#5764](https://github.com/GoogleContainerTools/skaffold/pull/5764)
* fix SecurityContext typo [#5769](https://github.com/GoogleContainerTools/skaffold/pull/5769)
* disable housekeeping messages for render [#5770](https://github.com/GoogleContainerTools/skaffold/pull/5770)
* Update _index.md [#5752](https://github.com/GoogleContainerTools/skaffold/pull/5752)
* Update examples/typescript w/ recommended ENV=production info [#5777](https://github.com/GoogleContainerTools/skaffold/pull/5777)
* [v3] Schema Version upgrading for v1 and v2. [#5796](https://github.com/GoogleContainerTools/skaffold/pull/5796)


Designs:
* Add ko builder design proposal draft [#5611](https://github.com/GoogleContainerTools/skaffold/pull/5611)


Huge thanks goes out to all of our contributors for this release:

- Aaron Prindle
- Brian de Alwis
- Gaurav
- Halvard Skogsrud
- Joe Bowbeer
- Maggie Neterval
- Marlon Gamez
- Nick Kubala
- Tejal Desai
- Vladimir Ivanov
- Yuwen Ma
- aleksandrOranskiy


# v1.23.0 Release - 04/27/2021

**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.23.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.23.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v1.23.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.23.0`

Note: This release comes with a new config version, `v2beta15`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:
* Add build-concurrency flag [#5699](https://github.com/GoogleContainerTools/skaffold/pull/5699)
* add skaffold build --push flag [#5708](https://github.com/GoogleContainerTools/skaffold/pull/5708)
* Added fix for RPC port detection [#5715](https://github.com/GoogleContainerTools/skaffold/pull/5715)

  New Features:
* Add BuildSubtask emission for v2 API [#5710](https://github.com/GoogleContainerTools/skaffold/pull/5710)
* Emit TaskEvents Protos for PortForwarding [#5689](https://github.com/GoogleContainerTools/skaffold/pull/5689)
* add host support for docker build [#5698](https://github.com/GoogleContainerTools/skaffold/pull/5698)
* Add taskevents Test, StatusCheck, and fix duplicates for Deploy [#5675](https://github.com/GoogleContainerTools/skaffold/pull/5675)

  Fixes:
* Fix go module names to be unique [#5724](https://github.com/GoogleContainerTools/skaffold/pull/5724)
* Fix Helm deployment check to only retrieve deployed YAML [#5723](https://github.com/GoogleContainerTools/skaffold/pull/5723)

  Updates and Refactors:
* Add `--user` flag to all minikube command invocations [#5732](https://github.com/GoogleContainerTools/skaffold/pull/5732)
* Add user flag with allowed user list to upload metrics [#5731](https://github.com/GoogleContainerTools/skaffold/pull/5731)
* Do not swallow parsing errors [#5722](https://github.com/GoogleContainerTools/skaffold/pull/5722)
* [V3] New V3 SkaffoldRunner [#5692](https://github.com/GoogleContainerTools/skaffold/pull/5692)
* Make status-check flag nillable [#5712](https://github.com/GoogleContainerTools/skaffold/pull/5712)
* Do not display `helm` warnings for multi-config projects [#5468](https://github.com/GoogleContainerTools/skaffold/pull/5468)
* [V3] Add  flag as v3 entrypoint. [#5694](https://github.com/GoogleContainerTools/skaffold/pull/5694)
* Implement pflag slice value interface for image types [#5575](https://github.com/GoogleContainerTools/skaffold/pull/5575)
* upgrade schema to v2beta15 [#5700](https://github.com/GoogleContainerTools/skaffold/pull/5700)

  Docs, Test, and Release Updates:
* Add Cluster Internal Service Error code along with runcontext [#5491](https://github.com/GoogleContainerTools/skaffold/pull/5491)
* Improve multi-config documentation [#5714](https://github.com/GoogleContainerTools/skaffold/pull/5714)
* fix file sync comment in examples/hot-reload/skaffold.yaml [#5693](https://github.com/GoogleContainerTools/skaffold/pull/5693)
* Fix typo [#5688](https://github.com/GoogleContainerTools/skaffold/pull/5688)

Huge thanks goes out to all of our contributors for this release:

- Aaron Prindle
- Boris Lau
- Brian de Alwis
- Gaurav
- Maggie Neterval
- Marlon Gamez
- Matthew Michihara
- Sladyn
- Tejal Desai
- Yuwen Ma
- kelsk
- wuxingzhong

# v1.22.0 Release - 04/14/2021

**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.22.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.22.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
 https://storage.googleapis.com/skaffold/releases/v1.22.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.22.0`

Note: This release comes with a new config version, `v2beta14`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:
* More granular control of `port-forward` options. Checkout the updated [documentation](https://skaffold.dev/docs/pipeline-stages/port-forwarding/) for details. 
* `InputDigest` image tagging strategy from the [Improve taggers proposal](https://github.com/GoogleContainerTools/skaffold/blob/main/docs/design_proposals/digest-tagger.md) has landed.

New Features:
* Revise port-forward behaviour [#5554](https://github.com/GoogleContainerTools/skaffold/pull/5554)
* Add InputDigest support to CustomTemplateTagger [#5661](https://github.com/GoogleContainerTools/skaffold/pull/5661)
* Support --cache-artifacts flag for render [#5652](https://github.com/GoogleContainerTools/skaffold/pull/5652)
* Adding testing phase to Skaffold run [#5594](https://github.com/GoogleContainerTools/skaffold/pull/5594)
* Added helm remote repo example [#5640](https://github.com/GoogleContainerTools/skaffold/pull/5640)

Fixes:
* fix: test dependencies triggering retest for all artifacts [#5679](https://github.com/GoogleContainerTools/skaffold/pull/5679)
* Forwarding resources should not allocate system ports [#5670](https://github.com/GoogleContainerTools/skaffold/pull/5670)
* Make test dependencies retrieval per artifact. [#5678](https://github.com/GoogleContainerTools/skaffold/pull/5678)
* Fix `default-repo` by supporting nil as default value for flags [#5654](https://github.com/GoogleContainerTools/skaffold/pull/5654)
* [kpt deployer] Fix non-kustomize manifests not rendered issue [#5627](https://github.com/GoogleContainerTools/skaffold/pull/5627)
* fix concurrency issue in multi-config [#5646](https://github.com/GoogleContainerTools/skaffold/pull/5646)
* Fix 5301: Build dependencies for sync inherited from `required` artifacts; cache build dependencies between devloops [#5614](https://github.com/GoogleContainerTools/skaffold/pull/5614)
* Fix config line number in DEVELOPMENT.md [#5619](https://github.com/GoogleContainerTools/skaffold/pull/5619)
* fix travis ci md badge link [#5607](https://github.com/GoogleContainerTools/skaffold/pull/5607)
* helm `render` needs to handle `repo` parameter [#5676](https://github.com/GoogleContainerTools/skaffold/pull/5676)
* Add service config to leeroy-web deployment.yaml [#5630](https://github.com/GoogleContainerTools/skaffold/pull/5630)

Updates and Refactors:
* Remove deprecated {{.IMAGES}} and {{.DIGEST_}} env vars [#5605](https://github.com/GoogleContainerTools/skaffold/pull/5605)
* Adding workspace `context` parameter to `test` definitions. [#5677](https://github.com/GoogleContainerTools/skaffold/pull/5677)
* Deprecate --render-only and --render-output flags [#5644](https://github.com/GoogleContainerTools/skaffold/pull/5644)
* Emit TaskEvent messages for DevLoop, Build, and Deploy phases [#5637](https://github.com/GoogleContainerTools/skaffold/pull/5637)
* Update Jib to 3.0 and set base images [#5651](https://github.com/GoogleContainerTools/skaffold/pull/5651)
* Add Event v2 package [#5558](https://github.com/GoogleContainerTools/skaffold/pull/5558)
* Adding events for Test phase [#5573](https://github.com/GoogleContainerTools/skaffold/pull/5573)
* Add instruction to install using Scoop [#5566](https://github.com/GoogleContainerTools/skaffold/pull/5566)
* Try reducing ttl to 30 seconds [#5663](https://github.com/GoogleContainerTools/skaffold/pull/5663)
* Adapting validation for docker container network mode to include ENV_VARS [#5589](https://github.com/GoogleContainerTools/skaffold/pull/5589)
* Set `redeploy` intent only when there are rebuilt artifacts [#5553](https://github.com/GoogleContainerTools/skaffold/pull/5553)
* Add event API v2 server handler [#5622](https://github.com/GoogleContainerTools/skaffold/pull/5622)
* Reset API intents on every dev cycle to avoid queueing [#5636](https://github.com/GoogleContainerTools/skaffold/pull/5636)
* Bring survey prompt back to 10 days and every 90 days. [#5631](https://github.com/GoogleContainerTools/skaffold/pull/5631)

Docs, Test, and Release Updates:
* Document portforwarding behavior for system ports [#5680](https://github.com/GoogleContainerTools/skaffold/pull/5680)
* Updating custom test documentation  [#5606](https://github.com/GoogleContainerTools/skaffold/pull/5606)
* Fix typo in docs site [#5585](https://github.com/GoogleContainerTools/skaffold/pull/5585)

Huge thanks goes out to all of our contributors for this release:

- Brian de Alwis
- Chanseok Oh
- Gaurav
- Ian Danforth
- Maggie Neterval
- Mario Fernández
- Marlon Gamez
- Mike Kamornikov
- Nick Kubala
- Parris Lucas
- Piotr Szybicki
- Priya Modali
- Tejal Desai
- Yury
- Yuwen Ma
- dhodun

# v1.21.0 Release - 03/18/2021

**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.21.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.21.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
 https://storage.googleapis.com/skaffold/releases/v1.21.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.21.0`

Note: This release comes with a new config version, `v2beta13`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:
* Skaffold now supports running custom scripts in the `skaffold test` command or as part of the `Test` phase in `skaffold dev`. For more information and documentation, look [here](https://skaffold.dev/docs/pipeline-stages/testers/custom/).
* New command `skaffold apply` for when you want Skaffold to simply deploy your pre-rendered Kubernetes manifests.
* Debugging Python application using `skaffold debug` now uses `debugpy` by default.
* New tutorials for [`artifact dependencies`](https://skaffold.dev/docs/tutorials/artifact-dependencies/) and [`configuration dependencies`](https://skaffold.dev/docs/tutorials/config-dependencies/) features.
* New example project [`examples/custom-buildx`](https://github.com/GoogleContainerTools/skaffold/tree/main/examples/custom-buildx) showing how to build multi-arch images using Skaffold.

New Features:
* Add 'skaffold apply' command [#5543](https://github.com/GoogleContainerTools/skaffold/pull/5543)
* Implement custom tester functionality in Skaffold [#5451](https://github.com/GoogleContainerTools/skaffold/pull/5451)
* Adding support for accessing built images in custom test [#5535](https://github.com/GoogleContainerTools/skaffold/pull/5535)
* Adding support for re-triggering tests when test dependency changes [#5533](https://github.com/GoogleContainerTools/skaffold/pull/5533)
* `skaffold debug` rewrites probe timeouts to avoid container restarts [#5474](https://github.com/GoogleContainerTools/skaffold/pull/5474)
* Add digest source 'tag' to use tag without digest [#5436](https://github.com/GoogleContainerTools/skaffold/pull/5436)
* Simpler host:port formatting for port-forwards [#5488](https://github.com/GoogleContainerTools/skaffold/pull/5488)


Fixes:
* Fix issue where default-repo wasn't being added to artifact tags (#5341) [#5397](https://github.com/GoogleContainerTools/skaffold/pull/5397)
* Fix setting of defaults for flags [#5548](https://github.com/GoogleContainerTools/skaffold/pull/5548)
* fix: `skaffold diagnose` outputs incorrect `yaml` for multiconfig projects [#5531](https://github.com/GoogleContainerTools/skaffold/pull/5531)
* fix logic that set absolute paths in the parsed configuration [#5452](https://github.com/GoogleContainerTools/skaffold/pull/5452)
* decouple `helm` deployer `chartPath` into `chartPath` and `remoteChart` [#5482](https://github.com/GoogleContainerTools/skaffold/pull/5482)
* Fixes #5404: Skaffold configs downloaded from a url can define remote config dependencies [#5405](https://github.com/GoogleContainerTools/skaffold/pull/5405)
* Add golint support for M1 macs (darwin/arm64 arch) [#5435](https://github.com/GoogleContainerTools/skaffold/pull/5435)
* Handle nil PortForward item on setting defaults [#5416](https://github.com/GoogleContainerTools/skaffold/pull/5416)
* skip validating Dockerfile using explicit syntax directive [#5441](https://github.com/GoogleContainerTools/skaffold/pull/5441)
* add explicit error codes for various config parsing errors [#5483](https://github.com/GoogleContainerTools/skaffold/pull/5483)
* Adding distinct error codes for custom test failures [#5501](https://github.com/GoogleContainerTools/skaffold/pull/5501)
* fix: Parsing fails for named multistage dockerfile using build artifact dependency [#5507](https://github.com/GoogleContainerTools/skaffold/pull/5507)

Updates and Refactors:
* Update 2021 Roadmap [#5514](https://github.com/GoogleContainerTools/skaffold/pull/5514)
* Give v2 proto different package name [#5557](https://github.com/GoogleContainerTools/skaffold/pull/5557)
* update v2/ proto files [#5512](https://github.com/GoogleContainerTools/skaffold/pull/5512)
* [refactor] Move tag package outside of build [#5547](https://github.com/GoogleContainerTools/skaffold/pull/5547)
* add number of releases in helm config [#5552](https://github.com/GoogleContainerTools/skaffold/pull/5552)
* Refactoring events to use Config interface for init [#5532](https://github.com/GoogleContainerTools/skaffold/pull/5532)
* add debug/iterations metric [#5359](https://github.com/GoogleContainerTools/skaffold/pull/5359)
* Update pack to 0.17.0 with Platform API 0.5 [#5360](https://github.com/GoogleContainerTools/skaffold/pull/5360)
* Update gcr.io/k8s-skaffold/pack to 0.17.0 [#5430](https://github.com/GoogleContainerTools/skaffold/pull/5430)
* Update Jib to 2.8.0 [#5457](https://github.com/GoogleContainerTools/skaffold/pull/5457)
* Add support for no-option-value; surface per-option-defaults [#5447](https://github.com/GoogleContainerTools/skaffold/pull/5447)
* Add metric for the count of skaffold configurations in current session; fix the build platform type metric to save list of all platforms [#5437](https://github.com/GoogleContainerTools/skaffold/pull/5437)
* Add repo field to helm release [#5410](https://github.com/GoogleContainerTools/skaffold/pull/5410)
* Update Paketo buildpack references [#5446](https://github.com/GoogleContainerTools/skaffold/pull/5446)
* Allow Argo Rollout resource manifests to be transformed (#5523) [#5524](https://github.com/GoogleContainerTools/skaffold/pull/5524)
* `skaffold debug` should use debugpy for Python [#5576](https://github.com/GoogleContainerTools/skaffold/pull/5576)
* Revise port-forwarding behaviour [#4832](https://github.com/GoogleContainerTools/skaffold/pull/4832)
* Add -o shorthand for --output flag to skaffold render [#5526](https://github.com/GoogleContainerTools/skaffold/pull/5526)

Docs, Test, and Release Updates:
* Documentation for Custom Test in Skaffold [#5521](https://github.com/GoogleContainerTools/skaffold/pull/5521)
* Fix custom build example [#5495](https://github.com/GoogleContainerTools/skaffold/pull/5495)
* add tutorial for buildpacks run image override [#5409](https://github.com/GoogleContainerTools/skaffold/pull/5409)
* Update DEVELOPMENT.md to include new information about changes to .proto files [#5506](https://github.com/GoogleContainerTools/skaffold/pull/5506)
* Add example to use docker buildx via the custom builder [#5426](https://github.com/GoogleContainerTools/skaffold/pull/5426)
* Link relevant Cloud Shell tutorials in doc site [#5545](https://github.com/GoogleContainerTools/skaffold/pull/5545)

Huge thanks goes out to all of our contributors for this release:

- AB
- Bobby Richard
- Brian de Alwis
- Dan
- Felix Beuke
- Feng Ye
- Gaurav
- Gregory Moon
- Idan Bidani
- Isaac Duarte
- Marlon Gamez
- Matthieu Blottière
- Mridula
- Nick Kubala
- Piotr Szybicki
- Priya Modali
- Ricardo La Rosa
- Ryan Moran
- Shin Jinwoo
- Tejal Desai
- dhodun

# v1.20.0 Release - 02/11/2021

**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.20.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.20.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
 https://storage.googleapis.com/skaffold/releases/v1.20.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.20.0`

Note: This release comes with a new config version, `v2beta12`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:
* Skaffold now supports defining remote git dependencies in the project configuration!

New Features:
* Implement defining remote git dependencies in the skaffold configuration. [#5361](https://github.com/GoogleContainerTools/skaffold/pull/5361)

Fixes:
* Fix absolute path substitution in configs imported as dependencies. [#5389](https://github.com/GoogleContainerTools/skaffold/pull/5389)
* Update dependencies to fix `getCPUInfo` error on darwin/arm64 [#5351](https://github.com/GoogleContainerTools/skaffold/pull/5351)
* Configure k3d to use registry-mirrors [#5344](https://github.com/GoogleContainerTools/skaffold/pull/5344)
* fix pulling secrets in cloudbuild release for latest builds [#5328](https://github.com/GoogleContainerTools/skaffold/pull/5328)

Updates and Refactors:
* Add error codes for test failures [#5385](https://github.com/GoogleContainerTools/skaffold/pull/5385)
* use status code string in error label [#5350](https://github.com/GoogleContainerTools/skaffold/pull/5350)
* track the platform type at the launch level [#5353](https://github.com/GoogleContainerTools/skaffold/pull/5353)
* Unhide --auto-{build,deploy,sync} and update debug notes [#5347](https://github.com/GoogleContainerTools/skaffold/pull/5347)
* Update telemetry prompt links [#5346](https://github.com/GoogleContainerTools/skaffold/pull/5346)
* export metrics related to user enum flags [#5322](https://github.com/GoogleContainerTools/skaffold/pull/5322)
* add `build-dependencies` metric [#5330](https://github.com/GoogleContainerTools/skaffold/pull/5330)
* Add prompt for users to pick manifests to generate during `skaffold init --generate-manifests` [#5312](https://github.com/GoogleContainerTools/skaffold/pull/5312)

Docs, Test, and Release Updates:
* Document steps to use sErrors.ErrDef class to provide actionable error messages [#5375](https://github.com/GoogleContainerTools/skaffold/pull/5375)
* Update docs with darwin/arm64 binaries [#5287](https://github.com/GoogleContainerTools/skaffold/pull/5287)
* Update release to build darwin/arm64 binary [#5286](https://github.com/GoogleContainerTools/skaffold/pull/5286)
* TypeScript support for the existing Node.js example [#5325](https://github.com/GoogleContainerTools/skaffold/pull/5325)
* Fix example `multi-config-microservices` broken due to missed runtime image update [#5337](https://github.com/GoogleContainerTools/skaffold/pull/5337)
* Make integration.WaitForPodsReady use pod `Ready` condition [#5308](https://github.com/GoogleContainerTools/skaffold/pull/5308)
* refactor instrumentation package [#5324](https://github.com/GoogleContainerTools/skaffold/pull/5324)
* add integration test for `skaffold init --artifact` [#5319](https://github.com/GoogleContainerTools/skaffold/pull/5319)
* add docs for `config dependencies` feature [#5321](https://github.com/GoogleContainerTools/skaffold/pull/5321)
* Update language runtime image versions [#5307](https://github.com/GoogleContainerTools/skaffold/pull/5307)

Huge thanks goes out to all of our contributors for this release:

- Alex Ashley
- Brian de Alwis
- Gaurav
- Isaac Duarte
- Marlon Gamez
- Nick Kubala
- Pat
- Priya Modali
- Tejal Desai


# v1.19.0 Release - 01/28/2021

**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.19.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.19.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
 https://storage.googleapis.com/skaffold/releases/v1.19.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.19.0`

From release v1.19.0, skaffold will collect anonymized Skaffold usage data.

You are **opted-in** by default and you can opt-out at any time with the skaffold config command. 

Learn more on what data is reported [here](https://skaffold.dev/docs/resources/telemetry/#example)
and [how to disable usage collection](https://skaffold.dev/docs/resources/telemetry)

Note: This is a small release with few improvements to `skaffold init` and skaffold documentation.

Huge thanks goes out to all of our contributors for this release:

- Brian de Alwis
- Isaac Duarte
- Jeff Wu
- Marlon Gamez
- Medya Ghazizadeh
- Priya Modali
- Sangeetha A

# v1.18.0 Release - 01/21/2021
**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.18.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.18.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
 https://storage.googleapis.com/skaffold/releases/v1.18.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.18.0`

Note: This release comes with a new config version, `v2beta11`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:
* Skaffold now supports providing multiple configuration files in a single session. This enables logical grouping of build artifacts and deployment configurations into modules, which can be individually selected for iteration during a dev session while still performing one time deployment of other modules alongside. For more information and documentation, see [our design document](tinyurl.com/skaffold-multi-configs).
* Skaffold now provides a standalone command `skaffold test` for running supported test implementations outside of a dev session.

New Features:
* Config dependencies [#5217](https://github.com/GoogleContainerTools/skaffold/pull/5217)
* Create nsis installer script [#5233](https://github.com/GoogleContainerTools/skaffold/pull/5233)
* Add transparent init [#5186](https://github.com/GoogleContainerTools/skaffold/pull/5186)
* Add go tag `filepath` that converts marked fields to absolute paths [#5205](https://github.com/GoogleContainerTools/skaffold/pull/5205)
* Allow multiple configs in single skaffold.yaml [#5199](https://github.com/GoogleContainerTools/skaffold/pull/5199)
* Add flag that prints timestamps in skaffold logs. [#5181](https://github.com/GoogleContainerTools/skaffold/pull/5181)
* Enable multi config support in Skaffold [#5160](https://github.com/GoogleContainerTools/skaffold/pull/5160)
* Adding new test command [#5118](https://github.com/GoogleContainerTools/skaffold/pull/5118)
* add transparent init function [#5155](https://github.com/GoogleContainerTools/skaffold/pull/5155)
* Add flag to save events to a file [#5125](https://github.com/GoogleContainerTools/skaffold/pull/5125)


Fixes:
* group all configs in each example project prior to validation test. [#5274](https://github.com/GoogleContainerTools/skaffold/pull/5274)
* Fix race condition when running tests for pkg/skaffold/instrumentation [#5267](https://github.com/GoogleContainerTools/skaffold/pull/5267)
* fix termination event not being sent [#5258](https://github.com/GoogleContainerTools/skaffold/pull/5258)
* Fix statik directory structure [#5250](https://github.com/GoogleContainerTools/skaffold/pull/5250)
* Fix test timeout failures in TestDebug/helm [#5252](https://github.com/GoogleContainerTools/skaffold/pull/5252)
* Configure maven connection pool TTL to avoid connection resets from stale connections [#5251](https://github.com/GoogleContainerTools/skaffold/pull/5251)
* Avoid possible hang in util.RunCmdOut by using byte buffers instead of pipes [#5220](https://github.com/GoogleContainerTools/skaffold/pull/5220)
* Enable MTU path discovery on Linux and setup registry-mirrors for kind [#5237](https://github.com/GoogleContainerTools/skaffold/pull/5237)
* Ensure init generates /-delimted paths [#5177](https://github.com/GoogleContainerTools/skaffold/pull/5177)
* Modify Travis directive restricting builds to master [#5219](https://github.com/GoogleContainerTools/skaffold/pull/5219)
* Set meter.Command before an error could occur with createNewRunner [#5168](https://github.com/GoogleContainerTools/skaffold/pull/5168)
* Fix port forward [#5225](https://github.com/GoogleContainerTools/skaffold/pull/5225)
* yaml encoders should be flushed. [#5196](https://github.com/GoogleContainerTools/skaffold/pull/5196)
* fix: lookup image id with tag rather than name during tryImport [#5165](https://github.com/GoogleContainerTools/skaffold/pull/5165)
* Profile with multiple activations should be processed only once. [#5182](https://github.com/GoogleContainerTools/skaffold/pull/5182)
* Fail Helm deployments early with missing templated values [#5158](https://github.com/GoogleContainerTools/skaffold/pull/5158)
* Fix `skaffold debug` for helm charts with skaffold config file other than default `skaffold.yaml` [#5138](https://github.com/GoogleContainerTools/skaffold/pull/5138)


Updates and Refactors:
* Configure Maven Wagon HTTP to retry on errors to successful connections [#5268](https://github.com/GoogleContainerTools/skaffold/pull/5268)
* Use gcr.io/google-appengine/openjdk:8 to avoid toomanyrequests [#5256](https://github.com/GoogleContainerTools/skaffold/pull/5256)
* change default status check timeout to 10 minutes [#5247](https://github.com/GoogleContainerTools/skaffold/pull/5247)
* Include commands and directory in run output [#5254](https://github.com/GoogleContainerTools/skaffold/pull/5254)
* Embed metrics credentials and upload metrics if they are present [#5157](https://github.com/GoogleContainerTools/skaffold/pull/5157)
* write metrics to file [#5135](https://github.com/GoogleContainerTools/skaffold/pull/5135)
* Capture Errors and dev iterations metrics [#5105](https://github.com/GoogleContainerTools/skaffold/pull/5105)
* Update jib to 2.7.1 [#5223](https://github.com/GoogleContainerTools/skaffold/pull/5223)
* Create a custom unmarshler for Volumes and VolumeMounts to fix #4175 [#5039](https://github.com/GoogleContainerTools/skaffold/pull/5039)
* Render uses Helm templated values-file [#5170](https://github.com/GoogleContainerTools/skaffold/pull/5170)
* enable `detect-minikube` by default. [#5154](https://github.com/GoogleContainerTools/skaffold/pull/5154)
* Unhide XXenableManifestGeneration for skaffold init, remove unnecessary print line [#5152](https://github.com/GoogleContainerTools/skaffold/pull/5152)
* issue #5076 Skaffold support for docker build '--squash' flag  [#5078](https://github.com/GoogleContainerTools/skaffold/pull/5078)
* remove wsl detection logic [#5124](https://github.com/GoogleContainerTools/skaffold/pull/5124)
* enabling using another container's network stack on build process [#5088](https://github.com/GoogleContainerTools/skaffold/pull/5088)


Docs, Test, and Release Updates:
* Add more unit tests for creating metrics, fix bug related to unmarshalling flags [#5169](https://github.com/GoogleContainerTools/skaffold/pull/5169)
* Release automation changes [#5203](https://github.com/GoogleContainerTools/skaffold/pull/5203)
* some fixes on documents [#5211](https://github.com/GoogleContainerTools/skaffold/pull/5211)
* [doc] fix profile activation sample [#5222](https://github.com/GoogleContainerTools/skaffold/pull/5222)
* Update boilerplate.py year check. [#5212](https://github.com/GoogleContainerTools/skaffold/pull/5212)
* CNCF Buildpacks => Cloud Native Buildpacks [#5202](https://github.com/GoogleContainerTools/skaffold/pull/5202)
* [KPT CODELAB] (3/3) kpt deployment & pruning [#5028](https://github.com/GoogleContainerTools/skaffold/pull/5028)
* update AlecAivazis/survey to v2 [#5129](https://github.com/GoogleContainerTools/skaffold/pull/5129)


Huge thanks goes out to all of our contributors for this release:

- Andrey Shlykov
- Appu
- Brian de Alwis
- Chanseok Oh
- Chulki Lee
- Gaurav
- Gunadhya
- Isaac Duarte
- Jakob Schmutz
- Jeff Wu
- Jeremy Lewi
- Mario Fernández
- Marlon Gamez
- Nick Kubala
- Priya Modali
- Saeid Bostandoust
- Tejal Desai
- Yuwen Ma
- Zbigniew Mandziejewicz
- mblottiere
- priyawadhwa
- rpunia7

# v1.17.2 Release - 12/08/2020
**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.17.2/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.17.2/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
 https://storage.googleapis.com/skaffold/releases/v1.17.2/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.17.2`

This is a minor release with a fix for sync issue for docker artifacts in `skaffold dev`. See [#5110](https://github.com/GoogleContainerTools/skaffold/issues/5110) & [#5115](https://github.com/GoogleContainerTools/skaffold/issues/5115)

Fixes:
* Recompute docker dependencies across dev loops. [#5121](https://github.com/GoogleContainerTools/skaffold/pull/5121)


# v1.17.1 Release - 12/01/2020
**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.17.1/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`
 
**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.17.1/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`
 
**Windows**
 https://storage.googleapis.com/skaffold/releases/v1.17.1/skaffold-windows-amd64.exe
 
**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.17.1`
 
This is a minor release with few updates.

Highlights:
* Improve deployment times to local kind/k3d by setting `kind-disable-load` and `k3d-disable-load` to true in global config [#5012](https://github.com/GoogleContainerTools/skaffold/pull/5012)

Fixes:
* Change default kaniko image to `gcr.io/k8s-skaffold/skaffold-helpers/busybox` from `busybox` [#5080](https://github.com/GoogleContainerTools/skaffold/pull/5080)
* Support multi-level repos for Artifact Registry [#5053](https://github.com/GoogleContainerTools/skaffold/pull/5053)

Updates:
* Add distinct error codes for all deploy errors [#5070](https://github.com/GoogleContainerTools/skaffold/pull/5070)
* Bump k8s and docker client library deps [#5038](https://github.com/GoogleContainerTools/skaffold/pull/5038)
* add docker build distinct error codes [#5059](https://github.com/GoogleContainerTools/skaffold/pull/5059)
* add jib tool errors [#5068](https://github.com/GoogleContainerTools/skaffold/pull/5068)
* Update to pack 0.15 and add debug support for CNB Platform API 0.4 [#5064](https://github.com/GoogleContainerTools/skaffold/pull/5064)


Huge thanks goes out to all of our contributors for this release:

- Brian de Alwis
- Gaurav
- Halvard Skogsrud
- Isaac Duarte
- Marlon Gamez
- Nick Kubala
- Tejal Desai
- Thomas Strömberg
- Zbigniew Mandziejewicz

# v1.17.0 Release - 11/23/2020
**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.17.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`
 
**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.17.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`
 
**Windows**
 https://storage.googleapis.com/skaffold/releases/v1.17.0/skaffold-windows-amd64.exe
 
**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.17.0`
 
Note: This release comes with a new config version, `v2beta10`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:
* Helm 2 support has been removed from Skaffold! Read [Helm's blog posts](https://helm.sh/blog/charts-repo-deprecation/) for more info.
* Build artifact dependencies can now be specified for all natively supported builders

New Features:
* Expand the skaffold init --artifact API to allow specifying artifact context [#5000](https://github.com/GoogleContainerTools/skaffold/pull/5000)
* resolve environment variables in helm template keys [#4899](https://github.com/GoogleContainerTools/skaffold/pull/4899)
* Add default dockerfile path to skaffold config when using skaffold init [#4989](https://github.com/GoogleContainerTools/skaffold/pull/4989)
* Implement required artifact resolution in buildpacks builder [#4962](https://github.com/GoogleContainerTools/skaffold/pull/4962)
* Implement required artifact resolution for custom builder [#4972](https://github.com/GoogleContainerTools/skaffold/pull/4972)
* Implement required artifact resolution for cluster builder [#4992](https://github.com/GoogleContainerTools/skaffold/pull/4992)
* Implement required artifact resolution in jib builder [#4997](https://github.com/GoogleContainerTools/skaffold/pull/4997)
* Implement artifact resolution for all GCB builders. [#5003](https://github.com/GoogleContainerTools/skaffold/pull/5003)

Fixes:
* `port-forward` should be able to select ports by service name [#5009](https://github.com/GoogleContainerTools/skaffold/pull/5009)
* GitTagger generates an invalid tag if there are uncommitted changes [#5034](https://github.com/GoogleContainerTools/skaffold/pull/5034)
* Fix Bug that prevents showing survey prompt [#5027](https://github.com/GoogleContainerTools/skaffold/pull/5027)
* Enable running tests for cached images [#5013](https://github.com/GoogleContainerTools/skaffold/pull/5013)
* Fix: Skaffold reloads unchanged, existing image again into the cluster [#4983](https://github.com/GoogleContainerTools/skaffold/pull/4983)
* Fix 4950: User-defined port forwarding resources ignore the namespace flag [#4987](https://github.com/GoogleContainerTools/skaffold/pull/4987)
* Fix Kaniko build args eval from config `env`. [#5002](https://github.com/GoogleContainerTools/skaffold/pull/5002)
* Added logic to handle nil during interface conversion of namespace. [#5001](https://github.com/GoogleContainerTools/skaffold/pull/5001)
* Do not log pruner context errors when Skaffold process is interrupted [#4894](https://github.com/GoogleContainerTools/skaffold/pull/4894)
* Fix description of some kaniko flags [#4988](https://github.com/GoogleContainerTools/skaffold/pull/4988)
* Only print port forward success message on actual success [#4968](https://github.com/GoogleContainerTools/skaffold/pull/4968)
* skaffold init --force supports cases with 1 image and multiple builders [#4973](https://github.com/GoogleContainerTools/skaffold/pull/4973)
* Fix parsing invalid Dockerfile [#4943](https://github.com/GoogleContainerTools/skaffold/pull/4943)

Updates:
* Support ko image references [#4952](https://github.com/GoogleContainerTools/skaffold/pull/4952)
* Replace util.SyncStore implementation to use singleflight and sync.Map. [#5016](https://github.com/GoogleContainerTools/skaffold/pull/5016)
* Remove support for Helm 2 [#5019](https://github.com/GoogleContainerTools/skaffold/pull/5019)
* Output logs in color for parallel builds [#5014](https://github.com/GoogleContainerTools/skaffold/pull/5014)
* Support jsonnet as configuration source. [#4855](https://github.com/GoogleContainerTools/skaffold/pull/4855)
* Cache `docker.getDependencies`  and skip inspecting remote images with old manifest [#4896](https://github.com/GoogleContainerTools/skaffold/pull/4896)
* Loose kustomize version requirements [#4994](https://github.com/GoogleContainerTools/skaffold/pull/4994)
* Adding distinct exit codes for cluster connection failures. [#4933](https://github.com/GoogleContainerTools/skaffold/pull/4933)

Docs Updates:
* Multi version docs [#5048](https://github.com/GoogleContainerTools/skaffold/pull/5048)
* Documentation - CI/CD Tutorial End to End with Skaffold [#4909](https://github.com/GoogleContainerTools/skaffold/pull/4909)
* [Proposal] Transparent skaffold init [#4915](https://github.com/GoogleContainerTools/skaffold/pull/4915)
* [KPT CODELAB] (1/3) New codelab dir + the sample application resources  [#5023](https://github.com/GoogleContainerTools/skaffold/pull/5023)
* Update `artifact-dependencies` status to `implemented` [#5021](https://github.com/GoogleContainerTools/skaffold/pull/5021)
* Force correct font to make magnifying glass appear [#5017](https://github.com/GoogleContainerTools/skaffold/pull/5017)
* Doc update; new example; new tutorial for artifact dependencies [#4971](https://github.com/GoogleContainerTools/skaffold/pull/4971)
* Update documentation for builders around artifact dependency. [#4998](https://github.com/GoogleContainerTools/skaffold/pull/4998)
* fix examples to appropriate type [#4974](https://github.com/GoogleContainerTools/skaffold/pull/4974)

Huge thanks goes out to all of our contributors for this release:

- Andrey Shlykov
- Gaurav
- Halvard Skogsrud
- Isaac
- Marlon Gamez
- Nick Kubala
- Priya Modali
- Ricardo La Rosa
- Sören Bohn
- Tejal Desai
- Vignesh T.V
- Yuwen Ma
- ilya-zuyev


# v1.16.0 Release - 10/27/2020
**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.16.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`
 
**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.16.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`
 
**Windows**
 https://storage.googleapis.com/skaffold/releases/v1.16.0/skaffold-windows-amd64.exe
 
**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.16.0`
 
Note: This release comes with a new config version, `v2beta9`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.


Highlights:
* Artifact Modules Support: Skaffold allow users to specify artifact dependencies for dockerfile artifacts. 

  To use, look at [microservice example](https://github.com/GoogleContainerTools/skaffold/tree/main/examples/microservices). *Docs coming soon*
* Skaffold init support for polyglot-maven projects [#4871](https://github.com/GoogleContainerTools/skaffold/pull/4871)
* Skaffold `debug` helper images now moved to gcr.io/k8s-skaffold/skaffold-debug-support [#4961](https://github.com/GoogleContainerTools/skaffold/pull/4961)

New Features:
* Implement env variable expansion for kaniko builds  [#4557](https://github.com/GoogleContainerTools/skaffold/pull/4557)
* Make it possible to disable auto-sync for buildpacks builder  [#4923](https://github.com/GoogleContainerTools/skaffold/pull/4923)
* Mute status check logs [#4907](https://github.com/GoogleContainerTools/skaffold/pull/4907)
* Support minikube 1.13.0 and later with `--vm-driver=none` [#4887](https://github.com/GoogleContainerTools/skaffold/pull/4887)
* Support kaniko v1.0.0 flags [#4900](https://github.com/GoogleContainerTools/skaffold/pull/4900)
* Prune prev images on build/run/{dev iteration start} [#4792](https://github.com/GoogleContainerTools/skaffold/pull/4792)
* [alpha] Support for deploying and hydrating manifests using [`kpt`](https://googlecontainertools.github.io/kpt/)
* Introduce `fromImage` field in jib builder interface. [#4873](https://github.com/GoogleContainerTools/skaffold/pull/4873)

Fixes:
* Fix `debug` for Helm on Windows [#4872](https://github.com/GoogleContainerTools/skaffold/pull/4872)
* validate tag policy constrain [#4890](https://github.com/GoogleContainerTools/skaffold/pull/4890)
* Don't single-quote SKAFFOLD_GO_GCFLAGS [#4864](https://github.com/GoogleContainerTools/skaffold/pull/4864)
* Fix return of error adding artifacts to cache when images are built remotely [#4850](https://github.com/GoogleContainerTools/skaffold/pull/4850)
* Only load images into k3d and kind when images are local [#4869](https://github.com/GoogleContainerTools/skaffold/pull/4869)

Updates:
* Name debug helper containers more explicitly [#4946](https://github.com/GoogleContainerTools/skaffold/pull/4946)
* Add an init phase to detect skaffold errors even before skaffold runner is created. [#4926](https://github.com/GoogleContainerTools/skaffold/pull/4926)
* Update build_deps versions to latest [#4910](https://github.com/GoogleContainerTools/skaffold/pull/4910)
* [errors] Add distinct error codes for docker not running [#4914](https://github.com/GoogleContainerTools/skaffold/pull/4914)
* Update mute-logs to not print output upon failure of build/deploy step [#4833](https://github.com/GoogleContainerTools/skaffold/pull/4833)

Docs Updates:
* Fix up doc: debug works for buildpacks [#4948](https://github.com/GoogleContainerTools/skaffold/pull/4948)

Huge thanks goes out to all of our contributors for this release:

- Andrey Shlykov
- Appu
- Brian de Alwis
- Daniel Sel
- Dustin Deus
- Gaurav
- Marlon Gamez
- Ricardo La Rosa
- Tejal Desai
- fang duan
- ilya-zuyev


# v1.15.0 Release - 09/29/2020

**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.15.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`
 
**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.15.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`
 
**Windows**
 https://storage.googleapis.com/skaffold/releases/v1.15.0/skaffold-windows-amd64.exe
 
**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.15.0`
 
Note: This release comes with a new config version, `v2beta8`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.


Highlights:
* `skaffold debug` now supports Helm applications!
* Quickstart docs now open directly in Cloud Shell
* Skaffold now falls back to Kubectl's built-in Kustomize if standalone binary not present
* Helm deployments can now optionally create a namespace before deploying
* Kubectl and Kustomize deployments can now configure a default namespace

New Features:
* skaffold deploy -t flag [#4778](https://github.com/GoogleContainerTools/skaffold/pull/4778)
* Add fallback to kubectl kustomize if kustomize binary isn't present [#4484](https://github.com/GoogleContainerTools/skaffold/pull/4484)
* Support docker build --secret flag [#4731](https://github.com/GoogleContainerTools/skaffold/pull/4731)
* Adds support for `debug` for Helm [#4732](https://github.com/GoogleContainerTools/skaffold/pull/4732)
* Support configuring default namespace for kubectl/kustomize deployers [#4374](https://github.com/GoogleContainerTools/skaffold/pull/4374)
* Add option for helm deployments to create namespace [#4765](https://github.com/GoogleContainerTools/skaffold/pull/4765)

Fixes:
* Fix `skaffold filter` to handle multiple yaml documents [#4829](https://github.com/GoogleContainerTools/skaffold/pull/4829)
* Fix `skaffold build | skaffold deploy` with `SKAFFOLD_DEFAULT_REPO` rewrites image name twice [#4817](https://github.com/GoogleContainerTools/skaffold/pull/4817)
* Ensure Windows console color enablement [#4798](https://github.com/GoogleContainerTools/skaffold/pull/4798)
* Add option to skaffold run similar to "skaffold build -b" [#4734](https://github.com/GoogleContainerTools/skaffold/pull/4734)
* Return error for invalid dockerfile instruction "COPY" with no arguments [#4795](https://github.com/GoogleContainerTools/skaffold/pull/4795)
* Make FakeAPIClient threadsafe [#4790](https://github.com/GoogleContainerTools/skaffold/pull/4790)
* Pass correct build args to `CreateDockerTarContext` [#4768](https://github.com/GoogleContainerTools/skaffold/pull/4768)
* Surface error for render [#4758](https://github.com/GoogleContainerTools/skaffold/pull/4758)

Updates:
* [kpt deployer] Customize the manipulated resource directory. [#4819](https://github.com/GoogleContainerTools/skaffold/pull/4819)
* Pass docker.Config instead of InsecureRegistries [#4755](https://github.com/GoogleContainerTools/skaffold/pull/4755)
* Move deployers into separate packages [#4812](https://github.com/GoogleContainerTools/skaffold/pull/4812)
* [kpt deployer] Add "local-config" annotation to kpt fn configs. [#4803](https://github.com/GoogleContainerTools/skaffold/pull/4803)
* [kpt deployer] Improve skaffold.yaml docs. [#4799](https://github.com/GoogleContainerTools/skaffold/pull/4799)
* Try to import docker images before falling back to building [#3891](https://github.com/GoogleContainerTools/skaffold/pull/3891)
* Make flag order deterministic for helm's `--setFiles` [#4779](https://github.com/GoogleContainerTools/skaffold/pull/4779)
* Expand home directory for setFiles in helm deployment [#4619](https://github.com/GoogleContainerTools/skaffold/pull/4619)
* Pass a context to DefaultAuthHelper.GetAllConfigs() [#4760](https://github.com/GoogleContainerTools/skaffold/pull/4760)

Docs Updates:
* Add Cloud Shell for simpler Quickstart [#4830](https://github.com/GoogleContainerTools/skaffold/pull/4830)
* Add design proposal for supporting dependencies between build artifacts [#4794](https://github.com/GoogleContainerTools/skaffold/pull/4794)
* Document how to disable autosync for buildpacks [#4805](https://github.com/GoogleContainerTools/skaffold/pull/4805)
* Clarify usage of ArtifactOverrides, ImageStrategy [#4487](https://github.com/GoogleContainerTools/skaffold/pull/4487)


Huge thanks goes out to all of our contributors for this release:

- Alexander Lyon
- Andreas Sommer
- Andrey Shlykov
- Brian de Alwis
- David Gageot
- Gaurav
- Kri5
- Marlon Gamez
- Nick Kubala
- Paul "TBBle" Hampson
- Thomas Strömberg
- Yuwen Ma
- ilya-zuyev

# v1.14.0 Release - 09/02/2020

**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.14.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`
 
**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.14.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`
 
**Windows**
 https://storage.googleapis.com/skaffold/releases/v1.14.0/skaffold-windows-amd64.exe
 
**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.14.0`
 
Note: This release comes with a new config version, `v2beta7`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.


Highlights:
* Skaffold can now detect minikube clusters regardless of profile name
* Skaffold now has support for debugging .NET Core containers
* Statuscheck phase is seeing some UX improvements


New Features:
* Build debuggable containers [#4606](https://github.com/GoogleContainerTools/skaffold/pull/4606)
* Add .NET Core container debugging support [#4699](https://github.com/GoogleContainerTools/skaffold/pull/4699)
* Identify minikube cluster for any profile name [#4701](https://github.com/GoogleContainerTools/skaffold/pull/4701)
* Add SKAFFOLD_CMDLINE environment variable to pass command-line [#4704](https://github.com/GoogleContainerTools/skaffold/pull/4704)
* Hide minikube detection behind flag [#4745](https://github.com/GoogleContainerTools/skaffold/pull/4745)


Fixes:
* Handle ctrl-c in the middle of GetAllAuthConfigs() [#4603](https://github.com/GoogleContainerTools/skaffold/pull/4603)
* Fix 4748: Panic with skaffold dev [#4750](https://github.com/GoogleContainerTools/skaffold/pull/4750)


Updates:
* Introduce Config interfaces [#4598](https://github.com/GoogleContainerTools/skaffold/pull/4598)
* add pod initialization logic in diag and follow up some minor reporting changes. [#4690](https://github.com/GoogleContainerTools/skaffold/pull/4690)
* Kpt Deployer Render() implementation and tests [#4708](https://github.com/GoogleContainerTools/skaffold/pull/4708)
* Use ParseTolerant to parse Helm version because of missing patch version [#4712](https://github.com/GoogleContainerTools/skaffold/pull/4712)
* Add explicit tests for Helm version parsing [#4715](https://github.com/GoogleContainerTools/skaffold/pull/4715)
* drop codecov patch threshold to 40% [#4716](https://github.com/GoogleContainerTools/skaffold/pull/4716)
* Add Kustomize Hydration to Kpt Deployer's Render method [#4719](https://github.com/GoogleContainerTools/skaffold/pull/4719)
* Move kubernetes client into its own package [#4720](https://github.com/GoogleContainerTools/skaffold/pull/4720)
* Move `DetectWSL` function into util package [#4721](https://github.com/GoogleContainerTools/skaffold/pull/4721)
* Kpt Deployer Deploy() Implementation/Tests [#4723](https://github.com/GoogleContainerTools/skaffold/pull/4723)
* Add test for using helm setFiles [#4735](https://github.com/GoogleContainerTools/skaffold/pull/4735)
* Extending Workflow for Kpt Deployer (accepting additional arguments) [#4736](https://github.com/GoogleContainerTools/skaffold/pull/4736)
* Fix slow test [#4740](https://github.com/GoogleContainerTools/skaffold/pull/4740)
* Fix slow tests [#4741](https://github.com/GoogleContainerTools/skaffold/pull/4741)
* Minikube cluster detection followup [#4742](https://github.com/GoogleContainerTools/skaffold/pull/4742)
* Rename NewFromRunContext() to NewCLI() [#4743](https://github.com/GoogleContainerTools/skaffold/pull/4743)
* Use the newer notation for integration tests [#4744](https://github.com/GoogleContainerTools/skaffold/pull/4744)
* Leverage Config interfaces to simplify tests [#4754](https://github.com/GoogleContainerTools/skaffold/pull/4754)


Dependency Updates:
* Bump golangci lint v1.30.0 [#4739](https://github.com/GoogleContainerTools/skaffold/pull/4739)


Docs Updates:
* Update log-tailing.md [#4636](https://github.com/GoogleContainerTools/skaffold/pull/4636)
* Change kpt deployer doc from Beta to Alpha [#4728](https://github.com/GoogleContainerTools/skaffold/pull/4728)


Huge thanks goes out to all of our contributors for this release:

- Appu Goundan
- Boris Dudelsack
- Brian C
- Brian de Alwis
- David Gageot
- Felix Tran
- Gaurav
- Hasso Mehide
- Julien Ammous
- Marlon Gamez
- MrLuje
- Mridula
- Nick Kubala
- Tejal Desai
- Tyler Schroeder
- Yuwen Ma

# v1.13.2 Release - 08/20/2020
 
**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.13.2/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`
 
**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.13.2/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`
 
**Windows**
 https://storage.googleapis.com/skaffold/releases/v1.13.2/skaffold-windows-amd64.exe
 
**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.13.2`

**This point release contains several usability fixes that should improve user experience, especially when running Skaffold through Cloud Code.**

Highlights:
* Suppress clutter from docker-credential-gcloud error messages [#4705](https://github.com/GoogleContainerTools/skaffold/pull/4705)
* Remove remote rules [#4698](https://github.com/GoogleContainerTools/skaffold/pull/4698)
* Simplify devLoopEvent message text [#4684](https://github.com/GoogleContainerTools/skaffold/pull/4684)
* Return deployment status code when status check can't retrieve pods from cluster [#4683](https://github.com/GoogleContainerTools/skaffold/pull/4683)
* Improved error message when skaffold config not found [#4679](https://github.com/GoogleContainerTools/skaffold/pull/4679)
* Move all update checks to single function; enforce honoring updateCheck flag [#4677](https://github.com/GoogleContainerTools/skaffold/pull/4677)
* Enable watch trigger only when either one of autoBuild, autoSync or autoDeploy is active [#4676](https://github.com/GoogleContainerTools/skaffold/pull/4676)
* Move context validation to build phase so as to not interfere with deploy [#4657](https://github.com/GoogleContainerTools/skaffold/pull/4657)
* Send update-check message to stderr [#4655](https://github.com/GoogleContainerTools/skaffold/pull/4655)
* Make event handling sequential and set the correct timestamp [#4644](https://github.com/GoogleContainerTools/skaffold/pull/4644)

# v1.13.1 Release - 08/04/2020
 
**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.13.1/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`
 
**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.13.1/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`
 
**Windows**
 https://storage.googleapis.com/skaffold/releases/v1.13.1/skaffold-windows-amd64.exe
 
**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.13.1`

**This is a hotfix release for a breaking issue causing our survey link to automatically open itself in a browser. The survey has been disabled completely as
we investigate and fix the root cause of the issue. Sincere apologies to anyone who was adversely affected by this.**

Highlights:
* Fix CustomTagger docs [#4621](https://github.com/GoogleContainerTools/skaffold/pull/4621)
* Disable survey prompt until the next release [#4629](https://github.com/GoogleContainerTools/skaffol
d/pull/4629)
* Clarify 'survey' command text [#4625](https://github.com/GoogleContainerTools/skaffold/pull/4625)


# v1.13.0 Release - 07/30/2020
 
**Linux**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.13.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`
 
**macOS**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.13.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`
 
**Windows**
 https://storage.googleapis.com/skaffold/releases/v1.13.0/skaffold-windows-amd64.exe
 
**Docker image**
`gcr.io/k8s-skaffold/skaffold:v1.13.0`
 
Note: This release comes with a new config version, `v2beta6`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.
 
 
Highlights:
* Skaffold now supports muting both build, test, and deploy logs through the `--mute-logs` flag for more succinct output.
* All extraneous labels added to deployed resources are now added as annotations in Kubernetes.
* Skaffold now supports a new tagging strategy, `customTemplate`, allowing combinations of multiple tagging strategies.
* Specification of the {{.IMAGE_NAME}} component of the `envTemplate` tagger has been deprecated.
* Many other usability fixes and updates in this release!
 
 
New Features:
* Add helper to create log files [#4563](https://github.com/GoogleContainerTools/skaffold/pull/4563)
* Tag template tagger [#4567](https://github.com/GoogleContainerTools/skaffold/pull/4567)
* Add suppress-logs flag [#4530](https://github.com/GoogleContainerTools/skaffold/pull/4530)
* Add `deploy.logs` section to skaffold.yaml [#4509](https://github.com/GoogleContainerTools/skaffold/pull/4509)
* Allow deeply nested property definition for Helm properties [#4511](https://github.com/GoogleContainerTools/skaffold/pull/4511)
* Add Agones custom kinds to the Allow-List [#4488](https://github.com/GoogleContainerTools/skaffold/pull/4488)
* Support deprecated extensions/v1beta1 workload resources [#4478](https://github.com/GoogleContainerTools/skaffold/pull/4478)
 
 
Fixes:
* Use alternative service port-forwarding scheme [#4590](https://github.com/GoogleContainerTools/skaffold/pull/4590)
* Ignore "" namespaces in collectHelmReleasesNamespaces [#4568](https://github.com/GoogleContainerTools/skaffold/pull/4568)
* Wait for pending deletions to complete before a deploy [#4531](https://github.com/GoogleContainerTools/skaffold/pull/4531)
* SKAFFOLD_UPDATE_CHECK should also be a global flag [#4510](https://github.com/GoogleContainerTools/skaffold/pull/4510)
* fix: remove the dev override of the force flag [#4513](https://github.com/GoogleContainerTools/skaffold/pull/4513)
* Error on invalid artifact workspace [#4492](https://github.com/GoogleContainerTools/skaffold/pull/4492)
 
 
Updates:
* Log when values are taken from global config file [#4566](https://github.com/GoogleContainerTools/skaffold/pull/4566)
* Muted test logs [#4595](https://github.com/GoogleContainerTools/skaffold/pull/4595)
* Support short build logs [#4528](https://github.com/GoogleContainerTools/skaffold/pull/4528)
* Fail when k8s client can’t be obtained [#4584](https://github.com/GoogleContainerTools/skaffold/pull/4584)
* Deprecating EnvTemplate's use of {{.IMAGE_NAME}} [#4533](https://github.com/GoogleContainerTools/skaffold/pull/4533)
* Get digest of multi-arch images [#4475](https://github.com/GoogleContainerTools/skaffold/pull/4475)
* Reduce volume of debug-level logging [#4552](https://github.com/GoogleContainerTools/skaffold/pull/4552)
* Remove labels from builders and deployers [#4499](https://github.com/GoogleContainerTools/skaffold/pull/4499)
* Update k3d cli 'load image' to 'image import' (#4498) [#4507](https://github.com/GoogleContainerTools/skaffold/pull/4507)
* Disable update check and survey prompt in non-interactive mode [#4508](https://github.com/GoogleContainerTools/skaffold/pull/4508)
* Map container status PodInitializing to STATUSCHECK_SUCCESS [#4471](https://github.com/GoogleContainerTools/skaffold/pull/4471)
* Use runCtx.Namespaces to get deployments for status checks [#4460](https://github.com/GoogleContainerTools/skaffold/pull/4460)
 
 
Dependency Updates:
* Update pack to v0.12.0 [#4474](https://github.com/GoogleContainerTools/skaffold/pull/4474)
* Include k3d 3.0.0 in Skaffold image [#4545](https://github.com/GoogleContainerTools/skaffold/pull/4545)
* Update cross compilation image [#4543](https://github.com/GoogleContainerTools/skaffold/pull/4543)
* Upgrade go-containerregistry to v0.1.1 [#4476](https://github.com/GoogleContainerTools/skaffold/pull/4476)
 
 
Docs Updates:
* Fix documentation for Helm `artifactOverride` [#4503](https://github.com/GoogleContainerTools/skaffold/pull/4503)
* Fail fast and point to docs for 'skaffold init' on helm projects [#4396](https://github.com/GoogleContainerTools/skaffold/pull/4396)
* Fix example for generate-pipeline to use "latest" as image tag [#4458](https://github.com/GoogleContainerTools/skaffold/pull/4458)
 
 
Huge thanks goes out to all of our contributors for this release:
 
- Alex Lewis
- Alexander Shirobokov
- Andreas Sommer
- Andrew den Hertog
- Appu Goundan
- Balint Pato
- Brian de Alwis
- Chanseok Oh
- Chris Ge
- Daniel Sel
- David Gageot
- Felix Tran
- Gaurav
- Gergo Tolnai
- Keerthan Jaic
- Kent Hua
- Lennox Stevenson
- Marlon Gamez
- Miklos Kiss
- Nicholas Hawkes
- Nick Kubala
- Nils Breunese
- Oliver Hughes
- Paul Vollmer
- Sarmad Abualkaz
- Stefan Büringer
- Tejal Desai
- Zhou Wenzong

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
