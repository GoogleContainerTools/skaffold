---
title: "Google Cloud Run [NEW]"
linkTitle: "Google Cloud Run [NEW]"
weight: 60
featureId: deploy.cloudrun
---

{{< alert title="Note" >}}
This feature is currently experimental and subject to change. Not all Skaffold features e.g. log tailing, debugging are supported.
{{< /alert >}}

Cloud Run is a managed compute platform on Google Cloud that allows you to run containers on Google's infrastructure.


## Deploying applications to Cloud Run

Skaffold can deploy services to Cloud Run. If this deployer is used, all provided manifests must be valid Cloud Run services, using the serving.knative.dev/v1 schema.
See the [Cloud Run YAML reference](https://cloud.google.com/run/docs/reference/yaml/v1) for supported fields.

This deployer will use the [application default credentials](https://cloud.google.com/docs/authentication/production#automatically) to deploy.  You can configure this to use your user credentials by running `gcloud auth application-default login`.

## Configuring Cloud Run

To deploy to Cloud Run, use the `cloudrun` type in the `deploy` section of `skaffold.yaml`.

The `cloudrun` type offers the following options:

{{< schema root="CloudRunDeploy" >}}

### Example

The following `deploy` section instructs Skaffold to deploy
artifacts to Cloud Run:

{{% readfile file="samples/deployers/cloudrun.yaml" %}}

{{< alert title="Note" >}}
Images listed to be deployed with the Cloud Run deployer must be present in Google Artifact
Registry or Google Container Registry. If you are using Skaffold to build the images, ensure `push` is 
set to true.
{{< /alert >}}
