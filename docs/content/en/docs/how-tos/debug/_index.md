---
title: "Debugging with Skaffold"
linkTitle: "Debugging"
weight: 100
---

This page describes `skaffold debug`, a zero-configuration solution for
setting up containers for debugging on a Kubernetes cluster. 

{{< alert title="Note" >}}
This functionality is in an alpha state and may change without warning.
{{< /alert >}}

## Debugging with Skaffold

`skaffold debug` acts like `skaffold dev`, but it configures containers in pods
for debugging as required for each container's runtime technology.
The associated debugging ports are exposed and labelled and port-forwarded to the
local machine.  Helper metadata is also added to allow IDEs to detect the debugging
configuration parameters.
 
## How it works

`skaffold debug` examines the built artifacts to determine the underlying runtime technology
(currently supported: Java, NodeJS, and Python).  Any Kubernetes manifest that references these
artifacts are transformed to enable the runtime technology's debugging functions:

  - a JDWP agent is configured for Java applications,
  - the Chrome DevTools inspector is configured for NodeJS applications,
  - Python applications are configured to use [`ptvsd`](https://github.com/microsoft/ptvsd/).

`skaffold debug` uses a set of heuristics to identify the runtime
technology.  The Kubernetes manifests are transformed on-the-fly
such that the on-disk representations are untouched.

{{< alert title="Caution" >}}
`skaffold debug` does not support deprecated versions of Workload API objects such as `apps/v1beta1`.
{{< /alert >}}


## Limitations

`skaffold debug` has some limitations:

  - Only the `kubectl` and `kustomize` deployers are supported at the moment: support for
    the Helm deployer is not yet available.
  - File sync is disabled for all artifacts.
  - Only JVM, NodeJS, and Python applications are supported:
      - JVM applications are configured using the `JAVA_TOOL_OPTIONS` environment variable
        which causes extra debugging output on launch.
      - NodeJS applications must be launched using `node` or `nodemon`, or `npm`
          - `npm` scripts shouldn't then invoke `nodemon` as the DevTools inspector
            configuration will be picked up by `nodemon` rather than the actual application
      - Python applications are configured to use [`ptvsd`](https://github.com/microsoft/ptvsd/),
        a wrapper around `pydevd` that uses the
        [_debug adapter protocol_ (DAP)](https://microsoft.github.io/debug-adapter-protocol/). 
        The DAP is supported by Visual Studio Code,
        [Eclipse LSP4e](https://projects.eclipse.org/projects/technology.lsp4e),
        [and other editors and IDEs](https://microsoft.github.io/debug-adapter-protocol/implementors/tools/).
        DAP is not yet supported by JetBrains IDEs like PyCharm.
  
 Support for additional language runtimes will be forthcoming.
