---
title: "Quickstart"
linkTitle: "Quickstart"
weight: 20
---

Follow this tutorial to learn about Skaffold on a small Kubernetes app built with [Docker](https://www.docker.com/) inside [minikube](https://minikube.sigs.k8s.io)
and deployed with [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)! 

{{< alert title="Note">}}
Aside from `Docker` and `kubectl`, Skaffold also supports a variety of other tools
and workflows; see [Tutorials]({{<relref "/docs/tutorials">}}) for
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

1. Change to the `examples/getting-started` in skaffold directory.

    ```bash
    cd skaffold/examples/getting-started
    ```

## `skaffold dev`: continuous build & deploy on code changes

Run `skaffold dev` to build and deploy your app continuously.
You should see some outputs similar to the following entries:

```
Listing files to watch...
 - skaffold-example
Generating tags...
 - skaffold-example -> skaffold-example:v1.1.0-113-g4649f2c16
Checking cache...
 - skaffold-example: Not found. Building
Found [docker-desktop] context, using local docker daemon.
Building [skaffold-example]...
Sending build context to Docker daemon  3.072kB
Step 1/6 : FROM golang:1.12.9-alpine3.10 as builder
 ---> e0d646523991
Step 2/6 : COPY main.go .
 ---> Using cache
 ---> e4788ffa88e7
Step 3/6 : RUN go build -o /app main.go
 ---> Using cache
 ---> 686396d9e9cc
Step 4/6 : FROM alpine:3.10
 ---> 965ea09ff2eb
Step 5/6 : CMD ["./app"]
 ---> Using cache
 ---> be0603b9d79e
Step 6/6 : COPY --from=builder /app .
 ---> Using cache
 ---> c827aa5a4b12
Successfully built c827aa5a4b12
Successfully tagged skaffold-example:v1.1.0-113-g4649f2c16
Tags used in deployment:
 - skaffold-example -> skaffold-example:c827aa5a4b12e707163842b803d666eda11b8ec20c7a480198960cfdcb251042
   local images can't be referenced by digest. They are tagged and referenced by a unique ID instead
Starting deploy...
 - pod/getting-started created
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

For getting started with your project, see the [Getting Started With Your Project]({{<relref "/docs/workflows/getting-started-with-your-project" >}}) workflow.

For more in-depth topics of Skaffold, explore [Configuration]({{< relref "/docs/design/config.md" >}}),
[Skaffold Pipeline]({{<relref "/docs/pipeline-stages" >}}), and [Architecture and Design]({{< relref "/docs/design" >}}).

To learn more about how Skaffold builds, tags, and deploys your app, see the How-to Guides on
using [Builders]({{<relref "/docs/pipeline-stages/builders" >}}), [Taggers]({{< relref "/docs/pipeline-stages/taggers">}}), and [Deployers]({{< relref "/docs/pipeline-stages/deployers" >}}).

[Skaffold Tutorials]({{< relref "/docs/tutorials" >}}) details some of the common use cases of Skaffold.

:mega: **Please fill out our [quick 5-question survey](https://forms.gle/BMTbGQXLWSdn7vEs6)** to tell us how satisfied you are with Skaffold, and what improvements we should make. Thank you! :dancers:
