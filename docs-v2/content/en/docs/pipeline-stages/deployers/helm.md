---
title: "Helm [UPDATED]"
linkTitle: "Helm [UPDATED]"
weight: 40
featureId: deploy
---

[`helm`](https://helm.sh/) is a package manager for Kubernetes that helps you
manage Kubernetes applications. Skaffold natively supports iterative development
for projects configured to use helm.

{{< alert title="Note" >}}
To use `helm` with Skaffold, the `helm` binary must be installed on your machine. Skaffold will not install it for you.
{{< /alert >}}


# Configuring your Helm Project with Skaffold

Skaffold supports projects set up to render and/or deploy with Helm, but certain aspects of the project need to be configured correctly in order for Skaffold to work properly. This guide should demystify some of the nuance around using Skaffold with Helm to help you get started quickly.

{{< alert title="No more `artifactOverrides` or `imageStrategy`" >}}
Skaffold no longer requires the intricate configuring of `artifactOverrides` or `imageStrategy` fields. See docs [here]({{< relref "#image-reference-strategies" >}}) on how old `artifactOverrides` and `imageStrategy` values translate to `setValues` entires in the latest Skaffold schemas (`apiVersion: skaffold/v3alpha1` or skaffold binary version `v2.0.0` onwards)
{{< /alert >}}

{{< alert title="Note" >}}
In Skaffold `v2` the primary difference between the helm renderer (`manifest.helm.*`) and the helm deployer (`deploy.helm.*`) is the use of `helm template` vs `helm install`
{{< /alert >}}

## How `helm` render support works in Skaffold
In the latest version of Skaffold, the primary methods of using `helm` templating with Skaffold involve the `deploy.helm.setValues` and the `deploy.helm.setValueTemplates` fields.  `deploy.helm.setValues` supplies the key:value pair to substitute from a users `values.yaml` file (a standard `helm` file for rendering).  `deploy.helm.setValueTemplates` does a similar thing only the key:value value comes from an environment variable instead of a given value.  The thing to note here is that when using a value that is the same name as an artifact name, that value will be replaced by the fully qualified artifact name (eg: `image: gcr.io/example-repo/skaffold-helm-image:latest@sha256:<sha256-hash>`) for the current build (or the specified value from additional configuration eg: CLI flags or config from `skaffold.yaml`).  Depending on how a user's `values.yaml` and charts specify `image: $IMAGE_TEMPLATE`, the docs [here]({{< relref "#image-reference-strategies" >}})  explain the proper `setValues` to use.  When migrating from schema version `v2beta29` or less, Skaffold will automatically configure these values to continue to work.


`helm` deploy support in Skaffold is accomplished by calling `helm install ...` and using the `skaffold` binary as a `helm` `--post-renderer`.  This works by having Skaffold run `helm install ...` taking into consideration all of the supplied flags, skaffold.yaml configuration, etc. and creating an intermediate yaml manifest with all helm replacements except that the fully qualified image from the current run is NOT added but instead a placeholder with the artifact name - eg: `skaffold-helm-image`.  Then the skaffold post-renderer is called to convert `image: skaffold-helm-image` -> `image: gcr.io/example-repo/skaffold-helm-image:latest@sha256:<sha256-hash>` in specified locations (specific allowlisted k8s objects and/or k8s object fields).  This above replacement is nearly identical to how it works for values.yaml files using only the `image` key in `values.yaml` - eg:
`image: "{{.Values.image}}"`

When using `image.repoistory` + `image.tag` or `image.registry` + `image.repository` + `image.tag` - eg:
`image: "{{.Values.image.repository}}:{{.Values.image.tag}}"`
`image: "{{.Values.image.registry}}/{{.Values.image.repository}}:{{.Values.image.tag}}"`
there is some specialized logic that the skaffold `post-renderer` uses to properly handling these cases.  See the docs [here]({{< relref "#image-reference-strategies" >}}) on the correct way to specify these for Skaffold using `setValues`

## Image Configuration
The normal Helm convention for defining image references is through the `values.yaml` file. Often, image information is configured through an `image` stanza in the values file, which might look something like this:

```project_root/values.yaml```
```yaml
image:
  repository: my-image
  tag: v1.2.0
  pullPolicy: IfNotPresent
```

This image would then be referenced in a templated resource file, maybe like this:

```project_root/templates/deployment.yaml:```
```yaml
spec:
  template:
    spec:
      containers:
        - name: {{ .Chart.Name }}
          image: {{ .Values.image.repository }}:{{ .Values.image.tag}}
          imagePullPolicy: {{ .Values.image.pullPolicy }}
```

**IMPORTANT: To get Skaffold to work with Helm, the `image` key must be configured in the skaffold.yaml.**

Associating the Helm image key allows Skaffold to track the image being built, and then configure Helm to substitute it in the proper resource definitions to be deployed to your cluster. In practice, this looks something like this:

```yaml
build:
  artifacts:
    - image: my-image # must match in setValues
    - image: my-image-2 # must match in setValues
deploy:
  helm:
    releases:
    - name: my-release
      setValues:
        image: my-image # no tag present!
        image2: my-image-2 # no tag present!
```

The `setValues` configuration binds a Helm key to the specified value.

### Multiple image overrides

To override multiple images (ie a Pod with a side car) you can simply add additional variables. For example, the following helm install:

```yaml
spec:
  containers:
    - name: firstContainer
      image: "{{.Values.firstContainerImage}}"
      ....
    - name: secondContainer
      image: "{{.Values.secondContainerImage}}"
      ...
```

can be overriden with:

```yaml
deploy:
  helm:
    releases:
    - name: my-release
      setValues:
        firstContainerImage: gcr.io/my-project/first-image # no tag present!
        secondContainerImage: gcr.io/my-project/second-image # no tag present!
```

### Image reference strategies

Skaffold supports three _image reference strategies_ for Helm:

1. `fqn`: provides a fully-qualified image reference (default);
2. `helm`: provides separate repository and tag portions (shown above);
3. `helm+explicitRegistry`: provides separate registry, repository, and tag portions.

#### `fqn` strategy: single fully-qualified name (default)

With the fully-qualified name strategy, Skaffold configures Helm by setting a key to the fully-tagged image reference.

The `skaffold.yaml` setup:
```yaml
build:
  artifacts:
    - image: my-image  # must match in setValues
    - image: my-image-2  # must match in setValues
deploy:
  helm:
    releases:
      - name: my-chart
        chartPath: helm
        setValues:
          image: my-image # no tag present!
          image2: my-image-2 # no tag present!
```
The `values.yaml` (note that Skaffold overrides this value):
```
image: gcr.io/other-project/other-image:latest
image2: gcr.io/other-project/other-image:latest
```

The chart template:
```yaml
spec:
  containers:
    - name: {{ .Chart.Name }}
      image: "{{.Values.image}}"
    - name: {{ .Chart.Name }}
      image: "{{.Values.image2}}"
```

Skaffold will invoke
```
helm install <chart> <chart-path> --set-string image=<artifact-name>,image2=<artifact-name> --post-renderer=<path-to-skaffold-binary-from-original-invocation>
```

#### `helm` strategy: split repository and tag

Skaffold can be configured to provide Helm with a separate repository and tag.  The key used in the `artifactOverrides` is used as base portion producing two keys `{key}.repository` and `{key}.tag`.

The `skaffold.yaml` setup:
```yaml
build:
  artifacts:
    - image: my-image # must match in setValues
    - image: my-image-2 # must match in setValues
deploy:
  helm:
    releases:
      - name: my-chart
        chartPath: helm
        setValues:
          image.repository: my-image
          image.tag: my-image
          image2.repository: my-image-2
          image2.tag: my-image-2
```

The `values.yaml` (note that Skaffold overrides these values):
```
image:
  repository: gcr.io/other-project/other-image
  tag: latest
image2:
  repository: gcr.io/other-project/other-image
  tag: latest
```

The chart template:
```yaml
spec:
  containers:
    - name: {{ .Chart.Name }}
      image: "{{.Values.image.repository}}:{{.Values.image.tag}}"
    - name: {{ .Chart.Name }}
      image: "{{.Values.image2.repository}}:{{.Values.image2.tag}}"
```

Skaffold will invoke
```
helm install <chart> <chart-path> --set-string image.repository=<artifact-name>,image.tag=<artifact-name>,image2.repository=<artifact-name>,image2.tag=<artifact-name>  --post-renderer=<path-to-skaffold-binary-from-original-invocation>
```

#### `helm`+`explicitRegistry` strategy: split registry, repository, and tag

Skaffold can also be configured to provide Helm with a separate repository and tag.  The key used in the `artifactOverrides` is used as base portion producing three keys: `{key}.registry`, `{key}.repository`, and `{key}.tag`.

The `skaffold.yaml` setup:
```yaml
build:
  artifacts:
    - image: my-image # must match in setValues
    - image: my-image-2 # must match in setValues
deploy:
  helm:
    releases:
      - name: my-chart
        chartPath: helm
        setValues:
          image.registry: my-image
          image.repository: my-image
          image.tag: my-image
          image2.registry: my-image-2
          image2.repository: my-image-2
          image2.tag: my-image-2
```

The `values.yaml` (note that Skaffold overrides these values):
```
image:
  registry: gcr.io
  repository: other-project/other-image
  tag: latest
image2:
  registry: gcr.io
  repository: other-project/other-image
  tag: latest
```

The chart template:
```yaml
spec:
  containers:
    - name: {{ .Chart.Name }}
      image: "{{.Values.image.registry}}/{{.Values.image.repository}}:{{.Values.image.tag}}"
```

Skaffold will invoke
```
helm install <chart> <chart-path> --set-string image.registry=<artifact-name>,image.repository=<artifact-name>,image.tag=<artifact-name>,image2.registry=<artifact-name>,image2.repository=<artifact-name>,image2.tag=<artifcact-name>  --post-renderer=<path-to-skaffold-binary-from-original-invocation>
```

### Helm Build Dependencies

The `skipBuildDependencies` flag toggles whether dependencies of the Helm chart are built with the `helm dep build` command. This command manipulates files inside the `charts` subfolder of the specified Helm chart.

If `skipBuildDependencies` is `false` then `skaffold dev` does **not** watch the `charts` subfolder of the Helm chart, in order to prevent a build loop - the actions of `helm dep build` always trigger another build.

If `skipBuildDependencies` is `true` then `skaffold dev` watches all files inside the Helm chart.

### `skaffold.yaml` Configuration

The `helm` type offers the following options:

{{< schema root="HelmDeploy" >}}

Each `release` includes the following fields:

{{< schema root="HelmRelease" >}}
