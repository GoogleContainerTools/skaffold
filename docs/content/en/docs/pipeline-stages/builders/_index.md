---
title: "Build"
linkTitle: "Build"
weight: 10
featureId: build
aliases: [/docs/how-tos/builders]
no_list: true
---

Skaffold supports different tools for building images:

|    | Local Build | In Cluster Build | Remote on Google Cloud Build |
|----|:-----------:|:----------------:|:----------------------------:|
| **Dockerfile** | [Yes]({{< relref "/docs/pipeline-stages/builders/docker#dockerfile-locally" >}}) | [Yes]({{< relref "/docs/pipeline-stages/builders/docker#dockerfile-in-cluster-with-kaniko" >}}) | [Yes]({{< relref "/docs/pipeline-stages/builders/docker#dockerfile-remotely-with-google-cloud-build" >}}) |
| **Jib Maven and Gradle** | [Yes]({{< relref "/docs/pipeline-stages/builders/jib#jib-maven-and-gradle-locally" >}}) | - | [Yes]({{< relref "/docs/pipeline-stages/builders/jib#remotely-with-google-cloud-build" >}}) |
| **Cloud Native Buildpacks** | [Yes]({{< relref "/docs/pipeline-stages/builders/buildpacks" >}}) | - | [Yes]({{< relref "/docs/pipeline-stages/builders/buildpacks" >}}) |
| **Bazel** | [Yes]({{< relref "/docs/pipeline-stages/builders/bazel" >}}) | - | - |
| **ko** | [Yes]({{< relref "/docs/pipeline-stages/builders/ko" >}}) | - | - |
| **Custom Script** | [Yes]({{<relref "/docs/pipeline-stages/builders/custom#custom-build-script-locally" >}}) | [Yes]({{<relref "/docs/pipeline-stages/builders/custom#custom-build-script-in-cluster" >}}) | - |

**Configuration**

The `build` section in the Skaffold configuration file, `skaffold.yaml`,
controls how artifacts are built. To use a specific tool for building
artifacts, add the value representing the tool and options for using that tool
to the `build` section.

For detailed per-builder [Skaffold Configuration]({{< relref "/docs/design/config.md" >}}) options,
see [skaffold.yaml References]({{< relref "/docs/references/yaml" >}}).

## Local Build
Local build execution is the default execution context.
Skaffold will use your locally-installed build tools (such as Docker, Bazel, Maven or Gradle) to execute the build.

**Configuration**

To configure the local execution explicitly, add build type `local` to the build section of `skaffold.yaml`

```yaml
build:
  local: {}
```

The following options can optionally be configured:

{{< schema root="LocalBuild" >}}

### Faster builds

There are a few options for achieving faster local builds.

#### Avoiding pushes

When deploying to a [local cluster]({{<relref "/docs/environment/local-cluster" >}}), 
Skaffold defaults `push` to `false` to speed up builds.  The `push`
setting can be set from the command-line with `--push`.

#### Parallel builds

The `concurrency` controls the number of image builds that are run in parallel.
Skaffold disables concurrency by default for local builds as several
image builder types (`custom`, `jib`) may change files on disk and
result in side-effects.
`concurrency` can be set to `0` to enable full parallelism, though
this may consume significant resources.
The concurrency setting can be set from the command-line with the
`--build-concurrency` flag.

When artifacts are built in parallel, the build logs are still printed in sequence to make them easier to read.

#### Build avoidance with `tryImportMissing`

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


## In Cluster Build

Skaffold supports building in cluster via [Kaniko]({{< relref "/docs/pipeline-stages/builders/docker#dockerfile-in-cluster-with-kaniko" >}}) 
or [Custom Build Script]({{<relref "/docs/pipeline-stages/builders/custom#custom-build-script-in-cluster" >}}).

**Configuration**

To configure in-cluster Build, add build type `cluster` to the build section of `skaffold.yaml`. 

```yaml
build:
  cluster: {}
```

The following options can optionally be configured:

{{< schema root="ClusterDetails" >}}

**Faster builds**

Skaffold can build multiple artifacts in parallel, by settings a value higher than `1` to `concurrency`.
For in-cluster builds, the default is to build all the artifacts in parallel. If your cluster is too
small, you might want to reduce the `concurrency`. Setting `concurrency` to `1` will cause artifacts to be built sequentially.

