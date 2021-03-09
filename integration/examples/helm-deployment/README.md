### Example: deploy multiple releases with Helm

You can deploy multiple releases with skaffold, each will need a chartPath, a values file, and namespace.
Skaffold can inject intermediate build tags in the the values map in the skaffold.yaml.

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
      artifactOverrides:
        image: skaffold-helm
      valuesFiles:
      - values.yaml
```

This part tells Skaffold to set the `image` parameter of the values file to the built `skaffold-helm` image and tag.

```yaml
      artifactOverrides:
        image: skaffold-helm
```
