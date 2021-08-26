### Example: bazel

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
