---
title: "Skaffold Pipeline Stages"
linkTitle: "Skaffold Pipeline Stages"
weight: 4
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


| Skaffold References  |
|----------|
| [Init]({{< relref "/docs/pipeline-stages/init" >}}) |
| [Build]({{< relref "/docs/pipeline-stages/builders" >}}) |
| [Tag]({{< relref "/docs/pipeline-stages/taggers" >}}) |
| [Test]({{< relref "/docs/pipeline-stages/testers" >}}) |
| [Deploy]({{< relref "/docs/pipeline-stages/deployers" >}}) |
| [File Sync]({{< relref "/docs/pipeline-stages/filesync" >}}) |
| [Log Tailing]({{< relref "/docs/pipeline-stages/log-tailing" >}}) |
| [Port Forwarding]({{< relref "/docs/pipeline-stages/port-forwarding" >}}) |
| [Cleanup]({{< relref "/docs/pipeline-stages/cleanup" >}}) |

