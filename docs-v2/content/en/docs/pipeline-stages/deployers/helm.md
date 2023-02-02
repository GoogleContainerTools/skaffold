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
Skaffold no longer requires the intricate configuring of `artifactOverrides` or `imageStrategy` fields. See docs [here]({{< relref "#image-reference-strategies" >}}) on how old `artifactOverrides` and `imageStrategy` values translate to `setValueTemplates` entries in the latest Skaffold schemas (`apiVersion: skaffold/v3alpha1` or skaffold binary version `v2.0.0` onwards)
{{< /alert >}}

{{< alert title="Note" >}}
In Skaffold `v2` the primary difference between the helm renderer (`manifest.helm.*`) and the helm deployer (`deploy.helm.*`) is the use of `helm template` vs `helm install`
{{< /alert >}}


## How `helm` render support works in Skaffold
In the latest version of Skaffold, the primary methods of using `helm` templating with Skaffold involve the `deploy.helm.setValueTemplates` and the `deploy.helm.setValues` fields.  `deploy.helm.setValues` supplies the key:value pair to substitute from a users `values.yaml` file (a standard `helm` file for rendering).  `deploy.helm.setValueTemplates` does a similar thing only the key:value value comes from an environment variable instead of a given value. Depending on how a user's `values.yaml` and how `charts/templates` specify `image: $IMAGE_TEMPLATE`, the docs [here]({{< relref "#image-reference-strategies" >}})  explain the proper `setValueTemplates` to use.  When migrating from schema version `v2beta29` or less, Skaffold will automatically configure these values to continue to work.


`helm` deploy support in Skaffold is accomplished by calling `helm install ...` with the appropriate `--set` flags for the variables Skaffold will inject as well as uses the `skaffold` binary as a `helm` `--post-renderer`.  Using `skaffold` as a post-renderer is done to inject Skaffold specific labels primarily the `run-id` label which Skaffold uses to tag K8s objects it will manage via it's status checking.


This works by having Skaffold run `helm install ...` taking into consideration all of the supplied flags, skaffold.yaml configuration, etc. and creating an intermediate yaml manifest with all helm replacements except that the fully qualified image from the current run is NOT added but instead a placeholder with the artifact name - eg: `skaffold-helm-image`.  Then the skaffold post-renderer is called to convert `image: skaffold-helm-image` -> `image: gcr.io/example-repo/skaffold-helm-image:latest@sha256:<sha256-hash>` in specified locations (specific allowlisted k8s objects and/or k8s object fields).  This above replacement is nearly identical to how it works for values.yaml files using only the `image` key in `values.yaml` - eg:
`image: "{{.Values.image}}"`

When using `image.repository` + `image.tag` or `image.registry` + `image.repository` + `image.tag` - eg:
`image: "{{.Values.image.repository}}:{{.Values.image.tag}}"`
`image: "{{.Values.image.registry}}/{{.Values.image.repository}}:{{.Values.image.tag}}"`
there is some specialized logic that the skaffold `post-renderer` uses to properly handling these cases.  See the docs [here]({{< relref "#image-reference-strategies" >}}) on the correct way to specify these for Skaffold using `setValueTemplates`

{{< alert title="Note" >}}
Starting in Skaffold `v2.1.0`, Skaffold will output additional `setValueTemplates`
{{< /alert >}}

## Image Configuration
The normal Helm convention for defining image references is through the `values.yaml` file. Often, image information is configured through an `image` stanza in the values file, which might look something like this:

```project_root/values.yaml```
```yaml
image:
  repository: gcr.io/my-repo # default repo
  tag: v1.2.0 # default tag 
  pullPolicy: IfNotPresent # default PullPolicy
image2:
  repository: gcr.io/my-repo-2 # default repo
  tag: latest # default tag 
  pullPolicy: IfNotPresent # default PullPolicy
```

This images would then be referenced in a templated resource file, maybe like this:

```project_root/templates/deployment.yaml:```
```yaml
spec:
  template:
    spec:
      containers:
        - name: {{ .Chart.Name }}
          image: {{ .Values.image.repository }}:{{ .Values.image.tag}}
          imagePullPolicy: {{ .Values.image.pullPolicy }}
        - name: {{ .Chart.Name }}
          image: {{ .Values.image2.repository }}:{{ .Values.image2.tag}}
          imagePullPolicy: {{ .Values.image2.pullPolicy }}
```

**IMPORTANT: To get Skaffold to work with Helm, the `image` key must be configured in the skaffold.yaml.**

Associating the Helm image key allows Skaffold to track the image being built, and then configure Helm to substitute it in the proper resource definitions to be deployed to your cluster. In practice, this looks something like this:

