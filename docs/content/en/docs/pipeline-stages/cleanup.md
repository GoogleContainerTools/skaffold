---
title: "Cleanup"
linkTitle: "Cleanup"
weight: 60
featureId: cleanup
---

Skaffold works with [image builders]({{<relref "/docs/pipeline-stages/builders">}}) and [deployers]({{<relref "/docs/pipeline-stages/deployers">}}) that both have side effects on the environment, namely: 

- resources are created in one or more namespaces in a Kubernetes cluster 
- images are created on the local Docker daemon and registries
    
Skaffold offers functionality to cleanup these changes:

- Kubernetes resource cleanup - `skaffold delete` command and `--cleanup=true` flag for `skaffold dev` and `skaffold debug`  
- Image pruning - for local Docker daemon images 

## Kubernetes resource cleanup 

After you ran `skaffold run` or `skaffold deploy` and deployed your application to a cluster, running `skaffold delete` will remove all the resources you deployed.
The easiest to think about `skaffold delete` is that it is equivalent to `kubectl delete -f <all your manifests>`.

In case of `skaffold dev` and `skaffold debug`, pressing `Ctrl+C` will kick off the cleanup process which will run a `skaffold delete` essentially.  
 
## Image pruning 
 

