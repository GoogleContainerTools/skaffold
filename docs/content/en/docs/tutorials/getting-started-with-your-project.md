---
title: "Getting Started With Your Project"
linkTitle: "Getting Started With Your Project"
weight: 100
---

Skaffold requires a `skaffold.yaml`, but - for supported projects - Skaffold can generate a simple config for you that you can get started with. To configure Skaffold for your application you can run [`skaffold init`]({{<relref "docs/references/cli#skaffold-init" >}}).

Running `skaffold init` at the root of your project directory will walk you through a wizard
and create a `skaffold.yaml` with [build](#build-config-initialization) and [deploy](#deploy-config-initialization) config.

You can further set up [File Sync]({{<relref "docs/pipeline-stages/filesync" >}}) for file dependencies
that do not need a rebuild.

If your project contain resources other than services, you can set-up [port-forwarding]({{<relref "docs/pipeline-stages/port-forwarding" >}})
to port-forward these resources in [`dev`]({{<relref "docs/workflows/dev" >}}) or [`debug`]({{<relref "docs/workflows/debug" >}}) mode.

## Build Config Initialization
`skaffold init` currently supports build detection for two builders.

1. [Docker]({{<relref "/docs/pipeline-stages/builders#dockerfile-locally-with-docker">}})
2. [Jib]({{<relref "/docs/pipeline-stages/builders#jib-maven-and-gradle-locally">}})

`skaffold init` will walk your project directory and look for any `Dockerfiles` 
or `build.gradle/pom.xml`.

If you have multiple `Dockerfile` or `build.gradle/pom.xml` files, Skaffold will provide an option
to pair an image with one of the file.

E.g.:                                                                                                                                                                                                                                                                                                                                 For a multi-services [microservices example](https://github.com/GoogleContainerTools/skaffold/tree/master/examples/microservices)
```bash
skaffold init
? Choose the builder to build image gcr.io/k8s-skaffold/leeroy-app  [Use arrows to move, space to select, type to filter]
> Docker (leeroy-app/Dockerfile)
  Docker (leeroy-web/Dockerfile)
  None (image not built from these sources)

```


{{< alert title="Note" >}}
You can choose <code>None (image not built from these sources)</code> in case none of the suggested 
options are correct. <br>
You will have to manually set up build config for this artifact
{{</alert>}}

`skaffold` init also recognizes a maven or gradle project and will auto-suggest [`jib`]({{<relref "/docs/pipeline-stages/builders#jib-maven-and-gradle-locally">}}) builder.
Currently `jib` artifact detection is disabled by default, you can turn it on using the flag `--XXenableJibInit`.

You can try it this out on example [jib project](https://github.com/GoogleContainerTools/skaffold/tree/master/examples/jib-multimodule)
```bash
$cd examples/jib-multimodule
$skaffold init --XXenableJibInit
? Choose the builder to build image gcr.io/k8s-skaffold/skaffold-jib-1  [Use arrows to move, space to select, type to filter]
> Jib Maven Plugin (skaffold-project-1, pom.xml)
  Jib Maven Plugin (skaffold-project-2, pom.xml)
  None (image not built from these sources)
```


In case you want to configure build artifacts on your own, use `--skip-build` flag.

## Deploy Config Initialization
`skaffold init` currently supports only [`Kubeclt` deployer]({{<relref "/docs/pipeline-stages/deployers#deploying-with-kubectl" >}})
Skaffold will walk through all the `yaml` files in your project and find valid kubernetes manifest files.

These files will be added to `deploy` config in `skaffold.yaml`.

```yaml
deploy:
  kubectl:
    manifests:
    - leeroy-app/kubernetes/deployment.yaml
    - leeroy-web/kubernetes/deployment.yaml
```



