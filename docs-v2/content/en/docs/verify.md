---
title: "Verify [NEW]"
linkTitle: "Verify [NEW]"
weight: 44
featureId: verify
aliases: [/docs/how-tos/verify, /docs/pipeline-stages/verify]
---

Skaffold `v2.0.0`+ supports running post-deployment verification tests. You define these tests as a list of test containers that are either standalone or built by Skaffold. Skaffold runs these containers after the [deploy]({{< relref "/docs/deployers/" >}}) stage and monitors them for success or failure.

You can configure and execute post-deployment verification tests using the [`verify` command]({{< relref "/docs/references/cli/#skaffold-verify" >}}) and associated [`skaffold.yaml` schema configuration]({{< relref "/docs/references/yaml#verify" >}}).

## Execution modes

You can run post-deployment verification tests in the following execution environments:

* A local Docker environment
* A Kubernetes cluster environment

### Local

When Skaffold runs a post-deployment verifications test in the local execution mode, it uses the `docker` CLI to run the test container on the host machine. This is the default execution mode.

### Kubernetes cluster

When Skaffold runs a post-deployment verification test in the Kubernetes cluster execution mode, it uses the `kubectl` CLI to run the test container as a Kubernetes Job.

There are two ways to optionally customize the Skaffold-generated Kubernetes Job:

* To selectively overwrite the configuration of the Skaffold-generated Kubernetes Job with inline JSON, use the `overrides` configuration option. This is similar to the `--overrides` option provided by `kubectl run`.
* To use your own Kubernetes Job manifest and have Skaffold replace the containers with those specified in the `containers` stanza of your `verify` configuration, use the `jobManifestPath` configuration option.

## Examples

Below is an example of a `skaffold.yaml` file with a `verify` configuration that runs three successful verification tests against deployments:

* A user-built `integration-test-container`, run in the Kubernetes cluster execution mode with optional `overrides` specified.
* A user-built `metrics-test-container`, run in the Kubernetes cluster execution mode with optional `jobManifestPath` specified.
* A simple health check done via "off the shelf" alpine using its installed `wget`, run in the local execution mode.

`skaffold.yaml`
{{% readfile file="samples/verify/verify.yaml" %}}


Running `skaffold verify` against this `skaffold.yaml` (and associated Dockerfiles where relevant) yields:
``` console
$ skaffold verify -a build.artifacts 
Tags used in verification:
 - integration-test-container -> gcr.io/aprindle-test-cluster/integration-test-container:latest@sha256:6d6da2378765cd9dda71cbd20f3cf5818c92d49ab98a2554de12d034613dfa6a
 - metrics-test-container -> gcr.io/aprindle-test-cluster/metrics-test-container:latest@sha256:3fbce881177ead1c2ae00d58974fd6959c648d7691593f6448892c04139355f7
3.15.4: Pulling from library/alpine
Digest: sha256:4edbd2beb5f78b1014028f4fbb99f3237d9561100b6881aabbf5acce2c4f9454
Status: Downloaded newer image for alpine:3.15.4
[integration-test-container] Integration Test 1/4 Running ...
[metrics-test-container] Metrics test in progress...
[metrics-test-container] Metrics test passed!
[alpine-wget] Connecting to www.google.com (142.251.46.196:80)
[alpine-wget] saving to 'index.html'
[alpine-wget] index.html           100% |********************************| 13990  0:00:00 ETA
[alpine-wget] 'index.html' saved
[integration-test-container] Integration Test 1/4 Passed!
[integration-test-container] Integration Test 2/4 Running...!
[integration-test-container] Integration Test 2/4 Passed!
[integration-test-container] Integration Test 3/4 Running...!
[integration-test-container] Integration Test 3/4 Passed!
[integration-test-container] Integration Test 4/4 Running...!
[integration-test-container] Integration Test 4/4 Passed!
$ echo $?
0
```
and `skaffold verify` will exit with error code `0`

If a test fails, for example changing the `alpine-wget` test to point to a URL that doesn't exist:
```yaml
- name: alpine-wget
  container:
    name: alpine-wget
    image: alpine:3.15.4
    command: ["/bin/sh"]
    args: ["-c", "wget http://incorrect-url"]
```

The following will occur (simulating a single test failure on one of the three tests):
```console
$ skaffold verify -a build.artifacts 
Tags used in verification:
 - integration-test-container -> gcr.io/aprindle-test-cluster/integration-test-container:latest@sha256:6d6da2378765cd9dda71cbd20f3cf5818c92d49ab98a2554de12d034613dfa6a
 - metrics-test-container -> gcr.io/aprindle-test-cluster/metrics-test-container:latest@sha256:3fbce881177ead1c2ae00d58974fd6959c648d7691593f6448892c04139355f7
3.15.4: Pulling from library/alpine
Digest: sha256:4edbd2beb5f78b1014028f4fbb99f3237d9561100b6881aabbf5acce2c4f9454
Status: Image is up to date for alpine:3.15.4
[integration-test-container] Integration Test 1/4 Running ...
[metrics-test-container] Metrics test in progress...
[metrics-test-container] Metrics test passed!
[integration-test-container] Integration Test 1/4 Passed!
[alpine-wget] wget: bad address 'incorrect-url'
[integration-test-container] Integration Test 2/4 Running...!
[integration-test-container] Integration Test 2/4 Passed!
[integration-test-container] Integration Test 3/4 Running...!
[integration-test-container] Integration Test 3/4 Passed!
[integration-test-container] Integration Test 4/4 Running...!
[integration-test-container] Integration Test 4/4 Passed!
1 error(s) occurred:
* verify test failed: "alpine-wget" running container image "alpine:3.15.4" errored during run with status code: 1
$ echo $?
1
```
and `skaffold verify` will exit with error code `1`