---
title: "Custom Build Script"
linkTitle: "Custom"
weight: 40
featureId: build.custom
---

Custom build scripts allow Skaffold users the flexibility to build artifacts with any builder they desire. 
Users can write a custom build script which must abide by the following contract for Skaffold to work as expected:

### Contract between Skaffold and Custom Build Script

Skaffold will pass in the following additional environment variables to the custom build script:

| Environment Variable         | Description           | Expectation  |
| ------------- |-------------| -----|
| $IMAGE     | The fully qualified image name. For example, "gcr.io/image1:tag" | The custom build script is expected to build this image and tag it with the name provided in $IMAGE. The image should also be pushed if `$PUSH_IMAGE=true`. | 
| $PUSH_IMAGE      | Set to true if the image in `$IMAGE` is expected to exist in a remote registry. Set to false if the image is expected to exist locally.      |   The custom build script will push the image `$IMAGE` if `$PUSH_IMAGE=true` | 
| $BUILD_CONTEXT  | An absolute path to the directory this artifact is meant to be built from. Specified by artifact `context` in the skaffold.yaml.      | None. | 
| Local environment variables | The current state of the local environment (e.g. `$HOST`, `$PATH)`. Determined by the golang [os.Environ](https://golang.org/pkg/os#Environ) function.| None. |

As described above, the custom build script is expected to:

1. Build and tag the `$IMAGE` image
2. Push the image if `$PUSH_IMAGE=true`

Once the build script has finished executing, Skaffold will try to obtain the digest of the newly built image from a remote registry (if `$PUSH_IMAGE=true`) or the local daemon (if `$PUSH_IMAGE=false`).
If Skaffold fails to obtain the digest, it will error out.

### Configuration

To use a custom build script, add a `custom` field to each corresponding artifact in the `build` section of the `skaffold.yaml`.
Supported schema for `custom` includes:

{{< schema root="CustomArtifact" >}}

`buildCommand` is *required* and points Skaffold to the custom build script which will be executed to build the artifact.
The [Go templates](https://golang.org/pkg/text/template/) syntax can be used to inject environment variables into the build
command. For example: `buildCommand: ./build.sh --flag={{ .SOME_FLAG }}` will replace `{{ .SOME_FLAG }}` with the value of
the `SOME_FLAG` environment variable.

#### Custom Build Script Locally

In addition to these [environment variables](#contract-between-skaffold-and-custom-build-script)
Skaffold will pass in the following additional environment variables for local builder:

| Environment Variable         | Description           | Expectation  |
| ------------- |-------------| -----|
| Docker daemon environment variables     | Inform the custom builder of which docker daemon endpoint we are using. Allows custom build scripts to work with tools like Minikube. For Minikube, this is the output of `minikube docker-env`.| None. | 

**Configuration**

To configure custom build script locally, in addition to adding a [`custom` field](#configuration) to each corresponding artifact in the `build`
add `local` to you `build` config.

#### Custom Build Script in Cluster

In addition to these [environment variables](#contract-between-skaffold-and-custom-build-script)
Skaffold will pass in the following additional environment variables for cluster builder:

| Environment Variable         | Description           | Expectation  |
| ------------- |-------------| -----|
| $KUBECONTEXT    | The expected kubecontext in which the image will be built.| None. | 
| $NAMESPACE      | The expected namespace in which the image will be built.| None. | 
| $PULL_SECRET_NAME    | The name of the secret with authentication required to pull a base image/push the final image built on cluster.| None. | 
| $DOCKER_CONFIG_SECRET_NAME    | The secret containing any required docker authentication for custom builds on cluster.| None. | 
| $TIMEOUT        | The amount of time an on cluster build is allowed to run.| None. | 

**Configuration**

To configure custom build script locally, in addition to adding a [`custom` field](#configuration) to each corresponding artifact in the `build`, add `cluster` to you `build` config.

#### Custom Build Script on Google Cloud Build

This configuration is currently not supported.

### Dependencies for a Custom Artifact

`dependencies` tells the skaffold file watcher which files should be watched to trigger rebuilds and file syncs.  Supported schema for `dependencies` includes:

{{< schema root="CustomDependencies" >}}

#### Paths and Ignore

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

#### Dockerfile

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

{{< alert title="Warning" >}}
`buildArgs` are not passed to the custom build script. They are only used to resolve
values of `ARG` instructions in the the given Dockerfile when listing the dependencies.
{{< /alert >}}

#### Dependencies from a command

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

### File Sync

Syncable files must be included in both the `paths` section of `dependencies`, so that the skaffold file watcher knows to watch them, and the `sync` section, so that skaffold knows to sync them.  

### Logging

`STDOUT` and `STDERR` from the custom build script will be redirected and displayed within skaffold logs.


**Example**

The following `build` section instructs Skaffold to build an image `gcr.io/k8s-skaffold/example` with a custom build script `build.sh`:

{{% readfile file="samples/builders/custom.yaml" %}}

A sample `build.sh` file, which builds an image with bazel and docker:

{{% readfile file="samples/builders/build.sh" %}}
