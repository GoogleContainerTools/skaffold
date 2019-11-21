---
title: "Local Cluster"
linkTitle: "Local Cluster"
weight: 60
aliases: [/docs/concepts/local_development]
---

Skaffold can be easily configured to deploy against a cluster hosted locally, most commonly with
[`minikube`](https://github.com/kubernetes/minikube/) or `docker-for-desktop`.
The advantage of this setup is that no images need to be pushed, since the local cluster
uses images straight from your local docker daemon.

For non-standard local setups, such as a custom `minikube` profile or [kind](https://github.com/kubernetes-sigs/kind),
some extra configuration is necessary. The essential steps are:

1. Ensure that Skaffold builds the images with the docker daemon, which also runs the containers.
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
