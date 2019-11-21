---
title: "Port Forwarding"
linkTitle: "Port Forwarding"
weight: 50
featureId: portforward
aliases: [/docs/how-tos/portforward]
---

Skaffold has built-in support for forwarding ports for exposed Kubernetes resources on your cluster
to your local machine when running in either `dev` or `debug` mode.

**Port forwarding is disabled by default; it can be enabled with the `--port-forward` flag.**
**If this flag is not set, no port forwarding will occur!**

When port forwarding is enabled, Skaffold will:

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


| Field        | Values           | Mandatory  |
| ------------- |-------------| -----|
| resourceType     | `pod`, `service`, `deployment`, `replicaset`, `statefulset`, `replicationcontroller`, `daemonset`, `job`, `cronjob` | Yes | 
| resourceName     | Name of the resource to forward.     | Yes | 
| namespace  | The namespace of the resource to port forward.     | No. Defaults to current namespace, or `default` if no current namespace is defined | 
| port | Port is the resource port that will be forwarded. | Yes |
| address | Address is the address on which the forward will be bound. | No. Defaults to `127.0.0.1` |
| localPort | LocalPort is the local port to forward too. | No. Defaults to value set for `port`. |


Skaffold will run `kubectl port-forward` on all user defined resources.
`kubectl port-forward` will select one pod created by that resource to forward too.

For example, forwarding a deployment that creates 3 replicas could look like this:

```yaml
portForward:
- resourceType: deployment
  resourceName: myDep
  namespace: mynamespace
  port: 8080
  localPort: 9000
```

![portforward_deployment](/images/portforward.png)

If you want the port forward to to be available from other hosts and not from the local host only, you can bind
the port forward to the address `0.0.0.0`:

```yaml
portForward:
- resourceType: deployment
  resourceName: myDep
  namespace: mynamespace
  port: 8080
  address: 0.0.0.0
  localPort: 9000
```
