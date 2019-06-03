---
title: "Port forwarding"
linkTitle: "Port forwarding"
weight: 50
---

This page discusses how to set up port forwarding with Skaffold for `skaffold dev` and `skaffold debug`.

Port forwarding is set to false by default; it is enabled with the `--port-forward` flag.
If this flag is not set, no port forwarding will occur. 
If the flag is set, Skaffold will:

1. Set up automatic port forwarding as described in the following section
2. Port forward any user defined resources in the Skaffold config


### Automatic Port Forwarding

Skaffold will perform automatic port forwarding as follows:

* automatic port forwarding of services for `skaffold dev` and `skaffold debug`
* automatic port forwarding of pods for `skaffold debug`

Skaffold will autmatically port forward all services it deploys for both `skaffold dev` and `skaffold debug`.

Skaffold will also automatically port forward pods, including only containers that run artifacts built by skaffold, for `skaffold debug`. 


### User Defined Port Forwarding

Users can also define additional resources to port forward in the skaffold config.
This is useful for forwarding additional resources like deployments or replica sets.
This is also useful for forwarding additional containers which run images not built by Skaffold.

For example:

```yaml
portForward:
- resourceType: pod
  resourceName: myPod
  namespace: mynamespace 
  port: 8080
  targetPort: 9000 # *Optional*
```

For this example, Skaffold will attempt to forward port 8080 to `localhost:9000`.
If port 9000 is unavailable, Skaffold will forward to a random open port. 
 
Skaffold will run `kubectl port-forward` on each of these resources in addition to the automatic port forwarding described above.
Acceptable resource types include: `pod`, `deployment`, `replicaset`, and `service`. 
