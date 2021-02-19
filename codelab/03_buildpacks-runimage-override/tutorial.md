# Using custom run image for CNB builder

## Introduction

### What is CNB?
[Cloud Native Buildpacks](https://buildpacks.io/) (CNB) enable building
a container image from source code without the need for a Dockerfile.
Skaffold supports building with CNB, requiring only
a local Docker daemon. 

CNB uses two images when building an application image:
  - A _run image_ serves as the base image for the resulting application image.
  - A _build image_ acts as the host for performing the build.

### What you'll learn

- how to annotate an arbitrary image and use it as a CNB 
[run image](https://buildpacks.io/docs/concepts/components/stack/).
- how to integrate the custom run image with a sample Buildpacks application using Skaffold's *artifact dependencies* feature.

___

**Time to complete**: <walkthrough-tutorial-duration duration=10></walkthrough-tutorial-duration>

Click the **Start** button to move to the next step.

## First steps

### Start a Minikube cluster

We'll use `minikube` as our local kubernetes cluster of choice.

Run:
```bash
minikube start
```

## Write a custom CNB run image

CNB run and build images require some additional metadata to identify the _Stack ID_ and user/group accounts to be used.  A [_stack_](https://buildpacks.io/docs/concepts/components/stack/) is a specification or contract.  For example, the `io.buildpacks.stacks.bionic` stack defines that it provides the same packages as installed on Ubuntu 18.04.

We add a base artifact with a single <walkthrough-editor-open-file filePath="base/Dockerfile">Dockerfile</walkthrough-editor-open-file> that defines the required metadata, and reference this as an <walkthrough-editor-select-line filePath="skaffold.yaml" startLine="4" startCharacterOffset="4" endLine="6" endCharacterOffset="0">artifact</walkthrough-editor-select-line> called `base` in our `skaffold.yaml`.

Next we'll use this artifact as the run image for a sample Buildpacks app.

<walkthrough-footnote>
    We will use the `gcr.io/buildpacks/builder:v1` builder image which supports the Stack ID `google`. So that's what we added to the Dockerfile.
</walkthrough-footnote>

## Use it in a sample Buildpacks app

We use a simple <walkthrough-editor-open-file filePath="app/main.go">Go application</walkthrough-editor-open-file> and reference it as an <walkthrough-editor-select-line filePath="skaffold.yaml" startLine="6" startCharacterOffset="4" endLine="8" endCharacterOffset="0">artifact</walkthrough-editor-select-line> called `app` in our `skaffold.yaml`. 

To use the `base` artifact as the custom run image we:
- add an <walkthrough-editor-select-line filePath="skaffold.yaml" startLine="13" startCharacterOffset="4" endLine="15" endCharacterOffset="0">artifact dependency</walkthrough-editor-select-line> for `app` artifact on the `base` artifact.
- set the <walkthrough-editor-select-line filePath="skaffold.yaml" startLine="10" startCharacterOffset="6" endLine="11" endCharacterOffset="0">runImage</walkthrough-editor-select-line> property of the `app` artifact to be the `base` artifact.

## Run it!

Run this command and Skaffold should take care of building the artifacts in order and deploying the provided <walkthrough-editor-open-file filePath="k8s/web.yaml">manifest</walkthrough-editor-open-file>.

```bash
skaffold dev --port-forward
```

Once the image has been built and deployed click on the <walkthrough-web-preview-icon></walkthrough-web-preview-icon> icon and select `Preview on port 8080`. This should redirect to the running service and show the output:

```
Hello, World!
```

The Go app reads <walkthrough-editor-open-file filePath="base/hello.txt">hello.txt</walkthrough-editor-open-file> that's provided by the base artifact. Lets change the text from <walkthrough-editor-select-line filePath="base/hello.txt" startLine="0" startCharacterOffset="0" endLine="1" endCharacterOffset="0">Hello, World!</walkthrough-editor-select-line> to `Hello, Buildpacks!`. This should trigger a rebuild of the `base` artifact which in turn triggers a rebuild and redeploy for the `app` artifact. Once that completes click on the <walkthrough-web-preview-icon></walkthrough-web-preview-icon> icon again and select `Preview on port 8080`. This should redirect to the running service and show the output:

```
Hello, Buildpacks!
```

## Congratulations

<walkthrough-conclusion-trophy></walkthrough-conclusion-trophy>

All done!

You now know how to use Buildpacks with custom run images and use Skaffold to tie the loop together.

