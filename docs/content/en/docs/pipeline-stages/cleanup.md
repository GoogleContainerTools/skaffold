---
title: "Cleanup"
linkTitle: "Cleanup"
weight: 60
featureId: cleanup
---

Skaffold works with [image builders]({{<relref "/docs/pipeline-stages/builders">}}) and [deployers]({{<relref "/docs/pipeline-stages/deployers">}}) 
that both have side effects on both your local and deployment environments: 

- resources are created in one or more namespaces in a Kubernetes cluster 
- images might be created on the local Docker daemon
- images might be pushed to registries
- application might have arbitrary side effects 
    
Skaffold offers cleanup functionality to negate some of these side effects:

- Kubernetes resource cleanup - `skaffold delete`, and automatic cleanup on `Ctrl+C` for `skaffold dev` and `skaffold debug`   
- Image pruning - for local Docker daemon images only, automatically on `Ctrl+C` for `skaffold dev` and `skaffold debug` 

For pushed images in registries and application side effects the user has to take care of cleanup. 

## Kubernetes resource cleanup 
 
After running `skaffold run` or `skaffold deploy` and deploying your application to a cluster, running `skaffold delete` will remove all the resources you deployed.
Cleanup is enabled by default, it can be turned off by `--cleanup=false`. 

## Ctrl + C 

When running `skaffold dev` or `skaffold debug`, pressing `Ctrl+C` (`SIGINT` signal) will kick off the cleanup process which will mimic the behavior of `skaffold delete`.
If for some reason the Skaffold process was unable to catch the `SIGINT` signal, `skaffold delete` can always be run later to clean up the deployed Kubernetes resources.
 
### Image pruning 
 
Images that are built by Skaffold and stored on the local Docker daemon can easily pile up, taking up a significant amount of disk space.
To avoid this, users can turn on image pruning that deletes the images built by Skaffold on `SIGTERM` from `skaffold dev` and `skaffold debug`.  

{{< alert title="Note" >}}
Image pruning is only available if artifact caching is disabled.<br>
As artifact caching is enabled by default, image pruning is disabled by default.
{{</alert>}}

To enable image pruning, you can run Skaffold with both `--no-prune=false` and `--cache-artifacts=false`:

 ```bash
skaffold dev --no-prune=false --cache-artifacts=false
```

outputs: 

```bash
Listing files to watch...
 - gcr.io/k8s-skaffold/skaffold-example
Generating tags...
 - gcr.io/k8s-skaffold/skaffold-example -> gcr.io/k8s-skaffold/skaffold-example:v0.41.0-148-gd2f3e0539
Building [gcr.io/k8s-skaffold/skaffold-example]...
Sending build context to Docker daemon  3.072kB
Step 1/6 : FROM golang:1.12.9-alpine3.10 as builder
 ---> e0d646523991
Step 2/6 : COPY main.go .
 ---> Using cache
 ---> 964ce43c7a63
Step 3/6 : RUN go build -o /app main.go
 ---> Using cache
 ---> 1fece4643da6
Step 4/6 : FROM alpine:3.10
 ---> 961769676411
Step 5/6 : CMD ["./app"]
 ---> Using cache
 ---> 256b146875d2
Step 6/6 : COPY --from=builder /app .
 ---> Using cache
 ---> f7a2f5c3a2f6
Successfully built f7a2f5c3a2f6
Successfully tagged gcr.io/k8s-skaffold/skaffold-example:v0.41.0-148-gd2f3e0539
Tags used in deployment:
 - gcr.io/k8s-skaffold/skaffold-example -> gcr.io/k8s-skaffold/skaffold-example:v0.41.0-148-gd2f3e0539@sha256:00d7fa06c313f7d06ad3d4701026e0ee65f8f437c703172f160df37c0059b3b1
Starting deploy...
 - pod/getting-started created
Watching for changes...
[getting-started] Hello world!
[getting-started] Hello world!
[getting-started] Hello world!
```

And after hitting Ctrl+C:

```bash 
^CCleaning up...
 - pod "getting-started" deleted
Pruning images...
untagged image gcr.io/k8s-skaffold/skaffold-example:v0.41.0-148-gd2f3e0539
untagged image gcr.io/k8s-skaffold/skaffold-example:v0.41.0-58-g8c428b975
untagged image gcr.io/k8s-skaffold/skaffold-example@sha256:00d7fa06c313f7d06ad3d4701026e0ee65f8f437c703172f160df37c0059b3b1
deleted image sha256:f7a2f5c3a2f6721989598d09a09ad70134936db398d93303ebb3545de2d32e22
deleted image sha256:c069434a51c8d96f68a95c13bbccd3512849f9ccfe5defbb80af7e342a48bbba

```

