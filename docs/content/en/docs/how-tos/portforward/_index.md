---
title: "Port forwarding"
linkTitle: "Port forwarding"
weight: 50
---

This page discusses how Skaffold sets up port forwarding for container ports from pods. 
Port forwarding is set to false by default; you can enable it with the `--port-forward` flag for `skaffold dev` and `skaffold debug`. 
When this flag is set, skaffold will automatically forward any ports mentioned in the pod spec.

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
If port 8000 isn't available, another random port will be chosen. Currently, only containers that contain images specified as skaffold artifacts will be port forwarded. In other words, port forwarding will not work for containers which reference images not built by the skaffold itself (e.g. official images hosted on 3rd party container registries such as Docker Hub, docker.elastic.co, etc.). We're working on adding user defined port-forwarding, which would allow you to specify additional containers to port-forward.
{{< /alert >}}
