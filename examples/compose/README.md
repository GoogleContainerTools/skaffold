### Example: running Skaffold with docker-compose files

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/GoogleContainerTools/skaffold&cloudshell_open_in_editor=README.md&cloudshell_workspace=examples/compose)

This example provides a simple application set up to run with 
[Docker Compose](https://docs.docker.com/compose/).

Notice there is no `skaffold.yaml` configuration present.
To run this example, use:

```bash
skaffold init --compose-file docker-compose.yaml
```

1. This will invoke the [kompose](https://github.com/kubernetes/kompose) binary to generate
kubernetes manifests based off of the Docker Compose configuration.
2. This will generate the `skaffold.yaml` configuration.
