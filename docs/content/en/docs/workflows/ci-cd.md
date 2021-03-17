---
title: "Continuous Delivery"
linkTitle: "Continuous Delivery"
weight: 40
---

Skaffold provides several features and sub-command "building blocks" that make it very useful for integrating with (or creating entirely new) CI/CD pipelines.
The ability to use the same `skaffold.yaml` for iterative development and continuous delivery eases handing off an application from a development team to an ops team.

Let's start with the simplest use case: a single, full deployment of your application.

## `skaffold run`
{{< maturity "run" >}}

`skaffold run` is a single command for a one-off deployment. It runs through every major phase of the Skaffold lifecycle: building your application images, tagging these images (and optionally pushing them to a remote registry), deploying your application to the target cluster, and monitoring the created resources for readiness.

We recommend `skaffold run` for the simplest Continuous Delivery setup, where it is sufficient to have a single step that deploys from version control to a cluster.

For more sophisticated Continuous Delivery pipelines, Skaffold offers building blocks:

- [healthcheck]({{<relref "/docs/workflows/ci-cd#waiting-for-skaffold-deployments-using-healthcheck">}}) - 
wait for `deployments` to stabilize and succeed only if all deployments are successful
- [`skaffold build`]({{<relref "/docs/workflows/ci-cd#skaffold-build-skaffold-deploy">}}) - build, tag and push artifacts to a registry
- [`skaffold deploy`]({{<relref "/docs/workflows/ci-cd#skaffold-build-skaffold-deploy">}})  - deploy built artifacts to a cluster
- [`skaffold render`]({{<relref "/docs/workflows/ci-cd#skaffold-render-skaffold-apply">}})  - export the transformed Kubernetes manifests for GitOps workflows
- [`skaffold apply`]({{<relref "/docs/workflows/ci-cd#skaffold-render-skaffold-apply">}}) - send hydrated Kubernetes manifests to the API server to create resources on the target cluster

## Waiting for Skaffold deployments using `healthcheck`
{{< maturity "deploy.status_check" >}}

`skaffold deploy` optionally performs a `healthcheck` for resources of kind [`Deployment`](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) and waits for them to be stable.
This feature can be very useful in Continuous Delivery pipelines to ensure that the deployed resources are
healthy before proceeding with the next steps in the pipeline.

{{<alert title="Note">}}
`healthcheck` is enabled by default; it can be disabled with the `--status-check=false` flag.
{{</alert>}}

To determine if a `Deployment` resource is up and running, Skaffold relies on `kubectl rollout status` to obtain its status.

```bash
Waiting for deployments to stabilize
 - default:deployment/leeroy-app Waiting for rollout to finish: 0 of 1 updated replicas are available...
 - default:deployment/leeroy-web Waiting for rollout to finish: 0 of 1 updated replicas are available...
 - default:deployment/leeroy-web is ready. [1/2 deployment(s) still pending]
 - default:deployment/leeroy-app is ready.
Deployments stabilized in 2.168799605s
```

**Configuring status check time for deploy `healthcheck`**

You can also configure the time for deployments to stabilize with the `statusCheckDeadlineSeconds` config field in the `skaffold.yaml`.

For example, to configure deployments to stabilize within 5 minutes:
{{% readfile file="samples/deployers/healthcheck.yaml" %}}

With the `--status-check` flag, for each `Deployment` resource, `skaffold deploy` will wait for
the time specified by [`progressDeadlineSeconds`](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#progress-deadline-seconds)
from the deployment configuration.

If the `Deployment.spec.progressDeadlineSeconds` is not set, Skaffold will either wait for

the time specified in the `statusCheckDeadlineSeconds` field of the deployment config stanza in the `skaffold.yaml`, or
default to 10 minutes if this is not specified.

In the case that both `statusCheckDeadlineSeconds` and `Deployment.spec.progressDeadlineSeconds` are set, precedence
is given to `Deployment.spec.progressDeadline` **only if it is less than** `statusCheckDeadlineSeconds`.

For example, the `Deployment` below with `progressDeadlineSeconds` set to 5 minutes,

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: getting-started
spec:
  progressDeadlineSeconds: 300
  template:
    spec:
      containers:
      - name: cannot-run
        image: gcr.io/k8s-skaffold/getting-started-foo
```

if the `skaffold.yaml` overrides the deadline to make sure deployment stabilizes in a 60 seconds,

```yaml
apiVersion: skaffold/v1
deploy:
  statusCheckDeadlineSeconds: 60
  kubectl:
    manifests:
    - k8s-*
```

Running `skaffold deploy`

```code
skaffold deploy --status-check
```
will result in an error after waiting for 1 minute:

```bash
Tags used in deployment:
Starting deploy...
kubectl client version: 1.11+
kubectl version 1.12.0 or greater is recommended for use with Skaffold
 - deployment.apps/getting-started created
Waiting for deployments to stabilize
 - default:deployment/getting-started Waiting for rollout to finish: 0 of 1 updated replicas are available...
 - default:deployment/getting-started failed. Error: received Ctrl-C or deployments could not stabilize within 1m: kubectl rollout status command interrupted.
FATA[0006] 1/1 deployment(s) failed
```

## Traditional continuous delivery: `skaffold build | skaffold deploy`

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


## GitOps-style continuous delivery: `skaffold render` | `skaffold apply`
{{< maturity "apply" >}}

GitOps-based CD pipelines traditionally see fully-hydrated Kubernetes manifests committed to a configuration Git repository (separate from the application source), which triggers a deployment pipeline that applies the changes to resources on the cluster. Skaffold has two built-in commands that enable easy GitOps pipeline workflows - `skaffold render` and `skaffold apply`.

`skaffold render` builds all application images from your artifacts, templates the newly-generated image tags into your Kubernetes manifests (based on your project's deployment configuration), and then prints out the final hydrated manifests to a file or your terminal. This allows you to capture the full, declarative state of your application in configuration rather than actually applying changes to your cluster, and use this configuration in a GitOps pipeline by committing it to a separate Git repository.

`skaffold apply` consumes one or more fully-hydrated Kubernetes manifests, and then sends the results directly to the Kubernetes control plane via `kubectl` to create resources on the target cluster. After creating the resources on your cluster, `skaffold apply` uses Skaffold's built-in health checking to monitor the created resources for readiness. See [resource health checks]({{<relref "/docs/workflows/ci-cd#waiting-for-skaffold-deployments-using-healthcheck">}}) for more information on how Skaffold's resource health checking works.

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

### Example: Hydrating Kubernetes resources using `skaffold render`, then sending them to the cluster using `skaffold apply`:

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
