---
title: "Kpt [NEW]"
linkTitle: "Kpt [NEW]"
weight: 30
featureId: render
aliases: [/docs/pipeline-stages/renderers/kpt]
---

{{< alert title="Note" >}}
kpt CLI must be installed on your machine for the below functionality. Skaffold will not
install it.
{{< /alert >}}

## `manifests.transform` and `manifests.validate` functionality powered by kpt
With Skaffold V2, skaffold now has a new `render` phase and associated `manifests` top level config field.   Along with these changes,  a `manifests.transform` and a `manifests.validate` field were added which allows users to specify `kpt` manifest transformations and validations to be done in the `render` phase.  `manifests.transform` allows users to create a pipeline of manifest transformations which transform manifests via the specified container.  For more information on the `manifests.transform` functionality see, the docs for `kpt` `mutators` [here](https://kpt.dev/book/04-using-functions/01-declarative-function-execution).  For a list of `kpt` supported containers to use in the `manifests.transform` schema see the list [here](https://catalog.kpt.dev/) with the tag `mutator`.  `manifests.validate` allows users to create a pipeline of manifest validations that run serially, checking the yaml manifests for the specified validation test.  For more information on the `manifests.validate` functionality, see the docs for `kpt` `validators` [here](https://kpt.dev/book/04-using-functions/01-declarative-function-execution).  For a list of `kpt` supported containers to use in the `manifests.validate` schema see the list [here](https://catalog.kpt.dev/) with the tag `validator`.

Conceptually these top level fields remove the necessity of a separate Kptfile allowing more users to adopt the powerful rendering functionality `kpt` enables.  Functionally, these fields are identical to having a seperate `Kptfile` with the `manifests.transform` -> `pipeline.mutators` and `manifests.validate` -> `pipeline.validators`.

An example showing how these fields can be used is below.  Run `skaffold render` in a directory with the following files:

`skaffold.yaml`
{{% readfile file="samples/renderers/kpt-manifest-fields.yaml" %}}


`kpt-k8s-pod.yaml`
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  app: guestbook
spec:
  containers:
  - name: getting-started
    image: nginx
```

The aboveconfiguration above adds a field `metadata.annotations.author` with value `fake-author`, adds a `kpt` "setter" comment (` # kpt-set: ${app}`) to the intermediate yaml, modifies the value at the location of the `kpt` "setter" field with the provided `app` value (`app: guestbook-fake-author`) and then validates that the yaml is valid yaml via `kubeval`.


## Rendering with kpt using a Kptfile

[`kpt`](https://kpt.dev/) allows Kubernetes
developers to customize raw, template-free YAML files for multiple purposes.
Skaffold can work with `kpt` by calling its command-line interface.


### Configuration

To use kpt with Skaffold, add render type `kpt` to the `manifests`
section of `skaffold.yaml`.

The `kpt` configuration accepts a list of paths to folders containing a Kptfile.

### Example

The following `manifests` section instructs Skaffold to render
artifacts using kpt.  Each entry should point to a folder with a Kptfile.

{{% readfile file="samples/renderers/kpt.yaml" %}}

