
---
title: "Deployers"
linkTitle: "Deployers"
weight: 20
---

This page discusses how to set up Skaffold to use the tool of your choice
to deploy your app to a Kubernetes cluster.

When Skaffold deploys an application the following steps happen: 

* the Skaffold deployer _renders_ the final kubernetes manifests: Skaffold replaces the image names in the kubernetes manifests with the final tagged image names. 
Also, in case of the more complicated deployers the rendering step involves expanding templates (in case of helm) or calculating overlays (in case of kustomize). 
* the Skaffold deployer _deploys_ the final kubernetes manifests to the cluster

Skaffold supports the following tools for deploying applications:

* [`kubectl`](#deploying-with-kubectl) 
* [Helm](#deploying-with-helm) 
* [kustomize](#deploying-with-kustomize)

The `deploy` section in the Skaffold configuration file, `skaffold.yaml`,
controls how Skaffold builds artifacts. To use a specific tool for deploying
artifacts, add the value representing the tool and options for using the tool
to the `build` section. For a detailed discussion on Skaffold configuration,
see [Skaffold Concepts: Configuration](/docs/concepts/#configuration) and
[Skaffold.yaml References](https://github.com/GoogleContainerTools/skaffold/blob/master/examples/annotated-skaffold.yaml).

## Deploying with kubectl

[`kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/) is 
Kubernetes command-line tool for deploying and managing
applications on Kubernetes clusters. Skaffold can work with `kubectl` to
deploy artifacts on any Kubernetes cluster, including
[Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine)
clusters and local [Minikube](https://github.com/kubernetes/minikube) clusters.

To use `kubectl`, add deploy type `kubectl` to the `deploy` section of
`skaffold.yaml`. The `kubectl` type offers the following options:

|Option|Description|
|----|----|
|`manifests`| OPTIONAL. A list of paths to Kubernetes Manifests. Default value is `k8s/*.yaml`.|
|`remoteManifests`|OPTIONAL. A list of paths to Kubernetes Manifests in remote clusters.|
|`flags`| OPTIONAL. Additional flags to pass to `kubectl`. You can specify three types of flags: <ul> <li>`global`: flags that apply to every command.</li> <li>`apply`: flags that apply to creation commands.</li> <li>`delete`: flags that apply to deletion commands.</li><ul>|

The following `deploy` section, for example, instructs Skaffold to deploy
artifacts using `kubectl`:

```yaml
deploy:
    kubectl:
      manifests:
        - k8s-*
    # Uncomment the following lines to add remote manifests and flags
    # remoteManifests:
    #    - YOUR-REMOTE-MANIFESTS
    # flags:
    #    global:
    #    - YOUR-GLOBAL-FLAGS
    #    apply:
    #    - YOUR-APPLY-FLAGS
    #    delete:
    #    - YOUR-DELETE-FLAGS
```

## Deploying with Helm

[Helm](https://helm.sh/) is a package manager for Kubernetes that helps you
manage Kubernetes applications. Skaffold can work with Helm by calling its
command-line interface.

To use Helm with Skaffold, add deploy type `helm` to the `deploy` section
of `skaffold.yaml`. The `helm` type offers the following options:

|Option|Description|
|----|----|
|`releases`|<b>Required</b> A list of Helm releases. See the table below for the schema of `releases`.|

Each release includes the following fields:

|Option|Description|
|----|----|
|`name`| <b>Required</b> The name of the Helm release.|
|`chartPath`|<b>Required</b> The path to the Helm chart.|
|`valuesFilePath`| The path to the Helm `values` file.|
|`values`| A list of key-value pairs supplementing the Helm `values` file.|
|`namespace`| The Kubernetes namespace.|
|`version`| The version of the chart.|
|`setValues`| A list of key-value pairs; if present, Skaffold will sent `--set` flag to Helm CLI and append all pairs after the flag.|
|`setValueTemplates`| A list of key-value pairs; if present, Skaffold will try to parse the value part of each key-value pair using environment variables in the system, then send `--set` flag to Helm CLI and append all parsed pairs after the flag.|
|`wait`| A boolean value; if `true`, Skaffold will send `--wait` flag to Helm CLI.|
|`recreatePods`| A boolean value; if `true`, Skaffold will send `--recreate-pods` flag to Helm CLI.|
|`overrides`| A list of key-value pairs; if present, Skaffold will build a Helm `values` file that overrides the original and use it to call Helm CLI (`--f` flag).|
|`packaged`|Packages the chart (`helm package`) Includes two fields: <ul> <li>`version`: Version of the chart.</li> <li>`appVersion`: Version of the app.</li> </ul>| |`imageStrategy`|Add image configurations to the Helm `values` file. Includes one of the two following fields: <ul> <li> `fqn`: The image configuration uses the syntax `IMAGE-NAME=IMAGE-REPOSITORY:IMAGE-TAG`. </li> <li>`helm`: The image configuration uses the syntax `IMAGE-NAME.repository=IMAGE-REPOSITORY, IMAGE-NAME.tag=IMAGE-TAG`.</li> </ul> |

The following `deploy` section, for example, instructs Skaffold to deploy
artifacts using `helm`:

```yaml
deploy:
  helm:
    releases:
    - name: skaffold-helm
      chartPath: skaffold-helm
      values:
        image: gcr.io/k8s-skaffold/skaffold-helm
      # Uncomment the following lines to specify more parameters
      # valuesFilePath: YOUR-VALUES-FILE-PATH
      # namespace: YOUR-NAMESPACE
      # version: YOUR-VERSION
      # setValues:
      #     SOME-KEY: SOME-VALUE
      # setValues:
      #     SOME-KEY: SOME-VALUE-TEMPLATE
      # setValueTemplates:
      #     SOME-KEY: "{{.SOME-ENV-VARIABLE}}"
      # wait: true
      # recreatePods: true
      # overrides:
      #     SOME-KEY: SOME-VALUE
      #     SOME-MORE-KEY:
      #         SOME-KEY: SOME-VALUE
      # packaged:
      #     version: YOUR-VERSION
      #     appVersion: YOUR-APP-VERSION
      # imageStrategy:
      #     helm: {}
      # OR
      #     fqn: {}
```

## Deploying with kustomize

[kustomize](https://github.com/kubernetes-sigs/kustomize) allows Kubernetes
developers to customize raw, template-free YAML files for multiple purposes.
Skaffold can work with kustomize by calling its command-line interface.

To use kustomize with Skaffold, add deploy type `kustomize` to the `deploy`
section of `skaffold.yaml`. The `kustomize` type offers the following options:

|Option|Description|
|----|----|
|`path`| <b>Optional</b> Path to Kustomization files. The default value is `.` (current directory).|
|`flags`| OPTIONAL. Additional flags to pass to `kubectl`. You can specify three types of flags: <ul> <li>`global`: flags that apply to every command.</li> <li>`apply`: flags that apply to creation commands.</li> <li>`delete`: flags that apply to deletion commands.</li> <ul> |

The following `deploy` section, for example, instructs Skaffold to deploy
artifacts using kustomize:

```yaml
deploy:
  kustomize:
    path: "."
# The deploy section above is equal to
# deploy:
#   kustomize: {}
```
