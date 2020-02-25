---
title: "Building Artifacts with a Custom Build Script"
linkTitle: "Custom Build Script"
weight: 100
---

This page describes building Skaffold artifacts using a custom build script, which builds images using [ko](https://github.com/google/ko).
ko builds containers from Go source code, without the need for a Dockerfile or
even installing Docker.

## Before you begin

First, you will need to have Skaffold and a Kubernetes cluster set up.
To learn more about how to set up Skaffold and a Kubernetes cluster, see the [quickstart docs]({{< relref "/docs/quickstart" >}}).

## Tutorial - Hello World in Go

This tutorial will be based on the [custom example](https://github.com/GoogleContainerTools/skaffold/tree/master/examples/custom) in our repository.


## Adding a Custom Builder to Your Skaffold Project

We'll need to configure your Skaffold config to build artifacts with [ko](https://github.com/google/ko).
To do this, we will take advantage of the [custom builder]({{<relref "/docs/pipeline-stages/builders/custom" >}}) in Skaffold.

First, add a `build.sh` file which Skaffold will call to build artifacts:

{{% readfile file="samples/builders/custom-buildpacks/build.sh" %}}

Then, configure artifacts in your `skaffold.yaml` to build with `build.sh`: 

{{% readfile file="samples/builders/custom-buildpacks/skaffold.yaml" %}}

List the file dependencies for each artifact; in the example above, Skaffold watches all files in the build context.
For more information about listing dependencies for custom artifacts, see the documentation [here]({{<relref "/docs/pipeline-stages/builders/custom#dependencies-from-a-command" >}}).

You can check custom builder is properly configured by running `skaffold build`.
This command should build the artifacts and exit successfully.
