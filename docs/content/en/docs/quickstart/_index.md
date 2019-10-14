---
title: "Quickstart"
linkTitle: "Quickstart"
weight: 10
---

Follow this tutorial to learn about Skaffold on a small Kubernetes app built with [Docker](https://www.docker.com/) inside [minikube](https://minikube.sigs.k8s.io)
and deployed with [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)! 

{{< alert title="Note">}}
Aside from <code>Docker</code> and <code>kubectl</code>, Skaffold also supports a variety of other tools
and workflows; see <a href="/docs/how-tos">How-to Guides</a> and <a href="/docs/tutorials">Tutorials</a> for
more information.
{{</alert>}}

In this quickstart, you will:

* Install Skaffold,
* Download a sample go app,
* Use `skaffold dev` to build and deploy your app every time your code changes,
* Use `skaffold run` to build and deploy your app once, similar to a CI/CD pipeline

## Before you begin

* [Install Skaffold]({{< relref "/docs/install" >}})
* [Install kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* [Install minikube](https://minikube.sigs.k8s.io/docs/start/)

{{< alert title="Note">}}
Skaffold will build the app using the Docker daemon hosted inside minikube. 
If you want to deploy against a different Kubernetes cluster, e.g. Kind, GKE clusters, you will have to install Docker to build this app.
{{</alert>}}

## Downloading the sample app

1. Clone the Skaffold repository:

    ```bash
    git clone https://github.com/GoogleContainerTools/skaffold
    ```

1. Change to the `examples/getting-started` directory.

    ```bash
    cd examples/getting-started
    ```

## `skaffold dev`: continuous build & deploy on code changes

Run `skaffold dev` to build and deploy your app continuously.
You should see some outputs similar to the following entries:

```
Listing files to watch...
 - gcr.io/k8s-skaffold/skaffold-example
List generated in 2.46354ms
Generating tags...
 - gcr.io/k8s-skaffold/skaffold-example -> gcr.io/k8s-skaffold/skaffold-example:v0.39.0-131-g1759410a7-dirty
Tags generated in 65.661438ms
Starting build...
Found [minikube] context, using local docker daemon.
Building [gcr.io/k8s-skaffold/skaffold-example]...
Sending build context to Docker daemon  3.072kB
Step 1/6 : FROM golang:1.12.9-alpine3.10 as builder
 ---> e0d646523991
Step 2/6 : COPY main.go .
 ---> Using cache
 ---> 2d4b0b8a9dda
Step 3/6 : RUN go build -o /app main.go
 ---> Using cache
 ---> 3eae8e329453
Step 4/6 : FROM alpine:3.10
 ---> 961769676411
Step 5/6 : CMD ["./app"]
 ---> Using cache
 ---> ce76e22da3bd
Step 6/6 : COPY --from=builder /app .
 ---> dec4a50e0fd1
Successfully built dec4a50e0fd1
Successfully tagged gcr.io/k8s-skaffold/skaffold-example:v0.39.0-131-g1759410a7-dirty
Build complete in 232.935849ms
Starting test...
Test complete in 4.189µs
Tags used in deployment:
 - Since images are not pushed, they can't be referenced by digest
   They are tagged and referenced by a unique ID instead
 - gcr.io/k8s-skaffold/skaffold-example -> gcr.io/k8s-skaffold/skaffold-example:dec4a50e0fd1ca2f56c6aad2a6c6e1d3806e5f6bd8aa2751e0a10db0d46faaba
Starting deploy...
 - pod/getting-started created
Deploy complete in 374.060415ms
Watching for changes...
[getting-started] Hello world!
[getting-started] Hello world!
[getting-started] Hello world!

```

`skaffold dev` watches your local source code and executes your Skaffold pipeline
every time a change is detected. `skaffold.yaml` provides specifications of the
workflow - in this example, the pipeline is

* Building a Docker image from the source using the Dockerfile
* Tagging the Docker image with the `sha256` hash of its contents
* Updating the Kubernetes manifest, `k8s-pod.yaml`, to use the image built previously
* Deploying the Kubernetes manifest using `kubectl apply -f`
* Streaming the logs back from the deployed app

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

When you save the file, Skaffold will see this change and repeat the workflow described in
`skaffold.yaml`, rebuilding and redeploying your application. Once the pipeline
is completed, you should see the changes reflected in the output in the terminal:

```
[getting-started] Hello Skaffold!
```

<span style="font-size: 36pt">✨</span>

## `skaffold run`: build & deploy once 

If you prefer building and deploying once at a time, run `skaffold run`.
Skaffold will perform the workflow described in `skaffold.yaml` exactly once.

## What's next

For more in-depth topics of Skaffold, explore [Skaffold Concepts: Configuration]({{< relref "/docs/concepts#configuration" >}}),
[Skaffold Concepts: Workflow](/docs/concepts#workflow), and [Skaffold Concepts: Architecture]({{< relref "/docs/concepts#architecture" >}}).

To learn more about how Skaffold builds, tags, and deploys your app, see the How-to Guides on
using [Builders](/docs/how-tos/builders), [Taggers](/docs/how-tos/taggers), and [Deployers]({{< relref "/docs/how-tos/deployers" >}}).

[Skaffold Tutorials]({{< relref "/docs/tutorials" >}}) details some of the common use cases of Skaffold.
