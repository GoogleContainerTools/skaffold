---
title: "Tutorials"
linkTitle: "Tutorials"
weight: 90
simple_list: true
---

See the [Github Examples page](https://github.com/GoogleContainerTools/skaffold/tree/main/examples) for more examples.

{{< alert title="Deploying examples to a remote cluster" >}}
When deploying to a remote cluster you have to point Skaffold to your default image repository in one of the four ways:

 1. flag: `skaffold dev --default-repo <myrepo>`
 1. env var: `SKAFFOLD_DEFAULT_REPO=<myrepo> skaffold dev`
 1. global skaffold config (one time): `skaffold config set --global default-repo <myrepo>`
 1. skaffold config for current kubectl context: `skaffold config set default-repo <myrepo>`
{{< /alert >}}
