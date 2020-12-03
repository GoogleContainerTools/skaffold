# Validate Your Configurations in skaffold Workflow

<walkthrough-disable-features toc></walkthrough-disable-features>

## Introduction

#### What is kpt?

Kpt is [an OSS tool](https://github.com/GoogleContainerTools/kpt) for Kubernetes packaging, which uses a standard format to bundle, publish, customize, update, and apply configuration manifests.

#### What kpt can help you with
    
*  You will get an hands-on off-the-shelf experience about the **GitOps** CI/CD workflow in skaffold.
*  You can validate each of your config changes **declaratively**.
*  You **won't** encounter **version conflict** if the config hydration (a.k.a kustomize) mismatch with the deployment tool (e.g. kubectl). 
*  You can prune your resources accurately with [a three-way merge strategy](https://kubectl.docs.kubernetes.io/pages/app_management/field_merge_semantics.html). 

#### What you'll learn
    
*  How to add a validation to your config changes. 
*  How to define validation rules in the form of a declarative configuration.
*  How to use the validation in kustomized resources. 

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
    sudo install skaffold /usr/bin/ | bash
    ```  
*   To upgrade skaffold to a newer version, run
    ```bash
    sudo rm -f /usr/bin/skaffold && curl -Lo skaffold https://storage.googleapis.com/skaffold/builds/latest/skaffold-linux-amd64 && \
    sudo install skaffold /usr/bin/ | bash
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


*   In your
    <walkthrough-editor-spotlight spotlightId="menu-terminal-new-terminal">terminal</walkthrough-editor-spotlight>,
    run this command to start your local minikube cluster:
    
    ```bash
    minikube start
    ```
    
*   Once your cluster is set up, the following message is displayed:
    
    ```terminal
    Done! kubectl is now configured to use "minikube"
    ```
    
*   Now let's use `kpt pkg` to download the application example. 
        
    ```bash
    kpt pkg get https://github.com/GoogleContainerTools/skaffold.git/codelab/01_kpt-validate/resources/sample-app guestbook-cl && cd guestbook-cl
    ```

<walkthrough-pin-section-icon></walkthrough-pin-section-icon>
**Extended Reading**
>  `kpt pkg` downloads the resource from a remote repository, a branch or a subdirectory.
> It does not contain the git version control history but only the specific git reference, 
> so you can compose your own "package" from multiple repositories.

> Read more about the kpt package from [here](https://googlecontainertools.github.io/kpt/reference/pkg/)

## Taking a deeper look at the skaffold.yaml

Time to complete: **About 3 minutes**


The example has already configured the <walkthrough-editor-open-file filePath="guestbook-cl/skaffold.yaml">skaffold.yaml</walkthrough-editor-open-file> 
for you. 
    
```yaml
apiVersion: skaffold/v2beta8
kind: Config
metadata:
  name: kpt-cl
build:
  artifacts:
  - image: "frontend"
    context: php-redis
deploy:
  kpt:
    dir: config
```
    
This file contains two stages: `build` and `deploy`. 

- <walkthrough-editor-select-line filePath="guestbook-cl/skaffold.yaml" startLine="4" endLine="4" startCharacterOffset="0" endCharacterOffset="5">build</walkthrough-editor-select-line> defines the methods to **build** and **upload** the application images (by default, it uses docker)
- <walkthrough-editor-select-line filePath="guestbook-cl/skaffold.yaml" startLine="8" endLine="8" startCharacterOffset="0" endCharacterOffset="6">deploy</walkthrough-editor-select-line> defines the methods to **manage** the app configuration and **deploy** the application to the bundled cluster.
        
In the example, the configurations will build an image <walkthrough-editor-select-line filePath="guestbook-cl/skaffold.yaml" startLine="6" endLine="6" startCharacterOffset="12" endCharacterOffset="20">frontend</walkthrough-editor-select-line> and use `kpt` to *hydrate[1]* 
and deploy the applications to the cluster. 

The <walkthrough-editor-select-line filePath="guestbook-cl/skaffold.yaml" startLine="6" endLine="6" startCharacterOffset="12" endCharacterOffset="20">frontend</walkthrough-editor-select-line> source code is stored in 
<walkthrough-editor-open-file filePath="guestbook-cl/php-redis/guestbook.php">guestbook-cl/php-redis</walkthrough-editor-open-file> 
and its configurations are stored in <walkthrough-editor-open-file filePath="guestbook-cl/config/frontend/deployment.yaml">guestbook-cl/config</walkthrough-editor-open-file>.

<walkthrough-pin-section-icon></walkthrough-pin-section-icon>
**Glossary**

> [1] *`Hydrate`* means rendering a *kustomize* directory or a *kpt* package to a flatten 
> configuration, each of whose resources contains the full set of the object information.

## Running skaffold

Time to complete: **About 1 minutes**

```bash
skaffold dev
```
    
`skaffold dev` is the essential skaffold command. It builds the application and then deploys 
the applications to the bundled cluster. 

Once the deployment is complete, you can exit with `Ctrl+C`.

<walkthrough-notification-menu-icon></walkthrough-notification-menu-icon>
**Tips**

> `skaffold dev` can automatically detect file changes and kick off a re-deploy.
> So you don’t need to rerun the command if the file changes.

## Validating the config

Time to complete: **About 10 minutes**

Validating the configurations helps both the app development and devOps to be efficient in a 
fragile environment. 

This step uses a `kubeval` example to show how `kpt` functions[2] 
can validate the app configuration and makes the validation itself **as a declarative config.**

*   Download the kpt function resource
    
    ```bash
    kpt pkg get https://github.com/GoogleContainerTools/skaffold.git/codelab/01_kpt-validate/resources/validation-kubeval validation-kubeval
    ```
    
*   Now let's update the skaffold.yaml to use the new validator. Replace the following code with the 
`.deploy` section in the <walkthrough-editor-select-line filePath="guestbook-cl/skaffold.yaml" startLine="8" endLine="10" startCharacterOffset="0" endCharacterOffset="100">skaffold.yaml</walkthrough-editor-select-line>

    See the full [skaffold.yaml](https://github.com/yuwenma/sample-app/blob/kubeval/skaffold.yaml#L12) 
    ```yaml
    deploy:
      kpt:
        dir: config
        fn:
          fnPath: validation-kubeval
          network: true
    ```

-  <walkthrough-editor-select-line filePath="guestbook-cl/skaffold.yaml" startLine="12" endLine="12" startCharacterOffset="6" endCharacterOffset="12">.deploy.kpt.fn.fnPath</walkthrough-editor-select-line> refers to the kpt function directory we just downloaded. 
-  <walkthrough-editor-select-line filePath="guestbook-cl/skaffold.yaml" startLine="13" endLine="13" startCharacterOffset="6" endCharacterOffset="13">.deploy.kpt.fn.network</walkthrough-editor-select-line> enables the kpt access to the network. This is required to run the function in a docker container.


<walkthrough-pin-section-icon></walkthrough-pin-section-icon>
**Glossary**
> [2] `kpt function` is a kubernetes resource with a `config.kubernetes.io/function` annotation. 
Read more [here](https://googlecontainertools.github.io/kpt/concepts/functions/)


## Verifying the validation 

### "happy path" 

Let's run `skaffold dev`. Now it will validate the resource according to the functions in 
<walkthrough-editor-select-line filePath="guestbook-cl/skaffold.yaml" startLine="14" endLine="14" startCharacterOffset="7" endCharacterOffset="13">`fnPath`, 
and *then* deploy the resource to the cluster.

```bash
skaffold dev
```


### "sad path"

Since the validation passes silently, let's break the kubeval to prove the validation works! 

* In <walkthrough-editor-open-file filePath="guestbook-cl/config/frontend/deployment.yaml">config/frontend/deployment.yaml</walkthrough-editor-open-file>, 
    remove the <walkthrough-editor-select-line filePath="guestbook-cl/config/frontend/deployment.yaml" startLine="18" endLine="18" startCharacterOffset="0" endCharacterOffset="30">spec.template.spec.containers.image</walkthrough-editor-select-line> field (in line 19)
   
    *You can **either** manually delete the line **or** run the following command*

    ```bash
    sed -i '19d' ./config/frontend/deployment.yaml
    ```

*   Check the skaffold dev output.
    
    ```bash
    skaffold dev
    ```
    You should expect the following warning about the missing field.
    
    ```terminal
    The Deployment "frontend" is invalid: spec.template.spec.containers[0].image: Required value
    ```

*   You can add back the removed line by copying the following command and running it in the terminal.  

    ```shell script
    sed -i '18 a \        image: "frontend"'     ./config/frontend/deployment.yaml
    ```
    
<walkthrough-notification-menu-icon></walkthrough-notification-menu-icon>
**Tips**
> You can find all the `kpt` validation functions from this [catalog](https://googlecontainertools.github.io/kpt/guides/consumer/function/catalog/validators/)
> 
> Or write your own versions. See [instructions](https://googlecontainertools.github.io/kpt/guides/consumer/function/).

## Conclusion

<walkthrough-conclusion-trophy></walkthrough-conclusion-trophy>

Congratulations, you known how to validate configurations in skaffold! You can explore the full kpt configuration 
from the skaffold.yaml [reference doc](https://skaffold.dev/docs/references/yaml/#deploy-kpt). 


You can also try out other kpt features like `kpt pkg` and `kpt cfg` from 
[the user guide](https://googlecontainertools.github.io/kpt/reference/). They will be supported 
in skaffold soon. Stay tuned!  
