### Example: buildpacks (Go)

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/GoogleContainerTools/skaffold&cloudshell_open_in_editor=README.md&cloudshell_workspace=examples/buildpacks)

This is an example demonstrating:

* **building** a single Go file app built with [Cloud Native Buildpacks](https://buildpacks.io/)
* **tagging** using the default tagPolicy (`gitCommit`)
* **deploying** a single container pod using `kubectl`
