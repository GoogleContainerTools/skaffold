---
title: "Building Artifacts with CNCF Buildpacks"
linkTitle: "Buildpacks"
weight: 100
---

This page describes building Skaffold artifacts with [buildpacks](https://buildpacks.io/).
Buildpacks enable building language-based containers from source code, without the need for a Dockerfile.

## Before you begin
For this tutorial to work, you will need to have Skaffold and a Kubernetes cluster set up.
To learn more about how to set up Skaffold and a Kubernetes cluster, see the [getting started docs](https://skaffold.dev/docs/getting-started/).

To use buildpacks with Skaffold, please install the following additional tools:

* [pack](https://buildpacks.io/docs/install-pack/)
* [docker](https://docs.docker.com/install/)

## Tutorial - Hello World in Go

This tutorial will demonstrate how Skaffold can build a simple Hello World Go application with buildpacks and deploy it to a Kubernetes cluster.

First, clone the Skaffold [repo](https://github.com/GoogleContainerTools/skaffold) and navigate to the [buildpacks example](https://github.com/GoogleContainerTools/skaffold/tree/master/examples/buildpacks) for sample code:

```shell
$ git clone https://github.com/GoogleContainerTools/skaffold
$ cd skaffold/examples/buildpacks
```

Set the default buildpack to one that can build Go applications: 

``` shell
$ pack set-default-builder heroku/buildpacks
```



Take a look at the `build.sh` file, which uses `pack` to containerize source code with buildpacks:

```shell
$ cat build.sh
#!/bin/bash
set -e
images=$(echo $IMAGES | tr " " "\n")

for image in $images
do
    pack build $image
    if $PUSH_IMAGE
    then
        docker push $image
    fi
done
```

and the skaffold config, which configures artifact `gcr.io/k8s-skaffold/skaffold-example` to build with `build.sh`:

```yaml
$ cat skaffold.yaml
apiVersion: skaffold/v1beta14
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/skaffold-example
    custom:
      buildCommand: ./build.sh
      dependencies:
        paths:
        - .
deploy:
  kubectl:
    manifests:
      - k8s-*
```
For more information about how this works, see the Skaffold custom builder [documentation](https://skaffold.dev/docs/how-tos/builders/#custom-build-script-run-locally).

Now, use Skaffold to deploy this application to your Kubernetes cluster:

```shell
$ skaffold run --tail --default-repo <your repo>
```
With this command, Skaffold will build the `skaffold-example` artifact with buildpacks and deploy the application to Kubernetes.
You should be able to see "Hello, World!" printed every second in the Skaffold logs.

To clean up your Kubernetes cluster, run:

```shell
$ skaffold delete --default-repo <your repo>
```


## Adding Buildpacks to Your Skaffold Project

To use buildpacks with your own project, you must choose a buildpack image to build your artifacts.
To see a list of available buildpacks, run:

```shell
$ pack suggest-builders
```

Choose a buildpack from the list, making sure your chosen image can detect the runtime your project is written in.
Set your default buildpack:

```shell
$ pack set-default-builder <insert buildpack image here>
```

Now, configure your Skaffold config to build artifacts with buildpacks.
To do this, we will take advantage of the [custom builder](../builders) in Skaffold.

First, add a `build.sh` file which Skaffold will call to build artifacts:

{{% readfile file="samples/buildpacks/build.sh" %}}


Then, configure artifacts in your `skaffold.yaml` to build with `build.sh`: 

{{% readfile file="samples/buildpacks/skaffold.yaml" %}}

List the file dependencies for each artifact; in the example above, Skaffold watches all files in the build context.
For more information about listing dependencies for custom artifacts, see the documentation [here](../builders).


You can check buildpacks are properly configured by running `skaffold build`.
This command should build the artifacts and exit successfully.


