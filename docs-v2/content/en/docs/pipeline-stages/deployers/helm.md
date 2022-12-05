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

Skaffold supports projects set up to deploy with Helm, but certain aspects of the project need to be configured correctly in order for Skaffold to work properly. This guide should demystify some of the nuance around using Skaffold with Helm to help you get started quickly.

{{< alert title="No more `artifactOverrides`" >}}
Skaffold no longer requires the intricate configuring of `artifactOverrides` and image naming strategies.
{{< /alert >}}


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

{{% readfile file="samples/helm/helmClusterDeploy.yaml" %}}

The `artifactOverrides` binds a Helm value key to a build artifact.  The `imageStrategy` configures the image reference strategy for informing Helm of the image reference to a newly built artifact.

### Multiple image overrides

To override multiple images (ie a Pod with a side car) you can simply add additional variables. For example, the following helm template:

```yaml
spec:
  containers:
    - name: firstContainer
      image: "{{.Values.firstContainerImage}}"
      # ....
    - name: secondContainer
      image: "{{.Values.secondContainerImage}}"
       # ...
```

can be overriden with:

```
deploy:
  helm:
    releases:
    - name: my-release
      artifactOverrides:
        firstContainerImage: gcr.io/my-project/first-image # no tag present!
        secondContainerImage: gcr.io/my-project/second-image # no tag present!
      imageStrategy:
        helm: {}
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
