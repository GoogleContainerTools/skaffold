---
title: "Log Tailing"
linkTitle: "Log Tailing"
weight: 40
featureId: logging
---

Skaffold has built-in support for tailing logs for containers **built and deployed by Skaffold** on your cluster
to your local machine when running in either `dev`, `debug` or `run` mode.

{{< alert title="Note" >}}
Log Tailing is **enabled by default** for [`dev`]({{<relref "/docs/workflows/dev" >}}) and [`debug`]({{<relref "/docs/workflows/debug" >}}).<br>
Log Tailing is **disabled by default** for `run` mode; it can be enabled with the `--tail` flag.
{{< /alert >}}


## Log Structure
To view log structure, run `skaffold run --tail` on [examples microserices](https://github.com/GoogleContainerTools/skaffold/tree/master/examples/microservices)

```bash
skaffold run --tail
```

will produce an output like this

![logging-output](/images/logging-output.png)


For every log line, skaffold will prefix the pod name and container name if they're not the same.

![logging-output](/images/log-line-single.png)

In the above example, `leeroy-web-75ff54dc77-9shwm` is the pod name and `leeroy-web` is container name
defined in the spec for this deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: leeroy-web
  labels:
    app: leeroy-web
spec:
  replicas: 1
  selector:
    matchLabels:
      app: leeroy-web
  template:
    metadata:
      labels:
        app: leeroy-web
    spec:
      containers:
        - name: leeroy-web
          image: gcr.io/k8s-skaffold/leeroy-web
          ports:
            - containerPort: 8080 
```

Skaffold will choose a unique color for each container to make it easy for users to read the logs.

