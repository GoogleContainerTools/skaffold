---
title: "Init"
linkTitle: "Init"
weight: 1
featureId: init
---

`skaffold init` is an easy way to get your project up and running in seconds.

Skaffold auto-generates `build` and `deploy` config for supported builders and deployers.


## Build Config Initialization
`skaffold init` currently supports build detection for two builders.

1. [Docker]({{<relref "/docs/pipeline-stages/builders/docker">}})
2. [Jib]({{<relref "/docs/pipeline-stages/builders/jib">}})

`skaffold init` will walk your project directory and look for any `Dockerfiles` 
or `build.gradle/pom.xml`. Please note, `skaffold init` skips files that are larger than 500MB.

If you have multiple `Dockerfile` or `build.gradle/pom.xml` files, Skaffold will provide an option
to pair an image with one of the file.

E.g. For a multi-services [microservices example](https://github.com/GoogleContainerTools/skaffold/tree/master/examples/microservices)

```bash
skaffold init
```
![microservices](/images/microservices-init-flow.png)


{{< alert title="Note" >}}
You can choose <code>None (image not built from these sources)</code> in case none of the suggested 
options are correct. <br>
You will have to manually set up build config for this artifact
{{</alert>}}

`skaffold` init also recognizes a maven or gradle project and will auto-suggest [`jib`]({{<relref "/docs/pipeline-stages/builders#/local#jib-maven-and-gradle">}}) builder.
Currently `jib` artifact detection is disabled by default, you can turn it on using the flag `--XXenableJibInit`.

You can try it this out on example [jib project](https://github.com/GoogleContainerTools/skaffold/tree/master/examples/jib-multimodule)

```bash
skaffold init --XXenableJibInit
```

![jib-multimodule](/images/jib-multimodule-init-flow.png)


In case you want to configure build artifacts on your own, use `--skip-build` flag.

## Deploy Config Initialization
`skaffold init` currently supports only [`kubectl` deployer]({{<relref "/docs/pipeline-stages/deployers#deploying-with-kubectl" >}})
Skaffold will walk through all the `yaml` files in your project and find valid kubernetes manifest files.

These files will be added to `deploy` config in `skaffold.yaml`.

```yaml
deploy:
  kubectl:
    manifests:
    - leeroy-app/kubernetes/deployment.yaml
    - leeroy-web/kubernetes/deployment.yaml
```


## Init API
`skaffold init` also exposes an api which tools like IDEs can integrate with via flags.

This API can be used to 

1. Analyze a project workspace and discover all build definitions (e.g. `Dockerfile`s) and artifacts (image names from the Kubernetes manifests) - this then provides an ability for tools to ask the user to pair the artifacts with Dockerfiles interactively. 
2. Given a pairing between the image names (artifacts) and build definitions (e.g. Dockerfiles), generate Skaffold `build` config for a given artifact.

**Init API contract**

| API | flag | input/output |
| ---- | --- | --- |
| Analyze | `--analyze` | json encoded output of builders and images|  
| Generate | `--artifact`| "`=` delimited" build definition/image pair (for example: `=path1/Dockerfile=artifact1`) or <br>JSON string (for example: `{"builder":"Docker","payload":{"path":"Dockerfile"},"image":"artifact")`|


### Analyze API
Analyze API walks through all files in your project workspace and looks for 
`Dockerfile` files.

To get all image names and dockerfiles, run
```bash
skaffold init --analyze | jq
```
will give you a json output
```json
{
  "dockerfiles": [
    "leeroy-app/Dockerfile",
    "leeroy-web/Dockerfile"
  ],
  "images": [
    "gcr.io/k8s-skaffold/leeroy-app",
    "gcr.io/k8s-skaffold/leeroy-web"
  ]
}
```

### Generate API
To generate a skaffold `build` config, use the `--artifact` flag per artifact.

For multiple artifacts, use `--artifact` multiple times.

```bash
microservices$skaffold init \
  -a '{"builder":"Docker","payload":{"path":"leeroy-app/Dockerfile"},"image":"gcr.io/k8s-skaffold/leeroy-app"}' \
  -a '{"builder":"Docker","payload":{"path":"leeroy-web/Dockerfile"},"image":"gcr.io/k8s-skaffold/leeroy-web"}'
```

will produce an `skaffold.yaml` config like this
```yaml
apiVersion: skaffold/v1
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
```
