---
title: "Debugging With Skaffold"
linkTitle: "Debugging"
weight: 30
featureId: debug
---

`skaffold debug` acts like `skaffold dev`, but it configures containers in pods
for debugging as required for each container's runtime technology.
The associated debugging ports are exposed and labelled so that they can be port-forwarded to the
local machine. Helper metadata is also added to allow IDEs to detect the debugging
configuration parameters.
 
## How it Works

`skaffold debug` examines the built artifacts to determine the underlying runtime technology.
Any Kubernetes manifest that references these artifacts are transformed to enable the runtime technology's
debugging functions.

`skaffold debug` uses a set of heuristics to identify the runtime technology.  
The Kubernetes manifests are transformed on-the-fly such that the on-disk representations
are untouched.

Each Pod will have an `debug.cloud.google.com/config` annotation with a JSON object
describing the debug configurations for the pod's containers (linebreaks for readability):
```  
	debug.cloud.google.com/config={
		"<containerName>":{"runtime":"<runtimeId>",...},
		"<containerName>":{"runtime":"<runtimeId>",...},
		}
```

For example the following annotation indicates that the container named `web` is a Go application
that is being debugged by a headless Delve session on port `56268`:
```
debug.cloud.google.com/config={"web":{"dlv":56268,"runtime":"go"}}
```

Some language runtimes require additional support files to enable debugging.
For these languages, a special set of [runtime-specific images](https://github.com/GoogleContainerTools/container-debug-support)
are configured as _init-containers_ to populate a shared-volume that is mounted into
each of the appropriate containers.  These images are hosted at `gcr.io/gcp-dev-tools/duct-tape`.


### Supported Language Runtimes

Debugging is currently supported for Go, Java (and JVM languages), NodeJS, and Python.

#### Go

Go-based applications are configured to run under [Delve](https://github.com/go-delve/delve) in its headless-server mode.

  - Go application should self-identify by setting one of the [standard Go runtime
    environment variables](https://godoc.org/runtime) such as `GODEBUG`, `GOGC`, `GOMAXPROCS`,
    or `GOTRACEBACK`. `GOTRACEBACK=all` is a generally useful configuration.
  - Go applications should be built without optimizations, so your build should be capable of building with
    `-gcflags='all=-N -l'`. Skaffold [_Profiles_]({{< relref "/docs/environment/profiles.md" >}}) are a useful option.

Note for users of [VS Code's debug adapter for Go](https://github.com/Microsoft/vscode-go): Delve seems
to treat the source location for headless launches as being relative to `/go`.  The following
[remote launch configuration](https://github.com/Microsoft/vscode-go/wiki/Debugging-Go-code-using-VS-Code#remote-debugging) was useful:
```json
{
  "name": "Skaffold Debug",
  "type": "go",
  "request": "launch",
  "mode": "remote",
  "host": "localhost",
  "port": 56268,
  "remotePath": "/go/",
  "program": "/Users/login/go/src/github.com/GoogleContainerTools/skaffold/integration/testdata/debug/go/",
}
```

#### Java and Other JVM Languages

Java/JVM applications are configured to expose the JDWP agent using the `JAVA_TOOL_OPTIONS`
environment variable.  
Note that the use of `JAVA_TOOL_OPTIONS` causes extra debugging output from the JVM on launch.

#### NodeJS

NodeJS applications are configured to use the Chrome DevTools inspector.  
NodeJS applications must be launched using `node` or `nodemon`, or `npm`.
Note that `npm` scripts should not then invoke `nodemon` as the DevTools inspector
configuration will be picked up by `nodemon` rather than the actual application.

Note that the client must first obtain [the inspector UUID](https://github.com/nodejs/node/issues/9185#issuecomment-254872466).
  
#### Python

Python applications are configured to use [`ptvsd`](https://github.com/microsoft/ptvsd/), a
wrapper around [`pydevd`](https://github.com/fabioz/PyDev.Debugger) that uses the
[_debug adapter protocol_ (DAP)](https://microsoft.github.io/debug-adapter-protocol/). 

The DAP is supported by Visual Studio Code, [Eclipse LSP4e](https://projects.eclipse.org/projects/technology.lsp4e),
[and other editors and IDEs](https://microsoft.github.io/debug-adapter-protocol/implementors/tools/).
DAP is not yet supported by JetBrains IDEs like PyCharm.


## Limitations

`skaffold debug` has some limitations.

### Unsupported Container Entrypoints

`skaffold debug` requires being able to examine and alter the
command-line used in the container entrypoint.  This transformation
will not work with images that use intermediate launch scripts or
binaries.  For example, `debug` cannot work with an image produced
by the Cloud Native Buildpacks builder as it uses a `launcher`
binary to run commands that are specified in a set of configuration
files.

### Supported Deployers

`skaffold debug` is only supported with the `kubectl` and `kustomize` deployers at the moment: support for
the Helm deployer is not yet available ([#2350](https://github.com/GoogleContainerTools/skaffold/issues/2350)).

### Deprecated Workload API Objects

`skaffold debug` does not support deprecated versions of Workload API objects:

  - `apps/v1beta1` was [deprecated in Kubernetes 1.8](https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG-1.8.md#other-notable-changes-16)
  - `apps/v1beta2` was [deprecated in Kubernetes 1.9](https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG-1.9.md#apps)

Both have been [removed in Kubernetes 1.16](https://kubernetes.io/blog/2019/07/18/api-deprecations-in-1-16/).
Applications should transition to the `apps/v1` APIs,
[introduced in Kubernetes 1.9](https://kubernetes.io/blog/2017/12/kubernetes-19-workloads-expanded-ecosystem/).

