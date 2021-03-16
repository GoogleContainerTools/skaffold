---
title: "Port Forwarding"
linkTitle: "Port Forwarding"
weight: 50
featureId: portforward
aliases: [/docs/how-tos/portforward]
---

Skaffold has built-in support for forwarding ports from exposed Kubernetes resources on your cluster
to your local machine when running in either `dev` or `debug` mode.

### Automatic Port Forwarding

Skaffold supports automatic port forwarding the following classes of resources:

- `user`: explicit port-forwards defined in the `skaffold.yaml` (called [_user-defined port forwards_](#UDPF))
- `services`: ports exposed on services deployed by Skaffold.
- `debug`: debugging ports as enabled by `skaffold debug` for Skaffold-built images.
- `pods`: all `containerPort`s on deployed pods for Skaffold-built images.

Skaffold enables certain classes of forwards by default depending on the Skaffold command used.
These defaults can be overridden with the `--port-forward` flag, and port-forwarding can be
disabled with `--port-forward=none`.

Command-line                          | Default modes
------------------------------------- | -------------------
`skaffold dev`                        | `user`
`skaffold dev --port-forward`         | `user`, `services`
`skaffold dev --port-forward=none`    | _no ports forwarded_
`skaffold debug`                      | `user`, `debug`
`skaffold debug --port-forward`       | `user`, `services`, `debug` <small>(<em>see note below</em>)</small>
`skaffold debug --port-forward=none`  | _no ports forwarded_

{{< alert title="Compatibility Note" >}}
Note that `skaffold debug --port-forward` previously enabled the
equivalent of `pods` as Skaffold did not have an equivalent of `debug`. 
We have replaced `pods` as it caused confusion.
{{< /alert >}}

### User-Defined Port Forwarding {#UDPF}

Users can define additional resources to port forward in the skaffold config, to enable port forwarding for 

* additional resource types supported by `kubectl port-forward` e.g.`Deployment`or `ReplicaSet`.
* additional pods running containers which run images not built by Skaffold.

For example:

```yaml
portForward:
- resourceType: deployment
  resourceName: myDep
  namespace: mynamespace
  port: 8080
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
