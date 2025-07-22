# buildx transparent support

* Author(s): Mariano Reingart (@reingart)
* Design Shepherd:
* Date: 2025-02-02
* Status: Draft

## Objectives

Transparent local and remote container builds via buildx, a Docker CLI plugin for extended capabilities with BuildKit.

## Background

[buildx](https://docs.docker.com/reference/cli/docker/buildx/) is an enhanced container builder using BuildKit that can replace traditional
`docker build` command, with almost the same syntax (transparently).

This adds several distributed features and advanced capabilities:
* local & remote build-kit builders, either running standalone, in docker containers or Kubernetes clusters
* improved cache support (registry destination, pushing full metadatada and multi-stage layers)
* improved multi-platform image building support (eg. x86_64 and arm64 combined, with emulation or cross-compilation)

This features are very useful for corporate CI/CD, for example when using GitLab, where the use case requires:
* using ephemeral remote buildkit instances (rootless)
* using remote docker registries for cache, with multi-stage and multi-platform images support
* using a different cache destination tag for flexibility and workflows separation

The remote BuildKit [rootless](https://github.com/moby/buildkit/blob/master/docs/rootless.md) support is useful in cases
where a privileged docker daemonis not possible or desirable due security policies (eg. untrusted images or review pipelines).
Daemon-less mode is also useful to offload container building from developers notebooks, sharing and reusing remote caches more effectively.

The buildx command also supports exporting the build cache using `--cache-to`, useful for remote shared caches in distributed use cases.
Beside speed-ups thanks to caching improvements for metadata and multi-stage layers, this could allow different branches
to have different cache destinations (production vs development cache, with different permissions).

Multi-platform combined image builds are directly supported by buildx, including improved caching of common layers and emulation / cross-compilation.
This could simplify pipelines and provide faster builds of complex codebases.

References:

* https://www.docker.com/blog/image-rebase-and-improved-remote-cache-support-in-new-buildkit/
* https://www.docker.com/blog/faster-multi-platform-builds-dockerfile-cross-compilation-guide/

## Proposal

This proposal aims to improve Skaffold's build process by adding transparent support for docker buildx.
The primary goal is to enable users to leverage the advanced features of buildx, such as remote buildkit builders and improved caching, without significant changes to their existing Skaffold configurations.

New Global Configs:

* `buildx-builder`: Enables automatic detection of buildx and multiple builder support.
* `cache-tag`: Allows overriding the default cache tagging strategy, useful for managing caches across different branches or environments.

Skaffold Schema changes:

* `cacheTo` to specify custom cache destinations, adding `--cache-to` support to the Docker CLI build process when using buildx.

Build Process Enhancements:

* The build process now intelligently detects and utilizes buildx if available and configured.
* Reduced dependency on the local Docker daemon when using buildx, enhancing security and flexibility.

Backward Compatibility:

All changes are designed to be backward compatible.
If buildx is not detected or used, Skaffold will fall back to the traditional Docker builder.

This cache behavior is intended to provide transparent user experience, with similar functionality compared to traditional local docker builds,
without additional boilerplate or different configuration / patches for a CI workflow.

## Design approach

`docker buildx build` can be configured to execute against a [remote buildkit instance](https://docs.docker.com/build/builders/drivers/remote/).
No docker daemon is necessary for this to work, but also that is supported by default using the [docker driver](https://docs.docker.com/build/builders/drivers/docker/).
Additionally, [docker container driver](https://docs.docker.com/build/builders/drivers/docker-container)
or [kubernetes driver](https://docs.docker.com/build/builders/drivers/kubernetes/) are available too for advanced use cases.

Since `docker build` and `docker buildx build` essentially share the same options, no major modifications are needed to Skaffold for backward compatibility.

This proposal implements the logic to detect if buildx is the default builder (looking for an alias in the docker config).
To [set buildx as the default builder](https://github.com/docker/buildx?tab=readme-ov-file#set-buildx-as-the-default-builder), the command `docker buildx install` should be used.

Then, additional buildx features are availables, like multi-platform support and different cache destinations.

Cache adjustment is extended via the the `cache-tag`, useful to point to latest or a generic cache tag, instead of the generated one for this build
(via `tagPolicy` that would be useless in most distributed cases, as it can be invalidated by minor changes, specially if using `inputDigest`).

Multi-platform images now can be built nativelly, without multiplexing the pipeline nor additional steps to stich the different images.

### User experience

To use BuildKit transparently, BuildX should be installed as default builder (this creates an alias in docker config):

```
docker buildx install
```

Then Skaffold should be configured to detect buildx (default builder) and set a generic cache tag:

```
skaffold config set -g buildx-builder default
skaffold config set -g cache-tag cache
```

Example basic config, this will be sufficient for many users:

```yaml
apiVersion: skaffold/v4beta13
build:
  artifacts:
  - image: my-app-image
    context: my-app-image
    docker:
      dockerfile: Dockerfile
      cacheFrom:
      - "my-app-image"
      cacheTo:
      - "my-app-image"
  local:
    useBuildkit: true
    useDockerCLI: true
    tryImportMissing: true
```

* If no tag is specified for cache, the configured `cache-tag` will be used (in this example my-app-image:cache)
* If `cacheTo` destination is not specified, the `cacheFrom` image name and tag will be adjusted, adding `type=registry,mode=max` (only if push images is enabled)

Advanced users would prefer to create buildkit instances for multiplatform images, e.g. using the docker container driver:

```
docker buildx create --driver docker-container --name local
skaffold config set -g buildx-builder local
```

Remote builds are possible, pointing to a remote instance that could be deployed in another host or container:
```
docker buildx create --name remote --driver remote tcp://buildkitd:2375
skaffold config set -g buildx-builder remote
```

### Errors

Example for missing builder (`ERROR: no builder "defaultx" found` if skaffold was configured incorrectly):
```
skaffold build --default-repo localhost:5000 --platform linux/arm64,linux/amd64 --cache-artifacts=false --detect-minikube=false --push 
Generating tags...
 - my-app -> localhost:5000/my-app:v2.11.1-121-gc703038c9-dirty
Starting build...
Building [my-app]...
Target platforms: [linux/arm64,linux/amd64]
ERROR: no builder "defaultx" found
exit status 1. Docker build ran into internal error. Please retry.
If this keeps happening, please open an issue..
```

Example of improper configuration for multi-platform emulation builds:

```
$ /src/out/skaffold build --default-repo localhost:5000 --platform linux/arm64,linux/amd64 --cache-artifacts=false --detect-minikube=false --push 
Generating tags...
 - my-app -> localhost:5000/my-app:v2.11.1-122-ga0fb3239a-dirty
Starting build...
Building [my-app]...
Target platforms: [linux/arm64,linux/amd64]
#0 building with "default" instance using docker driver

...

#19 [linux/arm64 builder 3/3] RUN go build -o /app main.go
#19 0.639 exec /bin/sh: exec format error
#19 ERROR: process "/bin/sh -c go build -o /app main.go" did not complete successfully: exit code: 1
------
 > [linux/arm64 builder 3/3] RUN go build -o /app main.go:
0.639 exec /bin/sh: exec format error
------
Dockerfile:6
--------------------
   4 |     
   5 |     COPY main.go .
   6 | >>> RUN go build -o /app main.go
   7 |     
   8 |     FROM alpine:3
--------------------
ERROR: failed to solve: process "/bin/sh -c go build -o /app main.go" did not complete successfully: exit code: 1
running build: exit status 1. To run cross-platform builds, use a proper buildx builder. To create and select it, run:

	docker buildx create --driver docker-container --name buildkit

	skaffold config set buildx-builder buildkit

For more details, see https://docs.docker.com/build/building/multi-platform/.

```

This is a corner case, as the default docker builder should not be used for multi-platform builds.
An actionable error was returned with the step-by-step instructions.

## Implementation plan

For the minimal viable product (initial PR):

* Add a new global config buildx-builder to enable buildx detection and support for different builders
* Implements logic to detect buildx is the default builder (via docker config alias)
* Remove dependency on docker daemon if using BuildX (still supported to load images if using minikube or similar)
* Add a new global config cache-tag to override default cache tagging (instead of generated tag)
* Add cacheTo to sakffold schema for the cache destination (optional, default is to use new cacheFrom + cache tag)
* Add --cache-to support in Docker CLI build if using BuildX CLI
* Add multiplatform images building support for buildkit under buildx

Additional features, error handling and examples can be implemented in the future.

Future work: include more BuildKit advanced features support like 
[multiple build contexts](https://www.docker.com/blog/dockerfiles-now-support-multiple-build-contexts/),
for an improved mono-repo experience for large code-bases.

Other advanced featues of buildx and buildkit includes [Attestations](https://docs.docker.com/build/metadata/attestations/).
Note: default attestation is disabled for multiplatform builds, as it creates metadata with "unknown/unknown" arch/OS,
causing issues with registry tools an libraries, see [GH discussion](https://github.com/orgs/community/discussions/45969).

## Release plan

The buildx support could go through the release stages Alpha -> Beta -> Stable.
This will allow time for community feedback and avoid unnecessary features or rework.

The following features would be released at each stage:

**Alpha**

Implement minimal changes to support buildx (initial PR):
- buildx detection 
- cache-to export
- multiplatform images

**Beta**

- Implement additional buildx features needed by the community, like multiple context support
- Implement additional actionable errors, if needed
- Update user-facing documentation
- Implement a new buildkit basic example using buildx

**Stable**

- Remove custom buildkit example using custom builder
- Implement a new buildkit advanced example using buildx and remote rootless daemon for CI

## Automated test plan

New test cases were implemented to cover the new functionality (added to existing test suites):

1. Unit tests for buildx, similar to docker build.

 * `TestDockerCLIBuild`: "buildkit buildx load", "buildkit buildx push" (including both buildx detection and cache-to)

2. Integration tests, idem:

 * `TestBuild`: "docker buildx"
 * `TestBuildWithWithPlatform`: "docker buildx linux/amd64", "docker buildx linux/arm64"
 * `TestBuildWithMultiPlatforms`: "build multiplatform images with buildx"

3. Add basic and comprehensive buildx examples to the `integration/examples`
   directory.

Note that for multi-platform images and advanced examples, a registry is needed to push the images.
A docker container with a local registry was implemented in the setup of integration tests.
With this approach, all buildx tests can be run locally (not needing GCP nor any other cloud resource like a remote registry).
This avoids modifications to docker daemon config (like insecure-registry exclusions), but it needs a buildkit running within the host network, as both uses localhost.

## Credits

This proposal is related to [#8172](https://github.com/GoogleContainerTools/skaffold/pull/8172): "Add buildx option for daemon-less BuildKit support".
Initial code was inspired by by a prior [ebekebe fork](https://github.com/ebekebe/skaffold/commit/1c1fdeb18f4d2847e65e283fba498a14745039af).

A more general approach was implemented in this initial PR [#9648](https://github.com/GoogleContainerTools/skaffold/pull/9648),
with configurable builders, actionable errors and multi-platform support.

This actually fixes existing issues like:
* [#5018](https://github.com/GoogleContainerTools/skaffold/issues/5018): "Docker Buildx Integration"
* [#6732](https://github.com/GoogleContainerTools/skaffold/issues/6732): "Support building securely against remote buildkitd"
* [#9197](https://github.com/GoogleContainerTools/skaffold/issues/9197): "Support additional docker buildkit options"

The fix proposed here uses buildx nativelly, avoiding custom build scripts, like the one in the official example:
[custom-buildx](https://github.com/GoogleContainerTools/skaffold/tree/main/examples/custom-buildx)

Future work is related to [#2110](https://github.com/GoogleContainerTools/skaffold/issues/2110): "feature request: more control over the docker build context"

