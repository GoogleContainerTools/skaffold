---
title: "Kpt"
linkTitle: "Kpt"
weight: 30
featureId: render
---

## Rendering with kpt

[`kpt`](https://kpt.dev/) allows Kubernetes
developers to customize raw, template-free YAML files for multiple purposes.
Skaffold can work with `kpt` by calling its command-line interface.

### Configuration

To use kpt with Skaffold, add deploy type `kpt` to the `deploy`
section of `skaffold.yaml`.

The `kpt` type offers the following options:

{{< schema root="KptDeploy" >}}

Each entry in `paths` should point to a folder with a kustomization file.

`flags` section offers the following options:

{{< schema root="KubectlFlags" >}}

### Example

The following `deploy` section instructs Skaffold to deploy
artifacts using kpt:

{{% readfile file="samples/deployers/kpt.yaml" %}}

{{< alert title="Note" >}}
kpt CLI must be installed on your machine. Skaffold will not
install it.
{{< /alert >}}
