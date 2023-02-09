---
title: "Managing ARM workloads [NEW]"
linkTitle: "Managing ARM workloads [NEW]"
featureId: build.platforms
weight: 50
---

Skaffold has a lot of intelligence built-in to simplify working with ARM workloads. Whether developing on an Apple Silicon Macbook that uses an ARM based chip, or deploying to a GKE Kubernetes cluster having ARM nodes, Skaffold can take away the complexities that arise when the architecture of your development machine and Kubernetes cluster don't match.

## Why is image architecture important?

Container images are built targeting specific [Instruction Set Architectures](https://en.wikipedia.org/wiki/Instruction_set_architecture) like `amd64`, `arm64`, etc. **You must use container images that are compatible with the architecture of the node where you intend to run the workloads.** For example, to deploy to a GKE cluster running ARM nodes, the image needs to be built for `linux/arm64` platform.

All image builders build for different default architecture and not all support cross-architecture builds. For instance [Docker]({{<relref "/docs/pipeline-stages/builders/builder-types/docker">}}) will build the image for the same architecture as the host machine, whereas [Buildpacks]({{<relref "/docs/pipeline-stages/builders/builder-types/buildpacks">}}) will always build it for `amd64`.

Additionally, the following combination of development machine and cluster node architectures can make it difficult to build and deploy images correctly:

* Dev machine architecture is `amd64` while the target cluster runs `arm64` nodes.
* Dev machine architecture is `arm64` (say Apple Silicon Macbooks) while the target cluster runs `amd64` nodes.
* The target cluster runs both `arm64` and `amd64` nodes (mixed node pools).

🎉 *Skaffold provides an opionated way to handle all these cases effectively.* 🎉

## Skaffold can set the image architecture automatically

When running Skaffold in an interactive mode like `skaffold dev`, `skaffold debug` or `skaffold run` where the intention is to build an image from the application code, and immediately deploy it to a Kubernetes cluster, Skaffold will check the active Kubernetes cluster node architecture and provide that as an argument to the respective image builder. If the cluster has multiple architecture nodes, then Skaffold will also create appropriate Kubernetes [`affinity`](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity) rules so that the Kubernetes Pods with these images are assigned to matching architecture nodes.

{{< alert title="Note" >}}
Skaffold will create platform node `affinity` rules only for clusters having multiple architecture nodes. You can also force this using the flag `--enable-platform-node-affinity=true` to always create these affinity rules in the Kubernetes manifests for built images.
{{< /alert >}}

Let's test this in a [sample Golang](https://github.com/GoogleContainerTools/skaffold/tree/main/examples/cross-platform-builds) project:

* The `skaffold.yaml` file defines a single [Docker build artifact](https://github.com/GoogleContainerTools/skaffold/blob/main/examples/cross-platform-builds/Dockerfile) and deploys it in a [`Kubernetes Pod`](https://github.com/GoogleContainerTools/skaffold/blob/main/examples/cross-platform-builds/k8s-pod.yaml).

* First set the active Kubernetes context to a cluster having only `linux/amd64` nodes, and run:

  ```cmd
  skaffold dev --default-repo=your/container/registy
  ```

  Skaffold will detect the cluster node platform `linux/amd64` and build the image for this platform:

  ```cmd
  skaffold dev --default-repo=gcr.io/k8s-skaffold
  Listing files to watch...
  - skaffold-example
  Generating tags...
  - skaffold-example -> gcr.io/k8s-skaffold/skaffold-example:latest
  Starting build...
  Building [skaffold-example]...
  Target platforms: [linux/amd64]
  ...
  Build [skaffold-example] succeeded
  Starting deploy...
  - pod/getting-started created
  Waiting for deployments to stabilize...
  - pods is ready.
  Deployments stabilized in 7.42 seconds
  Press Ctrl+C to exit
  Watching for changes...
  [getting-started] Hello world! Running on linux/amd64
  ```

* Now set the active Kubernetes context to a cluster containing only `linux/arm64` nodes. See [here](https://cloud.google.com/kubernetes-engine/docs/how-to/prepare-arm-workloads-for-deployment) to know how you can create an ARM GKE cluster.

  Re-running the `dev` command will now build a `linux/arm64` image.

  ```cmd
  skaffold dev --default-repo=gcr.io/k8s-skaffold
  ...
  ...
  [getting-started] Hello world! Running on linux/arm64
  ```

* Now set the active Kubernetes context to a cluster containing both `linux/arm64` and `linux/amd64` nodes. You can create a GKE cluster with 2 node pools, one having `linux/amd64` nodes, and the other having `linux/arm64` nodes.

  Re-run the `dev` command but with an explicit platform target this time via the `--platform` flag. If we don't provide the target platform explicitly then Skaffold will choose one between `linux/amd64` and `linux/arm64`, trying to match your local dev machine architecture.

  ```cmd
  skaffold dev --default-repo=your/container/registy --platform=linux/amd64
  ```

  Skaffold will build a `linux/amd64` image and insert a `nodeAffinity` definition to the `Pod` so that it gets scheduled on the matching architecture node.

  ```cmd
  skaffold dev --default-repo=gcr.io/k8s-skaffold --platform=linux/amd64
  ...
  ...
  [getting-started] Hello world! Running on linux/amd64
  ```

* Validate that the `nodeAffinity` was applied by running the command (skip `| jq` if you don't have `jq` installed):

  ```cmd
  kubectl get pod getting-started  -o=jsonpath='{.spec.affinity}' | jq
  {
    "nodeAffinity": {
      "requiredDuringSchedulingIgnoredDuringExecution": {
        "nodeSelectorTerms": [
          {
            "matchExpressions": [
              {
                "key": "kubernetes.io/os",
                "operator": "In",
                "values": [
                  "linux"
                ]
              },
              {
                "key": "kubernetes.io/arch",
                "operator": "In",
                "values": [
                  "amd64"
                ]
              }
            ]
          }
        ]
      }
    }
  }
  ```

This example will run the same whether you're using an `arm64` machine (say an Apple Silicon Macbook) or an `amd64` machine.

Skaffold also supports cross-architecture builds on [Google Cloud Build](https://cloud.google.com/build). You can rerun this example, with the additional flag `--profile cloudbuild` to all the `dev` commands to build on `Google Cloud Build` instead of the local Docker daemon.

## What about multi-arch images?

A [multi-arch image](https://www.docker.com/blog/multi-arch-build-and-images-the-simple-way/) is an image that can support multiple architectures. It looks like a single image with a single tag, but is actually a list of images targeting multiple architectures organized by an [image index](https://github.com/opencontainers/image-spec/blob/main/image-index.md). When you deploy a multi-arch image to a cluster, the container runtime automatically chooses the right image that is compatible with the architecture of the node to which it is being deployed. This simplifies targeting multiple clusters of different architecture nodes, and/or mixed-architecture nodes.

Skaffold supports building multi-platform images natively using the [jib builder]({{<relref "/docs/pipeline-stages/builders/builder-types/jib" >}}), the [ko builder]({{<relref "/docs/pipeline-stages/builders/builder-types/ko">}}) and the [custom builder]({{<relref "/docs/pipeline-stages/builders/builder-types/custom" >}}). For other builders that support building cross-architecture images, Skaffold will iteratively build a single platform image for each target architecture and stitch them together into a multi-platform image, and push it to the registry.

![multi-arch-flow](/images/multi-arch-flow.png)

Let's test this in the same [sample Golang](https://github.com/GoogleContainerTools/skaffold/tree/main/examples/cross-platform-builds) project as before:

* Run this command to build for the target architectures `linux/amd64` and `linux/arm64`:

  ```cmd
  skaffold build -t latest --default-repo=your/container/registy --platform=linux/amd64,linux/arm64
  ...
  Building [skaffold-example]...
  Target platforms: [linux/amd64,linux/arm64]
  ...
  [+] Building 0.3s (13/13) FINISHED
  ...
  => => writing image sha256:10af3142e460566f5791c48758f0040cef6932cbcb0766082dcbb0d8db7653e7
  => => naming to gcr.io/k8s-skaffold/skaffold-example:latest_linux_amd64
  ...
  latest_linux_amd64: digest: sha256:15bd4f2380e99b3563f8add1aba9691e414d4cc5701363d9c74960a20fb276c4 size: 739
  ...
  [+] Building 52.8s (13/13) FINISHED
  ...
  => => writing image sha256:68866691e2d6f079b116e097ae4e67a53eaf89e825b52d6f31f2e9cc566974de
  => => naming to gcr.io/k8s-skaffold/skaffold-example:latest_linux_arm64
  ...
  4e0c2525c370: Pushed
  latest_linux_arm64: digest: sha256:868d0aec1cc7d2ed1fa1e840f38ff1aa50c3cc3d3232ea17a065618eaec4e82b size: 739
  Build [skaffold-example] succeeded
  ```

* Validate that the image just built was multi-arch, by running the following `docker` command:

  ```cmd
  docker manifest inspect your/container/registry/skaffold-example:latest | grep -A 3 "platform"
  ```

  Outputs:

  ```cmd
    "platform": {
        "architecture": "amd64",
        "os": "linux"
    }
  --
    "platform": {
        "architecture": "arm64",
        "os": "linux"
    }
  ```

* Now if we render the Kubernetes Pod manifest for this multi-arch image, then it'll have platform affinity definition targeting both `linux/amd64` and `linux/arm64` architectures.

  ```cmd
  skaffold render --default-repo=your/container/registry --enable-platform-node-affinity
  ```
  
  Outputs:

  ```cmd
  apiVersion: v1
  kind: Pod
  metadata:
    name: getting-started
    namespace: default
  spec:
    affinity:
      nodeAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
          nodeSelectorTerms:
          - matchExpressions:
            - key: kubernetes.io/os
              operator: In
              values:
              - linux
            - key: kubernetes.io/arch
              operator: In
              values:
              - amd64
          - matchExpressions:
            - key: kubernetes.io/os
              operator: In
              values:
              - linux
            - key: kubernetes.io/arch
              operator: In
              values:
              - arm64
    containers:
    - image: gcr.io/k8s-skaffold/skaffold-example:latest@sha256:9ecf4e52f7ff64b35deacf9d6eedc03f35d69e0b4bf3679b97ba492f4389f784
      name: getting-started
  ```

{{< alert title="Note" >}}

* Multi-arch images need to be pushed to a container registry, as the local Docker deamon doesn't yet support storing multi-arch images.

* For interactive modes like `skaffold dev` and `skaffold debug` requiring fast and repeated `build-render-deploy` iterations, Skaffold will choose only one build architecture and build a single-platform image, even if you specify multiple target platforms.

* If you need to build a multi-arch image with an interactive mode then use `skaffold run`. This will build the multi-arch image and deploy it to the active Kubernetes cluster.
{{< /alert >}}
