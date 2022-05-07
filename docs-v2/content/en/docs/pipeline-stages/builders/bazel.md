---
title: "Bazel"
linkTitle: "Bazel"
weight: 30
featureId: build
---

[Bazel](https://bazel.build/) is a fast, scalable, multi-language, and
extensible build system.

Skaffold can help build artifacts using Bazel locally; after Bazel finishes building
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