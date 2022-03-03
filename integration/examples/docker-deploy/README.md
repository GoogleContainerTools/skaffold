### Example: Deploying a simple go app to Docker

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/GoogleContainerTools/skaffold&cloudshell_open_in_editor=README.md&cloudshell_workspace=examples/docker-deploy)

This is a simple example based on:

* **building** a two single Go file apps, each with a multistage `Dockerfile` using local docker to build
* **tagging** using the default tagPolicy (`gitCommit`)
* **deploying** to docker by simply running two individual containers
