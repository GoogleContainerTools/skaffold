---
title: "Debugging With Skaffold"
linkTitle: "Debugging"
weight: 30
featureId: debug
aliases: [/docs/how-tos/debug]
---

Skaffold lets you set breakpoints and step through your application, even when deployed to remote Kubernetes clusters, as if the code were running locally on your machine. Skaffold detects the language runtimes of your container images, reconfigures the pods for debugging, forwards debugging ports, and then monitors for when debuggable containers come online.  IDE integrations like Cloud Code leverage Skaffold's events to automatically set up debugging sessions.

Debugging is currently supported for five language runtimes.

  - Go 1.13+ (runtime ID: `go`) using [Delve](https://github.com/go-delve/delve)
  - NodeJS (runtime ID: `nodejs`) using the NodeJS Inspector (Chrome DevTools)
  - Java and JVM languages (runtime ID: `jvm`) using JDWP
  - Python 3.5+ (runtime ID: `python`) using `debugpy` (Debug Adapter Protocol) or `pydevd`
  - .NET Core (runtime ID: `netcore`) using `vsdbg` (only for VS Code)

Skaffold can usually detect the correct language runtime if present. However if you encounter difficulties then checkout the [Supported Language Runtimes]({{< relref "#supported-language-runtimes">}}) section for the exact heuristics that Skaffold uses and you can modify your application accordingly, or read about [how you can manually configure your container image]({{<relref "#can-images-be-debugged-without-the-runtime-support-images">}}).

## (Recommended) Debugging using Cloud Code

The easiest way to debug using Skaffold is by using the [Cloud Code IDE extension]({{< relref "../install/#managed-ide">}}) for Visual Studio Code, JetBrains IDEs, and Cloud Shell. Cloud Code will automatically configure container images for debugging so you can debug Kubernetes services just like how you would debug a local service in your IDE.

For more information, see the corresponding documentation for [VS Code](https://cloud.google.com/code/docs/vscode/debug), [IntelliJ](https://cloud.google.com/code/docs/intellij/kubernetes-debugging) and [Cloud Shell](https://cloud.google.com/code/docs/shell/debug).

## Detailed debugger configuration and setup

This section describes how Skaffold configures the container image for debugging for the supported language runtimes, and how to setup with certain IDEs.

Note that many debuggers may require additional information for the location of source files.
We are looking for ways to identify this information and to pass it back if found.

{{% tabs %}}
{{% tab "GO" %}}

#### Go (runtime: `go`, protocols: `dlv`)

Go-based applications are configured to run under
[Delve](https://github.com/go-delve/delve) in its headless-server mode.

On recognizing a Go-based container image, `debug` rewrites the container image's
entrypoint to invoke your application using `dlv`:

```
dlv exec --headless --continue --accept-multiclient --listen=:56268 --api-version=2 <app> -- <args> ...
```

##### Skaffold debug using the JetBrains GoLand and IntelliJ Ultimate

Debugging is only supported in JetBrains IDEs for Go applications built using Go Modules.
The IDE settings must also be explicitly configured to use Go Modules.  Errors like
`cannot find debugger path` indicate misconfiguration.


##### Skaffold debug using the VS Code Go extension

If you use the debug functionality of the
[VS Code Go extension](https://github.com/golang/vscode-go) to connect to
`dlv`, you may need to configure _local_ and _remote_ source paths, via
either the `cwd` and `remotePath` properties, or the `substitutePath` array
property
([reference](https://github.com/golang/vscode-go/blob/master/docs/debugging-legacy.md#launch-configurations)).

The `cwd` or `from` properties should be the absolute path of the top-level
directory of your source files and should generally match the artifact's
`context` directory in `skaffold.yaml`. You can represent this using the
`${workspaceFolder}` variable.

The `remotePath` or `to` properties should be set to the absolute path of the
source location _during compilation_. For example, the `golang` images, which
are
[often used in multi-stage builds](https://github.com/GoogleContainerTools/skaffold/tree/main/examples/getting-started/Dockerfile),
have a default working directory of `/go`. The following
[configuration](https://github.com/golang/vscode-go/blob/master/docs/debugging-legacy.md#remote-debugging)
works in this case:

```json
{
    "name": "Skaffold Debug",
    "type": "go",
    "request": "attach",
    "mode": "remote",
    "host": "localhost",
    "port": 56268,
    "cwd": "${workspaceFolder}",
    "remotePath": "/go"
}
```

You can use the `substitutePath` property instead of `cwd` and `remotePath`:

```json
{
    "name": "Skaffold Debug",
    "type": "go",
    "request": "attach",
    "mode": "remote",
    "host": "localhost",
    "port": 56268,
    "substitutePath": [
        {
            "from": "${workspaceFolder}",
            "to": "/go",
        },
    ],
}
```

If you use the `ko` local builder, the source path during compilation matches
the source path in your IDE. In this case, the value for `remotePath` will
match `cwd`. If the values of both `cwd` and `remotePath` are
`${workspaceFolder}`, you can omit these properties.

{{% /tab %}}
{{% tab "NODEJS" %}}

#### NodeJS (runtime: `nodejs`, protocols: `devtools`)

NodeJS applications are configured to use the Chrome DevTools inspector via the `--inspect` argument.

On recognizing a NodeJS-based container image, `debug` rewrites the container image's
entrypoint to invoke your application with `--inspect`:
```
node --inspect=9229 <app.js>
```

{{< alert title="Note" >}}
Many applications use NodeJS-based tools as part of their launch, like <tt>npm</tt>, rather than
invoke <tt>node</tt> directly.  These intermediate <tt>node</tt> instances may interpret the
<tt>--inspect</tt> arguments.  Skaffold introduces a <tt>node</tt> wrapper that
only invokes the real <tt>node</tt> with <tt>--inspect</tt> if running an application script,
and skips scripts located in <tt>node_modules</tt>.  For more details see the
<a href="https://github.com/GoogleContainerTools/container-debug-support/pull/34">associated PR</a>.
{{< /alert >}}

A debugging client must first obtain [the inspector UUID](https://github.com/nodejs/node/issues/9185#issuecomment-254872466).

{{% /tab %}}
{{% tab "JAVA" %}}

#### Java and Other JVM Languages (runtime: `jvm`, protocols: `jdwp`)

Java/JVM applications are configured to expose the JDWP agent using the `JAVA_TOOL_OPTIONS`
environment variable.
Note that the use of `JAVA_TOOL_OPTIONS` causes extra debugging output from the JVM on launch.

On recognizing a JVM-based container image, `debug` rewrites the container image's
environment to set:
```
JAVA_TOOL_OPTIONS=-agentlib:jdwp=transport=dt_socket,server=y,address=5005,suspend=n,quiet=y
```
{{% /tab %}}
{{% tab "PYTHON" %}}

#### Python (runtime: `python`, protocols: `dap` or `pydevd`)

Python applications are configured to use either  [`debugpy`](https://github.com/microsoft/debugpy/) or
[`pydevd`](https://github.com/fabioz/PyDev.Debugger).  `debugpy` uses the
[_debug adapter protocol_ (DAP)](https://microsoft.github.io/debug-adapter-protocol/) which
is supported by Visual Studio Code, [Eclipse LSP4e](https://projects.eclipse.org/projects/technology.lsp4e),
[and other editors and IDEs](https://microsoft.github.io/debug-adapter-protocol/implementors/tools/).

On recognizing a Python-based container image, `debug` rewrites the container image's
entrypoint to invoke Python using either the `pydevd` or `debugpy` modules:
```
python -m debugpy --listen 5678 <app>
```

or
```
python -m pydevd --server --port 5678 <app.py>
```

{{< alert title="Note" >}}
As many Python web frameworks use launcher scripts, like `gunicorn`, Skaffold now uses
a debug launcher that examines the app command-line.
{{< /alert >}}

{{% /tab %}}
{{% tab ".NETCORE" %}}

#### .NET Core (runtime: `dotnet`, protocols: `vsdbg`)

.NET Core applications are configured to be deployed along with `vsdbg`
for VS Code.

{{< alert title="JetBrains Rider" >}}
This set up does not yet work automatically with [Cloud Code for IntelliJ in JetBrains Rider](https://github.com/GoogleCloudPlatform/cloud-code-intellij/issues/2903).
There is [a manual workaround](https://github.com/GoogleCloudPlatform/cloud-code-intellij/wiki/Manual-set-up-for-remote-debugging-in-Rider).
{{< /alert >}}

{{< alert title="Omnisharp for VS Code" >}}
For users of [VS Code's debug adapter for C#](https://github.com/OmniSharp/omnisharp-vscode):**
the following configuration can be used to debug a container. It assumes that your code is deployed
in `/app` or `/src` folder in the container. If that is not the case, the `sourceFileMap` property
should be changed to match the correct folder. `processId` is usually 1 but might be different if you
have an unusual entrypoint. You can also use `"${command:pickRemoteProcess}"` instead if supported by
your base image.  (`//` comments must be stripped.)
```json
{
    "name": "Skaffold Debug",
    "type": "coreclr",
    "request": "attach",
    "processId" : 1,
    "justMyCode": true, // set to `true` in debug configuration and `false` in release configuration
    "pipeTransport": {
        "pipeProgram": "kubectl",
        "pipeArgs": [
            "exec",
            "-i",
            "<NAME OF YOUR POD>", // name of the pod you debug.
            "--"
        ],
        "pipeCwd": "${workspaceFolder}",
        "debuggerPath": "/dbg/netcore/vsdbg", // location where vsdbg binary installed.
        "quoteArgs": false
    },
    "sourceFileMap": {
        // Change this mapping if your app in not deployed in /src or /app in your docker image
        "/src": "${workspaceFolder}",
        "/app": "${workspaceFolder}"
        // May also be like this, depending of your repository layout
        // "/src": "${workspaceFolder}/src",
        // "/app": "${workspaceFolder}/src/<YOUR PROJECT TO DEBUG>"
    }
}
```
{{< /alert >}}

{{% /tab %}}
{{% /tabs %}}

## Supported Language Runtimes

This section describes how `debug` recognizes the language runtime used in a
container image for specific language runtimes.

{{% tabs %}}
{{% tab "GO" %}}
#### Go (runtime: `go`)

Go-based container images are recognized by:

- the presence of one of the
  [standard Go runtime environment variables](https://godoc.org/runtime):
  `GODEBUG`, `GOGC`, `GOMAXPROCS`, or `GOTRACEBACK`, or
- the presence of the
  [`KO_DATA_PATH` environment variable](https://github.com/google/ko#static-assets)
  in container images built by
  [`ko`]({{< relref "/docs/pipeline-stages/builders/ko" >}}), or
- is launching using `dlv`.

Unless you built your container image using `ko`, set one of the standard Go
environment variables to allow Skaffold to detect your container as Go.
`GOTRACEBACK=single` is the default setting for Go, so you can add this
environment variable without changing behavior. Alternatively, `GOTRACEBACK=all`
is a useful setting in many situtions.

Your application should be built with the `-gcflags='all=-N -l'` flag to
disable optimizations and inlining. If you do not add this flag, debugging can
becoming confusing due to seemingly-random execution jumps that happen due to
statement reordering and inlining. Skaffold configures Docker builds with a
`SKAFFOLD_GO_GCFLAGS` build argument that you should use in your `Dockerfile`:

```
FROM golang
ENV GOTRACEBACK=all
COPY . .
ARG SKAFFOLD_GO_GCFLAGS
RUN go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o /app .
```

Note that the Alpine Linux-based `golang` container images do not include a
C compiler which is required to build with `-gcflags='all=-N -l'`.

The ko builder adds `-gcflags='all=-N -l'` automatically when you use
`skaffold debug`.

We recommend that you do _not_ add the `-trimpath` flag when debugging, as this
flag removes information from the resulting binary that aids debugging.

{{% /tab %}}
{{% tab "JAVA" %}}

#### Java and Other JVM Languages (runtime: `jvm`)

JVM application are recognized by:
- the presence of a `JAVA_VERSION` or `JAVA_TOOL_OPTIONS` environment variable, or
- the container command-line invokes `java`.

{{% /tab %}}
{{% tab "NODEJS" %}}

#### NodeJS (runtime: `nodejs`)

NodeJS images are recognized by:
- the presence of a `NODE_VERSION`, `NODEJS_VERSION`, or `NODE_ENV` environment variable, or
- the container command-line invokes `node` or `npm`.

{{% /tab %}}
{{% tab "PYTHON" %}}

#### Python (runtime: `python`)

Python application are recognized by:
- the presence of a standard Python environment variable:
  `PYTHON_VERSION`, `PYTHONVERBOSE`, `PYTHONINSPECT`, `PYTHONOPTIMIZE`,
  `PYTHONUSERSITE`, `PYTHONUNBUFFERED`, `PYTHONPATH`, `PYTHONUSERBASE`,
  `PYTHONWARNINGS`, `PYTHONHOME`, `PYTHONCASEOK`, `PYTHONIOENCODING`,
  `PYTHONHASHSEED`, `PYTHONDONTWRITEBYTECODE`, or
- the container command-line invokes `python`, `python2`, or `python3`.

{{% /tab %}}
{{% tab ".NETCORE" %}}

#### .NET Core (runtime: `dotnet`)

.NET Core application are recognized by:
- the presence of a standard .NET environment variable:
  `ASPNETCORE_URLS`, `DOTNET_RUNNING_IN_CONTAINER`,
  `DOTNET_SYSTEM_GLOBALIZATION_INVARIANT`, or
- the container command-line invokes the [dotnet](https://github.com/dotnet/sdk) cli

Furthermore, your app must be built with the `--configuration Debug` options to disable optimizations.

{{% /tab %}}
{{% /tabs %}}

## How It works

`skaffold debug` acts like `skaffold dev`, but it configures containers in pods
for debugging as required for each container's runtime technology.
The associated debugging ports are exposed and labelled so that they can be port-forwarded to the
local machine.
One notable difference from `skaffold dev` is that `debug` disables image rebuilding and
syncing as it leads to users accidentally terminating debugging sessions by saving file changes.
These behaviours can be re-enabled with the `--auto-build`, `--auto-deploy`, and `--auto-sync`
flags, or triggering a [devloop iteration]({{< relref "/docs/design/api#control-api" >}}) using the Skaffold API.

Enabling debugging has two phases:

1. **Configuring:** Skaffold automatically examines each built container image and
   attempts to recognize the underlying language runtime.  Container images can be
   explicitly configured too.
3. **Monitoring:** Skaffold watches the cluster to detect when debuggable containers
   start execution.

### Configuring container images for debugging

`skaffold debug` examines the *built artifacts* to determine the underlying language runtime technology.
Kubernetes manifests that reference these artifacts are transformed on-the-fly to enable the
language runtime's debugging functionality.  These transforms add or alter environment variables
and entrypoints, and more.

Some language runtimes require additional support files to enable debugging.
For these languages, a special set of [runtime-specific images](https://github.com/GoogleContainerTools/container-debug-support)
are configured as _init-containers_ to populate a shared-volume that is mounted into
each of the appropriate containers.  These images are hosted at
`gcr.io/k8s-skaffold/skaffold-debug-support`; alternative locations can be
specified in [Skaffold's global configuration]({{< relref "/docs/design/global-config.md" >}}).

For images that are successfully recognized, Skaffold adds a `debug.cloud.google.com/config`
annotation to the corresponding Kubernetes pod-spec that encode the debugging parameters.

### Monitoring for debuggable containers

Once the application is deployed, `debug` monitors the cluster looking for debuggable pods with a
`debug.cloud.google.com/config` annotation.  For each new debuggable pod,  Skaffold emits
an event that can be used by tools like IDEs to establish a debug session.

### Additional changes

`debug` makes some other adjustments to simplify the debug experience:

  - *Replica Counts*: `debug` rewrites  the replica counts to 1 for
    deployments, replica sets, and stateful sets.  This results in
    requests being serialized so that one request is processed at a time.

  - *Kubernetes Probes*:  `debug` changes the timeouts on HTTP-based
    [liveness, readiness, and startup probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
    to 600 seconds (10 minutes) from the default of 1 second.
    This change allows probes to be debugged, and avoids negative
    consequences from blocked probes when the app is already suspended
    during a debugging session.
    Failed liveness probes in particular result in the container
    being terminated and restarted.

	The probe timeout value can be set on a per-podspec basis by setting
	a `debug.cloud.google.com/probe/timeouts` annotation on the podspec's metadata
	with a valid duration (see [Go's time.ParseDuration()](https://pkg.go.dev/time#ParseDuration)).
    This probe timeout-rewriting can be skipped entirely by using `skip`.  For example:
    ```yaml
    metadata:
      annotations:
        debug.cloud.google.com/probe/timeouts: skip
    spec: ...
    ```

## Troubleshooting

### My container is not being made debuggable?

**Was this image built by Skaffold?**
`debug` only works for images that were built by
 Skaffold so as to avoid affecting system- or infrastructure-level containers such as proxy sidecars.

**Was Skaffold able to recognize the image?**
`debug` emits a warning when it is unable to configure an image for debugging:
```
WARN[0005] Image "image-name" not configured for debugging: unable to determine runtime for "image-name"
```

See the language runtime section details on how container images are recognized.

**Is your container command-line using shell constructs?**

Automatic debugging of Go and Python applications requires that Skaffold
adjust the command-line to use a specific launcher (`dlv` for Go, or our
custom Python launcher to configure the appropriate debugging module).
These launchers require that the container command-line invokes the
binary or launch script (i.e., a `pip`-generated launcher script such as `gunicorn`).
Skaffold cannot debug Go and Python applications that use user-created
shell scripts, or that use shell constructs like `exec` or `eval`.
Either rewrite your container image command-line or
[manually configure your container for debugging]({{ relref "#can-images-be-debugged-without-the-runtime-support-images" }}).
For example:
```
❌ CMD launch.sh
❌ CMD exec gunicorn app:app
✅ CMD gunicorn app:app
✅ CMD [gunicorn, app:app]
✅ CMD [python, -m, gunicorn, app:app]
```


### Why aren't my breakpoints being hit?

**Missing required language extension.**
Go, Python, and JavaScript debugging in IDEA Ultimate requires the
corresponding language plugin to be installed. If it is not installed,
the debugger will not attach and breakpoints will not be hit.

**Breakpoints are not hit during program startup.**
Skaffold configures the containers to run-on-start (sometimes called
_continue_ mode), such that the containers do not wait for the
debugger to connect.  The container will have likely finished its
startup by the time the debugger is able to connect and configure
the breakpoints, and so breakpoints in the normal startup path are
unlikely to be hit.

**File mapping misconfiguration between local and container filesystems.**
Some language runtimes require configuring the IDE with a _source map_
to map files in the container to their corresponding local files.
If this mapping is incorrect, then the remote debugging runtime
will not be able to find the source file for the breakpoint. The Cloud Code plugins generally prompt the user when required.

- Some builds use multi-stage dockerfiles to compile the binary using
  a build-time image and then copy the binary into a smaller image
  intended for deployment.  In these cases, the source mapping must
  use the build-time paths.  For example, many Go builds use a `golang`
  image for building, where the source files are typically copied into `/go`.
- JetBrains GoLand and IDEA Ultimate only support remote breakpoints
  for Go applications built using Go Modules.  The IDE settings must
  also be explicitly configured to use Go Modules.
- JVM languages do not require the mapping as the JVM uses class
  names rather than file paths.

### Can images be debugged without the runtime support images?

The special [runtime-support images](https://github.com/GoogleContainerTools/container-debug-support)
are provided as a convenience for automatic configuration.  You can
manually configure your images for debugging by one of the following
means:

- `skaffold debug` usually recognizes already-configured container images
  that use the appropriate debugger.  For example, you could use a
  multi-stage Dockerfile with a `debug` stage that invokes the
  application using the debugger.
- Use a `debug.cloud.google.com/config` workload annotation on the
  pod-spec to describe the debug configuration of each container
  image in the pod, as described in [_Workload Annotations_](#workload-annotations).

## Limitations

`skaffold debug` has some limitations.

### Unsupported Container Entrypoints

`skaffold debug` requires being able to examine and alter the
command-line used in the container entrypoint.  This transformation
will not work with images that use intermediate launch scripts or
binaries.

### Supported Deployers

`skaffold debug` is only supported with the `kubectl`, `kustomize`, and `helm` deployers.

{{< alert title="Note" >}}
Helm support requires using Helm v3.1.0 or greater.
{{< /alert >}}


### Deprecated Workload API Objects

`skaffold debug` does not support deprecated versions of Workload API objects:

  - `extensions/v1beta1` and `apps/v1beta1` was [deprecated in Kubernetes 1.8](https://github.com/kubernetes/kubernetes/blob/HEAD/CHANGELOG/CHANGELOG-1.8.md#other-notable-changes-16)
    and [removed in Kubernetes 1.16](https://kubernetes.io/blog/2019/07/18/api-deprecations-in-1-16/).
  - `apps/v1beta2` was [deprecated in Kubernetes 1.9](https://github.com/kubernetes/kubernetes/blob/HEAD/CHANGELOG/CHANGELOG-1.9.md#apps)
    and [removed in Kubernetes 1.16](https://kubernetes.io/blog/2019/07/18/api-deprecations-in-1-16/).

Applications should transition to the `apps/v1` APIs,
[introduced in Kubernetes 1.9](https://kubernetes.io/blog/2017/12/kubernetes-19-workloads-expanded-ecosystem/#workloads-api-ga).

----

## Appendix: IDE Support via Events and Metadata {#metadata-events}

`debug` provides additional support for IDEs to detect the debuggable containers and to determine
appropriate configuration parameters.

### Workload Annotations

Each transformed workload object carries a `debug.cloud.google.com/config` annotation with
a JSON object describing the debug configurations for the pod's containers (linebreaks for readability):
```
	debug.cloud.google.com/config={
		"<containerName>":{"runtime":"<runtimeId>",...},
		"<containerName>":{"runtime":"<runtimeId>",...},
		}
```

For example the following annotation indicates that the container named `web` is a Go application
that is being debugged by a headless Delve session on port `56268` (linebreaks for readability):
```
debug.cloud.google.com/config={
  "web":{
    "artifact":"gcr.io/random/image",
    "runtime":"go",
    "ports":{"dlv":56268},
    "workingDir":"/some/path"}}
```

`artifact` is the corresponding artifact's image name in the `skaffold.yaml`.
`runtime` is the language runtime detected (one of: `go`, `jvm`, `nodejs`, `python`).
`ports` is a list of debug ports keyed by the language runtime debugging protocol.
`workingDir` is the working directory (if not an empty string).


### API: Events

Each debuggable container being started or stopped raises a _debug-container-event_ through
Skaffold's event mechanism ([gRPC](../references/api/grpc/#debuggingcontainerevent),
[REST](../references/api/swagger/#/SkaffoldService/Events)).

<details>
<summary>`/v1/events` stream of `skaffold debug` within `examples/jib`</summary>

In this example, we do a `skaffold debug`, and then kill the deployed pod.  The deployment starts a new pod.  We get a terminated event for the container for the killed pod.

```json
{
  "result": {
    "timestamp": "2020-02-05T03:27:30.114354Z",
    "event": {
      "debuggingContainerEvent": {
        "status": "Started",
        "podName": "web-f6d56bcc5-6csgs",
        "containerName": "web",
        "namespace": "default",
        "artifact": "skaffold-jib",
        "runtime": "jvm",
        "debugPorts": {
          "jdwp": 5005
        }
      }
    },
    "entry": "Debuggable container started pod/web-f6d56bcc5-6csgs:web (default)"
  }
}
```

</details>



### API: State

The API's _state_ ([gRPC](../references/api/grpc/#skaffoldservice), [REST](../references/api/swagger/#/SkaffoldService/GetState)) also includes a list of debuggable containers.

<details>
<summary>The `/v1/state` listing debugging containers</summary>

```json
{
  "buildState": {
    "artifacts": {
      "skaffold-jib": "Complete"
    }
  },
  "deployState": {
    "status": "Complete"
  },
  "forwardedPorts": {
    "5005": {
      "localPort": 5005,
      "remotePort": 5005,
      "podName": "web-f6d56bcc5-6csgs",
      "containerName": "web",
      "namespace": "default",
      "portName": "jdwp",
      "resourceType": "pod",
      "resourceName": "web-f6d56bcc5-6csgs",
      "address": "127.0.0.1"
    },
    "8080": {
      "localPort": 8080,
      "remotePort": 8080,
      "namespace": "default",
      "resourceType": "service",
      "resourceName": "web",
      "address": "127.0.0.1"
    },
    "8081": {
      "localPort": 8081,
      "remotePort": 8080,
      "podName": "web-f6d56bcc5-6csgs",
      "containerName": "web",
      "namespace": "default",
      "resourceType": "pod",
      "resourceName": "web-f6d56bcc5-6csgs",
      "address": "127.0.0.1"
    }
  },
  "statusCheckState": {
    "status": "Not Started"
  },
  "fileSyncState": {
    "status": "Not Started"
  },
  "debuggingContainers": [
    {
      "status": "Started",
      "podName": "web-f6d56bcc5-6csgs",
      "containerName": "web",
      "namespace": "default",
      "artifact": "skaffold-jib",
      "runtime": "jvm",
      "debugPorts": {
        "jdwp": 5005
      }
    }
  ]
}

```

</details>

