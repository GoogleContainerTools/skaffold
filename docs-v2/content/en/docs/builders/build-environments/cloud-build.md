---
title: "Google Cloud Build"
linkTitle: "Google Cloud Build"
weight: 30
---

Skaffold supports building remotely with Google Cloud Build.

[Cloud Build](https://cloud.google.com/cloud-build/) is a
[Google Cloud Platform](https://cloud.google.com) service that executes
your builds using Google infrastructure. To get started with Cloud
Build, see [Cloud Build Quickstart](https://cloud.google.com/cloud-build/docs/quickstart-docker).

Skaffold automatically connects to Cloud Build and runs your builds
with it. After Cloud Build finishes building your artifacts, they are
saved to the specified remote registry, such as
[Google Container Registry](https://cloud.google.com/container-registry/).

Skaffold's Cloud Build process differs from the gcloud command
[`gcloud builds submit`](https://cloud.google.com/sdk/gcloud/reference/builds/submit).
Skaffold does the following:
* Creates a list of dependent files
* Uploads a tar file of the dependent files to Google Cloud Storage
* Submits the tar file to Cloud Build
* Generates a single-step `cloudbuild.yaml`
* Starts the build

Skaffold does not honor `.gitignore` or `.gcloudignore` exclusions. If you need to ignore files, use `.dockerignore`.
Any `cloudbuild.yaml` found will not be used in the build process.

## Configuration

To use Cloud Build, add build type `googleCloudBuild` to the `build`
section of `skaffold.yaml`. 

```yaml
build:
  googleCloudBuild: {}
```

The following options can optionally be configured:

{{< schema root="GoogleCloudBuild" >}}

## Faster builds

By default, Cloud Build (invoked by Skaffold) builds all artifacts in parallel. Set `concurrency` to a non-zero
value to specify the maximum number of artifacts to build concurrently. Consider reducing `concurrency` if you
hit a quota restriction.

{{<alert title="Note">}}
When Skaffold builds artifacts in parallel, it still prints the build logs in sequence to make them easier to read.
{{</alert>}}

## Restrictions

Skaffold currently supports the following [builder types]({{<relref "/docs/builders/builder-types">}})
when building remotely with Cloud Build:
* [Docker]({{<relref "/docs/builders/builder-types/docker#dockerfile-remotely-with-google-cloud-build">}})
* [Jib]({{<relref "/docs/builders/builder-types/jib#remotely-with-google-cloud-build">}})
