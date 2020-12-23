# Validate Your Configurations in skaffold Workflow

This codelab introduces how to validate your app configurations before deploying them to the cluster. You are **not** required to know `kpt` or `skaffold` before starting the codelab. 

After this session, you will have obtained more powerful tools to validate and manage your configurations.

## What is kpt?

Kpt is [an OSS tool](https://github.com/GoogleContainerTools/kpt) for Kubernetes packaging, which uses a standard format to bundle, publish, customize, update, and apply configuration manifests.

## What kpt can help you with

-  You can validate each of your config changes **declaratively**.
-  You will get an hands-on off-the-shelf experience about the **GitOps** CI/CD workflow in skaffold.
-  You **won't** encounter **version conflict** if the config hydration (a.k.a kustomize) mismatch with the deployment tool (e.g. kubectl). 
-  You can prune your resources accurately with [a three-way merge strategy](https://kubectl.docs.kubernetes.io/pages/app_management/field_merge_semantics.html). 

## What you'll learn

-  How to add a validation to your config changes. 
-  How to define validation rules in the form of a declarative configuration.
-  How to use the validation in kustomized resources. 
-  How to reconcile your configuration changes with the live state


You can try out this interactive codelab using [Cloud Shell](https://cloud.google.com/shell). 
[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.png)](https://ssh.cloud.google.com/cloudshell/open?cloudshell_git_repo=https://github.com/GoogleContainerTools/skaffold&cloudshell_working_dir=codelab/01_kpt-validate&cloudshell_workspace=codelab/01_kpt-validate&cloudshell_tutorial=tutorial.md)

## What's next?

Try "02_kpt-deploy" to learn why you should use `kpt` to deploy your application in skaffold in lieu of `kubectl` or `kustomize`.

