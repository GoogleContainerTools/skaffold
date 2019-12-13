### Example: kaniko

This is an example demonstrating:

* **building** a single Go file app and with a single stage `Dockerfile` using [kaniko](https://github.com/GoogleContainerTools/kaniko) to build on a K8S cluster
* **tagging** using the default tagPolicy (`gitCommit`)
* **deploying** a single container pod using `kubectl`

<a href="vscode://googlecloudtools.cloudcode/shell?repo=https://github.com/GoogleContainerTools/skaffold.git&subpath=/examples/kaniko"><img width="286" height="50" src="/docs/static/images/open-cloud-code.png"></a>
