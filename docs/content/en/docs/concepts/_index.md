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
project directory; when you run the `skaffold` command, Skaffold will try to
read the configuration file from the current directory.

`skaffold.yaml` consists of five different components:

| Component  | Description |
| ---------- | ------------|
| `apiVersion` | The Skaffold API version you would like to use. The current API version is {{< skaffold-version >}}. |
| `kind`  |  The Skaffold configuration file has the kind `Config`.  |
| `build`  |  Specifies how Skaffold builds artifacts. You have control over what tool Skaffold can use, how Skaffold tags artifacts and how Skaffold pushes artifacts. Skaffold supports using local Docker daemon, Google Cloud Build, Kaniko, or Bazel to build artifacts. See [Builders](/docs/how-tos/builders) and [Taggers](/docs/how-tos/taggers) for more information. |
| `test` |  Specifies how Skaffold tests artifacts. Skaffold supports [container-structure-tests](https://github.com/GoogleContainerTools/container-structure-test) to test built artifacts. See [Testers](/docs/how-tos/testers) for more information. |
| `deploy` |  Specifies how Skaffold deploys artifacts. Skaffold supports using `kubectl`, `helm`, or `kustomize` to deploy artifacts. See [Deployers](/docs/how-tos/deployers) for more information. |
| `profiles`|  Profile is a set of settings that, when activated, overrides the current configuration. You can use Profile to override the `build`, `test` and `deploy` sections. |

You can [learn more](/docs/references/yaml) about the syntax of `skaffold.yaml`.

## Global configuration (~/.skaffold/config)

Some context specific settings can be configured in a global configuration file, defaulting to `~/.skaffold/config`. Options can be configured globally or for specific contexts.

The options are:

| Option | Type | Description |
| ------ | ---- | ----------- |
| `default-repo` | string | The image registry where images are published (See below). |
| `insecure-registries` | list of strings | A list of image registries that may be accesses without TLS. |
| `local-cluster` | boolean | If true, do not try to push images after building. By default, contexts with names `docker-for-desktop`, `docker-desktop`, or `minikube` are treated as local. |

For example, to treat any context as local by default:

```bash
skaffold config set --global local-cluster true
```

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

Skaffold allows for automatically rewriting image names to your repository.
This way you can grab a Skaffold project and just `skaffold run` it to deploy to your cluster.
The way to achieve this is the `default-repo` functionality:

1. Via `default-repo` flag

    ```bash
    skaffold dev --default-repo <myrepo>
    ```

1. Via `SKAFFOLD_DEFAULT_REPO` environment variable

    ```bash
    SKAFFOLD_DEFAULT_REPO=<myrepo> skaffold dev
    ```

1. Via Skaffold's global config

    ```bash
    skaffold config set default-repo <myrepo>
    ```

If Skaffold doesn't find `default-repo`, there is no automated image name rewriting.

The image name rewriting strategies are designed to be *conflict-free*:
the full image name is rewritten on top of the default-repo so similar image names don't collide in the base namespace (e.g.: repo1/example and repo2/example would collide in the target_namespace/example without this)

Automated image name rewriting strategies are determined based on the default-repo and the original image repository:

* default-repo does not begin with gcr.io
  * **strategy**: 		escape & concat & truncate to 256

    ```
     original image: 	gcr.io/k8s-skaffold/skaffold-example1
     default-repo:      aws_account_id.dkr.ecr.region.amazonaws.com
     rewritten image:   aws_account_id.dkr.ecr.region.amazonaws.com/gcr_io_k8s-skaffold_skaffold-example1
    ```

* default-repo begins with "gcr.io" (special case - as GCR allows for infinite deep image repo names)
  * **strategy**: concat unless prefix matches
  * **example1**: prefix doesn't match:

    ```
      original image: 	gcr.io/k8s-skaffold/skaffold-example1
      default-repo: 	gcr.io/myproject/myimage
      rewritten image:  gcr.io/myproject/myimage/gcr.io/k8s-skaffold/skaffold-example1
    ```
  * **example2**: prefix matches:

    ```
      original image: 	gcr.io/k8s-skaffold/skaffold-example1
      default-repo: 	gcr.io/k8s-skaffold
      rewritten image:  gcr.io/k8s-skaffold/skaffold-example1	
    ```
  * **example3**: shared prefix:

    ```
      original image: 	gcr.io/k8s-skaffold/skaffold-example1
      default-repo: 	gcr.io/k8s-skaffold/myimage
      rewritten image:  gcr.io/k8s-skaffold/myimage/skaffold-example1	
    ```
    
### Insecure image registries

During development you may be forced to push images to a registry that does not support HTTPS.
By itself, Skaffold will never try to downgrade a connection to a registry to plain HTTP.
In order to access insecure registries, this has to be explicitly configured per registry name.

There are several levels of granularity to allow insecure communication with some registry:

1. Per Skaffold run via the repeatable `--insecure-registry` flag

    ```bash
    skaffold dev --insecure-registry insecure1.io --insecure-registry insecure2.io
    ```
    
1. Per Skaffold run via `SKAFFOLD_INSECURE_REGISTRY` environment variable

    ```bash
    SKAFFOLD_INSECURE_REGISTRY='insecure1.io,insecure2.io' skaffold dev
    ```
    
1. Per project via the Skaffold pipeline config `skaffold.yaml`
    
    ```yaml
    build:
        insecureRegistries:
        - insecure1.io
        - insecure2.io
    ```

1. Per user via Skaffold's global config

    ```bash
    skaffold config set insecure-registries insecure1.io
    skaffold config set insecure-registries insecure2.io
    ```
    
    Note that multiple set commands _add_ to the existing list of insecure registries.
    To clear the list, run `skaffold config unset insecure-registries`.
    
Skaffold will join the lists of insecure registries, if configured via multiple sources.

## Architecture

Skaffold is designed with pluggability in mind:

![architecture](/images/architecture.png)

The architecture allows you to use Skaffold with the tool you prefer. Skaffold
provides built-in support for the following tools:

* **Build**
  * Dockerfile locally, in-cluster with kaniko or using Google Cloud Build
  * Bazel locally
  * Jib Maven and Jib Gradle locally or using Google Cloud Build
* **Test**
  * [container-structure-test](https://github.com/GoogleContainerTools/container-structure-test)
* **Tag**
  * Git tagger
  * Sha256 tagger
  * Env Template tagger
  * DateTime tagger
* **Deploy**
  * Kubernetes Command-Line Interface (`kubectl`)
  * [Helm](https://helm.sh/)
  * [kustomize](https://github.com/kubernetes-sigs/kustomize)

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
profiles from [Profiles](/docs/how-tos/profiles).

## Operating modes

Skaffold provides two separate operating modes:

* `skaffold dev`, the continuous development mode, enables monitoring of the
    source repository, so that every time you make changes to the source code,
    Skaffold will build and deploy your application.
* `skaffold run`, the standard mode, instructs Skaffold to build and deploy
    your application exactly once. When you make changes to the source code,
    you will have to call `skaffold run` again to build and deploy your
    application.

Skaffold command-line interface also provides other functionalities that may
be helpful to your project. For more information, see [CLI References](/docs/references/cli).

## Local development

Local development means that Skaffold can skip pushing built container images, because the images are already present where they are run.
For standard development setups such as `minikube` and `docker-for-desktop`, this works out of the box.

However, for non-standard local setups, such as [minikube](https://github.com/kubernetes/minikube/) with custom profile or [kind](https://github.com/kubernetes-sigs/kind), some extra configuration is necessary.
The essential steps are:

1. Ensure that Skaffold builds the images with the docker daemon, which also runs the containers.
2. Tell Skaffold to skip pushing images either by configuring

    ```yaml
    build:
      local:
        push: false
    ```
   
   or by marking a kubernetes context as local (see the following example).

For example, when running `minikube` with a custom profile, such as `minikube start -p my-profile`:

1. Set up the docker environment for Skaffold with `source <(minikube docker-env -p my-profile)`.
   This should set some environment variables for docker (check with `env | grep DOCKER`).
   It is important to do this in the same shell where Skaffold is executed.
   
2. Tell Skaffold that the kubernetes context `my-profile` refers to a local cluster with

    ```bash
    skaffold config set --kube-context my-profile local-cluster true
    ```
