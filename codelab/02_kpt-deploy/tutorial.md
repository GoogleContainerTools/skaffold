# Deploy Your Resource via `kpt` 

<walkthrough-disable-features toc></walkthrough-disable-features>

## Introduction


#### What is kpt?

Kpt is [an OSS tool](https://github.com/GoogleContainerTools/kpt) for Kubernetes packaging, which uses a standard format to bundle, publish, customize, update, and apply configuration manifests.

#### What you'll learn in this codelab
    
*   The difference between `kpt` pruning and `kubectl` pruning
*   How to enable kpt to deploy your configurations in their live state.
*   How to use kpt with kustomize.

This codelab is the second session. Check out the [01_kpt-validate](https://github.com/GoogleContainerTools/skaffold/tree/master/codelab/01_kpt-validate) about how to use kpt to validate your configuration before deploying to the cluster.

## Prerequisites

If you are new to skaffold, you can check out [the skaffold tutorials](https://skaffold.dev/docs/tutorials/) to get a basic idea. 
Or just follow this codelab, we will explain what happens in each step.

#### Install `skaffold`

*   Check if `skaffold` is installed and the installed version is >= v1.17.0
    ```bash
    skaffold version
    ```

*   If you haven't installed `skaffold` previously, run
    ```bash
    curl -Lo skaffold https://storage.googleapis.com/skaffold/builds/latest/skaffold-linux-amd64 && \
    sudo install skaffold /usr/local/bin/ | bash
    ```  
*   To upgrade skaffold to a newer version, run
    ```bash
    sudo rm -f /usr/local/bin/skaffold && curl -Lo skaffold https://storage.googleapis.com/skaffold/builds/latest/skaffold-linux-amd64 && \
    sudo install skaffold /usr/local/bin/ | bash
    ```
    
#### Install `kpt`

*   Check if `kpt` is installed and the installed version is >= 0.34.0
    ```bash
    kpt version
    ```
*   If you haven't installed `kpt` previously, run
    ```bash
    sudo apt-get install google-cloud-sdk-kpt
    ```  
    
#### Install `kustomize`

*   Check if `kustomize` is installed and the installed version is >= v3.4.3
    ```bash
    kustomize version
    ```
*   If you haven't installed `kustomize` previously, run
    ```bash
    curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"  | bash &&  sudo mv kustomize /usr/local/go/bin
    ```  
    
## Getting started

Time to complete: **About 3 minutes**


*   Run this command to start your local minikube cluster:
    
    ```bash
    minikube start
    ```
    
*   Once your cluster is set up, the following message is displayed:
    
    ```terminal
    Done! kubectl is now configured to use "minikube"
    ```
    
*   Now let's use `kpt pkg` to download the application example. 
        
    ```bash
    kpt pkg get https://github.com/GoogleContainerTools/skaffold.git/codelab/02_kpt-deploy/resources/sample-app guestbook-cl2 && cd guestbook-cl2
    ```

<walkthrough-pin-section-icon></walkthrough-pin-section-icon>
**Tips**
>  If you haven't used `skaffold dev`, you can go through this [codelab](https://github.com/GoogleContainerTools/skaffold/tree/master/codelab/01_kpt-validate).

## Deploy
Time to complete: **About 2 minutes**

`Skaffold` supports using `kpt` to deploy your configurations to the cluster.

This deployment is equivalent to [`kpt live`](https://googlecontainertools.github.io/kpt/reference/live/) which reconciles the resources in their live states. 

Compared to other deployment tools, one key advantage of the kpt deployment is the "pruning" feature. "Prune" means removing the resources from the cluster if the resources do not appear in the applied configuration. 

#### "prune"

You may have used the following command to remove resources that are not shown up in the `DIR`. However, the prune operation cannot always remove the desired resource.   

```terminal
kubectl apply --prune -f DIR
```

<walkthrough-pin-section-icon></walkthrough-pin-section-icon>
**Extended Reading**
> This [KEP](https://github.com/kubernetes/enhancements/pull/810) gives a full context 
> about the problems the `kubectl` "prune" may cause.

Next, you will compare the pruning result between `kpt` and `kubectl` to see why `kpt` is more reliable and accurate.

## `kubectl` pruning
Time to complete: **About 5 minutes**

*  To enable the `kubectl` pruning, you can configure the <walkthrough-editor-open-file filePath="guestbook-cl2/skaffold.yaml">skaffold.yaml</walkthrough-editor-open-file>, and replace the following code in the <walkthrough-editor-select-line filePath="guestbook-cl2/skaffold.yaml" startLine="8" endLine="10" startCharacterOffset="0" endCharacterOffset="100">deploy</walkthrough-editor-select-line> section.

    ```yaml
    deploy:
      kubectl:
        flags:
          apply:
          - "--prune=true"
          - "--all=true"
          - "--namespace=default"
        manifests:
        - config/frontend/*.yaml
    ```

*   Run `skaffold dev` to deploy the resources.
    
    ```bash
    skaffold dev 
    ```

*   Now, let's open a new <walkthrough-editor-spotlight spotlightId="menu-terminal-new-terminal">terminal</walkthrough-editor-spotlight> and run the following command in the terminal. This will move the <walkthrough-editor-open-file filePath="guestbook-cl2/config/frontend/deployment.yaml">deployment.yaml</walkthrough-editor-open-file> out of the <walkthrough-editor-select-line filePath="guestbook-cl2/skaffold.yaml" startLine="16" endLine="16" startCharacterOffset="5" endCharacterOffset="28">config/frontend</walkthrough-editor-select-line>, meaning the  `Deployment` resource *should be pruned*.

    ```text
    cd guestbook-cl2
    mv config/frontend/deployment.yaml .
    ```

*   Check if the Deployment resource is pruned. You can run the following command in the terminal opened from the previous step. 
    
    ```text
    kubectl get deployment
    ```

    You should expect to see the following information, meaning the *Deployment* resource is not pruned.
     
    ```terminal
    NAME       READY   UP-TO-DATE   AVAILABLE   AGE
    frontend   1/1     1            1           6m17s
    ```
<walkthrough-notification-menu-icon></walkthrough-notification-menu-icon>
**Note**
> Do not exit the `skaffold dev` in the first cloud 
> shell tab, otherwise the Deployment resource is cleaned up due to the exist. 


## `kpt` pruning

#### How `kpt` prunes the resource

kpt uses a [three-way merge strategy](https://pwittrock-kubectl.firebaseapp.com/pages/app_management/field_merge_semantics.html) 
to compare the resources' previous state, current state and current desired state. This allows kpt to make changes more wisely.

#### Let's try it out. 

*   First, you can exit `skaffold dev` with *Ctrl+ C*

*   You should add back the deployment.yaml from the previous step.

    ```bash
    mv deployment.yaml config/frontend/
    ```

*   In <walkthrough-editor-open-file filePath="guestbook-cl2/skaffold.yaml">skaffold.yaml</walkthrough-editor-open-file>, replace the <walkthrough-editor-select-line filePath="guestbook-cl2/skaffold.yaml" startLine="8" endLine="16" startCharacterOffset="0" endCharacterOffset="28">deploy</walkthrough-editor-select-line> with the following content. 

    ```yaml
    deploy:
      kpt:
        dir: config
    ```

    **Tips**
    > `kpt` enables pruning by default.

*   Run `skaffold dev` to deploy the resources.
    
    ```bash
    skaffold dev
    ```

*   Open a new <walkthrough-editor-spotlight spotlightId="menu-terminal-new-terminal">terminal</walkthrough-editor-spotlight>, remove the <walkthrough-editor-select-line filePath="guestbook-cl2/config/kustomization.yaml" startLine="1" endLine="1" startCharacterOffset="0" endCharacterOffset="26">deployment.yaml</walkthrough-editor-select-line> from the <walkthrough-editor-open-file filePath="guestbook-cl2/config/kustomization.yaml">kustomization.yaml</walkthrough-editor-open-file>.  

    *You can **either** manually delete the line **or** run the following command in
    the new terminal*

    ```text
    cd guestbook-cl2
    sed -i '2d' ./config/kustomization.yaml
    ```

    `kpt` is compatible with kustomize by default. Thus, removing the file reference from the kustomization.yaml resource will exclude deployment.yaml from the deployment.*

*   Check resource on the cluster side. You can run the following command in the terminal opened from the previous step. 

    ```text
    kubectl get deployment
    ```
    You should expect to see the following message.
    ```terminal
    No resources found in default namespace.
    ```

## Conclusion

<walkthrough-conclusion-trophy></walkthrough-conclusion-trophy>

Congratulations, you know how to use kpt in skaffold! You can explore other kpt features 
from the skaffold.yaml[ reference doc](https://skaffold.dev/docs/references/yaml/#deploy-kpt). 

You can also try out other kpt features like `kpt pkg` and `kpt cfg` from 
[the user guide](https://googlecontainertools.github.io/kpt/reference/). They will be supported 
in the skaffold soon. Stay tuned!  