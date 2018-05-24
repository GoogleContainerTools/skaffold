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
* Docker build args passsed to Google Container Builder
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
* Helm deployer now acceptes namespace and values file
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

