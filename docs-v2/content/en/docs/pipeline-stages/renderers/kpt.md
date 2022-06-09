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

To use kpt with Skaffold, add render type `kpt` to the `manifests`
section of `skaffold.yaml`.

The `kpt` configuration accepts a list of paths to folders containing a Kptfile.

### Example

The following `manifests` section instructs Skaffold to render
artifacts using kpt.  Each entry should point to a folder with a Kptfile.

{{% readfile file="samples/renderers/kpt.yaml" %}}

{{< alert title="Note" >}}
kpt CLI must be installed on your machine. Skaffold will not
install it.
{{< /alert >}}
