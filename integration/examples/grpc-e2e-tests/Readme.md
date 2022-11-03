
### Example: E2E environment and tests with Skaffold

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/GoogleContainerTools/skaffold&cloudshell_open_in_editor=README.md&cloudshell_workspace=examples/grpc-e2e-tests)

In this example:

* Start a GRPC service(Visitor counter)
* Start a cloud spanner emulator and initialize database/tables using startup script
* Test locally
* Execute end to end tests

In the real world, Kubernetes deployments will consist of an application and a database for persisting any state. And write an E2E test which executes against local E2E environment.

#### Running the example on minikube

From this directory, run

```bash
skaffold run
```

Hit the service using [grpcurl](https://github.com/fullstorydev/grpcurl) command

```bash

# Visit request
$ grpcurl -plaintext -d '{"visitor": {"name": "testuser"}}'  $(minikube service visitor-counter --url | sed 's~http[s]*://~~g')  skaffold.examples.e2e.visitor.VisitorCounter/UpdateVisitor


# Get visit count for an user'
grpcurl -plaintext -d '{"visitor": {"name": "testuser"}}'  $(minikube service visitor-counter --url | sed 's~http[s]*://~~g')  skaffold.examples.e2e.visitor.VisitorCounter/GetVisitCount
```

Run [Ginkgo](https://onsi.github.io/ginkgo/) E2E tests using command 
```bash
export VISITOR_COUNTER_SERVICE=$(minikube service visitor-counter --url) & ginkgo tests

```
