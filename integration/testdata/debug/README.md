# Integration Tests for `skaffold debug`

These are a set of test projects for `skaffold debug`.  There are two
configurations:

  - `skaffold.yaml` configures docker- and jib-based builders
  - `skaffold-bp.yaml` configures buildpacks-based builders

The test projects endeavour to support both docker or jib, and buildpacks.
