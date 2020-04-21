# Title

* Author: Nick Kubala
* Design Shepherd: Any volunteers? :)
* Date: 04/11/19
* Status: [Reviewed/Cancelled/Under implementation/Complete]

## Final decision

After multiple days invested in this direction we still feel that it is not worthwhile to go down this direction: the complexity of adding go channels adds a lot of complexity and the design more error prone for subtle bugs. The current arguments for going towards this direction is the Event API redesign to centralize event management in the Runner. This in itself is a valuable idea but at this point the current design is simpler to reason about this new design with the go channels, plus if we want to extend the events sent by builders, e.g. progress bar events, we will have to send data through the go channels anyway that represents those events and convert them to events, or directly call the `event` package - at which point the complexity of "handling events from the builders" is going to be the same. 
We are open to introduce a streaming/reactive design in Skaffold on the long run if we have a strong enough argument/featureset for it. For this redesign I don't see that it is worthwhile. I am closing this and the related PRs. 

Thank you @nkubala  and @tejal29  for all the hard work in exploring this avenue in detail in code with testing and thoughtful conversations! 
## Background

Currently there are two major pain points with the event API:

* calling code is clunky, ugly, and not very portable
* event handler is racy and generally not concurrency-friendly

The second issue here comes more from bad execution rather than bad design. Some fixes for this have already been proposed and/or merged ([#1786](https://github.com/GoogleContainerTools/skaffold/pull/1786) and [#1801](https://github.com/GoogleContainerTools/skaffold/pull/1801)), so progress has already been made, but we can probably improve on this more.

The first issue is what I'll be focusing on in this issue. @dgageot has already merged one improvement to the design [here](https://github.com/GoogleContainerTools/skaffold/pull/1829), but this is only improving on an already flawed design. My goal for this redesign will be to reduce and potentially eliminate the overhead the event API has on maintainers and contributors to continue evolving the skaffold codebase.

___

## Design

Right now, events are handled directly through the main loop for each builder and deployer. The internal state is directly updated through the the individual builder/deployer, and anyone adding a new one is responsible for ensuring that these events are passed to the event handler correctly. Examples:

```golang
 if err != nil { 
 	event.BuildFailed(artifact.ImageName, err) 
 	return nil, fmt.Errorf("building [%s]: %w", artifact.ImageName, err)
 } 
  
 event.BuildComplete(artifact.ImageName) 
```
and
```golang
 event.DeployInProgress() 
  
 manifests, err := k.readManifests(ctx) 
 if err != nil { 
 	event.DeployFailed(err) 
 	return fmt.Errorf("reading manifests: %w", err) 
 } 
  
 if len(manifests) == 0 { 
 	return nil 
 }
```

Even though the event handling functions have been reduced to one line, the fact that they still have to be manually called from the build or deploy code is a lot of overhead for contributors, especially people who are considering adding a new builder and deployer.

Every build or deploy that skaffold performs always comes through the runner, so it would follow that the runner could handle all of the eventing for builds and deploys, as long as it can retrieve the necessary information from the builder or deployer.
* For deployers, this is easy: we just need to know whether or not the deploy completed successfully, and if not, what the error encountered was. This is already passed back to the runner.
* For builders, it's a little less straightforward. For each artifact, we'll need to know whether or not the build was successful, and if not, the error encountered. Currently, the Build() interface returns a single error back to the runner, containing information about the first unsuccessful build that it saw during the build process. Based on this, if we were building two artifacts, and one completed while the other failed, the runner has know way to know this, and assumes both failed to build.

To address this, **I propose that we change the Build() interface to return a list of build results, one for each artifact, with information about the original artifact skaffold attempted to build, and either the built artifact, or the error the builder encountered.**

This will allow the runner to:
* know which artifacts were built successfully, and which failed
* log events for each artifact, complete with a detailed error message passed back from the runner in the event that the build didn't complete successfully

With this information, we could move all of the event handling code into a decorator that we wrap the builders and deployers in during the creation of the runner, similar to the way we handle timings currently. The semantics for each state (Not Started, In Progress, Complete, Failed) and the events that transition the states should all be the same between builders and deployers, and we can share the logic for handling each of these events in a separate location. This has the added bonus of not requiring any event-related code to be written by contributors if/when they add new builders/deployers, or make changes to existing ones.


## Implementation plan

Implementing this should be pretty straightforward:

1) **Rework the Build() interface**

The current method signature is 

```golang
Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]Artifact, error)
```

This will be changed to return a list of build results for each artifact, along with the artifact information itself. The artifact information can be embedded in the build result:

```golang
type BuildResult struct {
  Target *latest.Artifact
  Result *build.Artifact
  Error error
}
```
and the new method signature will be
```golang
Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]BuildResult, error)
```

2) **Consume the new build results in the runner**

The runner can then take control of all the eventing code. Deploy results are already returned by the Deploy() interface implementations, so with the Build results we have all the information we need. Using Build() as an example (Deploy() will look similar):

* For each artifact provided to Build(), an `BuildInProgress` event is triggered
* Once the build is complete, for each returned result, trigger a `BuildFailed` event for results with errors, and a `BuildComplete` event for those without.

```golang
for _, a := range artifactsToBuild {
  event.BuildInProgress(a)
}
bRes, err := r.Build(ctx, out, tags, artifactsToBuild)
for _, res := range bRes {
  if res.Error != nil {
    event.BuildFailed(res.Target, res.Error)
  } else {
    event.BuildComplete(res.Target)
  }
}
```

The main drawback here is that for builds run in parallel, the statuses will not truly reflect reality until all builds are completed, but I don't believe users will see much of an effect from this.
___


## Integration test plan

The current integration tests around eventing and retrieving skaffold state should be sufficient to cover this new change. Skaffold runs are triggered, the state is retrieved at various times during the run, and the tests verify the correct results are retrieved.

Additional unit tests can be added as necessary to ensure the correct event handling methods are being called from the runner.
