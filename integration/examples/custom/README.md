### Example: use the custom builder with ko

This example shows how the custom builder can be used to
build artifacts with [ko](https://github.com/google/ko).

* **building** a single Go file app with ko
* **tagging** using the tagPolicy (`sha256`), to mimic the behavior of ko
* **deploying** a single container pod using `kubectl`

#### Before you begin

For this tutorial to work, you will need to have Skaffold and a Kubernetes cluster set up.
To learn more about how to set up Skaffold and a Kubernetes cluster, see the [getting started docs](https://skaffold.dev/docs/getting-started).

#### Tutorial

This tutorial will demonstrate how Skaffold can build a simple Hello World Go application with ko and deploy it to a Kubernetes cluster.

First, clone the Skaffold [repo](https://github.com/GoogleContainerTools/skaffold) and navigate to the [custom example](https://github.com/GoogleContainerTools/skaffold/tree/master/examples/custom) for sample code:

```shell
$ git clone https://github.com/GoogleContainerTools/skaffold
$ cd skaffold/examples/custom
```

Take a look at the `build.sh` file, which uses `ko` to containerize source code:

```shell
$ cat build.sh
#!/usr/bin/env bash
set -e

if ! [ -x "$(command -v ko)" ]; then
    pushd $(mktemp -d)
    go mod init tmp; GOFLAGS= go get github.com/google/ko/cmd/ko@v0.6.0
    popd
fi

output=$(ko publish --local --preserve-import-paths --tags= . | tee)
ref=$(echo $output | tail -n1)

docker tag $ref $IMAGE
if $PUSH_IMAGE; then
    docker push $IMAGE
fi
```

and the skaffold config, which configures image `ko://github.com/GoogleContainerTools/skaffold/examples/custom` to build with `build.sh`:

```yaml
$ cat skaffold.yaml
apiVersion: skaffold/v2beta9
kind: Config
build:
  artifacts:
  - image: ko://github.com/GoogleContainerTools/skaffold/examples/custom
    custom:
      buildCommand: ./build.sh
      dependencies:
        paths:
        - "go.mod"
        - "**.go"
  tagPolicy:
    sha256: {}
```

The `k8s/pod.yaml` manifest file uses the same image reference:

```yaml
$ cat k8s/pod.yaml
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - name: getting-started
    image: ko://github.com/GoogleContainerTools/skaffold/examples/custom
```

For more information about how this works, see the Skaffold custom builder [documentation](https://skaffold.dev/docs/how-tos/builders/#custom-build-script-run-locally).

Now, use Skaffold to deploy this application to your Kubernetes cluster:

```shell
$ skaffold run --tail --default-repo <your repo>
```

With this command, Skaffold will build the `github.com/googlecontainertools/skaffold/examples/custom` artifact with ko and deploy the application to Kubernetes.
You should be able to see *Hello, World!* printed every second in the Skaffold logs.

#### Cleanup

To clean up your Kubernetes cluster, run:

```shell
$ skaffold delete
```
