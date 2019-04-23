# Title

* Author(s): Cornelius Weig (@corneliusweig)
* Design Shepherd: Priya Wadhwa (@priyawadhwa)
* Date: 21/04/2019
* Status: [Reviewed/Cancelled/Under implementation/Complete]

## Background

Skaffold offers the possibility to watch pod logs for the `run`, `dev`, and `deploy` subcommands.
So far, the pods to watch are determined by looking for known artifact images in the containers.
This is restricting for two reasons:
1. It does not allow to add additional pods to log aggregator which are not deployed by Skaffold (#666).
2. It does not allow to exclude uninteresting pods deployed by Skaffold. For example, when working with istio, all the irrelevant istio logs spam the log (#588, #1991) 

## Design
I suggest to change the selection of pods for the log aggregator based on labels on this pod.

For that, we need to change the definition of the `SkaffoldRunner` as follows:
```go
type SkaffoldRunner struct {
	build.Builder
	deploy.Deployer
	test.Tester
	tag.Tagger
	sync.Syncer
	watch.Watcher

	cache             *cache.Cache
	opts              *config.SkaffoldOptions
	labellers         []deploy.Labeller
	builds            []build.Artifact
	hasBuilt          bool
	hasDeployed       bool
	
	// <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
	// OLD
	// imageList         *kubernetes.ImageList
	// NEW
	podSelector       kubernetes.PodSelector
	// <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
	
	RPCServerShutdown func() error
}
```
This change does not affect the public API.

To support the new use-cases of altering the pod selector, I propose a new top-level config section in the pipeline config:
```yaml
log:
  # a list of deployment.spec.selector items, rules are OR'ed
  - matchExpressions: # in one item, rules are AND'ed
      - key: XY
        values: ['val1', 'val2']
        operator: In NotIn Exists DoesNotExist
    # or
    matchLabels: # in one item, rules are AND'ed
      tail: "true"
```
This structure is based on the [standard selector spec](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors) for k8s resources.
Using a standard k8s configuration structure increases the usability for our users.
For one selector spec item, the semantics should be exactly the same as for deployments, i.e. several conditions are AND'ed. Quoting the kubernetes docs (`kubectl explain deployment.spec.selector`):
```txt
KIND:     Deployment
VERSION:  extensions/v1beta1

RESOURCE: selector <Object>

DESCRIPTION:
     Label selector for pods. Existing ReplicaSets whose pods are selected by
     this will be the ones affected by this deployment.

     A label selector is a label query over a set of resources. The result of
     matchLabels and matchExpressions are ANDed. An empty label selector matches
     all objects. A null label selector matches no objects.

FIELDS:
   matchExpressions     <[]Object>
     matchExpressions is a list of label selector requirements. The requirements
     are ANDed.

   matchLabels  <map[string]string>
     matchLabels is a map of {key,value} pairs. A single {key,value} in the
     matchLabels map is equivalent to an element of matchExpressions, whose key
     field is "key", the operator is "In", and the values array contains only
     "value". The requirements are ANDed
```

In addition, Skaffold needs to support the use-case to add further pods to a selection.
To address that, the log spec has to be a list of selector specs which is OR'ed.
For example, to add the traefik controller to the watch list:
```yaml
log:
  - matchExpressions:
      - key: app
        values: ['traefik']
        operator: In
  - matchLabels:
      tail: "true"
```

This config change should be made backwards compatible by defaulting the log selector to
```yaml
log:
  - matchLabels:
      tail: "true"
```
Skaffold already labels all deployed pods with `tail=true` unless run with the `--tail=false` CLI option.

### Open Issues/Question

Please list any open questions here in the format.

**\<Is the `tail=true` label good enough?\>** In principle, it would be possible that other pods also have a `tail=true` label by accident (for example a previous `skaffold run --tail` followed by `skaffold dev`).
Do we need to exclude such cases, e.g. by adding a unique `skaffold-run: <uuid>` label?

Resolution: A UUID label is not required for now but can be added later when users request it.

## Implementation plan
1. Switch the selection of pods for the log selector from image lists to `tail=true` label (#1910).
2. Add log section to the Skaffold config and select pods based on this
3. Integration test for log selection.
4. Add documentation for the new log section.

## Integration test plan

- Add a test case with two deployments, where one is selected based on the log spec and the other is excluded.
  Make sure that the excluded pod does not show up in the logs.
