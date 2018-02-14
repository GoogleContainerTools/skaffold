skaffold
=============

[![Build Status](https://travis-ci.com/GoogleCloudPlatform/skaffold.svg?token=NyoV8n1D3L8EzmetKFNB&branch=master)](https://travis-ci.com/GoogleCloudPlatform/skaffold)

Skaffold is a tool that makes the onboarding of existing applications to Kubernetes simple and repeatable. It currently handles the build and deploy lifecycle actions for Kubernetes. Skaffold has a simple, pluggable architecture that allows you to use some of the best community tooling available, such as `docker` or `helm`.

Some of the main features
* No server-side component. No additional overhead to your cluster.
* Supports existing tooling and workflows. Build and deploy APIs make each implementation composable to support all types of workflows.
* Multi-image and multi-manifest support.
* Switch from development mode to run mode with no configuration changes

Skaffold has two main modes: `skaffold run` and `skaffold dev`

## skaffold dev
This is the development mode for skaffold.

* A filesystem watcher to watch the dependencies of your docker images
* Streaming logs from deployed containers
* Continuous build-deploy loop, warn on errors

Use for:
* Local development

## skaffold run
Run runs a skaffold pipeline once, exiting on any errors in the pipeline.

Use for:
* Continuous Integration or Continuous Deployment tools

## Getting Started

### Prerequisites

What you'll need installed

* **skaffold**
    * We don't ship a binary yet, so you'll have to build it yourself!
    * Using `go >= 1.9`, run `make install` in the root directory to install `skaffold` to your `$GOBIN`.
* **Kubernetes Cluster**
* **kubectl**
  * configured with the current-context of your target cluster
  * In the future, we'll support more deployment strategies and drop this dependency
* **docker**
    * In the future, we'll support more build strategies and drop this dependency
* **Docker Image Repository**
    * Your docker client should be configured to push to an external docker image repository.  If you're using a minikube cluster, you can skip this requirement.

If you're using minikube you'll only need
* **skaffold**
* **minikube**
* **kubectl**

### Development

To get started, change the `imageName` and `IMAGE_NAME` parameters of `examples/getting-started/skaffold.yaml`.  This should be a fully qualified image name that your docker client is configured to push to.

From the root directory of this repository,

```shell
$ skaffold dev -f examples/getting-started/skaffold.yaml
```

You should see the output (for verbose output, append `-v debug`)

```shell
Skaffold v0.1.0
Starting build...
Sending build context to Docker daemon   7.68kB
Step 1/5 : FROM golang:1.9.2
 ---> 138bd936fa29
Step 2/5 : WORKDIR /go/src/github.com/GoogleCloudPlatform/skaffold/examples/getting-started
 ---> Using cache
 ---> bd3002e4d850
Step 3/5 : COPY main.go .
 ---> Using cache
 ---> 7dcfea090d8f
Step 4/5 : RUN go build -o app main.go
 ---> Using cache
 ---> a69a1073e6da
Step 5/5 : CMD ./app
 ---> Using cache
 ---> 5c6041e4ee28
Successfully built 5c6041e4ee28
Successfully tagged b93e205a1a6d9da76ccb0b4a65f2df16:latest
Successfully tagged changeme:5c6041e4ee28b49ffca084eb25fa95e580b83992754d65ec60f0e90df6ee2f98
INFO[0000] Starting deploy...
INFO[0000] Deploying examples/getting-started/k8s-deployment.yaml
INFO[0000] Found dependencies for dockerfile [examples/getting-started/main.go]
INFO[0000] Added watch for examples/getting-started/Dockerfile
INFO[0000] Added watch for examples/getting-started/main.go

```

At this point, you should be able to see some output from the pod with kubectl

```shell
$ kubectl logs getting-started
Hello world!
Hello world!
...
```

Now, lets update `examples/getting-started/main.go`

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

 Now you should see the application has updated

 ```
$ kubectl logs getting-started
Hello jerry!
Hello jerry!
...
```

### Run pipeline once

There may be some cases where you don't want to run build and deploy continuously. To run once, use

```shell
$ skaffold run -f examples/getting-started/skaffold.yaml
```

### Docker commands

Skaffold exposes some the of the dockerfile introspection functionality that it uses under the hood.

#### skaffold docker deps

Many build systems require the developer to manually list the dependencies of a dockerfile.  Skaffold can inspect a dockerfile and output the file dependencies.  

```
$ skaffold docker deps --context examples/getting-started -v error
examples/getting-started/main.go
```

**Example Makefile rule for conditional docker image rebuilds**
```Makefile
out/getting-started: $(shell skaffold docker deps -c examples/getting-started -v error)
	docker build -t getting-started . -q > out/getting-started
```


#### skaffold docker context

This command generates the context tarball for the given dockerfile context. 
