---
title: "Jib Build"
linkTitle: "Jib"
weight: 20
featureId: build
---

[Jib](https://github.com/GoogleContainerTools/jib#jib) is a set of plugins for
[Maven](https://github.com/GoogleContainerTools/jib/blob/master/jib-maven-plugin) and
[Gradle](https://github.com/GoogleContainerTools/jib/blob/master/jib-gradle-plugin)
for building optimized OCI-compliant container images for Java applications
without a Docker daemon.

Skaffold can help build artifacts using Jib; Jib builds the container images and then
pushes them to the local Docker daemon or to remote registries as instructed by Skaffold.

Skaffold requires using Jib v1.4.0 or later.

Skaffold supports building with Jib

1. [locally]({{< relref "/docs/pipeline-stages/builders/jib#jib-maven-and-gradle-locally" >}}) and
2. [remotely on Google Cloud Build]({{< relref "/docs/pipeline-stages/builders/jib#remotely-with-google-cloud-build" >}})

## Jib Maven and Gradle locally
**Configuration**

To use Jib, add a `jib` field to each artifact you specify in the
`artifacts` part of the `build` section. `context` should be a path to
your Maven or Gradle project.

{{< alert title="Note" >}}
Your project must be configured to use Jib already.
{{< /alert >}}

The `jib` type offers the following options:

{{< schema root="JibArtifact" >}}

Skaffold's jib support chooses the underlying builder (Maven or Gradle) 
based on the presence of standard build files in the `artifact`'s
`context` directory:

  - _Maven_: `pom.xml`, or `.mvn` directory.
  - _Gradle_: `build.gradle`, `gradle.properties`, `settings.gradle`,
    or the Gradle wrapper script (`gradlew`, `gradlew.bat`, or
    `gradlew.cmd`).

**Artifact Dependency**

You can define dependency on other artifacts using the `requires` keyword. This can be useful to specify another artifact image as the `fromImage`.

{{% readfile file="samples/builders/artifact-dependencies/jib-local.yaml" %}}

**Example**

See the [Skaffold-Jib demo project](https://github.com/GoogleContainerTools/skaffold/blob/main/examples/jib/)
for an example.

### Multi-Module Projects

Skaffold can be configured for _multi-module projects_ too. A multi-module project
has several _modules_ (Maven terminology) or _sub-projects_ (Gradle terminology) that
each produce a separate container image.

#### Maven

To build a Maven multi-module project, first identify the sub-projects (also called _modules_
in Maven) that should produce a container image. Then for each such sub-project:

  - Create a Skaffold `artifact` in the `skaffold.yaml`.
  - Set the `artifact`'s `context` field to the root project location.
  - Add a `jib` element and set its `project` field to the sub-project's
    `:artifactId`, `groupId:artifactId`, or the relative path to the sub-project
    _within the project_.

{{< alert title="Updating from earlier versions" >}}
Skaffold had required Maven multi-module projects bind a Jib
`build` or `dockerBuild` goal to the **package** phase.  These bindings are
no longer required with Jib 1.4.0 and should be removed.
{{< /alert >}}

#### Gradle

To build a multi-module project with Gradle, first identify the sub-projects that should produce
a container image.  Then for each such sub-project:

  - Create a Skaffold `artifact` in the `skaffold.yaml`.
  - Set the `artifact`'s `context` field to the root project location.
  - Add a `jib` element and set its `project` field to the sub-project's name (the directory, by default).


## Remotely with Google Cloud Build

Skaffold can build artifacts using Jib remotely on [Google Cloud Build]({{<relref "/docs/pipeline-stages/builders#remotely-on-google-cloud-build">}}).

**Configuration**

To configure, add `googleCloudBuild` to `build` section to `skaffold.yaml`.
The following options can optionally be configured:

{{< schema root="GoogleCloudBuild" >}}

**Example**

Following configuration instructs skaffold to build
 `gcr.io/k8s-skaffold/project1` with Google Cloud Build using Jib builder:

{{% readfile file="samples/builders/gcb-jib.yaml" %}}

## Advanced usage

### Additional arguments

The `jib` build artifact supports an [`args`](https://skaffold.dev/docs/references/yaml/#build-artifacts-jib-args) field to provide additional arguments to the Maven or Gradle command-line.  For example, a Gradle user may choose to use:
```
artifacts:
- image: jib-gradle-image
  jib:
    args: [--no-daemon]
```

### Using the `custom` builder

Some users may have more complicated builds that may be better suited to using the [`custom` builder](https://skaffold.dev/docs/pipeline-stages/builders/custom/).  For example, the `jib` builder normally invokes the `prepare-package` goal rather than `package` as Jib packages the `.class` files rather than package in the jar.  But some plugins require the `package` goal.
```
artifacts:
- image: jib-gradle-image
  custom:
    buildCommand: mvn -Dimage=$IMAGE package $(if [ $PUSH_IMAGE = true ]; then echo jib:build; else echo jib:dockerBuild; fi)
    dependencies:
      paths:
      - src/**
```