```yaml
build:
  artifacts:
    - image: myFirstImage # must match in setValueTemplates
    - image: mySecondImage # must match in setValueTemplates
deploy:
  helm:
    releases:
    - name: my-release
      setValueTemplates:
        image.repository: "{{.IMAGE_REPO_myFirstImage}}"
        image.tag: "{{.IMAGE_TAG_myFirstImage}}@{{.IMAGE_DIGEST_myFirstImage}}"
        image2.repository: "{{.IMAGE_REPO_mySecondImage}}"
        image2.tag: "{{.IMAGE_TAG_mySecondImage}}@{{.IMAGE_DIGEST_mySecondImage}}"
      setValues:
        image.pullPolicy: "IfNotPresent"
        image2.pullPolicy: "IfNotPresent"
```

The `setValues` configuration binds a Helm key to the specified value. The `setValueTemplates` configuration binds a Helm key to an environment variable.  Skaffold generates useful environment variables (available via `setValueTemplates`) for each build artifact (value in build.artifacts\[x\].image).  Currenty these include:

| Helm Template Value | Example Value |
| --------------- | --------------- |
| `{{.IMAGE_FULLY_QUALIFIED_<artifact-name>}}` | `gcr.io/example-repo/myImage:latest@sha256:<sha256-hash>`|
| `{{.IMAGE_REPO_<artifact-name>}}` | `gcr.io/example-repo/myImage` |
| `{{.IMAGE_TAG_<artifact-name>}}` | `latest` |
| `{{.IMAGE_DIGEST_<artifact-name>}}` | `sha256:<sha256-hash>` |
| `{{.IMAGE_DOMAIN_<artifact-name>}}` | `gcr.io` |
| `{{.IMAGE_REPO_NO_DOMAIN_<artifact-name>}}` | `example-repo` |

### Sanitizing the artifact name from invalid go template characters
The `<artifact-name>` (eg: `{{.IMAGE_FULLY_QUALIFIED_<artifact-name>}}`) when used with `setValueTemplates` cannot have `/`, `-`, or `.` characters.  If you have an artifact name with these characters (eg: `localhost/nginx` or `gcr.io/foo-image/foo`), change them to use `_` in place of these characters in the `setValueTemplates` field

| Artifact name | Sanitized Name |
| --------------- | --------------- |
| `localhost/nginx` | `localhost_nginx`|
| `gcr.io/example-repo/myImage` | `gcr_io_example_repo_myImage` |

Example
```yaml
build:
  artifacts:
    - image: localhost/nginx  # must match in setValueTemplates w/ `/` & `-` changed to `_`
deploy:
  helm:
    releases:
      - name: my-chart
        chartPath: helm
        setValueTemplates:
          image: {{.IMAGE_FULLY_QUALIFIED_localhost_nginx}}

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
    - image: myFirstImage  # must match in setValueTemplates
    - image: mySecondImage  # must match in setValueTemplates
deploy:
  helm:
    releases:
      - name: my-chart
        chartPath: helm
        setValueTemplates:
          image: {{.IMAGE_FULLY_QUALIFIED_myFirstImage}}
          image2: {{.IMAGE_FULLY_QUALIFIED_mySecondImage}}
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
    - image: myFirstImage # must match in setValueTemplates
    - image: mySecondImage # must match in setValueTemplates
deploy:
  helm:
    releases:
      - name: my-chart
        chartPath: helm
        setValueTemplates:
          image.repository: "{{.IMAGE_REPO_myFirstImage}}"
          image.tag: "{{.IMAGE_TAG_myFirstImage}}@{{.IMAGE_DIGEST_myFirstImage}}"
          image2.repository: "{{.IMAGE_REPO_mySecondImage}}"
          image2.tag: "{{.IMAGE_TAG_mySecondImage}}@{{.IMAGE_DIGEST_mySecondImage}}"
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
    - image: myFirstImage # must match in setValueTemplates
    - image: mySecondImage # must match in setValueTemplates
deploy:
  helm:
    releases:
      - name: my-chart
        chartPath: helm
        setValueTemplates:
          image.registry: "{{.IMAGE_DOMAIN_myFirstImage}}"
          image.repository: "{{.IMAGE_REPO_NO_DOMAIN_myFirstImage}}"
          image.tag: "{{.IMAGE_TAG_myFirstImage}}@{{.IMAGE_DIGEST_myFirstImage}}"
          image2.registry: "{{.IMAGE_DOMAIN_mySecondImage}}"
          image2.repository: "{{.IMAGE_REPO_NO_DOMAIN_mySecondImage}}"
          image2.tag: "{{.IMAGE_TAG_mySecondImage}}@{{.IMAGE_DIGEST_mySecondImage}}"
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
