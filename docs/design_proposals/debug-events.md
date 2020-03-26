# Notify when debuggable containers are started and terminated 

* Author(s): Brian de Alwis (@briandealwis)
* Design Shepherd: Nick Kubala (@nkubala)
* Date: 2019-10-24
* Status: *Implemented* (#3609, #3645)
* Related: #2211

Note: implemented but there were some minor name changes.

## Background

Cloud Code for IntelliJ and VS Code currently launch `skaffold debug --port-forward` to start
an application in debugging mode.  `debug` modifies the podspecs to explicitly expose the debug port used
by the underlying language runtime via `containerPort` definitions.  Skaffold monitors the cluster
and initiated forwards to ports on any newly-started containers, and raises events once the
forwards are established.  The IDEs establish debug launch configurations by examining these
port-forward events for ports with specific names to (e.g., `jdwp` for Java/JVM,
`devtools` for NodeJS, or `dap` for connections using the _Debug Adapter Protocol_).
When a new debuggable container is created, Skaffold will
attempt to forward the remote debug port, which should result in a port-forward event, and thus
spare the IDEs from having to actively watch the cluster. 

But relying strictly on port-forward events carries some risk.  Although the likelihood of 
a port-name clash is low, the IDE may require additional configuration information to establish
a debugging connection.  For example, there may be no indication of the underlying project’s
location in the file-system, nor the relative path in the container. 

Although such information is published by `skaffold debug` via pod annotations, obtaining this
information would require that the IDEs actively track new pods and containers.  This tracking would
duplicate functionality already being performed by Skaffold, and also requiring this functionality to be
re-implemented in each IDE.

This design proposal recommends that Skaffold instead also watch and issue events when debuggable
containers are started and terminated.

----
## Design

This design proposes that Skaffold issue events to inform the IDEs of the following conditions:

- The appearance of a container that is debuggable.  This event will provide:
    - the namespace
    - the pod ID
    - the container ID
    - the contents of the `debug.cloud.google.com/config` configuration block for the container
      (a JSON object) including (! indicates new items)
         - the language runtime type (`go`, `jvm`, `nodejs`, `python`)
         - ! the image name (which should allow cross-referencing with the `skaffold.yaml`)
         - the port(s)
         - ! the container image's working directory
         - ! (optional) the container image's _remote root_ for source file resolving 

- The disappearance of a container that was debuggable.  This event will provide:
    - the namespace
    - the pod ID
    - the container ID
    - the contents of the `debug.cloud.google.com/config` configuration block for the container
      (a JSON object) including (! indicates new items)
         - the language runtime type (`go`, `jvm`, `nodejs`, `python`)
         - ! the corresponding artifact's `name` (which should allow cross-referencing with the `skaffold.yaml`)
         - the port(s)
         - ! the container image's working directory
         - ! (optional) the container image's _remote root_ for source file resolving 
         
Skaffold will also maintain state and report on the set of debuggable containers.

With this information, the IDE should be able to:

- Identify the corresponding artifact within the `skaffold.yaml`
- Identify the corresponding artifact from a port-forward event by correlating the
  remote port with the debug port from the previously-raised debug event
- Establish new debug launch configurations with the forwarded port,
  and the local source location, and any other necessary information
- Tear down existing debug launch configurations when a container is terminated
- (*In the future*, *if necessary*) Request port-forward of the debug port for the
  namespace/pod/container/port on-demand
 
### Example

#### Events in Action
```json
{
  "result": {
    "timestamp": "2019-10-23T18:50:38.916931Z",
    "event": {
      "deployEvent": {
        "status": "In Progress"
      }
    },
    "entry": "Deploy started"
  }
}
{
  "result": {
    "timestamp": "2019-10-23T18:50:39.629826Z",
    "event": {
      "deployEvent": {
        "status": "Complete"
      }
    },
    "entry": "Deploy complete"
  }
}
{
  "result": {
    "timestamp": "2019-10-23T18:50:41.004396Z",
    "event": {
      "debugEvent": {
        "status": "Started",
        "podName": "npm",
        "containerName": "web",
        "namespace": "default",
        "runtime": "nodejs",
        "configuration": "{\"name\":\"gcr.io/artifact/name/from/skaffold.yaml\",\"ports\":{\"devtools\":9229},\"runtime\":\"nodejs\",\"workingDir\":\"/app\"}"
      }
    },
    "entry": "Debuggable container started pod/npm:web (default)"
  }
}
{
  "result": {
    "timestamp": "2019-10-23T18:50:41.004551Z",
    "event": {
      "portEvent": {
        "localPort": 3000,
        "remotePort": 3000,
        "podName": "npm",
        "containerName": "web",
        "namespace": "default",
        "resourceType": "pod",
        "resourceName": "npm"
      }
    },
    "entry": "Forwarding container web to local port 3000"
  }
}
{
  "result": {
    "timestamp": "2019-10-23T18:50:41.004792Z",
    "event": {
      "portEvent": {
        "localPort": 9229,
        "remotePort": 9229,
        "podName": "npm",
        "containerName": "web",
        "namespace": "default",
        "portName": "devtools",
        "resourceType": "pod",
        "resourceName": "npm"
      }
    },
    "entry": "Forwarding container web to local port 9229"
  }
}
```

#### State in Action

```json
$ curl -s localhost:50052/v1/state | jq .
{
  "buildState": {
    "artifacts": {
      "gcr.io/k8s-skaffold/skaffold-jib": "Complete"
    }
  },
  "deployState": {
    "status": "Complete"
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
      "podName": "web-6d7c9db467-gqr5x",
      "containerName": "web",
      "namespace": "default",
      "runtime": "jvm",
      "configuration": "{\"name\":\"gcr.io/artifact/name/from/skaffold.yaml\",\"ports\":{\"jdwp\":5005},\"runtime\":\"jvm\",\"workingDir\":\"/app\"}"
    }
  ]
}
```

### Open Issues/Questions

**\<Why is a new Debug Event necessary?\>**

One alternative explored was to piggy-back the debug information into the existing port-forward events.
This design was eliminated as:
 
  - There are some language runtimes, like .NET Core, that do not use debug ports and instead
    require a `kubectl exec` to launch a process in the remote container. 
  - Port-forwards may be dropped and re-established
  
There may also be a desire for the IDEs to selectively initiate port-forwards on an as-needed
basis.  This may be useful for large microservice deployments.  Indeed the debugging podspec
transformations could be applied on-the-fly too. 

## Implementation plan

The idea here is that `skaffold debug` internally configures a watcher that looks for and notifies
of pods with debuggable containers, similar to how the [port-forwarding manager](https://github.com/GoogleContainerTools/skaffold/blob/master/pkg/skaffold/kubernetes/portforward/pod_forwarder.go) 
listens for pods or services.

1. `skaffold debug` will set a `DebugMode` option to `true`. This value will be used in
  `SkaffoldRunner.Dev()` to install the debug container watch manager.
1. Add a new `DebugEvent` type to the protobuf definitions.  The status field will be one
   of `"Started"` or `"Terminated"`.
1. Add new debuggable container state to the general state.
1. Generalize the port-forward manager or create a new parallel debug watcher to notify
   of pod events.
  
___
## Integration test plan

Tests will need to verify that pod- and container-start successfully raise debugging _Started_ events,
and container termination and pod-deletion raises _Terminated_ events. 
