
---
title: "Using Builders"
linkTitle: "Using Builders"
weight: 40
---

This page discusses how to set up Skaffold to use the tool of your choice
to build Docker images.

At this moment, Skaffold supports the following tools to build your image:

* Dockerfile locally with Docker
* Dockerfile remotely with Google Cloud Build
* Dockerfile in-cluster with Kaniko  
* Bazel locally with bazel and Docker daemon 
* JIB Maven and Gradle projects locally  

  
The `build` section in the Skaffold configuration file, `skaffold.yaml`,
controls how Skaffold builds artifacts. To use a specific tool for building
artifacts, add the value representing the tool and options for using the tool
to the `build` section. For a detailed discussion on Skaffold configuration,
see [Skaffold Concepts: Configuration](/docs/concepts/#configuration) and
[skaffold.yaml References](/docs/references/config).

## Dockerfile locally with Docker

If you have [Docker Desktop](https://www.docker.com/products/docker-desktop)
installed on your machine, you can configure Skaffold to build artifacts with
the local Docker daemon. 

By default, Skaffold connects to the local Docker daemon using
[Docker Engine APIs](https://docs.docker.com/develop/sdk/). You can, however,
ask Skaffold to use the [command-line interface](https://docs.docker.com/engine/reference/commandline/cli/)
instead. Additionally, Skaffold offers the option to build artifacts with
[BuildKit](https://github.com/moby/buildkit). After the artifacts are
successfully built, Skaffold will try pushing the Docker
images to the remote registry. You can choose to skip this step.

To use the local Docker daemon, add build type `local` to the `build` section
of `skaffold.yaml`. The `local` type offers the following options:

<table>
    <thead>
        <tr>
            <th>Option</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td><code>skipPush</code></td>
            <td>
                OPTIONAL
                <p>Skips pushing images.</p>
                <p>Default value is <code>false</code>.</p>
            </td>
        </tr>
        <tr>
            <td><code>useDockerCLI</code></td>
            <td>
                OPTIONAL
                <p>Uses Docker command-line interface instead of Docker Engine APIs.</p>
                <p>Default value is <code>false</code>.</p>
            </td>
        </tr>
        <tr>
            <td><code>useBuildkit</code></td>
            <td>
                OPTIONAL
                <p>Uses BuildKit to build Docker images.</p>
                <p>Default value is <code>false</code>.</p>
            </td>
        </tr>
    </tbody>
<table>

The following `build` section, for example, instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with the local Docker daemon: 

```
build:
    artifacts:
    - imageName: gcr.io/k8s-skaffold/example
    # Use local Docker daemon to build artifacts
    local:
        skipPush: false
        useDockerCLI: false
        useBuildkit: false
# The build section above is equal to
# build:
#   artifacts:
#   - imageName: gcr.io/k8s-skaffold/example
#   local: {}
```

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
section of `skaffold.yaml`. The `googleCloudBuild` type offers the following
options:

<table>
    <thead>
        <tr>
            <th>Option</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td><code>projectId</code></td>
            <td>
                <b>REQUIRED</b>
                <p>The ID of your Google Cloud Platform Project.</p>
            </td>
        </tr>
        <tr>
            <td><code>DiskSizeGb</code></td>
            <td>
                OPTIONAL
                <p>The disk size of the VM that runs the build. See <a href="https://cloud.google.com/cloud-build/docs/api/reference/rest/v1/projects.builds#buildoptions">Cloud Build API Reference: Build Options</a> for more information.</p>
            </td>
        </tr>
        <tr>
            <td><code>machineType</code></td>
            <td>
                OPTIONAL
                <p>The type of the VM that runs the build. See <a href="https://cloud.google.com/cloud-build/docs/api/reference/rest/v1/projects.builds#buildoptions">Cloud Build API Reference: Build Options</a> for more information.</p>
            </td>
        </tr>
        <tr>
            <td><code>timeOut</code></td>
            <td>
                OPTIONAL
                <p>The amount of time (in seconds) that this build should be allowed to run. See <a href="https://cloud.google.com/cloud-build/docs/api/reference/rest/v1/projects.builds#resource-build">Cloud Build API Reference: Resource/Build</a> for more information.</p>
            </td>
        </tr>
        <tr>
            <td><code>dockerImage</code></td>
            <td>
                OPTIONAL
                <p>The name of the image that will run the build. See <a href="https://cloud.google.com/cloud-build/docs/api/reference/rest/v1/projects.builds#buildstep">Cloud Build API Reference: BuildStep</a> for more information.</p>
                <p>Default value is `gcr.io/cloud-builders/docker`.</p>
            </td>
        </tr>
    </tbody>
<table>

The following `build` section, for example, instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with Google Cloud Build: 

```
build:
    artifacts:
    - imageName: gcr.io/k8s-skaffold/example
    # Use Google Cloud Build to build artifacts
    googleCloudBuild:
        projectId: YOUR-GCP-PROJECT
```

## Dockerfile in-cluster with Kaniko  

[Kaniko](https://github.com/GoogleContainerTools/kaniko) is a Google-developed
open-source tool for building images from a Dockerfile inside a container or
Kubernetes cluster. Kaniko enables building container images in environments
that cannot easily or securely run a Docker daemon.

Skaffold can help build artifacts in a Kubernetes cluster using the Kaninko
image; after the artifacts are built, kaniko can push them to remote registries.
To use Kaniko, add build type `kaniko` to the `build` section of
`skaffold.yaml`. The `kaniko` type offers the following options:

<table>
    <thead>
        <tr>
            <th>Option</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td><code>buildContext</code></td>
            <td>
                OPTIONAL
                <p>The Kaninko build context. See <a href="https://github.com/GoogleContainerTools/kaniko#using-kaniko">Kaniko Documentation: Using Kaniko</a> for more information.</p>
            </td>
        </tr>
        <tr>
            <td><code>pullSecret</code></td>
            <td>
                OPTIONAL
                <p>The path to the secret key file. See <a href="https://github.com/GoogleContainerTools/kaniko#running-kaniko-in-a-kubernetes-cluster">Kaniko Documentation: Running Kaniko in a Kubernetes cluster</a> for more information.</p>
            </td>
        </tr>
        <tr>
            <td><code>pullSecretName</code></td>
            <td>
                OPTIONAL
                <p>The name of the Kubernetes secret for pulling the files from the build context and pushing the final image. See <a href="https://github.com/GoogleContainerTools/kaniko#running-kaniko-in-a-kubernetes-cluster">Kaniko Documentation: Running Kaniko in a Kubernetes cluster</a> for more information.</p>
                <p>Default value is `kaniko-secret`.</p>
            </td>
        </tr>
        <tr>
            <td><code>namespace</code></td>
            <td>
                OPTIONAL
                <p>The Kubernetes namespace.</p>
                <p>Default value is the current namespace in Kubernetes configuration.</p>
            </td>
        </tr>
        <tr>
            <td><code>timeout</code></td>
            <td>
                OPTIONAL
                <p>The amount of time (in seconds) that this build should be allowed to run.</p>
                <p>Default value is 20 minutes (`20m`).</p>
            </td>
        </tr>
    </tbody>
<table>

The following `build` section, for example, instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with Kaniko: 

```
build:
    artifacts:
    - imageName: gcr.io/k8s-skaffold/example
    # Use Kaniko to build artifacts
    kaniko:
        buildContext: gs://YOUR-BUCKET/SOURCE-CODE.tar.gz
```

## Bazel locally with bazel and Docker daemon 

[Bazel](https://bazel.build/) is a fast, scalable, multi-language, and
extensible build system. 

Skaffold can help build artifacts using Bazel; after Bazel finishes building
container images, they will be loaded into the local Docker daemon. To use
Bazel, add `workspace` and `bazel` fields to each artifact you specify in the
`artifacts` part of the `build` section, and use the build type `local`.
`workspace` should be a path containing the bazel files
(`WORKSPACE` and `BUILD`); The `bazel` field should have a `target`
specification, which Skaffold will use to load the image to the Docker daemon.

The following `build` section, for example, instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with Bazel:

```
build:
    artifacts:
        - imageName: gcr.io/k8s-skaffold/example
          workspace: .
          bazel:
            target: //:example.tar
    local: {}
```
