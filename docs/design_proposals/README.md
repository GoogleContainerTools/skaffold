# Design Proposals process

Hello Contributors!

This document describes the process for proposing a new feature or making any
big code changes to `skaffold`.

Having a proposal, will likely reduce the back and forth between the contributor
and the core team. It also makes sure, each new feature or a big change has a
design review.

For any new feature, config or big changes, please add a design proposal document
as described in [Design Proposal Template](./design-proposal-template.md).

Once you create a PR with the proposal, someone from the core team will be
assigned as a design shepherd. The role of the design shepherd will be to make
sure,

1. The feature/change is within Skaffold Philosophy and not a one off
   solution for a specific use case.
2. The feature/change scope is well defined.
3. When changing any existing feature, the implementation plan adheres to
   [skaffold deprecation policy](./../../deprecation-policy.md)

Once the proposal is in a reasonale shape, we can discuss it in Skaffold bi-weekly
meeting to address any open concerns, and reach to a decision i.e. accept or
punt.
