### Example: Running container-structure-test on built images

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/GoogleContainerTools/skaffold&cloudshell_open_in_editor=README.md&cloudshell_workspace=examples/structure-tests)

This example shows how to run
[structure tests](https://github.com/GoogleContainerTools/container-structure-test)
on newly built images in your skaffold dev loop. Tests are associated with single
artifacts, and one or more test files can be provided. Tests are configured in
your `skaffold.yaml` in the `test` stanza, e.g.

```yaml
test:
    - image: skaffold-example
    structureTests:
        - ./test/*
```

Tests can also be configured through profiles, e.g.

```yaml
profiles:
  - name: test
    test:
      - image: skaffold-example
        structureTests:
          - ./test/profile_structure_test.yaml
```
