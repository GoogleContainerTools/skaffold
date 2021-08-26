### Example: kaniko

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/GoogleContainerTools/skaffold&cloudshell_open_in_editor=README.md&cloudshell_workspace=examples/kaniko)

This is an example demonstrating:

* **building** a single Go file app and with a single stage `Dockerfile` using [kaniko](https://github.com/GoogleContainerTools/kaniko) to build on a K8S cluster
* **tagging** using the default tagPolicy (`gitCommit`)
* **deploying** a single container pod using `kubectl`

### GCP

If you are on GCP, create a [Service Account](https://cloud.google.com/iam/docs/understanding-service-accounts) for Kaniko that has permissions to pull and push images from/to `gcr.io`.

Download the json service account file, rename the file to `kaniko-secret` (do not append .json to the filename) and create a Kubernetes secret using the following example:

```
kubectl create secret generic kaniko-secret --from-file=kaniko-secret
```

Note the name of the secret *AND* the key must be `kaniko-secret`
