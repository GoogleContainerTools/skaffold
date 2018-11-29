
---
title: "Concepts"
linkTitle: "Concepts"
weight: 80
---

This document discusses some concepts that can help you develop a deep
understanding of Skaffold.


## Configuration of the Skaffold pipeline (skaffold.yaml)

You can configure Skaffold with the Skaffold configuration file,
`skaffold.yaml`. The configuration file should be placed in the root of your
project directory; when you run the `Skaffold` command, Skaffold will try to
read the configuration file from the current directory.

`skaffold.yaml` consists of five different components:

<table>
    <thead>
        <tr>
            <th>Component</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td>API Version (<code>apiVersion</code>)</td>
            <td>
                The Skaffold API version you would like to use.
                <p>The current API version is {{< skaffold-version >}}.</p>
            </td>
        </tr>
        <tr>
            <td>Kind (<code>kind</code>)</td>
            <td>
                The Skaffold configuration file has the kind `Config`.
            </td>
        </tr>
        <tr>
            <td>Build Configuration (<code>build</code>)</td>
            <td>
                Specifies how Skaffold should build artifacts. You have control over what tool Skaffold can use, how Skaffold tags artifacts and how Skaffold pushes artifacts.
                <p>At this moment Skaffold supports using local Docker daemon, Google Cloud Build, Kaniko, or Bazel to build artifacts.</p>
                <p>See <a href="/docs/how-tos/builders">Using Builders</a> and <a href="/docs/how-tos/taggers">Using Taggers</a> for more information.</p>
            </td>
        </tr>
        <tr>
            <td>Deploy Configuration (<code>deploy</code>)</td>
            <td>
                Specifies how Skaffold should deploy artifacts.
                <p>At this moment Skaffold supports using `kubectl`, Helm, or Kustomize to deploy artifacts.</p>
                <p>See <a href="/docs/how-tos/builders">Using Deployers</a> for more information.</p>
            </td>
        </tr>
        <tr>
            <td>Profiles (<code>profiles</code>)</td>
            <td>
                Profile is a set of settings that, when activated, overrides the current configuration.
                <p>You can use Profile to override the <code>build</code> and the <code>deploy</code> section.</p>
            </td>
        </tr>
    </tbody>
<table>

You can learn more about the syntax of `skaffold.yaml` at
[`skaffold.yaml References`](/docs/references/config).

## Workflow

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

## Image repository handling 

{{% todo 1327 %}}

Skaffold allows for automatically rewriting image names to your repository.
This way you can grab a skaffold project and just `skaffold run` it to deploy to your cluster.  
The way to achieve this is the `default-repo` functionality: 

1. Via `default-repo` flag
  
        skaffold dev --default-repo <myrepo> 
  
1. Via `SKAFFOLD_DEFAULT_REPO` environment variable

        SKAFFOLD_DEFAULT_REPO=<myrepo> skaffold dev  

1. Via skaffold's global config           

If there is no default image repository set, there is no automated image name rewriting. 

The following image name rewriting strategies are designed to be *conflict-free*:  

* if there are multiple users using the same repo, they won't overwrite each others images.
* the full namespace of the image is rewritten on top of the base so similar image names don't collide in the base namespace (e.g.: repo1/example and repo2/example would collide in the target_namespace/example without this)

Automated image name rewriting strategies are determined based on the target repository: 

* Target: gcr.io
  * **strategy**: 		concat unless prefix matches
  * **example1**: prefix doesn't match:
    
    ````
      example base: 	gcr.io/k8s-skaffold/skaffold-example1
      example target: 	gcr.io/myproject/myuser
      example result:
      gcr.io/myproject/myuser/gcr.io/k8s-skaffold/skaffold-example1
    ````	
  * **example2**: prefix matches:
    
    ```
      example base: 	gcr.io/k8s-skaffold/skaffold-example1
      example target: 	gcr.io/k8s-skaffold/myuser
      example result:
      gcr.io/k8s-skaffold/myuser/skaffold-example1	
    ```
* Target: not gcr.io
  * **strategy**: 		escape & concat & truncate to 256
  
    ```
     example base: 	gcr.io/k8s-skaffold/skaffold-example1
     example target: 	aws_account_id.dkr.ecr.region.amazonaws.com
     example result:  aws_account_id.dkr.ecr.region.amazonaws.com/gcr_io_k8s-skaffold_skaffold-example1
    ```


## Architecture

Skaffold features a pluggable architecture:

![architecture](/images/architecture.png)

The architecture allows you to use Skaffold with the tool you prefer. Skaffold
provides built-in support for the following tools:

* Build
  * Local Docker Daemon
  * Google Cloud Build
  * Kaniko
  * Bazel
* Deploy 
  * Kubernetes Command-Line Interface (`kubectl`)
  * Helm
  * Kustomize
* Taggers
  * Git tagger 
  * Sha256 tagger
  * Env Template tagger 
  * DateTime tagger
 
And you can combine the tools as you see fit in Skaffold. For experimental
projects, you may want to use local Docker daemon for building artifacts, and
deploy them to a Minikube local Kubernetes cluster with `kubectl`:

![workflow_local](/images/workflow_local.png)

However, for production sites, you might find it better to build with Google
Cloud Build and deploy using Helm:

![workflow_gcb](/images/workflow_gcb.png)

Skaffold also supports development profiles. You can specify multiple different
profiles in the configuration and use whichever best serves your need in the
moment without having to modify the configuration file. You can learn more about
profiles from [Using Profiles](/docs/how-tos/profiles).

## Operating modes

Skaffold provides two separate operating modes:

* `skaffold dev`, the continuous development mode, enables monitoring of the
    source repository, so that every time you make changes to the source code,
    Skaffold will build and deploy your application.
* `skaffold run`, the standard mode, instructs Skaffold to build and deploy
    your application exactly once. When you make changes to the source code,
    you will have to call `skaffold run` again to build and deploy your
    application.

Skaffold command-line interfact also provides other functionalities that may
be helpful to your project. For more information, see [CLI References](/docs/references/cli).
