### Example: Running custom tests on built images

This example shows how to run _custom tests_ on newly built images in the skaffold dev loop. 

Custom tests are associated with single image artifacts. When test dependencies change, no build will happen but tests would get re-run. Tests are configured in the `skaffold.yaml` in the `test` stanza, e.g.

```yaml
test:
    - image: skaffold-example
      custom:
      - command: <command>
        timeoutSeconds: <timeout in seconds>
        dependencies: <dependencies for this command>
          paths: <file dependencies>
          - <paths glob>
```

As tests take time, you might prefer to configure tests using [profiles](https://skaffold.dev/docs/https://skaffold.dev/docs/environment/profiles/) so that they can be automatically enabled or disabled, e.g.
If the `command` exits with a non-zero return code then the test will have failed, and deployment will not continue.

```yaml
profiles:
  - name: test
    test:
    - image: skaffold-example
      custom:
        - command: <command>
          timeoutSeconds: <timeout in seconds>
          dependencies: <dependencies for this command>
            paths: <file dependencies>
            - <paths glob>
```