---
title: "CI/CD with Skaffold"
linkTitle: "CI/CD with Skaffold"
weight: 30

---

Skaffold offers several sub-commands for its workflows that make it quite flexible when integrating with CI/CD pipelines.

## `skaffold build | skaffold deploy`

`skaffold build` will build your project's artifacts, and push the build images to the specified registry. If your project is already configured to run with Skaffold, `skaffold build` can be a very lightweight way of setting up builds for your CI pipeline. Passing the `--file-output` flag to Skaffold build will also write out your built artifacts in JSON format to a file on disk, which can then by passed to `skaffold deploy` later on. This is a great way of "committing" your artifacts when they have reached a state that you're comfortable with, especially for projects with multiple artifacts for multiple services.

Example using the current git state as a unique file ID to "commit" build state:

Storing the build result in a commit specific JSON file:
```bash
export STATE=$(git rev-list -1 HEAD --abbrev-commit)
skaffold build --file-output build-$STATE.json
```
outputs the tag generation and cache output from Skaffold:
```bash 
Generating tags...
 - gcr.io/k8s-skaffold/skaffold-example:v0.41.0-17-g3ad238db
Checking cache...
 - gcr.io/k8s-skaffold/skaffold-example: Found. Tagging
```

The content of the JSON file
```bash 
cat build-$STATE.json
```
looks like: 
```json
{"builds":[{"imageName":"gcr.io/k8s-skaffold/skaffold-example","tag":"gcr.io/k8s-skaffold/skaffold-example:v0.41.0-17-g3ad238db@sha256:eeffb639f53368c4039b02a4d337bde44e3acc728b309a84353d4857ee95c369"}]}
```

We can then use this build result file to deploy with Skaffold:
```bash
skaffold deploy -a build-$STATE.json
```
and as we'd expect, we see a bit of deploy-related output from Skaffold:
```bash
Tags used in deployment:
 - gcr.io/k8s-skaffold/skaffold-example -> gcr.io/k8s-skaffold/skaffold-example:v0.41.0-17-g3ad238db@sha256:eeffb639f53368c4039b02a4d337bde44e3acc728b309a84353d4857ee95c369
Starting deploy...
 - pod/getting-started configured
```


## `skaffold render` 
{{< maturity "render" >}}

Skaffold also has another built-in command, `skaffold render`, that will perform builds on all artifacts in your project, template the newly built image tags into your Kubernetes deployment configuration files (based on your configured deployer), and instead of sending these through the deployment process, print out the final deployment artifacts. This allows your to snapshot your project's builds, but also integrate those builds into your deployment configs to snapshot your deployment as well. This can be very useful when integrating with GitOps based workflows: these templated deployment configurations can be committed to a Git repository as a way to deploy using GitOps.

Example of running `skaffold render` to render Kubernetes manifests, then sending them directly to `kubectl`:

Running `skaffold render --output render.txt && cat render.txt` outputs:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  namespace: default
spec:
  containers:
  - image: gcr.io/k8s-skaffold/skaffold-example:v0.41.0-57-gbee90013@sha256:eeffb639f53368c4039b02a4d337bde44e3acc728b309a84353d4857ee95c369
    name: getting-started
```

We can then pipe this yaml to kubectl:
```code
cat render.txt | kubectl apply -f -
```
which shows
```
pod/getting-started configured
```

Or, if we want to skip the file writing altogether:

```code
skaffold render | kubectl apply -f -
```

gives us the one line output telling us the only thing we need to know:
```code
pod/getting-started configured
```

## Waiting for Skaffold deployments using `healthcheck`
{{< maturity "deploy.status_check" >}}
