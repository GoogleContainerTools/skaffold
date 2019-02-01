
---
title: "Templated fields"
linkTitle: "Templated fields"
weight: 90
---

Skaffold config allows for certain fields to have values injected that are either environment variables or calculated by Skaffold.
For example: 

```yaml
build:
    artifacts:
    - image: gcr.io/k8s-skaffold/example
    tagPolicy:
        envTemplate:
            template: "{{.IMAGE_NAME}}:{{.FOO}}"
    local: {}
```

Suppose the value of the `FOO` environment variable is `v1`, the image built
will be `gcr.io/k8s-skaffold/example:v1`.

List of fields that support templating: 

* `build.tagPolicy.envTemplate.template` (see [envTemplate tagger](/docs/how-tos/taggers/##envtemplate-using-values-of-environment-variables-as-tags))
* `deploy.helm.releases.setValueTemplates` (see [Deploying with helm](/docs/how-tos/deployers/#deploying-with-helm))

List of variables that are available for templating: 

* all environment variables passed to the Skaffold process as startup 
* `IMAGE_NAME` - the artifacts' image name - the [image name rewriting](/docs/concepts/#image-repository-handling) acts after the template was calculated  
* `DIGEST` - the image digest calculated by the docker registry after pushing the image 
* if `DIGEST` is of format `algo:hex`, `DIGEST_ALGO` and `DIGEST_HEX` parts correspond to the parts of the string otherwise `DIGEST_HEX`=`DIGEST` is set
 

