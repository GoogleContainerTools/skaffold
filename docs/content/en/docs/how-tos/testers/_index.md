---
title: "Testers"
linkTitle: "Testers"
weight: 15
---

This page discusses how to set up Skaffold to run container structure tests after building an artifact.

Container structure tests are consistency checks for containers.
Skaffold relies on [container-structure-test](https://github.com/GoogleContainerTools/container-structure-test) to execute those tests, and requires its [binary](https://github.com/GoogleContainerTools/container-structure-test/releases) to be installed.

Container structure tests are defined per image in the Skaffold config.
Every time an artifact is rebuilt, Skaffold runs the associated structure tests on that image.
If the tests fail, Skaffold will not continue on to the deploy stage.
If frequent tests are prohibitive, long-running tests should be moved to a dedicated Skaffold profile.

### Example
This following example shows the `test` section from a `skaffold.yaml`.
It instructs Skaffold to run all container structure tests in the `structure-test` folder relative to the Skaffold root directory:

{{% readfile file="samples/testers/test.yaml" %}}

The files matched by the `structureTests` key are `container-structure-test` test configurations, such as:

{{% readfile file="samples/testers/structureTest.yaml" %}}

For a reference how to write container structure tests, see its [documentation](https://github.com/GoogleContainerTools/container-structure-test#command-tests).

In order to restrict the executed structure tests, a `profile` section can override the file pattern:

{{% readfile file="samples/testers/testProfile.yaml" %}}

To execute the tests once, run `skaffold build --profile quickcheck`.
