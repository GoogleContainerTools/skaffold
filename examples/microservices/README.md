### Example: µSvcs with Skaffold

In this example:

* Deploy multiple applications with skaffold
* In development, only rebuild and redeploy the artifacts that have changed
* Deploy multiple applications outside the working directory

In the real world, Kubernetes deployments will consist of multiple applications that work together.
In this example, we'll walk through using skaffold to develop and deploy two applications, an exposed "web" frontend which calls an unexposed "app" backend.

**WARNING: If you're running this on a cloud cluster, this example will create a service and expose a webserver.
It's highly suggested that you only run this example on a local, private cluster like minikube or Kubernetes in Docker for Desktop.**

#### Running the example on minikube

From this directory, run

```bash
skaffold dev
```

Now, in a different terminal, hit the `leeroy-web` endpoint

```bash
$ curl $(minikube service leeroy-web --url)
leeroooooy app!
```

Now, let's change the message in `leeroy-app` without changing `leeroy-web`.
Add a few exclamations points because this is exhilarating stuff.

In `leeroy-app/app.go`, change the message here

```golang
func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "leeroooooy app!!!\n")
}
```

Once you see the log message

```
[leeroy-app-5b4dfdcbc6-6vf6r leeroy-app] 2018/03/30 06:28:47 leeroy app server ready
```

Your service will be ready to hit again with

```bash
$ curl $(minikube service leeroy-web --url)
leeroooooy app!!!
```

#### Configuration walkthrough

Let's walk through the first part of the skaffold.yaml

```yaml
  artifacts:
  - image: leeroy-web
    context: ./leeroy-web/
  - image: leeroy-app
    context: ./leeroy-app/
```

We're deploying a `leeroy-web` image, which we build in the context of its subdirectory and a `leeroy-app` image built in a similar manner.

`leeroy-web` will listen for requests, and then make a simple HTTP call to `leeroy-app` using Kubernetes service discovery and return that result.

In the deploy stanza, we use the glob matching pattern to deploy all YAML and JSON files in the respective Kubernetes manifest directories.

```yaml
deploy:
  kubectl:
    manifests:
    - ./leeroy-web/kubernetes/*
    - ./leeroy-app/kubernetes/*
```

