---
title: "Tutorials"
linkTitle: "Tutorials"
weight: 90
---

See the [Github Examples page](https://github.com/GoogleContainerTools/skaffold/tree/master/examples) for examples.

### Deploying examples to a remote cluster

When deploying to a remote cluster you have to point Skaffold to your default image repository in one of the four ways:

 1. flag: `skaffold dev --default-repo <myrepo>`
 1. env var: `SKAFFOLD_DEFAULT_REPO=<myrepo> skaffold dev`
 1. global skaffold config (one time): `skaffold config set --global default-repo <myrepo>`
 1. skaffold config for current kubectl context: `skaffold config set default-repo <myrepo>`

### Survey

:mega: **Please fill out our [quick 5-question survey](https://forms.gle/BMTbGQXLWSdn7vEs6)** to tell us how satisfied you are with Skaffold, and what improvements we should make. Thank you! :dancers:

### What's next

Take a look at our other guides:

| Detailed Tutorials |
|--------------------|
| [Custom Build Script]({{< relref "/docs/tutorials/custom-builder" >}}) |
