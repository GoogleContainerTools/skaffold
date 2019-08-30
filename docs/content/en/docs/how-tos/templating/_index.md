---
title: "Templated fields"
linkTitle: "Templated fields"
weight: 90
---

Skaffold config allows for certain fields to have values injected that are either environment variables or calculated by Skaffold.
For example:

{{% readfile file="samples/templating/env.yaml" %}}

Suppose the value of the `FOO` environment variable is `v1`, the image built
will be `gcr.io/k8s-skaffold/example:v1`.

List of fields that support templating:

* `build.artifacts.[].docker.buildArgs` (see [builders](/docs/how-tos/builders/))
* `build.tagPolicy.envTemplate.template` (see [envTemplate tagger](/docs/how-tos/taggers/##envtemplate-using-values-of-environment-variables-as-tags))
* `deploy.helm.releases.setValueTemplates` (see [Deploying with helm](/docs/how-tos/deployers/#deploying-with-helm))
* `deploy.helm.releases.name` (see [Deploying with helm](/docs/how-tos/deployers/#deploying-with-helm))

_Please note, this list is not exhaustive_

List of variables that are available for templating:

* all environment variables passed to the Skaffold process at startup
* `IMAGE_NAME` - the artifacts' image name - the [image name rewriting](/docs/concepts/#image-repository-handling) acts after the template is calculated
