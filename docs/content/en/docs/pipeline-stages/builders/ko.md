---
title: "ko"
linkTitle: "ko"
weight: 60
featureId: build.ko
---

[`ko`](https://github.com/google/ko) enables fast, standardized, reproducible,
configuration-less, Docker-less, and multi-platform container image builds for
Go apps.

Skaffold embeds `ko` as a library, so you do not need to download `ko`
to use the ko builder.

## Benefits of the ko builder

Compared to ...

- [the Cloud Native buildpacks builder]({{< relref "/docs/pipeline-stages/builders/buildpacks" >}}),
  the ko builder is
  [fast](https://cloud.google.com/blog/topics/developers-practitioners/ship-your-go-applications-faster-cloud-run-ko),
  doesn't require Docker, and uses a default base image that has a small
  attack surface
  ([distroless](https://github.com/GoogleContainerTools/distroless)).

- [the Docker builder]({{< relref "/docs/pipeline-stages/builders/docker" >}}),
  the ko builder standardizes builds, avoiding artisanal
  [snowflake](https://martinfowler.com/bliki/SnowflakeServer.html)
  `Dockerfile`s. It also doesn't require the Docker daemon, so builds can run
  in environments where Docker isn't available for security reasons.

- [the Kaniko builder]({{< relref "/docs/pipeline-stages/builders/docker#dockerfile-in-cluster-with-kaniko" >}}),
  the ko builder doesn't need a Kubernetes cluster, and it avoids the
  previously-mentioned artisanal `Dockerfile`s.

- [the Bazel builder]({{< relref "/docs/pipeline-stages/builders/bazel" >}}),
  the ko builder doesn't require users to adopt Bazel. However, we recommend
  the Bazel builder for users who already use Bazel for their Go apps.

- [the custom builder]({{< relref "/docs/pipeline-stages/builders/custom" >}}),
  the ko builder standardizes builds, as it doesn't require running `ko` using
  custom shell scripts.

## Configuring the ko builder

The ko builder default configuration is sufficient for many Go apps. To use
the ko builder with its default configuration, provide an empty map in the
`ko` field, e.g.:

```yaml
build:
  artifacts:
  - image: my-simple-go-app
    ko: {}
```

### Base image

`ko` uses the [Distroless](https://github.com/GoogleContainerTools/distroless)
image `gcr.io/distroless/static:nonroot` as the default base image. This is a
small image that provides a
[minimal environment for Go binaries](https://github.com/GoogleContainerTools/distroless/tree/main/base).
The  default base image does not provide a shell, and it does not include
`glibc`.

You can specify a different base image using the ko builder `fromImage` config
field. For instance, if you want to use a base image that contains `glibc` and
a shell, you can use this configuration:

```yaml
    ko:
      fromImage: gcr.io/distroless/base:debug-nonroot
```

### Multi-platform images

The ko builder supports building multi-platform images. The default platform
is `linux/amd64`, but you can configure a list of platforms using the
`platforms` configuration field, e.g.:

```yaml
    ko:
      platforms:
      - linux/amd64
      - linux/arm64
```

You can also supply `[all]` as the value of `platforms`. `all` means that the
ko builder builds images for all platforms supported by the base image.

### Labels / annotations

Use the `labels` configuration field to add
[annotations](https://github.com/opencontainers/image-spec/blob/main/annotations.md)
(a.k.a. [`Dockerfile` `LABEL`s](https://docs.docker.com/engine/reference/builder/#label)),
e.g.:

```yaml
    ko:
      labels:
        org.opencontainers.image.licenses: Apache-2.0
        org.opencontainers.image.source: https://github.com/GoogleContainerTools/skaffold
```

### Build time environment variables

Use the `env` configuration field to specify build-time environment variables.

Example:

```yaml
    ko:
      env:
      - GOCACHE=/workspace/.gocache
      - GOPRIVATE=git.internal.example.com,source.developers.google.com
```

### Dependencies

The `dependencies` section configures what files Skaffold should watch for
changes when in [`dev` mode]({{< relref "/docs/workflows/dev" >}}).

`paths` and `ignore` are arrays that list file patterns to include and ignore.
Any patterns in `ignore` will be ignored by the Skaffold file watcher, even if
they are also specified in `paths`. `ignore` is only used when `paths` is not
empty.

Example:

```yaml
    ko:
      dependencies:
        paths:
        - cmd
        - go.mod
        - pkg
        ignore:
        - vendor
```

If no `dependencies` are specified, the default values are as follows:

```yaml
    ko:
      dependencies:
        paths: ["**/*.go"]
        ignore: []
```

### Build flags

Use the `flags` configuration field to provide flag arguments to `go build`,
e.g.:

```yaml
    ko:
      flags:
      - -v
```

Use the `ldflags` configuration field to provide linker flag arguments, e.g.:

```yaml
    ko:
      ldflags:
      - -s
      - -w
```

`ko` supports templating of `flags` and `ldflags` using environment variables,
e.g.:

```yaml
    ko:
      ldflags:
      - -X main.version={{.Env.VERSION}}
```

These templates are passed through to `ko` and are expanded using
[`ko`'s template expansion implementation](https://github.com/google/ko/blob/v0.9.3/pkg/build/gobuild.go#L632-L660).

### Source file locations

If your Go source files and `go.mod` are not in the `context` directory,
use the `dir` configuration field to specify the path, relative to the
`context` directory, e.g.:

```yaml
    ko:
      dir: ./compat-go114
```

If your `package main` is not in the `context` directory (or in `dir` if
specified), use the `main` configuration field to specify the path or target,
e.g.:

```yaml
    ko:
      main: ./cmd/foo
```

If your `context` directory only contains one `package main` directory, you
can use the `...` wildcard in the `main` field value, e.g., `./...`.

Both `dir` and `main` default to `.`.

## Existing `ko` users

Useful tips for existing `ko` users:

- Specify your destination image registry using Skaffold's
  [`default-repo` functionality]({{< relref "/docs/environment/image-registries" >}}).
  The ko builder does _not_ read the `KO_DOCKER_REPO` environment variable.

- The ko builder supports reading
  [base image configuration](https://github.com/google/ko#overriding-base-images)
  from the `.ko.yaml` file. If you already configure your base images using
  this file, you do not need to specify the `fromImage` field for the
  artifact in `skaffold.yaml`.

- The ko builder supports reading
  [build configs](https://github.com/google/ko#overriding-go-build-settings)
  from the `.ko.yaml` file if `skaffold.yaml` does not specify any of the build
  config fields (`dir`, `main`, `env`, `flags`, and `ldflags`). If you already
  specify these fields in `.ko.yaml`, you do not need to repeat them in
  `skaffold.yaml`.

- When you use the Skaffold ko builder, Skaffold takes care of replacing the
  image placeholder name in your Kubernetes manifest files using its
  [render]({{< relref "/docs/pipeline-stages" >}}) functionality.

- The ko builder supports image name placeholders that consist of the
  `ko://` prefix, followed by the Go import path to the main package.
  This means that Skaffold should work with existing Kubernetes manifest
  files that use this image name placeholder format.

  If you previously rendered Kubernetes manifest using `ko`, e.g.:

  ```shell
  ko resolve --filename k8s/*.yaml > out.yaml
  ```

  You can instead use Skaffold's
  [`render` subcommand]({{< relref "/docs/references/cli#skaffold-render" >}}):

  ```shell
  skaffold render --digest-source local --offline=true --output out.yaml
  ```

## Advanced usage

### Debugging

[Cloud Code](https://cloud.google.com/code/docs) and
[`skaffold debug`]({{< relref "/docs/references/cli#skaffold-debug" >}})
can debug images built using `ko`.

Images built using `ko` are identified as Go apps by the presence of the
`KO_DATA_PATH` environment variable.

### File sync

File `sync` is not supported while the ko builder feature is in Alpha.

### Remote builders

Only `local` builds are supported while the ko builder feature is in Alpha.

### Using the `custom` builder

If the ko builder doesn't support your use of `ko`, you can instead use the
[`custom` builder]({{< relref "/docs/pipeline-stages/builders/custom" >}}).

See the `custom` builder
[example](https://github.com/GoogleContainerTools/skaffold/tree/main/examples/custom).

```yaml
build:
  artifacts:
  - image: ko://github.com/GoogleContainerTools/skaffold/examples/custom
    custom:
      buildCommand: ./build.sh
      dependencies:
        paths:
        - "**/*.go"
        - go.mod
```

If you need to use `ko` via the custom builder rather than the ko builder,
please consider filing an
[issue](https://github.com/GoogleContainerTools/skaffold/issues/new)
that describes your use case.
