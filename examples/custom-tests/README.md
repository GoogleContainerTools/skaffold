### Example: Running custom tests on built images

This example shows how to run
[custom tests]
on newly built images in the skaffold dev loop. 

Custom tests are associated with single image artifacts. When test dependencies change, no build will happen but tests would get re-run. Tests are configured in the `skaffold.yaml` in the `test` stanza, e.g.

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
        custom:
        - command: <command>
```
