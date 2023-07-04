---
title: "Templated Fields"
linkTitle: "Templated Fields"
weight: 90
featureId: templating
aliases: [/docs/how-tos/templating]
---

Skaffold allows for certain fields in the config to be templated with values either from environment variables, or certain special values computed by Skaffold.

{{% readfile file="samples/templating/env.yaml" %}}

Suppose the value of the `FOO` environment variable is `v1`, the image built
will be `gcr.io/k8s-skaffold/example:v1`.

List of fields that support templating:

* `build.artifacts[].docker.buildArgs` (see [builders]({{< relref "/docs/builders" >}}))
* `build.artifacts[].ko.{env,flags,labels,ldflags}` (see [`ko` builder]({{< relref "/docs/builders/builder-types/ko" >}}))
* `build.tagPolicy.envTemplate.template` (see [envTemplate tagger]({{< relref "/docs/taggers#envtemplate-using-values-of-environment-variables-as-tags)" >}}))
* `deploy.helm.releases[].chartPath` (see [Deploying with helm]({{< relref "/docs/deployers#deploying-with-helm)" >}}))
* `deploy.helm.releases[].name` (see [Deploying with helm]({{< relref "/docs/deployers#deploying-with-helm)" >}}))
* `deploy.helm.releases[].namespace` (see [Deploying with helm]({{< relref "/docs/deployers#deploying-with-helm)" >}}))
* `deploy.helm.releases[].repo` (see [Deploying with helm]({{< relref "/docs/deployers#deploying-with-helm)" >}}))
* `deploy.helm.releases[].setValueTemplates` (see [Deploying with helm]({{< relref "/docs/deployers#deploying-with-helm)" >}}))
* `deploy.helm.releases[].version` (see [Deploying with helm]({{< relref "/docs/deployers#deploying-with-helm)" >}}))
* `deploy.kubectl.defaultNamespace`
* `deploy.kustomize.defaultNamespace`
* `manifests.kustomize.paths.[]`
* `portForward.namespace`
* `portForward.resourceName`

_Please note, this list is not exhaustive._

List of variables that are available for templating:

* all environment variables passed to the Skaffold process at startup
* For the `envTemplate` tagger:
  * `IMAGE_NAME` - the artifact's image name - the [image name rewriting]({{< relref "/docs/environment/image-registries.md" >}}) acts after the template is calculated
* For Helm deployments:
  * `IMAGE_NAME`, `IMAGE_TAG`, `IMAGE_DIGEST, IMAGE_DOMAIN, IMAGE_REPO_NO_DOMAIN` - the first (by order of declaration in `build.artifacts`) artifact's image name, repo, tag, sha256 digest, registry/domain and repository w/o the registry/domain prefixed . Note: the [image name rewriting]({{< relref "/docs/environment/image-registries.md" >}}) acts after the template is calculated.
  * `IMAGE_NAME_<artifact-name>`, `IMAGE_REPO_<artifact-name>`, `IMAGE_TAG_<artifact-name>`, `IMAGE_DIGEST_<artifact-name>` - the named artifact's image name, repo, tag, and sha256 digest. NOTE: When used in for templating all `/` and `-` chars must be changed to `_` characters as go templates do not accept `/` and `-`.
  * `IMAGE_NAME2`, `IMAGE_REPO2`, `IMAGE_TAG2`, `IMAGE_DIGEST2` - the 2nd artifact's image name, tag, and sha256 digest
  * `IMAGE_NAME2`, `IMAGE_REPON`, `IMAGE_TAGN`, `IMAGE_DIGESTN` - the Nth artifact's image name, tag, and sha256 digest

### Template functions
Besides the [built-in template functions](https://pkg.go.dev/text/template) in golang, skaffold also provides the following ones:
- `default` : Default values can be specified using the `{{default "bar" .FOO}}` expression syntax, which results in "bar" if .FOO is nil or a zero value.
- `cmd`: This allows users to use the result from external commands in template, for example `{{cmd "bash" "-c" "xxx xxx xxx"}}` can be used to execute bash script and get the result into the template.
