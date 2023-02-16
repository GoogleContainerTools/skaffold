---
title: "Verify [NEW]"
linkTitle: "Verify [NEW]"
weight: 44
featureId: verify
aliases: [/docs/how-tos/verify, /docs/pipeline-stages/verify]
---

In skaffold `v2.0.0-beta3`, skaffold now supports running post-deployment verification tests.  This is done via a new `verify` phase and associated [`verify` schema configuration]({{< relref "/docs/references/yaml#verify" >}}) that allows users to add a list of test containers (either standalone containers or built by skaffold) to be run post-deployment and monitored for success/failure. Currently in the `v2.0.0-beta3` release the `verify` phase is not run in the `skaffold dev` flow but is only accessible via the standalone `skaffold verify` command which runs only the `verify` phase.

Below is an example of a `skaffold.yaml` file with a `verify` configuration that runs 3 verification tests (which all succeed) against deployments including a user built `integration-test-container`, a user built `metrics-test-container`, and a simple health check done via "off the shelf" alpine using its installed `wget`:

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