---
title: "Architecture and Design"
linkTitle: "Architecture and Design"
weight: 50
aliases: [/docs/concepts,/docs/concepts/architecture]
---

Skaffold is designed with pluggability in mind:

![architecture](/images/architecture.png)

The architecture allows you to use Skaffold with the tool you prefer. Skaffold
provides built-in support for the following tools:

* **Build**
  * Dockerfile locally, in-cluster with kaniko or on cloud using Google Cloud Build
  * Jib Maven and Jib Gradle locally or on cloud using Google Cloud Build
  * Bazel locally
  * Cloud Native Buildpacks locally or on cloud using Google Cloud Build
  * Custom script locally or in-cluster
* **Test**
  * [container-structure-test](https://github.com/GoogleContainerTools/container-structure-test)
* **Tag**
  * Git tagger
  * Sha256 tagger
  * Env Template tagger
  * DateTime tagger
* **Deploy**
  * Kubernetes Command-Line Interface (`kubectl`)
  * [Helm](https://helm.sh/)
  * [kustomize](https://github.com/kubernetes-sigs/kustomize)

You can combine the tools as you see fit in Skaffold. For experimental
projects, you may want to use local Docker daemon for building artifacts, and
deploy them to a Minikube local Kubernetes cluster with `kubectl`:

![workflow_local](/images/workflow_local.png)

However, for production applications, you might find it more appropriate to build
with Google Cloud Build and deploy using Helm:

![workflow_gcb](/images/workflow_gcb.png)

Skaffold also supports development profiles. You can specify multiple different
profiles in your configuration and use the one that best serves your needs
without having to modify the configuration file. You can learn more about
profiles [here]({{< relref "../environment/profiles.md" >}}).
