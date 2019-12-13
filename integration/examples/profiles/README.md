### Example: Getting started with a simple go app

This is a simple show-case of Skaffold profiles

#### Init

Use the `--profile` option to enable a particular profile `skaffold dev --profile staging-profile`

#### Workflow

* Build only the `world-service` when using the main profile
* Activate `minikube-profile` automatically when the current context is `minikube`. Only build the `hello-service` in that case.
* Build both services when the `staging-profile` is used. Override the kube-context to `staging` in that case.
