# Transparent Init

* Author(s): Marlon Gamez
* Design Shepherd: \<skaffold-core-team-member\>
* Date: 10/13/2020
* Status: [Reviewed/Cancelled/Under implementation/Complete]

## Background

This proposal is brought about by an older issue, [#1273](https://github.com/GoogleContainerTools/skaffold/issues/1273)

With the recent focus on skaffold UX, we've decided to see how we can improve the onboarding experience when using skaffold.
One way we'd like to do this is by allowing the user to simply install skaffold and run any of the skaffold commands in their project, without having to first run `skaffold init` and go through the interactive portion of creating a config for their project.
To do this, we'd like to automatically create a config upon invocation of skaffold commands, so that the user doesn't have to see something like this
```
❯ skaffold dev
skaffold config file skaffold.yaml not found - check your current working directory, or try running `skaffold init`
```
when trying to run skaffold.

## Scope

This proposal is meant to design the flow of a user running a command without a skaffold config file already on disk. It does not cover the actual functionality of `skaffold init --force`, and doesn't address the further changes that will be made to that functionality.

Additionally, this functionality is planned to work with any command that requires parsing a skaffold config.

## Design

With transparent init, a user's first run of `skaffold dev` could look something like this:

Given a simple project structure:
```
project-dir
├── Dockerfile
├── README.md
├── k8s-pod.yaml
└── main.go
```
Running skaffold dev:
```
❯ skaffold dev
Skaffold config file not found. Generating config file 'skaffold.yaml'.
Remove this file and run 'skaffold init' if you'd like to interactively create a config.

Listing files to watch...
 - skaffold-example
Generating tags...
 - skaffold-example -> skaffold-example:v1.14.0-59-g7e5ae4cbc-dirty
Checking cache...
 - skaffold-example: Not found. Building
Found [minikube] context, using local docker daemon.
Building [skaffold-example]...
Sending build context to Docker daemon  3.072kB
...
```

There are two approaches to implementation that I've considered:

**Creating a temporary `skaffold.yaml` by running `skaffold init --force`**

Under the hood view:
- `skaffold dev` is run
- skaffold searches for a `skaffold.yaml` file in the directory, finds nothing
- skaffold creates a new config using `skaffold init --force`
- the new config is parsed and brought into memory
- the config file is deleted from disk
- `skaffold dev` continues like normal

**Creating the config in memory using init functions**

Under the hood view:
- `skaffold dev` is run
- skaffold searches for a `skaffold.yaml` file in the directory, finds nothing
- skaffold creates a new config and keeps in memory
- `skaffold dev` continues like normal

## Open Issues/Questions

**Should generation of the config all happen in memory? By refactoring `DoInit()` we could probably generate the config and start using it without having to write/read the `skaffold.yaml` from disk**

I think this would be preferred. We could eliminate an unecessary read from disk by doing so.

**Should we prompt the user to ask if they'd like to save this config for future use?**

I believe that it would be fine to write to disk automatically. The logging will make it clear that this is being done and after the skaffold session is finished they can decide to remove the file or not. 

**Should the config being used be printed to the terminal? Tradeoff of information vs clutter**

I would say that we don't need to. If the user needs to view the config used, they can view the skaffold config that is written to disk.

## Implementation plan

If we decide upon the approach in which the `skaffold.yaml` is written to/read from disk, it would be pretty straightforward to implement this.

If we decide to have the config exist only in memory, we may want to split this into a couple PRs. Maybe something like this:
1. Refactor `DoInit()` so that it is broken into parts that we can use later
2. User the new parts to implement the automatic generation of the config


## Integration test plan

New integration tests can be added to check that invocations of skaffold commands in folders without a skaffold config still run properly. With the other plans to change skaffold init, these will have to be updated as well.
