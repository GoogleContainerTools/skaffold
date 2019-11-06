---
title: "Building Artifacts with a Custom Build Script"
linkTitle: "Custom Build Script"
weight: 100
---

This page describes building Skaffold artifacts using a custom build script, which builds images using [buildpacks](https://buildpacks.io/).
Buildpacks enable building language-based containers from source code, without the need for a Dockerfile.

## Before you begin
First, you will need to have Skaffold and a Kubernetes cluster set up.
To learn more about how to set up Skaffold and a Kubernetes cluster, see the [quickstart docs]({{< relref "/docs/quickstart" >}}).

For this tutorial, to use buildpacks as a custom builder with Skaffold, please install the following additional tools:

* [pack](https://buildpacks.io/docs/install-pack/)
* [docker](https://docs.docker.com/install/)


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

## Tutorial - Hello World in Go

This tutorial will be based on the [buildpacks example](https://github.com/GoogleContainerTools/skaffold/tree/master/examples/buildpacks) in our repository.


## Adding a Custom Builder to Your Skaffold Project

We'll need to configure your Skaffold config to build artifacts with this custom builder.
To do this, we will take advantage of the [custom builder]({{<relref "docs/pipeline-stages/builders#custom-build-script-run-locally" >}}) in Skaffold.

First, add a `build.sh` file which Skaffold will call to build artifacts:

{{% readfile file="samples/builders/custom-buildpacks/build.sh" %}}


Then, configure artifacts in your `skaffold.yaml` to build with `build.sh`: 

{{% readfile file="samples/builders/custom-buildpacks/skaffold.yaml" %}}

List the file dependencies for each artifact; in the example above, Skaffold watches all files in the build context.
For more information about listing dependencies for custom artifacts, see the documentation [here]({{<relref "docs/pipeline-stages/builders#getting-dependencies-from-a-command" >}}).


You can check custom builder is properly configured by running `skaffold build`.
This command should build the artifacts and exit successfully.


