# Supporting dependencies between build artifacts

## Background
Refer [this](https://tinyurl.com/skaffold-modules) document which presents the design goals for introducing the concept of _modules_ in Skaffold. A prerequisite to being able to define modules and supporting cross module dependency is to first consider the latest version of the skaffold config (_currently `v2beta8`_) as an implicit module and allowing dependencies between artifacts defined in it. The current document aims to capture the major code changes necessary for achieving this.

>*Note: Omitted some details for brevity, assuming reader's familiarity with the skaffold [codebase](https://github.com/GoogleContainerTools/skaffold).*

## Config schema

We introduce an `ArtifactDependency` slice within `Artifact` in `config.go`

```go
type ArtifactDependency struct {
	ImageName string `yaml:"image" yamltags:"required"`
	Alias string `yaml:"alias,omitempty"`
}
```

This allows us to define a build stanza like below where image `leeroy-app` requires image `simple-go-app`:

```yaml
build:
 artifacts:
   - image: simple-go-app
   - image: leeroy-app
     requires:
       - image: simple-go-app
         alias: BASE
```

Alias is a token that will be replaced with the image reference in the builder definition files. If no value is provided for `alias` then it defaults to the value of `image`.

## Config validation

We add two new validations to the [validation](https://github.com/GoogleContainerTools/skaffold/blob/10275c66a142719897894308b9e566953712a0fe/pkg/skaffold/schema/validation/validation.go#L37) package after the introduction of artifact dependencies:
- Cyclic references among artifacts.
  - We cannot have image `A` depend on image `B` depend on image `C` depend on image `A`.
  - We run a simple depth first search cycle detection algorithm treating our `Artifact` slice like a directed graph- image `A` depending on image `B` implies a directed edge from `A` to `B`.
- Unique artifact aliases.
  - We ensure that within *each* artifact dependency slice the aliases are unique. 

## Referencing dependencies

### Docker builder

The `docker` builder will use the `alias` of an `ArtifactDependency` as a build argument key.
 
 ```yaml
 build:
  artifacts:
    - image: simple-go-app
    - image: leeroy-app
      requires:
        - image: simple-go-app
          alias: BASE
 ```

Here `BASE=gcr.io/X/simple-go-app:<tag>@sha:<sha>` is passed as a `--build-arg` (or a buildKit parameter) when building `leeroy-app`.

### Custom builder

The `custom` builder will be supplied each `ArtifactDependency`'s `alias` and image reference as environment variables keyed on `alias`. So they can be easily referenced in user-defined build definitions.

### Buildpacks builder

Buildpacks supports overriding the run-image and the builder-image in its current schema. We extend this to allow specifying `ArtifactDependency` aliases as the value for the `runImage` and `builder` fields.

```yaml
build:
  artifacts:
  - image: skaffold-buildpacks
    buildpacks:
      builder: "{{ .BUILDER_IMAGE }}"
      runImage: "{{ .RUN_IMAGE }}"
    requires:
      - image: builder-image
        alias: BUILDER_IMAGE
      - image: run-image
        alias: RUN_IMAGE
```

If there are any additional images in the `required` section it only enforces that they get built prior to the current image. However, the buildpacks builder cannot really reference them in any other way.

### Jib builder

The Jib builder supports [changing the base image](https://cloud.google.com/java/getting-started/jib#base-image). We allow substituting the `required` image aliases with the image references in the specific builder `args` section.
This will allow overriding the `jib.from.image` property that sets the base image.

For Maven:

```yaml
build:
  artifacts:
  - image: test-jib-maven
    jib:
      type: maven
      args: 
      - -DbaseImage=docker://{{ .BASE_IMAGE }}
```

where `baseImage` is added as a property to the `pom.xml`

```xml
<properties>
  <baseImage>image</baseImage>
</properties>
...
<configuration>
  <from>
    <image>${baseImage}</image>
  </from>
</configuration>
```

Similarly, for Gradle:

```yaml
build:
  artifacts:
  - image: test-jib-gradle
    jib:
      type: gradle
      args: 
      - -Djib.from.image=registry://{{ .BASE_IMAGE }}
```

where `jib.from.image` is added to the Gradle configuration.

### Bazel builder

The bazel builder doesn't support referencing images directly. Also, it natively supports setting up nested builds. We allow defining `required` artifacts even though they can't be referenced by the builder. We do this to give the user a way of ordering these builds; with the future work around [Skaffold Hooks](https://github.com/GoogleContainerTools/skaffold/issues/1441) there might be some usecases where pre and post build scripts might want a certain ordering of builds. 

## Builder interfaces
There are two builder abstractions -- in [build.go](https://github.com/GoogleContainerTools/skaffold/blob/10275c66a142719897894308b9e566953712a0fe/pkg/skaffold/build/build.go#L37) and [parallel.go](https://github.com/GoogleContainerTools/skaffold/blob/10275c66a142719897894308b9e566953712a0fe/pkg/skaffold/build/parallel.go#L34)

```go
type Builder interface {
	Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]Artifact, error)
}
```
```go
type ArtifactBuilder func(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error)
```
The first describes a builder for a list of artifacts. There are a few implementations like `cache`, `local` and `cluster` builders.
The second describes a per artifact builder. Again there are a few implementations like `docker`, `buildpacks`, etc.

We modify both of them to:

```go
type Builder interface {
	Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact, existing []Artifact) ([]Artifact, error)
}
```
```go
type ArtifactBuilder func(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string, artifactResolver ArtifactResolver) (string, error)
```
where we define `ArtifactResolver` interface, as:
```go
// ArtifactResolver provides an interface to resolve built artifacts by image name.
type ArtifactResolver interface {
	GetImageTag(imageName string) string
}
```

This necessitates all multi-artifact builder implementations to also accept a slice of already built artifacts' information. All single artifact builders require an `ArtifactResolver` that can provide the required artifacts.

## Build controller

[InSequence](https://github.com/GoogleContainerTools/skaffold/blob/10275c66a142719897894308b9e566953712a0fe/pkg/skaffold/build/sequence.go) and [InParallel](https://github.com/GoogleContainerTools/skaffold/blob/10275c66a142719897894308b9e566953712a0fe/pkg/skaffold/build/parallel.go) are two build controllers for deciding how to schedule the run of multiple builds together. `InSequence` runs all builds sequentially whereas `InParallel` runs them parallely with a max concurrency defined by a `concurrency` field.

After introducing inter-artifact dependencies we'll need to run the builds in a topologically sorted order.
We introduce a new controller `scheduler.go` and deprecate `sequence.go` and `parallel.go`. Here we model the `Artifact` slice graph using a set of `go channels` to achieve the topologically sorted build order.

```go
type status struct {
  imageName string
  success   chan interface{}
  failure   chan interface{}
}

type artifactChanModel struct {
	artifact                 *latest.Artifact
	artifactStatus           status
	requiredArtifactStatuses []status
}

func (a *artifactChanModel) markSuccess() {
	// closing channel notifies all listeners waiting for this build that it succeeded
	close(a.status.success)
}

func (a *artifactChanModel) markFailure() {
	// closing channel notifies all listeners waiting for this build that it failed
	close(a.status.failure)
}
func (a *artifactChanModel) waitForDependencies(ctx context.Context) error {
	for _, depStatus := range a.requiredArtifactChans {
		// wait for required builds to complete
		select {
		case <-ctx.Done():
		case <-depStatus.failure:
			return fmt.Errorf("failed to build required artifact: %q", depStatus.imageName)
		case <-dep:
		}
	}
    return nil
}
```

Each artifact has a success and a failure channel that it closes once it completes building by calling either `markSuccess` or `markFailure` respectively. This notifies *all* listeners waiting for this artifact of a successful or failed build.

Additionally it has a reference to the channels for each of its dependencies.
Calling `waitForDependencies` ensures that all required artifacts' channels have already been closed and as such have finished building before the current artifact build starts.

> *<ins>Alternative approach</ins>:  Another way to do this is to run any popular topologically sorting algorithm on the `Artifact` slice, treating it as a directed graph. However, we can get a simpler implementation at the expense of a few additional `goroutines` the way described above.*

This class also provides an implementation of the interface `buildStatusRecorder` that should be safe for concurrent access.

```go
type buildStatusRecorder interface {
  Record(imageName string, imageTag string, err error)
  GetImageTag(imageName string) string
}
```

Finally we have the only exported function in `scheduler.go` that orchestrates all the builds:

```go
func InOrder(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact, existing []Artifact,  buildArtifact ArtifactBuilder, concurrency int) ([]Artifact, error) {
```

This function maintains an instance of `buildStatusRecorder` implementation and can pass it as an `ArtifactResolver` to the various `ArtifactBuilder`s while recording the status after each build completion.

## Build concurrency

Skaffold currently allows specifying the `concurrency` property in `build` which affects how many builds can be running at the same time. However it doesn't address the issue of certain builders (`jib` and `bazel`) not being safe for multiple concurrent runs against the same workspace or context. We can fix this also since we are reworking the build controller anyways. 

We define a concept of lease on workspaces by preprocessing the list of artifacts. Each builder tries to acquire a lease on the context/workspace prior to starting the build. Only workspaces associated with concurrency-safe builders allot multiple leases, otherwise it assigns one lease at a time.

This integrates with the `InOrder` build controller above.

## Build logs reporting

The code below is the current way the `InParallel` build controller reports the build logs.

```go
func collectResults(out io.Writer, artifacts []*latest.Artifact, results *sync.Map, outputs []chan string) ([]Artifact, error) {
	var built []Artifact
	for i, artifact := range artifacts {
		// Wait for build to complete.
		printResult(out, outputs[i])
		v, ok := results.Load(artifact.ImageName)
		if !ok {
			return nil, fmt.Errorf("could not find build result for image %s", artifact.ImageName)
		}
		switch t := v.(type) {
		case error:
			return nil, fmt.Errorf("couldn't build %q: %w", artifact.ImageName, t)
		case Artifact:
			built = append(built, t)
		default:
			return nil, fmt.Errorf("unknown type %T for %s", t, artifact.ImageName)
		}
	}
	return built, nil
}
```

There are two quirks in this: 
- It reports in the order of artifacts in the `Artifact` slice instead of the actual order in which they get built. 
- It only reports a single artifact build failure even though there could have been multiple failures.

This will prove misleading after the introduction of artifact dependencies since we can have out of order artifact definitions in the skaffold config which with the current reporting strategy would appear to be building in the wrong order, and also build failures due to failed required artifact builds won't be immediately apparant.

So we introduce a new `BuildLogger` interface as a facade to achieve two things: 
- Print log messages in the order that it builds.
> *<ins>Future work</ins>: This however doesn't solve the problem for concurrently running builds as the current skaffold UX can't show parallel statuses. This will need to be addressed separately when we have a different UX with status bars that can show multiple statuses.
Until then, we can limit the max concurrency to 1 (this is what we currently do anyways)*
- Report about *all* build failures.

## Image cache

[hash.go](https://github.com/GoogleContainerTools/skaffold/blob/10275c66a142719897894308b9e566953712a0fe/pkg/skaffold/build/cache/hash.go) provides the `getHashForArtifact` function that needs to recursively be called for each of its dependencies and all those values aggregated together would be the hashcode for the artifact. This would ensure that for a cache hit all the cascading dependencies are unchanged. 

> Note: Dependencies provided as environment variables and build args are not resolved yet during hash calculation. That doesn't matter since they are already accounted for above.

Since `cache` package provides a `Builder` implementation it should additionally append all cache hits to the `existing` `Artifact` slice (see [Builder interfaces](#builder-interfaces) above).

## Dev-loop integration

### Build

[dev.go](https://github.com/GoogleContainerTools/skaffold/blob/10275c66a142719897894308b9e566953712a0fe/pkg/skaffold/runner/dev.go#L149) sets up the file monitor callback functions to queue the affected artifact to need rebuild or resync.

Now we'll have to queue the affected artifact along with all the monitored artifacts that are dependent on it and cascade. To do this we'll need the transpose graph of the `Artifact` slice directed graph that we currently have. One way to implement that would be as follows, which is also safe for concurrent access.

```go
type artifactDAG struct {
	m *sync.Map
}

func getArtifactDAG(artifacts []*latest.Artifact) *artifactDAG {
	dag := &artifactDAG{m: new(sync.Map)}
	for _, a := range artifacts {
		for _, d := range a.Dependencies {
			slice, ok := dag.m.Load(d.ImageName)
			if !ok {
				slice = make([]*latest.Artifact, 0)
			} else {
				slice = slice.([]*latest.Artifact)
			}
			dag.m.Store(d.ImageName, append(slice.([]*latest.Artifact), a))
		}
	}
	return dag
}

func (dag *artifactDAG) allDependents(artifact *latest.Artifact) []*latest.Artifact {
	slice, ok := dag.m.Load(artifact.ImageName)
	if !ok {
		return nil
	}
	return slice.([]*latest.Artifact)
}
```

This lets us query for _all artifacts that require a given artifact_.

```go
func addRebuild(dag *artifactDAG, artifact *latest.Artifact, rebuild func(*latest.Artifact), isTarget func(*latest.Artifact) bool) {
	if isTarget(artifact) {
		rebuild(artifact)
	}
	for _, a := range dag.allDependents(artifact) {
		addRebuild(dag, a, rebuild, isTarget)
	}
}
```

 Now we can request rebuild for all affected artifacts as a callback to the file monitoring event by setting it  in the `Dev` [function](https://github.com/GoogleContainerTools/skaffold/blob/10275c66a142719897894308b9e566953712a0fe/pkg/skaffold/runner/dev.go#L161)

```diff
-    r.changeSet.AddRebuild(artifact)
+    addRebuild(artifactDAG, artifact, r.changeSet.AddRebuild, r.runCtx.Opts.IsTargetImage)

```

### Sync

In this first implementation, we ignore all sync rules in base artifacts. This is because it isn't feasible to propagate sync rules between different builder types. 

We should notify the user that sync rules for a specific artifact are being ignored.
```
Warn: Ignoring sync rules for image "simple-go-app" as it is being used as a required artifact for other images.
```

> *<ins>Alternative approach</ins>: We could consider disallowing sync rules altogether in base artifacts. However, the next iteration of this would be supporting individual modules. In that case we would want to support sync rules when the base module runs separately but ignore the rules when run along with its dependents. 
> As such we prefer to implement ignoring sync rules behavior right now itself.*

> *<ins>Future work</ins>: We might be able to support propagating manual sync rules from base to derived artifacts. However, that's a lot of complexity to handle, and we can consider it if there is a user ask.*
