### Example: use the custom builder with Cloud Native Buildpacks

This example shows how the custom builder can be used to
build artifacts with Cloud Native Buildpacks.

* **building** a single Go file app with buildpacks
* **tagging** using the default tagPolicy (`gitCommit`)
* **deploying** a single container pod using `kubectl`

#### Before you begin

For this tutorial to work, you will need to have Skaffold and a Kubernetes cluster set up.
To learn more about how to set up Skaffold and a Kubernetes cluster, see the [getting started docs](https://skaffold.dev/docs/getting-started).

To use buildpacks with Skaffold, please install the following additional tools:

* [pack](https://buildpacks.io/docs/install-pack/)
* [docker](https://docs.docker.com/install/)

#### Tutorial

This tutorial will demonstrate how Skaffold can build a simple Hello World Go application with buildpacks and deploy it to a Kubernetes cluster.

First, clone the Skaffold [repo](https://github.com/GoogleContainerTools/skaffold) and navigate to the [buildpacks example](https://github.com/GoogleContainerTools/skaffold/tree/master/examples/buildpacks) for sample code:

```shell
$ git clone https://github.com/GoogleContainerTools/skaffold
$ cd skaffold/examples/buildpacks
```

Take a look at the `build.sh` file, which uses `pack` to containerize source code with buildpacks:

```shell
$ cat build.sh
#!/usr/bin/env bash
set -e

pack build --builder=heroku/buildpacks $IMAGE

if $PUSH_IMAGE; then
    docker push $IMAGE
fi
```

and the skaffold config, which configures artifact `gcr.io/k8s-skaffold/skaffold-example` to build with `build.sh`:

```yaml
$ cat skaffold.yaml
apiVersion: skaffold/v2alpha1
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/skaffold-example
    custom:
      buildCommand: ./build.sh
```

For more information about how this works, see the Skaffold custom builder [documentation](https://skaffold.dev/docs/how-tos/builders/#custom-build-script-run-locally).

Now, use Skaffold to deploy this application to your Kubernetes cluster:

```shell
$ skaffold run --tail --default-repo <your repo>
```

With this command, Skaffold will build the `skaffold-example` artifact with buildpacks and deploy the application to Kubernetes.
You should be able to see *Hello, World!* printed every second in the Skaffold logs.

#### Cleanup

To clean up your Kubernetes cluster, run:

```shell
$ skaffold delete
```
