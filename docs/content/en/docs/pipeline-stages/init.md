---
title: "Init"
linkTitle: "Init"
weight: 1
featureId: init
---

`skaffold init` is an easy way to get your project up and running in seconds.

Skaffold auto-generates `build` and `deploy` config for supported builders and deployers.

## Build Config Initialization

`skaffold init` currently supports build detection for those builders:

1. [Docker]({{<relref "/docs/pipeline-stages/builders/docker">}})
2. [Jib]({{<relref "/docs/pipeline-stages/builders/jib">}}) (with `--XXenableJibInit` flag)
2. [Buildpacks]({{<relref "/docs/pipeline-stages/builders/buildpacks">}}) (with `--XXenableBuildpacksInit` flag)

`skaffold init` walks your project directory and looks for any build configuration files such as `Dockerfile`,
`build.gradle/pom.xml`, `package.json`, `requirements.txt` or `go.mod`. `init` skips files that are larger
than 500MB.

If there are multiple build configuration files, Skaffold will prompt you to pair your build configuration files
with any images detected in your deploy configuration.

E.g. For an application with [two microservices](https://github.com/GoogleContainerTools/skaffold/tree/master/examples/microservices):

```bash
skaffold init
```
![microservices](/images/microservices-init-flow.png)


{{< alert title="Note" >}}
You can choose <code>None (image not built from these sources)</code> if none of the suggested 
options are correct, or this image is not built by any of your source code.<br>
If this image is one you want Skaffold to build, you'll need to manually set up the build configuration for this artifact.
{{</alert>}}

`skaffold` init also recognizes Maven and Gradle projects, and will auto-suggest the [`jib`]({{<relref "/docs/pipeline-stages/builders#/local#jib-maven-and-gradle">}}) builder.
Currently `jib` artifact detection is disabled by default, but can be enabled using the flag `--XXenableJibInit`.

You can try this out on our example [jib project](https://github.com/GoogleContainerTools/skaffold/tree/master/examples/jib-multimodule)

```bash
skaffold init --XXenableJibInit
```

![jib-multimodule](/images/jib-multimodule-init-flow.png)


## Deploy Config Initialization
`skaffold init` support bootstrapping projects set up to deploy with [`kubectl`]({{<relref "/docs/pipeline-stages/deployers#deploying-with-kubectl" >}})
or [`kustomize`]({{<relref "/docs/pipeline-stages/deployers#deploying-with-kubectl" >}}).
For projects deploying straight through `kubectl`, Skaffold will walk through all the `yaml` files in your project and find valid Kubernetes manifest files.

These files will be added to `deploy` config in `skaffold.yaml`.

```yaml
deploy:
  kubectl:
    manifests:
    - leeroy-app/kubernetes/deployment.yaml
    - leeroy-web/kubernetes/deployment.yaml
```

For projects deploying with `kustomize`, Skaffold will scan your project and look for `kustomization.yaml`s as well as Kubernetes manifests.
It will attempt to infer the project structure based on the recommended project structure from the Kustomize project: thus, 
**it is highly recommended to match your project structure to the recommended base/ and overlay/ structure from Kustomize!**

This generally looks like this:

```
app/      <- application source code, along with build configuration
  main.go
  Dockerfile
...
base/     <- base deploy configuration
  kustomization.yaml
  deployment.yaml
overlays/ <- one or more nested directories, each with modified environment configuration
  dev/
    deployment.yaml
    kustomization.yaml
  prod/
...
```

When overlay directories are found, these will be listed in the generated Skaffold config as `paths` in the `kustomize` deploy stanza. However, it generally does not make sense to have multiple overlays applied at the same time, so **Skaffold will attempt to choose a default overlay, and put each other overlay into its own profile**. This can be specified by the user through the flag `--default-kustomization`; otherwise, Skaffold will use the following heuristic:

1) Any overlay with the name `dev`
2) If none present, the **first** overlay that isn't named `prod`

*Note: order is guaranteed, since Skaffold's directory parsing is always deterministic.*

## Init API
`skaffold init` also exposes an API which tools like IDEs can integrate with via flags.

This API can be used to 

1. Analyze a project workspace and discover all build definitions (e.g. `Dockerfile`s) and artifacts (image names from the Kubernetes manifests) - this then provides an ability for tools to ask the user to pair the artifacts with Dockerfiles interactively. 
2. Given a pairing between the image names (artifacts) and build definitions (e.g. Dockerfiles), generate Skaffold `build` config for a given artifact.

The resulting `skaffold.yaml` will look something like this:

```yaml
apiVersion: skaffold/v2beta5
...
deploy:
  kustomize:
    paths:
    - overlays/dev
profiles:
- name: prod
  deploy:
    kustomize:
      paths:
      - overlays/prod
```

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

### Exit Codes

When `skaffold init` fails, it exits with an code that depends on the error:

| Exit Code | Error |
| ---- | --- |
| 101 | No build configuration could be found |
| 102 | No k8s manifest could be found or generated |
| 102 | An existing skaffold.yaml was found |
| 104 | Couldn't match builder with image names automatically |
