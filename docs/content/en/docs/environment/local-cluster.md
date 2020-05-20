---
title: "Local Cluster"
linkTitle: "Local Cluster"
weight: 60
aliases: [/docs/concepts/local_development]
---

Skaffold can be easily configured to deploy against a cluster hosted locally, most commonly
with [`minikube`] or [`Docker Desktop`].

The advantage of this setup is that no images need to be pushed, since the local cluster
uses images straight from your local docker daemon. It leads to much faster development cycles.

### Auto detection

Skaffold's heuristic to detect local clusters is based on the Kubernetes context name.
The following context names are checked:

| Kubernetes context | Local cluster type | Notes |
| ------------------ | ------------------ | ----- |
| docker-desktop     | [`Docker Desktop`] | |
| docker-for-desktop | [`Docker Desktop`] | This context name is deprecated |
| minikube           | [`minikube`]       | |
| kind-(.*)          | [`kind`]           | This pattern is used by kind >= v0.6.0 |
| (.*)@kind          | [`kind`]           | This pattern was used by kind < v0.6.0 |
| k3d-(.*)           | [`k3d`]            | This pattern is used by k3d >= v3.0.0 |

For any other name, Skaffold assumes that the cluster is remote and that images
have to be pushed.

 [`minikube`]: https://github.com/kubernetes/minikube/
 [`Docker Desktop`]: https://www.docker.com/products/docker-desktop
 [`kind`]: https://github.com/kubernetes-sigs/kind
 [`k3d`]: https://github.com/rancher/k3d

### Manual override

For non-standard local setups, such as a custom `minikube` profile,
some extra configuration is necessary. The essential steps are:

1. Ensure that Skaffold builds the images with the same docker daemon that runs the pods' containers.
1. Tell Skaffold to skip pushing images either by configuring

    ```yaml
    build:
      local:
        push: false
    ```
   
   or by marking a Kubernetes context as local (see the following example).

For example, when running `minikube` with a custom profile (e.g. `minikube start -p my-profile`):

1. Set up the docker environment for Skaffold with `source <(minikube docker-env -p my-profile)`.
   This should set some environment variables for docker (check with `env | grep DOCKER`).
   **It is important to do this in the same shell where Skaffold is executed.**
   
2. Tell Skaffold that the Kubernetes context `my-profile` refers to a local cluster with

    ```bash
    skaffold config set --kube-context my-profile local-cluster true
    ```

