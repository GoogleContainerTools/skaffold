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

<a href="vscode://googlecloudtools.cloudcode/shell?repo=https://github.com/GoogleContainerTools/skaffold.git&subpath=/examples/jib-gradle"><img width="286" height="50" src="/docs/static/images/open-cloud-code.png"></a>
