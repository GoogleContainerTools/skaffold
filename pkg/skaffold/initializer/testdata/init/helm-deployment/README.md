### Example: deploy multiple releases with Helm

You can deploy multiple releases with Skaffold, each will need a chartPath, a values file, and an optional namespace.
Skaffold can inject intermediate build tags in the the values map in the `skaffold.yaml`.

Let's walk through the skaffold yaml:

We'll be building an image called `skaffold-helm`, and it's a dockerfile, so we'll add it to the artifacts.

```yaml
build:
  artifacts:
  - image: skaffold-helm
```

Now, we want to deploy this image with helm.
We add a new release in the helm part of the deploy stanza.

```yaml
deploy:
  helm:
    releases:
    - name: skaffold-helm
      chartPath: charts
      # namespace: skaffold
      valuesFiles:
      - values.yaml
```
