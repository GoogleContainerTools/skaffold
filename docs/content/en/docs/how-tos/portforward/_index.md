
---
title: "Using port forwarding"
linkTitle: "Using port forwarding"
weight: 50
---

This page discusses how to set up Skaffold to setup port forwarding for container ports from pods with `skaffold dev`.

### Flags

|Flag|Description|
|-----|-----|
|`port-forward`| OPTIONAL. Set to false to disable automatic port-forwarding. |                    
|`port`| OPTIONAL. Specify a port to forward to in the form pod/container:localPort:containerPort. Set multiple times for multiple ports. |                    


### Background

Given the following pod definition: 

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: hello
spec:
  containers:
  - name: hello
    image: gcr.io/k8s-skaffold/hello-port-forward
    ports:
    - containerPort: 8080
```

skaffold will automatically attempt to forward container port 8080 to port 8080 on your local machine using `kubectl port-forward`.
If the port is unavailable locally, skaffold will forward to a random open port.

Similarly, upon a port collision, skaffold will forward any colliding ports to a random open port.

### Specifying a Local Port

To specify which local port a container port should be forwarded too, pass in the `--port` flag with arguments formatted as follows:

```
skaffold dev --port <my pod name>/<my container name>:localPort:containerPort
```

where the containerPort will be forwarded to the localPort on your machine. 
You can set this flag multiple times to specify multiple ports. 

