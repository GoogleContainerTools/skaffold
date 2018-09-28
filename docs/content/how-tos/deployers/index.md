---
title: "Skaffold Documentation: Using Deployers"
date: 2018-09-08T00:00:00-07:00
draft: false
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
* [Kustomize](#deploying-with-kustomize)

The `deploy` section in the Skaffold configuration file, `skaffold.yaml`,
controls how Skaffold builds artifacts. To use a specific tool for deploying
artifacts, add the value representing the tool and options for using the tool
to the `build` section. For a detailed discussion on Skaffold configuration,
see [Skaffold Concepts: Configuration](/concepts/config) and
[Skaffold.yaml References](/references/config).

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
            <td><code>manifests</code></td>
            <td>
                OPTIONAL
                <p>A list of paths to Kubernetes Manifests.</p>
                <p>Default value is <code>k8s/*.yaml</code>; Skaffold will ask <code>kubectl</code> to deploy all the YAML files under directory <code>k8s</code>.</p>
            </td>
        </tr>
        <tr>
            <td><code>remoteManifests</code></td>
            <td>
                OPTIONAL
                <p>A list of paths to Kubernetes Manifests in remote clusters.</p>
            </td>
        </tr>
        <tr>
            <td><code>flags</code></td>
            <td>
                OPTIONAL
                <p>Additional flags to pass to <code>kubectl</code>.</p>
                <p>You can specify three types of flags:</p>
                <ul>
                    <li><code>global</code>: flags that apply to every command.</li>
                    <li><code>apply</code>: flags that apply to creation commands.</li>
                    <li><code>delete</code>: flags that apply to deletion commands.</li>
                <ul>
            </td>
        </tr>
    </tbody>
<table>

The following `deploy` section, for example, instructs Skaffold to deploy
artifacts using `kubectl`:

```
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
            <td><code>releases</code></td>
            <td>
                <b>Required</b>
                <p>A list of Helm releases.</p>
                <p>See the table below for the schema of <code>releases</code>.</p>
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
            <td><code>name</code></td>
            <td>
                <b>Required</b>
                <p>The name of the Helm release.</p>
            </td>
        </tr>
        <tr>
            <td><code>chartPath</code></td>
            <td>
                <b>Required</b>
                <p>The path to the Helm chart.</p>
            </td>
        </tr>
        <tr>
            <td><code>valuesFilePath</code></td>
            <td>
                <p>The path to the Helm <code>values</code> file.</p>
            </td>
        </tr>
        <tr>
            <td><code>values</code></td>
            <td>
                <p>A list of key-value pairs supplementing the Helm <code>values</code> file.</p>
            </td>
        </tr>
        <tr>
            <td><code>namespace</code></td>
            <td>
                <p>The Kubernetes namespace.</p>
            </td>
        </tr>
        <tr>
            <td><code>version</code></td>
            <td>
                <p>The version of the chart.</p>
            </td>
        </tr>
        <tr>
            <td><code>setValues</code></td>
            <td>
                <p>A list of key-value pairs; if present, Skaffold will sent <code>--set</code> flag to Helm CLI and append all pairs after the flag.</p>
            </td>
        </tr>
        <tr>
            <td><code>setValuesTemplates</code></td>
            <td>
                <p>A list of key-value pairs; if present, Skaffold will try to parse the value part of each key-value pair using environment variables in the system, then send <code>--set</code> flag to Helm CLI and append all parsed pairs after the flag.</p>
            </td>
        </tr>
        <tr>
            <td><code>wait</code></td>
            <td>
                <p>A boolean value; if <code>true</code>, Skaffold will send <code>--wait</code> flag to Helm CLI.</p>
            </td>
        </tr>
        <tr>
            <td><code>recreatePods</code></td>
            <td>
                <p>A boolean value; if <code>true</code>, Skaffold will send <code>--recreate-pods</code> flag to Helm CLI.</p>
            </td>
        </tr>
        <tr>
            <td><code>overrides</code></td>
            <td>
                <p>A list of key-value pairs; if present, Skaffold will build a Helm <code>values</code> file that overrides the original and use it to call Helm CLI (<code>--f</code> flag).</p>
            </td>
        </tr>
        <tr>
            <td><code>packaged</code></td>
            <td>
                <p>Packages the chart (<code>helm package</code>)</p>
                <p>Includes two fields:</p>
                <ul>
                    <li><code>version</code>: Version of the chart.</li>
                    <li><code>appVersion</code>: Version of the app.</li>
                </ul>
            </td>
        </tr>
        <tr>
            <td><code>imageStrategy</code></td>
            <td>
                <p>Add image configurations to the Helm <code>values</code> file.</p>
                <p>Includes one of the two following fields:</p>
                <ul>
                    <li>
                        <code>fqn</code>: The image configuration uses the syntax <code>IMAGE-NAME=IMAGE-REPOSITORY:IMAGE-TAG</code>.
                    </li>
                    <li><code>helm</code>: The image configuration uses the syntax <code>IMAGE-NAME.repository=IMAGE-REPOSITORY, IMAGE-NAME.tag=IMAGE-TAG</code>.</li>
                </ul>
            </td>
        </tr>
    </tbody>
<table>

The following `deploy` section, for example, instructs Skaffold to deploy
artifacts using `helm`:

```
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

## Deploying with Kustomize

[Kustomize](https://github.com/kubernetes-sigs/kustomize) allows Kubernetes
developers to customize raw, template-free YAML files for multiple purposes.
Skaffold can work with Kustomize by calling its command-line interface.

To use Kustomize with Skaffold, add deploy type `kustomize` to the `deploy`
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
            <td><code>kustomizePath</code></td>
            <td>
                <b>Optional</b>
                <p>Path to Kustomization files..</p>
                <p>The default value is `.` (current directory).</p>
            </td>
        </tr>
        <tr>
            <td><code>flags</code></td>
            <td>
                OPTIONAL
                <p>Additional flags to pass to <code>kubectl</code>.</p>
                <p>You can specify three types of flags:</p>
                <ul>
                    <li><code>global</code>: flags that apply to every command.</li>
                    <li><code>apply</code>: flags that apply to creation commands.</li>
                    <li><code>delete</code>: flags that apply to deletion commands.</li>
                <ul>
            </td>
        </tr>
    </tbody>
<table>

The following `deploy` section, for example, instructs Skaffold to deploy
artifacts using Kustomize:

```
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
