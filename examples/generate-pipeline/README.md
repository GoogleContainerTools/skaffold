### Example: Getting started with skaffold and CI/CD using Tekton

This is a simple example to show users how to run the generate-pipeline command

_Please keep in mind that the generate-pipeline command is still a WIP_

Prerequisites:

* Install [tekton](https://github.com/tektoncd/pipeline/blob/master/docs/install.md) on your cluster
* Have [kaniko](https://github.com/GoogleContainerTools/kaniko) secrets setup
* Container registry must be public
* Give your default service account the cluster-admin role (necessary to have pipeline access secrets)

```shell
kubectl create clusterrolebinding serviceaccounts-cluster-admin \
--clusterrole=cluster-admin \
--user=system:serviceaccount:default:default
```

To generate and run a pipeline:

* Run skaffold generate-pipeline
* Modify skaffold.yaml to use a valid GCSbucket for kaniko
* Commit and push updated skaffold.yaml
* kubectl apply -f pipeline.yaml
* Create a pipelinerun.yaml
* kubectl apply -f pipelinerun.yaml
