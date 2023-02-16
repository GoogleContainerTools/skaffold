---
title: "Skaffold Pipeline Stages"
linkTitle: "Skaffold Pipeline Stages"
weight: 40
aliases: [/docs/concepts/pipeline]
no_list: true
---

Skaffold features a multi-stage workflow:

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
| [Init]({{< relref "/docs/init" >}}) | generate a starting point for Skaffold configuration | 
| [Build]({{< relref "/docs/builders" >}}) | build images with different builders | 
| [Render]({{< relref "/docs/renderers" >}}) | render manifests with different renderers | 
| [Tag]({{< relref "/docs/taggers" >}}) | tag images based on different policies |
| [Test]({{< relref "/docs/testers" >}}) | run tests with testers |
| [Deploy]({{< relref "/docs/deployers" >}}) |  deploy with kubectl, kustomize or helm |
| [Verify]({{< relref "/docs/verify" >}}) |  verify deployments with specified test containers |
| [File Sync]({{< relref "/docs/filesync" >}}) |  sync changed files directly to containers |
| [Log Tailing]({{< relref "/docs/log-tailing" >}}) |  tail logs from workloads |
| [Port Forwarding]({{< relref "/docs/port-forwarding" >}}) | forward ports from services and arbitrary resources to localhost  |
| [Deploy Status Checking]({{< relref "/docs/status-check" >}}) | wait for deployed resources to stabilize  |
| [Lifecycle Hooks]({{< relref "/docs/lifecycle-hooks" >}}) | run code triggered by different events during the skaffold process lifecycle  |
| [Cleanup]({{< relref "/docs/cleanup" >}}) | cleanup manifests and images |
