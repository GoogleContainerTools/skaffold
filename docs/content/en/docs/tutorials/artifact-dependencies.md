---
title: "Defining dependencies between artifacts"
linkTitle: "Inter-Artifact Dependencies"
weight: 100
---

This page describes how to define dependencies between artifacts and reference them in the [docker builder]({{<relref "/docs/pipeline-stages/builders/docker" >}}).

## Before you begin

First, you will need to have Skaffold and a Kubernetes cluster set up.
To learn more about how to set up Skaffold and a Kubernetes cluster, see the [quickstart docs]({{< relref "/docs/quickstart" >}}).

## Tutorial - µSvcs with inter-artifact dependencies

This tutorial will be based on the [microservices example](https://github.com/GoogleContainerTools/skaffold/tree/master/examples/microservices) in our repository.


## Adding an artifact dependency

We have a `base` artifact which has a single Dockerfile that we build with the [docker builder]({{<relref "/docs/pipeline-stages/builders/docker" >}}):
 {{% readfile file="samples/builders/artifact-dependencies/Dockerfile.base" %}}

This artifact is used as the base image for the `leeroy-app` and `leeroy-web` artifacts. We express this dependency in the `skaffold.yaml` using the `requires` expression.
{{% readfile file="samples/builders/artifact-dependencies/skaffold.yaml" %}}

This allows us to use `BASE` as a build-arg in the Dockerfile for `leeroy-app` (and similarly for `leeroy-web`):
 
{{% readfile file="samples/builders/artifact-dependencies/Dockerfile.app" %}}

Skaffold orchestrates these builds in dependency order and injects the value of `BASE` with the image built for the `base` artifact while building `leeroy-web` and `leeroy-app`.
You can run `skaffold build` to build these artifacts, or `skaffold dev` to build and deploy with file change monitoring.