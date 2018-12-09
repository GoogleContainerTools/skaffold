
---
title: "skaffold.yaml References"
linkTitle: "skaffold.yaml References"
weight: 120
---

This page discusses the Skaffold configuration file, `skaffold.yaml`.
As an alternative, you can also refer to [annotated-skaffold.yaml](https://github.com/GoogleContainerTools/skaffold/blob/master/examples/annotated-skaffold.yaml), a self-contained config reference. 

`skaffold.yaml` has 4 components:

* [Metadata](#metadata) (`apiVersion` and `kind`)
* [Build Configuration](#build-configuration-build) (`build`)
* [Deploy Configuration](#deploy-configuration-deploy) (`deploy`)
* [Profiles](#profiles-`profiles`)(`profiles`)

The following example showcases a `skaffold.yaml` that uses API version
{{< skaffold-version >}}, builds the artifact `gcr.io/k8s-skaffold/skaffold-example`
with local Docker daemon, and deploys it to Kubernetes with `kubectl`
using the Kubernetes manifest `k8s-pod.yaml`. It also includes a profile
for using Google Cloud Build, `gcb`.

```yaml
apiVersion: skaffold/v1beta1
kind: Config
build:
    artifacts:
    - imageName: gcr.io/k8s-skaffold/skaffold-example
    local: {}
deploy:
    kubectl:
    manifests:
        - k8s-pod
profiles:
    - name: gcb
      build:
        googleCloudBuild:
            projectId: k8s-skaffold
```
## Metadata 
### API Version (`apiVersion`)

API Version specifies the version of Skaffold API you would like to use. 
Latest version is {{< skaffold-version >}}.

Different versions require different schemas of the `skaffold.yaml` file.

{{% todo 1060 "to be updated to latest version" %}}

### Kind (`kind`)

The Skaffold configuration file has the kind `Config`.

## Build Configuration (`build`)

The `build` section has three parts:

|Stanza|Description|
|-----|-----|
|Artifacts (`artifacts`)| A list of artifacts to build. See the Artifact section below for more information.|
|Tag Policy (`tagPolicy`)|The tag policy Skaffold uses to tag artifacts. See [Using Taggers](/docs/how-tos/tagger) for more information.|            
|Build Type| Specifies which tool Skaffold should use for building artifacts. Skaffold supports using local Docker daemon, Google Cloud Build, Kaniko, or Bazel to build artifacts. See [Using Builders](/docs/how-tos/builders) for more information. |

### Artifacts (`artifacts`)

Each artifact item has the following three fields:

|Field|Description|
|---|---|
|Image Name (`imageName`)| <b>Required</b> The name of the artifact, e.g. `grc.io/k8s-skaffold/skaffold-example`. |            
|Workspace (`workspace`)| Optional.The Docker workspace. See [Using Taggers](/docs/how-tos/taggers/) for more information.|
|Artifact Type| Optional. There are two available artifact types: Docker Artifact (`docker`) and Bazel Artifact (`bazel`). Both types offers additional parameters that you can configure. Default value is `docker: {}`|

The Docker Artifact type features the following parameters:

|Field|Description|
|---|---|
| Dockerfile Path (`dockerfilePath`)| Optional.  Path to the Dockerfile.|
|Build Args (`buildArgs`)|Optional. Arguments to be passed to the Docker daemon.|
|Cache From (`cacheFrom`)|A list of images used as a cache source on build. See [Docker Documentation: docker build Command](https://docs.docker.com/edge/engine/reference/commandline/build/) for more information.|
|Target (`target`)|Set the target build stage to build .See [Docker Documentation: docker build Command](https://docs.docker.com/edge/engine/reference/commandline/build/) for more information.|

The following example showcases a `build` section that builds two artifacts,
`gcr.io/k8s-skaffold/skaffold-example-1` and `gcr.io/k8s-skaffold/skaffold-example-2`:

```yaml
build:
    artifacts:
    - imageName: gcr.io/k8s-skaffold/skaffold-example-1
      docker:
        dockerfilePath: DOCKERFILE-PATH
        buildArgs:
            SOME-ARG: SOME-VALUE
            SOME-MORE-ARG: SOME-MORE-VALUE
        cacheFrom:
        - IMAGE-AS-CACHE
        - IMAGE-AS-CACHE
        target: TARGET
    - imageName: gcr.io/k8s-skaffold/skaffold-example-2
      docker:
        dockerfilePath: DOCKERFILE-PATH
        buildArgs:
            SOME-ARG: SOME-VALUE
            SOME-MORE-ARG: SOME-MORE-VALUE
        cacheFrom:
        - IMAGE-AS-CACHE
        - IMAGE-AS-CACHE
        target: TARGET
    local: {}
```

And the Bazel Artifact type features the following parameters:

|Field|Description|
|---|---|
|Build Target (`target`)| <b>Required</b> The Bazel build target.|

## Deploy Configuration (`deploy`)

See [Using Deployers](/docs/how-tos/deployers) for more information.

## Profiles (`profiles`)

See [Using Profiles](/docs/how-tos/profiles) for more information.
