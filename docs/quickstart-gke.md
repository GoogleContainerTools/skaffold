# Quick start for GKE

## Prerequisites

1. GCP Account. Sign up for [a free trial here](https://console.cloud.google.com/freetrial).

## Setup

1. [Open a Cloud Shell window](https://console.cloud.google.com/cloudshell)

1. Create a Kubernetes Engine cluster if you don't already have one.

    ```shell
    gcloud container clusters create skaffold --zone us-west1-a
    ```

1. Clone the Skaffold repository then change directories to the sample application.

    ```shell
    git clone https://github.com/GoogleCloudPlatform/skaffold.git
    cd skaffold/examples/getting-started
    ```
1. Install `skaffold`.

    ```shell
    curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-amd64
    chmod +x skaffold
    sudo mv skaffold /usr/local/bin
    ```

## Continuous development
The sample application you will use is a simple Go process that logs a statement every second.

As a new developer on-boarding you need to start Skaffold in `dev` mode to begin iterating
on the application and seeing the updates happen in real time. The development team working on the application
has already setup the Dockerfile, Kubernetes manifests, and Skaffold manifest necessary to get you started.

1. Change the references in `skaffold.yaml` to point to your Container Registry.

    ```shell
    sed -i s#k8s-skaffold#${GOOGLE_CLOUD_PROJECT}#g skaffold.yaml
    ```

1. Take a look at the contents of `skaffold.yaml`. You'll notice a profile named `gcb` that will be using Google Container Builder to build
   and push your image. The deploy section is configured to use kubectl to apply the Kubernetes manifests.
   
   ```shell
   cat skaffold.yaml
   ```

1. Run Skaffold in `dev` mode with the `gcb` profile enabled. This will use Container Builder to build a new image from the local source code,
   push it to your Container Registry and then deploy your application to your Kubernetes Engine cluster.

    ```shell
    skaffold dev -p gcb
    ```
1. You will see the application's logs printing to the screen.

    ```shell
    Starting deploy...
    Deploying k8s-pod.yaml...
    Deploy complete.
    [getting-started getting-started] Hello world!
    [getting-started getting-started] Hello world!
    [getting-started getting-started] Hello world!
    ```
 
1. Click the editor toggle button ![editor button](img/gcp-quickstart/cloud-shell-editor.png) in the top right of the Cloud Shell interface.
   The Cloud Shell editor is now open and displaying the contents of your Cloud Shell home directory.

1. Navigate to the `skaffold/examples/getting-started` directory in the left hand file navigation pane.

1. Click the `main.go` file to open it. 

1. Edit the `Hello World` message to say something different. Your change will be saved automatically by the editor.
   Once the save is complete Skaffold will detect that a file has been changed and then
   rebuild, repush and redeploy the change. You will see your new log line now streaming back from the Kubernetes cluster.
