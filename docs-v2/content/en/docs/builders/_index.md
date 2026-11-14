---
title: "Build"
linkTitle: "Build"
weight: 42
featureId: build
aliases: [/docs/how-tos/builders, docs/pipeline-stages/builders]
no_list: true
---

Skaffold supports different [tools]({{< relref "/docs/builders/builder-types" >}}) for building images across different [build environments]({{< relref "/docs/builders/build-environments" >}}).

|    | [Local Build]({{< relref "/docs/builders/build-environments/local" >}}) | [In Cluster Build]({{< relref "/docs/builders/build-environments/in-cluster" >}}) | [Remote on Google Cloud Build]({{< relref "/docs/builders/build-environments/cloud-build" >}}) |
|----|:-----------:|:----------------:|:----------------------------:|
| **Dockerfile** | [Yes]({{< relref "/docs/builders/builder-types/docker#dockerfile-locally" >}}) | [Yes]({{< relref "/docs/builders/builder-types/docker#dockerfile-in-cluster-with-kaniko" >}}) | [Yes]({{< relref "/docs/builders/builder-types/docker#dockerfile-remotely-with-google-cloud-build" >}}) |
| **Jib Maven and Gradle** | [Yes]({{< relref "/docs/builders/builder-types/jib#jib-maven-and-gradle-locally" >}}) | - | [Yes]({{< relref "/docs/builders/builder-types/jib#remotely-with-google-cloud-build" >}}) |
| **Cloud Native Buildpacks** | [Yes]({{< relref "/docs/builders/builder-types/buildpacks" >}}) | - | [Yes]({{< relref "/docs/builders/builder-types/buildpacks" >}}) |
| **Bazel** | [Yes]({{< relref "/docs/builders/builder-types/bazel" >}}) | - | - |
| **ko** | [Yes]({{< relref "/docs/builders/builder-types/ko" >}}) | - | [Yes]({{< relref "/docs/builders/builder-types/ko#remote-builds" >}}) |
| **Custom Script** | [Yes]({{<relref "/docs/builders/builder-types/custom#custom-build-script-locally" >}}) | [Yes]({{<relref "/docs/builders/builder-types/custom#custom-build-script-in-cluster" >}}) | - |

## Configuration

The `build` section in the Skaffold configuration file, `skaffold.yaml`,
controls how artifacts are built. To use a specific tool for building
artifacts, add the value representing the tool and options for using that tool
to the `build` section.

For detailed per-builder [Skaffold Configuration]({{< relref "/docs/design/config.md" >}}) options,
see [skaffold.yaml References]({{< relref "/docs/references/yaml" >}}).

## Build caching

Skaffold caches built artifacts, and on each build it hashes everything that
feeds an artifact — its configuration, its dependency files, its build args and
its target platforms — to decide whether the artifact can be reused. When the
hash matches a previous build whose image can still be found, the artifact is
reused; otherwise it is rebuilt.

Cache misses are reported with the reason they happened:

```text
Checking cache...
 - app: Not found. Building (no cached build for the current inputs)
 - api: Not found. Building (cached image is no longer available)
```

`no cached build for the current inputs` means something the artifact is built
from has changed since the last build, or that it has never been built.
`cached image is no longer available` means the inputs still match a previous
build, but the image it produced can no longer be found locally or in the
registry — it was pruned or evicted, and the rebuild simply recreates it.

### Finding out which input changed

To see *which* inputs changed rather than just that they did, run with
`--debug-cache`:

```bash
skaffold build --debug-cache
```

Skaffold then records the inputs to each artifact's hash next to the cache file
and compares them against the previous run, listing what differs:

```text
Checking cache...
 - app: Not found. Building (inputs changed)
     ~ src/handler.go
     + src/middleware.go
     - src/old_handler.go
```

`~` marks a changed input, `+` an added one and `-` a removed one. Dependency
files are listed relative to the artifact's context, so the same project built
from two different checkouts compares equal instead of reporting every file as
moved. Alongside them, the artifact's `config`, `build args` and `platforms` can
appear in this list. The first run with `--debug-cache` has nothing to compare
against, so it reports the ordinary reason instead.

Only digests of the inputs are recorded, never their values, so build args
holding credentials are not written to disk. The recorded snapshots live beside
the cache file, so `--cache-file` also moves them.
