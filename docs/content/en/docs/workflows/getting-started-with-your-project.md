---
title: "Getting Started With Your Project"
linkTitle: "Getting Started With Your Project"
weight: 10
---

Skaffold requires a `skaffold.yaml`, but - for supported projects - Skaffold can generate a simple config for you that you can get started with. To configure Skaffold for your application you can run [`skaffold init`]({{<relref "docs/references/cli#skaffold-init" >}}).

Running `skaffold init` at the root of your project directory will walk you through a wizard
and create a `skaffold.yaml` with [build](#build-config-initialization) and [deploy](#deploy-config-initialization) config.

```bash
microservices$ skaffold init
? Choose the builder to build image gcr.io/k8s-skaffold/leeroy-app Docker (leeroy-app/Dockerfile)
? Choose the builder to build image gcr.io/k8s-skaffold/leeroy-web Docker (leeroy-web/Dockerfile)
apiVersion: skaffold/v1beta15
kind: Config
metadata:
  name: microservices
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/leeroy-app
    context: leeroy-app
  - image: gcr.io/k8s-skaffold/leeroy-web
    context: leeroy-web
deploy:
  kubectl:
    manifests:
    - leeroy-app/kubernetes/deployment.yaml
    - leeroy-web/kubernetes/deployment.yaml

Do you want to write this configuration to skaffold.yaml? [y/n]: y
Configuration skaffold.yaml was written
You can now run [skaffold build] to build the artifacts
or [skaffold run] to build and deploy
or [skaffold dev] to enter development mode, with auto-redeploy
```

You can further set up [File Sync]({{<relref "docs/pipeline-stages/filesync" >}}) for file dependencies
that do not need a rebuild.

If your project contain resources other than services, you can set-up [port-forwarding]({{<relref "docs/pipeline-stages/port-forwarding" >}})
to port-forward these resources in [`dev`]({{<relref "docs/workflows/dev" >}}) or [`debug`]({{<relref "docs/workflows/debug" >}}) mode.


## What's next

For more understanding on how init works, see [`skaffold init`]({{<relref "docs/pipeline-stages/init" >}})

Try out [dev]({{<relref "docs/workflows/dev" >}}), [debug]({{<relref "docs/workflows/debug" >}}) workflows.



