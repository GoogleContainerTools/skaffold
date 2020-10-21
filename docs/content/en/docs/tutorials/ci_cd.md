**Skaffold End to End**

Skaffold is non-opinionated about the CI tool you should use and acts as a platform agnostic solution for both development and deploying to production.

To facilitate building and deploying images in CI/CD, skaffold offers docker images which you can build on top of. If that doesn’t work, you can make your own Dockerfile and pull in skaffold and its dependencies using curl.

Let us have a look at how we can use Skaffold with Gitlab CI.

## Step 1: Getting the project and Dockerfile ready

The first step is to obviously have your project which you want to deploy ready with a Dockerfile setup for the same. Skaffold supports multiple builders like Docker, Kaniko, Bazel and more (including custom builders). We will use Docker in this demonstration.

For instance, if you would like to deploy nginx, your Dockerfile would look something like this:


```
FROM nginxinc/nginx-unprivileged:1.19.3-alpine
WORKDIR /app/server

COPY ./web ./web

EXPOSE 8080

CMD [ "/bin/bash", "-c", "nginx -g 'daemon off;'" ]
```


Skaffold can use your dockerfile to automatically build, tag and push images to the registry when needed.

## Step 2: Choosing the deployment method

Skaffold supports deployment through kubectl, helm and a lot of other mechanisms. You can have a look at the yaml reference for complete list here: [https://skaffold.dev/docs/references/yaml/](https://skaffold.dev/docs/references/yaml/)

If you choose to deploy through kubectl, you must have all your yaml files ready, and if you would like to deploy through helm charts, you must have all your charts ready.

## Step 3: Choosing the image and tag strategy

Skaffold automatically tags images based on different strategies as documented here: [https://skaffold.dev/docs/pipeline-stages/taggers/](https://skaffold.dev/docs/pipeline-stages/taggers/). The tagging strategy used is configurable, so choose the mechanism which is right for you. By default, skaffold uses the git sha tagger.

If you are using helm as your deployer, you might want to use the helm image strategy if you would like to follow helm specific conventions, as skaffold will pass in all the details for you.

## Step 4: Plugins (optional)

Skaffold also supports plugins like Helm Secrets ([https://github.com/zendesk/helm-secrets/](https://github.com/zendesk/helm-secrets/)). So, if you would like to deploy secrets that have been encrypted using KMS or any other mechanism supported by SOPS ([https://github.com/mozilla/sops](https://github.com/mozilla/sops)) you can actually set **useHelmSecrets** option to true and skaffold will handle everything automatically for you.

## Step 5: The skaffold.yaml file

Your skaffold.yaml file will change depending on the way you would like to build and deploy your project. It is recommended that you set up skaffold before you setup your CI pipeline since the CI pipeline can just call skaffold to do all the builds and deployments, and you can do the same locally as well.

A sample skaffold file can look something like this:


```
apiVersion: skaffold/v2beta8
kind: Config
profiles:
  - name: my-profile-name
    build:
      artifacts:
        - image: asia.gcr.io/my-project/my-image
          docker:
            buildArgs:
              NPM_REGISTRY: '{{.NPM_REGISTRY}}'
              NPM_TOKEN: '{{.NPM_TOKEN}}'
    deploy:
      helm:
        releases:
          - name: my-release
            namespace: default
            useHelmSecrets: true
            chartPath: ./charts/my-chart
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


Here, we are



*   **Creating a profile** - You can set up multiple profiles using skaffold and deploy them separately if you want using **skaffold run -p &lt;profilename>**
  
*   **Specifying the artifacts to build from** - In this case, we are asking skaffold to build from a Dockerfile, passing its args and also specifying the destination where the image needs to be pushed. The values can also come from the env (NPM_REGISTRY and NPM_TOKEN in this example)

*   **Specifying the deployment strategy** - In this case, helm is specified with the location of the chart, all the configuration as documented in the yaml reference and also the values to be passed to it. We are also using the helm secrets plugin above.


If you want to deploy via kubectl, your skaffold file can look something like this:


```
apiVersion: skaffold/v2beta8
kind: Config
profiles:
    - name: dev-svc
      build:
          artifacts:
              - image: asia.gcr.io/my-project/my-image
                docker:
                    buildArgs:
                        NPM_REGISTRY: '{{.NPM_REGISTRY}}'
                        NPM_TOKEN: '{{.NPM_TOKEN}}'
      deploy:
          statusCheckDeadlineSeconds: 600
          kubectl:
              manifests:
                  - k8/dev/*.yml
```


**Please refer to the yaml reference for all that you can specify in skaffold.yaml file**


## Step 6: Development & Testing locally

Before we move on to setting up the CI pipeline, we should test it out locally. Use **skaffold run** if you are deploying it to the cluster, **skaffold build **to just build and push the artifacts, **skaffold dev **if you are developing (this will enable auto reload on changes) or debug using **skaffold debug**

You can find docs about all these workflows here: [https://skaffold.dev/docs/workflows/](https://skaffold.dev/docs/workflows/)

Also, note that you can also set up file synchronization to enable faster development workflows. You can read more about that here: [https://skaffold.dev/docs/pipeline-stages/filesync/](https://skaffold.dev/docs/pipeline-stages/filesync/)


## Step 7: Getting the CI pipeline setup

Now that we have skaffold ready and working locally, the CI pipeline requires little to no effort since it will just call skaffold to do all the builds and deployments.

Here is a sample pipeline using Gitlab CI


```
stages:
    - my-stage

my-job:
    image:
        name: gcr.io/k8s-skaffold/skaffold:v1.15.0
    tags:
        - development
    stage: my-stage
    retry: 2
    script:
        - docker login $REG_DEV_URL -u $REG_DEV_UNAME -p $REG_DEV_PASSWORD
        - echo "$GCP_DEV_SERVICE_KEY" > gcloud-service-key.json
        - gcloud auth activate-service-account --key-file gcloud-service-key.json
        - gcloud config set project $PROJECT_DEV_NAME
        - gcloud config set compute/zone $PROJECT_DEV_REGION
        - gcloud container clusters get-credentials $CLUSTER_DEV_NAME
        - kubectl config get-contexts
        - DOCKER_HOST="$DIND_DEV" NPM_REGISTRY="$NPM_REGISTRY_DEV" NPM_TOKEN="$NPM_TOKEN_DEV" skaffold run --kube-context $CLUSTER_DEV_CONTEXT -n default -l skaffold.dev/run-id=deploydep -p dev-svc --status-check
    only:
        refs:
            - master
```


Before we look at the pipeline, you might want to look at Gitlab CI/CD environment variables here: [https://docs.gitlab.com/ee/ci/variables/](https://docs.gitlab.com/ee/ci/variables/)

This is how we will configure the various environment variables to be passed in to the pipeline. Make sure that you mask/protect them using Gitlab so that they are not visible in the logs.

Here, we are:



*   Logging into the docker registry specifying the Registry URL, User ID and password (You can choose to authenticate using service tokens as well or other auth providers as well)
*   Logging in to our gcp account using the service account key
*   Specifying the project, zone, cluster to deploy our application in.
*   We are using DIND (Docker In Docker) to facilitate our builds (more about that below)
*   Then we just call the skaffold run in the pipeline passing flags we want to pass at runtime (in our case our cluster context, env vars, namespace and some labels)
*   And we trigger this only on changes to master branch

So, ultimately our Gitlab CI pipeline does not know anything about the helm charts or build strategy or any deployment methods and relies completely on skaffold to do the job for us.

This works equally well irrespective of whether you choose the Docker Executor ([https://docs.gitlab.com/runner/executors/docker.html](https://docs.gitlab.com/runner/executors/docker.html)) or the Kubernetes Executor ([https://docs.gitlab.com/runner/executors/kubernetes.html](https://docs.gitlab.com/runner/executors/kubernetes.html)) and would require little to no changes.

**NOTE:** If you are running Skaffold using Docker, make sure that you are running the runner with appropriate permissions or pod security policies since running DIND requires access to the Docker socket and being in privileged mode which might actually tend to be insecure. An alternative can be to use kaniko or custom builders like Buildah for building the image.


## Step 8: Using DIND Service (Optional)

Docker images are not cached if running using DIND and this can be very slow since it has to download the images again and again for every build even if some layers are already available.

To avoid this, we can set up a DIND service in our cluster to enable caching and storage of image layers for us. A sample dind deployment file can look something like this:


```
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


```
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


```
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


## Step 9: That’s all folks

Hope this post was informative. Good luck with your journey with Skaffold.
