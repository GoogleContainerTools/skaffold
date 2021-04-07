### Example: deploy remote helm chart

This example shows how to deploy a remote helm chart from a remote repo. This can be helpful  for consuming other helm packages as part of your app.


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
