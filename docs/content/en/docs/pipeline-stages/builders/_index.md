---
title: "Build"
linkTitle: "Build"
weight: 10
featureId: build
---

Skaffold has native support for several different tools for building images:

* [Dockerfile]({{< relref "/docs/pipeline-stages/builders/local#dockerfile-with-docker" >}})
  - locally with Docker
  - in-cluster with [Kaniko]({{< relref "/docs/pipeline-stages/builders/local#dockerfile-in-cluster-with-kaniko" >}})
  - on cloud with [Google Cloud Build]({{< relref "/docs/pipeline-stages/builders/remote" >}})
* [Jib]({{< relref "/docs/pipeline-stages/builders/local#jib-maven-and-gradle" >}}) Maven and Gradle
  - locally
  - on cloud with [Google Cloud Build]({{< relref "/docs/pipeline-stages/builders/remote" >}})
* [Bazel]({{< relref "/docs/pipeline-stages/builders/local#bazel" >}}) locally
* [Custom script locally]({{< relref "/docs/pipeline-stages/builders/custom" >}})
* CNCF Buildpacks [TODO #2904](https://github.com/GoogleContainerTools/skaffold/issues/2904)

The `build` section in the Skaffold configuration file, `skaffold.yaml`,
controls how artifacts are built. To use a specific tool for building
artifacts, add the value representing the tool and options for using that tool
to the `build` section.

For a detailed discussion on [Skaffold Configuration]({{< relref "/docs/design/config.md" >}}),
see [skaffold.yaml References]({{< relref "/docs/references/yaml" >}}).

Skaffold can perform builds [locally]({{< relref "/docs/pipeline-stages/builders/local">}})
or [remotely]({{< relref "/docs/pipeline-stages/builders/remote">}}).








