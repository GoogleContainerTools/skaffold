# Pull Request Guidelines

Writing good pull requests increase development velocity and minimize frustration.
 
If pull requests are small they will result in being opened and merged quickly.

Having big pull requests that bounce back and forth between developers
and reviewers can slow progress down significantly, causing developers to waste a 
lot of time dealing with merge conflicts.

Big pull requests also increase the risk of breaking things; since changes are so big, 
 - it’s hard to scrutinize all the changes and 
 - test them for regressions.

In order to promote good PR quality, the skaffold team will push back
against big PRs. See [0](https://github.com/GoogleContainerTools/skaffold/pull/2750#pullrequestreview-283621241), [1](https://github.com/GoogleContainerTools/skaffold/pull/2917#pullrequestreview-291415741)

## Breaking down Pull Requests

### Feature development.
Adding a new command or skaffold config results in large number of changes due to
all the boilerplate code required for adding a new command, generating docs etc.

It makes sense to break down these changes into 2 categories:
1. Incremental code changes which are not accessed by control flow.
2. User affecting changes like
   1. Adding a new command which exercises the above code changes
   1. Adding config change and generating docs.

#### Config Change Example   
Below is an example of how you can break a config change into small individual PRs.

A contributor wanted to support [custom build args to `kustomize` deployer](https://github.com/GoogleContainerTools/skaffold/issues/2488).
This required, skaffold to support a `buildArgs` field to `deploy.kustomize` config. Plumb the config to `kustomize` command.

This small 10 line change results into 100+ lines of code along with test code.
It would greatly simplify code review process if we break this change into 
1. Coding logic and test coverage.
2. User facing documentation.

In this case, the contributor split this PR in 2 small changes
1. Introduce a [place holder for a new config](https://github.com/GoogleContainerTools/skaffold/pull/2870).
   
   This PR highlights the logic changes and makes it easier for reviewers to review code logic and tests.
   
   Note: The code added in this PR does not get exercised other than in tests.
2. [Add a field to skaffold config](https://github.com/GoogleContainerTools/skaffold/pull/2871) and pass it to the place holder.
   This PR makes it easier for reviewers to review the user facing config changes and make sure all the precautions like
   - good user documentation
   - deprecation policy if applicable
   - upgrade policy etc were followed. 

#### Adding new functionality
Below is an example on how you can introduce a new functionality to skaffold in smaller incremental PRs.

To add ability to [render templated k8 manifests without deploying](https://github.com/GoogleContainerTools/skaffold/issues/1187), required us to make a bunch of changes in skaffold
1. Add this feature to `kubectl`, `helm` and `kustomize` deployers supported by skaffold.
2. support a new flag for `skaffold deploy` and `skaffold run` command
3. may be add a new command to only render k8 manifests.

Item 2 and 3 in the above list of changes, are user facing features and could be abstracted out.
Item 1 can further be implemented incrementally by supporting this new feature for each of the deployer.

To implement this feature, we 
1. First added an [method to interface `deployer` with empty stub](https://github.com/GoogleContainerTools/skaffold/pull/2834).
   
   Note: The code added in this PR does not get exercised other than in tests.
2. Then add render implementation for each [kubectl deployer](https://github.com/GoogleContainerTools/skaffold/pull/2943), kustomize and helm.
3. Add a [render command](https://github.com/GoogleContainerTools/skaffold/pull/2942)

Remember, its ok to land code which is not exercised. However please mention the follow up work
in **Next PRs** section when creating a Pull Request [e.g. here.](https://github.com/GoogleContainerTools/skaffold/pull/2811)


Finally, please use our best judgement when submitting pull requests.
