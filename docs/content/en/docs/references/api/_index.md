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



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| artifact | [string](#string) |  |  |
| status | [string](#string) |  |  |
| err | [string](#string) |  |  |






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



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  |  |
| err | [string](#string) |  |  |






<a name="proto.DeployState"></a>

### DeployState
DeployState contains the status of the current deploy


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  |  |






<a name="proto.Event"></a>

### Event



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metaEvent | [MetaEvent](#proto.MetaEvent) |  |  |
| buildEvent | [BuildEvent](#proto.BuildEvent) |  |  |
| deployEvent | [DeployEvent](#proto.DeployEvent) |  |  |
| portEvent | [PortEvent](#proto.PortEvent) |  |  |
| statusCheckEvent | [StatusCheckEvent](#proto.StatusCheckEvent) |  |  |
| resourceStatusCheckEvent | [ResourceStatusCheckEvent](#proto.ResourceStatusCheckEvent) |  |  |
| fileSyncEvent | [FileSyncEvent](#proto.FileSyncEvent) |  |  |






<a name="proto.FileSyncEvent"></a>

### FileSyncEvent



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| fileCount | [int32](#int32) |  |  |
| image | [string](#string) |  |  |
| status | [string](#string) |  |  |
| err | [string](#string) |  |  |






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



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| timestamp | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  |  |
| event | [Event](#proto.Event) |  |  |
| entry | [string](#string) |  |  |






<a name="proto.MetaEvent"></a>

### MetaEvent



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entry | [string](#string) |  |  |






<a name="proto.PortEvent"></a>

### PortEvent



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| localPort | [int32](#int32) |  |  |
| remotePort | [int32](#int32) |  |  |
| podName | [string](#string) |  |  |
| containerName | [string](#string) |  |  |
| namespace | [string](#string) |  |  |
| portName | [string](#string) |  |  |
| resourceType | [string](#string) |  |  |
| resourceName | [string](#string) |  |  |






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

