### Example: use the custom builder with `docker buildx`

[Docker Buildx](https://github.com/docker/buildx#buildx) is an
experimental feature for building container images for multiple
platforms.

This example shows how `docker buildx` can be used as a
Skaffold _custom builder_ to create container images for
for two different platforms: `linux/arm64` and `linux/amd64`.

#### Before you begin

For this tutorial to work you need to ensure Skaffold and a Kubernetes
cluster are set up.  To learn more about how to set up Skaffold and
a Kubernetes cluster, see the [getting started docs](https://skaffold.dev/docs/getting-started).

Note that this example builds for two different platforms and
requires pushing images to a container registry such as
[Google Artifact Registry](https://cloud.google.com/artifact-registry).

#### Tutorial

This tutorial will demonstrate how Skaffold can build a simple
_Hello World_ Go application for linux/amd64 and linux/arm64 using
`docker buildx` and deploy it to a Kubernetes cluster.

##### Step 1: Configure _Docker Buildx_

To use `docker buildx` you must first create a named _builder_ with
the set of platforms to be built.  Run the following to create a
builder named `skaffold-builder` to build for `linux/arm64` and
`linux/amd64`:

```
docker buildx create --name skaffold-builder --platform linux/arm64,linux/amd64
```

##### Step 2: Obtain the example

First, clone the Skaffold [repo](https://github.com/GoogleContainerTools/skaffold)
and navigate to the [`custom-buildx` example](https://github.com/GoogleContainerTools/skaffold/tree/master/examples/custom) for sample code:

```shell
$ git clone https://github.com/GoogleContainerTools/skaffold
$ cd skaffold/examples/custom-buildx
```

Take a look at the [skaffold.yaml](skaffold.yaml), which uses a
_custom builder_ to invoke `docker buildx` to containerize source
code for two platforms.
For more information about custom builders, see the Skaffold custom
builder [documentation](https://skaffold.dev/docs/how-tos/builders/#custom-build-script-run-locally).
Note that Skaffold builders are different from `docker buildx` builders.

Next look at the Kubernetes [pod descriptor](k8s/pod.yaml) and notice
the use of _node affinities_ to instruct Kubernetes to scheduler the workload
on nodes running either `linux/arm64` or `linux/amd64`.  It is important
to realize that Kubernetes does not examine the referenced container images
to determine the possible platforms.


##### Step 3: Build and deploy the example

Now, use Skaffold to deploy this application to your Kubernetes cluster:

```shell
$ skaffold run --tail --default-repo <your repo>
```

With this command, Skaffold will build the artifact with `docker buildx`
and deploy the application to Kubernetes.  You should be able to
see *Hello, World!* printed every second in the Skaffold logs.

We need to use `--default-repo` to push to a repository as Docker does
not support loading multi-platform images with the same name.

#### Cleanup

To clean up your Kubernetes cluster, run:

```shell
$ skaffold delete
```