{{<alert title="Note">}}
When artifacts are built in parallel, the build logs are still printed in sequence to make them easier to read.
{{</alert>}}

## Remotely on Google Cloud Build

Skaffold supports building remotely with Google Cloud Build.

[Google Cloud Build](https://cloud.google.com/cloud-build/) is a
[Google Cloud Platform](https://cloud.google.com) service that executes
your builds using Google infrastructure. To get started with Google
Build, see [Cloud Build Quickstart](https://cloud.google.com/cloud-build/docs/quickstart-docker).

Skaffold can automatically connect to Cloud Build, and run your builds
with it. After Cloud Build finishes building your artifacts, they will
be saved to the specified remote registry, such as
[Google Container Registry](https://cloud.google.com/container-registry/).

Skaffold Google Cloud Build process differs from the gcloud command
`gcloud builds submit`. Skaffold will create a list of dependent files
and submit a tar file to GCB. It will then generate a single step `cloudbuild.yaml`
and will start the building process. Skaffold does not honor `.gitignore` or `.gcloudignore`
exclusions. If you need to ignore files use `.dockerignore`. Any `cloudbuild.yaml` found will not
be used in the build process.

**Configuration**

To use Cloud Build, add build type `googleCloudBuild` to the `build`
section of `skaffold.yaml`. 

```yaml
build:
  googleCloudBuild: {}
```

The following options can optionally be configured:

{{< schema root="GoogleCloudBuild" >}}

**Faster builds**

Skaffold can build multiple artifacts in parallel, by settings a value higher than `1` to `concurrency`.
For Google Cloud Build, the default is to build all the artifacts in parallel. If you hit a quota restriction,
you might want to reduce  the `concurrency`.

{{<alert title="Note">}}
When artifacts are built in parallel, the build logs are still printed in sequence to make them easier to read.
{{</alert>}}

**Restrictions**

Skaffold currently supports [Docker]({{<relref "/docs/pipeline-stages/builders/docker#dockerfile-remotely-with-google-cloud-build">}}),
[Jib]({{<relref "/docs/pipeline-stages/builders/jib#remotely-with-google-cloud-build">}})
on Google Cloud Build.

## Cross-platform build support

Skaffold selectively supports building for a platform that is different than the host machine platform. The target platform for an artifact can be specified in one of the following ways:

- The pipeline's `platforms` property in the `skaffold.yaml` file.
{{% readfile file="samples/builders/platforms/pipeline-constraints.yaml" %}}

- The artifact's `platforms` constraints in the `skaffold.yaml` file. This overrides the value specified in the pipeline's `platforms` property.
{{% readfile file="samples/builders/platforms/artifact-constraints.yaml" %}}

- The CLI flag `--platform` which overrides the values set in both the previous ways.

```cmd
skaffold build --platform=linux/arm64
```

Additionally, for `skaffold dev`, `skaffold debug` and `skaffold run` commands, where the build output gets deployed immediately, skaffold checks the platform for the kubernetes cluster nodes and attempts to build artifacts for that target platform.

The final list of target platforms need to ultimately be supported by the target builder, otherwise it'll fail the build. The cross-platform build support for the various builders can be summarized in the following table:

|    | Local Build | In Cluster Build | Remote on Google Cloud Build |
|----|:-----------:|:----------------:|:----------------------------:|
| **Dockerfile** | Cross platform supported | Cross platform supported but platform should match cluster node running the pod. Not yet implemented | Can support. Not yet implemented |
| **Jib Maven and Gradle** | Cross platform supported | - | Can support. Not yet implemented |
| **Cloud Native Buildpacks** | Only supports `linux/amd64` | - | Only supports `linux/amd64` |
| **Bazel** | Cross platform supported but requires explicit platform specific rules. Not yet implemented | - | - |
| **ko** | Cross platform supported | - | - |
| **Custom Script** | Cross platform supported but requires user to implement it in the build script | Can support. Not yet implemented | - |

{{< alert title="Note" >}}
Multi-arch image build is not yet supported for any builders other than the [jib builder]({{<relref "/docs/pipeline-stages/builders/jib" >}}) and [custom builder]({{<relref "/docs/pipeline-stages/builders/custom" >}}) in Skaffold 
{{< /alert >}}