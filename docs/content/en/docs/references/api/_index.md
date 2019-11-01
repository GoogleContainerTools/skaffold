# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [skaffold.proto](#skaffold.proto)
    - [BuildEvent](#proto.BuildEvent)
    - [BuildState](#proto.BuildState)
    - [BuildState.ArtifactsEntry](#proto.BuildState.ArtifactsEntry)
    - [DeployEvent](#proto.DeployEvent)
    - [DeployState](#proto.DeployState)
    - [Event](#proto.Event)
    - [FileSyncEvent](#proto.FileSyncEvent)
    - [FileSyncState](#proto.FileSyncState)
    - [Intent](#proto.Intent)
    - [LogEntry](#proto.LogEntry)
    - [MetaEvent](#proto.MetaEvent)
    - [PortEvent](#proto.PortEvent)
    - [Request](#proto.Request)
    - [ResourceStatusCheckEvent](#proto.ResourceStatusCheckEvent)
    - [Response](#proto.Response)
    - [State](#proto.State)
    - [State.ForwardedPortsEntry](#proto.State.ForwardedPortsEntry)
    - [StateResponse](#proto.StateResponse)
    - [StatusCheckEvent](#proto.StatusCheckEvent)
    - [StatusCheckState](#proto.StatusCheckState)
    - [StatusCheckState.ResourcesEntry](#proto.StatusCheckState.ResourcesEntry)
    - [UserIntentRequest](#proto.UserIntentRequest)
  
  
  
    - [SkaffoldService](#proto.SkaffoldService)
  

- [Scalar Value Types](#scalar-value-types)



<a name="skaffold.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## skaffold.proto



<a name="proto.BuildEvent"></a>

### BuildEvent
BuildEvent describes the build status per artifact, and will be emitted by Skaffold anytime a build starts or finishes, successfully or not.
If the build fails, an error will be attached to the event.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| artifact | [string](#string) |  | artifact name |
| status | [string](#string) |  | artifact build status oneof: InProgress, Completed, Failed |
| err | [string](#string) |  | error when build status is Failed. |






<a name="proto.BuildState"></a>

### BuildState
BuildState contains a map of all skaffold artifacts to their current build
states


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| artifacts | [BuildState.ArtifactsEntry](#proto.BuildState.ArtifactsEntry) | repeated |  |






<a name="proto.BuildState.ArtifactsEntry"></a>

### BuildState.ArtifactsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="proto.DeployEvent"></a>

### DeployEvent
DeployEvent gives the status of a deployment, and will be emitted by Skaffold
anytime a deployment starts or completes, successfully or not.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  | deployment status oneof: InProgress, Completed, Failed |
| err | [string](#string) |  | error when status is Failed |






<a name="proto.DeployState"></a>

### DeployState
DeployState contains the status of the current deploy


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  |  |






<a name="proto.Event"></a>

### Event
Event is one of the following events.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metaEvent | [MetaEvent](#proto.MetaEvent) |  | contains general information regarding Skaffold like version info |
| buildEvent | [BuildEvent](#proto.BuildEvent) |  | describes if the build status per artifact. Status could be one of &#34;InProgress&#34;, &#34;Completed&#34; or &#34;Failed&#34;. |
| deployEvent | [DeployEvent](#proto.DeployEvent) |  | describes if the deployment has started, is in progress or is complete. |
| portEvent | [PortEvent](#proto.PortEvent) |  | describes each port forwarding event. |
| statusCheckEvent | [StatusCheckEvent](#proto.StatusCheckEvent) |  | describes if the Status check has started, is in progress, has succeeded or failed. |
| resourceStatusCheckEvent | [ResourceStatusCheckEvent](#proto.ResourceStatusCheckEvent) |  | indicates progress for each kubernetes deployment. |
| fileSyncEvent | [FileSyncEvent](#proto.FileSyncEvent) |  | describes the sync status. |






<a name="proto.FileSyncEvent"></a>

### FileSyncEvent
FileSyncEvent describes the sync status.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| fileCount | [int32](#int32) |  | number of files synced |
| image | [string](#string) |  | the container image to which files are sycned. |
| status | [string](#string) |  | status of file sync. one of: Not Started, In progress, Succeeded, Failed. |
| err | [string](#string) |  | error in case of status failed. |






<a name="proto.FileSyncState"></a>

### FileSyncState
FileSyncState contains the status of the current file sync


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  |  |






<a name="proto.Intent"></a>

### Intent



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| build | [bool](#bool) |  |  |
| sync | [bool](#bool) |  |  |
| deploy | [bool](#bool) |  |  |






<a name="proto.LogEntry"></a>

### LogEntry
LogEntry describes an event and a string description of the event.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| timestamp | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  | timestamp of the event. |
| event | [Event](#proto.Event) |  | Event |
| entry | [string](#string) |  | description of the event. |






<a name="proto.MetaEvent"></a>

### MetaEvent
MetaEvent gives general information regarding Skaffold like version info


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entry | [string](#string) |  |  |






<a name="proto.PortEvent"></a>

### PortEvent
PortEvent Event describes each port forwarding event.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| localPort | [int32](#int32) |  | local port for forwarded resource |
| remotePort | [int32](#int32) |  | remote port is the resource port that will be forwarded. |
| podName | [string](#string) |  | pod name if port forwarded resourceType is Pod |
| containerName | [string](#string) |  | container name if specified in the kubernetes spec |
| namespace | [string](#string) |  | the namespace of the resource to port forward. |
| portName | [string](#string) |  |  |
| resourceType | [string](#string) |  | resource type e.g. &#34;pod&#34;, &#34;service&#34;. |
| resourceName | [string](#string) |  | name of the resource to forward. |






<a name="proto.Request"></a>

### Request



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |






<a name="proto.ResourceStatusCheckEvent"></a>

### ResourceStatusCheckEvent



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource | [string](#string) |  |  |
| status | [string](#string) |  |  |
| message | [string](#string) |  |  |
| err | [string](#string) |  |  |






<a name="proto.Response"></a>

### Response



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| msg | [string](#string) |  |  |






<a name="proto.State"></a>

### State
State represents the current state of the Skaffold components


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| buildState | [BuildState](#proto.BuildState) |  |  |
| deployState | [DeployState](#proto.DeployState) |  |  |
| forwardedPorts | [State.ForwardedPortsEntry](#proto.State.ForwardedPortsEntry) | repeated |  |
| statusCheckState | [StatusCheckState](#proto.StatusCheckState) |  |  |
| fileSyncState | [FileSyncState](#proto.FileSyncState) |  |  |






<a name="proto.State.ForwardedPortsEntry"></a>

### State.ForwardedPortsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [int32](#int32) |  |  |
| value | [PortEvent](#proto.PortEvent) |  |  |






<a name="proto.StateResponse"></a>

### StateResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| state | [State](#proto.State) |  |  |






<a name="proto.StatusCheckEvent"></a>

### StatusCheckEvent
StatusCheck Event describes if the Status check has started, is in progress, has succeeded or failed.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  |  |
| message | [string](#string) |  |  |
| err | [string](#string) |  |  |






<a name="proto.StatusCheckState"></a>

### StatusCheckState
StatusCheckState contains the state of status check of current deployed resources.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  |  |
| resources | [StatusCheckState.ResourcesEntry](#proto.StatusCheckState.ResourcesEntry) | repeated |  |






<a name="proto.StatusCheckState.ResourcesEntry"></a>

### StatusCheckState.ResourcesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="proto.UserIntentRequest"></a>

### UserIntentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| intent | [Intent](#proto.Intent) |  |  |





 

 

 


<a name="proto.SkaffoldService"></a>

### SkaffoldService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetState | [.google.protobuf.Empty](#google.protobuf.Empty) | [State](#proto.State) |  |
| EventLog | [LogEntry](#proto.LogEntry) stream | [LogEntry](#proto.LogEntry) stream |  |
| Events | [LogEntry](#proto.LogEntry) stream | [LogEntry](#proto.LogEntry) stream |  |
| Handle | [Event](#proto.Event) | [.google.protobuf.Empty](#google.protobuf.Empty) |  |
| Execute | [UserIntentRequest](#proto.UserIntentRequest) | [.google.protobuf.Empty](#google.protobuf.Empty) |  |

 



## Scalar Value Types

| .proto Type | Notes | C++ Type | Java Type | Python Type |
| ----------- | ----- | -------- | --------- | ----------- |
| <a name="double" /> double |  | double | double | float |
| <a name="float" /> float |  | float | float | float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long |
| <a name="bool" /> bool |  | bool | boolean | boolean |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str |

