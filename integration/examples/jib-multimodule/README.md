### Example: Jib Multi-Module

[Jib](https://github.com/GoogleContainerTools/jib) is one of the supported builders in Skaffold.
It builds Docker and OCI images
for your Java applications and is available as plugins for Maven and Gradle.

Sometimes a project is configured to have multiple _modules_ to create several
container images.  Skaffold can work with Jib to build these containers as
required.

The way you configure it in `skaffold.yaml` is the following build stanza:

```yaml
build:
     artifacts:
     - image: skaffold-jib-1
       # context is the root of the multi-module project
       context: .
       jib:
          # project is the location relative to the artifact `context`
          # For Maven, this is either the path relative to the artifact's
          # `context`, or `:artifactId` or `groupId:artifactId`.
          # For Gradle, this is the project name (defaults to the
          # directory name).
          project: moduleLocation
     - image: skaffold-jib-2
       context: .
       jib:
          project: :artifactId
```

There are a few caveats:

- The `jib-maven-plugin` must be either be configured referenced in the
root module of the project.  This is easily done through a `pluginManagement`
block.

- The artifact modules must have a `jib:xxx` goal bound to the `package` phase.
