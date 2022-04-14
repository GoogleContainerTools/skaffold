### Example: deploy helm charts with local dependencies

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/GoogleContainerTools/skaffold&cloudshell_open_in_editor=README.md&cloudshell_workspace=examples/helm-deployment-dependencies)

This example follows the [helm](../helm-deployment) example, but with a local chart as a dependency.

The `skipBuildDependencies` option is used to skip the `helm dep build` command. This must be disabled for charts with local dependencies.

```yaml
deploy:
  helm:
    releases:
    - name: skaffold-helm
      chartPath: skaffold-helm
      namespace: skaffold
      skipBuildDependencies: true # Skip helm dep build
      valuesFiles:
        - helm-values-file.yaml
```
