---
title: "Deployers"
linkTitle: "Deployers"
weight: 30
---

This page discusses how to set up Skaffold to use the tool of your choice
to deploy your app to a Kubernetes cluster.

When Skaffold deploys an application the following steps happen:

* the Skaffold deployer _renders_ the final kubernetes manifests: Skaffold replaces the image names in the kubernetes manifests with the final tagged image names.
Also, in case of the more complicated deployers the rendering step involves expanding templates (in case of helm) or calculating overlays (in case of kustomize).
* the Skaffold deployer _deploys_ the final kubernetes manifests to the cluster

### Supported deployers

Skaffold supports the following tools for deploying applications:

* [`kubectl`](#deploying-with-kubectl)
* [helm](#deploying-with-helm)
* [kustomize](#deploying-with-kustomize)

The `deploy` section in the Skaffold configuration file, `skaffold.yaml`,
controls how Skaffold builds artifacts. To use a specific tool for deploying
artifacts, add the value representing the tool and options for using the tool
to the `deploy` section.

For a detailed discussion on Skaffold configuration, see
[Skaffold Concepts](/docs/concepts/#configuration) and
[skaffold.yaml References](/docs/references/yaml).

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

## Deploying with Helm

[Helm](https://helm.sh/) is a package manager for Kubernetes that helps you
manage Kubernetes applications. Skaffold can work with Helm by calling its
command-line interface.

### Configuration

To use Helm with Skaffold, add deploy type `helm` to the `deploy` section of `skaffold.yaml`.

The `helm` type offers the following options:

{{< schema root="HelmDeploy" >}}

Each `release` includes the following fields:

{{< schema root="HelmRelease" >}}

### Helm Build Dependencies

The `skipBuildDependencies` flag toggles whether depenedencies of the Helm chart are built with the `helm dep build` command. This command manipulates files inside the `charts` subfolder of the specified Helm chart.

If `skipBuildDependencies` is `false` then `skaffold dev` does **not** watch the `charts` subfolder of the Helm chart, in order to prevent a build loop - the actions of `helm dep build` always trigger another build.

If `skipBuildDependencies` is `true` then `skaffold dev` watches all files inside the Helm chart.


### Example

The following `deploy` section instructs Skaffold to deploy
artifacts using `helm`:

{{% readfile file="samples/deployers/helm.yaml" %}}

{{< alert title="Note" >}}
helm CLI must be installed on your machine. Skaffold will not
install it.
Also, it has to be installed in a version that's compatible with your cluster.
{{< /alert >}}

## Deploying with kustomize

[kustomize](https://github.com/kubernetes-sigs/kustomize) allows Kubernetes
developers to customize raw, template-free YAML files for multiple purposes.
Skaffold can work with `kustomize` by calling its command-line interface.

### Configuration

To use kustomize with Skaffold, add deploy type `kustomize` to the `deploy`
section of `skaffold.yaml`.

The `kustomize` type offers the following options:

{{< schema root="KustomizeDeploy" >}}

`flags` section offers the following options:

{{< schema root="KubectlFlags" >}}

### Example

The following `deploy` section instructs Skaffold to deploy
artifacts using kustomize:

{{% readfile file="samples/deployers/kustomize.yaml" %}}

{{< alert title="Note" >}}
kustomize CLI must be installed on your machine. Skaffold will not
install it.
{{< /alert >}}
