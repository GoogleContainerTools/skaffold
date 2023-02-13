---
title: "Google Cloud Build"
linkTitle: "Google Cloud Build"
weight: 30
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

Skaffold can build multiple artifacts in parallel, by settings a value higher than `1` to `concurrency`.
For Google Cloud Build, the default is to build all the artifacts in parallel. If you hit a quota restriction,
you might want to reduce  the `concurrency`.

{{<alert title="Note">}}
When artifacts are built in parallel, the build logs are still printed in sequence to make them easier to read.
{{</alert>}}

## Restrictions

Skaffold currently supports [Docker]({{<relref "/docs/pipeline-stages/builders/builder-types/docker#dockerfile-remotely-with-google-cloud-build">}}),
[Jib]({{<relref "/docs/pipeline-stages/builders/builder-types/jib#remotely-with-google-cloud-build">}})
on Google Cloud Build.
