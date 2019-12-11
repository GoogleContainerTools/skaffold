### Example: Getting started with a simple go app

This is a simple example based on:

* **building** a single Go file app and with a multistage `Dockerfile` using [kaniko](https://github.com/GoogleContainerTools/kaniko) in Google Cloud Build
* **tagging** using the default tagPolicy (`gitCommit`)
* **deploying** a single container pod using `kubectl`
