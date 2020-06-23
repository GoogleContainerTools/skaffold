# Examples

Each of those examples can be tried with `skaffold dev`. For example:

```
cd getting-started
skaffold dev
```

Read the [Quickstart](https://skaffold.dev/docs/quickstart/) for more detailed instructions.

These examples are made to work with the latest release of Skaffold.

If you are running Skaffold at HEAD or have built it from source, please use the examples at `integration/examples`.

*Note for contributors*: If you wish to make changes to these examples, please edit the ones at `integration/examples`,
as those will be synced on release.

## Deploying to a local cluster

When deploying to a [local cluster](https://skaffold.dev/docs/environment/local-cluster/) such as minikube or Docker Desktop, no additional configuration step is required.

## Deploying to a remote cluster

When deploying to a remote cluster you have to point Skaffold to your default image repository in one of the four ways:

* flag: `skaffold dev --default-repo <myrepo>`
* env var: `SKAFFOLD_DEFAULT_REPO=<myrepo> skaffold dev`
* global skaffold config (one time): `skaffold config set --global default-repo <myrepo>`
* skaffold config for current kubectl context: `skaffold config set default-repo <myrepo>`

