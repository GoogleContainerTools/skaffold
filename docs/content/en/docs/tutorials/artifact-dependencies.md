---
title: "Defining dependencies between artifacts"
linkTitle: "Build Dependencies"
weight: 100
---

This page describes how to define dependencies between artifacts and reference them in the [docker builder]({{<relref "/docs/pipeline-stages/builders/docker" >}}).

## Before you begin

First, you will need to have Skaffold and a Kubernetes cluster set up.
To learn more about how to set up Skaffold and a Kubernetes cluster, see the [quickstart docs]({{< relref "/docs/quickstart" >}}).

## Tutorial - Simple artifact dependency

This tutorial will be based on the [simple-artifact-dependency](https://github.com/GoogleContainerTools/skaffold/tree/master/examples/simple-artifact-dependency) example in our repository.


## Adding an artifact dependency

We have a `base` artifact which has a single Dockerfile that we build with the [docker builder]({{<relref "/docs/pipeline-stages/builders/docker" >}}):
 {{% readfile file="samples/builders/artifact-dependencies/Dockerfile.base" %}}

This artifact is used as the base image for the `app` artifact. We express this dependency in the `skaffold.yaml` using the `requires` expression.
{{% readfile file="samples/builders/artifact-dependencies/skaffold.yaml" %}}

The image alias `BASE` is now available as a build-arg in the Dockerfile for `app`:
 
{{% readfile file="samples/builders/artifact-dependencies/Dockerfile.app" %}}

## Build and Deploy

In the [simple-artifact-dependency](https://github.com/GoogleContainerTools/skaffold/tree/master/examples/simple-artifact-dependency) directory, run:

```text
skaffold dev
```

If this is the first time you're running this, then it should build the artifacts, starting with `base` and later `app`. Skaffold can handle any arbitrary dependency graph between artifacts and schedule builds in the right order. It'll also report an error if it detects dependency cycles or self-loops.

```text
Checking cache...
 - base: Not found. Building
 - app: Not found. Building

Building [base]...
<docker build logs here>

Building [app]...
<docker build logs here>
```
It will then deploy a single container pod, while also monitoring for file changes.

```text
Watching for changes...
[simple-artifact-dependency-app] Hello World
[simple-artifact-dependency-app] Hello World
```

Modify the text in file `base/hello.txt` to something else instead:

```text
Hello World!!!
```

This will trigger a rebuild for the `base` artifact, and since `app` artifact depends on `base` it'll also trigger a rebuild for that. After deployment stabilizes, it should now show the logs reflecting this change:

```text
Watching for changes...
[simple-artifact-dependency-app] Hello World!!!
[simple-artifact-dependency-app] Hello World!!!
```

## Cleanup

Hitting `Ctrl + C` on a running Skaffold process should end it and cleanup its deployments. If there are still persisting objects then you can issue `skaffold delete` command to attempt the cleanup again.