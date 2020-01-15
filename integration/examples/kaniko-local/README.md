### Example: kaniko

This is an example demonstrating:

* **building** a single Go file app and with a single stage `Dockerfile` using [kaniko](https://github.com/GoogleContainerTools/kaniko) to build on a K8S cluster directly from a local build context
* **tagging** using the default tagPolicy (`gitCommit`)
* **deploying** a single container pod using `kubectl`
