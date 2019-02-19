---
title: "Tutorials"
linkTitle: "Tutorials"
weight: 90
---

See the [Github Examples page](https://github.com/GoogleContainerTools/skaffold/tree/master/examples) for examples. 

To run the examples, you either have to manually replace the image repositories in the examples from gcr.io/k8s-skaffold to yours or you can point skaffold to your default image repository in one of the four ways:

* flag: `skaffold dev --default-repo <myrepo>`
* env var: `SKAFFOLD_DEFAULT_REPO=<myrepo> skaffold dev`
* global skaffold config (one time): `skaffold config set --global default-repo <myrepo>`
* skaffold config for current kubectl context: `skaffold config set default-repo <myrepo>`
