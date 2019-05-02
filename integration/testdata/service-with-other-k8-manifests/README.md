This example is to test cases found in #1737, #2019

The hello-service used to test in this example lies [here](build-service)

This example is run as part of integration test suite [TestDeploy](../../deploy_test.go)

If you want to change the service code:
1. Make changes [here](build-service)
2. Run `skaffold build`
3. Run `docker push gcr.io/k8s-skaffold/hello-service:latest`.
