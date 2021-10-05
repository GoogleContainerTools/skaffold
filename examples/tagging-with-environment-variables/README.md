### Example: using the envTemplate tag policy

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/GoogleContainerTools/skaffold&cloudshell_open_in_editor=README.md&cloudshell_workspace=examples/tagging-with-environment-variables)

This example uses an environment variable `FOO` to tag the image.
The way you configure it in `skaffold.yaml` is the following build stanza:

```yaml
build:
     artifacts:
     - image: skaffold-example
     tagPolicy:
       envTemplate:
         template: "{{.FOO}}"
```

1. define `tagPolicy` to be `envTemplate`
2. use [go templates](https://golang.org/pkg/text/template) syntax