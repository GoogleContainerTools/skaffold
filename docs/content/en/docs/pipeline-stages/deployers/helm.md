---
title: "Helm"
linkTitle: "Helm"
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

Skaffold supports projects set up to deploy with Helm, but certain aspects of the project need to be configured correctly in order for Skaffold to work properly. This guide should demystify some of the nuance around using Skaffold with Helm to help you get started quickly.

## Image Configuration
The normal Helm convention for defining image references is through the `values.yaml` file. Often, image information is configured through an `image` stanza in the values file, which might look something like this:

```project_root/values.yaml```
```yaml
image:
  repository: gcr.io/my-project/my-image
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
    - image: gcr.io/my-project/my-image # must match in artifactOverrides
deploy:
  helm:
    releases:
    - name: my-release
      artifactOverrides:
        image: gcr.io/my-project/my-image # no tag present!
      imageStrategy:
        helm: {}
```

The `artifactOverrides` binds a Helm value key to a build artifact.  The `imageStrategy` configures the image reference strategy for informing Helm of the image reference to a newly built artifact.

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
    - image: gcr.io/my-project/my-image
deploy:
  helm:
    releases:
      - name: my-chart
        chartPath: helm
        artifactOverrides:
          imageKey: gcr.io/my-project/my-image
        imageStrategy:
          fqn: {}
```

Note that the `fqn` strategy is the default and the `imageStrategy` can be omitted.

The `values.yaml` (note that Skaffold overrides this value):
```
imageKey: gcr.io/other-project/other-image:latest
```

The chart template:
```yaml
spec:
  containers:
    - name: {{ .Chart.Name }}
      image: "{{.Values.imageKey}}"
```

Skaffold will invoke
```
helm install <chart> <chart-path> --set-string imageKey=gcr.io/my-project/my-image:generatedTag@sha256:digest
```

#### `helm` strategy: split repository and tag

Skaffold can be configured to provide Helm with a separate repository and tag.  The key used in the `artifactOverrides` is used as base portion producing two keys `{key}.repository` and `{key}.tag`.

The `skaffold.yaml` setup:
```yaml
build:
  artifacts:
    - image: gcr.io/my-project/my-image
deploy:
  helm:
    releases:
      - name: my-chart
        chartPath: helm
        artifactOverrides:
          imageKey: gcr.io/my-project/my-image
        imageStrategy:
          helm: {}
```

The `values.yaml` (note that Skaffold overrides these values):
```
imageKey:
  repository: gcr.io/other-project/other-image
  tag: latest
```

The chart template:
```yaml
spec:
  containers:
    - name: {{ .Chart.Name }}
      image: "{{.Values.imageKey.repository}}:{{.Values.imageKey.tag}}"
```

Skaffold will invoke
```
helm install <chart> <chart-path> --set-string imageKey.repository=gcr.io/my-project/my-image,imageKey.tag=generatedTag@sha256:digest
```

#### `helm`+`explicitRegistry` strategy: split registry, repository, and tag

Skaffold can also be configured to provide Helm with a separate repository and tag.  The key used in the `artifactOverrides` is used as base portion producing three keys: `{key}.registry`, `{key}.repository`, and `{key}.tag`.

The `skaffold.yaml` setup:
```yaml
build:
  artifacts:
    - image: gcr.io/my-project/my-image
deploy:
  helm:
    releases:
      - name: my-chart
        chartPath: helm
        artifactOverrides:
          imageKey: gcr.io/my-project/my-image
        imageStrategy:
          helm:
            explicitRegistry: true
```

The `values.yaml` (note that Skaffold overrides these values):
```
imageKey:
  registry: gcr.io
  repository: other-project/other-image
  tag: latest
```

The chart template:
```yaml
spec:
  containers:
    - name: {{ .Chart.Name }}
      image: "{{.Values.imageKey.registry}}/{{.Values.imageKey.repository}}:{{.Values.imageKey.tag}}"
```

Skaffold will invoke
```
helm install <chart> <chart-path> --set-string imageKey.registry=gcr.io,imageKey.repository=my-project/my-image,imageKey.tag=generatedTag@sha256:digest
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
