---
title: "Skaffold Pipeline"
linkTitle: "Skaffold Pipeline"
weight: 40
aliases: [/docs/concepts/config]
---

You can configure Skaffold with the Skaffold configuration file,
`skaffold.yaml`. The configuration file should be placed in the root of your
project directory; when you run the `skaffold` command, Skaffold will try to
read the configuration file from the current directory.

`skaffold.yaml` consists of five different components:

| Component  | Description |
| ---------- | ------------|
| `apiVersion` | The Skaffold API version you would like to use. The current API version is {{< skaffold-version >}}. |
| `kind`  |  The Skaffold configuration file has the kind `Config`.  |
| `build`  |  Specifies how Skaffold builds artifacts. You have control over what tool Skaffold can use, how Skaffold tags artifacts and how Skaffold pushes artifacts. Skaffold supports using local Docker daemon, Google Cloud Build, Kaniko, or Bazel to build artifacts. See [Builders](/docs/pipeline-stages/builders) and [Taggers]({{< relref "/docs/pipeline-stages/taggers" >}}) for more information. |
| `test` |  Specifies how Skaffold tests artifacts. Skaffold supports [container-structure-tests](https://github.com/GoogleContainerTools/container-structure-test) to test built artifacts. See [Testers]({{< relref "/docs/pipeline-stages/testers" >}}) for more information. |
| `deploy` |  Specifies how Skaffold deploys artifacts. Skaffold supports using `kubectl`, `helm`, or `kustomize` to deploy artifacts. See [Deployers]({{< relref "/docs/pipeline-stages/deployers" >}}) for more information. |
| `profiles`|  Profile is a set of settings that, when activated, overrides the current configuration. You can use Profile to override the `build`, `test` and `deploy` sections. |

You can [learn more]({{< relref "/docs/references/yaml" >}}) about the syntax of `skaffold.yaml`.
