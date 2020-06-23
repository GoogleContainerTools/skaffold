### Example: deploy helm charts with local dependencies

This example follows the [helm](../helm-deployment) example, but with a local chart as a dependency.

The `skipBuildDependencies` option is used to skip the `helm dep build` command. This must be disabled for charts with local dependencies.

The image can be passed to the subchart using the standard Helm format of `subchart-name.value`.

```yaml
deploy:
  helm:
    releases:
    - name: skaffold-helm
      chartPath: skaffold-helm
      namespace: skaffold
      skipBuildDependencies: true # Skip helm dep build
      artifactOverrides :
        image: skaffold-helm
        "subchart.image": skaffold-helm # Set image for subchart
      valuesFiles:
        - helm-values-file.yaml
```
