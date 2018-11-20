
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

### Kind (`kind`)

The Skaffold configuration file has the kind `Config`.

## Build Configuration (`build`)

The `build` section has three parts:

<table>
    <thead>
        <tr>
            <th>Part</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td>Artifacts (<code>artifacts</code>)</td>
            <td>
                A list of artifacts to build.
                <p>See the Artifact section below for more information.</p>
            </td>
        </tr>
        <tr>
            <td>Tag Policy (<code>tagPolicy</code>)</td>
            <td>
                The tag policy Skaffold uses to tag artifacts.
                <p>See [Using Taggers](/docs/how-tos/tagger) for more information.</p>
            </td>
        </tr>
        <tr>
            <td>Build Type</td>
            <td>
                Specifies which tool Skaffold should use for building artifacts.
                <p>At this moment Skaffold supports using local Docker daemon, Google Cloud Build, Kaniko, or Bazel to build artifacts.</p>
                <p>See <a href="/docs/how-tos/builders">Using Builders</a> for more information.</p>
            </td>
        </tr>
    </tbody>
<table>

### Artifacts (`artifacts`)

Each artifact item has the following three fields:

<table>
    <thead>
        <tr>
            <th>Field</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td>Image Name (<code>imageName</code>)</td>
            <td>
                <b>Required</b>
                <p>The name of the artifact, e.g. <code>grc.io/k8s-skaffold/skaffold-example</code>.</p>
            </td>
        </tr>
        <tr>
            <td>Workspace (<code>workspace</code>)</td>
            <td>
                Optional
                <p>The Docker workspace.</p>
                <p>See [Using Taggers](/docs/how-tos/taggers/) for more information.</p>
            </td>
        </tr>
        <tr>
            <td>Artifact Type</td>
            <td>
                Optional
                <p>There are two available artifact types: Docker Artifact (<code>docker</code>) and Bazel Artifact (<code>bazel</code>).</p>
                <p>Both types offers additional parameters that you can configure.</p>
                <p>Default value is <code>docker: {}</code></p>
            </td>
        </tr>
    </tbody>
<table>

The Docker Artifact type features the following parameters:

<table>
    <thead>
        <tr>
            <th>Field</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td>Dockerfile Path (<code>dockerfilePath</code>)</td>
            <td>
                Optional
                <p>Path to the Dockerfile.</p>
            </td>
        </tr>
        <tr>
            <td>Build Args (<code>buildArgs</code>)</td>
            <td>
                Optional
                <p>Arguments to be passed to the Docker daemon.</p>
            </td>
        </tr>
        <tr>
            <td>Cache From (<code>cacheFrom</code>)</td>
            <td>
                A list of images used as a cache source on build.
                <p>See <a href="https://docs.docker.com/edge/engine/reference/commandline/build/">Docker Documentation: docker build Command</a> for more information.</p>
            </td>
        </tr>
        <tr>
            <td>Target (<code>target</code>)</td>
            <td>
                Set the target build stage to build.
                <p>See <a href="https://docs.docker.com/edge/engine/reference/commandline/build/">Docker Documentation: docker build Command</a> for more information.</p>
            </td>
        </tr>
    </tbody>
<table>

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

<table>
    <thead>
        <tr>
            <th>Field</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td>Build Target (<code>target</code>)</td>
            <td>
                <b>Required</b>
                <p>The Bazel build target.</p>
            </td>
        </tr>
    </tbody>
<table>

## Deploy Configuration (`deploy`)

See [Using Deployers](/docs/how-tos/deployers) for more information.

## Profiles (`profiles`)

See [Using Profiles](/docs/how-tos/profiles) for more information.
