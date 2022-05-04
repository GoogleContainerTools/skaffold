---
title: "Kustomize"
linkTitle: "Kustomize"
weight: 30
featureId: deploy
---

## Deploying with kustomize

[`kustomize`](https://github.com/kubernetes-sigs/kustomize) allows Kubernetes
developers to customize raw, template-free YAML files for multiple purposes.
Skaffold can work with `kustomize` by calling its command-line interface.

### Configuration

To use kustomize with Skaffold, add deploy type `kustomize` to the `deploy`
section of `skaffold.yaml`.

The `kustomize` type offers the following options:

{{< schema root="KustomizeDeploy" >}}

Each entry in `paths` should point to a folder with a kustomization file.

`flags` section offers the following options:

{{< schema root="KubectlFlags" >}}

### Example

The following `deploy` section instructs Skaffold to deploy
artifacts using kustomize:

{{% readfile file="samples/deployers/kustomize.yaml" %}}

{{< alert title="Note" >}}
kustomize CLI must be installed on your machine. Skaffold will not
install it.
{{< /alert >}}
