# Title

* Author(s): \<your name\>
* Design Shepherd: \<skaffold-core-team-member\>

    If you are already working with someone mention their name.
    If not, please leave this empty, it will be assigned to a core team member.
* Date: \<date\>
* Status: [Reviewed/Cancelled/Under implementation/Complete]

Here is a brief explanation of the Statuses

1. Reviewed: The proposal PR has been accepted, merged and ready for
   implementation.
2. Under implementation: An accepted proposal is being implemented by actual work.
   Note: The design might change in this phase based on issues during
   implementation.
3. Cancelled: During or before implementation the proposal was cancelled.
   It could be due to:
   * other features added which made the current design proposal obsolete.
   * No longer a priority.
4. Complete: This feature/change is implemented.

## Background

In this section, please mention and describe the new feature, redesign
or refactor.

Please provide a brief explanation for the following questions:

1. Why is this required?
2. If this is a redesign, what are the drawbacks of the current implementation?
3. Is there any another workaround, and if so, what are its drawbacks?
4. Mention related issues, if there are any.

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
already contained in the Dockerfile for docker build (see #1166, #1581).
In addition, the syncing needs to handle special cases for globbing and often
requires a long list of sync patterns (#1807).
___

## Design

Please describe your solution. Please list any:

* new config changes
* interface changes
* design assumptions

For a new config change, please mention:

* Is it backwards compatible? If not, what is the deprecation policy?
  Refer to the [Skaffold Deprecation Policy](http://skaffold.dev/docs/references/deprecation/).
  for details.

### Open Issues/Questions

Please list any open questions here in the following format:

**\<Question\>**

Resolution: Please list the resolution if resolved during the design process or
specify __Not Yet Resolved__

## Implementation plan
As a team, we've noticed that larger PRs can go unreviewed for long periods of
time. Small incremental changes get reviewed faster and are also easier for
reviewers.

For a new feature, list the major tasks required for the implementation. Given the example artifact sync proposal, some of the smaller tasks could be:
___

1. Add new config key `infer` to `artifact.sync` and test schema validation.
2. Add inference logic for docker and examples.
3. Support both `infer` and user defined map with precedence rules implemented.
4. Finally, support builder plugins to add sync patterns.

___


## Integration test plan

Please describe what new test cases you are going to consider.
