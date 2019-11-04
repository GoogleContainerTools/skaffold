---
title: "Init"
linkTitle: "Init"
weight: 1
---

`skaffold init` is an easy way to get your project up and running in seconds.

Skaffold auto-generates `build` and `deploy` config for supported builders and deployers.


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


## Init API
`skaffold init` also exposes an api which tools like IDEs can integrate with via flags.

This API can be used to 

1. Analyze a project workspace and discover all build definitions (e.g. `Dockerfile`s) and artifacts (image names from the Kubernetes manifests) - this then provides an ability for tools to ask the user to pair the artifacts with Dockerfiles interactively. 
2. Given a pairing between the image names (artifacts) and build definitions (e.g. Dockerfiles), generate Skaffold `build` config for a given artifact.

**Init API contract**

| API | flag | input/output |
| ---- | --- | --- |
| Analyze | `--analyze` and `--XXenableJibInit`| json encoded output of builders and images|  
| Generate | `--artifact`| "`=` delimited" build definition/image pair (for example: `=path1/Dockerfile=artifact1`) or JSON string (for example: ...) |


### Analyze API
Analyze API walks through all files in your project workspace and looks for 
`Dockerfile`, `build.gradle` and `pom.xml` files.

To get all image names and image builders, run
```json
skaffold init --analyze --XXenableJibInit | jq
{
  {
    "builders": [
      {
        "name": "Docker",
        "payload": {
          "path": "microservices/leeroy-app/Dockerfile"
        }
      },
      {
        "name": "Jib Maven Plugin",
        "payload": {
          "image": "gcr.io/k8s-skaffold/project1",
          "path": "pom.xml",
          "project": "skaffold-project-1"
        }
      },
      {
        "name": "Jib Maven Plugin",
        "payload": {
          "image": "gcr.io/k8s-skaffold/project2",
          "path": "pom.xml",
          "project": "skaffold-project-2"
        }
      }
    ],
    "images": [
      {
        "name": "gcr.io/k8s-skaffold/skaffold-jib-1",
        "foundMatch": false
      },
      {
        "name": "gcr.io/k8s-skaffold/skaffold-jib-2",
        "foundMatch": false
      },
      {
        "name": "gcr.io/k8s-skaffold/leeroy-app",
        "foundMatch": false
      },
    ]
  }
}
```

### Generate API
To generate a skaffold `build` config, use the `--artifact` flag per artifact.

For multiple artifacts, use `--artifact` multiple times.

```bash
multimodule$skaffold init \
  -a '{"builder":"Docker","payload":{"path":"web/Dockerfile.web"},"image":"gcr.io/web-project/image"}' \
  -a '{"builder":"Jib Maven Plugin","payload":{"path":"backend/pom.xml"},"image":"gcr.io/backend/image"}' \
  --XXenableJibInit

apiVersion: skaffold/v1beta15
kind: Config
metadata:
  name: multimodule
build:
  artifacts:
  - image: gcr.io/web-project/image
    context: web
    docker:
      dockerfile: web/Dockerfile.web
  - image: gcr.io/backend/image
    context: backend
    jib:
      args:
      - -Dimage=gcr.io/backend/image
deploy:
  kubectl:
    manifests:
    - web/kubernetes/web.yaml
    - backend/kubernetes/deployment.yaml

```
