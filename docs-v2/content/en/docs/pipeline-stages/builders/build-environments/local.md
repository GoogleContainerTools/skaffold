---
title: "Local build"
linkTitle: "Local build"
weight: 10
---

Local build execution is the default execution context.
Skaffold will use your locally-installed build tools (such as Docker, Bazel, Maven or Gradle) to execute the build.

## Configuration

To configure the local execution explicitly, add build type `local` to the build section of `skaffold.yaml`

```yaml
build:
  local: {}
```

The following options can optionally be configured:

{{< schema root="LocalBuild" >}}

## Faster builds

There are a few options for achieving faster local builds.

### Avoiding pushes

When deploying to a [local cluster]({{<relref "/docs/environment/local-cluster" >}}), 
Skaffold defaults `push` to `false` to speed up builds.  The `push`
setting can be set from the command-line with `--push`.

### Parallel builds

The `concurrency` controls the number of image builds that are run in parallel.
Skaffold disables concurrency by default for local builds as several
image builder types (`custom`, `jib`) may change files on disk and
result in side-effects.
`concurrency` can be set to `0` to enable full parallelism, though
this may consume significant resources.
The concurrency setting can be set from the command-line with the
`--build-concurrency` flag.

When artifacts are built in parallel, the build logs are still printed in sequence to make them easier to read.

### Build avoidance with `tryImportMissing`

`tryImportMissing: true` causes Skaffold to avoid building an image when
the tagged image already exists in the destination.  This setting can be
useful for images that are expensive to build.

`tryImportMissing` is disabled by default to avoid the risk from importing
a _stale image_, where the imported image is different from the image
that would have been built from the artifact source.
`tryImportMissing` is best used with a
[tagging policy]({{<relref "/docs/pipeline-stages/taggers" >}}) such as
`imageDigest` or `gitCommit`'s `TreeSha` or `AbbrevTreeSha` variants,
where the tag is computed using the artifact's contents.
