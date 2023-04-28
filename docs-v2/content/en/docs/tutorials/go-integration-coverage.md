---
title: "Go integration test coverage profiles"
linkTitle: "Go integration test coverage profiles"
weight: 100
---

This tutorial describes how to use Skaffold to collect
[coverage profile data](https://go.dev/testing/coverage/)
from Go applications when running
[integration tests](https://go.dev/testing/coverage/#glos-integration-test).
These more comprehensive tests, often called end-to-end tests, are run against
a deployed application, typically testing multiple user journeys.

## Background

Go 1.20 introduced support for collecting coverage profile data from running Go
applications. To enable coverage collection, build the binary with the `-cover`
flag. The application records coverage profile data in a local directory set by
the `GOCOVERDIR` environment variable.

When the application runs on Kubernetes, there is an additional challenge of
copying the coverage profile data files to permanent storage before the pod
terminates.

By default, the coverage profile data files are written on application exit.
This tutorial shows how you can send a signal to write these files without
exiting the application, and then copy the files out of the pods.

## Steps

Skaffold orchestrates the steps of:

1.  Building binary and the container image, with support for collecting
    coverage profiles.
2.  Deploying the application to a Kubernetes cluster.
3.  Running the integration tests.
4.  Sending the signal to write coverage profile data files.
5.  Collecting the counter-data files from the application pods.

For steps 3-5, this tutorial uses Skaffold
[lifecycle hooks]({{<relref "/docs/lifecycle-hooks" >}})
to run these steps automatically.

## The example application

This tutorial refers to the files in the
[`go-integration-coverage`](https://github.com/GoogleContainerTools/skaffold/tree/main/examples/go-integration-coverage)
example.

You may find it helpful to refer to these files as you go through this
tutorial.

## Sending signals for writing coverage profile data files

By default, coverage profile data files are only written on application exit,
specifically on return from `main.main()` or by calling `os.Exit()`. This is
problematic in a Kubernetes pod, as the  application exit triggers pod
termination.

To work around this, add a signal handler to the application. This handler
writes the coverage profile data files when it receives the configured signal,
using the functions in the built-in
[`coverage` package](https://pkg.go.dev/runtime/coverage).
It also clears (resets) the counters, which can be useful if you want separate
coverage profile reports for different sets of tests.

The snippet below is a Go function that sets up a signal handler. It uses the
[`SIGUSR1`](https://www.gnu.org/software/libc/manual/html_node/Miscellaneous-Signals.html)
signal, but you can use another signal in your application.

{{% highlight go "hl_lines=3 8 12-13" %}}
// Note: This snippet omits error handling for brevity.
func SetupCoverageSignalHandler() {
	coverDir, exists := os.LookupEnv("GOCOVERDIR")
	if !exists {
		return
	}
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGUSR1)
	go func() {
		for {
			<-c
			coverage.WriteCountersDir(coverDir)
			coverage.ClearCounters() // only works with -covermode=atomic
        }
	}()
}
{{% / highlight %}}

You can call this function from your `main.main()` function to set up the
signal handler early on in the application lifecycle.

If the `GOCOVERDIR` environment variable is not set, the function returns
without setting up the signal handler. This means that you can control enabling
and disabling the signal handler by whether this environment variable is set.

## Building the binary and the container image

To build the container image with support for coverage profile collection,
compile the binary with the `-cover` flag, and optionally also the `-covermode`
flag.

The image must contain the `tar` command to enable copying the counter-data
files from the pod.

The following snippet shows how to configure the image build using the Skaffold
[ko builder]({{<relref "/docs/builders/builder-types/ko" >}}):

{{% readfile file="samples/builders/ko-flags-cover.yaml" %}}

Using other builders is also possible, by adding the flags to the `go build`
command or by setting the `GOFLAGS` environment variable.

## Running the integration tests

The integration tests can be implemented in a number of ways, since they do not
run in-process with the application.

For instance, you can implement them using Go tests, a shell script with a
sequence of `curl` commands against an HTTP server, or other integration and
end-to-end test frameworks.

Use Skaffold post-deploy hooks to run the tests automatically after deploying
the application. These hooks can run either on the
[`host`]({{<relref "/docs/lifecycle-hooks#before-deploy-and-after-deploy" >}})
where you run Skaffold, or in the deployed
[`container`]({{<relref "/docs/lifecycle-hooks#before-deploy-and-after-deploy-1" >}}).

This tutorial uses a `host` hook that runs a shell script. The shell script
sets up port-forwarding to the service and then runs the integration test. The
arguments to the shell script are used to configure port forwarding.

For this tutorial, the integration test is simply a `curl` command that sends a
HTTP request to the application.

{{% highlight yaml "hl_lines=4" %}}
    hooks:
      after:
      - host:
          command: ["./integration-test/run.sh", "service/go-integration-coverage", "default", "4503", "80"]
          os: [darwin, linux]
{{% / highlight %}}

The arguments to the shell script are:

1.  the Kubernetes resource to port-forward to, e.g., `service/myapp` or
    `deployment/myapp` (required),
2.  the namespace of the Kubernetes resource (defaults to `default`),
3.  the local port (defaults to `4503`), and
4.  the remote port (defaults to `8080`).

After running the integration tests, a `container` hook sends `SIGUSR1` to the
application process (PID 1) using the `kill` command:

{{% highlight yaml "hl_lines=2" %}}
      - container:
          command: ["kill", "-USR1", "1"]
          podName: go-integration-coverage-*
          containerName: app
{{% / highlight %}}

The `podName` and `containerName` fields are required and must match the values
from the Pod spec in your Kubernetes manifest.

If you create multiple pods, the hook will run in all matching pods.

## Copying coverage profile data files

A `host` post-deploy hook runs a shell script that copies the counter-data
files from the pods to the host where you run Skaffold:

{{% highlight yaml "hl_lines=2" %}}
      - host:
          command: ["./integration-test/coverage.sh"]
          os: [darwin, linux]
{{% / highlight %}}

First, the shell script below locates all pods deployed by the Skaffold run
using a selector on the
[`skaffold.dev/run-id` label]({{<relref "/docs/tutorials/skaffold-resource-selector" >}}).

Next, the script iterates over the pods and uses `kubectl exec` to run `tar` in
the containers to package up the counter-data files and pipe them to the host.
On the other end of the pipe, `tar` extracts the files to a report directory on
the host where you run Skaffold.

Finally, the `go tool covdata` command reports the coverage as percentage on
the terminal.

Skaffold provides the
[`SKAFFOLD_KUBE_CONTEXT` and `SKAFFOLD_RUN_ID` environment variables]({{<relref "/docs/lifecycle-hooks#environment-variables" >}})
to the shell script.

## Profiles

The Go binary must be compiled with the `-cover` flag to collect coverage
metrics. However, you may not want to use this flag when compiling for
production use.

Additionally, to simplify metrics reporting, you may want to only specify one
replica in the Kubernetes Deployment resource.

Skaffold [profiles](https://skaffold.dev/docs/environment/profiles/) enable
different configurations for different contexts.

The `skaffold.yaml` file for this tutorial contains a `coverage` profile that
overrides the base configuration as follows:

1.  Specify a base image that contains the `tar` command. `tar` is required to
    copy the coverage profile data files from the pod.

2.  Build the Go binary with the
    [`-cover` and `-covermode` flags](https://go.dev/blog/cover).

3.  Patch the Deployment resource to add a volume and volume mount to the pod
    template spec for the coverage profile data files. This tutorial uses
    [Kustomize](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/kustomization/)
    to patch the resource, but you can use another tool for this in your own
    environment.

4.  Add post-deploy hooks for running integration tests and collecting coverage
    profile data.

To activate the profile, add the flag `--profile coverage` to Skaffold
commands.

## Running the steps

To run the steps, follow the instructions in the
[README.md](https://github.com/GoogleContainerTools/skaffold/tree/main/examples/go-integration-coverage).

## References

- [Go: Coverage profiling support for integration tests](https://go.dev/testing/coverage/)
- [`runtime/coverage` package in the Go standard library](https://pkg.go.dev/runtime/coverage)
