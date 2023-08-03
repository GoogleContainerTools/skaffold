---
title: "Google Cloud Run [NEW]"
linkTitle: "Google Cloud Run [NEW]"
weight: 60
featureId: deploy.cloudrun
aliases: [/docs/pipeline-stages/deployers/cloudrun]
---

{{< alert title="Note" >}}
This feature is currently experimental and subject to change. Not all Skaffold features are supported, for example `debug` is currently not supported in Cloud Run (but is on our roadmap).
{{< /alert >}}

[Cloud Run](https://cloud.google.com/run) is a managed compute platform on Google Cloud that allows you to run containers on Google's infrastructure. With Skaffold, now you are able to configure your dev loop to build, test, sync and use Cloud Run as the deployer for your images.


## Deploying applications to Cloud Run
Skaffold can deploy [Services](https://cloud.google.com/run/docs/reference/rest/v1/namespaces.services#resource:-service) and [Jobs](https://cloud.google.com/run/docs/reference/rest/v1/namespaces.jobs#resource:-job) to Cloud Run. If this deployer is used, all provided manifests must be valid Cloud Run services, using the `serving.knative.dev/v1` schema, or valid Cloud Run jobs.
See the [Cloud Run YAML reference](https://cloud.google.com/run/docs/reference/yaml/v1) for supported fields.

### Environment setup
In order to use this deployer you'll need to configure some tools first.

The deployer uses the `gcloud` CLI to perform its tasks, so be sure it is installed in your environment. It will use the [application default credentials](https://cloud.google.com/docs/authentication/production#automatically) to deploy.  You can configure this to use your user credentials by running:
```bash
gcloud auth application-default login
```

To enable [Log streaming]({{< relref "#log-streaming" >}}) and [Port forwarding]({{< relref "#port-forwarding" >}}) some extra components are needed from `gcloud`. To install them run the following comand in your terminal:
```bash
gcloud components install --quiet \
    alpha \
    beta \
    log-streaming \
    cloud-run-proxy
```

From the previous command, `alpha` and `log-streaming` components are needed for [Log streaming]({{< relref "#log-streaming" >}}), `beta` and `cloud-run-proxy` components are needed for [Port forwarding]({{< relref "#port-forwarding" >}}).

## Features

### Cloud Run Services and Jobs deployment
With Skaffold you can deploy Cloud Run [Services](https://cloud.google.com/run/docs/overview/what-is-cloud-run#services) and [Jobs](https://cloud.google.com/run/docs/overview/what-is-cloud-run#jobs) just referencing them from the `skaffold.yaml` file. The following example ilustrates a project using the Cloud Run deployer:

With the following project folder structure:
```yaml
resources/
  cloud-run-service.yaml
  cloud-run-job.yaml
skaffold.yaml
```

`cloud-run-service.yaml` content:
```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: cloud-run-service-name # this service will be created in Cloud Run via Skaffold
spec:
  template:
    spec:
      containers:
      - image: gcr.io/cloudrun/hello
```

`cloud-run-job.yaml` content:
```yaml
apiVersion: run.googleapis.com/v1
kind: Job
metadata:
  name: cloud-run-job-name # this job will be created in Cloud Run via Skaffold
  annotations:
    run.googleapis.com/launch-stage: BETA
spec:
  template:
    spec:
      template:
        spec:
          containers:
          - image: us-docker.pkg.dev/cloudrun/container/job
```

`skaffold.yaml` content:
{{% readfile file="samples/deployers/cloud-run/simple-service-job-deployment.yaml" %}}

Running `skaffold run` will deploy one Cloud Run service, and one Cloud Run job in the `YOUR-GCP-PROJECT` project, inside the given `GCP-REGION`.

{{< alert title="Note" >}}
The previous example will deploy a Cloud Run job, however, it will not trigger an execution for that job. To read more about jobs execution you can check the [Cloud Run docs](https://cloud.google.com/run/docs/execute/jobs).
{{< /alert >}}

### Port forwarding {#port-forwarding}

Skaffold will manage automatically the necessary configuration to open the deployed Cloud Run services URLs locally, even if they are private services, using the [Cloud Run proxy](https://cloud.google.com/sdk/gcloud/reference/beta/run/services/proxy) and Skaffold's Port Forwarding. To enable this, you will have to either add the `--port-forward` flag running Skaffold, or add a `portForward` stanza in your `skaffold.yaml` file. From the previous example, running `skaffold dev --port-forward` will result in the following output:

```
...
Deploying Cloud Run service:
         cloud-run-job-name
Deploying Cloud Run service:
         cloud-run-service-name
Cloud Run Job cloud-run-job-name finished: Job started. 1/2 deployment(s) still pending
cloud-run-service-name: Service starting: Deploying Revision. Waiting on revision cloud-run-service-name-2246v.
cloud-run-service-name: Service starting: Deploying Revision. Waiting on revision cloud-run-service-name-2246v.
Cloud Run Service cloud-run-service-name finished: Service started. 0/2 deployment(s) still pending
Forwarding service projects/<YOUR-GCP-PROJECT>/locations/<GCP-REGION>/services/cloud-run-service-name to local port 8080
...
```

Here you'll see the port to use to access the deployed Cloud Run service, in this case you can access it through `localhost:8080`. If you need to change the local port used, you'll need to add a `portForward` stanza:

Using the previous example, changing `skaffold.yaml` to:
{{% readfile file="samples/deployers/cloud-run/service-port-forward.yaml" %}}

Running `skaffold dev --port-forward`, will result in:

```
...
Forwarding service projects/<YOUR-GCP-PROJECT>/locations/<GCP-REGION>/services/cloud-run-service-name to local port 9001
...
```

Now you will be able to access the deployed service through `localhost:9001`.


### Log streaming {#log-streaming}

When doing [local development]({{< relref "docs/workflows/dev">}}), Skaffold will log stream to your console the output from the Cloud Run services deployed. From the previous example, running `skaffold dev --port-forward` or `skaffold run --tail --port-forward` in your terminal, you will see the following output:

```
...
Cloud Run Service cloud-run-service-name finished: Service started. 0/2 deployment(s) still pending
Forwarding service projects/<YOUR-GCP-PROJECT>/locations/<GCP-REGION>/services/cloud-run-service-name to local port 9001
No artifacts found to watch
Press Ctrl+C to exit
Watching for changes...
[cloud-run-service-name] streaming logs from <YOUR-GCP-PROJECT>
...
```

Now Skaffold is log streaming the output from the service. If you access it through `localhost:9001`, you'll see the logs:

```
...
[cloud-run-service-name] streaming logs from renzo-friction-log-cloud-run
[cloud-run-service-name] 2023-01-27 00:52:22 2023/01/27 00:52:22 Hello from Cloud Run! The container started successfully and is listening for HTTP requests on $PORT
[cloud-run-service-name] 2023-01-27 00:52:22 GET 200 https://cloud-run-service-name-6u2evvstna-uc.a.run.app/
```

## Configuring Cloud Run

To deploy to Cloud Run, use the `cloudrun` type in the `deploy` section, together with `manifests.rawYaml` stanza of `skaffold.yaml`.

The `cloudrun` type offers the following options:

{{< schema root="CloudRunDeploy" >}}


### Example

The following `deploy` section instructs Skaffold to deploy the artifacts under `manifests.rawYaml` to Cloud Run:

{{% readfile file="samples/deployers/cloud-run/cloud-run.yaml" %}}

## Cloud Run deployer + Skaffold Local build

With Skaffold you can configure your project to [locally build]({{< relref "docs/builders/build-environments/local" >}}) your images and deploy them to Cloud Run. The following example demonstrates how to set up Skaffold for this:

With the following project folder structure:
```yaml
resources/
  service.yaml
skaffold.yaml
Dockerfile
```

`skaffold.yaml` file content:
{{% readfile file="samples/deployers/cloud-run/local-build-image.yaml" %}}


`resources/service.yaml` file content:
```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: cloud-run-service
spec:
  template:
    spec:
      containers:
      - image: my-img # <- Same image name from skaffold.yaml file
```

A simple `Dockerfile` file:
```docker
FROM gcr.io/cloudrun/hello
```

Running a Skaffold command like `skaffold run --default-repo=gcr.io/your-registry` will build your local images, push them to the specified registry, and deploy them to Cloud Run. Please notice the following from the previous example:

### Build local push option
When you use [Skaffold Local build]({{< relref "docs/builders/build-environments/local#avoiding-pushes" >}}), the `push` option is set to `false` by default. However, Cloud Run will need your images published in a registry that it has access to. Therefore, we need to set this to `true`.

### Platform
According to the [Cloud Run runtime contract](https://cloud.google.com/run/docs/container-contract#languages), your images must be compiled for a specific architecture. Skaffold can help us with this by using its [Cross/multi-platform build support]({{< relref "docs/builders/cross-platform" >}}).

### Registry
You'll need to specify a registry so Skaffold can push your images. For this, we can use the `--default-repo` flag when running a command to include it in all your local images.
