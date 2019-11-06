---
title: "Local Build"
linkTitle: "Local"
weight: 10
featureId: build
---

Skaffold supports building locally with following builders.

1. [Docker](#dockerfile-with-docker)
2. [Jib](#jib-maven-and-gradle)
3. [Bazel](#bazel)

## Dockerfile with Docker

If you have [Docker](https://www.docker.com/products/docker-desktop)
installed, Skaffold can be configured to build artifacts with the local
Docker daemon.

By default, Skaffold connects to the local Docker daemon using
[Docker Engine APIs](https://docs.docker.com/develop/sdk/), though
it can also use the Docker
[command-line interface](https://docs.docker.com/engine/reference/commandline/cli/)
instead, which enables artifacts with [BuildKit](https://github.com/moby/buildkit).

After the artifacts are successfully built, Docker images will be pushed
to the remote registry. You can choose to skip this step.

**Configuration**

To use the local Docker daemon, add build type `local` to the `build` section
of `skaffold.yaml`. The following options can optionally be configured:

{{< schema root="LocalBuild" >}}

**Example**

The following `build` section instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with the local Docker daemon:

{{% readfile file="samples/builders/local.yaml" %}}

Which is equivalent to:

{{% readfile file="samples/builders/local-full.yaml" %}}


## Dockerfile in-cluster with Kaniko

[Kaniko](https://github.com/GoogleContainerTools/kaniko) is a Google-developed
open source tool for building images from a Dockerfile inside a container or
Kubernetes cluster. Kaniko enables building container images in environments
that cannot easily or securely run a Docker daemon.

Skaffold can help build artifacts in a Kubernetes cluster using the Kaniko
image; after the artifacts are built, kaniko must push them to a registry.

**Configuration**

To use Kaniko, add build type `kaniko` to the `build` section of
`skaffold.yaml`. The following options can optionally be configured:

{{< schema root="KanikoArtifact" >}}

The `buildContext` can be either:

{{< schema root="KanikoBuildContext" >}}

Since Kaniko builds images directly to a registry, it requires active cluster credentials.
These credentials are configured in the `cluster` section with the following options:

{{< schema root="ClusterDetails" >}}

To set up the credentials for Kaniko refer to the [kaniko docs](https://github.com/GoogleContainerTools/kaniko#kubernetes-secret).
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
Note that the Kubernetes secret must not be of type `kubernetes.io/dockerconfigjson` which stores the config json under the key `".dockerconfigjson"`, but an opaque secret with the key `"config.json"`.

**Example**

The following `build` section, instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with Kaniko:

{{% readfile file="samples/builders/kaniko.yaml" %}}

## Jib Maven and Gradle

[Jib](https://github.com/GoogleContainerTools/jib#jib) is a set of plugins for
[Maven](https://github.com/GoogleContainerTools/jib/blob/master/jib-maven-plugin) and
[Gradle](https://github.com/GoogleContainerTools/jib/blob/master/jib-gradle-plugin)
for building optimized OCI-compliant container images for Java applications
without a Docker daemon.

Skaffold can help build artifacts using Jib; Jib builds the container images and then
pushes them to the local Docker daemon or to remote registries as instructed by Skaffold.

Skaffold requires using Jib v1.4.0 or later.

**Configuration**

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

**Example**

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

{{< alert title="Updating from earlier versions" >}}
Skaffold had required Maven multi-module projects bind a Jib
`build` or `dockerBuild` goal to the **package** phase.  These bindings are
no longer required with Jib 1.4.0 and should be removed.
{{< /alert >}}

#### Gradle

To build a multi-module project with Gradle, first identify the sub-projects that should produce
a container image.  Then for each such sub-project:

  - Create a Skaffold `artifact` in the `skaffold.yaml`.
  - Set the `artifact`'s `context` field to the root project location.
  - Add a `jib` element and set its `project` field to the sub-project's name (the directory, by default).

## Bazel

[Bazel](https://bazel.build/) is a fast, scalable, multi-language, and
extensible build system.

Skaffold can help build artifacts using Bazel; after Bazel finishes building
container images, they will be loaded into the local Docker daemon.

**Configuration**

To use Bazel, `bazel` field to each artifact you specify in the
`artifacts` part of the `build` section, and use the build type `local`.
`context` should be a path containing the bazel files
(`WORKSPACE` and `BUILD`). The following options can optionally be configured:

{{< schema root="BazelArtifact" >}}

{{% alert title="Not any Bazel target can be used" %}}
The target specified must produce a bundle compatible
with docker load. See
<a href="https://github.com/bazelbuild/rules_docker#using-with-docker-locally">https://github.com/bazelbuild/rules_docker#using-with-docker-locally</a>
{{% /alert %}}


**Example**

The following `build` section instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with Bazel:

{{% readfile file="samples/builders/bazel.yaml" %}}

