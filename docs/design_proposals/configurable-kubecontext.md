# Configurable kubecontext

* Author(s): Cornelius Weig (@corneliusweig)
* Design Shepherd: tbd
* Date: 29 June 2019
* Status: [Reviewed/Cancelled/Under implementation/Complete]

## Background

So far, Skaffold always uses the currently active kubecontext when interacting with a kubernetes cluster.
This is problematic when users want to deploy multiple projects with different kubecontexts, because the user needs to manually switch the context before starting Skaffold.
In particular when working on multiple such projects in parallel, the current behavior is limiting.

Open issues concerning this problem are

- Allow option to specify the kubectl context (#511)
- Support kube.config and kube.context for specifying alternative Kubernetes config file or context (#2325)
- Feature: Support regex in profile activation via kubeContext (#1677)
- Skaffold.yaml files are not portable (#480)

There also was an attempt to add a configuration option to `skaffold.yaml` (Support for overriding kubectl context during deployment #1540).

The goal of this document is to create an agreement on what options should be supported and identify edge cases.

## Design

There are four places where kubecontext activation can be added:
<table>
    <thead>
        <th>Precedence</th> <th>Kind</th> <th>Details</th>
    </thead>
    <tbody>
        <tr>
            <td>1. (highest)</td>
            <td>CLI option</td>
            <td>
              The kubernetes standard to set the kubecontext is <code>--context</code>.
              However, in Skaffold this term is so overloaded that it should more precisely be named <code>--kube-context</code>.
              This flag is necessary for IDE integration.
            </td>
        </tr>
        <tr>
            <td>2.</td>
            <td>Env variable</td>
            <td>
              <code>SKAFFOLD_KUBE_CONTEXT</code>, similar to other Skaffold flags.
            </td>
        </tr>
        <tr>
            <td>3.</td>
            <td><code>skaffold.yaml</code></td>
            <td>
              Json-path <code>deploy.kubeContext</code>.
              This option is shareable, but also creates a lot of complexity due to profile activation (see detailed discussion below).
            </td>
        </tr>
        <tr>
            <td>4. (lowest)</td>
            <td>Global Skaffold config</td>
            <td>
              This should give users the possibility to define a default context globally or per project.
              This variant is not shareable.
            </td>
        </tr>
    </tbody>
</table>

---

Beside the kubecontext, also the namespace needs to be specified.
It should also be possible to specify in various places:

1. `--namespace` CLI flag
2. `SKAFFOLD_NAMESPACE` env var
3. In `skaffold.yaml` at json-path `deploy.namespace` 
4. globally or per project in the Skaffold config

### Detailed discussion
#### Option in `skaffold.yaml`
A configuration option in `skaffold.yaml` has the advantage of being most discoverable:
it is in the place where users configure all aspects of their pipeline.

However, it also has questionable implications:

- `skaffold.yaml` is meant to be shared, but kubecontext names vary across users.
  Sharing therefore makes only sense in a corporate setting where context names are the same across many users.
  There is however a risk of abuse in settings where sharing the context name does not make sense, for example in open source projects.
- Due to profile activation by Skaffold profiles, there can be circular problems.
  For example, the current context is `A` and activates some profile.
  This profile activates a different kubecontext `B`.
  A solution could be to forbid specifying a kubecontext in a profile if it is activated by a kubecontext (and validate that).
 
The best place for the config in `skaffold.yaml` is in `latest.DeployConfig`, resulting in a json path `deploy.kubeContext`.

#### Option in global Skaffold config
Specifying a default kubecontext globally is straightforward. For example, via new config option
```yaml
global:
  default-context: docker-for-desktop
```

On the other hand, building a relation between projects and kubecontext needs to solve two questions:

1. How to identify projects
2. How to save the project/kubecontext relation

##### How to identify projects

There are at least two possibilities:

- Identify projects by their absolute host path.
  This is guaranteed to be unique, but may break if a user moves his project to another location.
- Identify projects by a new `metadata.name` entry in `skaffold.yaml` (see also #2200).
  This has the drawback of being potentially not unique, so that users accidentally pin the kubecontext for more projects than intended.
  On the other hand, this is the standard approach taken by kubernetes resources.
- Identify project by their initial commit.
  This variant is stable against relocations.
  It is also unique unless a user forks a project and wants to define different kubecontexts for each fork.
  Compared to the other alternatives, it is rather slow.
  
**\<What option has the best tradeoffs?\>**

Resolution: __Not Yet Resolved__


##### How to save project/kubecontext relations

Currently, the Skaffold config uses the kubecontext as identifier.

There are two possibilities to add the relation:

- Reverse the mapping project/kubecontext and save as list under `kube-context` entries:
  ```yaml
  kubecontexts:
  - kube-context: my-context
    projects:
    - /home/user/projects/skaffold-project
    - project-name
    - d76524a31c58657b06404f4442e1e64949f4d762
  ```
  The drawback here is that the data structure does not forbid conflicting entries, such as this:
  ```yaml
  kubecontexts:
  - kube-context: context1
    projects:
    - my-project
  - kube-context: context2
    projects:
    - my-project
  ```
- Add a new top-level entry in Skaffold config:
  ```yaml
  global: {}
  kubecontexts: []
  projects:
    my-project: # project name as the key
      kube-context: context1
  ```
  This option will be more complex to implement wrt `skaffold config`.

**\<What Skaffold config structure has the best tradeoffs?\>**

Resolution: __Not Yet Resolved__

### Open Issues/Questions

**\<Should there be a config option in `skaffold.yaml`?\>**

Resolution: __Not Yet Resolved__

## Implementation plan
1. Implement the CLI flag and env var variant first. This should also be the most important for the IDE integration.
2. Implement the global Skaffold config variant.
3. Implement `skaffold.yaml` variant if applicable.
4. Implement `skaffold config set` adaptions.
5. Implement the namespace functionality.

## Integration test plan

A single test covers the overall kubecontext override functionality sufficiently.
Other test-cases such as precedence of the different variants and error cases should be covered by unit tests.
