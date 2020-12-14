---
title: "Using Skaffold for CI/CD with Gitlab"
linkTitle: "Skaffold in CI/CD"
weight: 100
---
Contributed by: [@tvvignesh](https://github.com/tvvignesh)

Skaffold is a tool which is non-opinionated about the CI/CD tool you should use and acts as a platform agnostic solution for both development and deploying to production.

To facilitate building and deploying images in CI/CD, skaffold offers docker images which you can build on top of. If that doesn’t work for your use-case, you can make your own Dockerfile and pull in skaffold and its dependencies using curl.

Let us have a look at how we can use Skaffold with Gitlab CI.

## Step 1: Getting the project and Dockerfile ready

The first step is to obviously have your project which you want to deploy ready with a Dockerfile setup for the same. 

Skaffold supports multiple builders like Docker, Kaniko, Bazel and more and if the builder you are looking for is not supported out of the box, you can [script it out](https://skaffold.dev/docs/tutorials/custom-builder/) as well. We will use Docker in this demonstration.

For instance, if you would like to deploy nginx, your Dockerfile would look something like this:


```Dockerfile
FROM nginxinc/nginx-unprivileged:1.19.3-alpine
WORKDIR /app/server

COPY ./web ./web

EXPOSE 8080

CMD [ "/bin/bash", "-c", "nginx -g 'daemon off;'" ]
```


Skaffold can use your dockerfile to automatically build, tag and push images to the registry when needed.

## Step 2: Choosing the deployment method

Skaffold supports deployment through kubectl, helm and a lot of other mechanisms. You can have a look at the complete list of deployers available [here](https://skaffold.dev/docs/pipeline-stages/deployers/).

If you choose to deploy through kubectl, you must have all your yaml files ready, or if you would like to deploy through helm charts, you must have all your charts ready.

## Step 3: Choosing the image and tag strategy

Skaffold automatically tags images based on different strategies as documented here: [https://skaffold.dev/docs/pipeline-stages/taggers/](https://skaffold.dev/docs/pipeline-stages/taggers/). The tagging strategy used is configurable, so choose the mechanism which is right for you. By default, skaffold uses the git sha tagger.

If you are using helm as your deployer, you might want to use the helm image strategy if you would like to follow helm specific conventions, as skaffold will pass in all the details for you.

## Step 4: Plugins (optional)

If you want to do some kind of processing before deployment like decrypting the secrets in the CI/CD pipeline, skaffold also supports plugins like [Helm Secrets](https://github.com/zendesk/helm-secrets/). So, if you would like to deploy secrets that have been encrypted using KMS or any other mechanism supported by [SOPS](https://github.com/mozilla/sops](https://github.com/mozilla/sops) you can actually set **useHelmSecrets** option to true and skaffold will handle everything automatically for you.

## Step 5: The `skaffold.yaml` file

Your `skaffold.yaml` file will change depending on the way you would like to build and deploy your project. It is recommended that you set up skaffold and run it manually before you setup your CI pipeline since the CI pipeline can just call skaffold to do all the builds and deployments which can be tested out locally via skaffold.

A sample skaffold file can look something like this:


```yml
apiVersion: skaffold/v2beta8
kind: Config
profiles:
  # Specify the profile name which you can run later using skaffold run -p <profilename>
  - name: my-profile-name
    build:
      artifacts:
        # Skaffold will use this as your image name and push it here after building
        - image: asia.gcr.io/my-project/my-image
          # We are using Docker as our builder here
          docker:
            # Pass the args we want to Docker during build
            buildArgs:
              NPM_REGISTRY: '{{.NPM_REGISTRY}}'
              NPM_TOKEN: '{{.NPM_TOKEN}}'
    deploy:
      # Using Helm as the deployment strategy
      helm:
        # Pass the parameters according to https://skaffold.dev/docs/references/yaml/
        releases:
          - name: my-release
            namespace: default
            # Using Helm secrets plugin to process secrets before deploying
            useHelmSecrets: true
            # Location of the chart - here, we use a local chart
            chartPath: ./charts/my-chart
            # Path to the values file
            valuesFiles:
              - ./deployment/dev/values.yaml
              - ./deployment/dev/secrets.yaml
            skipBuildDependencies: true
            artifactOverrides:
              image: asia.gcr.io/my-project/my-image
            imageStrategy:
              helm: {}
        flags:
          upgrade:
            - --install
```

Please have a look at the comments above to understand what the yaml does.
  
Rather, if you want to deploy via kubectl, your skaffold file can look something like this:


```yml
apiVersion: skaffold/v2beta8
kind: Config
profiles:
    # Specify the profile name which you can run later using skaffold run -p <profilename>
    - name: dev-svc
      build:
          artifacts:
              # Skaffold will use this as your image name and push it here after building
              - image: asia.gcr.io/my-project/my-image
                # We are using Docker as our builder here
                docker:
                    # Pass the args you want to Docker during build
                    buildArgs:
                        # Pass the args we want to Docker during build
                        NPM_REGISTRY: '{{.NPM_REGISTRY}}'
                        NPM_TOKEN: '{{.NPM_TOKEN}}'
      deploy:
          # In case we enable status check, this will be the timeout till which skaffold will wait
          statusCheckDeadlineSeconds: 600
          # Using kubectl as our deployer
          kubectl:
              # Location to our yaml files
              # Refer https://skaffold.dev/docs/references/yaml/ for more options
              manifests:
                  - k8/dev/*.yml
```

## Step 6: Development & Testing locally

Before we move on to setting up the CI pipeline, we should test it out locally. Use `skaffold run` if you are deploying it to the cluster, `skaffold build` to just build and push the artifacts, `skaffold dev` if you are developing (this will enable auto reload on changes) or debug using `skaffold debug`

You can find docs about all these workflows [here](https://skaffold.dev/docs/workflows/)

Also, note that you can also set up file synchronization to enable faster development workflows. You can read more about that [here](https://skaffold.dev/docs/pipeline-stages/filesync/).

Enabling file synchronization would avoid the need to rebuild the images repeatedly during development and testing and this can accelerate the inner dev loop.

## Step 7: Setting up authentication with the registry

As you may already know, there are a lot of places where you can host images namely [Docker Hub](https://hub.docker.com/), [GCR](https://cloud.google.com/container-registry), [Gitlab Registry](https://docs.gitlab.com/ee/user/packages/container_registry/), [Quay](https://quay.io/) and so on.

The way you authenticate is specific to the registry provider of your choice. For eg. you may choose to do a `username-password` authentication with DockerHub or authenticate using service accounts in `GCR` and so on.

**NOTE:** Username-Password authentication is not recommended for CI/CD pipelines since they can be easily compromised and can sometimes be logged as well.

We will be using GCR in our example. You can refer [this page](https://cloud.google.com/container-registry/docs/advanced-authentication) to understand how to authenticate with it.

## Step 8: Using DIND Service (Optional)

Docker images are typically not cached if you are running your builds/pipeline within Docker containers(i.e. [DIND](https://docs.gitlab.com/ee/ci/docker/using_docker_build.html#use-docker-in-docker-workflow-with-docker-executor) or Docker-in-Docker) and this can be very slow since your CI runner has to download the images again and again for every build even if some layers are already available.

**NOTE:** If you are not looking to run your pipeline within Docker containers or are fine with slower pipelines/pipelines without caching, you can completely skip this step and proceed to Step 9.

To avoid this, we can set up a DIND service in our cluster to enable caching and storage of image layers for us. A sample dind deployment file as deployed in Kubernetes can look something like this:


```yml
# Setup a DIND deployment within the Kubernetes cluster to cache images as we build images in our pipelines
apiVersion: apps/v1
kind: Deployment
metadata:
    name: my-dind
spec:
    selector:
        matchLabels:
            app: my-dind
    strategy:
        type: Recreate
    template:
        metadata:
            labels:
                app: my-dind
        spec:
            containers:
                - image: docker:dind
                  imagePullPolicy: Always
                  name: my-dind
                  ports:
                      - containerPort: 2375
                        name: my-dind
                  env:
                      - name: DOCKER_HOST
                        value: tcp://localhost:2375
                      - name: DOCKER_TLS_CERTDIR
                        value: ''
                  securityContext:
                      privileged: true
                  volumeMounts:
                      - name: my-dind-storage
                        mountPath: /var/lib/docker
                  resources:
                      limits:
                          memory: '2Gi'
                          cpu: '1000m'
                      requests:
                          memory: '1Gi'
                          cpu: '250m'
            volumes:
                - name: my-dind-storage
                  persistentVolumeClaim:
                      claimName: my-dind-pv-claim
```


Service File:


```yml
# Expose DIND as a service so that all the pipelines can access it
apiVersion: v1
kind: Service
metadata:
    name: my-dind-svc
spec:
    selector:
        app: my-dind
    type: LoadBalancer
    ports:
        - port: 2375
          targetPort: 2375
          protocol: TCP
          name: http
```


**PV Claim:**


```yml
# Provision a Persistent Volume Claim to be used to store the cached artifacts by DIND
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
    name: my-dind-pv-claim
    labels:
        app: my-dind
spec:
    accessModes:
        - ReadWriteOnce
    resources:
        requests:
            storage: 200Gi
    storageClassName: standard
```


And we would need to pass the URL to the DIND service when we do builds in the pipeline and all the caching will happen in the DIND storage speeding up the pipeline a lot. Make sure that you clean up the DIND storage periodically or setup cleanup scripts for the same else it might fill up soon.

## Step 9: Getting the CI pipeline setup

Now that we have Skaffold ready and working locally, we look at the CI pipeline.

Here is a sample pipeline using Gitlab CI and if you are new to it, you can refer the docs [here](https://docs.gitlab.com/ee/ci/quick_start/). Also, you might want to look at how to define Gitlab CI/CD environment variables [here](https://docs.gitlab.com/ee/ci/variables/)

This pipeline might look complicated at first glance, but most of it related to configuring access for deploying to GCP and configure Gitlab's Docker-in-Docker support to cache artifacts between builds. What is important to note is that we use Skaffold to do all the builds and deployments.


```yml
stages:
    # The name of the pipeline stage
    - my-stage

# The name of the job
my-job:
    # The image to be used as the base for this job
    image:
        name: gcr.io/k8s-skaffold/skaffold:v1.15.0
    # Pipeline tags (if you have specified tags to your Gitlab Runner)
    tags:
        - development
    stage: my-stage
    retry: 2
    script:
          # Logging in to our gcp account using the service account key (specified in GITLAB variables)
        - echo "$GCP_DEV_SERVICE_KEY" > gcloud-service-key.json
        - gcloud auth activate-service-account --key-file gcloud-service-key.json
          # Specifying the project, zone, cluster to deploy our application in.
        - gcloud config set project $PROJECT_DEV_NAME
        - gcloud config set compute/zone $PROJECT_DEV_REGION
        - gcloud container clusters get-credentials $CLUSTER_DEV_NAME
        - kubectl config get-contexts
          # Pass in all the environment variables to be passed to skaffold during build and deployment with all the args as documented in https://skaffold.dev/docs/references/cli/ (in our case our cluster context, env vars, namespace and some labels)
        - DOCKER_HOST="$DIND_DEV" NPM_REGISTRY="$NPM_REGISTRY_DEV" NPM_TOKEN="$NPM_TOKEN_DEV" skaffold run --kube-context $CLUSTER_DEV_CONTEXT -n default -l skaffold.dev/run-id=deploydep -p dev-svc --status-check
    only:
        # Run this only for changes in the master branch
        refs:
            - master
```

So, ultimately our Gitlab CI pipeline does not know anything about the helm charts or build strategy or any deployment methods and relies completely on skaffold to do the job for us.

This works equally well irrespective of whether you choose the [Docker Executor](https://docs.gitlab.com/runner/executors/docker.html) or the [Kubernetes Executor](https://docs.gitlab.com/runner/executors/kubernetes.html) and would require very little changes between the two.

While Gitlab supports many executors (including Docker, Kubernetes, etc.) as documented [here](https://docs.gitlab.com/runner/executors/), we are working with Kubernetes executor here since it is vendor-agnostic, has the ability to auto-scale up/down, private (if you deploy the Runner in your Kubernetes cluster) and also is well supported for the forseeable future.

**NOTE:** If you are running Skaffold using the Kubernetes executor, make sure that you are running the runner with appropriate permissions or pod security policies. If you are running using DIND, it requires access to the Docker socket and being in privileged mode which might actually tend to be insecure. An alternative can be to use [kaniko](https://github.com/GoogleContainerTools/kaniko), [buildkit](https://github.com/moby/buildkit) or other custom builders like [Buildah](https://github.com/containers/buildah) for building the image.

## Step 10: That’s all folks

If all went well so far, you can try pushing a commit and see your pipeline running, picking up the changes, building and pushing the image and also deploying the same. 

Hope this post was informative. Good luck with your journey with Skaffold.
