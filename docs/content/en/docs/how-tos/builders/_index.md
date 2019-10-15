---
title: "Builders"
linkTitle: "Builders"
weight: 10
---

This page discusses how to set up Skaffold to use the tool of your choice
to build Docker images.

Skaffold supports the following tools to build your image:

* [Dockerfile](https://docs.docker.com/engine/reference/builder/)
  - locally with Docker
  - in-cluster with [Kaniko](https://github.com/GoogleContainerTools/kaniko)
  - on cloud with [Google Cloud Build](https://cloud.google.com/cloud-build/docs/)
* [Jib](https://github.com/GoogleContainerTools/jib) Maven and Gradle
  - locally
  - on cloud with [Google Cloud Build](https://cloud.google.com/cloud-build/docs/)
* [Bazel](https://bazel.build/) locally
* Custom script locally
* [Building with CNCF Buildpacks](../buildpacks)

The `build` section in the Skaffold configuration file, `skaffold.yaml`,
controls how artifacts are built. To use a specific tool for building
artifacts, add the value representing the tool and options for using that tool
to the `build` section.

For a detailed discussion on Skaffold configuration, see
[Skaffold Concepts]({{< relref "/docs/concepts#configuration" >}}) and
[skaffold.yaml References]({{< relref "/docs/references/yaml" >}}).

## Dockerfile locally with Docker

If you have [Docker Desktop](https://www.docker.com/products/docker-desktop)
installed, Skaffold can be configured to build artifacts with the local
Docker daemon.

By default, Skaffold connects to the local Docker daemon using
[Docker Engine APIs](https://docs.docker.com/develop/sdk/). Skaffold can, however,
be asked to use the [command-line interface](https://docs.docker.com/engine/reference/commandline/cli/)
instead. Additionally, Skaffold offers the option to build artifacts with
[BuildKit](https://github.com/moby/buildkit).

After the artifacts are successfully built, Docker images will be pushed
to the remote registry. You can choose to skip this step.

### Configuration

To use the local Docker daemon, add build type `local` to the `build` section
of `skaffold.yaml`. The following options can optionally be configured:

{{< schema root="LocalBuild" >}}

### Example

The following `build` section instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with the local Docker daemon:

{{% readfile file="samples/builders/local.yaml" %}}

Which is equivalent to:

{{% readfile file="samples/builders/local-full.yaml" %}}

## Dockerfile remotely with Google Cloud Build

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

### Configuration

To use Cloud Build, add build type `googleCloudBuild` to the `build`
section of `skaffold.yaml`. The following options can optionally be configured:

{{< schema root="GoogleCloudBuild" >}}

### Example

The following `build` section, instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with Google Cloud Build:

{{% readfile file="samples/builders/gcb.yaml" %}}

## Dockerfile in-cluster with Kaniko

[Kaniko](https://github.com/GoogleContainerTools/kaniko) is a Google-developed
open-source tool for building images from a Dockerfile inside a container or
Kubernetes cluster. Kaniko enables building container images in environments
that cannot easily or securely run a Docker daemon.

Skaffold can help build artifacts in a Kubernetes cluster using the Kaniko
image; after the artifacts are built, kaniko must push them to a registry.

### Configuration

To use Kaniko, add build type `kaniko` to the `build` section of
`skaffold.yaml`. The following options can optionally be configured:

{{< schema root="KanikoArtifact" >}}

The `buildContext` can be either:

{{< schema root="KanikoBuildContext" >}}

Since Kaniko must push images to a registry, it is required to set up cluster credentials.
These credentials are configured in the `cluster` section with the following options:

{{< schema root="ClusterDetails" >}}

To set up the credentials for kaniko have a look at the [kaniko docs](https://github.com/GoogleContainerTools/kaniko#kubernetes-secret).
The recommended way is to store the pull secret in Kubernetes and configure `pullSecretName`.
Alternatively, the path to a credentials file can be set with the `pullSecret` option:
```yaml
build:
  cluster:
    pullSecretName: pull-secret-in-kubernetes
    # OR
    pullSecret: path-to-service-account-key-file
```
Similarly, when pushing to a docker registry:
```yaml
build:
  cluster:
    dockerConfig:
      path: ~/.docker/config.json
      # OR
      secretName: docker-config-secret-in-kubernetes
```
Note that the kubernetes secret must not be of type `kubernetes.io/dockerconfigjson` which stores the config json under the key `".dockerconfigjson"`, but an opaque secret with the key `"config.json"`.

### Example

The following `build` section, instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with Kaniko:

{{% readfile file="samples/builders/kaniko.yaml" %}}

## Jib Maven and Gradle locally

[Jib](https://github.com/GoogleContainerTools/jib#jib) is a set of plugins for
[Maven](https://github.com/GoogleContainerTools/jib/blob/master/jib-maven-plugin) and
[Gradle](https://github.com/GoogleContainerTools/jib/blob/master/jib-gradle-plugin)
for building optimized Docker and OCI images for Java applications
without a Docker daemon.

Skaffold can help build artifacts using Jib; Jib builds the container images and then
pushes them to the local Docker daemon or to remote registries as instructed by Skaffold.

Skaffold requires using Jib v1.4.0 or later.

### Configuration

To use Jib, add a `jib` field to each artifact you specify in the
`artifacts` part of the `build` section. `context` should be a path to
your Maven or Gradle project.  

{{< alert title="Note" >}}
Your project must be configured to use Jib already.
{{< /alert >}}

The `jib` type offers the following options:

{{< schema root="JibArtifact" >}}

Skaffold's jib support chooses the underlying builder (Maven or Gradle) 
based on the presence of standard build files in the `artifact`'s
`context` directory:

  - _Maven_: `pom.xml`, or `.mvn` directory.
  - _Gradle_: `build.gradle`, `gradle.properties`, `settings.gradle`,
    or the Gradle wrapper script (`gradlew`, `gradlew.bat`, or
    `gradlew.cmd`).

### Example

See the [Skaffold-Jib demo project](https://github.com/GoogleContainerTools/skaffold/blob/master/examples/jib/)
for an example.

### Multi-Module Projects

Skaffold can be configured for _multi-module projects_ too. A multi-module project
has several _modules_ (Maven terminology) or _sub-projects_ (Gradle terminology) that
each produce a separate container image.

#### Maven

To build a Maven multi-module project, first identify the sub-projects (also called _modules_
in Maven) that should produce a container image. Then for each such sub-project:

  - Create a Skaffold `artifact` in the `skaffold.yaml`.
  - Set the `artifact`'s `context` field to the root project location.
  - Add a `jib` element and set its `project` field to the sub-project's
    `:artifactId`, `groupId:artifactId`, or the relative path to the sub-project
    _within the project_.

{{% alert title="Updating from earlier versions" %}}
Skaffold had required Maven multi-module projects bind a Jib
`build` or `dockerBuild` goal to the *package* phase.  These bindings are
no longer required with Jib 1.4.0 and should be removed.
{{% /alert %}}

#### Gradle

To build a multi-module project with Gradle, first identify the sub-projects that should produce
a container image.  Then for each such sub-project:

  - Create a Skaffold `artifact` in the `skaffold.yaml`.
  - Set the `artifact`'s `context` field to the root project location.
  - Add a `jib` element and set its `project` field to the sub-project's name (the directory, by default).

## Jib Maven and Gradle remotely with Google Cloud Build

{{% todo 1299 %}}

## Bazel locally

[Bazel](https://bazel.build/) is a fast, scalable, multi-language, and
extensible build system.

Skaffold can help build artifacts using Bazel; after Bazel finishes building
container images, they will be loaded into the local Docker daemon.

### Configuration

To use Bazel, `bazel` field to each artifact you specify in the
`artifacts` part of the `build` section, and use the build type `local`.
`context` should be a path containing the bazel files
(`WORKSPACE` and `BUILD`). The following options can optionally be configured:

{{< schema root="BazelArtifact" >}}

### Example

The following `build` section instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with Bazel:

{{% readfile file="samples/builders/bazel.yaml" %}}

## Custom Build Script Run Locally

Custom build scripts allow skaffold users the flexibility to build artifacts with any builder they desire. 
Users can write a custom build script which must abide by the following contract for skaffold to work as expected:

### Contract between Skaffold and Custom Build Script

Skaffold will pass in the following environment variables to the custom build script:

| Environment Variable         | Description           | Expectation  |
| ------------- |-------------| -----|
| $IMAGES     | An array of fully qualified image names, separated by spaces. For example, "gcr.io/image1 gcr.io/image2" | The custom build script is expected to build an image and tag it with each image name in $IMAGES. Each image should also be pushed if `$PUSH_IMAGE=true`. | 
| $PUSH_IMAGE      | Set to true if each image in `$IMAGES` is expected to exist in a remote registry. Set to false if each image in `$IMAGES` is expected to exist locally.      |   The custom build script will push each image in `$IMAGES` if `$PUSH_IMAGE=true` | 
| $BUILD_CONTEXT  | An absolute path to the directory this artifact is meant to be built from. Specified by artifact `context` in the skaffold.yaml.      | None. | 
| Local environment variables | The current state of the local environment (e.g. `$HOST`, `$PATH)`. Determined by the golang [os.Environ](https://golang.org/pkg/os#Environ) function.| None. |

As described above, the custom build script is expected to:

1. Build and tag each image in `$IMAGES`
2. Push each image in `$IMAGES` if `$PUSH_IMAGE=true`

Once the build script has finished executing, skaffold will try to obtain the digest of the newly built image from a remote registry (if `$PUSH_IMAGE=true`) or the local daemon (if `$PUSH_IMAGE=false`).
If skaffold fails to obtain the digest, it will error out.

#### Additional Environment Variables

Skaffold will pass in the following additional environment variables for the following builders:

##### Local builder
| Environment Variable         | Description           | Expectation  |
| ------------- |-------------| -----|
| Docker daemon environment variables     | Inform the custom builder of which docker daemon endpoint we are using. Allows custom build scripts to work with tools like Minikube. For Minikube, this is the output of `minikube docker-env`.| None. | 

##### Cluster Builder
| Environment Variable         | Description           | Expectation  |
| ------------- |-------------| -----|
| $KUBECONTEXT    | The expected kubecontext in which the image will be built.| None. | 
| $NAMESPACE      | The expected namespace in which the image will be built.| None. | 
| $PULL_SECRET_NAME    | The name of the secret with authentication required to pull a base image/push the final image built on cluster.| None. | 
| $DOCKER_CONFIG_SECRET_NAME    | The secret containing any required docker authentication for custom builds on cluster.| None. | 
| $TIMEOUT        | The amount of time an on cluster build is allowed to run.| None. | 

### Configuration

To use a custom build script, add a `custom` field to each corresponding artifact in the `build` section of the skaffold.yaml.
Currently, this only works with the `local` and `cluster` build types. Supported schema for `custom` includes:

{{< schema root="CustomArtifact" >}}

`buildCommand` is *required* and points skaffold to the custom build script which will be executed to build the artifact.

#### Dependencies for a Custom Artifact

`dependencies` tells the skaffold file watcher which files should be watched to trigger rebuilds and file syncs.  Supported schema for `dependencies` includes:

{{< schema root="CustomDependencies" >}}

##### Paths and Ignore

`Paths` and `Ignore` are arrays used to list dependencies. 
Any paths in `Ignore` will be ignored by the skaffold file watcher, even if they are also specified in `Paths`.
`Ignore` will only work in conjunction with `Paths`, and with none of the other custom artifact dependency types.

```yaml
custom:
  buildCommand: ./build.sh
  dependencies:
    paths:
    - pkg/**
    - src/*.go
    ignore:
    - vendor/**
```

##### Dockerfile

Skaffold can calculate dependencies from a Dockerfile for a custom artifact.
Passing in the path to the Dockerfile and any build args, if necessary, will allow skaffold to do dependency calculation.

{{< schema root="DockerfileDependency" >}}

```yaml
custom:
  buildCommand: ./build.sh
  dependencies:
    dockerfile:
      path: path/to/Dockerfile
      buildArgs:
        file: foo
```

##### Getting dependencies from a command

Sometimes you might have a builder that can provide the dependencies for a given artifact.
For example bazel has the `bazel query deps` command.
Custom artifact builders can ask Skaffold to execute a custom command, which Skaffold can use to get the dependencies for the artifact for file watching.

The command *must* return dependencies as a JSON array, otherwise skaffold will error out.

For example, the following configuration is valid, as executing the dependency command returns a valid JSON array.

```yaml
custom:
  buildCommand: ./build.sh
  dependencies:
    command: echo ["file1","file2","file3"]
```

#### Custom Build Scripts and File Sync

Syncable files must be included in both the `paths` section of `dependencies`, so that the skaffold file watcher knows to watch them, and the `sync` section, so that skaffold knows to sync them.  

#### Custom Build Scripts and Logging

`STDOUT` and `STDERR` from the custom build script will be redirected and displayed within skaffold logs.


### Example

The following `build` section instructs Skaffold to build an image `gcr.io/k8s-skaffold/example` with a custom build script `build.sh`:

{{% readfile file="samples/builders/custom.yaml" %}}

A sample `build.sh` file, which builds an image with bazel and docker:

{{% readfile file="samples/builders/build.sh" %}}
