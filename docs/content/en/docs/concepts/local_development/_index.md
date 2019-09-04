---
title: "Local development"
linkTitle: "Local development"
weight: 60
---

This page discusses how to develop locally with Skaffold.


Local development means that Skaffold can skip pushing built container images, because the images are already present where they are run.
For standard development setups such as `minikube` and `docker-for-desktop`, this works out of the box.

However, for non-standard local setups, such as [minikube](https://github.com/kubernetes/minikube/) with custom profile or [kind](https://github.com/kubernetes-sigs/kind), some extra configuration is necessary.
The essential steps are:

1. Ensure that Skaffold builds the images with the docker daemon, which also runs the containers.
2. Tell Skaffold to skip pushing images either by configuring

    ```yaml
    build:
      local:
        push: false
    ```
   
   or by marking a kubernetes context as local (see the following example).

For example, when running `minikube` with a custom profile, such as `minikube start -p my-profile`:

1. Set up the docker environment for Skaffold with `source <(minikube docker-env -p my-profile)`.
   This should set some environment variables for docker (check with `env | grep DOCKER`).
   It is important to do this in the same shell where Skaffold is executed.
   
2. Tell Skaffold that the kubernetes context `my-profile` refers to a local cluster with

    ```bash
    skaffold config set --kube-context my-profile local-cluster true
    ```
