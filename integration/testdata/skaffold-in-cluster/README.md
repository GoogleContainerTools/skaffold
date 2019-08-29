# Skaffold in-cluster test data 

Skaffold aims to be used as a buildstep in CI/CD pipelines. 
What if this pipeline step is running inside a K8s cluster?
 
In that case Skaffold needs to be able to work with an in-cluster k8s context to setup the secret and to create the pod for a kaniko build. 
This test case is testing that flow.
 
The `skaffold.yaml` describes _both_ the creation of an imaginary buildstep.
The buildstep is implemented with a k8s Job under `build-step` and an image,
 `gcr.io/k8s-skaffold/skaffold-in-cluster-builder` that contains the freshly built version of skaffold and kubectl.

The build target that the buildstep is building using kaniko is a simple `Dockerfile` under `test-build`.

The flow of the integration test is thus: 

`buildtest -> skaffold run -p create-build-step -> creates job -> creates pod -> skaffold build -p build-step -> kicks off kaniko pod to build test-build` 

Thus at the end we should have a successfully completed job.
