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
* `build.tagPolicy.envTemplate.template` (see [envTemplate tagger]({{< relref "/docs/pipeline-stages/taggers#envtemplate-using-values-of-environment-variables-as-tags)" >}}))
* `deploy.helm.releases.setValueTemplates` (see [Deploying with helm]({{< relref "/docs/pipeline-stages/deployers#deploying-with-helm)" >}}))
* `deploy.helm.releases.name` (see [Deploying with helm]({{< relref "/docs/pipeline-stages/deployers#deploying-with-helm)" >}}))

_Please note, this list is not exhaustive._

List of variables that are available for templating:

* all environment variables passed to the Skaffold process at startup
* `IMAGE_NAME` - the artifacts' image name - the [image name rewriting]({{< relref "/docs/environment/image-registries.md" >}}) acts after the template is calculated
