---
title: "Kubectl"
linkTitle: "Kubectl"
weight: 20
featureId: deploy
---

## Deploying with kubectl

`kubectl` is Kubernetes
[command-line tool](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
for deploying and managing
applications on Kubernetes clusters.

Skaffold can work with `kubectl` to
deploy artifacts on any Kubernetes cluster, including
[Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine)
clusters and local [Minikube](https://github.com/kubernetes/minikube) clusters.

### Configuration

To use `kubectl`, add deploy type `kubectl` to the `deploy` section of
`skaffold.yaml`.

The `kubectl` type offers the following options:

{{< schema root="KubectlDeploy" >}}

`flags` section offers the following options:

{{< schema root="KubectlFlags" >}}

### Example

The following `deploy` section instructs Skaffold to deploy
artifacts using `kubectl`:

{{% readfile file="samples/deployers/kubectl.yaml" %}}

{{< alert title="Note" >}}
kubectl CLI must be installed on your machine. Skaffold will not
install it.
Also, it has to be installed in a version that's compatible with your cluster.
{{< /alert >}}