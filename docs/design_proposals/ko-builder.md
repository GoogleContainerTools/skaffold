# ko builder

* Author(s): Halvard Skogsrud (@halvards)
* Design Shepherd: \<skaffold-core-team-member\>
* Date: 2021-03-29
* Status: Draft

## Objectives

Fast, standardized, reproducible, configuration-less, Docker-less, and
secure-by-default container image builds for Go apps.

## Background

[ko](https://github.com/google/ko) is a container image builder for Go. It's
[fast](https://cloud.google.com/blog/topics/developers-practitioners/ship-your-go-applications-faster-cloud-run-ko),
doesn't use a `Dockerfile` or rely on the Docker daemon, and uses
[distroless](https://github.com/GoogleContainerTools/distroless) base images by
default. ko is to Go apps what
[Jib](https://github.com/GoogleContainerTools/jib) is to JVM-based apps
(approximately).

The [Knative](https://knative.dev/) and [Tekton](https://tekton.dev/) open
source projects use ko.

## Proposal

This proposal adds a new `ko` builder to Skaffold, based on the `ko publish`
command. The integration does _not_ include other ko functionality related to
[rendering](https://github.com/google/ko#ko-resolve) manifests,
[deploying](https://github.com/google/ko#ko-apply) to Kubernetes clusters, and
[file watching](https://github.com/google/ko/blob/f7df8106196518df5c6c35432843421e33990329/pkg/commands/resolver.go#L240).

Compared to ...

- [the Cloud Native buildpacks builder](https://skaffold.dev/docs/pipeline-stages/builders/buildpacks/),
  the ko builder is
  [fast](https://cloud.google.com/blog/topics/developers-practitioners/ship-your-go-applications-faster-cloud-run-ko),
  doesn't require Docker, and uses a default base image that has a small attack
  surface ([distroless](https://github.com/GoogleContainerTools/distroless)).

- [the Docker builder](https://skaffold.dev/docs/pipeline-stages/builders/docker/),
  the ko builder standardizes builds, avoiding artisanal
  [snowflake](https://martinfowler.com/bliki/SnowflakeServer.html)
  `Dockerfile`s. It also doesn't require the Docker daemon, so builds can
  run in security-constrained environments.

- [the Kaniko builder](https://skaffold.dev/docs/pipeline-stages/builders/docker/#dockerfile-in-cluster-with-kaniko),
  the ko builder doesn't need a Kubernetes cluster, and avoids the
  previously-mentioned artisanal `Dockerfile`s.

- [the Bazel builder](https://skaffold.dev/docs/pipeline-stages/builders/bazel/),
  the ko builder doesn't require users to adopt Bazel. However, users who
  already use Bazel for their Go app should use the Bazel builder.

- [the custom builder](https://skaffold.dev/docs/pipeline-stages/builders/custom/),
  the ko builder is portable:

  1.  The Skaffold config can be shared with other developers and ops teams,
      and used in CI/CD pipelines, without requiring users to install
      additional tools such as Docker Engine or
      [crane](https://github.com/google/go-containerregistry/blob/main/cmd/crane/README.md)
      (or even ko, depending on how the builder is implemented). This eases the
      path to adoption of Skaffold and reduces friction for users, both for
      local development, and for anyone using Skaffold in CI/CD pipelines.

  2.  The ko builder doesn't require running custom shell scripts. This means
      more standardized builds, a desirable trait for enterprise users.

The ko builder supports and enhances these Skaffold
[features](https://skaffold.dev/docs/):

- _fast local workflow_: building with ko is
  [fast](https://cloud.google.com/blog/topics/developers-practitioners/ship-your-go-applications-faster-cloud-run-ko).

- _share with other developers_: no additional tools are required to
  `skaffold run` with the ko builder, not even Docker. Though if we don't embed
  ko in Skaffold, ko will be a tool that all developers in a team would have to
  install.

- works great as a _CI/CD building block_: when using the ko builder, pipeline
  steps can run using the default Skaffold container image, without
  installing additional tools or keeping toolchain versions in sync across
  local development and CI/CD.

## Design

Adding the ko builder requires making config changes to the Skaffold schema.

1.  Add an entry to BuilderType enum in `proto/enums/enums.proto`:

    ```proto
        // ko Builder
        KO = 7;
    ```

2.  Add a `KoArtifact` type:

    ```go
    type KoArtifact struct {
    	// Annotations are key-value string pairs to add to the image manifest.
    	// Also known as `LABEL` in `Dockerfile`s.
    	// Ref: https://github.com/opencontainers/image-spec/blob/master/annotations.md
    	Annotations map[string]string `yaml:"annotations,omitempty"`

    	// BaseImage overrides the default ko base image.
    	// Corresponds to, and overrides, the `defaultBaseImage` in `.ko.yaml`.
    	BaseImage string `yaml:"fromImage,omitempty"`

    	// Env are environment variables, in the `key=value` form, passed to the build.
    	// For example: `CGO_ENABLED=1`.
    	Env []string `yaml:"env,omitempty"`

    	// Platforms is the list of platforms to build images for. Each platform
    	// is of the format `os/arch[/variant]`, e.g., `linux/amd64`.
    	// By default, the ko builder builds for `all` platforms supported by the
    	// base image.
    	Platforms []string `yaml:"platforms,omitempty"`

    	// SourceDateEpoch is the `created` time of the container image.
    	// Specify as the number of seconds since January 1st 1970, 00:00 UTC.
    	// You can override this value by setting the `SOURCE_DATE_EPOCH`
    	// environment variable.
    	SourceDateEpoch unit64 `yaml:"sourceDateEpoch,omitempty"`
    }
    ```

3.  Add a `KoArtifact` field to the `ArtifactType` struct:

    ```go
    type ArtifactType struct {
    	// KoArtifact builds images using [ko](https://github.com/google/ko).
    	KoArtifact *KoArtifact `yaml:"ko,omitempty" yamltags:"oneOf=artifact"`
    }
    ```

Example basic config, this will be sufficient for many users:

```yaml
apiVersion: skaffold/v2beta14
kind: Config
build:
  artifacts:
  - image: ko://example.com/helloworld
    ko: {}
```

The value of the `image` field is the Go import path of the app entry point,
[prefixed by `ko://`](https://github.com/google/ko/pull/58).

ko requires setting a
[`KO_DOCKER_REPO`](https://github.com/google/ko#choose-destination)
environment variable. This determines where container images are pushed.
The Skaffold
[default repo](https://skaffold.dev/docs/environment/image-registries/)
maps directly to this value.

### Open Questions

1.  Should Skaffold embed ko (as a Go module), or shell out?

    Benefits of embedding:

    - Skaffold can pin the ko version it supports in its `go.mod` file. Users
      wouldn't raise bugs/issues for incompatible version pairings of Skaffold
      and ko.

    - Reduce toolchain maintenance toil for users. Skaffold users wouldn't need
      to synchronize ko versions used by different team members or in their CI
      build, since the Skaffold version determines the ko version.

    - Portability. Skaffold+ko users only need one tool for their container
      image building needs: the `skaffold` binary. (Plus the Go distribution,
      of course.) The current `gcr.io/k8s-skaffold/skaffold` container image
      could serve as a build and deploy image for CI/CD pipeline steps.

    Embedding ko would require some level of documented behavioural stability
    guarantees for the most ko interfaces that Skaffold would use, such as
    [`build.Interface`](https://github.com/google/ko/blob/82cabb40bae577ce3bc016e5939fd85889538e8b/pkg/build/build.go#L24)
    and
    [`publish.Interface`](https://github.com/google/ko/blob/82cabb40bae577ce3bc016e5939fd85889538e8b/pkg/publish/publish.go#L24),
    or others?

    Benefits of shelling out:

    - It's an established pattern used by other Skaffold builders.

    - It would allow Skaffold to support a range of ko versions. On the other
      hand, these versions would need to be tracked and documented.

    - No need to resolve dependency version differences between Skaffold and
      ko.

    - If a new ko version provided a significant bug fix, there would be no
      need to release a new version of Skaffold for this fix.

    Shelling out to ko would require some stability guarantees for the
    `ko publish` subcommand.

    Suggest embedding as a Go module. __Not Yet Resolved__

2.  Should the ko builder be the default for `skaffold init`, instead of
    buildpacks, for Go apps, when there's no Dockerfile and no Bazel workspace
    file?

    Suggest yes, to make Skaffold a compelling choice for Go developers.
    __Not Yet Resolved__

3.  Should Skaffold use base image settings from
    [`.ko.yaml`](https://github.com/google/ko#configuration) if the ko builder
    definition in `skaffold.yaml` doesn't specify a base image?

    Suggest yes to simplify adoption of Skaffold for existing ko users.
    __Not Yet Resolved__

4.  If a config value is set both as an environment variable, and as a config
    value, which takes precedence? E.g., `ko.sourceDateEpoch` vs
    `SOURCE_DATE_EPOCH`.

    Follow existing Skaffold pattern - is there one? __Not Yet Resolved__

## -- Sections below haven't been fleshed out --

## Implementation plan

1. TBC

## Integration test plan

Please describe what new test cases you are going to consider.

1.  Unit and integration tests for ko builder, similar to other builders.

2.  Test that the ko flag
    [`--disable-optimization`](https://github.com/google/ko/blob/f7df8106196518df5c6c35432843421e33990329/pkg/commands/options/build.go#L34)
    is added for debugging.

3.  File sync testing?

4.  Add ko example to the `examples` directory.
