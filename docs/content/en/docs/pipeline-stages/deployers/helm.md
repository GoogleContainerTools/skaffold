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
  repository: gcr.io/my-project/
  name: my-image
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
          image: {{ .Values.image.repository }}{{ .Values.image.name }}:{{ .Values.image.tag}}
          imagePullPolicy: {{ .Values.image.pullPolicy }}
```


Unfortunately, because of the way Skaffold continuously tags rebuilt images and hydrates Kubernetes resources before sending them to the cluster, it doesn't know how to handle images defined in this way.

**IMPORTANT: To get Skaffold to work with Helm, your image needs to be defined directly as a Helm value in the skaffold.yaml.**

This allows Skaffold to track the image being built, and correctly substitute it in the proper resource definition before sending it to Helm to be deployed to your cluster. In practice, this looks something like this:

```yaml
deploy:
  helm:
    releases:
    - name: my-release
      artifactOverrides:
        image: gcr.io/my-project/my-image # no tag present!
        # Skaffold continuously tags your image, so no need to put one here.
```

By configuring your project this way, note that **you may need to rewrite any templates referencing your image to reflect this new value!** Using the above templated `deployment.yaml` as an example, we would need to rewrite the template like so:

```project_root/templates/deployment.yaml:```
```yaml
spec:
  template:
    spec:
      containers:
        - name: {{ .Chart.Name }}
          image: {{ .Values.image }} # this value comes from the skaffold.yaml

          # this imagePullPolicy value is now invalid,
          # because it was overwritten through the `image` value from the skaffold.yaml!

          # let's redefine it in the `values.yaml` so we can keep it here.
          imagePullPolicy: {{ .Values.imageConfig.pullPolicy }}
```

and quickly redefining the `imagePullPolicy` in the ```project_root/values.yaml:```
```yaml
imageConfig:
  pullPolicy: IfNotPresent
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
