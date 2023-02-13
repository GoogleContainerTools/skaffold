---
title: "Cross-platform and multi-platform build support"
linkTitle: "Cross/multi-platform"
weight: 40
---

Skaffold selectively supports building for an architecture that is different than the development machine architecture (`cross-platform` build) or building for multiple architectures (`multiple-platform` build). The target platforms for an artifact can be specified in one of the following ways:

- The pipeline's `platforms` property in the `skaffold.yaml` file.
{{% readfile file="samples/builders/platforms/pipeline-constraints.yaml" %}}

- The artifact's `platforms` constraints in the `skaffold.yaml` file. This overrides the value specified in the pipeline's `platforms` property.
{{% readfile file="samples/builders/platforms/artifact-constraints.yaml" %}}

- The CLI flag `--platform` which overrides the values set in both the previous ways.

```cmd
skaffold build --platform=linux/arm64,linux/amd64
```

Additionally, for `skaffold dev`, `skaffold debug` and `skaffold run` commands, where the build output gets deployed immediately, skaffold checks the platform for the kubernetes cluster nodes and attempts to build artifacts for that target platform.

The final list of target platforms need to ultimately be supported by the target builder, otherwise it'll fail the build. The cross-platform build support for the various builders can be summarized in the following table:

|    | Local Build | In Cluster Build | Remote on Google Cloud Build |
|----|:-----------:|:----------------:|:----------------------------:|
| **Dockerfile** | Cross-platform and multi-platform supported | Cross-platform supported but platform should match cluster node running the pod. | Cross-platform and multi-platform supported |
| **Jib Maven and Gradle** | Cross-platform and multi-platform supported | - | Cross-platform and multi-platform supported |
| **Cloud Native Buildpacks** | Only supports `linux/amd64` | - | Only supports `linux/amd64` |
| **Bazel** | Cross-platform supported but requires explicit platform specific rules. Not yet implemented | - | - |
| **ko** | Cross-platform and multi-platform supported | - | Cross-platform and multi-platform supported |
| **Custom Script** | Cross-platform and multi-platform supported but requires user to implement it in the build script | Cross-platform and multi-platform supported but requires user to implement it in the build script | - |

{{< alert title="Note" >}}
Skaffold supports multi-platform image builds natively for the [jib builder]({{<relref "/docs/pipeline-stages/builders/builder-types/jib" >}}), the [ko builder]({{<relref "/docs/pipeline-stages/builders/builder-types/ko">}}) and the [custom builder]({{<relref "/docs/pipeline-stages/builders/builder-types/custom" >}}). For other builders that support building cross-architecture images, Skaffold will iteratively build a single platform image for each target architecture and stitch them together into a multi-platform image, and push it to the registry.
{{< /alert >}}
