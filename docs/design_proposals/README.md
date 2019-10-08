# Design Proposal Process

Hello Contributors!

This document describes the process for proposing a new feature or making any
large code changes to `skaffold`. By large we mean,  large in impact or large in
size.

Examples for large impact changes would be:

1. Introduce templating, or
2. Arbitrary command execution in certain places

These could be small code changes but large in impact.

Submitting a proposal before a pull request will likely reduce the back and
forth between the contributor and the core team. A proposal also ensures that
each new feature or a large change has a design review.

For any new feature, config or large change, please add a design proposal document
as described in [Design Proposal Template](./design-proposal-template.md).

Once you create a PR with the proposal, one of the maintainers will be
assigned as a design shepherd. The role of the design shepherd will be to make
sure:

1. The feature/change is aligned with the Skaffold roadmap and the team's general
   philosophy for the tool and not a one off solution for a specific use case.
2. The feature/change scope is well defined.
3. When changing any existing feature, the implementation plan adheres to
   [Skaffold Deprecation Policy](http://skaffold.dev/docs/references/deprecation/).

Once the proposal has been approved, we can move discussions to our bi-weekly
meetings to address any open concerns,and to reach a final decision on whether
or not to accept the feature or change.
