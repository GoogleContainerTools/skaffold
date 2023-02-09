---
title: "Build"
linkTitle: "Build"
weight: 10
featureId: build
aliases: [/docs/how-tos/builders]
no_list: true
---

Skaffold supports different [tools]({{< relref "/docs/pipeline-stages/builders/builder-types" >}}) for building images across different [build environments]({{< relref "/docs/pipeline-stages/builders/build-environments" >}}).

|    | Local Build | In Cluster Build | Remote on Google Cloud Build |
|----|:-----------:|:----------------:|:----------------------------:|
| **Dockerfile** | [Yes]({{< relref "/docs/pipeline-stages/builders/builder-types/docker#dockerfile-locally" >}}) | [Yes]({{< relref "/docs/pipeline-stages/builders/builder-types/docker#dockerfile-in-cluster-with-kaniko" >}}) | [Yes]({{< relref "/docs/pipeline-stages/builders/builder-types/docker#dockerfile-remotely-with-google-cloud-build" >}}) |
| **Jib Maven and Gradle** | [Yes]({{< relref "/docs/pipeline-stages/builders/builder-types/jib#jib-maven-and-gradle-locally" >}}) | - | [Yes]({{< relref "/docs/pipeline-stages/builders/builder-types/jib#remotely-with-google-cloud-build" >}}) |
| **Cloud Native Buildpacks** | [Yes]({{< relref "/docs/pipeline-stages/builders/builder-types/buildpacks" >}}) | - | [Yes]({{< relref "/docs/pipeline-stages/builders/builder-types/buildpacks" >}}) |
| **Bazel** | [Yes]({{< relref "/docs/pipeline-stages/builders/builder-types/bazel" >}}) | - | - |
| **ko** | [Yes]({{< relref "/docs/pipeline-stages/builders/builder-types/ko" >}}) | - | [Yes]({{< relref "/docs/pipeline-stages/builders/builder-types/ko#remote-builds" >}}) |
| **Custom Script** | [Yes]({{<relref "/docs/pipeline-stages/builders/builder-types/custom#custom-build-script-locally" >}}) | [Yes]({{<relref "/docs/pipeline-stages/builders/builder-types/custom#custom-build-script-in-cluster" >}}) | - |

## Configuration

The `build` section in the Skaffold configuration file, `skaffold.yaml`,
controls how artifacts are built. To use a specific tool for building
artifacts, add the value representing the tool and options for using that tool
to the `build` section.

For detailed per-builder [Skaffold Configuration]({{< relref "/docs/design/config.md" >}}) options,
see [skaffold.yaml References]({{< relref "/docs/references/yaml" >}}).
