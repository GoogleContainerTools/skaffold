---
title: "Docker"
linkTitle: "Docker"
weight: 20
featureId: deploy.docker
---

{{< alert title="Note" >}}
This feature is currently experimental and subject to change.
{{< /alert >}}

## Deploying applications to a local Docker daemon

For simple container-based applications that don't rely on
Kubernetes resource types, Skaffold can "deploy" these applications
by running application containers directly in your local Docker daemon.
This enables application developers who are not yet ready to make the jump
to Kubernetes to take advantage of the streamlined development experience
Skaffold provides.

Additionally, deploying to Docker bypasses the overhead of pushing
images to a remote registry, and provides a faster time to running
application than traditional Kubernetes deployments.

### Configuration

To deploy to your local Docker daemon, specify the `docker` deploy type
in the `deploy` section of your `skaffold.yaml`.

The `docker` deploy type offers the following options:

{{< schema root="DockerDeploy" >}}

### Example

The following `deploy` section instructs Skaffold to deploy
the application image `my-image` to the local Docker daemon:

{{% readfile file="samples/deployers/docker.yaml" %}}

{{< alert title="Note" >}}
Images listed to be deployed with the `docker` deployer **must also have a corresponding build artifact built by Skaffold.**
{{< /alert >}}
