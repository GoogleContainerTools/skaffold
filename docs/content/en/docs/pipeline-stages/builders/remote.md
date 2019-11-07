

## Dockerfile with Docker

The following `build` section, instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with Google Cloud Build:

{{% readfile file="samples/builders/gcb.yaml" %}}

## Jib Maven and Gradle

The following `build` section, instructs Skaffold to build
 `gcr.io/k8s-skaffold/project1` with Google Cloud Build using Jib builder:

{{% readfile file="samples/builders/gcb-jib.yaml" %}}