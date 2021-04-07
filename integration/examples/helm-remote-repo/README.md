### Example: deploy remote helm chart

This example shows how to deploy a remote helm chart from a remote repo. This can be helpfulf or consuming other helm packages as part of your app.


```yaml
deploy:
  helm:
    releases:
    - name: redis-release
      repo: https://charts.bitnami.com/bitnami 
      remoteChart: redis
```
