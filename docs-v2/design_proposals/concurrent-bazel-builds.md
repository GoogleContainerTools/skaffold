# Concurrent Bazel Builds

* Author(s): Seth Nelson
* Design Shepherd: 
* Date: 2024/05/21
* Status: [Reviewed/Cancelled/Under implementation/Complete]

## Background

Bazel supports high levels of concurrency in its action graph model; however, Skaffold is not currently able to 
take advantage of that concurrency to build multiple targets because it is modeled around a separate builder
invocation per artifact. This does not play nicely with Bazel's workspace lock model; invoking multiple builds
in parallel as separate processes forces them to serialize. This effectively renders Skaffold's `--build-concurrency`
null and void (see https://github.com/GoogleContainerTools/skaffold/issues/6047), and results in longer overall
build times than if multiple target's action execution could interleave.

This document proposes changes to Skaffold which will enable multiple targets to be build in a single Bazel invocation.


## Design

### Where to break the "one artifact per build" abstraction

As part of this design, I considered three places to introduce the concept of multi-artifact builds:

1. For all builds - in the `builder_mux`, group builds before scheduling.
2. For the local builder pipeline - allow any underlying artifactBuilder to implement a `BuildMultiple` method,
   and wrap that in a `BatchingBuilder` which can queue up multiple incoming builds before sending groups into `BuildMultiple`.
3. For Bazel only, where batching happens as an implementation detail with `bazel/build.go`.

Ultimately, this design proposes #3. 

* While #1 results (in my opinion) in the cleanest abstraction model (after all, builder_mux is the demux point for 
  turning a list of artifacts into a set of builds to be scheduled), it was rejected due to its invasiveness and 
  complexity. Implementing this at the `builder_mux` level results in a
  ton of method forking (into multi-artifact variants); it also introduces significant scheduling complexity (to avoid 
  cycles introduced by batching build targets). This complexity did not feel warranted for a feature that is likely
  to only be used by Bazel.
* #2 was rejected because it would incorrectly (continue to) couple the compilation vs load/push phases of Bazel's `artifactBuilder` implementation.
  We only want to group up the actual `bazel build` invocation, not the subsequent `docker.Push`/`b.loadImage` which
  is also part of the `Build` implementation. Batching at the `Build` implementation level would (a) reduces the speed gains, because we would serialize the loads/pushes,
  and (b) adds implementation complexity, because we need to handle BatchingBuilders with different `artifactBuilder`
  constructor parameters (like loadImages).


### Design Overview

This design proposes implementing build batching for Bazel builds by altering the Bazel Builder to build multiple
JARs, and then multiplexing contemporaneous builds (e.g. waiting for a period of time for builds targets to 
"collect" and then building them all together). 

* This changes no existing externally-visible methods/interfaces.
* This does not _require_ config changes, though we could consider (a) initially implementing this functionality hidden behind a configuration flag, for rollout safety, and (b) configuring the batch window.
* This design naturally respects the `--build_concurrency` flag, which limits how many builds will be scheduled contemporaneously.

### Design Elements

#### Bazel Builder can build multiple TARs

This is an in-place change; we replace `buildTar` with `buildTars`, which builds up a list of `buildTargets`.

#### computeBatchKey function

A `computeBatchKey` function which returns the same value if two different builds can be grouped together. Elements of
the Bazel batch key include:
* The sorted list of build args
* The (currently, single) bazel platform flag
* The artifact's workspace

#### batchingBuilder

A BatchingBuilder wraps an underlying multi-artifact build function (the `buildTars` function), and maintains a map of
non-started `MultiArtifactBuild` objects (per batch key).

```go
type batchingBuilder struct {
	builder   multiJarBuilder
	batchSize time.Duration
	builds    map[string]*MultiArtifactBuild
	mu        sync.Mutex
}
```

```go
type MultiArtifactBuild struct {
	ctx       context.Context
	artifacts []*latest.Artifact
	platforms platform.Matcher
	workspace string
	mu        sync.Mutex
	doneCond  chan bool
	running   bool
	out       io.Writer
	digest    map[*latest.BazelArtifact]string
	err       error
}
```

When it is asked to build an artifact, it first checks to see if there is an existing build for the batch key.
* If there is, it simply adds the artifact to the build's `artifacts` list. 
* If there is not, it creates a new MultiArtifact build, and kicks off a goroutine that will wait `batchSize` before
  invoking its `multiJarBuilder`.

#### How this impacts logging and metrics:

Given our choice in where to break the "one artifact per build" abstraction, this design presents some rough edges
around logging and metrics that are worth acknowledging:
* If any build in a batch fails, they are all reflected as failed to the Events API / console output.
  * This is fundamentally unavoidable in any implementation that involves a single bazel invocation building multiple 
  targets.
* Only one build's `OutputWriter` is actually used (the first one to create the `MultiArtifactBuild` object). 
  * This
  presents some rough edges in my POC that I could use some help ironing out - depending on scheduling order, sometimes
  it fails to stream output to the console until the build is complete.

### Open Questions

**\<Should we include either of the configuration knobs listed in the Design Overview>**

Resolution: __Not Yet Resolved__


## Implementation plan

I believe this change is well-contained enough to be implemented in a single PR, please let me know what you think. 
You can find a functional proof-of-concept here: https://github.com/GoogleContainerTools/skaffold/compare/main...sethnelson-tecton:skaffold:sethn/parallel-builds

___


## Integration test plan

New test cases (in addition to status quo test cases of unbatched builds):

1. Batched builds that can all batch together (2 artifacts, no dependencies, build-concurrency>=2)
1. Batched builds that should not batch together due to concurrency limits (4 artifacts, build-concurrency>=2)
1. Batched builds that should not batch together due to scheduling (2 artifacts, one depends on the other, build-concurrency>=2)
1. Batched builds with different build args
