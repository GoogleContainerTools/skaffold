---
title: "Docker Build"
linkTitle: "Docker"
weight: 10
featureId: build
---

Skaffold supports building with Dockerfile

1. [locally]({{< relref "/docs/pipeline-stages/builders/docker#dockerfile-with-docker-locally">}})
2. [in cluster]({{< relref "/docs/pipeline-stages/builders/docker#dockerfile-in-cluster-with-kaniko">}})
3. [on Google CloudBuild ]({{< relref "/docs/pipeline-stages/builders/docker#dockerfile-remotely-with-google-cloud-build">}})

## Dockerfile with Docker locally

If you have [Docker](https://www.docker.com/products/docker-desktop)
installed, Skaffold can be configured to build artifacts with the local
Docker daemon.

By default, Skaffold connects to the local Docker daemon using
[Docker Engine APIs](https://docs.docker.com/develop/sdk/), though
it can also use the Docker
[command-line interface](https://docs.docker.com/engine/reference/commandline/cli/)
instead, which enables artifacts with [BuildKit](https://github.com/moby/buildkit).

After the artifacts are successfully built, Docker images will be pushed
to the remote registry. You can choose to skip this step.

**Configuration**

To use the local Docker daemon, add build type `local` to the `build` section
of `skaffold.yaml`. The following options can optionally be configured:

{{< schema root="LocalBuild" >}}

**Example**

The following `build` section instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with the local Docker daemon:

{{% readfile file="samples/builders/local.yaml" %}}

Which is equivalent to:

{{% readfile file="samples/builders/local-full.yaml" %}}

## Dockerfile in-cluster with Kaniko

[Kaniko](https://github.com/GoogleContainerTools/kaniko) is a Google-developed
open source tool for building images from a Dockerfile inside a container or
Kubernetes cluster. Kaniko enables building container images in environments
that cannot easily or securely run a Docker daemon.

Skaffold can help build artifacts in a Kubernetes cluster using the Kaniko
image; after the artifacts are built, kaniko must push them to a registry.


**Configuration**

To use Kaniko, add build type `kaniko` to the `build` section of
`skaffold.yaml`. The following options can optionally be configured:

{{< schema root="KanikoArtifact" >}}

Since Kaniko builds images directly to a registry, it requires active cluster credentials.
These credentials are configured in the `cluster` section with the following options:

{{< schema root="ClusterDetails" >}}

To set up the credentials for Kaniko refer to the [kaniko docs](https://github.com/GoogleContainerTools/kaniko#kubernetes-secret) (**Note**: Rename the downloaded JSON key to *kaniko-secret* without appending *.json*).
Alternatively, the path to a credentials file can be set with the `pullSecretPath` option:
```yaml
build:
  cluster:
    pullSecretName: pull-secret-in-kubernetes
    # OR
    pullSecretPath: path-to-service-account-key-file
```
Similarly, when pushing to a docker registry:
```yaml
build:
  cluster:
    dockerConfig:
      path: ~/.docker/config.json
      # OR
      secretName: docker-config-secret-in-kubernetes
```
Note that the Kubernetes secret must not be of type `kubernetes.io/dockerconfigjson` which stores the config json under the key `".dockerconfigjson"`, but an opaque secret with the key `"config.json"`.

**Example**

The following `build` section, instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with Kaniko:

{{% readfile file="samples/builders/kaniko.yaml" %}}

## Dockerfile remotely with Google Cloud Build

Skaffold can build the Dockerfile image remotely with [Google Cloud Build]({{<relref "/docs/pipeline-stages/builders#remotely-on-google-cloud-build">}}).

**Configuration**

To configure, add `googleCloudBuild` to `build` section to `skaffold.yaml`.
The following options can optionally be configured:

{{< schema root="GoogleCloudBuild" >}}

**Example**

The following `build` section, instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with Google Cloud Build:

{{% readfile file="samples/builders/gcb.yaml" %}}
