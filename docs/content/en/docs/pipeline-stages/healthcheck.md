---
title: "Deploy Healthchecks"
linkTitle: "Deploy Healthchecks"
weight: 50
featureId: deploy.status_check
aliases: [/docs/how-tos/healthchecks]
---

This page describes how skaffold runs healthchecks for deployed resources - waits for them to stabilize, and reports errors on failure.

### Overview

Commands that trigger a deployment like `skaffold dev`, `skaffold deploy`, `skaffold run`, and `skaffold apply` perform a `healthcheck` for select Kubernetes resources and wait for them to be stable.

Skaffold supports the following resource types for monitoring status after creation in the cluster:
* [`Pod`](https://kubernetes.io/docs/concepts/workloads/pods/)
* [`Deployment`](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) 
* [`Stateful Sets`](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/) 
* [`Google Cloud Config Connector resources`](https://cloud.google.com/config-connector/docs/overview)

{{<alert title="Note">}}
`healthcheck` is enabled by default; it can be disabled with the `--status-check=false`
flag, or by setting the `statusCheck` field of the deployment config stanza in
the `skaffold.yaml` to false.

If there are multiple skaffold `modules` active, then setting `statusCheck` field of the deployment config stanza will only disable healthcheck for that config. However using the `--status-check=false` flag will disable it for all modules.
{{</alert>}}

To determine if a `Deployment` or `StatefulSet` resource is up and running, Skaffold relies on `kubectl rollout status` to obtain its status. For standalone `Pods` it checks that the pod and its containers are in a `Ready` state.

```bash
Waiting for deployments to stabilize
 - default:deployment/leeroy-app Waiting for rollout to finish: 0 of 1 updated replicas are available...
 - default:deployment/leeroy-web Waiting for rollout to finish: 0 of 1 updated replicas are available...
 - default:deployment/leeroy-web is ready. [1/2 deployment(s) still pending]
 - default:deployment/leeroy-app is ready.
Deployments stabilized in 2.168799605s
```

### Configuring timeout for deploy `healthchecks`

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

### Configuring `healthchecks` for multiple deployers or multiple modules

If you define multiple deployers, say `kubectl`, `helm` and `kustomize`, all in the same skaffold config, or compose a multi-config project by importing other configs as dependencies, then the `healthcheck` can be run in one of two ways:
- _Single status check after all deployers are run_. This is the default and it runs a single `healthcheck` at the end for resources deployed from all deployers across all skaffold configs.
- _Per-deployer status check_. This can be enabled by using the `--iterative-status-check=true` flag. This will run a `healthcheck` iteratively after every individual deployer runs. This can be especially useful when there are startup dependencies between services, or you need to strictly enforce the time and order in which resources are deployed. 
