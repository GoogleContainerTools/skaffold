---
title: "Build"
linkTitle: "Build"
weight: 10
featureId: build
aliases: [/docs/how-tos/builders]
---

Skaffold supports different tools for building images:

|    | Local Build | In Cluster Build | Remote on Google Cloud Build |
|----|:-----------:|:----------------:|:----------------------------:|
| **Dockerfile** | [Yes]({{< relref "/docs/pipeline-stages/builders/docker#dockerfile-locally" >}}) | [Yes]({{< relref "/docs/pipeline-stages/builders/docker#dockerfile-in-cluster-with-kaniko" >}}) | [Yes]({{< relref "/docs/pipeline-stages/builders/docker#dockerfile-remotely-with-google-cloud-build" >}}) |
| **Jib Maven and Gradle** | [Yes]({{< relref "/docs/pipeline-stages/builders/jib#jib-maven-and-gradle-locally" >}}) | - | [Yes]({{< relref "/docs/pipeline-stages/builders/jib#remotely-with-google-cloud-build" >}}) |
| **Cloud Native Buildpacks** | [Yes]({{< relref "/docs/pipeline-stages/builders/buildpacks" >}}) | - | [Yes]({{< relref "/docs/pipeline-stages/builders/buildpacks" >}}) |
| **Bazel** | [Yes]({{< relref "/docs/pipeline-stages/builders/bazel" >}}) | - | - |
| **Custom Script** | [Yes]({{<relref "/docs/pipeline-stages/builders/custom#custom-build-script-locally" >}}) | [Yes]({{<relref "/docs/pipeline-stages/builders/custom#custom-build-script-in-cluster" >}}) | - |

**Configuration**

The `build` section in the Skaffold configuration file, `skaffold.yaml`,
controls how artifacts are built. To use a specific tool for building
artifacts, add the value representing the tool and options for using that tool
to the `build` section.

For a detailed discussion on [Skaffold Configuration]({{< relref "/docs/design/config.md" >}}),
see [skaffold.yaml References]({{< relref "/docs/references/yaml" >}}).

## Local Build
Local build execution is the default execution context.
Skaffold will use the build tools locally installed on your machine to execute the build.

**Configuration**

To configure the local execution explicitly, add build type `local` to the build section of `skaffold.yaml`

```yaml
build:
  local:
    ...
```

{{< schema root="LocalBuild" >}}

If you are deploying to [local cluster]({{<relref "/docs/environment/local-cluster" >}}), you can additional set `push` to `false` to speed up builds.


## In Cluster Build
Skaffold supports building in cluster via [Kaniko]({{< relref "/docs/pipeline-stages/builders/docker#dockerfile-in-cluster-with-kaniko" >}}) 
or [Custom Build Script]({{<relref "/docs/pipeline-stages/builders/custom#custom-build-script-in-cluster" >}}).

**Configuration**

To configure in-cluster Build, add build type `cluster` to the build section of `skaffold.yaml`. 

```yaml
build:
  cluster:
    ...
```

The following options can optionally be configured:

{{< schema root="ClusterDetails" >}}

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
  googleCloudBuild:
    ...
```

The following options can optionally be configured:

{{< schema root="GoogleCloudBuild" >}}
