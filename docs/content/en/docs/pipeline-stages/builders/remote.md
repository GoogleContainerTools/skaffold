---
title: "Remote Build"
linkTitle: "Remote"
weight: 20
featureId: build
---

Skaffold supports building remotely with Google Cloud Build.

[Google Cloud Build](https://cloud.google.com/cloud-build/) is a
[Google Cloud Platform](https://cloud.google.com) service that executes
your builds using Google infrastructure. To get started with Google
Build, see [Cloud Build Quickstart](https://cloud.google.com/cloud-build/docs/quickstart-docker).

Skaffold can automatically connect to Cloud Build, and run your builds
with it. After Cloud Build finishes building your artifacts, they will
be saved to the specified remote registry, such as
[Google Container Registry](https://cloud.google.com/container-registry/).

Skaffold Google Cloud Build process differs from the gcloud command
`gcloud builds submit`. Skaffold will create a list of dependent files
and submit a tar file to GCB. It will then generate a single step `cloudbuild.yaml`
and will start the building process. Skaffold does not honor `.gitignore` or `.gcloudignore`
exclusions. If you need to ignore files use `.dockerignore`. Any `cloudbuild.yaml` found will not
be used in the build process.

**Configuration**

To use Cloud Build, add build type `googleCloudBuild` to the `build`
section of `skaffold.yaml`. The following options can optionally be configured:

{{< schema root="GoogleCloudBuild" >}}


Skaffold supports following two Google Cloud Builders.

1. [Docker](#dockerfile-with-docker)
2. [Jib](#jib-maven-and-gradle)

## Dockerfile with Docker

The following `build` section, instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with Google Cloud Build:

{{% readfile file="samples/builders/gcb.yaml" %}}

## Jib Maven and Gradle

The following `build` section, instructs Skaffold to build
 `gcr.io/k8s-skaffold/project1` with Google Cloud Build using Jib builder:

{{% readfile file="samples/builders/gcb-jib.yaml" %}}