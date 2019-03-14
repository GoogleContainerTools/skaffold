# Examples

To run the examples, you either have to manually replace the image repositories in the examples from `gcr.io/k8s-skaffold`
to yours or you can point Skaffold to your default image repository in one of the four ways:

* flag: `skaffold dev --default-repo <myrepo>`
* env var: `SKAFFOLD_DEFAULT_REPO=<myrepo> skaffold dev`
* global skaffold config (one time): `skaffold config set --global default-repo <myrepo>`
* skaffold config for current kubectl context: `skaffold config set default-repo <myrepo>`

These examples are made to work with the latest release of Skaffold.

If you are running Skaffold at HEAD or have built it from source, please use the examples at `integration/examples`.

*Note for contributors*: If you wish to make changes to these examples, please edit the ones at `integration/examples`,
as those will be synced on release.
