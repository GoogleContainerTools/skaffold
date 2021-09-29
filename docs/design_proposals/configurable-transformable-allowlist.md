# Configurable transformableAllowList

* Author(s): Ke Zhu (@shawnzhu)
* Design Shepherd: Tejal Desai (tejal29)
* Date: 2021-07-16
* Status: Draft

## Objectives

Configurable transformableAllowList for transforming manifests.

## Background

Skaffold can only transform manifests from a non-extensible allowlist. When
using any CRD out of this allowlist, skaffold can not transform it.

Open issues concerning this problem:

* Image not recognized in crd k8s manifest ([#4081](https://github.com/GoogleContainerTools/skaffold/issues/4081))

There was comments out of the above issue to make this allowlist extensible.

The goal of this document is to create an agreement on the configuration option
and specification to extend `skaffold.kubernetes.manifest.transformableAllowList`.

## Design

Add configuration `deploy.config.transformableAllowList` in `skaffold.yaml`:

Notice that any new configuration option will be appended to existing allowlist. 

### Detailed discussion

Option in `skaffold.yaml`

```YAML
deploy:
  config:
    transformableAllowList:
    - type: pod # no group, implicitly all versions
    - type: batch/Job # group, implicitly all versions
    - type: openfaas.com/v1/Function
      image: [spec.image]
      labels: [spec.metadata.labels, spec.labels]    # https://www.openfaas.com/blog/manage-functions-with-kubectl/
    - type: apps/v1beta1/Deployment
      image: [spec.template.spec.initContainers.*.image, spec.template.spec.containers.*.image]
      labels: [spec.metadata.labels, spec.template.metadata.labels]
```

The value of `type` field points to a resource type. So it's case sensitive
and should support API groups and resource versions:

* When not specifying group, it will transform given resource type of any group or versions.
* When providing group but not resource version, it will transform given
resource type of any versions.

The value of `labels` field is a list of JSON-path-like paths to apply `labels`
block to. If no `labels` field configured, it will simply apply `labels` block
if missing.

The value of `image` field is also a list of JSON-path-like paths to rewrite. If
no `image` field configured, it will rewrite any field named `image`.

## Open issues/Questions

Since it is an allowlist, neither options could disable transformation on any
built-in resource like `ReplicaSet` or `Deployment`. However, it may need to
refactor [current allowlist](https://github.com/GoogleContainerTools/skaffold/blob/27c38228ab929ddaf2636637b43f17fda1686652/pkg/skaffold/kubernetes/manifest/visitor.go#L28-L43).

Is there any need to work out a deny list?

## Implementation plan

1. `pkg/skaffold/schema/latest/v1/config.go` - Add config option
`transformableAllowList` to `DeployConfig`.
2. `pkg/skaffold/kubernetes/manifest/visitor.go` - Refactor allowlist and add
new parameter `transformableAllowList` to `*ManifestList.Visit()` by appending
it to existing coded `transformableAllowList`
    - Support `labels` field
    - Support `image` field
3. `pkg/skaffold/kubernetes/manifest/images.go` - Add new parameter to `*ManifestList.ReplaceImages()`
to support given `transformableAllowList`
4. Instrument each deployer to use the new parameter `transformableAllowList`

## Integration test plan

Please describe what new test cases you are going to consider.

1.  Unit and integration tests for `visitor.go`.

    The integration tests should be written to catch situations such as this
    configurable allowlist is either empty or empty array.

3.  Document this new configuration option
