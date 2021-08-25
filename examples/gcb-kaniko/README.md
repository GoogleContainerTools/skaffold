### Example: Getting started with a simple go app

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/GoogleContainerTools/skaffold&cloudshell_open_in_editor=README.md&cloudshell_workspace=examples/gcb-kaniko)

This is a simple example based on:

* **building** a single Go file app and with a multistage `Dockerfile` using [kaniko](https://github.com/GoogleContainerTools/kaniko) in Google Cloud Build
* **tagging** using the default tagPolicy (`gitCommit`)
* **deploying** a single container pod using `kubectl`
