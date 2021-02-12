### Example: Running custom tests on built images

This example shows how to run
[custom tests]
on newly built images in your skaffold dev loop. Tests are associated with single
artifacts. Tests are configured in
your `skaffold.yaml` in the `test` stanza, e.g.

```yaml
test:
    - image: skaffold-example
    Custom:
        - command: <command>
```

Tests can also be configured through profiles, e.g.

```yaml
profiles:
  - name: test
    test:
      - image: skaffold-example
        Custom:
        - command: <command>
```
