### Example: Jib (Maven)

[Jib](https://github.com/GoogleContainerTools/jib) is one of the supported builders in Skaffold.
It builds Docker and OCI images
for your Java applications and is available as plugins for Maven and Gradle.

The way you configure it in `skaffold.yaml` is the following build stanza:

```yaml
build:
     artifacts:
     - image: skaffold-example
       context: .
       jib: {}
```

Please note that this example is for a standalone Maven project, where
all dependencies are resolved from outside. The Jib builder requires
that the projects are configured to use the Jib plugins for Maven or Gradle.
Multi-module builds require a bit additional configuration.
