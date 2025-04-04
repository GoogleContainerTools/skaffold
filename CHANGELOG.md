# v2.15.0 Release - 04/03/2025
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.15.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.15.0/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.15.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.15.0/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.15.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.15.0`

Note: This release comes with a new config version, `v4beta13`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:

New Features and Additions:
* feat(helm): add depBuild and template flags to HelmDeployFlags schema [#9696](https://github.com/GoogleContainerTools/skaffold/pull/9696)
* feat: allow ValuesFile from GCS [#9182](https://github.com/GoogleContainerTools/skaffold/pull/9182)

Fixes:
* fix: fix lifecycle version in go.mod [#9774](https://github.com/GoogleContainerTools/skaffold/pull/9774)
* fix: port-forward error logs `failed to port forward` (#9728) [#9759](https://github.com/GoogleContainerTools/skaffold/pull/9759)
* fix(verify): use container name from configuration in verify tests [#9753](https://github.com/GoogleContainerTools/skaffold/pull/9753)
* fix: gcb builder incorrectly assumes target project from worker pool project [#9725](https://github.com/GoogleContainerTools/skaffold/pull/9725)
* fix: port-forward error logs `failed to port forward` [#9728](https://github.com/GoogleContainerTools/skaffold/pull/9728)
* fix: kustomize render should workwith inline patches [#9732](https://github.com/GoogleContainerTools/skaffold/pull/9732)
* fix(helm): Fix helm package installation order [#9693](https://github.com/GoogleContainerTools/skaffold/pull/9693)
* fix(helm): Fix helm package installation order (#9693) [#9709](https://github.com/GoogleContainerTools/skaffold/pull/9709)
* fix: (helm) Add expand env template for dependsOn, fix concurrent installation (#9689) [#9707](https://github.com/GoogleContainerTools/skaffold/pull/9707)
* fix(helm): Add expand env template for dependsOn, fix concurrent installation [#9689](https://github.com/GoogleContainerTools/skaffold/pull/9689)

Docs, Test, and Release Updates:
* chore: Fix BuildContextCompressionLevel description, output the level [#9688](https://github.com/GoogleContainerTools/skaffold/pull/9688)
* chore: update dockerfile and integration skaffold dependencies [#9776](https://github.com/GoogleContainerTools/skaffold/pull/9776)
* chore: Update go deps for 2.15 release [#9773](https://github.com/GoogleContainerTools/skaffold/pull/9773)
* chore: bump go version to 1.24.1 [#9772](https://github.com/GoogleContainerTools/skaffold/pull/9772)
* chore: bump github/codeql-action from 3.28.12 to 3.28.13 [#9767](https://github.com/GoogleContainerTools/skaffold/pull/9767)
* chore: bump github.com/golang-jwt/jwt/v4 from 4.5.1 to 4.5.2 [#9763](https://github.com/GoogleContainerTools/skaffold/pull/9763)
* chore: bump actions/upload-artifact from 4.6.1 to 4.6.2 [#9761](https://github.com/GoogleContainerTools/skaffold/pull/9761)
* chore: bump github/codeql-action from 3.28.11 to 3.28.12 [#9760](https://github.com/GoogleContainerTools/skaffold/pull/9760)
* chore: bump golang.org/x/net from 0.33.0 to 0.36.0 in /examples/grpc-e2e-tests/service [#9758](https://github.com/GoogleContainerTools/skaffold/pull/9758)
* chore: bump github.com/containerd/containerd from 1.7.25 to 1.7.27 [#9756](https://github.com/GoogleContainerTools/skaffold/pull/9756)
* chore: bump rack from 2.2.11 to 2.2.13 in /integration/examples/ruby/backend [#9752](https://github.com/GoogleContainerTools/skaffold/pull/9752)
* chore: bump golang.org/x/net from 0.23.0 to 0.36.0 in /hack/tools [#9750](https://github.com/GoogleContainerTools/skaffold/pull/9750)
* chore: bump golang.org/x/net from 0.35.0 to 0.36.0 [#9751](https://github.com/GoogleContainerTools/skaffold/pull/9751)
* chore: bump rack from 2.2.11 to 2.2.13 in /examples/ruby/backend [#9749](https://github.com/GoogleContainerTools/skaffold/pull/9749)
* chore: bump github/codeql-action from 3.28.10 to 3.28.11 [#9748](https://github.com/GoogleContainerTools/skaffold/pull/9748)
* chore: new schema version v4beta13 [#9741](https://github.com/GoogleContainerTools/skaffold/pull/9741)
* chore: bump github.com/go-jose/go-jose/v4 from 4.0.4 to 4.0.5 [#9737](https://github.com/GoogleContainerTools/skaffold/pull/9737)
* chore: bump golang.org/x/net from 0.23.0 to 0.33.0 in /examples/grpc-e2e-tests/service [#9736](https://github.com/GoogleContainerTools/skaffold/pull/9736)
* chore: bump actions/upload-artifact from 4.6.0 to 4.6.1 [#9733](https://github.com/GoogleContainerTools/skaffold/pull/9733)
* chore: bump ossf/scorecard-action from 2.4.0 to 2.4.1 [#9734](https://github.com/GoogleContainerTools/skaffold/pull/9734)
* chore: bump github/codeql-action from 3.28.9 to 3.28.10 [#9735](https://github.com/GoogleContainerTools/skaffold/pull/9735)
* chore: bump github/codeql-action from 3.28.8 to 3.28.9 [#9724](https://github.com/GoogleContainerTools/skaffold/pull/9724)
* chore: bump github.com/grpc-ecosystem/grpc-gateway/v2 from 2.26.0 to 2.26.1 [#9715](https://github.com/GoogleContainerTools/skaffold/pull/9715)
* Revert back to only allowing security updates from dependabot. [#9727](https://github.com/GoogleContainerTools/skaffold/pull/9727)
* chore: bump rack from 2.2.8.1 to 2.2.11 in /integration/examples/ruby/backend [#9719](https://github.com/GoogleContainerTools/skaffold/pull/9719)
* chore: bump rack from 2.2.8.1 to 2.2.11 in /examples/ruby/backend [#9720](https://github.com/GoogleContainerTools/skaffold/pull/9720)
* chore: bump golang.org/x/net from 0.23.0 to 0.33.0 in /integration/examples/grpc-e2e-tests/cloud-spanner-bootstrap [#9726](https://github.com/GoogleContainerTools/skaffold/pull/9726)
* chore: bump google.golang.org/api from 0.219.0 to 0.221.0 [#9723](https://github.com/GoogleContainerTools/skaffold/pull/9723)
* chore: bump github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace from 1.25.0 to 1.26.0 [#9697](https://github.com/GoogleContainerTools/skaffold/pull/9697)
* chore: bump github.com/spf13/pflag from 1.0.5 to 1.0.6 [#9701](https://github.com/GoogleContainerTools/skaffold/pull/9701)
* chore: bump github.com/evanphx/json-patch from 5.9.0+incompatible to 5.9.11+incompatible [#9698](https://github.com/GoogleContainerTools/skaffold/pull/9698)
* chore: bump github/codeql-action from 3.28.5 to 3.28.8 [#9702](https://github.com/GoogleContainerTools/skaffold/pull/9702)
* chore: bump github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric from 0.49.0 to 0.50.0 [#9700](https://github.com/GoogleContainerTools/skaffold/pull/9700)
* chore: bump google.golang.org/api from 0.218.0 to 0.219.0 [#9699](https://github.com/GoogleContainerTools/skaffold/pull/9699)
* chore: bump google.golang.org/grpc from 1.69.4 to 1.70.0 [#9683](https://github.com/GoogleContainerTools/skaffold/pull/9683)
* chore: bump github/codeql-action from 3.28.1 to 3.28.5 [#9685](https://github.com/GoogleContainerTools/skaffold/pull/9685)
* chore: bump google.golang.org/protobuf from 1.36.3 to 1.36.4 [#9684](https://github.com/GoogleContainerTools/skaffold/pull/9684)

Huge thanks goes out to all of our contributors for this release:

- Angel Montero
- Artem Kamenev
- ASHOK KUMAR KS
- Bogdan Nazarenko
- coperni
- dependabot[bot]
- menahyouyeah
- Michael Plump
- SeongChan Lee
- Suleiman Dibirov

# v2.14.0 Release - 01/15/2025
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.14.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.14.0/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.14.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.14.0/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.14.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.14.0`

