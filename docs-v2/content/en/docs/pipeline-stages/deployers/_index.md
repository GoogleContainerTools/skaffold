---
title: "Deploy"
linkTitle: "Deploy"
weight: 10
featureId: deploy
aliases: [/docs/how-tos/deployers]
no_list: true
---

When Skaffold deploys your application to Kubernetes, it (usually) goes through these steps:

In the default case (no manifest provided using the kubectl or kpt deployer ), skaffold deploy will do the following:
* the Skaffold renderer _renders_ the final Kubernetes manifests: Skaffold replaces untagged image names in the Kubernetes manifests with the final tagged image names.
It also might go through the extra intermediate step of expanding templates (for helm) or calculating overlays (for kustomize).
* the Skaffold deployer _deploys_ the final Kubernetes manifests to the cluster (or to local docker for the docker deployer)
* the Skaffold deployer performs [status checks]({{< relref "/docs/pipeline-stages/status-check" >}}) and waits for the deployed resources to stabilize.

In the case that a manifest file is supplied via the --TODO flag or the deployer does not accept manifests (eg: helm and docker) skaffold deploy will only do the following steps:
* the Skaffold deployer _deploys_ the supplied Kubernetes manifests to the cluster (or deploys the specified container to docker for the docker deployer)
* the Skaffold deployer performs [status checks]({{< relref "/docs/pipeline-stages/status-check" >}}) and waits for the deployed resources to stabilize

### Supported deployers

Skaffold supports the following tools for deploying applications:

* [`kubectl`]({{< relref "./kubectl.md" >}})
* [`helm`]({{< relref "./helm.md" >}})
* [`kpt`]({{< relref "./kpt.md" >}})
* [`docker`]({{< relref "./docker.md" >}}) (does not deploy to Kubernetes: see documentation for more details)

Skaffold's deploy configuration is set through the `deploy` section
of the `skaffold.yaml`. See each deployer's page for more information
on how to configure them for use in Skaffold. It's also possible to use
a combination of multiple deployers in a single project.

For a detailed discussion on Skaffold configuration, see
[Skaffold Concepts]({{< relref "/docs/design/config.md" >}}) and
[skaffold.yaml References]({{< relref "/docs/references/yaml" >}}).