---
title: "Workflow"
linkTitle: "Workflow"
weight: 10
---

This page discusses the development workflow with Skaffold.


Skaffold features a five-stage workflow:

![workflow](/images/workflow.png)

When you start Skaffold, it collects source code in your project and builds
artifacts with the tool of your choice; the artifacts, once successfully built,
are tagged as you see fit and pushed to the repository you specify. In the
end of the workflow, Skaffold also helps you deploy the artifacts to your
Kubernetes cluster, once again using the tools you prefer.

Skaffold allows you to skip stages. If, for example, you run Kubernetes
locally with [Minikube](https://kubernetes.io/docs/setup/minikube/), Skaffold
will not push artifacts to a remote repository.

