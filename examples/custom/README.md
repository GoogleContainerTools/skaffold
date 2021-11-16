### Example: use the custom builder with ko

**Note:** Skaffold now includes a
[`ko` builder](https://skaffold.dev/docs/pipeline-stages/builders/ko/).
When you use the `ko` builder, you do not need to provide a custom build shell
script or install the `ko` binary.

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/GoogleContainerTools/skaffold&cloudshell_open_in_editor=README.md&cloudshell_workspace=examples/custom)

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

First, clone the Skaffold [repo](https://github.com/GoogleContainerTools/skaffold) and navigate to the [custom example](https://github.com/GoogleContainerTools/skaffold/tree/main/examples/custom) for sample code:

```sh
git clone https://github.com/GoogleContainerTools/skaffold.git
```
```sh
cd skaffold/examples/custom
```

Take a look at the `build.sh` file, which uses `ko` to containerize source code:

[embedmd]:# (build.sh bash)
```bash
#!/usr/bin/env bash
set -e
set -x

if ! [ -x "$(go env GOPATH)/bin/ko" ]; then
    pushd $(mktemp -d)
    curl -L https://github.com/google/ko/archive/v0.9.3.tar.gz | tar --strip-components 1 -zx
    go build -o $(go env GOPATH)/bin/ko .
    popd
fi

output=$($(go env GOPATH)/bin/ko publish --local --preserve-import-paths --tags= . | tee)
ref=$(echo "$output" | tail -n1)

docker tag "$ref" "$IMAGE"
if [[ "${PUSH_IMAGE}" == "true" ]]; then
    echo "Pushing $IMAGE"
    docker push "$IMAGE"
else
    echo "Not pushing $IMAGE"
fi
```

and the skaffold config, which configures image `ko://github.com/GoogleContainerTools/skaffold/examples/custom` to build with `build.sh`:

[embedmd]:# (skaffold.yaml yaml)
```yaml
apiVersion: skaffold/v2beta24
kind: Config
build:
  artifacts:
  - image: ko://github.com/GoogleContainerTools/skaffold/examples/custom
    custom:
      buildCommand: ./build.sh
      dependencies:
        paths:
        - "**/*.go"
        - go.mod
        - .ko.yaml
  tagPolicy:
    sha256: {}
```

The `k8s/pod.yaml` manifest file uses the same image reference:

[embedmd]:# (k8s/pod.yaml yaml)
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: getting-started-custom
spec:
  containers:
  - name: getting-started-custom
    image: ko://github.com/GoogleContainerTools/skaffold/examples/custom
```

For more information about how this works, see the Skaffold custom builder [documentation](https://skaffold.dev/docs/how-tos/builders/#custom-build-script-run-locally).

Now, use Skaffold to deploy this application to your Kubernetes cluster:

```sh
skaffold run --tail --default-repo <your repo>
```

With this command, Skaffold will build the `github.com/googlecontainertools/skaffold/examples/custom` artifact with ko and deploy the application to Kubernetes.
You should be able to see *Hello, World!* printed every second in the Skaffold logs.

#### Cleanup

To clean up your Kubernetes cluster, run:

```sh
skaffold delete
```
