# Skaffold

[![Build Status](https://travis-ci.org/GoogleContainerTools/skaffold.svg?branch=master)](https://travis-ci.org/GoogleContainerTools/skaffold)

Skaffold is a command line tool that facilitates continuous development for Kubernetes applications. You can iterate on your 
application source code locally then deploy to local or remote Kubernetes clusters. Skaffold handles the workflow for building,
pushing and deploying your application. It can also be used in an automated context such as a CI/CD pipeline to leverage the same 
workflow and tooling when moving applications to production.

- [Skaffold](#skaffold)
  - [Features](#features)
  - [Pluggability](#pluggability)
- [Operating modes](#operating-modes)
  - [skaffold dev](#skaffold-dev)
  - [skaffold run](#skaffold-run)
- [Demo](#demo)  
- [Getting Started with Local Tooling](#getting-started-with-local-tooling)
  - [Installation](#installation)
  - [Iterative Development](#iterative-development)
  - [Run a deployment pipeline once](#run-a-deployment-pipeline-once)
- [Future](#future)
- [Community](#community)

### Features
-  No server-side component. No overhead to your cluster.
-  Detect changes in your source code and automatically build/push/deploy.
-  Image tag management. Stop worrying about updating the image tags in Kubernetes manifests to push out changes during development.
-  Supports existing tooling and workflows. Build and deploy APIs make each implementation composable to support many different workflows.
-  Support for multiple application components. Build and deploy only the pieces of your stack that have changed.
-  Deploy regularly when saving files or run one off deployments using the same configuration.

### Pluggability
Skaffold has a pluggable architecture that allows you to choose the tools in the developer workflow that work best for you.
![Plugability Diagram](docs/img/plugability.png)

## Operating modes
### skaffold dev
Updates your deployed application continually:
-  Watches your source code and the dependencies of your docker images for changes and runs a build and deploy when changes are detected
-  Streams logs from deployed containers
-  Continuous build-deploy loop, only warn on errors

### skaffold run
Runs a Skaffold pipeline once, exits on any errors in the pipeline.  
Use for:
-  Continuous integration or continuous deployment pipelines
-  Sanity checking after iterating on your application

## Demo

![Demo](/docs/img/intro.gif)

## Getting Started with Local Tooling

For getting started with Google Kubernetes Engine and Container Builder [go here](docs/quickstart-gke.md). Otherwise continue
below to get started with a local Kubernetes cluster.

### Installation

You will need the following components to get started with Skaffold:

1. skaffold
   -  To download the latest Linux build, run:
      -  `curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`
   -  To download the latest OSX build, run:
      -  `curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

1. Kubernetes Cluster
   -  [Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/),
      [GKE](https://cloud.google.com/kubernetes-engine/docs/how-to/creating-a-container-cluster),
      [Docker for Mac (Edge)](https://docs.docker.com/docker-for-mac/install/) and [Docker for Windows (Edge)](https://docs.docker.com/docker-for-windows/install/)
      have been tested but any Kubernetes cluster will work.

1. [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
   -  If you're not using Minikube, configure the current-context with your target cluster for development

1. docker

1. Docker image registry
   -  Your docker client should be configured to push to an external docker image repository. If you're using a minikube or Docker for Desktop cluster, you can skip this requirement.
   -  If you are using Google Container Registry (GCR), choose one of the following:
        1. Use `gcloud`'s Docker credential helper: Run [`gcloud auth configure-docker`](https://cloud.google.com/sdk/gcloud/reference/auth/configure-docker)
        1. Install and configure GCR's standalone cred helper: [`docker-credential-gcr`](https://github.com/GoogleCloudPlatform/docker-credential-gcr#installation-and-usage)
        1. Run `gcloud docker -a` before each development session.

### Iterative Development

1. Clone this repostiory to get access to the examples.

    ```shell
    git clone https://github.com/GoogleContainerTools/skaffold
    ```

1. Change directories to the `getting-started` example.

    ```shell
    cd examples/getting-started
    ```

1. In the skaffold.yaml file, update the `build.artifacts.imageName` stanza to the `<registry/image:tag>` registry where your image will be pushed. For example `gcr.io/<your-project-ID>/your-image`, where `gcr.io/<your-project-ID>` is your GCP container registry location. As mentioned earlier, your docker client should be configured to push to this location.

1. Run `skaffold dev`.

    ```console
    $ skaffold dev
    Starting build...
    Found [minikube] context, using local docker daemon.
    Sending build context to Docker daemon  6.144kB
    Step 1/5 : FROM golang:1.9.4-alpine3.7
     ---> fb6e10bf973b
    Step 2/5 : WORKDIR /go/src/github.com/GoogleContainerTools/skaffold/examples/getting-started
     ---> Using cache
     ---> e9d19a54595b
    Step 3/5 : CMD ./app
     ---> Using cache
     ---> 154b6512c4d9
    Step 4/5 : COPY main.go .
     ---> Using cache
     ---> e097086e73a7
    Step 5/5 : RUN go build -o app main.go
     ---> Using cache
     ---> 9c4622e8f0e7
    Successfully built 9c4622e8f0e7
    Successfully tagged 930080f0965230e824a79b9e7eccffbd:latest
    Successfully tagged gcr.io/k8s-skaffold/skaffold-example:9c4622e8f0e7b5549a61a503bf73366a9cf7f7512aa8e9d64f3327a3c7fded1b
    Build complete in 657.426821ms
    Starting deploy...
    Deploying k8s-pod.yaml...
    Deploy complete in 173.770268ms
    [getting-started] Hello world!
    ```

1. Skaffold has done the following for you:

   - Build an image from the local source code
   - Tag it with its sha256
   - Sets that image in the Kubernetes manifests defined in `skaffold.yaml`
   - Deploy the Kubernetes manifests using `kubectl apply -f`

1. You will see the output of the pod that was deployed:

    ```console
    [getting-started] Hello world!
    [getting-started] Hello world!
    [getting-started] Hello world!
    ```

Now, update `main.go`

```diff
diff --git a/examples/getting-started/main.go b/examples/getting-started/main.go
index 64b7bdfc..f95e053d 100644
--- a/examples/getting-started/main.go
+++ b/examples/getting-started/main.go
@@ -7,7 +7,7 @@ import (

 func main() {
        for {
-               fmt.Println("Hello world!")
+               fmt.Println("Hello jerry!")
                time.Sleep(time.Second * 1)
        }
 }
```

Once you save the file, you should see the pipeline kick off again to redeploy your application:
```console
[getting-started] Hello jerry!
[getting-started] Hello jerry!
```

### Run a deployment pipeline once
There may be some cases where you don't want to run build and deploy continuously. To run once, use:
```console
$ skaffold run
```

### More examples

* [Deploying with Helm](./examples/helm-deployment)
* [Microservices/Multiple applications](./examples/microservices)
* [Deploying with no Kubernetes manifests](./examples/no-manifest)

## Community
- [skaffold-users mailing list](https://groups.google.com/forum/#!forum/skaffold-users)
- [#skaffold on Kubernetes Slack](https://kubernetes.slack.com/messages/CABQMSZA6/)
