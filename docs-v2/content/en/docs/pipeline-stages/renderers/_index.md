---
title: "Render"
linkTitle: "Render"
weight: 10
featureId: render
aliases: [/docs/how-tos/renderers]
no_list: true
---

When Skaffold renders your application to Kubernetes, it (usually) goes through these steps:

* the Skaffold renderer _renders_ the final Kubernetes manifests: Skaffold replaces untagged image names in the Kubernetes manifests with the final tagged image names.
It also might go through the extra intermediate step of expanding templates (for helm) or calculating overlays (for kustomize).

### Supported renderers

Skaffold supports the following tools for rendering applications:

* [`rawYaml`]({{< relref "./rawYaml.md" >}}) - use this if you don't currently use a rendering tool
* [`helm`]({{< relref "./helm.md" >}})
* [`kpt`]({{< relref "./kpt.md" >}})
* [`kustomize`]({{< relref "./kustomize.md" >}})

Skaffold's render configuration is set through the `manifests` section
of the `skaffold.yaml`. See each renderer's page for more information
on how to configure them for use in Skaffold. It's also possible to use
a combination of multiple renderers in a single project.

For a detailed discussion on Skaffold configuration, see
[Skaffold Concepts]({{< relref "/docs/design/config.md" >}}) and
[skaffold.yaml References]({{< relref "/docs/references/yaml" >}}).