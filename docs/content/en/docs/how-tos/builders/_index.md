---
title: "Builders"
linkTitle: "Builders"
weight: 10
---

This page discusses how to set up Skaffold to use the tool of your choice
to build Docker images.

Skaffold supports the following tools to build your image:

* Dockerfile locally with Docker
* Dockerfile remotely with [Google Cloud Build](https://cloud.google.com/cloud-build/docs/)
* Dockerfile in-cluster with [Kaniko](https://github.com/GoogleContainerTools/kaniko)
* Bazel locally
* [Jib](https://github.com/GoogleContainerTools/jib) Maven and Gradle projects locally
* [Jib](https://github.com/GoogleContainerTools/jib) Maven and Gradle projects remotely with [Google Cloud Build](https://cloud.google.com/cloud-build/docs/)

The `build` section in the Skaffold configuration file, `skaffold.yaml`,
controls how Skaffold builds artifacts. To use a specific tool for building
artifacts, add the value representing the tool and options for using the tool
to the `build` section. For a detailed discussion on Skaffold configuration,
see [Skaffold Concepts: Configuration](/docs/concepts/#configuration) and
[skaffold.yaml References](https://github.com/GoogleContainerTools/skaffold/blob/master/examples/annotated-skaffold.yaml).

## Dockerfile locally with Docker

If you have [Docker Desktop](https://www.docker.com/products/docker-desktop)
installed on your machine, you can configure Skaffold to build artifacts with
the local Docker daemon. 

By default, Skaffold connects to the local Docker daemon using
[Docker Engine APIs](https://docs.docker.com/develop/sdk/). You can, however,
ask Skaffold to use the [command-line interface](https://docs.docker.com/engine/reference/commandline/cli/)
instead. Additionally, Skaffold offers the option to build artifacts with
[BuildKit](https://github.com/moby/buildkit). After the artifacts are
successfully built, Skaffold will push the Docker
images to the remote registry. You can choose to skip this step.

To use the local Docker daemon, add build type `local` to the `build` section
of `skaffold.yaml`.

The `local` type offers the following options:

| Option        | Description | Default |
|---------------|-------------|---------|
| `push`        | Should images be pushed to a registry | `false` for local clusters, `true` for remote clusters. |
| `useDockerCLI`| Uses `docker` command-line interface instead of Docker Engine APIs | `false` |
| `useBuildkit` | Uses BuildKit to build Docker images | `false` |

The following `build` section, for example, instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with the local Docker daemon: 

{{% readfile file="samples/builders/local.yaml" %}}

## Dockerfile remotely with Google Cloud Build

[Google Cloud Build](https://cloud.google.com/cloud-build/) is a
[Google Cloud Platform](https://cloud.google.com) service that executes
your builds using Google infrastructure. To get started with Google 
Build, see [Cloud Build Quickstart](https://cloud.google.com/cloud-build/docs/quickstart-docker).

Skaffold can automatically connect to Google Cloud Build, and run your builds
with it. After Google Cloud Build finishes building your artifacts, they will
be saved to the specified remote registry, such as
[Google Container Registry](https://cloud.google.com/container-registry/).

To use Google Cloud Build, add build type `googleCloudBuild` to the `build`
section of `skaffold.yaml`.

The `googleCloudBuild` type offers the following options:

| Option        | Description | Default |
|---------------|-------------|---------|
| `projectId`   | **Required** The ID of your Google Cloud Platform Project | |
| `diskSizeGb`  | The disk size of the VM that runs the build. See [Cloud Build API Reference: Build Options](https://cloud.google.com/cloud-build/docs/api/reference/rest/v1/projects.builds#buildoptions) for more information | |
| `machineType` | The type of the VM that runs the build. See [Cloud Build API Reference: Build Options](https://cloud.google.com/cloud-build/docs/api/reference/rest/v1/projects.builds#buildoptions) for more information | |
| `timeOut`     | The amount of time (in seconds) that this build should be allowed to run. See [Cloud Build API Reference: Resource/Build](https://cloud.google.com/cloud-build/docs/api/reference/rest/v1/projects.builds#resource-build) for more information. | |
| `dockerImage` | The name of the image that will run a docker build. See [Cloud builders](https://cloud.google.com/cloud-build/docs/cloud-builders) for more information | `gcr.io/cloud-builders/docker` |
| `gradleImage` | The name of the image that will run a gradle build. See [Cloud builders](https://cloud.google.com/cloud-build/docs/cloud-builders) for more information | `gcr.io/cloud-builders/gradle` |
| `mavenImage`  | The name of the image that will run a maven build. See [Cloud builders](https://cloud.google.com/cloud-build/docs/cloud-builders) for more information | `gcr.io/cloud-builders/mvn` |

The following `build` section, for example, instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with Google Cloud Build: 

{{% readfile file="samples/builders/gcb.yaml" %}}

## Dockerfile in-cluster with Kaniko  

[Kaniko](https://github.com/GoogleContainerTools/kaniko) is a Google-developed
open-source tool for building images from a Dockerfile inside a container or
Kubernetes cluster. Kaniko enables building container images in environments
that cannot easily or securely run a Docker daemon.

Skaffold can help build artifacts in a Kubernetes cluster using the Kaniko
image; after the artifacts are built, kaniko can push them to remote registries.
To use Kaniko, add build type `kaniko` to the `build` section of
`skaffold.yaml`.

The `kaniko` type offers the following options:

| Option          | Description | Default |
|-----------------|-------------|---------|
| `buildContext`  | The Kaniko build context: `gcsBucket` or `localDir` | `localDir` |
| `pullSecret`    | The path to the secret key file. See [Kaniko Documentation: Running Kaniko in a Kubernetes cluster](https://github.com/GoogleContainerTools/kaniko#running-kaniko-in-a-kubernetes-cluster) for more information | |
| `pullSecretName`| The name of the Kubernetes secret for pulling the files from the build context and pushing the final image | `kaniko-secret` |
| `namespace`     | The Kubernetes namespace | Current namespace in Kubernetes configuration |
| `timeout`       | The amount of time (in seconds) that this build should be allowed to run | 20 minutes (`20m`) |

The following `build` section, for example, instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with Kaniko: 

{{% readfile file="samples/builders/kaniko.yaml" %}}

## Jib Maven and Gradle locally 

{{% todo 1299 %}} 

## Jib Maven and Gradle remotely with Google Cloud Build 

{{% todo 1299 %}} 

## Bazel locally

[Bazel](https://bazel.build/) is a fast, scalable, multi-language, and
extensible build system. 

Skaffold can help build artifacts using Bazel; after Bazel finishes building
container images, they will be loaded into the local Docker daemon. To use
Bazel, `bazel` field to each artifact you specify in the
`artifacts` part of the `build` section, and use the build type `local`.
`context` should be a path containing the bazel files
(`WORKSPACE` and `BUILD`).

The `bazel` type offers the following options:

| Option    | Description |
|-----------|-------------|
| `target`  | **Required** The `bazel build` target to run |
| `args`    | Additional args to pass to `bazel build` |

The following `build` section, for example, instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with Bazel:

{{% readfile file="samples/builders/bazel.yaml" %}}
