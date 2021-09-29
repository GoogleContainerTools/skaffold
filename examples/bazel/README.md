### Example: bazel

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/GoogleContainerTools/skaffold&cloudshell_open_in_editor=README.md&cloudshell_workspace=examples/bazel)

Bazel is one of the supported builders in Skaffold.

The way you configure it in `skaffold.yaml` is the following build stanza:

```yaml
build:
  artifacts:
  - image: skaffold-example
    context: .
    bazel:
      target: //:skaffold_example.tar
```

1. make sure the `context` contains the bazel files (`WORKSPACE`, `BUILD`)
2. add `bazel` section to each artifact
3. specify `target` - our builder will use this to load to the image to the Docker daemon
