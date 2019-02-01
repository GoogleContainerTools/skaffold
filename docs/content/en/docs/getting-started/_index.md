
---
title: "Getting Started"
linkTitle: "Getting Started"
weight: 10
---

This document showcases how to get started with Skaffold using [Docker](https://www.docker.com/)
and Kubernetes command-line tool, [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/).
Aside from Docker and kubectl, Skaffold also supports a variety of other tools
and workflows; see [How-to Guides](/docs/how-tos) and [Tutorials](/docs/tutorials) for
more information.

In this quickstart, you will:

* Install Skaffold
* Download a sample go app
* Use `skaffold dev` to build and deploy your app every time your code changes
* Use `skaffold run` to build and deploy your app once, on demand

## Before you begin

* [Install Docker](https://www.docker.com/get-started)
* [Install kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* Configure kubectl to connect to a Kubernetes cluster. You can use
    * any Kubernetes platform with Skaffold; see [Picking the Right Solution](https://kubernetes.io/docs/setup/pick-right-solution/)
    from Kubernetes documentation for instructions on choosing the
    right platfrom.
    * [Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine/)
    is a hosted Kubernetes solution. To set up kubectl with Google Kubernetes Engine,
    see [Kubernetes Engine Quickstart](https://cloud.google.com/kubernetes-engine/docs/quickstart).
    * [Minikube](https://kubernetes.io/docs/setup/minikube/) is
    a local Kubernetes solution best for development and testing. To set up
    kubectl with Minikube, see [Installing Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/).</p>

{{< alert title="Note" >}}
If you use a non-local solution, your Docker client needs to be configured
to push Docker images to an external Docker image registry. For setting up
Docker with Google Container Registry, see <a href=https://cloud.google.com/container-registry/docs/quickstart>Google Container Registry Quickstart</a>.
{{< /alert >}}

## Installing Skaffold

{{% tabs %}}
{{% tab "LINUX" %}}
### Stable binary 
For the latest **stable** release download and place it in your `PATH`: 

https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-amd64 

Run these commands to download and place the binary in your /usr/local/bin folder: 
 
```bash
curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-amd64
chmod +x skaffold
sudo mv skaffold /usr/local/bin
```

### Latest bleeding edge binary

For the latest **bleeding edge** build, download and place it in your `PATH`: 

https://storage.googleapis.com/skaffold/builds/latest/skaffold-linux-amd64

Run these commands to download and place the binary in your /usr/local/bin folder:

```bash
curl -Lo skaffold https://storage.googleapis.com/skaffold/builds/latest/skaffold-linux-amd64
chmod +x skaffold
sudo mv skaffold /usr/local/bin
```


{{% /tab %}}

{{% tab "MACOS" %}}

### Homebrew

```bash
brew install skaffold
```

### Stable binary
For the latest **stable** release download and place it in your `PATH`: 

https://storage.googleapis.com/skaffold/releases/latest/skaffold-darwin-amd64 

Run these commands to download and place the binary in your /usr/local/bin folder: 
 
```bash
curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-darwin-amd64
chmod +x skaffold
sudo mv skaffold /usr/local/bin
```

### Bleeding edge binary

For the latest **bleeding edge** build, download and place it in your `PATH`: 

https://storage.googleapis.com/skaffold/builds/latest/skaffold-darwin-amd64

Run these commands to download and place the binary in your /usr/local/bin folder:

```bash
curl -Lo skaffold https://storage.googleapis.com/skaffold/builds/latest/skaffold-darwin-amd64
chmod +x skaffold
sudo mv skaffold /usr/local/bin
```
{{% /tab %}}


{{% tab "WINDOWS" %}}

### Chocolatey 

```bash
choco install skaffold
```

### Stable binary

For the latest **stable** release download and place it in your `PATH`: 

https://storage.googleapis.com/skaffold/releases/latest/skaffold-windows-amd64.exe 

For the latest **bleeding edge** build, download and place it in your `PATH`: 

https://storage.googleapis.com/skaffold/builds/latest/skaffold-windows-amd64.exe 


{{% /tab %}}
{{% /tabs %}}

## Downloading the sample app

1. Clone the Skaffold repository:

    ```
    git clone https://github.com/GoogleContainerTools/skaffold
    ```
1. Change to the `examples/getting-started` directory.

    ```
    cd examples/getting-started
    ```
  
## `skaffold dev`: Build and deploy your app every time your code changes

Run command `skaffold dev` to build and deploy your app continuously. You should
see some outputs similar to the following entries:

```bash
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

`skaffold dev` monitors your code repository and perform a Skaffold workflow
every time a change is detected. `skaffold.yaml` provides specifications of the
workflow, which, in this example, is

* Building a Docker image from the source using the Dockerfile
* Tagging the Docker image with the Sha256 Hash of its contents
* (If you use a hosted Kubernetes solution) Pushing the Docker image to the
  external Docker image registry
* Updating the Kubernetes manifest, `k8s-pod.yaml`, to use the image built previously
* Deploying the Kubernetes manifest using `kubectl apply -f`
* Streaming the logs back from the deployed app

{{< alert title="Note" >}}
For skaffold dev, if `imagePullPolicy` is set to `Always` in your Kubernetes manifest, it will expect the image to exist in a remote registry.
{{< /alert >}}


Let's re-trigger the workflow just by a single code change! 
Update `main.go` as follows:

```go
package main

import (
	"fmt"
	"time"
)

func main() {
	for {
		fmt.Println("Hello Skaffold!")
		time.Sleep(time.Second * 1)
	}
}
```

The moment you save the file, Skaffold will repeat the workflow described in
`skaffold.yaml` and eventually re-deploy your application. Once the pipeline
is completed, you should see updated outputs in the terminal:

```bash
[getting-started] Hello Skaffold!
```

<span style="font-size: 36pt">✨</span>

## `skaffold run`: Build and deploy your app once, on demand

If you prefer building and deploying once at a time, run command `skaffold run`.
Skaffold will perform the workflow described in `skaffold.yaml` exactly once.

## What's next

For more in-depth topics of Skaffold, explore [Skaffold Concepts: Configuration](/docs/concepts/#configuration),
[Skaffold Concepts: Workflow](/docs/concepts/#workflow), and [Skaffold Concepts: Architecture](/docs/concepts/#architecture).

To learn more about how Skaffold builds, tags, and deploys your app, see the How-to Guides on 
using [Builders](/docs/how-tos/builders), [Taggers](/docs/how-tos/taggers), and [Deployers](/docs/how-tos/deployers).

[Skaffold Tutorials](/docs/tutorials) details some of the common use cases of Skaffold.
