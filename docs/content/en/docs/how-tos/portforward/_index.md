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

Skaffold will perform automatic port forwarding for resources that it manages:

* all **services it deploys** for both `skaffold dev` and `skaffold debug`.
* all **pods it deploys**, but only including containers that run **skaffold built images**, for `skaffold debug`. 

### User Defined Port Forwarding

Users can also define additional resources to port forward in the skaffold config, to enable port forwarding for 

* additional resource types supported by `kubectl port-forward` e.g.`Deployment`or `ReplicaSet`.
* additional pods running containers which run images not built by Skaffold.

For example:

```yaml
portForward:
- resourceType: deployment
  resourceName: myDep
  namespace: mynamespace  # 
  port: 8080 # 
  localPort: 9000 # *Optional*
```

For this example, Skaffold will attempt to forward port 8080 to `localhost:9000`.
If port 9000 is unavailable, Skaffold will forward to a random open port. 
 
Skaffold will run `kubectl port-forward` on each of these resources in addition to the automatic port forwarding described above.
Acceptable resource types include: `Service`, `Pod` and Controller resource type that has a pod spec: `ReplicaSet`, `ReplicationController`, `Deployment`, `StatefulSet`, `DaemonSet`, `Job`, `CronJob`. 

//TODO: table/explain for which field is optional/mandatory in which cases 
//"little" diagram of port forwarding and log tailing from skaffold to one of the multiple pods


