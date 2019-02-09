---
title: "Port forwarding"
linkTitle: "Port forwarding"
weight: 50
---

This page discusses how skaffold sets up port forwarding for container ports from pods. When skaffold deploys an application, it will automatically forward any ports mentioned in the pod spec.

For example, if we have the following pod manifest, skaffold will forward port 8000 to port 8000 on our machine:

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

If port 8000 isn't available, another random port will be chosen.
