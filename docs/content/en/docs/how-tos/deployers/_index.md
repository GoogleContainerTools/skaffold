
---
title: "Using Deployers"
linkTitle: "Using Deployers"
weight: 50
---

This page discusses how to set up Skaffold to use the tool of your choice
to deploy your app to a Kubernetes cluster.

When skaffold deploys an application the following steps happen: 
* the skaffold deployer _renders_ the final kubernetes manifests: skaffold replaces the image names in the kubernetes manifests with the final tagged image names. 
Also, in case of the more complicated deployers the rendering step involves expanding templates (in case of helm) or calculating overlays (in case of kustomize). 
* the skaffold deployer _deploys_ the final kubernetes manifests to the cluster

At this moment, Skaffold supports the following tools for deploying applications:

* [`kubectl`](#deploying-with-kubectl) 
* [Helm](#deploying-with-helm) 
* [kustomize](#deploying-with-kustomize)

The `deploy` section in the Skaffold configuration file, `skaffold.yaml`,
controls how Skaffold builds artifacts. To use a specific tool for deploying
artifacts, add the value representing the tool and options for using the tool
to the `build` section. For a detailed discussion on Skaffold configuration,
see [Skaffold Concepts: Configuration](/docs/concepts/#configuration) and
[Skaffold.yaml References](/docs/references/config).

## Deploying with kubectl

[`kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/) is 
Kubernetes command-line tool for deploying and managing
applications on Kubernetes clusters. Skaffold can work with `kubectl` to
deploy artifacts on any Kubernetes cluster, including
[Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine)
clusters and local [Minikube](https://github.com/kubernetes/minikube) clusters.

To use `kubectl`, add deploy type `kubectl` to the `deploy` section of
`skaffold.yaml`. The `kubectl` type offers the following options:

<table>
    <thead>
        <tr>
            <th>Option</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td>`manifests`</td>
            <td>
                OPTIONAL
                A list of paths to Kubernetes Manifests.
                Default value is `kubectl`.
            </td>
        </tr>
        <tr>
            <td>`remoteManifests`</td>
            <td>
                OPTIONAL
                A list of paths to Kubernetes Manifests in remote clusters.
            </td>
        </tr>
        <tr>
            <td>`flags`</td>
            <td>
                OPTIONAL
                Additional flags to pass to `kubectl`.
                You can specify three types of flags:
                <ul>
                    <li>`global`: flags that apply to every command.</li>
                    <li>`apply`: flags that apply to creation commands.</li>
                    <li>`delete`: flags that apply to deletion commands.</li>
                <ul>
            </td>
        </tr>
    </tbody>
<table>

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

<table>
    <thead>
        <tr>
            <th>Option</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td>`releases`</td>
            <td>
                <b>Required</b>
                A list of Helm releases.
                See the table below for the schema of `releases`.
            </td>
        </tr>
    </tbody>
<table>

Each release includes the following fields:

<table>
    <thead>
        <tr>
            <th>Option</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td>`name`</td>
            <td>
                <b>Required</b>
                The name of the Helm release.
            </td>
        </tr>
        <tr>
            <td>`chartPath`</td>
            <td>
                <b>Required</b>
                The path to the Helm chart.
            </td>
        </tr>
        <tr>
            <td>`valuesFilePath`</td>
            <td>
                The path to the Helm `values` file.
            </td>
        </tr>
        <tr>
            <td>`values`</td>
            <td>
                A list of key-value pairs supplementing the Helm `values` file.
            </td>
        </tr>
        <tr>
            <td>`namespace`</td>
            <td>
                The Kubernetes namespace.
            </td>
        </tr>
        <tr>
            <td>`version`</td>
            <td>
                The version of the chart.
            </td>
        </tr>
        <tr>
            <td>`setValues`</td>
            <td>
                A list of key-value pairs; if present, Skaffold will sent `--set` flag to Helm CLI and append all pairs after the flag.
            </td>
        </tr>
        <tr>
            <td>`setValuesTemplates`</td>
            <td>
                A list of key-value pairs; if present, Skaffold will try to parse the value part of each key-value pair using environment variables in the system, then send `--set` flag to Helm CLI and append all parsed pairs after the flag.
            </td>
        </tr>
        <tr>
            <td>`wait`</td>
            <td>
                A boolean value; if `true</code>, Skaffold will send <code>--wait` flag to Helm CLI.
            </td>
        </tr>
        <tr>
            <td>`recreatePods`</td>
            <td>
                A boolean value; if `true</code>, Skaffold will send <code>--recreate-pods` flag to Helm CLI.
            </td>
        </tr>
        <tr>
            <td>`overrides`</td>
            <td>
                A list of key-value pairs; if present, Skaffold will build a Helm `values</code> file that overrides the original and use it to call Helm CLI (<code>--f` flag).
            </td>
        </tr>
        <tr>
            <td>`packaged`</td>
            <td>
                Packages the chart (`helm package`)
                Includes two fields:
                <ul>
                    <li>`version`: Version of the chart.</li>
                    <li>`appVersion`: Version of the app.</li>
                </ul>
            </td>
        </tr>
        <tr>
            <td>`imageStrategy`</td>
            <td>
                Add image configurations to the Helm `values` file.
                Includes one of the two following fields:
                <ul>
                    <li>
                        `fqn</code>: The image configuration uses the syntax <code>IMAGE-NAME=IMAGE-REPOSITORY:IMAGE-TAG`.
                    </li>
                    <li>`helm</code>: The image configuration uses the syntax <code>IMAGE-NAME.repository=IMAGE-REPOSITORY, IMAGE-NAME.tag=IMAGE-TAG`.</li>
                </ul>
            </td>
        </tr>
    </tbody>
<table>

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

<table>
    <thead>
        <tr>
            <th>Option</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td>`kustomizePath`</td>
            <td>
                <b>Optional</b>
                Path to Kustomization files..
                The default value is `.` (current directory).
            </td>
        </tr>
        <tr>
            <td>`flags`</td>
            <td>
                OPTIONAL
                Additional flags to pass to `kubectl`.
                You can specify three types of flags:
                <ul>
                    <li>`global`: flags that apply to every command.</li>
                    <li>`apply`: flags that apply to creation commands.</li>
                    <li>`delete`: flags that apply to deletion commands.</li>
                <ul>
            </td>
        </tr>
    </tbody>
<table>

The following `deploy` section, for example, instructs Skaffold to deploy
artifacts using kustomize:

```yaml
apiVersion: skaffold/v1alpha2
   kind: Config
   deploy:
     kustomize:
        kustomizePath: "."
# The deploy section above is equal to
# apiVersion: skaffold/v1alpha2
#    kind: Config
#    deploy:
#      kustomize: {}
```
