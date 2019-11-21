---
title: "Skaffold Pipeline Stages"
linkTitle: "Skaffold Pipeline Stages"
weight: 40
aliases: [/docs/concepts/pipeline]
---

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


| Skaffold Pipeline stages|Description| 
|----------|-------|------|
| [Init]({{< relref "/docs/pipeline-stages/init" >}}) | generate a starting point for Skaffold configuration | 
| [Build]({{< relref "/docs/pipeline-stages/builders" >}}) | build images with different builders | 
| [Tag]({{< relref "/docs/pipeline-stages/taggers" >}}) | tag images based on different policies |
| [Test]({{< relref "/docs/pipeline-stages/testers" >}}) |  test images with structure tests |
| [Deploy]({{< relref "/docs/pipeline-stages/deployers" >}}) |  deploy with kubectl, kustomize or helm |
| [File Sync]({{< relref "/docs/pipeline-stages/filesync" >}}) |  sync changed files directly to containers |
| [Log Tailing]({{< relref "/docs/pipeline-stages/log-tailing" >}}) |  tail logs from workloads |
| [Port Forwarding]({{< relref "/docs/pipeline-stages/port-forwarding" >}}) | forward ports from services and arbitrary resources to localhost  |
| [Cleanup]({{< relref "/docs/pipeline-stages/cleanup" >}}) | cleanup manifests and images |
