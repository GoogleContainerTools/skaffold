---
title: "Continuous Delivery"
linkTitle: "Continuous Delivery"
weight: 40
---

Skaffold provides several features and sub-command "building blocks" that make it very useful for integrating with (or creating entirely new) CI/CD pipelines.
The ability to use the same `skaffold.yaml` for iterative development and continuous delivery eases handing off an application from a development team to an ops team.

Let's start with the simplest use case: a single, full deployment of your application.

## Run entire pipeline end-to-end
{{< maturity "run" >}}

`skaffold run` is a single command for a one-off deployment. It runs through every major phase of the Skaffold lifecycle: building your application images, tagging these images (and optionally pushing them to a remote registry), deploying your application to the target cluster, and monitoring the created resources for readiness.

We recommend `skaffold run` for the simplest Continuous Delivery setup, where it is sufficient to have a single step that deploys from version control to a cluster.

For more sophisticated Continuous Delivery pipelines, Skaffold offers building blocks:

- [status-check]({{<relref "/docs/pipeline-stages/status-check">}}) - 
wait for `deployments` to stabilize and succeed only if all deployments are successful
- [`skaffold build`]({{<relref "/docs/workflows/ci-cd#skaffold-build-skaffold-deploy">}}) - build, tag and push artifacts to a registry
- [`skaffold deploy`]({{<relref "/docs/workflows/ci-cd#skaffold-build-skaffold-deploy">}})  - deploy built artifacts to a cluster
- [`skaffold render`]({{<relref "/docs/workflows/ci-cd#skaffold-render-skaffold-apply">}})  - export the transformed Kubernetes manifests
- [`skaffold apply`]({{<relref "/docs/workflows/ci-cd#skaffold-render-skaffold-apply">}}) - send hydrated Kubernetes manifests to the API server to create resources on the target cluster

## Traditional continuous delivery

`skaffold build` will build your project's artifacts, and push the build images to the specified registry. If your project is already configured to run with Skaffold, `skaffold build` can be a very lightweight way of setting up builds for your CI pipeline. Passing the `--file-output` flag to Skaffold build will also write out your built artifacts in JSON format to a file on disk, which can then by passed to `skaffold deploy` later on. This is a great way of "committing" your artifacts when they have reached a state that you're comfortable with, especially for projects with multiple artifacts for multiple services.

Example using the current git state as a unique file ID to "commit" build state:

Storing the build result in a commit specific JSON file:
```bash
export STATE=$(git rev-list -1 HEAD --abbrev-commit)
skaffold build --file-output build-$STATE.json
```
outputs the tag generation and cache output from Skaffold:
```bash 
Generating tags...
 - gcr.io/k8s-skaffold/skaffold-example:v0.41.0-17-g3ad238db
Checking cache...
 - gcr.io/k8s-skaffold/skaffold-example: Found. Tagging
```

The content of the JSON file
```bash 
cat build-$STATE.json
```
looks like: 
```json
{"builds":[{"imageName":"gcr.io/k8s-skaffold/skaffold-example","tag":"gcr.io/k8s-skaffold/skaffold-example:v0.41.0-17-g3ad238db@sha256:eeffb639f53368c4039b02a4d337bde44e3acc728b309a84353d4857ee95c369"}]}
```

We can then use this build result file to deploy with Skaffold:
```bash
skaffold deploy -a build-$STATE.json
```
and as we'd expect, we see a bit of deploy-related output from Skaffold:
```bash
Tags used in deployment:
 - gcr.io/k8s-skaffold/skaffold-example -> gcr.io/k8s-skaffold/skaffold-example:v0.41.0-17-g3ad238db@sha256:eeffb639f53368c4039b02a4d337bde44e3acc728b309a84353d4857ee95c369
Starting deploy...
 - pod/getting-started configured
```


## Separation of rendering and deployment
{{< maturity "apply" >}}

Skaffold allows separating the generation of fully-hydrated Kubernetes manifests from the actual deployment of those manifests, using the `skaffold render` and
`skaffold apply` commands. 

`skaffold render` builds all application images from your artifacts, templates the newly-generated image tags into your Kubernetes manifests (based on your project's deployment configuration), and then prints out the final hydrated manifests to a file or your terminal.
This allows you to capture the full, declarative state of your application in configuration, such that _applying_ the changes to your cluster can be done as a separate step.

`skaffold apply` consumes one or more fully-hydrated Kubernetes manifests, and then sends the results directly to the Kubernetes control plane via `kubectl` to create resources on the target cluster. After creating the resources on your cluster, `skaffold apply` uses Skaffold's built-in health checking to monitor the created resources for readiness. See [resource health checks]({{<relref "/docs/pipeline-stages/status-check">}}) for more information on how Skaffold's resource health checking works.

*Note: `skaffold apply` always uses `kubectl` to deploy resources to a target cluster, regardless of deployment configuration in the provided skaffold.yaml. Only a small subset of deploy configuration is honored when running `skaffold apply`:*
* deploy.statusCheckDeadlineSeconds
* deploy.kubeContext
* deploy.logs.prefix
* deploy.kubectl.flags
* deploy.kubectl.defaultNamespace
* deploy.kustomize.flags
* deploy.kustomize.defaultNamespace

{{<alert title="Note">}}
`skaffold apply` attempts to honor the deployment configuration mentioned above.  But when conflicting configuration is detected in a multi-configuration project, `skaffold apply` will not work.
{{</alert>}}

`skaffold apply` works with any arbitrary Kubernetes YAML, whether it was generated by Skaffold or not, making it an ideal counterpart to `skaffold render`.

### GitOps-style continuous delivery

You can use this separation of rendering and deploying to enable GitOps CD pipelines.

GitOps-based CD pipelines traditionally see fully-hydrated Kubernetes manifests committed to a configuration Git repository (separate from the application source), which triggers a deployment pipeline that applies the changes to resources on the cluster.

### Example: Hydrate manifests then deploy/apply to cluster

First, use `skaffold render` to hydrate the Kubernetes resource file with a newly-built image tag:

```code
$ skaffold render --output render.yaml
```
```yaml
# render.yaml
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  namespace: default
spec:
  containers:
  - image: gcr.io/k8s-skaffold/skaffold-example:v1.19.0-89-gdbedd2a20-dirty
    name: getting-started
```

Then, we can apply this output directly to the cluster:

```code
$ skaffold apply render.yaml

Starting deploy...
 - pod/getting-started created
Waiting for deployments to stabilize...
Deployments stabilized in 49.277055ms
```
