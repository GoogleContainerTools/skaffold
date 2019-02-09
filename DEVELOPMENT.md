# Development

This doc explains the development workflow so you can get started
[contributing](CONTRIBUTING.md) to Skaffold!

## Getting started

First you will need to setup your GitHub account and create a fork:

1. Create [a GitHub account](https://github.com/join)
1. Setup [GitHub access via
   SSH](https://help.github.com/articles/connecting-to-github-with-ssh/)
1. [Create and checkout a repo fork](#checkout-your-fork)

Once you have those, you can iterate on skaffold:

1. [Build your dev version of skaffold](#building-skaffold)
1. [Verify changes locally](#verifying-local-changes)
1. [Run skaffold tests](#testing-skaffold)
1. [Build docs](#building-skaffold-docs) if you are making doc changes

When you're ready, you can [create a PR](#creating-a-pr)!

You may also be interested in [contributing to the docs](#contributing-to-skaffold-docs).

## Checkout your fork

The Go tools require that you clone the repository to the `src/github.com/GoogleContainerTools/skaffold` directory
in your [`GOPATH`](https://github.com/golang/go/wiki/SettingGOPATH).

To check out this repository:

1. Create your own [fork of this
  repo](https://help.github.com/articles/fork-a-repo/)

1. Clone it to your machine:

   ```shell
   mkdir -p ${GOPATH}/src/github.com/GoogleContainerTools
   cd ${GOPATH}/src/github.com/GoogleContainerTools
   git clone git@github.com:${YOUR_GITHUB_USERNAME}/skaffold.git
   cd skaffold
   git remote add upstream git@github.com:GoogleContainerTools/skaffold.git
   git remote set-url --push upstream no_push
   ```

   _Adding the `upstream` remote sets you up nicely for regularly [syncing your
   fork](https://help.github.com/articles/syncing-a-fork/)._

## Building skaffold

To build with your local changes you have two options:

1. Build the skaffold binary:

   ```shell
   make
   ./out/skaffold version
   ```

   You can then run this binary directly, or copy/symlink it into your path.

1. Build and install the skaffold binary:

   ```shell
   make install
   skaffold version
   ```

   This will install skaffold via `go install` (note that if you have [manually downloaded
   and installed skaffold to `/usr/bin/local`](README.md#installation), this is will probably
   take precedence in your path over your `$GOPATH/bin`).

   _If you are unsure if you are running a released or locally built version of skaffold, you
   can run `skaffold version` - output which includes `dirty` indicates you have built the
   binary locally._

## Verifying local changes

If you are iterating on skaffold and want to see your changes in action, you can:

1. [Build skaffold](#building-skaffold)
2. [Use the quickstart example](README.md#iterative-development)

## Testing skaffold

skaffold has both [unit tests](#unit-tests) and [integration tests](#integration-tests).

### Unit Tests

The unit tests live with the code they test and can be run with:

```shell
make test
```

_These tests will not run correctly unless you have [checked out your fork into your `$GOPATH`](#checkout-your-fork)._

### Integration tests

The integration tests live in [`integration`](./integration) and run the [`examples`](./examples)
as tests. They can be run with:

```shell
make integration-test
```

_These tests require push access to a project in GCP, and so can only be run
by maintainers who have access. These tests will be kicked off by [reviewers](#reviews)
for submitted PRs._

## Building skaffold docs

The latest version of the skaffold site is based on the Hugo theme of the github.com/google/docsy template.  

### Testing docs locally 

Before [creating a PR](#creating-a-pr) with doc changes, we recommend that you locally verify the
generated docs with:

```shell
make preview-docs
```
Once PRs with doc changes are merged, they will get automatically published to the docs
for the latest build to https://skaffold-latest.firebaseapp.com.
which at release time will be published with the latest release to https://skaffold.dev.

### Previewing the docs on the PR

Mark your PR with `docs-modifications` label. Our PR review process will answer in comments in ~5 minutes with the URL of your preview and will remove the label. 

## Testing the Skaffold binary release process  

Skaffold release process works with Google Cloud Build within our own project `k8s-skaffold` and the skaffold release bucket, `gs://skaffold`. 

In order to be able to iterate/fix the release process you can pass in your own project and bucket as parameters to the build. 

We continuously release **builds** under `gs://skaffold/builds`. This is done by triggering `cloudbuild.yaml` on every push to master. 

To run a build on your own project: 

```
gcloud builds submit --config deploy/cloudbuild.yaml --substitutions=_RELEASE_BUCKET=<personal-bucket>,COMMIT_SHA=$(git rev-parse HEAD) --project <personalproject>
```  

We **release** stable versions under `gs://skaffold/releases`. This is done by triggering `cloudbuild-release.yaml` on every new tag in our Github repo.

To test a release on your own project:
                                                          
```
gcloud builds submit --config deploy/cloudbuild-release.yaml --substitutions=_RELEASE_BUCKET=<personal-bucket>,TAG_NAME=testrelease_v1234 --project <personalproject>
```                                                      

Note: if gcloud submit fails with something similar to the error message below, run `dep ensure && dep prune` to remove the broken symlinks   
```
ERROR: gcloud crashed (OSError): [Errno 2] No such file or directory: './vendor/github.com/karrick/godirwalk/testdata/symlinks/file-symlink'

```

To just run a release without Google Cloud Build only using your local Docker daemon, you can run: 

```
make -j release GCP_PROJECT=<personalproject> RELEASE_BUCKET=<personal-bucket>
``` 

## Creating a PR

When you have changes you would like to propose to skaffold, you will need to:

1. Ensure the commit message(s) describe what issue you are fixing and how you are fixing it
   (include references to [issue numbers](https://help.github.com/articles/closing-issues-using-keywords/)
   if appropriate)
1. [Create a pull request](https://help.github.com/articles/creating-a-pull-request-from-a-fork/)

### Reviews

Each PR must be reviewed by a maintainer. This maintainer will add the `kokoro:run` label
to a PR to kick of [the integration tests](#integration-tests), which must pass for the PR
to be submitted.
