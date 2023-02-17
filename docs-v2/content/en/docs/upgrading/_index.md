---
title: "Upgrading from Skaffold v1 to Skaffold v2 [NEW]"
linkTitle: "Upgrading from Skaffold v1 to Skaffold v2 [NEW]"
weight: 10
aliases: [/docs/upgrading-to-v2]
---

In Skaffold v2 what was previously the `deploy` phase of Skaffold is now split into a new `render` phase and `deploy` phase.  This clear boundary of separation between render and deploy phases allowed the team to simplify our code and CLI allowing us to clean up previously confusing or redundant flags like `skaffold deploy --render-only`, `skaffold deploy --skip-render`. 
This release comes with a new schema version `v3alpha1`. This schema introduced a new `manifests` section which declares all resources an application deploys e.g helm charts, kubernetes yaml, kustomize directories and kpt configuration. This decoupling of manifests declaration from the deploy section allows manifests to be used across deploy tools e.g.
* you can configure kpt deployer to render and apply kubernetes yaml, helm charts or
* you can configure the kubectl deployer to apply helm charts and helm to render the charts. 

Upgrading from skaffold `v1.*.*` to skaffold `v2.0.0-beta3` should not require any manual skaffold.yaml changes or CLI command modification for most common use cases.  Skaffold `v2` includes the same CLI surface as `v1` and has backwards compatibility for all previous skaffold.yaml schema `apiVersion` for example - `v2beta*`, `v1beta*` and `v1alpha*`.  

If you wish to update your skaffold.yaml to the latest `apiVersion` (`apiVersion: v3alpha1`) run `skaffold fix` which will output an updated skaffold.yaml with the schema fields updated for `v3alpha1`.  With this new `v3alpha1` configuration schema you can access the new v2 functionality via the `v3alpha1` configuration fields [here]({{< relref "/docs/references/yaml#manifests" >}})  Example usage of `skaffold fix`:
```console
$ cat skaffold.yaml | head -1
apiVersion: skaffold/v2beta29
$ skaffold fix
apiVersion: skaffold/v3alpha1
kind: Config
build:
  artifacts:
  - image: skaffold-example
manifests:
  rawYaml:
  - k8s-*
deploy:
  kubectl: {}
```

The list of features that were supported in skaffold `v1` but are no longer support or require manual changes for `v2.0.0-beta3` include:
* `v1` `kpt` deployer usage is not upgradeable via `skaffold fix` given the numerous changes made to the `kpt` workflow.  Manual changes might be required to get users pipelines working as expected.
* using multiple renderers WITH the `kpt` deployer being one of them (using combinations of any other renderer(s) works as it did previously).

Outside of the above, there are currently no known other regressions when migrating from skaffold v1 -> v2 but areas that are most likely to have possible issues/incompitibility include:
- `helm` renderer/deployer usage (see [helm docs]({{< relref "/docs/pipeline-stages/renderers/helm" >}}) for more details)
- v1 `kpt` deployer usage 
- `skaffold render` flags usage (see [render docs]({{< relref "/docs/pipeline-stages/renderers" >}}) and [render schema]({{< relref "/docs/references/yaml#manifests" >}}) for more details)


If you encounter any issues using skaffold `v2.0.0-beta3`, particularly any regressions that used to work differently or succeed in `v1`, please file an issue at [GoogleContainerTools/skaffold](https://github.com/GoogleContainerTools/skaffold/issues).
