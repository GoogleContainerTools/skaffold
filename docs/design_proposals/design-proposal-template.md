# Title

* Author(s): \<your name\>
* Design Shepherd: As mentioned in the document here.
* Date: \<date\>

## Background

In this section, please mention and describe the new feature, re-design
or re-factor.

Please provide an rationale covering following points:

1. Why is this required?
2. If its re-design, What are cons with current implementation?
3. Are there any another work-around and if yes, why not keep using it.

Here is an example snippet for a new feature:

___
Currently, skaffold config supports `artifact.sync` as a way to sync files
directly to pods. So far, artifact sync requires a specification of sync
patterns like

```yaml
sync:
  '*.js': app/
```

This is error prone and unnecessarily hard to use, because the destination is
already contained in the Dockerfile for docker build. (see #1166, #1581).
In addition, the syncing needs to handle special cases for globbing and often
requires a long list of sync patterns (#1807)
___

## Design

Please describe your solution. Please list any:

* new config changes
* interface changes
* design assumptions

For a new config change, please mention:

* If its a backward compatible config change
* If its a backward compatible config change, is there a migration path?

### Open Issues/Question

Please list any open questions here in the format.

**\<Question\>**

Resolution: Please list the resolution if resolved during the design process or
specify __Not Yet Resolved__

## Implementation plan
We have identified, huge PRs go unnoticed for a long time. Small incremental
changes get reviewed faster and also easier for reviewers.

For a design feature, list a summary of tasks breakdown for e.g.:
For the example desing proposal to infer artifact sync, some of the smaller task
could be:
___

1. Add new config key `infer` to `artifact.sync` and test schema validation.
2. Add inference logic for docker and examples.
3. Support both `infer` and user defined map with precedence rules implemented.
4. Finally, support builder plugins to add sync patterns.
___

For re-factor proposal identify smaller changes like:
___

1. Add new package skeleton. and any new objects with no functionality.
2. Move code from old place to newly added objects.

___

## Integration test plan

Please describe what new test cases are you going to consider.
