# integrationtest command

* Author(s): Dominic Werner (@daddz)
* Design Shepherd: Tejal Desai (@tejal29) 
* Date: 2019-08-16
* Status: Cancelled
* Reason: 
  
  We discussed this proposal in Skaffold Community Hours and decided at this
  moment, the design proposal is just an entry point for running a script.
  It is not built up on Skaffold's run knowledge so far.
  With that said, the core team agrees Skaffold does not a solution
  for [#2561](https://github.com/GoogleContainerTools/skaffold/issues/2561)
  and [#992](https://github.com/GoogleContainerTools/skaffold/issues/992) and we want to
  explore this space.
          
## Background

Currently, skaffold has no support for running integration/unit tests 
that are part of a built artifact. Since this is a crucial part of a CI/CD
pipeline it would be good to support this feature so skaffold can be used
in every step of a pipeline.

Proof-of-concept:
- [Integration test command (\#2594)](https://github.com/GoogleContainerTools/skaffold/pull/2594)

Related issues:
- [Integrationtest phase (\#2561)](https://github.com/GoogleContainerTools/skaffold/issues/2561)
- [Add \`test\` phase to skaffold runner (\#992)](https://github.com/GoogleContainerTools/skaffold/issues/992)
___

## Design

#### New configuration options in skaffold.yaml

```yaml
integrationtest:
    podSelector: 'app=skaffold-integration'
    testCommand: 'pytest tests/pass.py'
```

A top-level key called `integrationtest` holds the configuration for running
integration tests with skaffold.

`podSelector`: Define in which pod the tests shall be executed.
The pod will be looked up via the defined label across all namespaces.

`testCommand`: Define the test command that will be executed in the pod.

##### Backwards compatibility

The configuration is not required and thus should not impact older versions.

#### New skaffold CLI command

A new command called `skaffold integrationtest` is implemented that takes
the values from `skaffold.yaml` to execute the tests and report the output
and outcome.

This command can then be used locally or within the CI pipeline.

#### Usage in CI pipeline

The usage within a CI pipeline could look like this:

* build: `skaffold build --file-output build.json`
* test: 
  * `skaffold deploy -n $(pipeline_id) -a build.json`
  * `skaffold integrationtest`
  * `skaffold delete -n $(pipeline_id)`
* deploy: `skaffold deploy -a build.json`

### Open Issues/Questions

**Does this approach make sense for different languages/frameworks?**

Resolution: __Not Yet Resolved__

**Should the command look for the pod in all namespaces or shall it require it to be defined since
usually one would deploy to a fresh namespace and execute the tests there explicitly?**

Resolution: __Not Yet Resolved__

## Implementation plan

1. Add new top-level config key `integrationtest` and test schema validation
2. Add new config keys `podSelector` and `testCommand` to `integrationtest` and test schema validation
3. Add new command `integrationtest`


## Integration test plan

1. Test handling of different config variations (non-existent, empty, wrong/typo'd podSelector/testCommand)
2. Test proper return values (log output, return code) of `integrationtest` command
