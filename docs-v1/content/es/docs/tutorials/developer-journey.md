---
title: "Developer Journey with Buildpacks"
linkTitle: "Developer Journey"
weight: 100
---

# Skaffold Developer Journey with Buildpacks Tutorial

## Introduction

### What is this project?

Skaffold allows developers to easily transition from local development on minikube to remote development on an enterprise Kubernetes cluster managed by IT. During the transition from local to remote deployment, a security team might ask a developer to patch a library with a specific vulnerability in it. This is where Skaffold's support for buildpacks comes in handy. In this tutorial, you'll start out deploying an application locally, swap out buildpacks in the **skaffold.yaml** file to use the latest libraries, and then deploy the application to a remote Kubernetes cluster.

For a guided Cloud Shell tutorial on how a developer's journey might look in adopting Skaffold, follow:

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/GoogleContainerTools/skaffold&cloudshell_workspace=examples/dev-journey-buildpacks&cloudshell_tutorial=tutorial.md)

### What you'll learn

* How to develop on minikube locally and move to an enterprise managed kubernetes cluster with ease
* How to easily switch buildpack versions to comply with security demands

___

**Time to complete**: <walkthrough-tutorial-duration duration=15></walkthrough-tutorial-duration>
Click the **Start** button to move to the next step.

## Prepare the Environment

### Create a Project
Create a project called **skaffold-tutorial** with a valid billing account in the Google Cloud [Manage Resources Console](https://console.cloud.google.com/cloud-resource-manager). In the shell run:
```bash
gcloud config set project skaffold-tutorial
```

### Run the Setup Script
Run the start script to prepare the environment. The script will:
* Enable the GCE and GKE APIs, which is needed in order to spin up a GKE cluster
* Create a network and subnet for the GKE cluster to use
* Deploy a one node GKE cluster to optimize for cost
* Install the latest version of skaffold. Although Skaffold is already installed on cloud shell, we'll install the latest version for good measure.
```bash
chmod +x start.sh && ./start.sh
```
**Note:** answer **y** when prompted to enable the GKE and GCE APIs

### Start a Minikube cluster

We'll use minikube for local kubernetes development. Minikube is a tool that optimizes kubernetes for local deployments, which makes it perfect for development and testing. Cloud Shell already has minikube installed, but you can install it yourself by running **gcloud components install minikube** or following [these instructions](https://minikube.sigs.k8s.io/docs/start/).

Run:
```bash
minikube start
```

## Run the App to Minikube Using Skaffold

### Deploy the App
Start skaffold in development mode which will constantly monitor the current directory for changes and kick off a new build and deploy whenever changes are detected.
```bash
skaffold dev
```

**Important:** note the software versions under the **DETECTING** phase of the buildpack output. These will be important later.

We will be working in three terminals during this tutorial. The current terminal you're in will be referred to as **Terminal A**. Open a second terminal, which we will call **Terminal B**. In order to connect to the application to make sure its working, you need to start the minikube load balancer by running:
```bash
minikube tunnel
```

Open a third terminal, which we will refer to as **Terminal C**. Find out the load balancer IP by recording the external IP as **EXTERNAL_IP** after running:
```bash
kubectl get service
```

Ensure the app is responding by running:
```bash
curl http://EXTERNAL_IP:8080
```

### Change the App and Trigger a Redeploy

Edit the part of the application responsible for returning the "Hello World!" message you saw previously:
```bash
vim src/main/java/hello/HelloController.java
```
Change the return line to **return "Hello, Skaffold!"**.

Switch back to the **Terminal A** and see that its rebuilding and redeploying the app.

Switch back to the **Terminal C** and run the following command to watch until only one pod is running:
```bash
watch kubectl get pods
```

Once there is only one pod running, meaning the latest pod is deployed, see that your app changes are live:
```bash
curl http://EXTERNAL_IP:8080
```

### Updating Buildpacks

Perhaps the best benefit of buildpacks is that it reduces how much work developers need to do to patch their applications if the security team highlights a library vulnerability. [Google Cloud buildpacks](https://cloud.google.com/blog/products/containers-kubernetes/google-cloud-now-supports-buildpacks) use a managed base Ubuntu 18.04 image that is regularly scanned for security vulnerabilities; any detected vulnerabilities are automatically patched. These patches are included in the latest revision of the builder. A builder contains one or more buildpacks supporting several languages. Our **skaffold.yaml** points to an older builder release, which uses older buildpack versions that may pull in vulnerable libraries. We will be updating the builder release to use the most up-to-date buildpacks.

Edit the **skaffold.yaml** file:
```bash
vim skaffold.yaml
```
Update the builder line to **gcr.io/buildpacks/builder:v1**, which will use the latest builder that has more up-to-date buildpacks.

Switch back to the **Terminal A** and see that its rebuilding and redeploying the app.

**IMPORTANT:** compare the software versions under the **DETECTING** phase of the buildpack output to the ones you saw before. The builder is now using newer buildpack versions.

Switch back to **Terminal C** and run the following command to watch until only one pod is running:
```bash
watch kubectl get pods
```

Once there is only one pod running, meaning the latest pod is deployed, see that your app changes are live:
```bash
curl http://EXTERNAL_IP:8080
```

## Deploy the App to Enterprise GKE Cluster Using Skaffold

### Switch kubectl Context to the Enterprise GKE Cluster

Switch your local kubectl context to the enterprise GKE cluster and get the latest credentials:
```bash
gcloud container clusters get-credentials $(gcloud config get-value project)-cluster --region us-central1-a
```

See that kubectl is now configured to use the remote kubernetes cluster instead of minikube (denoted by the asterisk)
```bash
kubectl config get-contexts
```

### Deploy the App

Attempt to deploy the app by running:
```bash
skaffold dev --default-repo=gcr.io/$(gcloud config get-value project)
```

Switch back to **Terminal C** and run the following command to watch until only one pod is running:
```bash
watch kubectl get pods
```

Run the following command. Once an external IP is assigned to the service, record it as **EXTERNAL_IP**:
```bash
watch kubectl get service
```

See that your app changes are now live on an Internet-accessible IP:
```bash
curl http://EXTERNAL_IP:8080
```

## Congratulations!

That's the end of the tutorial. You now know how to seamlessly transition between local kubernetes development and remote development on a kubernetes cluster managed by your enterprise IT team. Along the way you learned how to quickly patch your application libraries to comply with security standards.

I hope this tutorial was informative. Good luck on your journey with Skaffold!