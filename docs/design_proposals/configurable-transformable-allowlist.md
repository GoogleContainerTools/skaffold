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
    - Group: example.com
      Kind: Application
    - Group: argoproj.io
      Kind: Workflow
    - Group: tekton.dev
      Kind: Task
```

## Open issues/Questions

Since it is an allowlist, neither options could disable transformation on any
built-in resource like `ReplicaSet` or `Deployment`.

Is there any need to work out a deny list?

## Implementation plan

1. `pkg/skaffold/schema/latest/v1/config.go` - Add config option
`transformableAllowList` to `DeployConfig`.
2. `pkg/skaffold/kubernetes/manifest/visitor.go` - Add new parameter `transformableAllowList` 
to `*ManifestList.Visit()` by appending it to existing coded `transformableAllowList`
3. `pkg/skaffold/kubernetes/manifest/images.go` - Add new parameter to `*ManifestList.ReplaceImages()`
to support given `transformableAllowList`
4. Instrument each deployer to use the new parameter `transformableAllowList`

## Integration test plan

Please describe what new test cases you are going to consider.

1.  Unit and integration tests for `visitor.go`.

    The integration tests should be written to catch situations such as this
    configurable allowlist is either empty or empty array.

3.  Document this new configuration option
