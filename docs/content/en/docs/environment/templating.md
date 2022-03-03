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

* `build.artifacts.[].docker.buildArgs` (see [builders]({{< relref "/docs/pipeline-stages/builders" >}}))
* `build.artifacts.[].ko.{env,flags,labels,ldflags}` (see [`ko` builder]({{< relref "/docs/pipeline-stages/builders/ko" >}}))
* `build.tagPolicy.envTemplate.template` (see [envTemplate tagger]({{< relref "/docs/pipeline-stages/taggers#envtemplate-using-values-of-environment-variables-as-tags)" >}}))
* `deploy.helm.releases.setValueTemplates` (see [Deploying with helm]({{< relref "/docs/pipeline-stages/deployers#deploying-with-helm)" >}}))
* `deploy.helm.releases.name` (see [Deploying with helm]({{< relref "/docs/pipeline-stages/deployers#deploying-with-helm)" >}}))
* `deploy.helm.releases.namespace` (see [Deploying with helm]({{< relref "/docs/pipeline-stages/deployers#deploying-with-helm)" >}}))
* `deploy.helm.releases.version` (see [Deploying with helm]({{< relref "/docs/pipeline-stages/deployers#deploying-with-helm)" >}}))
* `deploy.kubectl.defaultNamespace`
* `deploy.kustomize.defaultNamespace`
* `deploy.kustomize.paths.[]`
* `portForward.namespace`
* `portForward.resourceName`

_Please note, this list is not exhaustive._

List of variables that are available for templating:

* all environment variables passed to the Skaffold process at startup
* For the `envTemplate` tagger:
  * `IMAGE_NAME` - the artifact's image name - the [image name rewriting]({{< relref "/docs/environment/image-registries.md" >}}) acts after the template is calculated
* For Helm deployments:
  * `IMAGE_NAME`, `IMAGE_TAG`, `IMAGE_DIGEST` - the first (by order of declaration in `build.artifacts`) artifact's image name, tag, and sha256 digest. Note: the [image name rewriting]({{< relref "/docs/environment/image-registries.md" >}}) acts after the template is calculated.
  * `IMAGE_NAME2`, `IMAGE_TAG2`, `IMAGE_DIGEST2` - the 2nd artifact's image name, tag, and sha256 digest
  * `IMAGE_NAMEN`, `IMAGE_TAGN`, `IMAGE_DIGESTN` - the Nth artifact's image name, tag, and sha256 digest
