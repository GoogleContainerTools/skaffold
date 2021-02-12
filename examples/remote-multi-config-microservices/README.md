### Example: Remote config µSvcs with Skaffold

In this example:

* Deploy microservice applications from a remote git repository using Skaffold.

Skaffold can build and deploy from configurations defined in remote git repositories. In this example, we'll walk through using skaffold to deploy two applications, an exposed "web" frontend which calls an unexposed "app" backend from the [examples/multi-config-microservices](../multi-config-microservices) project as a remote dependency.

**WARNING: If you're running this on a cloud cluster, this example will create a service and expose a webserver.
It's highly suggested that you only run this example on a local, private cluster like minikube or Kubernetes in Docker for Desktop.**

#### Running the example on minikube

From this directory, run:

```bash
skaffold dev --port-forward
```

Now, in a different terminal, hit the `leeroy-web` endpoint

```bash
$ curl localhost:9000
leeroooooy app!!
```
Hitting `Ctrl + C` on the first terminal should kill the process and clean up the deployments.

#### Configuration walkthrough

The [`skaffold.yaml`](./skaffold.yaml) looks like:

```yaml
apiVersion: skaffold/v2beta11
kind: Config
requires:
- git:
    repo: https://github.com/GoogleContainerTools/skaffold
    path: examples/multi-config-microservices/leeroy-app
    ref: master

- git:
    repo: https://github.com/GoogleContainerTools/skaffold
    path: examples/multi-config-microservices/leeroy-web
    ref: master

```

There are two `git` dependencies from the same repository `GoogleContainerTools/skaffold`. You can add as many dependencies as you want across the same or different repositories; even between different branches of the same repository. Skaffold downloads each referenced repository (one copy per referenced branch) to its cache folder (`~/.skaffold/repos` by default).

The remote dependency caches should not be modified directly by the user. Skaffold will reset the cache to the latest from the remote on each run.