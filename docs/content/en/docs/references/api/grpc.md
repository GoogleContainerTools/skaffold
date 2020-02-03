---
title: "gRPC API"
linkTitle: "gRPC API"
weight: 30
---
<!--
******
WARNING!!!

The file docs/content/en/docs/references/api/grpc.md is generated based on proto/markdown.tmpl,
and generated with hack/generate_protos.sh!
Please edit the template file and not the markdown one directly!

******
-->
This is a generated reference for the [Skaffold API]({{<relref "/docs/design/api">}}) gRPC layer.

We also generate the [reference doc for the HTTP layer]({{<relref "/docs/references/api/swagger">}}).



<a name="skaffold.proto"></a>

## skaffold.proto

You can find the source for skaffold.proto [on Github](https://github.com/GoogleContainerTools/skaffold/blob/master/proto/skaffold.proto).



### Services

<a name="proto.SkaffoldService"></a>

#### SkaffoldService
Describes all the methods for the Skaffold API

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetState | [.google.protobuf.Empty](#google.protobuf.Empty) | [State](#proto.State) | Returns the state of the current Skaffold execution |
| EventLog | [LogEntry](#proto.LogEntry) stream | [LogEntry](#proto.LogEntry) stream | DEPRECATED. Events should be used instead. TODO remove (https://github.com/GoogleContainerTools/skaffold/issues/3168) |
| Events | [.google.protobuf.Empty](#google.protobuf.Empty) | [LogEntry](#proto.LogEntry) stream | Returns all the events of the current Skaffold execution from the start |
| Execute | [UserIntentRequest](#proto.UserIntentRequest) | [.google.protobuf.Empty](#google.protobuf.Empty) | Allows for a single execution of some or all of the phases (build, sync, deploy) in case autoBuild, autoDeploy or autoSync are disabled. |
| Handle | [Event](#proto.Event) | [.google.protobuf.Empty](#google.protobuf.Empty) | EXPERIMENTAL. It allows for custom events to be implemented in custom builders for example. |

 <!-- end services -->


### Data types



<a name="proto.BuildEvent"></a>
#### BuildEvent
`BuildEvent` describes the build status per artifact, and will be emitted by Skaffold anytime a build starts or finishes, successfully or not.
If the build fails, an error will be attached to the event.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| artifact | [string](#string) |  | artifact name |
| status | [string](#string) |  | artifact build status oneof: InProgress, Completed, Failed |
| err | [string](#string) |  | error when build status is Failed. |







<a name="proto.BuildState"></a>
#### BuildState
`BuildState` maps Skaffold artifacts to their current build states


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| artifacts | [BuildState.ArtifactsEntry](#proto.BuildState.ArtifactsEntry) | repeated | A map of `artifact name -> build-state`. Artifact name is defined in the `skaffold.yaml`. The `build-state` can be: <br> - `"Not started"`: not yet started <br> - `"In progress"`: build started <br> - `"Complete"`: build succeeded <br> - `"Failed"`: build failed |







<a name="proto.BuildState.ArtifactsEntry"></a>
#### BuildState.ArtifactsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |







<a name="proto.DebuggingContainerEvent"></a>
#### DebuggingContainerEvent
DebuggingContainerEvent is raised when a debugging container is started or terminated


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  | the container status oneof: Started, Terminated |
| podName | [string](#string) |  | the pod name with the debugging container |
| containerName | [string](#string) |  | the name of the container configured for debugging |
| namespace | [string](#string) |  | the namespace of the debugging container |
| artifact | [string](#string) |  | the corresponding artifact's image name |
| runtime | [string](#string) |  | the detected language runtime |
| workingDir | [string](#string) |  | the working directory in the container image |
| debugPorts | [DebuggingContainerEvent.DebugPortsEntry](#proto.DebuggingContainerEvent.DebugPortsEntry) | repeated | the exposed debugging-related ports |







<a name="proto.DebuggingContainerEvent.DebugPortsEntry"></a>
#### DebuggingContainerEvent.DebugPortsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [uint32](#uint32) |  |  |







<a name="proto.DeployEvent"></a>
#### DeployEvent
`DeployEvent` represents the status of a deployment, and is emitted by Skaffold
anytime a deployment starts or completes, successfully or not.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  | deployment status oneof: InProgress, Completed, Failed |
| err | [string](#string) |  | error when status is Failed |







<a name="proto.DeployState"></a>
#### DeployState
`DeployState` describes the status of the current deploy


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  |  |







<a name="proto.Event"></a>
#### Event
`Event` describes an event in the Skaffold process.
It is one of MetaEvent, BuildEvent, DeployEvent, PortEvent, StatusCheckEvent, ResourceStatusCheckEvent, FileSyncEvent, or DebuggingContainerEvent.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metaEvent | [MetaEvent](#proto.MetaEvent) |  | contains general information regarding Skaffold like version info |
| buildEvent | [BuildEvent](#proto.BuildEvent) |  | describes if the build status per artifact. Status could be one of "InProgress", "Completed" or "Failed". |
| deployEvent | [DeployEvent](#proto.DeployEvent) |  | describes if the deployment has started, is in progress or is complete. |
| portEvent | [PortEvent](#proto.PortEvent) |  | describes each port forwarding event. |
| statusCheckEvent | [StatusCheckEvent](#proto.StatusCheckEvent) |  | describes if the Status check has started, is in progress, has succeeded or failed. |
| resourceStatusCheckEvent | [ResourceStatusCheckEvent](#proto.ResourceStatusCheckEvent) |  | indicates progress for each kubernetes deployment. |
| fileSyncEvent | [FileSyncEvent](#proto.FileSyncEvent) |  | describes the sync status. |
| debuggingContainerEvent | [DebuggingContainerEvent](#proto.DebuggingContainerEvent) |  | describes the appearance or disappearance of a debugging container |







<a name="proto.FileSyncEvent"></a>
#### FileSyncEvent
FileSyncEvent describes the sync status.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| fileCount | [int32](#int32) |  | number of files synced |
| image | [string](#string) |  | the container image to which files are sycned. |
| status | [string](#string) |  | status of file sync. one of: Not Started, In progress, Succeeded, Failed. |
| err | [string](#string) |  | error in case of status failed. |







<a name="proto.FileSyncState"></a>
#### FileSyncState
`FileSyncState` contains the status of the current file sync


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  |  |







<a name="proto.Intent"></a>
#### Intent
Intent represents user intents for a given phase to be unblocked, once.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| build | [bool](#bool) |  | in case skaffold dev is ran with autoBuild=false, a build intent enables building once |
| sync | [bool](#bool) |  | in case skaffold dev is ran with autoSync=false, a sync intent enables file sync once |
| deploy | [bool](#bool) |  | in case skaffold dev is ran with autoDeploy=false, a deploy intent enables deploys once |







<a name="proto.LogEntry"></a>
#### LogEntry
LogEntry describes an event and a string description of the event.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| timestamp | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  | timestamp of the event. |
| event | [Event](#proto.Event) |  | the actual event that is one of |
| entry | [string](#string) |  | description of the event. |







<a name="proto.MetaEvent"></a>
#### MetaEvent
`MetaEvent` provides general information regarding Skaffold


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entry | [string](#string) |  | entry, for example: `"Starting Skaffold: {Version:v0.39.0-16-g5bb7c9e0 ConfigVersion:skaffold/v1 GitVersion: GitCommit:5bb7c9e078e4d522a5ffc42a2f1274fd17d75902 GitTreeState:dirty BuildDate01:29Z GoVersion:go1.13rc1 Compiler:gc Platform:linux/amd64}"` |







<a name="proto.PortEvent"></a>
#### PortEvent
PortEvent Event describes each port forwarding event.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| localPort | [int32](#int32) |  | local port for forwarded resource |
| remotePort | [int32](#int32) |  | remote port is the resource port that will be forwarded. |
| podName | [string](#string) |  | pod name if port forwarded resourceType is Pod |
| containerName | [string](#string) |  | container name if specified in the kubernetes spec |
| namespace | [string](#string) |  | the namespace of the resource to port forward. |
| portName | [string](#string) |  |  |
| resourceType | [string](#string) |  | resource type e.g. "pod", "service". |
| resourceName | [string](#string) |  | name of the resource to forward. |
| address | [string](#string) |  | address on which to bind |







<a name="proto.Request"></a>
#### Request



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |







<a name="proto.ResourceStatusCheckEvent"></a>
#### ResourceStatusCheckEvent
A Resource StatusCheck Event, indicates progress for each kubernetes deployment.
For every resource, there will be exactly one event with `status` *Succeeded* or *Failed* event.
There can be multiple events with `status` *Pending*.
Skaffold polls for resource status every 0.5 second. If the resource status changes, an event with `status` “Pending”, “Complete” and “Failed”
will be sent with the new status.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource | [string](#string) |  |  |
| status | [string](#string) |  |  |
| message | [string](#string) |  |  |
| err | [string](#string) |  |  |







<a name="proto.Response"></a>
#### Response



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| msg | [string](#string) |  |  |







<a name="proto.State"></a>
#### State
`State` represents the current state of the Skaffold components


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| buildState | [BuildState](#proto.BuildState) |  |  |
| deployState | [DeployState](#proto.DeployState) |  |  |
| forwardedPorts | [State.ForwardedPortsEntry](#proto.State.ForwardedPortsEntry) | repeated |  |
| statusCheckState | [StatusCheckState](#proto.StatusCheckState) |  |  |
| fileSyncState | [FileSyncState](#proto.FileSyncState) |  |  |
| debuggingContainers | [DebuggingContainerEvent](#proto.DebuggingContainerEvent) | repeated |  |







<a name="proto.State.ForwardedPortsEntry"></a>
#### State.ForwardedPortsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [int32](#int32) |  |  |
| value | [PortEvent](#proto.PortEvent) |  |  |







<a name="proto.StateResponse"></a>
#### StateResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| state | [State](#proto.State) |  |  |







<a name="proto.StatusCheckEvent"></a>
#### StatusCheckEvent
`StatusCheckEvent` describes if the status check for kubernetes rollout has started, is in progress, has succeeded or failed.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  |  |
| message | [string](#string) |  |  |
| err | [string](#string) |  |  |







<a name="proto.StatusCheckState"></a>
#### StatusCheckState
`StatusCheckState` describes the state of status check of current deployed resources.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  |  |
| resources | [StatusCheckState.ResourcesEntry](#proto.StatusCheckState.ResourcesEntry) | repeated | A map of `resource name -> status-check-state`. Where `resource-name` is the kubernetes resource name. The `status-check-state` can be <br> - `"Not started"`: indicates that `status-check` has just started. <br> - `"In progress"`: InProgress is sent after every resource check is complete. <br> - `"Succeeded"`: - `"Failed"`: |







<a name="proto.StatusCheckState.ResourcesEntry"></a>
#### StatusCheckState.ResourcesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |







<a name="proto.UserIntentRequest"></a>
#### UserIntentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| intent | [Intent](#proto.Intent) |  |  |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->