Note: This release comes with a new config version, `v4beta12`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

New Features and Additions:
* feat: default to ADC when `gcloud` cred helper is configured in docker/config.json when using docker go library [#9469](https://github.com/GoogleContainerTools/skaffold/pull/9469)
* feat: added retry on files sync error [#9261](https://github.com/GoogleContainerTools/skaffold/pull/9261)
* Use bazel info workspace to get workspace, check for MODULE.bazel [#9445](https://github.com/GoogleContainerTools/skaffold/pull/9445)
* feat(git): added commit hash support to git.ref [#9430](https://github.com/GoogleContainerTools/skaffold/pull/9430)
* feat(kaniko): Add kaniko cache run layers flag [#9465](https://github.com/GoogleContainerTools/skaffold/pull/9465)
* feat: new gcs client using cloud client libraries [#9518](https://github.com/GoogleContainerTools/skaffold/pull/9518)
* feat(sync): Add pod filter using FieldSelector [#9493](https://github.com/GoogleContainerTools/skaffold/pull/9493)
* feat(cluster): Add labels to cluster config [#9553](https://github.com/GoogleContainerTools/skaffold/pull/9553)
* feat(bin): Add graceful shutdown for helm command [#9520](https://github.com/GoogleContainerTools/skaffold/pull/9520)
* feat: Optimize helm deploy by using goroutines [#9451](https://github.com/GoogleContainerTools/skaffold/pull/9451)
* feat: transform imagePullPolicy when using local cluster [#9495](https://github.com/GoogleContainerTools/skaffold/pull/9495)
* Support TemplateField for build.artifacts.docker.cliFlags [#9582](https://github.com/GoogleContainerTools/skaffold/pull/9582)
* feat(kaniko): Optimize kaniko build by 50% using compression and add progress [#9476](https://github.com/GoogleContainerTools/skaffold/pull/9476)
* feat(verify.go): Add pod fail reason and message to output [#9589](https://github.com/GoogleContainerTools/skaffold/pull/9589)
* feat(helm): Add helm dependencies support [#9624](https://github.com/GoogleContainerTools/skaffold/pull/9624)
* feat: implement kaniko.imagePullSecret for pulling images from private registry w/ auth  [#9665](https://github.com/GoogleContainerTools/skaffold/pull/9665)

Fixes:
* fix: send maxRetries property when it is specified by the user in a cloud run job manifest [#9475](https://github.com/GoogleContainerTools/skaffold/pull/9475)
* fix: keep the original template if template expansion fails [#9503](https://github.com/GoogleContainerTools/skaffold/pull/9503)
* fix(wait): Add panic prevent WaitForPodInitialized [#9511](https://github.com/GoogleContainerTools/skaffold/pull/9511)
* fix(kaniko): replaces kaniko --snapshotMode argument with --snapshot-mode [#9458](https://github.com/GoogleContainerTools/skaffold/pull/9458)
* fix: emit CloudRunServiceReady event even if default url is disabled [#9523](https://github.com/GoogleContainerTools/skaffold/pull/9523)
* fix: Set the client DialContext to the connhelper dialer DOCKER_HOST is present [#9521](https://github.com/GoogleContainerTools/skaffold/pull/9521)
* fix(config): Replace json tag with yaml for VerifyEnvVar [#9558](https://github.com/GoogleContainerTools/skaffold/pull/9558)
* fix: Continue deployment even if ContainerRemove call returns error [#9561](https://github.com/GoogleContainerTools/skaffold/pull/9561)
* fix: Wrap errors when unmarshal Cloud Run deploy manifests fail. [#9578](https://github.com/GoogleContainerTools/skaffold/pull/9578)
* fix: Handle `StandalonePods` `Succeeded` case when checking status [#9580](https://github.com/GoogleContainerTools/skaffold/pull/9580)
* fix(sync): log a warning for empty pods [#9599](https://github.com/GoogleContainerTools/skaffold/pull/9599)
* fix: kustomize render should support components [#9636](https://github.com/GoogleContainerTools/skaffold/pull/9636)
* fix: Update the k8s Job container logic for custom actions to match v… [#9584](https://github.com/GoogleContainerTools/skaffold/pull/9584)
* fix: Helm deploy was not working with variable templatinging chart path [#9600](https://github.com/GoogleContainerTools/skaffold/pull/9600)
* fix: retry on errors when watching pods [#9373](https://github.com/GoogleContainerTools/skaffold/pull/9373)
* fix: Make defaultNamespace warning more useful [#9669](https://github.com/GoogleContainerTools/skaffold/pull/9669)
* fix: Add Dockerfile for digest calculation [#9666](https://github.com/GoogleContainerTools/skaffold/pull/9666)
* fix: make IMAGE_TAG available in buildArgs when used in docker FROM [9664](https://github.com/GoogleContainerTools/skaffold/pull/9664)

Docs, Test, and Release Updates:
* chore: bump actions/upload-artifact from 4.3.3 to 4.3.4 [#9468](https://github.com/GoogleContainerTools/skaffold/pull/9468)
* fix(docs): fix docs build for v1 and v2 [#9467](https://github.com/GoogleContainerTools/skaffold/pull/9467)
* docs: generate new config version v4beta12 [#9464](https://github.com/GoogleContainerTools/skaffold/pull/9464)
* chore: bump actions/upload-artifact from 4.3.4 to 4.4.0 [#9516](https://github.com/GoogleContainerTools/skaffold/pull/9516)
* chore: bump github/codeql-action from 3.25.2 to 3.26.6 [#9514](https://github.com/GoogleContainerTools/skaffold/pull/9514)
* chore: Update gcloud version from v423 to v496 [#9545](https://github.com/GoogleContainerTools/skaffold/pull/9545)
* chore: bump github.com/docker/docker from 25.0.5+incompatible to 25.0.6+incompatible [#9486](https://github.com/GoogleContainerTools/skaffold/pull/9486)
* chore: fix gcloud v496 SHA [#9547](https://github.com/GoogleContainerTools/skaffold/pull/9547)
* chore: fix SHA256 value of gcloud v496 [#9548](https://github.com/GoogleContainerTools/skaffold/pull/9548)
* chore: bump golang-jwt/jwt/v4 from 4.5.0 to 4.5.1 [#9556](https://github.com/GoogleContainerTools/skaffold/pull/9556)
* chore(logs): add log message for total time taken to complete skaffold dev loop [#9501](https://github.com/GoogleContainerTools/skaffold/pull/9501)
* docs: Fix `IMAGE_NAME` var name for the Nth artifact [#9517](https://github.com/GoogleContainerTools/skaffold/pull/9517)
* chore: bump puma from 5.6.8 to 5.6.9 in /integration/examples/ruby/backend [#9528](https://github.com/GoogleContainerTools/skaffold/pull/9528)
* chore: bump actions/upload-artifact from 4.4.0 to 4.4.3 [#9542](https://github.com/GoogleContainerTools/skaffold/pull/9542)
* chore: bump github/codeql-action from 3.26.6 to 3.27.0 [#9552](https://github.com/GoogleContainerTools/skaffold/pull/9552)
* chore: remove unused taggers field [#9513](https://github.com/GoogleContainerTools/skaffold/pull/9513)
* chore: bump github/codeql-action from 3.27.0 to 3.27.2 [#9564](https://github.com/GoogleContainerTools/skaffold/pull/9564)
* chore: bump actions/setup-go from 4 to 5 [#9213](https://github.com/GoogleContainerTools/skaffold/pull/9213)
* chore: bump ossf/scorecard-action from 2.3.1 to 2.4.0 [#9482](https://github.com/GoogleContainerTools/skaffold/pull/9482)
* chore: bump puma from 5.6.8 to 5.6.9 in /examples/ruby/backend [#9559](https://github.com/GoogleContainerTools/skaffold/pull/9559)
* chore: bump peter-evans/create-or-update-comment from 3.1.0 to 4.0.0 [#9276](https://github.com/GoogleContainerTools/skaffold/pull/9276)
* chore: bump github/codeql-action from 3.27.2 to 3.27.3 [#9566](https://github.com/GoogleContainerTools/skaffold/pull/9566)
* chore: bump flask from 3.0.3 to 3.1.0 in /examples [#9569](https://github.com/GoogleContainerTools/skaffold/pull/9569)
* chore: bump flask from 3.0.3 to 3.1.0 in /integration/examples [#9568](https://github.com/GoogleContainerTools/skaffold/pull/9568)
* chore: bump github/codeql-action from 3.27.3 to 3.27.4 [#9570](https://github.com/GoogleContainerTools/skaffold/pull/9570)
* chore: bump go version from 1.22 to 1.23 [#9571](https://github.com/GoogleContainerTools/skaffold/pull/9571)
* chore: upgrade buildpacks (and transitive dependencies) [#9572](https://github.com/GoogleContainerTools/skaffold/pull/9572)
* chore: bump xt0rted/pull-request-comment-branch from 2.0.0 to 3.0.0 [#9576](https://github.com/GoogleContainerTools/skaffold/pull/9576)
* chore: bump github/codeql-action from 3.27.4 to 3.27.5 [#9579](https://github.com/GoogleContainerTools/skaffold/pull/9579)
* chore: try to fix the security scorecard action [#9585](https://github.com/GoogleContainerTools/skaffold/pull/9585)
* chore: bump github/codeql-action from 3.27.5 to 3.27.6 [#9594](https://github.com/GoogleContainerTools/skaffold/pull/9594)
* chore: update the CODEOWNERS file [#9597](https://github.com/GoogleContainerTools/skaffold/pull/9597)
* fix(homepage): fix gem icon [#9596](https://github.com/GoogleContainerTools/skaffold/pull/9596)
* chore: remove MAINTAINERS [#9601](https://github.com/GoogleContainerTools/skaffold/pull/9601)
* test: Fix the Bazel integration test. [#9604](https://github.com/GoogleContainerTools/skaffold/pull/9604)
* chore: upgrade more dependencies [#9602](https://github.com/GoogleContainerTools/skaffold/pull/9602)
* ci: correctly tag the latest release with the "latest" tag. [#9606](https://github.com/GoogleContainerTools/skaffold/pull/9606)
* chore: bump golang.org/x/crypto from 0.21.0 to 0.31.0 in /hack/tools [#9610](https://github.com/GoogleContainerTools/skaffold/pull/9610)
* chore: bump github/codeql-action from 3.27.6 to 3.27.7 [#9608](https://github.com/GoogleContainerTools/skaffold/pull/9608)
* chore: bump golang.org/x/crypto from 0.30.0 to 0.31.0 [#9611](https://github.com/GoogleContainerTools/skaffold/pull/9611)
* chore: bump github/codeql-action from 3.27.7 to 3.27.9 [#9612](https://github.com/GoogleContainerTools/skaffold/pull/9612)
* ci: cleaning up references to skaffold slim as it is no longer used [#9615](https://github.com/GoogleContainerTools/skaffold/pull/9615)
* ci: remove deprecated workflow for creating release. This would prevent accidental trigger of this workflow [#9614](https://github.com/GoogleContainerTools/skaffold/pull/9614)
* chore: cleaning up final references to slim from skaffold [#9616](https://github.com/GoogleContainerTools/skaffold/pull/9616)
* chore: upgrade versions of integration test tooling [#9574](https://github.com/GoogleContainerTools/skaffold/pull/9574)
* chore: upgrade more dependencies [#9617](https://github.com/GoogleContainerTools/skaffold/pull/9617)
* docs: fixing yaml syntax [#9427](https://github.com/GoogleContainerTools/skaffold/pull/9427)
* docs: Document some undocumented config options [#9237](https://github.com/GoogleContainerTools/skaffold/pull/9237)
* chore: bump actions/upload-artifact from 4.4.3 to 4.5.0 [#9618](https://github.com/GoogleContainerTools/skaffold/pull/9618)
* chore: allow dependabot to upgrade more dependencies [#9619](https://github.com/GoogleContainerTools/skaffold/pull/9619)
* chore: bump github/codeql-action from 3.27.9 to 3.28.0 [#9625](https://github.com/GoogleContainerTools/skaffold/pull/9625)
* chore: a (hopefully) final set of upgrades before dependabot takes over [#9622](https://github.com/GoogleContainerTools/skaffold/pull/9622)
* chore: update go version used in the published container [#9642](https://github.com/GoogleContainerTools/skaffold/pull/9642)
* chore: upgrade all bundled tools in the Skaffold container [#9646](https://github.com/GoogleContainerTools/skaffold/pull/9646)
* chore: bump github.com/buildpacks/pack from 0.36.2 to 0.36.3 [#9655](https://github.com/GoogleContainerTools/skaffold/pull/9655)
* chore: bump google.golang.org/api from 0.215.0 to 0.216.0 [#9653](https://github.com/GoogleContainerTools/skaffold/pull/9653)
* chore: bump cloud.google.com/go/storage from 1.49.0 to 1.50.0 [#9652](https://github.com/GoogleContainerTools/skaffold/pull/9652)
* chore: bump github.com/spf13/afero from 1.11.0 to 1.12.0 [#9651](https://github.com/GoogleContainerTools/skaffold/pull/9651)
* chore: bump github/codeql-action from 3.28.0 to 3.28.1 [#9650](https://github.com/GoogleContainerTools/skaffold/pull/9650)
* chore: bump github.com/containerd/containerd from 1.7.24 to 1.7.25 [#9654](https://github.com/GoogleContainerTools/skaffold/pull/9654)
* chore: bump actions/upload-artifact from 4.5.0 to 4.6.0 [#9649](https://github.com/GoogleContainerTools/skaffold/pull/9649)
* docs: Propose build batching for Bazel. [#9425](https://github.com/GoogleContainerTools/skaffold/pull/9425)
* chore: remove GRPC package excludes [#9659](https://github.com/GoogleContainerTools/skaffold/pull/9659)
* chore(cloudbuild): add configurable source bucket [#9441](https://github.com/GoogleContainerTools/skaffold/pull/9441)
* chore: move deprecated library location to new location [#9661](https://github.com/GoogleContainerTools/skaffold/pull/9661)
* fix: upgrade gradle to 8.11.1 and set java version used to 21. [#9623](https://github.com/GoogleContainerTools/skaffold/pull/9623)
* chore: final cleanups of the go.mod file [#9663](https://github.com/GoogleContainerTools/skaffold/pull/9663)
* chore: bump k8s.io/apimachinery from 0.32.0 to 0.32.1 [#9673](https://github.com/GoogleContainerTools/skaffold/pull/9673)
* chore: bump cloud.google.com/go/cloudbuild from 1.19.2 to 1.20.0 [#9674](https://github.com/GoogleContainerTools/skaffold/pull/9674)
* chore: bump go.opentelemetry.io/otel/sdk/metric from 1.33.0 to 1.34.0 [#9676](https://github.com/GoogleContainerTools/skaffold/pull/9676)
* chore: bump github.com/buildpacks/pack from 0.36.3 to 0.36.4 [#9675](https://github.com/GoogleContainerTools/skaffold/pull/9675)
* chore: bump golang.org/x/net from 0.23.0 to 0.33.0 in /examples/grpc-e2e-tests/cloud-spanner-bootstrap [#9678](https://github.com/GoogleContainerTools/skaffold/pull/9678)
* chore: bump go.opentelemetry.io/otel/exporters/stdout/stdouttrace from 1.33.0 to 1.34.0 [#9677](https://github.com/GoogleContainerTools/skaffold/pull/9677)
* chore: one last PR of dependency upgrades before the release is cut [#9680](https://github.com/GoogleContainerTools/skaffold/pull/9680)

Huge thanks goes out to all of our contributors for this release:

- Abe Winter
- Andreas Bergmeier
- Angel Montero
- Aran Donohue
- Benjamin Kaplan
- Chris
- cui fliter
- David Herges
- Darien Lin
- dependabot[bot]
- ericzzzzzzz
- Jesse Ward
- joeyslalom
- Kallan Gerard
- Lucas Rodriguez
- Mathias Nicolajsen Kjærgaard
- Matt Santa
- menahyouyeah
- Michael Plump
- Mike Gelfand
- Renzo Rojas
- Ryo Kitagawa
- sce-taid
- Seth Nelson
- Shikanime Deva
- Suleiman Dibirov
- Travis Hein
- Vladimir Nachev
- Wassim Dhif
- Y.


# v2.13.0 Release - 07/08/2024
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.13.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.13.0/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.13.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.13.0/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.13.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.13.0`

Highlights:

New Features and Additions:
* feat: make ADC the default option for GCP authentication when using go-containerregistry [#9456](https://github.com/GoogleContainerTools/skaffold/pull/9456)
* feat: Optimized fs walker and util.IsEmptyDir [#9433](https://github.com/GoogleContainerTools/skaffold/pull/9433)

Fixes:
* fix: first and last image won't be detected as known image, do not add single quote to the jsonpath (#9448) [#9449](https://github.com/GoogleContainerTools/skaffold/pull/9449)
* fix(cmd): fixed err output for delete and deploy commands [#9437](https://github.com/GoogleContainerTools/skaffold/pull/9437)

Updates and Refactors:
* chore: upgrade-go-to-1.22.4 [#9454](https://github.com/GoogleContainerTools/skaffold/pull/9454)
* chore(logs): update log messages for better clarity [#9443](https://github.com/GoogleContainerTools/skaffold/pull/9443)

Huge thanks goes out to all of our contributors for this release:

- Renzo Rojas
- Roland Németh
- Suleiman Dibirov
- ericzzzzzzz

# v2.12.0 Release - 05/14/2024
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.12.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.12.0/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.12.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.12.0/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.12.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.12.0`

Note: This release comes with a new config version, `v4beta11`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:

New Features and Additions:
* feat: add `--destination` flag for kaniko build [#9415](https://github.com/GoogleContainerTools/skaffold/pull/9415)
* feat(exec|verify): enabled "namespace" option for exec and verify commands [#9307](https://github.com/GoogleContainerTools/skaffold/pull/9307)
* feat: support templating in diagnose command [#9393](https://github.com/GoogleContainerTools/skaffold/pull/9393)
* feat(docker-network): docker.network now supports any value [#9390](https://github.com/GoogleContainerTools/skaffold/pull/9390)

Fixes:
* fix: TestGenerateMavenBuildArgs-host-platform [#9410](https://github.com/GoogleContainerTools/skaffold/pull/9410)
* fix(kaniko): delete kaniko pod on graceful shutdown [#9270](https://github.com/GoogleContainerTools/skaffold/pull/9270)
* fix(tar): data race fix [#9309](https://github.com/GoogleContainerTools/skaffold/pull/9309)
* fix: add --load flag for local buildkit [#9387](https://github.com/GoogleContainerTools/skaffold/pull/9387)

Updates and Refactors:
* chore: bump github/codeql-action from 3.25.1 to 3.25.2 [#9402](https://github.com/GoogleContainerTools/skaffold/pull/9402)
* chore: bump actions/upload-artifact from 4.3.2 to 4.3.3 [#9403](https://github.com/GoogleContainerTools/skaffold/pull/9403)
* chore: bump github.com/sigstore/cosign/v2 from 2.2.1 to 2.2.4 [#9385](https://github.com/GoogleContainerTools/skaffold/pull/9385)
* chore: bump flask from 3.0.2 to 3.0.3 in /integration/examples [#9381](https://github.com/GoogleContainerTools/skaffold/pull/9381)
* chore: bump flask from 3.0.2 to 3.0.3 in /examples [#9379](https://github.com/GoogleContainerTools/skaffold/pull/9379)
* chore: bump golang.org/x/net from 0.17.0 to 0.23.0 in /integration/examples/grpc-e2e-tests/cloud-spanner-bootstrap [#9396](https://github.com/GoogleContainerTools/skaffold/pull/9396)
* chore: bump golang.org/x/net from 0.17.0 to 0.23.0 in /examples/grpc-e2e-tests/service [#9397](https://github.com/GoogleContainerTools/skaffold/pull/9397)
* chore: bump golang.org/x/net from 0.22.0 to 0.23.0 in /hack/tools [#9399](https://github.com/GoogleContainerTools/skaffold/pull/9399)
* chore: bump golang.org/x/net from 0.22.0 to 0.23.0 [#9400](https://github.com/GoogleContainerTools/skaffold/pull/9400)
* chore: bump golang.org/x/net from 0.17.0 to 0.23.0 in /integration/examples/grpc-e2e-tests/service [#9398](https://github.com/GoogleContainerTools/skaffold/pull/9398)
* chore: bump golang.org/x/net from 0.17.0 to 0.23.0 in /examples/grpc-e2e-tests/cloud-spanner-bootstrap [#9395](https://github.com/GoogleContainerTools/skaffold/pull/9395)
* chore: bump actions/upload-artifact from 4.3.1 to 4.3.2 [#9394](https://github.com/GoogleContainerTools/skaffold/pull/9394)
* schema: v4beta11 [#9401](https://github.com/GoogleContainerTools/skaffold/pull/9401)
* chore: bump github/codeql-action from 3.24.9 to 3.25.1 [#9391](https://github.com/GoogleContainerTools/skaffold/pull/9391)

Docs, Test, and Release Updates:
* docs: add bazel cross-platform documentation [#9363](https://github.com/GoogleContainerTools/skaffold/pull/9363)

Huge thanks goes out to all of our contributors for this release:

- Aran Donohue
- Hedi Nasr
- Michael Kuc
- Suleiman Dibirov
- dependabot[bot]
- ericzzzzzzz

# v2.11.0 Release - 04/02/2024
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.11.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.11.0/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.11.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.11.0/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.11.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.11.0`

Note: This release comes with a new config version, `v4beta10`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

Highlights:

New Features and Additions:
* feat: Support Bazel platform mappings [#9300](https://github.com/GoogleContainerTools/skaffold/pull/9300)
* feat: new repo resolver logic to fetch info from a gcbrepov2 [#9283](https://github.com/GoogleContainerTools/skaffold/pull/9283)
* feat: extracted kaniko copyTimeout and copyMaxRetries into config [#9267](https://github.com/GoogleContainerTools/skaffold/pull/9267)
* feat(tar): added logs to CreateTar func [#9271](https://github.com/GoogleContainerTools/skaffold/pull/9271)

Fixes:
* fix: revert cache lookup changes [#9313](https://github.com/GoogleContainerTools/skaffold/pull/9313)
* fix(lookupRemote): fixed lookup.go lookupRemote to compare remote and cached digests [#9278](https://github.com/GoogleContainerTools/skaffold/pull/9278)
* fix(helm): use secrets helm plugin to render when useHelmSecrets is true [#9295](https://github.com/GoogleContainerTools/skaffold/pull/9295)


Updates and Refactors:
* chore: upgrade cosign  from 2.0.3-0.20230523133326-0544abd8fc8a to 2.2.1 [#9369](https://github.com/GoogleContainerTools/skaffold/pull/9369)
* chore: bump gopkg.in/go-jose/go-jose.v2 from 2.6.1 to 2.6.3 [#9333](https://github.com/GoogleContainerTools/skaffold/pull/9333)
* chore: bump github.com/cloudflare/circl from 1.3.3 to 1.3.7 [#9242](https://github.com/GoogleContainerTools/skaffold/pull/9242)
* chore: bump flask from 3.0.1 to 3.0.2 in /integration/examples [#9297](https://github.com/GoogleContainerTools/skaffold/pull/9297)
* chore: bump rack from 2.2.6.4 to 2.2.8.1 in /examples/ruby/backend [#9328](https://github.com/GoogleContainerTools/skaffold/pull/9328)
* chore: bump rack from 2.2.6.4 to 2.2.8.1 in /integration/examples/ruby/backend [#9329](https://github.com/GoogleContainerTools/skaffold/pull/9329)
* chore: bump github/codeql-action from 3.24.8 to 3.24.9 [#9354](https://github.com/GoogleContainerTools/skaffold/pull/9354)
* chore: bump google.golang.org/protobuf from 1.30.0 to 1.33.0 in /integration/examples/grpc-e2e-tests/cloud-spanner-bootstrap [#9339](https://github.com/GoogleContainerTools/skaffold/pull/9339)
* chore: bump google.golang.org/protobuf from 1.30.0 to 1.33.0 in /examples/grpc-e2e-tests/cloud-spanner-bootstrap [#9341](https://github.com/GoogleContainerTools/skaffold/pull/9341)
* chore: bump github.com/docker/docker from 25.0.3+incompatible to 25.0.5+incompatible [#9366](https://github.com/GoogleContainerTools/skaffold/pull/9366)
* chore: bump moby/buildkit and opencontainers/runc versions, upgrade go to 1.22 [#9364](https://github.com/GoogleContainerTools/skaffold/pull/9364)
* chore: updating google api and opentelemetry version [#9352](https://github.com/GoogleContainerTools/skaffold/pull/9352)
* feat: extend `skaffold inspect config-dependencies add` to support GCB Repo v2 [#9349](https://github.com/GoogleContainerTools/skaffold/pull/9349)
* chore: bump github/codeql-action from 3.24.0 to 3.24.8 [#9348](https://github.com/GoogleContainerTools/skaffold/pull/9348)
* chore: bump google.golang.org/protobuf from 1.30.0 to 1.33.0 in /integration/examples/grpc-e2e-tests/service [#9342](https://github.com/GoogleContainerTools/skaffold/pull/9342)
* chore: new googleCloudBuildRepoV2 field to configure a remote dependency [#9293](https://github.com/GoogleContainerTools/skaffold/pull/9293)
* chore: upgrade go to v1.21.6 due to vuls [#9303](https://github.com/GoogleContainerTools/skaffold/pull/9303)
* chore: bump github.com/opencontainers/runc from 1.1.7 to 1.1.12 [#9290](https://github.com/GoogleContainerTools/skaffold/pull/9290)
* chore: bump flask from 3.0.1 to 3.0.2 in /examples [#9298](https://github.com/GoogleContainerTools/skaffold/pull/9298)
* chore: bump actions/upload-artifact from 4.3.0 to 4.3.1 [#9299](https://github.com/GoogleContainerTools/skaffold/pull/9299)
* chore: bump github/codeql-action from 3.23.1 to 3.24.0 [#9296](https://github.com/GoogleContainerTools/skaffold/pull/9296)
* chore: generate schema v4beta9 [#9287](https://github.com/GoogleContainerTools/skaffold/pull/9287)


Docs, Test, and Release Updates:


Huge thanks goes out to all of our contributors for this release:

- Angel Montero
- Aran Donohue
- Benjamin Kaplan
- Renzo Rojas
- dependabot[bot]
- ericzzzzzzz
- idsulik

# v2.10.0 Release - 01/09/2024
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.10.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.10.0/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.10.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.10.0/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.10.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.10.0`

Note: This release comes with a new config version, `v4beta9`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

New Features and Additions:
* feat: Skaffold post renderer [#9203](https://github.com/GoogleContainerTools/skaffold/pull/9203)

Fixes:
* fix: helm-deploy-chart-path-template [#9243](https://github.com/GoogleContainerTools/skaffold/pull/9243)
* fix: apply-setter and transformer should ignore non-k8s-resource for kustomize paramterization [#9240](https://github.com/GoogleContainerTools/skaffold/pull/9240)
* fix: Scope Issue with the 'entry' variable when looking up remote images and tests additions [#9211](https://github.com/GoogleContainerTools/skaffold/pull/9211)
* fix: remove global helm flags from flags sent to `skaffold filter` [#9212](https://github.com/GoogleContainerTools/skaffold/pull/9212)
* fix: puling images when working with a remote repository (#9177) [#9181](https://github.com/GoogleContainerTools/skaffold/pull/9181)
* fix: custom crd not printing streams logs [#9136](https://github.com/GoogleContainerTools/skaffold/pull/9136)
* fix: Enable docker build without cli [#9178](https://github.com/GoogleContainerTools/skaffold/pull/9178)
* Fix panic in Logger.Stop [#9159](https://github.com/GoogleContainerTools/skaffold/pull/9159)
* fix: sync slow 2.9 [#9168](https://github.com/GoogleContainerTools/skaffold/pull/9168)
* fix: sync slow [#9167](https://github.com/GoogleContainerTools/skaffold/pull/9167)

Updates and Refactors:
* chore: bump puma from 5.6.7 to 5.6.8 in /integration/examples/ruby/backend [#9244](https://github.com/GoogleContainerTools/skaffold/pull/9244)
* chore: bump github/codeql-action from 3.22.12 to 3.23.0 [#9241](https://github.com/GoogleContainerTools/skaffold/pull/9241)
* chore: bump golang.org/x/crypto from 0.12.0 to 0.17.0 [#9227](https://github.com/GoogleContainerTools/skaffold/pull/9227)
* chore: bump github/codeql-action from 2.22.9 to 3.22.12 [#9231](https://github.com/GoogleContainerTools/skaffold/pull/9231)
* chore: bump github.com/go-git/go-git/v5 from 5.8.1 to 5.11.0 [#9234](https://github.com/GoogleContainerTools/skaffold/pull/9234)
* chore: bump golang.org/x/crypto from 0.14.0 to 0.17.0 in /hack/tools [#9228](https://github.com/GoogleContainerTools/skaffold/pull/9228)
* chore: bump github/codeql-action from 2.22.8 to 2.22.9 [#9214](https://github.com/GoogleContainerTools/skaffold/pull/9214)
* chore: bump github/codeql-action from 2.22.7 to 2.22.8 [#9193](https://github.com/GoogleContainerTools/skaffold/pull/9193)
* chore: bump actions/upload-artifact from 3.1.3 to 4.0.0 [#9226](https://github.com/GoogleContainerTools/skaffold/pull/9226)
* chore: bump github/codeql-action from 2.22.6 to 2.22.7 [#9180](https://github.com/GoogleContainerTools/skaffold/pull/9180)
* chore: bump github/codeql-action from 2.22.5 to 2.22.6 [#9173](https://github.com/GoogleContainerTools/skaffold/pull/9173)
* chore: clean up example project deps [#9216](https://github.com/GoogleContainerTools/skaffold/pull/9216)
* chore: inject imageInfo when expanding templates for ko builder [#9207](https://github.com/GoogleContainerTools/skaffold/pull/9207)
* chore: change bazel example [#9218](https://github.com/GoogleContainerTools/skaffold/pull/9218)
* fix: add riscv64 to the install-golint.sh script [#9210](https://github.com/GoogleContainerTools/skaffold/pull/9210)
* chore: generate schema v4beta9 [#9204](https://github.com/GoogleContainerTools/skaffold/pull/9204)

Docs, Test, and Release Updates:
* docs: Add missing template field [#9186](https://github.com/GoogleContainerTools/skaffold/pull/9186)

Huge thanks goes out to all of our contributors for this release:

- Andreas Bergmeier
- Renzo Rojas
- beast
- dependabot[bot]
- ericzzzzzzz
- mboulton-fathom
- xord37
- xun

# v2.9.0 Release - 11/07/2023
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.9.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.9.0/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.9.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.9.0/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.9.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.9.0`

Note: This release comes with a new config version, `v4beta8`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

New Features and Additions:
* chore: add new skip-unreachable-dirs to not error on init command when a dir can not be read [#9163](https://github.com/GoogleContainerTools/skaffold/pull/9163)
* chore: add new config to control the pull behaviour for verify [#9150](https://github.com/GoogleContainerTools/skaffold/pull/9150)
* chore: change custom actions pull logic, to check if an image exists locally first before triggering a pull [#9147](https://github.com/GoogleContainerTools/skaffold/pull/9147)

Fixes:
* fix: kpt force named "false" in schema [#9074](https://github.com/GoogleContainerTools/skaffold/pull/9074)

Updates and Refactors:
* chore: bump golang.org/x/net from 0.7.0 to 0.17.0 in /hack/tools [#9129](https://github.com/GoogleContainerTools/skaffold/pull/9129)
* chore: bump golang.org/x/net from 0.7.0 to 0.17.0 in /examples/grpc-e2e-tests/service [#9130](https://github.com/GoogleContainerTools/skaffold/pull/9130)
* chore: bump golang.org/x/net from 0.7.0 to 0.17.0 in /integration/examples/grpc-e2e-tests/cloud-spanner-bootstrap [#9131](https://github.com/GoogleContainerTools/skaffold/pull/9131)
* chore: bump golang.org/x/net from 0.7.0 to 0.17.0 in /integration/examples/grpc-e2e-tests/service [#9128](https://github.com/GoogleContainerTools/skaffold/pull/9128)
* chore: bump google.golang.org/grpc from 1.55.0 to 1.56.3 [#9156](https://github.com/GoogleContainerTools/skaffold/pull/9156)
* chore: bump google.golang.org/grpc from 1.53.0 to 1.56.3 in /integration/examples/grpc-e2e-tests/service [#9154](https://github.com/GoogleContainerTools/skaffold/pull/9154)
* chore: bump google.golang.org/grpc from 1.53.0 to 1.56.3 in /examples/grpc-e2e-tests/service [#9153](https://github.com/GoogleContainerTools/skaffold/pull/9153)
* chore: bump google.golang.org/grpc from 1.53.0 to 1.56.3 in /examples/grpc-e2e-tests/cloud-spanner-bootstrap [#9152](https://github.com/GoogleContainerTools/skaffold/pull/9152)
* chore: bump google.golang.org/grpc from 1.53.0 to 1.56.3 in /integration/examples/grpc-e2e-tests/cloud-spanner-bootstrap [#9155](https://github.com/GoogleContainerTools/skaffold/pull/9155)
* chore: bump github/codeql-action from 2.22.4 to 2.22.5 [#9157](https://github.com/GoogleContainerTools/skaffold/pull/9157)
* chore: bump ossf/scorecard-action from 2.3.0 to 2.3.1 [#9149](https://github.com/GoogleContainerTools/skaffold/pull/9149)
* chore: bump schema version to v4beta8 [#9143](https://github.com/GoogleContainerTools/skaffold/pull/9143)
* chore: bump peter-evans/create-or-update-comment from 3.0.2 to 3.1.0 [#9142](https://github.com/GoogleContainerTools/skaffold/pull/9142)
* chore: bump github/codeql-action from 2.22.3 to 2.22.4 [#9146](https://github.com/GoogleContainerTools/skaffold/pull/9146)
* chore: bump github/codeql-action from 2.22.2 to 2.22.3 [#9137](https://github.com/GoogleContainerTools/skaffold/pull/9137)
* chore: bump golang.org/x/net from 0.7.0 to 0.17.0 in /examples/grpc-e2e-tests/cloud-spanner-bootstrap [#9132](https://github.com/GoogleContainerTools/skaffold/pull/9132)
* chore: bump github/codeql-action from 2.22.1 to 2.22.2 [#9133](https://github.com/GoogleContainerTools/skaffold/pull/9133)
* chore: bump ossf/scorecard-action from 2.2.0 to 2.3.0 [#9122](https://github.com/GoogleContainerTools/skaffold/pull/9122)
* chore: bump github/codeql-action from 2.22.0 to 2.22.1 [#9123](https://github.com/GoogleContainerTools/skaffold/pull/9123)
* chore: bump github/codeql-action from 2.21.9 to 2.22.0 [#9115](https://github.com/GoogleContainerTools/skaffold/pull/9115)
* chore: bump flask from 2.3.3 to 3.0.0 in /integration/examples [#9107](https://github.com/GoogleContainerTools/skaffold/pull/9107)
* chore: bump flask from 2.3.3 to 3.0.0 in /examples [#9106](https://github.com/GoogleContainerTools/skaffold/pull/9106)

Docs, Test, and Release Updates:
* docs: fix releaseNoteLink for v2.8.0 [#9125](https://github.com/GoogleContainerTools/skaffold/pull/9125)

Huge thanks goes out to all of our contributors for this release:

- Julian Tölle
- Renzo Rojas
- Zev Isert
- dependabot[bot]

# v2.8.0 Release - 10/03/2023
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.8.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.8.0/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.8.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.8.0/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.8.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.8.0`

Note: This release comes with a new config version, `v4beta7`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.

New Features and Additions:
* feat: Support post-renderer for helm deployer. [#9100](https://github.com/GoogleContainerTools/skaffold/pull/9100)
* feat: inject namespace from rendered manifests in post deploy hooks [#9090](https://github.com/GoogleContainerTools/skaffold/pull/9090)
* feat: Add skaffold inspect command for adding config dependencies [#9072](https://github.com/GoogleContainerTools/skaffold/pull/9072)
* feat: emit metrics for exec, verify and render [#9078](https://github.com/GoogleContainerTools/skaffold/pull/9078)
* feat: Add global build pre- and post-hooks  [#9047](https://github.com/GoogleContainerTools/skaffold/pull/9047)
* feat: allow specifying a remote config dependency from Google Cloud Storage [#9057](https://github.com/GoogleContainerTools/skaffold/pull/9057)

Updates and Refactors:
* chore: bump github/codeql-action from 2.21.8 to 2.21.9 [#9101](https://github.com/GoogleContainerTools/skaffold/pull/9101)
* chore: bump github/codeql-action from 2.21.7 to 2.21.8 [#9097](https://github.com/GoogleContainerTools/skaffold/pull/9097)
* chore: bump github/codeql-action from 2.21.6 to 2.21.7 [#9096](https://github.com/GoogleContainerTools/skaffold/pull/9096)
* chore: add set docker host by current context [#9094](https://github.com/GoogleContainerTools/skaffold/pull/9094)
* chore: bump github/codeql-action from 2.21.5 to 2.21.6 [#9093](https://github.com/GoogleContainerTools/skaffold/pull/9093)
* chore: cherry-pick upgrade ko (#9043) to v2.7 [#9089](https://github.com/GoogleContainerTools/skaffold/pull/9089)
* chore: verify should preserve job manifest envs [#9087](https://github.com/GoogleContainerTools/skaffold/pull/9087)
* chore: bump actions/upload-artifact from 3.1.2 to 3.1.3 [#9075](https://github.com/GoogleContainerTools/skaffold/pull/9075)
* chore: upgrade ko [#9043](https://github.com/GoogleContainerTools/skaffold/pull/9043)
* chore: bump actions/checkout from 3 to 4 [#9067](https://github.com/GoogleContainerTools/skaffold/pull/9067)

Docs, Test, and Release Updates:
* docs: Fix document tutorials/skaffold-resource-selector.md [#9083](https://github.com/GoogleContainerTools/skaffold/pull/9083)
* docs: add templatable field [#9088](https://github.com/GoogleContainerTools/skaffold/pull/9088)

Huge thanks goes out to all of our contributors for this release:

- Danilo Cianfrone
- Matt Santa
- Michael Plump
- Renzo Rojas
- Seita Uchimura
- dependabot[bot]
- ericzzzzzzz
- guangwu
- yosukei3108

# v2.7.0 Release - 08/30/2023
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.7.0/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.7.0/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.7.0/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.7.0/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v2.7.0/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v2.7.0`

Highlights:
* feat: crd status check [#9016](https://github.com/GoogleContainerTools/skaffold/pull/9016)

New Features and Additions:
* feat: enable skaffold render to track telemetry [#9020](https://github.com/GoogleContainerTools/skaffold/pull/9020)
* feat: support url as kustomize file path [#9023](https://github.com/GoogleContainerTools/skaffold/pull/9023)
* feat: configure verify and exec commands to emit metrics [#9013](https://github.com/GoogleContainerTools/skaffold/pull/9013)
* feat: support overrides in helm renderer [#8966](https://github.com/GoogleContainerTools/skaffold/pull/8966)
* feat: Add Sprig template functions [#9005](https://github.com/GoogleContainerTools/skaffold/pull/9005)
* feat: remove condition that checks if an image was built from Skaffold [#8935](https://github.com/GoogleContainerTools/skaffold/pull/8935)

Fixes:
* fix: status check lists all events [#9015](https://github.com/GoogleContainerTools/skaffold/pull/9015)
* fix: Use moby/patternmatcher for dockerignore [#9029](https://github.com/GoogleContainerTools/skaffold/pull/9029)
* fix: Ignore unset variables from minikube docker-env [#9018](https://github.com/GoogleContainerTools/skaffold/pull/9018)
* fix: #9006 - Filter port forwarding resources for docker deploy [#9008](https://github.com/GoogleContainerTools/skaffold/pull/9008)
* fix: documentation for Helm Template Value [#8991](https://github.com/GoogleContainerTools/skaffold/pull/8991)
* fix: status check connecting to the wrong k8s context [#8981](https://github.com/GoogleContainerTools/skaffold/pull/8981)
* fix: remote kustomize manifest being watched [#8979](https://github.com/GoogleContainerTools/skaffold/pull/8979)
* fix: Add integration tests for helm namespace [#8965](https://github.com/GoogleContainerTools/skaffold/pull/8965)
* fix: edit original file if the given skaffold path is a symlink [#8955](https://github.com/GoogleContainerTools/skaffold/pull/8955)
* fix: verify hangs if event-logs-file does not exist (#7613) [#8961](https://github.com/GoogleContainerTools/skaffold/pull/8961)
* fix: Fix typo in Cloud Run log tailing [#8944](https://github.com/GoogleContainerTools/skaffold/pull/8944)

Updates and Refactors:
* chore: remove latest tagging from release process [#8986](https://github.com/GoogleContainerTools/skaffold/pull/8986)
* chore: update the way the LTS images are built [#8953](https://github.com/GoogleContainerTools/skaffold/pull/8953)
* chore:  configure minikube to a static version in github ci  [#8951](https://github.com/GoogleContainerTools/skaffold/pull/8951)
* chore: disable edge image scanning [#8942](https://github.com/GoogleContainerTools/skaffold/pull/8942)
* chore: save public image tag [#8930](https://github.com/GoogleContainerTools/skaffold/pull/8930)
* chore: upgrade go 1.19.10 -> 1.20.7 [#8992](https://github.com/GoogleContainerTools/skaffold/pull/8992)
* chore: upgrade go to 1.21.0 [#8999](https://github.com/GoogleContainerTools/skaffold/pull/8999)
* chore: bump flask from 2.3.2 to 2.3.3 in /integration/examples [#9041](https://github.com/GoogleContainerTools/skaffold/pull/9041)
* chore: bump flask from 2.3.2 to 2.3.3 in /examples [#9042](https://github.com/GoogleContainerTools/skaffold/pull/9042)
* chore: bump github/codeql-action from 2.20.1 to 2.20.2 [#8928](https://github.com/GoogleContainerTools/skaffold/pull/8928)
* chore: bump github/codeql-action from 2.20.2 to 2.20.3 [#8937](https://github.com/GoogleContainerTools/skaffold/pull/8937)
* chore: bump github/codeql-action from 2.20.3 to 2.20.4 [#8950](https://github.com/GoogleContainerTools/skaffold/pull/8950)
* chore: bump github/codeql-action from 2.20.4 to 2.21.0 [#8964](https://github.com/GoogleContainerTools/skaffold/pull/8964)
* chore: bump github/codeql-action from 2.21.0 to 2.21.1 [#8975](https://github.com/GoogleContainerTools/skaffold/pull/8975)
* chore: bump github/codeql-action from 2.21.1 to 2.21.2 [#8980](https://github.com/GoogleContainerTools/skaffold/pull/8980)
* chore: bump github/codeql-action from 2.21.2 to 2.21.3 [#9000](https://github.com/GoogleContainerTools/skaffold/pull/9000)
* chore: bump github/codeql-action from 2.21.3 to 2.21.4 [#9022](https://github.com/GoogleContainerTools/skaffold/pull/9022)
* chore: bump github/codeql-action from 2.21.4 to 2.21.5 [#9053](https://github.com/GoogleContainerTools/skaffold/pull/9053)
* chore: bump github.com/sigstore/rekor from 1.1.1 to 1.2.0 [#8829](https://github.com/GoogleContainerTools/skaffold/pull/8829)
* chore: bump google.golang.org/grpc from 1.48.0 to 1.53.0 in /examples/grpc-e2e-tests/cloud-spanner-bootstrap [#8932](https://github.com/GoogleContainerTools/skaffold/pull/8932)
* chore: bump google.golang.org/grpc from 1.50.0 to 1.53.0 in /examples/grpc-e2e-tests/service [#8933](https://github.com/GoogleContainerTools/skaffold/pull/8933)
* chore: bump google.golang.org/grpc from 1.50.0 to 1.53.0 in /integration/examples/grpc-e2e-tests/service [#8934](https://github.com/GoogleContainerTools/skaffold/pull/8934)
* chore: bump google.golang.org/grpc from 1.48.0 to 1.53.0 in /integration/examples/grpc-e2e-tests/cloud-spanner-bootstrap [#8931](https://github.com/GoogleContainerTools/skaffold/pull/8931)
* chore: bump puma from 4.3.12 to 5.6.7 in /examples/ruby/backend [#9036](https://github.com/GoogleContainerTools/skaffold/pull/9036)
* chore: bump puma from 4.3.12 to 5.6.7 in /integration/examples/ruby/backend [#9037](https://github.com/GoogleContainerTools/skaffold/pull/9037)
* chore: bump ossf/scorecard-action from 2.1.3 to 2.2.0 [#8915](https://github.com/GoogleContainerTools/skaffold/pull/8915)

Docs, Test, and Release Updates:
* fix: verify flaky tests [#9050](https://github.com/GoogleContainerTools/skaffold/pull/9050)
* docs: Update documentation [#9017](https://github.com/GoogleContainerTools/skaffold/pull/9017)
* docs: add example to use cloudrun deployer + local build [#8983](https://github.com/GoogleContainerTools/skaffold/pull/8983)
* docs: add anchors to yaml paths [#8541](https://github.com/GoogleContainerTools/skaffold/pull/8541)
* docs: schema version mapping [#8973](https://github.com/GoogleContainerTools/skaffold/pull/8973)
* docs: remove duplicate page meta links [#8824](https://github.com/GoogleContainerTools/skaffold/pull/8824)
* docs: document cmd template function [#8929](https://github.com/GoogleContainerTools/skaffold/pull/8929)

Huge thanks goes out to all of our contributors for this release:

- Brian Topping
- dependabot[bot]
- ericzzzzzzz
- Frank Farzan
- Jack Wilsdon
- James C Scott III
- Maxim De Clercq
- Michael Plump
- Renzo Rojas

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
