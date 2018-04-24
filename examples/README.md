# KubeCon demo

## Full script

```
$ ls
$ kubectl get nodes
$ kubectl get pods
$ skaffold dev
# In a new tab
$ vi main.go

$ kctx gke_dga-demo_europe-west1-d_kubecon
$ kubectl get nodes
$ kubectl get pods
$ skaffold run -p gcb
$ kubectl get pods
$ kubectl logs -f getting-started

$ kctx -
$ skaffold dev
# In a new tab
$ cd ui/static
$ vi index.html

```