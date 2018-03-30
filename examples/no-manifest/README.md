## Deploy Images without Kubernetes Manifests

If you're using the `kubectl` deployer, you can deploy images without an accompanying kubernetes manifest. Skaffold will provide a very simple deployment template for the image.

From this directory, run

```bash
$ skaffold run
```

This will deploy a simple webserver example that echos back the URL path. Skaffold won't expose this service, so its up to you to either expose it or access it a different way. In this example, we'll exec into the pod and curl the endpoint from there.

Find the deployed pod

```
$ kubectl get po
NAME                          READY     STATUS    RESTARTS   AGE
skaffold-6b944b5787-6b8ml     1/1       Running   0          3s
```

Then, exec into the pod and run curl

```
$ kubectl exec -it skaffold-6b944b5787-6b8ml curl localhost:8080/hello
Hi there, I love hello!
```


## Configuration walkthrough

To have skaffold provide a deployment for you, simply leave the manifests list empty in the deploy stanza

```yaml
deploy:
  kubectl:
    manifests:
```
