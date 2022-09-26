---
title: "Helm"
linkTitle: "Helm"
weight: 40
featureId: render
---

[`helm`](https://helm.sh/) is a package manager for Kubernetes that helps you
manage Kubernetes applications. Skaffold natively supports iterative development
for projects configured to use helm.

{{< alert title="Note" >}}
To use `helm` with Skaffold, the `helm` binary must be installed on your machine. Skaffold will not install it for you.
{{< /alert >}}

# Rendering with helm
[`helm template`](https://helm.sh/docs/helm/helm_template/) allows Kubernetes
developers to locally render templates. Skaffold relies on `helm temple --post-render` functionality to substitute the images
in the rendered charts with Skaffold built images.


{{< alert title="Note" >}}
If you wish to deploy using helm, please see [Configuring Helm Deployer Section]({{< relref "/docs/pipeline-stages/deployers/helm.md" >}})
{{< /alert >}}

### Configuration

To use render using helm but deploy via kubectl deployer define your helm charts under
`helm` in `manifests` section of `skaffold.yaml`.


### Example
The following `manifests` section instructs Skaffold to render
artifacts using helm.

{{% readfile file="samples/renderers/helm.yaml" %}}


### `skaffold.yaml` Configuration

The `helm` type offers the following options:

{{< schema root="manifests-helm" >}}

Each `release` includes the following fields:

{{< schema root="HelmRelease" >}}

