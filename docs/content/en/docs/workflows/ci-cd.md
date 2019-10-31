---
title: "CI/CD with Skaffold"
linkTitle: "CI/CD with Skaffold"
weight: 3
---

Skaffold offers several sub-commands for its workflows that make it quite flexible when integrating with CI/CD pipelines.

## `skaffold build` | `skaffold deploy`

`skaffold build` will build your project's artifacts, and push the build images to the specified registry. If your project is already configured to run with Skaffold, `skaffold build` can be a very lightweight way of setting up builds for your CI pipeline. Passing the `--file-output` flag to Skaffold build will also write out your built artifacts in JSON format to a file on disk, which can then by passed to `skaffold deploy` later on. This is a great way of "committing" your artifacts when they have reached a state that you're comfortable with, especially for projects with multiple artifacts for multiple services.

Example using the current git state as a unique file ID to "commit" build state:

```code
➜  getting-started git:(docs) ✗ export STATE=$(git rev-list -1 HEAD --abbrev-commit)

➜  getting-started skaffold build --file-output build-$STATE.json
Generating tags...
 - gcr.io/k8s-skaffold/skaffold-example:v0.41.0-17-g3ad238db
Checking cache...
 - gcr.io/k8s-skaffold/skaffold-example: Found. Tagging

➜  getting-started cat build-$STATE.json
{"builds":[{"imageName":"gcr.io/k8s-skaffold/skaffold-example","tag":"gcr.io/k8s-skaffold/skaffold-example:v0.41.0-17-g3ad238db@sha256:eeffb639f53368c4039b02a4d337bde44e3acc728b309a84353d4857ee95c369"}]}

➜  getting-started git:(docs) ✗ skaffold deploy -a build-$STATE.json
Tags used in deployment:
 - gcr.io/k8s-skaffold/skaffold-example -> gcr.io/k8s-skaffold/skaffold-example:v0.41.0-17-g3ad238db@sha256:eeffb639f53368c4039b02a4d337bde44e3acc728b309a84353d4857ee95c369
Starting deploy...
 - pod/getting-started configured
```

## `skaffold render`

Skaffold also has another built-in command, `skaffold render`, that will perform builds on all artifacts in your project, template the newly built image tags into your Kubernetes deployment configuration files (based on your configured deployer), and instead of sending these through the deployment process, print out the final deployment artifacts. This allows your to snapshot your project's builds, but also integrate those builds into your deployment configs to snapshot your deployment as well. This can be very useful when integrating with GitOps based workflows: these templated deployment configurations can be committed to a Git repository as a way to deploy using GitOps.

Example of running `skaffold render` to render Kubernetes manifests, then sending them directly to `kubectl`:

```code
➜  getting-started skaffold render --output render.txt
➜  getting-started cat render.txt
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  namespace: default
spec:
  containers:
  - image: gcr.io/k8s-skaffold/skaffold-example:v0.41.0-57-gbee90013@sha256:eeffb639f53368c4039b02a4d337bde44e3acc728b309a84353d4857ee95c369
    name: getting-started

➜  getting-started cat render.txt | kubectl apply -f -
pod/getting-started configured
```

Or, skipping the file writing altogether:

```code
➜  getting-started skaffold render | kubectl apply -f -
pod/getting-started configured
```
