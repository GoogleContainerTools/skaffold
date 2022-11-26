---
title: "Kustomize"
linkTitle: "Kustomize"
weight: 30
featureId: render
---

## Rendering with kustomize

[`kustomize`](https://github.com/kubernetes-sigs/kustomize) allows Kubernetes
developers to customize raw, template-free YAML files for multiple purposes.
Skaffold can work with `kustomize` by calling its command-line interface.

### Configuration

To use kustomize with Skaffold, add manifests type `kustomize` to the `manifests`
section of `skaffold.yaml`.

The `kustomize` type offers the following options:

{{< schema root="Kustomize" >}}

Each entry in `paths` should point to a folder with a kustomization file.

### Example

The following `manifests` section instructs Skaffold to render
artifacts using kustomize:

{{% readfile file="samples/renderers/kustomize.yaml" %}}

{{< alert title="Note" >}}
kustomize CLI must be installed on your machine. Skaffold will not
install it.
{{< /alert >}}
