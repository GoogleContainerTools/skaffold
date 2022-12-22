---
title: "Quickstart"
linkTitle: "Quickstart"
weight: 20
---
{{% tabs %}}

{{% tab "STANDALONE" %}}

Follow this tutorial if you're using the Skaffold [standalone binary]({{< relref "../install/#standalone-binary" >}}). It walks through running Skaffold on a small Kubernetes app built with [Docker](https://www.docker.com/) inside [minikube](https://minikube.sigs.k8s.io)
and deployed with [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/).

{{< alert title="Note" >}}
If you are looking to generate a new project templated to use Skaffold best-practices and features, see the [Google Cloud Solutions Template](https://github.com/GoogleCloudPlatform/solutions-template).
{{< /alert >}}

{{< alert title="Note">}}
Aside from `Docker` and `kubectl`, Skaffold also supports a variety of other tools
and workflows; see [Tutorials]({{<relref "/docs/tutorials">}}) for
more information.
{{</alert>}}

In this quickstart, you will:

* Use **skaffold init** to bootstrap your Skaffold config.
* Use **skaffold dev** to automatically build and deploy your application when your code changes.
* Use **skaffold build** and **skaffold test** to tag, push, and test your container images.
* Use **skaffold render** and **skaffold apply** to generate and deploy Kubernetes manifests as part of a GitOps workflow.

## Set up

### Install Skaffold, minikube, and kubectl

This tutorial requires Skaffold, minikube, and kubectl.

1. [Install Skaffold]({{< relref "/docs/install" >}}).
1. [Install kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/).
1. [Install minikube](https://minikube.sigs.k8s.io/docs/start/).

This tutorial uses minikube because Skaffold knows how to build the app using the Docker daemon hosted
inside minikube. This means we don't need a registry to host the app's container images.

### Clone the sample app

Let's get a sample application set up to use Skaffold.

1. Clone the Skaffold repository:

    ```bash
    git clone https://github.com/GoogleContainerTools/skaffold
    ```

1. Change to the `examples/buildpacks-node-tutorial` directory.

    ```bash
    cd skaffold/examples/buildpacks-node-tutorial
    ```

## Initialize Skaffold

Your working directory is the application directory, `skaffold/examples/buildpacks-node-tutorial`. This will be our root Skaffold directory.

This sample application is written in Node, but Skaffold is language-agnostic and works with any containerized application.

### Bootstrap Skaffold configuration

1. Run the following command to generate a `skaffold.yaml` config file:


    ```bash
    skaffold init
    ```

1. When prompted to choose the builder, press enter to accept the default selection.

1. When asked which builders you would like to create Kubernetes resources for, press enter to accept the default selection.

1. When asked if you want to write this configuration to skaffold.yaml, type "y" for yes.

1. Open your new **skaffold.yaml**, generated at `skaffold/examples/buildpacks-node-tutorial/skaffold.yaml`. All of your Skaffold configuration lives in this file. We will go into more detail about how it works in later steps.

## Use Skaffold for continuous development

Skaffold speeds up your development loop by automatically building and deploying the application whenever your code changes.

### Start minikube

1. To see this in action, let's start up minikube so Skaffold has a cluster to run your application.

    ```bash
    minikube start --profile custom
    skaffold config set --global local-cluster true
    eval $(minikube -p custom docker-env)
    ```

This may take several minutes.

### Use `skaffold dev`


1. Run the following command to begin using Skaffold for continuous development:

    ```bash
    skaffold dev
    ```

    Notice how Skaffold automatically builds and deploys your application. You should see the following application output in your terminal:

    ```terminal
    Example app listening on port 3000!
    ```

    Skaffold is now watching for any file changes, and will rebuild your application automatically. Let's see this in action.


1. Open `skaffold/examples/buildpacks-node-tutorial/src/index.js` and change line 10 to the following:

    ```
    app.listen(port, () => console.log(`Example app listening on port ${port}! This is version 2.`))
    ```

    Notice how Skaffold automatically hot reloads your code changes to your application running in minikube, intelligently syncing only the file you changed. Your application is now automatically deployed with the changes you made, as it prints the following to your terminal:

    ```terminal
    Example app listening on port 3000! This is version 2.
    ```

### Exit dev mode

1. Let's stop continuous dev mode by pressing the following keys in your terminal:

    ```terminal
    Ctrl+C
    ```

    Skaffold will clean up all deployed artifacts and end dev mode.

## Use Skaffold for continuous integration

While Skaffold shines for continuous development, it can also be used for continuous integration (CI). Let's use Skaffold to build and test a container image.

### Build an image

Your CI pipelines can run `skaffold build` to build, tag, and push your container images to a registry. 

1. Try this out by running the following command:

    ```bash
    export STATE=$(git rev-list -1 HEAD --abbrev-commit)
    skaffold build --file-output build-$STATE.json
    ```

    Skaffold writes the output of the build to a JSON file, which we'll pass to our continuous delivery (CD) process in the next step.

### Test an image

Skaffold can also run tests against your images before deploying them.  Let's try this out by creating a simple custom test.

1. Open your<walkthrough-editor-open-file filePath="cloudshell_open/skaffold/examples/buildpacks-node-tutorial/skaffold.yaml">`skaffold.yaml`</walkthrough-editor-open-file> and add the following test configuration to the bottom, without any additional indentation:

    ```
    test:
    - image: skaffold-buildpacks-node
      custom:
        - command: echo This is a custom test commmand!
    ```

    Now you have a simple custom test set up that will run a bash command and await a successful response.

1. Run the following command to execute this test with Skaffold:

    ```bash
    skaffold test --build-artifacts build-$STATE.json
    ```

## Use Skaffold for continuous delivery

Let's learn how Skaffold can handle continuous delivery (CD).

### Deploy in a single step

1. For simple deployments, run `skaffold deploy`:

    ```bash
    skaffold deploy -a build-$STATE.json
    ```

    Skaffold hydrates your Kubernetes manifest with the image you built and tagged in the previous step, and deploys the application.

### Render and apply in separate steps

For GitOps delivery workflows, you may want to decompose your deployments into separate render and apply phases. That way, you can commit your hydrated Kubernetes manifests to source control before they are applied.

1. Run the following command to render a hydrated manifest:

    ```bash
    skaffold render -a build-$STATE.json --output render.yaml --digest-source local
    ```

    Open `skaffold/examples/buildpacks-node-tutorial/render.yaml` to check out the hydrated manifest.


1. Next, run the following command to apply your hydrated manifest:

    ```bash
    skaffold apply render.yaml
    ```

You have now successfully deployed your application in two ways.

## Congratulations, you successfully deployed with Skaffold!

You have learned how to use Skaffold for continuous development, integration, and delivery.

{{% /tab %}}

{{% tab "CLOUD CODE" %}}

Follow these quickstart guides if you're using Skaffold with the [Cloud Code]({{< relref "../install/#managed-ide" >}}) IDE extensions:

### [Cloud Code for VSCode](https://cloud.google.com/code/docs/vscode/quickstart-k8s)

Create, locally develop, debug, and run a Kubernetes application with Cloud Code for VSCode.

<a href="https://cloud.google.com/code/docs/vscode/quickstart-k8s">![vscode](/images/cloud-code-quick-deploy.gif)</a>

<br />

### [Cloud Code for IntelliJ](https://cloud.google.com/code/docs/intellij/quickstart-k8s)

Create, locally develop, debug, and run a Kubernetes application with Cloud Code for IntelliJ.

<a href="https://cloud.google.com/code/docs/intellij/quickstart-k8s">![intellij](/images/intellij-quickstart-runthrough.gif)</a>

{{% /tab %}}
{{% tab "CLOUD SHELL" %}}

Skip any setup by using Google Cloud Platform's [_Cloud Shell_](http://cloud.google.com/shell),
which provides a [browser-based terminal/CLI and editor](https://cloud.google.com/shell#product-demo).
Cloud Shell comes with Skaffold, Minikube, and Docker pre-installed, and is free
(requires a [Google Account](https://accounts.google.com/SignUp)).

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?show=ide%2Cterminal&cloudshell_git_repo=https://github.com/GoogleContainerTools/skaffold&walkthrough_id=skaffold--skaffold_onboarding&cloudshell_workspace=/examples/buildpacks-node-tutorial&cloudshell_open_in_editor=src/index.js)

{{% /tab %}}
{{% /tabs %}}

## What's next

For getting started with your project, see the [Getting Started With Your Project]({{<relref "/docs/workflows/getting-started-with-your-project" >}}) workflow.

For more in-depth topics of Skaffold, explore [Configuration]({{< relref "/docs/design/config.md" >}}),
[Skaffold Pipeline]({{<relref "/docs/pipeline-stages" >}}), and [Architecture and Design]({{< relref "/docs/design" >}}).

To learn more about how Skaffold builds, tags, and deploys your app, see the How-to Guides on
using [Builders]({{<relref "/docs/pipeline-stages/builders" >}}), [Taggers]({{< relref "/docs/pipeline-stages/taggers">}}), and [Deployers]({{< relref "/docs/pipeline-stages/deployers" >}}).

[Skaffold Tutorials]({{< relref "/docs/tutorials" >}}) details some of the common use cases of Skaffold.

Questions?  See our [Community section]({{< relref "/docs/resources#Community" >}}) for ways to get in touch.

:mega: **Please fill out our [quick 5-question survey](https://forms.gle/BMTbGQXLWSdn7vEs6)** to tell us how satisfied you are with Skaffold, and what improvements we should make. Thank you! :dancers:
