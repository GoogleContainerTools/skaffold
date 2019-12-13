### Example: Jib (Gradle)

[Jib](https://github.com/GoogleContainerTools/jib) is one of the supported builders in Skaffold.
It builds Docker and OCI images
for your Java applications and is available as plugins for Maven and Gradle.

The way you configure it in `skaffold.yaml` is the following build stanza:

```yaml
build:
     artifacts:
     - image: gcr.io/k8s-skaffold/skaffold-example
       context: .
       jib: {}
```
