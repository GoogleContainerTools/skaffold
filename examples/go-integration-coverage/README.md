# skaffold-go-integration-coverage

Example showing how to use Skaffold and ko to collect coverage profiles from
[integration tests](https://go.dev/testing/coverage/#glos-integration-test),
often called end-to-end tests, for Kubernetes workloads written in Go.

For a detailed explanation of how this example works, see the tutorial
[Go integration test coverage profiles](https://skaffold.dev/docs/tutorials/go-integration-coverage/)
on the Skaffold website.

## Requirements

- Go v1.20 or later
- Skaffold v2 or later

## Usage

1.  If you are using a remote Kubernetes cluster, configure Skaffold to use
    your image registry:

    ```shell
    export SKAFFOLD_DEFAULT_REPO=[your image registry, e.g., gcr.io/$PROJECT_ID]
    ```

    You can skip this step if you are using a local Kubernetes cluster such as
    kind or minikube.

2.  Build the container image with support for coverage profile collection,
    deploy the Kubernetes resource, run the integration tests, and collect the
    coverage profile data:

    ```shell
    skaffold run --profile=coverage
    ```

    The coverage profile data files will be in the directory `reports`.

## Cleaning up

When you are done, remove the resources from your cluster:

```shell
skaffold delete
```

## References

- [Go: Coverage profiling support for integration tests](https://go.dev/testing/coverage/)
- [`runtime/coverage` package in the Go standard library](https://pkg.go.dev/runtime/coverage)
