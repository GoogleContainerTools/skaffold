---
title: "Port forwarding"
linkTitle: "Port forwarding"
weight: 50
---

This page discusses how Skaffold sets up port forwarding for container ports from pods. When Skaffold deploys an application, it will automatically forward any ports mentioned in the pod spec.

### Example

With the following pod manifest, Skaffold will forward port 8000 to port 8000 on our machine:

```
apiVersion: v1
kind: Pod
metadata:
  name: example
spec:
  containers:
  - name: skaffold-example
    image: gcr.io/k8s-skaffold/skaffold-example
    ports:
      - name: web
        containerPort: 8000
```

{{< alert title="Note" >}}
If port 8000 isn't available, another random port will be chosen.
{{< /alert >}}
