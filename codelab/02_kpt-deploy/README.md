# Deploy Your Configurations Accurately

This codelab uses a real example to compare the `kpt` and `kubectl` in their `prune` feature. You will get a better understanding about why using `kpt live` to deploy your resources are more accurate and reliable.

## What is kpt?

Kpt is [an OSS tool](https://github.com/GoogleContainerTools/kpt) for Kubernetes packaging, which uses a standard format to bundle, publish, customize, update, and apply configuration manifests.

## What kpt can help you with

-  You can prune your resources accurately with [a three-way merge strategy](https://pwittrock-kubectl.firebaseapp.com/pages/app_management/field_merge_semantics.html). 

You can try out this interactive codelab using [Cloud Shell](https://cloud.google.com/shell).
[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.png)](https://ssh.cloud.google.com/cloudshell/open?cloudshell_git_repo=https://github.com/GoogleContainerTools/skaffold&cloudshell_working_dir=codelab/02_kpt-deploy&cloudshell_workspace=codelab/02_kpt-deploy&cloudshell_tutorial=tutorial.md)
