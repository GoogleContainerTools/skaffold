---
title: "Bazel"
linkTitle: "Bazel"
weight: 30
featureId: build
aliases: [/docs/pipeline-stages/builders/bazel]
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
The target specified must produce a .tar bundle compatible
with docker load. See
<a href="https://github.com/bazelbuild/rules_docker#using-with-docker-locally">https://github.com/bazelbuild/rules_docker#using-with-docker-locally</a>
{{% /alert %}}


**Example**

The following `build` section instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with Bazel:

{{% readfile file="samples/builders/bazel.yaml" %}}

The following `build` section shows how to use Skaffold's
cross-platform support to pass `--platforms` to Bazel. In this
example, the Bazel project defines the `//platforms:linux-x86_64`
`//platforms:linux-arm64` targets. Skaffold will pass `--platforms=//platforms:linux-x86_64` to `bazel build`
if its target build platform matches `linux/amd64`, `--platforms=//platforms:linux-arm64`
if its target build platform matches `linux/arm64`, and will not pass `--platforms` otherwise.

{{% readfile file="samples/builders/bazel-xplat.yaml" %}}

Note that Skaffold does not support intelligently selecting the most specific
variant for platforms with variants. For example, specifying `linux/arm64`
and `linux/arm64/v8` will not work. In this example it would be better to
specify `linux/arm64/v7` and `linux/arm64/v8` instead.