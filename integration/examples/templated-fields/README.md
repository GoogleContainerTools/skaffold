### Example: use image values in templated fields for build and deploy 

This example shows how `IMAGE_REPO` and `IMAGE_TAG` keywords are available in templated fields for custom build and helm deploy

* **building** a single Go file app with ko
* **tagging** using the default tagPolicy (`gitCommit`)
* **deploying** two replica container pods using `helm`

#### Before you begin

For this tutorial to work, you will need to have Skaffold, Helm and a Kubernetes cluster set up.
To learn more about how to set up, see the [getting started docs](https://skaffold.dev/docs/getting-started).

#### Tutorial

This tutorial will demonstrate how Skaffold can inject image repo and image tag values into the build and deploy stanzas.

First, clone the Skaffold [repo](https://github.com/GoogleContainerTools/skaffold) and navigate to the [templated-fields example](https://github.com/GoogleContainerTools/skaffold/tree/master/examples/templated-fields) for sample code:

```shell
$ git clone https://github.com/GoogleContainerTools/skaffold
$ cd skaffold/examples/templated-fields
```

`IMAGE_REPO` and `IMAGE_TAG` are available as templated fields in `build.sh` file

```shell
#from build.sh, line 13
img="${IMAGE_REPO}:${IMAGE_TAG}"
```

and also in the `helm` deploy section of the skaffold config, which configures artifact `skaffold-templated` to build with `build.sh`:

```yaml
// from skaffold.yaml, line 22-24
      setValueTemplates:
          imageRepo: "{{.IMAGE_REPO}}"
          imageTag: "{{.IMAGE_TAG}}"
```

These values are then being set as container environment variables `FOO_IMAGE_REPO` and `FOO_IMAGE_TAG` in the helm template `deployment.yaml` file, just as an example to show how they can be added to your helm templates.

```yaml
// from charts/templates/deployment.yaml, line 16-24
      containers:
      - name: {{ .Chart.Name }}
        image: {{ .Values.image }}
        env:
          - name: FOO_IMAGE_REPO
            value: {{ .Values.imageRepo }}
          - name: FOO_IMAGE_TAG
            value: {{ .Values.imageTag }}
```

For more information about how this works, see the Skaffold [custom builder](https://skaffold.dev/docs/how-tos/builders/#custom-build-script-run-locally) and [helm](https://skaffold.dev/docs/pipeline-stages/deployers/helm/) documentation.

Now, use Skaffold to deploy this application to your Kubernetes cluster:

```shell
$ skaffold run --tail --default-repo <your repo>
```

With this command, Skaffold will build the `skaffold-templated` artifact with ko and deploy the application to Kubernetes using helm.
You should be able to see something like:

```shell
Running image skaffold-templated:a866d5efd634062ea74662b20e172cd6e2d645f9f33f929bfaf8e856ec66bd94
```

 printed every second in the Skaffold logs, since the code being executed is `main.go`.
 
```go
// from main.go, line 12
fmt.Printf("Running image %v:%v\n", os.Getenv("FOO_IMAGE_REPO"), os.Getenv("FOO_IMAGE_TAG"))
```

#### Cleanup

To clean up your Kubernetes cluster, run:

```shell
$ skaffold delete
```
