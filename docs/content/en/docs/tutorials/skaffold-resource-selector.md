---
title: "Manage CRDs w/ Skaffold - Configuring Which K8s Resources & Fields Skaffold Manages"
linkTitle: "Manage CRDs w/ Skaffold - Configuring Which K8s Resources & Fields Skaffold Manages"
weight: 90
featureId: skaffold-resource-selector
aliases: [/docs/how-tos/skaffold-resource-selector]
---

Common Use Cases This Page Helps Resolve:
* Users who want skaffold to properly manage the rendering and deployment of their custom CRDs (as skaffold does with K8s objects like Pod, Deployment.apps, etc.)
  * Additionally users w/ a CRD that uses a different field name for `image:` (eg: `foo:`) and want skaffold to properly modify the value to instead have the image label for the image skaffold recently built
* Users who are seeing issues with skaffold's default resource field overwriting for a given resource - eg: skaffold errors as it tries to mutate immutable config on re-deployment

Currently skaffold modifies the manifests it renders and deploys for the following functionality:
- status checking - done by mutating the manifest/K8s-Object by adding a label - skaffold/dev/run-id.  Skaffold uses this run-id to identify ... (TODO add doc link to run-id explanation)
- image label overwriting - done by mutating the manifest/K8s-Object by substituting the `image:$ORIGINAL_IMAGE_TAG` value(s) in a manifest with `image:$RECENT_SKAFFOLD_BUILT_IMAGE`


Skaffold has by default the following resources set for management via field "labels:" and "image:" overwriting:

_The below list is derived from the values defined [here](https://github.com/GoogleContainerTools/skaffold/blob/main/pkg/skaffold/kubernetes/manifest/visitor.go)_


* Pod
* DaemonSet.apps
* Deployment.apps
* ReplicaSet.apps
* StatefulSet.apps (with the exception of `.spec.volumeClaimTemplates.*.metadata.labels` field(s))
* CronJob.batch
* Job.batch
* DaemonSet.extensions
* Deployment.extensions
* ReplicaSet.extension
* Service.serving.knative.dev
* Fleet.agones.dev
* GameServer.agones.dev
* Rollout.argoproj.io
* Workflow.argoproj.io
* CronWorkflow.argoproj.io
* WorkflowTemplate.argoproj.io
* ClusterWorkflowTemplate.argoproj.io
* *.cnrm.cloud.google.com

_This default overwriting modifies all JSON Paths for those GroupKinds of the form:_
_* *.metadata.labels (skaffold appends a `run-id` label to existing labels or adds a `labels `field with a `run-id` entry if it didn't exist prior)_
_* *.image (changes `image:` value to be the skaffold built image ONLY IF skaffold manages the original `image:` value)_


The GroupKind's that Skaffold manages (via resource field overwriting) are user configurable via the `resourceSelector:` top level configuration.  The `resourceSelector` configuration allows users to modify and extend which resources and what fields of those resources skaffold modifies.  Currently skaffold only supports `label:` and `.metadata.labels` related modifications.
(TODO add `resourceSelector` schema overview and allowable inputs)
`resourceSelector` spec (from `pkg/skaffold/schema/latest/v1/config.go`)
```
// ResourceSelector describes user defined filters describing how skaffold should treat objects/fields during rendering.
ResourceSelector ResourceSelectorConfig `yaml:"resourceSelector,omitempty"`

// ResourceSelectorConfig contains all the configuration needed by the deploy steps.
type ResourceSelectorConfig struct {
	// Allow configures an allowlist for transforming manifests.
	Allow []ResourceFilter `yaml:"allow,omitempty"`
	// Deny configures an allowlist for transforming manifests.
	Deny []ResourceFilter `yaml:"deny,omitempty"`
}

// ResourceFilter contains definition to filter which resource to transform.
type ResourceFilter struct {
	// GroupKind is the compact format of a resource type.
	GroupKind string `yaml:"groupKind" yamltags:"required"`
	// Image is an optional slice of JSON-path-like paths of where to rewrite images.
	Image []string `yaml:"image,omitempty"`
	// Labels is an optional slide of JSON-path-like paths of where to add a labels block if missing.
	Labels []string `yaml:"labels,omitempty"`
}
```

The values for `Image` and `Labels` support a JSON Path style string which designates a path to a field in the speified GroupKind.  Additionally there is a special `.*` value that can be used which means that skaffold will attempt to overwrite all relevant labels following the below rules:
- image: [".*"] -> replace all fields which follow `*.image:` where the value is an image that skaffold manages/builds
- labels: [".*"] -> append-to or create a field named `*.metadata.labels` if a field `*.metadata` is found

Some example use cases and motivations for the `resourceSelector` are shown below:
* Skaffold Management of Custom CRD - The below snippet using `resourceSelector` allows a user to configure skaffold to manage a custom CRD (eg: CustomDeployment.skaffold.dev) they've created for their application in a skaffold.  

_Without this snippet, skaffold would apply the yaml but would not properly wait for child resources or replace the `image:` values with skaffold built images_
{{% readfile file="samples/resource-selector/resource-selector-crd-example.yaml" %}}

Using the above configuation, skaffold will properly update the `image:` ... and .. allowing it to ... 

* Fix Issue With Skaffold Overwriting Immutable Field - The below snippet using `resourceSelector` shows a user configuring skaffold to change it's behaviour to NOT overwrite a resource's field to prevent K8s errors related to overwriting an immutable field:
_The below configuration is actually a part of skaffold's default configuration, just made into a snippet to use as an example_
{{% readfile file="samples/resource-selector/resource-selector-deny-example.yaml" %}}

* Allow `image:` Overwriting For Differently Named Image Field(s) - The below snippet using `resourceSelector` shows a user configuring skaffold to change it's behaviour to overwrite a resource's `foo:`field with skaffold built images.  This allows skaffold to properly support the images skaffold builds this resources which uses `foo:` instead of `image:` for an image value:
{{% readfile file="samples/resource-selector/resource-selector-allow-example.yaml" %}}
