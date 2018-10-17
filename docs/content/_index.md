---
title: "Skaffold Documentation: Overview"
date: 2018-09-04T00:00:00-07:00
type: index
draft: false
---

Skaffold is a command line tool that facilitates continuous development for
Kubernetes applications. You can iterate on your application source code
locally then deploy to local or remote Kubernetes clusters. Skaffold handles
the workflow for building, pushing and deploying your application. It can also
provide building blocks and describe customizations for a CI/CD pipeline.

## Features

* Fast local Kubernetes Development
  * **optimized source-to-k8s** - Skaffold detects changes in your source code and handles the pipeline to
  **build**, **push**, and **deploy** your application automatically with **policy based image tagging** and **highly optimized, fast local workflows**
  * **continuous feedback** - Skaffold automatically manages logging and port-forwarding   
* Skaffold projects work everywhere
  * **share with other developers** - Skaffold is the easiest way to **share your project** with the world: `git clone` and `skaffold run`
  * **context aware** - use Skaffold profiles, user level config, environment variables and flags to describe differences in environments
  * **CI/CD building blocks** - use `skaffold run` end-to-end or just part of skaffold stages from build to deployment in your CI/CD system 
* skaffold.yaml - a single pluggable, declarative configuration for your project  
  * **skaffold init** - Skaffold discovers your files and generates its own config file
  * **multi-component apps** - Skaffold supports applications consisting of multiple components 
  * **bring your own tools** - Skaffold has a pluggable architecture to allow for different implementations of the stages
* Lightweight 
  * **client-side only** - Skaffold does not require maintaining a cluster-side component, so there is no overhead or maintenance burden to
  your cluster.
  * **minimal pipeline** - Skaffold provides an opinionated, minimal pipeline to keep things simple  

## A Glance at Skaffold Workflow and Architecture

Skaffold simplies your development workflow by organizing common development
stages into one simple command. Every time you run `skaffold dev`, the system

1. Collects and watches your source code for changes
2. Builds artifacts from the source code
3. Tags the artifacts
4. Pushes the artifacts
5. Deploys the artifacts
6. Monitors the deployed artifacts
7. Cleans up deployed artifacts on exit (Ctrl+C) 

{{< note >}}
**Note** 

Skaffold also supports skipping stages if you want to. 
{{< /note >}}
   
What's more, the pluggable architecture is central to Skaffold's design, allowing you to use
the tool you prefer in each stage. Also, skaffold's `profiles` feature grants
you the freedom to switch tools as you see fit depending on the context. 

For example, if you are coding on a local machine, you can configure Skaffold to build artifacts
with local Docker daemon and deploy them to minikube
using `kubectl`, the Kubernetes command-line interface and when you finalize your
design, you can switch to the production profile and start building with
Google Cloud Build and deploy with Helm.

At this moment, Skaffold supports the following tools:

{{% tabs %}}
{{% tab "IMAGE BUILDERS" %}}
* Dockerfile to Local Docker Daemon
* Dockerfile to Registry using Kaniko
* Dockerfile to Registry using Google Cloud Build
* Bazel to Local Docker Daemon or Registry 
{{% /tab %}}

{{% tab "DEPLOYERS" %}}
* Kubernetes Command-Line Interface (`kubectl`)
* Helm
* Kustomize
{{% /tab %}}

{{% tab "TAG POLICIES" %}}
* tag by git commit
* tag by current date&time 
* tag by environment variables based template
* tag by checksum of the source code
{{% /tab %}}

{{% tab "PUSH STRATEGIES" %}}
* don't push - keep the image on the local daemon
* push to registry 
{{% /tab %}} 
{{% /tabs %}}


![architecture](/images/architecture.png)


Besides these skaffold also automatically manages the following utilities for you: 

* forward container ports to your local machine using `kubectl port-forward`
* aggregate all the logs from the deployed pods
