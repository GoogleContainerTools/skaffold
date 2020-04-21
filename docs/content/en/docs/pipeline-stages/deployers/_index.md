---
title: "Deploy"
linkTitle: "Deploy"
weight: 10
featureId: deploy
aliases: [/docs/how-tos/deployers]
---

When Skaffold deploys your application, it goes through these steps:

* the Skaffold deployer _renders_ the final Kubernetes manifests: Skaffold replaces untagged image names in the Kubernetes manifests with the final tagged image names.
It also might go through the extra intermediate step of expanding templates (for helm) or calculating overlays (for kustomize).
* the Skaffold deployer _deploys_ the final Kubernetes manifests to the cluster

### Supported deployers

Skaffold supports the following tools for deploying applications:

* [`kubectl`]({{< relref "./kubectl.md" >}})
* [`helm`]({{< relref "./helm.md" >}})
* [`kustomize`]({{< relref "./kustomize.md" >}})

Skaffold's deploy configuration is set through the `deploy` section
of the `skaffold.yaml`. See each deployer's page for more information
on how to configure them for use in Skaffold. It's also possible to use
a combination of multiple deployers in a single project.

For a detailed discussion on Skaffold configuration, see
[Skaffold Concepts]({{< relref "/docs/design/config.md" >}}) and
[skaffold.yaml References]({{< relref "/docs/references/yaml" >}}).