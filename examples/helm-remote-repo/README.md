### Example: deploy remote helm chart

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/GoogleContainerTools/skaffold&cloudshell_open_in_editor=README.md&cloudshell_workspace=examples/helm-remote-repo)

This example shows how to deploy a remote helm chart from a remote repo. This can be helpful for consuming other helm packages as part of your app.


```yaml
deploy:
  helm:
    releases:
    - name: redis-release
      repo: https://charts.bitnami.com/bitnami 
      remoteChart: redis
```

This is the equivalent of the following on helm CLI:

```bash
helm repo add bitnami https://charts.bitnami.com/bitnami
helm install redis-release bitnami/redis
```
