### Example: Multiple configs µSvcs with Skaffold

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/GoogleContainerTools/skaffold&cloudshell_open_in_editor=README.md&cloudshell_workspace=examples/multi-config-microservices)

In this example:

* Deploy microservice applications individually or in a group.
* Compose multiple skaffold configs together in a single config.

In the real world, Kubernetes deployments will consist of multiple applications that work together. These applications may not necessarily live in the same repository, and it might be difficult to define a single `skaffold.yaml` configuration to describe all these disparate applications together.

Skaffold solves this problem by allowing each application to define its own `skaffold.yaml` configuration file that is only scoped to that specific app. These related configs can also be grouped together into another `skaffold.yaml` config when needed.

In this example, we'll walk through using skaffold to develop and deploy two applications, an exposed "web" frontend which calls an unexposed "app" backend.

**WARNING: If you're running this on a cloud cluster, this example will create a service and expose a webserver.
It's highly suggested that you only run this example on a local, private cluster like minikube or Kubernetes in Docker for Desktop.**

#### Running the example on minikube

From the `multi-config-microservices/leeroy-app` directory, run

```bash
skaffold dev
```

Now, in a different terminal, from the `multi-config-microservices/leeroy-web` directory, again run

```bash
skaffold dev
```

Now, in a different terminal, hit the `leeroy-web` endpoint

```bash
$ curl $(minikube service leeroy-web --url)
leeroooooy app!
```

These are two independently managed instances of Skaffold running on the `leeroy-app` and `leeroy-web` applications. Hitting `Ctrl + C` on each should kill the process and clean up the deployments.

In order to iterate on both apps together we reference them as **required** configs in the `multi-config-microservices/skaffold.yaml` file.

```yaml
apiVersion: skaffold/v2beta11
kind: Config
requires:
- path: ./leeroy-app
- path: ./leeroy-web
```

Now, from the `multi-config-microservices` directory, again run:

```bash
skaffold dev
```

The two applications should be built and deployed like before but in the same Skaffold session. If you want to go back to iterating on the applications individually you can simply pass in the `--module` or `-m` flag with the `metadata.name` value of the config that you want to activate.

```bash
skaffold dev -m app-config
```
