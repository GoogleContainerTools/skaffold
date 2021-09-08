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

## Background: ko image names and Go import paths

Ko uses Go import paths to build images. The
[`ko publish`](https://github.com/google/ko#build-an-image) command takes a
required positional argument, which can be either a local file path or a Go
import path. If the argument is a local file path (as per
[`go/build.IsLocalImport()`](https://pkg.go.dev/go/build#IsLocalImport))
then, ko resolves the local file path to a Go import path (see
[`github.com/google/ko/pkg/build`](https://github.com/google/ko/blob/ab4d264103bd4931c6721d52bfc9d1a2e79c81d1/pkg/build/gobuild.go#L261)).

The import path must be of the package than contains the `main()` function.
For instance, to build Skaffold using ko, from the repository root directory:

```sh
ko publish ./cmd/skaffold
```

or

```sh
ko publish github.com/GoogleContainerTools/skaffold/cmd/skaffold
```

When the ko CLI is used to
[populate the image name in templated Kubernetes resource files](https://github.com/google/ko#kubernetes-integration),
only the Go import path option can be used, and the import path must be
prefixed by the `ko://` scheme, e.g.,
`ko://github.com/GoogleContainerTools/skaffold/cmd/skaffold`.

Ko determines the image name from the container image registry (provided by the
`KO_DOCKER_REPO` environment variable) and the Go import path. The Go import
path is appended in one of these ways:

- The last path segment (e.g., `skaffold`), followed by a hyphen and a MD5
  hash. This is the default behavior of the `ko publish` command.

- The last path segment (e.g., `skaffold`) only, if `ko publish` is invoked
  with the `-B` or `--base-import-paths` flag.

- The full import path, lowercased (e.g.,
  `github.com/googlecontainertools/skaffold/cmd/skaffold`), if `ko publish` is
  invoked with the `-P` or `--preserve-import-paths` flag. This is the option
  used by projects such as Knative (see the
  [`release.sh` script](https://github.com/knative/serving/blob/v0.24.0/vendor/knative.dev/hack/release.sh#L98))
  and Tekton
  (see the pipeline in
  [publish.yaml](https://github.com/tektoncd/pipeline/blob/v0.25.0/tekton/publish.yaml#L137)).

- No import path (just `KO_DOCKER_REPO`), if `ko publish` is invoked with the
  `--bare` flag.

## Supporting existing Skaffold users

The Skaffold ko builder follows the existing Skaffold image naming logic. This
means that the image naming behavior doesn't change for existing Skaffold users
who migrate from other builders to the ko builder.

The ko builder achieves this by using ko's
[`Bare`](https://github.com/google/ko/blob/ab4d264103bd4931c6721d52bfc9d1a2e79c81d1/pkg/commands/options/publish.go#L60)
naming option.

By using this option, the image name is not tied to the Go import path. If the
Skaffold
[default repo](https://skaffold.dev/docs/environment/image-registries/) value
is `gcr.io/k8s-skaffold` and the value of the `image` field in `skaffold.yaml`
is `skaffold`, the resulting image name will be `gcr.io/k8s-skaffold/skaffold`.

It is still necessary to resolve the Go import path for the underlying ko
implementation. To do so, the ko builder determines the import path based on
the value of the `target` config field. The `target` config field refers to the
location of a main package and corresponds to a `go build` target, e.g.,
`go build ./cmd/skaffold`. Using the `target` field results in deterministic
behavior even in cases where there are multiple main packages in different
directories.

If `target` is a relative path (and it will be most of the time), it is
relative to the current
[`context`](https://skaffold.dev/docs/references/yaml/#build-artifacts-context)
(a.k.a.
[`Workspace`](https://github.com/GoogleContainerTools/skaffold/blob/v1.27.0/pkg/skaffold/schema/latest/v1/config.go#L832))
directory.

For example, to build Skaffold itself, with `package main` in the
`./cmd/skaffold/` subdirectory, the config would be as follows:

```yaml
apiVersion: skaffold/v2beta19
kind: Config
build:
  artifacts:
  - image: skaffold
    context: .
    ko:
      target: ./cmd/skaffold
```

Users can specify `./...` as a target to make ko locate the main package. If
there are multiple main packages,
[ko fails](https://github.com/google/ko/blob/780c2812926cd706423e2ba65aeb1beb842c04af/pkg/build/gobuild.go#L270).

Implementation note: The value of `target` will be the input when invoking
[`QualifyImport()`](https://github.com/GoogleContainerTools/skaffold/blob/953594000be68fa8fad0ec4636ab03f8153a1c08/pkg/skaffold/build/ko/build.go#L93).

If the Go sources and the `go.mod` file are in a subdirectory of the `context`
directory, users can use the `dir` config field to specify the path where the
ko builder runs the `go` tool.

## Supporting existing ko users

To support existing ko users moving to Skaffold, the ko builder also supports
`image` names in `skaffold.yaml` that use the Go import path, prefixed by the
`ko://` scheme. Examples of such image references in Kubernetes manifest files
can be seen in projects such as
[Knative](https://github.com/knative/serving/blob/main/config/core/deployments/activator.yaml#L41)
and
[Tekton](https://github.com/tektoncd/pipeline/blob/v0.25.0/config/controller.yaml#L66).

In the case of `ko://`-prefixed image names, the Skaffold ko builder
constructs the image name by:

1.  Removing the `ko://` scheme prefix.
2.  Transforming the import path to a valid image name using the function
    [`SanitizeImageName()`](https://github.com/GoogleContainerTools/skaffold/blob/v1.27.0/pkg/skaffold/docker/reference.go#L83)
    (from the package
    `github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker`).
3.  Combining the Skaffold default repo with the transformed import path as per
    existing Skaffold image naming logic.

This will result in image names that match those produced by the `ko` CLI when
using the `-P` or `--preserve-import-paths` flag. For example, if the Skaffold
default repo is `gcr.io/k8s-skaffold` and the `image` name in `skaffold.yaml`
is `ko://github.com/GoogleContainerTools/skaffold/cmd/skaffold`, the resulting
image name will be
`gcr.io/k8s-skaffold/github.com/googlecontainertools/skaffold/cmd/skaffold`.

Real-world examples of image names that follow this naming convention can be
found in the Tekton and Knative release manifests. For instance, view the
images in the Knative Serving release YAMLs:

```sh
curl -sL https://github.com/knative/serving/releases/download/v0.24.0/serving-core.yaml | grep 'image: '
```

If the `image` field in `skaffold.yaml` starts with the `ko://` scheme prefix,
the Skaffold ko builder uses the Go import path that follows the prefix. For
example, to build Skaffold itself, with `package main` in the `./cmd/skaffold/`
subdirectory, the config would be as follows:

```yaml
apiVersion: skaffold/v2beta19
kind: Config
build:
  artifacts:
  - image: ko://github.com/GoogleContainerTools/skaffold/cmd/skaffold
    context: .
    ko: {}
```

The `target` field is ignored if the `image` field starts with the `ko://`
scheme prefix.

## Debugging ko images using Skaffold

Ko provides the
[`DisableOptimizations`](https://github.com/google/ko/blob/780c2812926cd706423e2ba65aeb1beb842c04af/pkg/commands/options/build.go#L34)
build option to
[set `gcflags` to disable optimizations and inlining](https://github.com/google/ko/blob/335c1ac8a6fdcc5eb0bb26579e4b44b4c62a9565/pkg/build/gobuild.go#L709-L712).
The ko builder will set `DisableOptimizations` to `true` when the Skaffold
`runMode` is `Debug`.

Skaffold can
[recognize Go-based container images](https://skaffold.dev/docs/workflows/debug/#go-runtime-go-protocols-dlv)
built by ko by the presence of the
[`KO_DATA_PATH` environment variable](https://github.com/google/ko/blob/9a256a4b1920ed5fd5fbf77d987fddbfb9733c40/pkg/build/gobuild.go#L803-L810).
This allows Skaffold to transform Pod specifications to enable remote
debugging.

The ko builder implementation work will add `KO_DATA_PATH` to
[the set of environment variables used to detect Go-based applications](https://github.com/GoogleContainerTools/skaffold/blob/c75c55133e709b2fee906eb158be13c7ccfa72cd/pkg/skaffold/debug/transform_go.go#L67-L72)
and updating the associated unit tests and
[documentation](https://github.com/GoogleContainerTools/skaffold/blob/01a833614efd780ddece99198aa7fdcf3f355706/docs/content/en/docs/workflows/debug.md#L103).

## Design

Adding the ko builder requires making config changes to the Skaffold schema.

1.  Add a `KoArtifact` type:

    ```go
    // KoArtifact builds images using [ko](https://github.com/google/ko).
    type KoArtifact struct {
    	// Asmflags are assembler flags passed to the builder.
    	Asmflags []string `yaml:"asmflags,omitempty"`

    	// BaseImage overrides the default ko base image.
    	// Corresponds to, and overrides, the `defaultBaseImage` in `.ko.yaml`.
    	BaseImage string `yaml:"fromImage,omitempty"`

    	// Dependencies are the file dependencies that skaffold should watch for both rebuilding and file syncing for this artifact.
    	Dependencies *KoDependencies `yaml:"dependencies,omitempty"`

      // Dir is the directory where the `go` tool will be run.
      // The value is a directory path relative to the `context` directory.
      // If empty, the `go` tool will run in the `context` directory.
      // Examples: `live-at-head`, `compat-go114`
      Dir string `yaml:"dir,omitempty"`

    	// Env are environment variables, in the `key=value` form, passed to the build.
    	// These environment variables are only used at build time.
    	// They are _not_ set in the resulting container image.
    	Env []string `yaml:"env,omitempty"`

    	// Flags are additional build flags passed to the builder.
    	// For example: `["-trimpath", "-v"]`.
    	Flags []string `yaml:"flags,omitempty"`

    	// Gcflags are Go compiler flags passed to the builder.
    	// For example: `["-m"]`.
    	Gcflags []string `yaml:"gcflags,omitempty"`

    	// Labels are key-value string pairs to add to the image config.
    	// For example: `{"foo":"bar"}`.
    	Labels map[string]string `yaml:"labels,omitempty"`

    	// Ldflags are linker flags passed to the builder.
    	// For example: `["-buildid=", "-s", "-w"]`.
    	Ldflags []string `yaml:"ldflags,omitempty"`

    	// Platforms is the list of platforms to build images for. Each platform
    	// is of the format `os[/arch[/variant]]`, e.g., `linux/amd64`.
    	// By default, the ko builder builds for `all` platforms supported by the
    	// base image.
    	Platforms []string `yaml:"platforms,omitempty"`

    	// SourceDateEpoch is the `created` time of the container image.
    	// Specify as the number of seconds since January 1st 1970, 00:00 UTC.
    	// You can override this value by setting the `SOURCE_DATE_EPOCH`
    	// environment variable.
    	SourceDateEpoch uint64 `yaml:"sourceDateEpoch,omitempty"`

      // Target is the location of the main package.
      // If target is specified as a relative path, it is relative to the `context` directory.
      // If target is empty, the ko builder looks for the main package in the `context` directory only, but not in any subdirectories.
      // If target is a pattern with wildcards, such as `./...`, the expansion must contain only one main package, otherwise ko fails.
      // Target is ignored if the `ImageName` starts with `ko://`.
      // Example: `./cmd/foo`
      Target string `yaml:"target,omitempty"`
    }
    ```

2.  Add a `KoArtifact` field to the `ArtifactType` struct:

    ```go
    type ArtifactType struct {
      [...]
    	// KoArtifact builds images using [ko](https://github.com/google/ko).
    	KoArtifact *KoArtifact `yaml:"ko,omitempty" yamltags:"oneOf=artifact"`
    }
    ```

3.  Define `KoDependencies`:

    ```go
    // KoDependencies is used to specify dependencies for an artifact built by ko.
    type KoDependencies struct {
	  	// Paths should be set to the file dependencies for this artifact, so that the skaffold file watcher knows when to rebuild and perform file synchronization.
	  	// Defaults to {"go.mod", "**.go"}
	  	Paths []string `yaml:"paths,omitempty" yamltags:"oneOf=dependency"`

	  	// Ignore specifies the paths that should be ignored by skaffold's file watcher.
    	// If a file exists in both `paths` and in `ignore`, it will be ignored, and will be excluded from both rebuilds and file synchronization.
	  	Ignore []string `yaml:"ignore,omitempty"`
    }
    ```

4.  Add `KO` to the `BuilderType` enum in `proto/enums/enums.proto`:

    ```proto
    enum BuilderType {
        // Could not determine builder type
        UNKNOWN_BUILDER_TYPE = 0;
        // JIB Builder
        JIB = 1;
        // Bazel Builder
        BAZEL = 2;
        // Buildpacks Builder
        BUILDPACKS = 3;
        // Custom Builder
        CUSTOM = 4;
        // Kaniko Builder
        KANIKO = 5;
        // Docker Builder
        DOCKER = 6;
        // Ko Builder
        KO = 7;
    }
    ```

5.  In `skaffold init`, default to the ko builder for any images where the
    name starts with the ko prefix `ko://`.

### Builder config schema

Example basic config, this will be sufficient for many users:

```yaml
apiVersion: skaffold/v2beta19
kind: Config
build:
  artifacts:
  - image: skaffold-example-ko
    ko: {}
```

The value of the `image` field is the Go import path of the app entry point,
[prefixed by `ko://`](https://github.com/google/ko/pull/58).

A more comprehensive example config:

```yaml
apiVersion: skaffold/v2beta19
kind: Config
build:
  artifacts:
  - image: ko://github.com/GoogleContainerTools/skaffold/examples/ko-complete
    ko:
      asmflags: []
      fromImage: gcr.io/distroless/base:nonroot
      dependencies:
        paths:
        - go.mod
        - "**.go"
      dir: '.'
      env:
      - GOPRIVATE=source.developers.google.com
      flags:
      - -trimpath
      - -v
      gcflags:
      - -m
      labels:
        foo: bar
        baz: frob
      ldflags:
      - -buildid=
      - -s
      - -w
      platforms:
      - linux/amd64
      - linux/arm64
      target: ./cmd/foo
```

ko requires setting a
[`KO_DOCKER_REPO`](https://github.com/google/ko#choose-destination)
environment variable to specify where container images are pushed. The Skaffold
[default repo](https://skaffold.dev/docs/environment/image-registries/)
maps directly to this value.

### Resolved questions

1.  Should Skaffold embed ko as a Go module, or shell out?

     __Resolved:__ Embed as a Go module

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

    Suggest embedding as a Go module.

2.  Should Skaffold use base image settings from
    [`.ko.yaml`](https://github.com/google/ko#configuration) if the ko builder
    definition in `skaffold.yaml` doesn't specify a base image?

    __Resolved:__ Yes, to simplify adoption of Skaffold for existing ko users.

3.  If a config value is set both as an environment variable, and as a config
    value, which takes precedence? E.g., `ko.sourceDateEpoch` vs
    `SOURCE_DATE_EPOCH`.

    __Resolved:__ Follow existing Skaffold patterns.

4.  Should the ko builder have a config option for
    [`SOURCE_DATE_EPOCH`](https://reproducible-builds.org/specs/source-date-epoch/),
    or should users specify the value via an environment variable?

    __Resolved__: Specify via the reproducible builds spec environment variable
    `SOURCE_DATE_EPOCH`, see
    <https://github.com/google/ko#why-are-my-images-all-created-in-1970> and
    <https://reproducible-builds.org/docs/source-date-epoch/>.

### Open questions

1.  Should we default dependency paths to `{"go.mod", "**.go"}` instead of
    `{"."}`.?

    The former is a useful default for many (most?) Go apps, and it's used
    in the `custom` example. The latter is the default for some other builders.

    __Not Yet Resolved__

2.  Add a Google Cloud Build (`gcb`) support for the ko builder?

    By embedding ko as a module, there is no need for a ko-specific Skaffold
    builder image.

    __Not Yet Resolved__

3.  File sync support: Should we limit this to
    [ko static assets](https://github.com/google/ko#static-assets) only?

    This is the only way to include additional files in a container image
    built by ko.

    __Not Yet Resolved__

4.  Should the ko builder be the default for `skaffold init`, instead of
    buildpacks, for Go apps, when there's no Dockerfile and no Bazel workspace
    file?

    Suggest yes, to make Skaffold a compelling choice for Go developers.

    __Not Yet Resolved__

## Approach

Implement the ko builder as a series of small PRs that can be merged one by one.
The PRs should not surface any new user-visible behavior until the feature is
ready.

This approach has a lower risk than implementing the entire feature on a
separate branch before merging all at once.

The steps roughly outlined:

1.  Add dependency on the `github.com/google/ko` module.

2.  Implement the core ko build and publish logic, including unit tests in the
    package `github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/ko`.

    Support this implementation with "dummy" schema definitions for types such
    as `KoArtifact` in a separate "temporary" package. This package only exists
    until the definitions are merged into the latest `v1` schema, and it allows
    for evolution of the types until they are committed to the schema.

3.  Add integration test for the ko builder to the `integration` package, and
    an example app + config to a new `integration/examples/ko` directory.

    To avoid failures in schema unit tests from the new example in the
    integration directory, add an `if` statement to `TestParseExamples` in the
    package `github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema` that
    skips directories called `ko`.

4.  Add the ko builder schema types (`KoArtifact` and `KoDependency`) to the
    latest unreleased schema (`v2beta18`?).

    For the `ArtifactType` struct, add a `KoArtifact` field, but set its
    YAML flag key to `"-"`
    (full syntax `` `yaml:"-,omitempty" yamltags:"oneOf=artifact"` ``).

    Do _not_ yet add the `KO = 7;` entry to the `BuilderType` enum in
    `proto/enums/enums.proto`.

5.  Plumb the code in the package `pkg/skaffold/build/ko` into the other parts
    of the Skaffold codebase that interacts with the config schema.

    E.g., a `case` statement for `a.KoArtifact` in the `newPerArtifactBuilder`
    function in `pkg/skaffold/build/local` (file `types.go`).

6.  When we are ready to add the ko builder as an alpha feature to an upcoming
    Skaffold release, set the `KoArtifact` YAML flag key to `ko` and add `KO`
    to the `BuilderType` enum.

## Implementation plan

1.  [Done] Define integration points in the ko codebase that allows ko to be
    used from Skaffold without duplicating existing ko CLI code.

    In the package `github.com/google/ko/pkg/commands`:

    [`resolver.go`](https://github.com/google/ko/blob/ee23538378722e060a2f7c7800f226e0b82e09e7/pkg/commands/resolver.go#L110)
    ```go
    // NewBuilder creates a ko builder
    func NewBuilder(ctx context.Context, bo *options.BuildOptions) (build.Interface, error)
    ```

    [`resolver.go`](https://github.com/google/ko/blob/ee23538378722e060a2f7c7800f226e0b82e09e7/pkg/commands/resolver.go#L146)
    ```go
    // NewPublisher creates a ko publisher
    func NewPublisher(po *options.PublishOptions) (publish.Interface, error)
    ```

    [`publisher.go`](https://github.com/google/ko/blob/ee23538378722e060a2f7c7800f226e0b82e09e7/pkg/commands/publisher.go#L28)
    ```go
    // PublishImages publishes images
    func PublishImages(ctx context.Context, importpaths []string, pub publish.Interface, b build.Interface) (map[string]name.Reference, error)
    ```

    Add build and publish options to support Skaffold config propagating to
    ko. In the package `github.com/google/ko/pkg/commands/options`:

    [`build.go`](https://github.com/google/ko/blob/ee23538378722e060a2f7c7800f226e0b82e09e7/pkg/commands/options/build.go#L25)
    ```go
    type BuildOptions struct {
	      // BaseImage enables setting the default base image programmatically.
	      // If non-empty, this takes precedence over the value in `.ko.yaml`.
	      BaseImage string

	      // WorkingDirectory allows for setting the working directory for invocations of the `go` tool.
	      // Empty string means the current working directory.
	      WorkingDirectory string

        // UserAgent enables overriding the default value of the `User-Agent` HTTP
	      // request header used when retrieving the base image.
	      UserAgent string

        [...]
    }
    ```

    [`publish.go`](https://github.com/google/ko/blob/ee23538378722e060a2f7c7800f226e0b82e09e7/pkg/commands/options/publish.go#L29)
    ```go
    type PublishOptions struct {
	      // DockerRepo configures the destination image repository.
	      // In normal ko usage, this is populated with the value of $KO_DOCKER_REPO.
	      DockerRepo string

	      // LocalDomain overrides the default domain for images loaded into the local Docker daemon. Use with Local=true.
	      LocalDomain string

	      // UserAgent enables overriding the default value of the `User-Agent` HTTP
	      // request header used when pushing the built image to an image registry.
	      UserAgent string

        [...]
    }
    ```

2.  Add ko builder with support for existing ko config options. Provide
    this as an Alpha feature in an upcoming Skaffold release.

    Config options supported, all are optional:

    -   `fromImage`, to override the default distroless base image
    -   `dependencies`, for Skaffold file watching
    -   `dir`, if Go sources are not in the `context` directory
    -   `env`, to support ko CLI users who currently set environment variables
        such as `GOFLAGS` when running ko.
    -   `labels`, e.g., to
        [link an image to a Git repository](https://github.com/opencontainers/image-spec/blob/main/annotations.md#pre-defined-annotation-keys)
    -   `platforms`
    -   `sourceDateEpoch`
    -   `flags`, e.g., `-v`, `-trimpath`
    -   `ldflags`, e.g., `-s`

    Example `skaffold.yaml` supported at this stage:

    ```yaml
    apiVersion: skaffold/v2beta19
    kind: Config
    build:
      artifacts:
      - image: skaffold-ko
        ko:
          fromImage: gcr.io/distroless/base:nonroot
          dependencies:
            paths:
            - go.mod
            - "**.go"
          dir: '.'
          env:
          - GOPRIVATE=source.developers.google.com
          labels:
            foo: bar
            baz: frob
          ldflags:
          - -s
          platforms:
          - linux/amd64
          - linux/arm64
          target: ./cmd/foo
    ```

3.  Implement Skaffold config support for additional ko config options not
    currently supported by ko:

    -   `asmflags`
    -   `gcflags`

    Provide this as a feature in an upcoming Skaffold release.

## Integration test plan

Please describe what new test cases you are going to consider.

1.  Unit and integration tests for ko builder, similar to other builders.

    The integration tests should be written to catch situations such as where
    changes to ko interfaces break the Skaffold ko builder.

2.  Test that the ko flag
    [`--disable-optimization`](https://github.com/google/ko/blob/f7df8106196518df5c6c35432843421e33990329/pkg/commands/options/build.go#L34)
    is added for debugging.

3.  Add basic and comprehensive ko examples to the `integration/examples`
    directory.

4.  TBC
