---
title: "gRPC API"
linkTitle: "gRPC API"
weight: 30
---
<!--
******
WARNING!!!

The file docs-v1/content/en/docs/references/api/grpc.md is generated based on proto/markdown.tmpl,
and generated with ./hack/generate_proto.sh!
Please edit the template file and not the markdown one directly!

******
-->
This is a generated reference for the [Skaffold API]({{<relref "/docs/design/api">}}) gRPC layer.

We also generate the [reference doc for the HTTP layer]({{<relref "/docs/references/api/swagger">}}).



<a name="v1/skaffold.proto"></a>

## v1/skaffold.proto

You can find the source for v1/skaffold.proto [on Github](https://github.com/GoogleContainerTools/skaffold/blob/main/proto/v1/skaffold.proto).



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
| AutoBuild | [TriggerRequest](#proto.TriggerRequest) | [.google.protobuf.Empty](#google.protobuf.Empty) | Allows for enabling or disabling automatic build trigger |
| AutoSync | [TriggerRequest](#proto.TriggerRequest) | [.google.protobuf.Empty](#google.protobuf.Empty) | Allows for enabling or disabling automatic sync trigger |
| AutoDeploy | [TriggerRequest](#proto.TriggerRequest) | [.google.protobuf.Empty](#google.protobuf.Empty) | Allows for enabling or disabling automatic deploy trigger |
| Handle | [Event](#proto.Event) | [.google.protobuf.Empty](#google.protobuf.Empty) | EXPERIMENTAL. It allows for custom events to be implemented in custom builders for example. |

 <!-- end services -->


### Data types



<a name="proto.ActionableErr"></a>
#### ActionableErr
`ActionableErr` defines an error that occurred along with an optional list of suggestions


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| errCode | [enums.StatusCode](#proto.enums.StatusCode) |  | error code representing the error |
| message | [string](#string) |  | message describing the error. |
| suggestions | [Suggestion](#proto.Suggestion) | repeated | list of suggestions |







<a name="proto.BuildEvent"></a>
#### BuildEvent
`BuildEvent` describes the build status per artifact, and will be emitted by Skaffold anytime a build starts or finishes, successfully or not.
If the build fails, an error will be attached to the event.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| artifact | [string](#string) |  | artifact name |
| status | [string](#string) |  | artifact build status oneof: InProgress, Completed, Failed |
| err | [string](#string) |  | Deprecated. Use actionableErr.message. error when build status is Failed. |
| errCode | [enums.StatusCode](#proto.enums.StatusCode) |  | Deprecated. Use actionableErr.errCode. status code representing success or failure |
| actionableErr | [ActionableErr](#proto.ActionableErr) |  | actionable error message |
| hostPlatform | [string](#string) |  | architecture of the host machine. For example `linux/amd64` |
| targetPlatforms | [string](#string) |  | comma-delimited list of build target architectures. For example `linux/amd64,linux/arm64` |







<a name="proto.BuildMetadata"></a>
#### BuildMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| numberOfArtifacts | [int32](#int32) |  |  |
| builders | [BuildMetadata.ImageBuilder](#proto.BuildMetadata.ImageBuilder) | repeated |  |
| type | [enums.BuildType](#proto.enums.BuildType) |  |  |
| additional | [BuildMetadata.AdditionalEntry](#proto.BuildMetadata.AdditionalEntry) | repeated | Additional key value pairs to describe the deploy pipeline |







<a name="proto.BuildMetadata.AdditionalEntry"></a>
#### BuildMetadata.AdditionalEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |







<a name="proto.BuildMetadata.ImageBuilder"></a>
#### BuildMetadata.ImageBuilder



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [enums.BuilderType](#proto.enums.BuilderType) |  |  |
| count | [int32](#int32) |  |  |







<a name="proto.BuildState"></a>
#### BuildState
`BuildState` maps Skaffold artifacts to their current build states


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| artifacts | [BuildState.ArtifactsEntry](#proto.BuildState.ArtifactsEntry) | repeated | A map of `artifact name -> build-state`. Artifact name is defined in the `skaffold.yaml`. The `build-state` can be: <br> - `"Not started"`: not yet started <br> - `"In progress"`: build started <br> - `"Complete"`: build succeeded <br> - `"Failed"`: build failed |
| autoTrigger | [bool](#bool) |  |  |
| statusCode | [enums.StatusCode](#proto.enums.StatusCode) |  |  |







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
| err | [string](#string) |  | Deprecated. Use actionableErr.message. error when status is Failed |
| errCode | [enums.StatusCode](#proto.enums.StatusCode) |  | Deprecated. Use actionableErr.errCode. status code representing success or failure |
| actionableErr | [ActionableErr](#proto.ActionableErr) |  | actionable error message |







<a name="proto.DeployMetadata"></a>
#### DeployMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| deployers | [DeployMetadata.Deployer](#proto.DeployMetadata.Deployer) | repeated |  |
| cluster | [enums.ClusterType](#proto.enums.ClusterType) |  |  |







<a name="proto.DeployMetadata.Deployer"></a>
#### DeployMetadata.Deployer



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [enums.DeployerType](#proto.enums.DeployerType) |  |  |
| count | [int32](#int32) |  |  |







<a name="proto.DeployState"></a>
#### DeployState
`DeployState` describes the status of the current deploy


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  |  |
| autoTrigger | [bool](#bool) |  |  |
| statusCode | [enums.StatusCode](#proto.enums.StatusCode) |  |  |







<a name="proto.DevLoopEvent"></a>
#### DevLoopEvent
`DevLoopEvent` marks the start and end of a dev loop.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| iteration | [int32](#int32) |  | dev loop iteration. 0 represents initialization loop. |
| status | [string](#string) |  | dev loop status oneof: In Progress, Completed, Failed |
| err | [ActionableErr](#proto.ActionableErr) |  | actionable error message |







<a name="proto.Event"></a>
#### Event
`Event` describes an event in the Skaffold process.
It is one of MetaEvent, BuildEvent, TestEvent, DeployEvent, PortEvent, StatusCheckEvent, ResourceStatusCheckEvent, FileSyncEvent, or DebuggingContainerEvent.


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
| devLoopEvent | [DevLoopEvent](#proto.DevLoopEvent) |  | describes a start and end of a dev loop. |
| terminationEvent | [TerminationEvent](#proto.TerminationEvent) |  | describes a skaffold termination event |
| TestEvent | [TestEvent](#proto.TestEvent) |  | describes if the test has started, is in progress or is complete. |







<a name="proto.FileSyncEvent"></a>
#### FileSyncEvent
FileSyncEvent describes the sync status.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| fileCount | [int32](#int32) |  | number of files synced |
| image | [string](#string) |  | the container image to which files are sycned. |
| status | [string](#string) |  | status of file sync. one of: Not Started, In progress, Succeeded, Failed. |
| err | [string](#string) |  | Deprecated. Use actionableErr.message. error in case of status failed. |
| errCode | [enums.StatusCode](#proto.enums.StatusCode) |  | Deprecated. Use actionableErr.errCode. status code representing success or failure |
| actionableErr | [ActionableErr](#proto.ActionableErr) |  | actionable error message |







<a name="proto.FileSyncState"></a>
#### FileSyncState
`FileSyncState` contains the status of the current file sync


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  |  |
| autoTrigger | [bool](#bool) |  |  |







<a name="proto.IntOrString"></a>
#### IntOrString
IntOrString is a type that can hold an int32 or a string.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [int32](#int32) |  | type of stored value |
| intVal | [int32](#int32) |  | int value |
| strVal | [string](#string) |  | string value |







<a name="proto.Intent"></a>
#### Intent
Intent represents user intents for a given phase.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| build | [bool](#bool) |  | in case skaffold dev is ran with autoBuild=false, a build intent enables building once |
| sync | [bool](#bool) |  | in case skaffold dev is ran with autoSync=false, a sync intent enables file sync once |
| deploy | [bool](#bool) |  | in case skaffold dev is ran with autoDeploy=false, a deploy intent enables deploys once |
| devloop | [bool](#bool) |  | in case skaffold dev is ran with autoDeploy=false, autoSync=false and autoBuild=false a devloop intent enables the entire dev loop once |







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
| metadata | [Metadata](#proto.Metadata) |  | Metadata describing skaffold pipeline |







<a name="proto.Metadata"></a>
#### Metadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| build | [BuildMetadata](#proto.BuildMetadata) |  |  |
| deploy | [DeployMetadata](#proto.DeployMetadata) |  |  |
| test | [TestMetadata](#proto.TestMetadata) |  |  |
| additional | [Metadata.AdditionalEntry](#proto.Metadata.AdditionalEntry) | repeated | Additional key value pairs to describe the build pipeline |







<a name="proto.Metadata.AdditionalEntry"></a>
#### Metadata.AdditionalEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |







<a name="proto.PortEvent"></a>
#### PortEvent
PortEvent Event describes each port forwarding event.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| localPort | [int32](#int32) |  | local port for forwarded resource |
| remotePort | [int32](#int32) |  | Deprecated. Uses targetPort.intVal. |
| podName | [string](#string) |  | pod name if port forwarded resourceType is Pod |
| containerName | [string](#string) |  | container name if specified in the kubernetes spec |
| namespace | [string](#string) |  | the namespace of the resource to port forward. |
| portName | [string](#string) |  |  |
| resourceType | [string](#string) |  | resource type e.g. "pod", "service". |
| resourceName | [string](#string) |  | name of the resource to forward. |
| address | [string](#string) |  | address on which to bind |
| targetPort | [IntOrString](#proto.IntOrString) |  | target port is the resource port that will be forwarded. |







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
| err | [string](#string) |  | Deprecated. Use actionableErr.message. |
| statusCode | [enums.StatusCode](#proto.enums.StatusCode) |  |  |
| actionableErr | [ActionableErr](#proto.ActionableErr) |  | actionable error message |







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
| metadata | [Metadata](#proto.Metadata) |  |  |
| testState | [TestState](#proto.TestState) |  |  |







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
| err | [string](#string) |  | Deprecated. Use actionableErr.message. |
| errCode | [enums.StatusCode](#proto.enums.StatusCode) |  | Deprecated. Use actionableErr.errCode. status code representing success or failure |
| actionableErr | [ActionableErr](#proto.ActionableErr) |  | actionable error message |







<a name="proto.StatusCheckState"></a>
#### StatusCheckState
`StatusCheckState` describes the state of status check of current deployed resources.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  |  |
| resources | [StatusCheckState.ResourcesEntry](#proto.StatusCheckState.ResourcesEntry) | repeated | A map of `resource name -> status-check-state`. Where `resource-name` is the kubernetes resource name. The `status-check-state` can be <br> - `"Not started"`: indicates that `status-check` has just started. <br> - `"In progress"`: InProgress is sent after every resource check is complete. <br> - `"Succeeded"`: - `"Failed"`: |
| statusCode | [enums.StatusCode](#proto.enums.StatusCode) |  | StatusCheck statusCode |







<a name="proto.StatusCheckState.ResourcesEntry"></a>
#### StatusCheckState.ResourcesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |







<a name="proto.Suggestion"></a>
#### Suggestion
Suggestion defines the action a user needs to recover from an error.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| suggestionCode | [enums.SuggestionCode](#proto.enums.SuggestionCode) |  | code representing a suggestion |
| action | [string](#string) |  | action represents the suggestion action |







<a name="proto.TerminationEvent"></a>
#### TerminationEvent
`TerminationEvent` marks the end of the skaffold session


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  | status oneof: Completed or Failed |
| err | [ActionableErr](#proto.ActionableErr) |  | actionable error message |







<a name="proto.TestEvent"></a>
#### TestEvent
`TestEvent` represents the status of a test, and is emitted by Skaffold
anytime a test starts or completes, successfully or not.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  | test status oneof: InProgress, Completed, Failed |
| actionableErr | [ActionableErr](#proto.ActionableErr) |  | actionable error message |







<a name="proto.TestMetadata"></a>
#### TestMetadata
TestMetadata describes the test pipeline


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| Testers | [TestMetadata.Tester](#proto.TestMetadata.Tester) | repeated |  |







<a name="proto.TestMetadata.Tester"></a>
#### TestMetadata.Tester



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [enums.TesterType](#proto.enums.TesterType) |  |  |
| count | [int32](#int32) |  |  |







<a name="proto.TestState"></a>
#### TestState
`TestState` describes the current state of the test


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  | Status of the current test |
| statusCode | [enums.StatusCode](#proto.enums.StatusCode) |  | Teststate status code |







<a name="proto.TriggerRequest"></a>
#### TriggerRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| state | [TriggerState](#proto.TriggerState) |  |  |







<a name="proto.TriggerState"></a>
#### TriggerState
TriggerState represents trigger state for a given phase.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  | enable or disable a trigger state |







<a name="proto.UserIntentRequest"></a>
#### UserIntentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| intent | [Intent](#proto.Intent) |  |  |





 <!-- end messages -->

 <!-- end HasExtensions -->





<a name="proto.enums.BuildType"></a>

### BuildType
Enum indicating build type i.e. local, cluster vs GCB

| Name | Number | Description |
| ---- |:------:| ----------- |
| UNKNOWN_BUILD_TYPE | 0 | Could not determine Build Type |
| CLUSTER | 1 | Cluster Build |
| GCB | 2 | GCB Build |
| LOCAL | 3 | Local Build |



<a name="proto.enums.BuilderType"></a>

### BuilderType
Enum indicating builders used

| Name | Number | Description |
| ---- |:------:| ----------- |
| UNKNOWN_BUILDER_TYPE | 0 | Could not determine builder type |
| JIB | 1 | JIB Builder |
| BAZEL | 2 | Bazel Builder |
| BUILDPACKS | 3 | Buildpacks Builder |
| CUSTOM | 4 | Custom Builder |
| KANIKO | 5 | Kaniko Builder |
| DOCKER | 6 | Docker Builder |
| KO | 7 | Ko Builder |



<a name="proto.enums.ClusterType"></a>

### ClusterType
Enum indicating cluster type the application is deployed to

| Name | Number | Description |
| ---- |:------:| ----------- |
| UNKNOWN_CLUSTER_TYPE | 0 | Could not determine Cluster Type |
| MINIKUBE | 1 | Minikube Cluster |
| GKE | 2 | GKE cluster |
| OTHER | 3 | All Cluster except Minikube and GKE |



<a name="proto.enums.DeployerType"></a>

### DeployerType
Enum indicating deploy tools used

| Name | Number | Description |
| ---- |:------:| ----------- |
| UNKNOWN_DEPLOYER_TYPE | 0 | Could not determine Deployer Type |
| HELM | 1 | Helm Deployer |
| KUSTOMIZE | 2 | Kustomize Deployer |
| KUBECTL | 3 | Kubectl Deployer |
| KPT | 4 | kpt Deployer |



<a name="proto.enums.LogLevel"></a>

### LogLevel
Enum indicating the log level of a line of output

| Name | Number | Description |
| ---- |:------:| ----------- |
| DEBUG | 0 | Debug Level |
| INFO | 1 | Info Level |
| WARN | 2 | Warn Level |
| ERROR | 3 | Error Level |
| FATAL | 4 | Fatal Level |
| PANIC | 5 | Panic Level |
| TRACE | 6 | Trace Level |
| STANDARD | 7 | User-visible output level |



<a name="proto.enums.RenderType"></a>

### RenderType
Enum indicating render manifests type

| Name | Number | Description |
| ---- |:------:| ----------- |
| UNKNOWN_RENDER_TYPE | 0 | Could not determine Render Type |
| RAWK8S | 1 | Raw Manifests |
| KUSTOMIZE_MANIFEST | 2 | kustomize manifests |
| HELM_CHART | 3 | helm charts |
| KPT_MANIFEST | 4 | kpt manifests |



<a name="proto.enums.StatusCode"></a>

### StatusCode
Enum for Status codes<br>
These error codes are prepended by Phase Name e.g.
INIT, BUILD, TEST, DEPLOY, STATUSCHECK, DEVINIT<br>
For Success Error codes, use range 200 to 250.<br>
For Unknown error codes, use range 500 to 600.<br>
For Cancelled Error code, use range 800 to 850.<br>

| Name | Number | Description |
| ---- |:------:| ----------- |
| OK | 0 | A default status code for events that do not have an associated phase. Typically seen with the DevEndEvent event on success. |
| STATUSCHECK_SUCCESS | 200 | Status Check Success |
| BUILD_SUCCESS | 201 | Build Success |
| RENDER_SUCCESS | 204 | Render Success |
| DEPLOY_SUCCESS | 202 | Deploy Success |
| TEST_SUCCESS | 203 | Test Success |
| BUILD_PUSH_ACCESS_DENIED | 101 | Build error due to push access denied |
| BUILD_PROJECT_NOT_FOUND | 102 | Build error due to GCP project not found. |
| BUILD_DOCKER_DAEMON_NOT_RUNNING | 103 | Docker build error due to docker daemon not running |
| BUILD_USER_ERROR | 104 | Build error due to user application code, e.g. compilation error, dockerfile error etc |
| BUILD_DOCKER_UNAVAILABLE | 105 | Build error due to docker not available |
| BUILD_DOCKER_UNAUTHORIZED | 106 | Docker build error due to user not authorized to perform the action |
| BUILD_DOCKER_SYSTEM_ERR | 107 | Docker system build error |
| BUILD_DOCKER_NOT_MODIFIED_ERR | 108 | Docker build error due to Docker build container is already in the desired state |
| BUILD_DOCKER_NOT_IMPLEMENTED_ERR | 109 | Docker build error indicating a feature not supported |
| BUILD_DOCKER_DATA_LOSS_ERR | 110 | Docker build error indicates that for given build, data was lost or there is data corruption |
| BUILD_DOCKER_FORBIDDEN_ERR | 111 | Docker build error indicates user is forbidden to perform the build or step/action. |
| BUILD_DOCKER_CONFLICT_ERR | 112 | Docker build error due to some internal error and docker container state conflicts with the requested action and can't be performed |
| BUILD_DOCKER_ERROR_NOT_FOUND | 113 | Docker build error indicates the requested object does not exist |
| BUILD_DOCKER_INVALID_PARAM_ERR | 114 | Docker build error indication invalid parameter sent to docker command |
| BUILD_DOCKERFILE_NOT_FOUND | 115 | Docker build failed due to dockerfile not found |
| BUILD_DOCKER_CACHE_FROM_PULL_ERR | 116 | Docker build failed due `cacheFrom` user config error |
| BUILD_DOCKER_GET_DIGEST_ERR | 117 | Build error due to digest for built artifact could not be retrieved from docker daemon. |
| BUILD_DOCKER_NO_SPACE_ERR | 127 | Build error due no space left in docker. |
| BUILD_REGISTRY_GET_DIGEST_ERR | 118 | Build error due to digest for built artifact could not be retrieved from registry. |
| BUILD_UNKNOWN_JIB_PLUGIN_TYPE | 119 | Build error indicating unknown Jib plugin type. Should be one of [maven, gradle] |
| BUILD_JIB_GRADLE_DEP_ERR | 120 | Build error determining dependency for jib gradle project. |
| BUILD_JIB_MAVEN_DEP_ERR | 121 | Build error determining dependency for jib gradle project. |
| INIT_DOCKER_NETWORK_LISTING_CONTAINERS | 122 | Docker build error when listing containers. |
| INIT_DOCKER_NETWORK_INVALID_CONTAINER_NAME | 123 | Docker build error indicating an invalid container name (or id). |
| INIT_DOCKER_NETWORK_CONTAINER_DOES_NOT_EXIST | 124 | Docker build error indicating the container referenced does not exists in the docker context used. |
| INIT_DOCKER_NETWORK_INVALID_MODE | 125 | Docker Network invalid mode |
| INIT_DOCKER_NETWORK_PARSE_ERR | 126 | Error parsing Docker Network mode |
| BUILD_GCB_CREATE_BUILD_ERR | 128 | GCB Create Build Error |
| BUILD_GCB_GET_BUILD_ID_ERR | 129 | GCB error indicating an error to fetch build id. |
| BUILD_GCB_GET_BUILD_STATUS_ERR | 130 | GCB error indicating an error to fetch build status. |
| BUILD_GCB_GET_BUILD_LOG_ERR | 131 | GCB error indicating an error to fetch build logs. |
| BUILD_GCB_COPY_BUILD_LOG_ERR | 132 | GCB error indicating an error to fetch build status. |
| BUILD_GCB_GET_BUILT_IMAGE_ERR | 133 | GCB error indicating an error retrieving the built image id. |
| BUILD_GCB_BUILD_FAILED | 134 | GCB error indicating build failure. |
| BUILD_GCB_BUILD_INTERNAL_ERR | 135 | GCB error indicating build failure due to internal errror. |
| BUILD_GCB_BUILD_TIMEOUT | 136 | GCB error indicating build failure due to timeout. |
| BUILD_GCB_GENERATE_BUILD_DESCRIPTOR_ERR | 137 | GCB error to generate the build descriptor. |
| BUILD_GCB_UPLOAD_TO_GCS_ERR | 138 | GCB error to upload to GCS. |
| BUILD_GCB_JIB_DEPENDENCY_ERR | 139 | GCB error to fetch jib artifact dependency. |
| BUILD_GCB_GET_DEPENDENCY_ERR | 140 | GCB error to fetch artifact dependency. |
| BUILD_GCB_GET_GCS_BUCKET_ERR | 141 | GCB error to get GCS bucket. |
| BUILD_GCB_CREATE_BUCKET_ERR | 142 | GCB error to create a GCS bucket. |
| BUILD_GCB_EXTRACT_PROJECT_ID | 143 | GCB error to extract Project ID. |
| BUILD_GET_CLOUD_STORAGE_CLIENT_ERR | 144 | GCB error to get cloud storage client to perform GCS operation. |
| BUILD_GET_CLOUD_BUILD_CLIENT_ERR | 145 | GCB error to get cloud build client to perform GCB operations. |
| BUILD_UNKNOWN_PLATFORM_FLAG | 150 | Value provided to --platform flag cannot be parsed |
| BUILD_CROSS_PLATFORM_ERR | 151 | Cross-platform build failures |
| BUILD_CROSS_PLATFORM_NO_REGISTRY_ERR | 152 | Multi-platfor build fails due to no container registry set |
| STATUSCHECK_IMAGE_PULL_ERR | 300 | Container image pull error |
| STATUSCHECK_CONTAINER_CREATING | 301 | Container creating error |
| STATUSCHECK_RUN_CONTAINER_ERR | 302 | Container run error |
| STATUSCHECK_CONTAINER_TERMINATED | 303 | Container is already terminated |
| STATUSCHECK_DEPLOYMENT_ROLLOUT_PENDING | 304 | Deployment waiting for rollout |
| STATUSCHECK_STANDALONE_PODS_PENDING | 305 | Standalone pods pending to stabilize |
| STATUSCHECK_CONTAINER_RESTARTING | 356 | Container restarting error |
| STATUSCHECK_UNHEALTHY | 357 | Readiness probe failed |
| STATUSCHECK_CONTAINER_EXEC_ERROR | 358 | Executable binary format error |
| STATUSCHECK_NODE_MEMORY_PRESSURE | 400 | Node memory pressure error |
| STATUSCHECK_NODE_DISK_PRESSURE | 401 | Node disk pressure error |
| STATUSCHECK_NODE_NETWORK_UNAVAILABLE | 402 | Node network unavailable error |
| STATUSCHECK_NODE_PID_PRESSURE | 403 | Node PID pressure error |
| STATUSCHECK_NODE_UNSCHEDULABLE | 404 | Node unschedulable error |
| STATUSCHECK_NODE_UNREACHABLE | 405 | Node unreachable error |
| STATUSCHECK_NODE_NOT_READY | 406 | Node not ready error |
| STATUSCHECK_FAILED_SCHEDULING | 407 | Scheduler failure error |
| STATUSCHECK_KUBECTL_CONNECTION_ERR | 409 | Kubectl connection error |
| STATUSCHECK_KUBECTL_PID_KILLED | 410 | Kubectl process killed error |
| STATUSCHECK_KUBECTL_CLIENT_FETCH_ERR | 411 | Kubectl client fetch err |
| STATUSCHECK_DEPLOYMENT_FETCH_ERR | 412 |  |
| STATUSCHECK_STANDALONE_PODS_FETCH_ERR | 413 |  |
| STATUSCHECK_CONFIG_CONNECTOR_RESOURCES_FETCH_ERR | 414 |  |
| STATUSCHECK_STATEFULSET_FETCH_ERR | 415 |  |
| STATUSCHECK_CUSTOM_RESOURCE_FETCH_ERR | 416 |  |
| STATUSCHECK_POD_INITIALIZING | 451 | Pod Initializing |
| STATUSCHECK_CONFIG_CONNECTOR_IN_PROGRESS | 452 | The actual state of the resource has not yet reached the desired state |
| STATUSCHECK_CONFIG_CONNECTOR_FAILED | 453 | The process of reconciling the actual state with the desired state has encountered an error |
| STATUSCHECK_CONFIG_CONNECTOR_TERMINATING | 454 | The resource is in the process of being deleted |
| STATUSCHECK_CONFIG_CONNECTOR_NOT_FOUND | 455 | The resource does not exist |
| STATUSCHECK_CUSTOM_RESOURCE_IN_PROGRESS | 456 | The actual state of the resource has not yet reached the desired state |
| STATUSCHECK_CUSTOM_RESOURCE_FAILED | 457 | The process of reconciling the actual state with the desired state has encountered an error |
| STATUSCHECK_CUSTOM_RESOURCE_TERMINATING | 458 | The resource is in the process of being deleted |
| STATUSCHECK_CUSTOM_RESOURCE_NOT_FOUND | 459 | The resource does not exist |
| UNKNOWN_ERROR | 500 | Could not determine error and phase |
| STATUSCHECK_UNKNOWN | 501 | Status Check error unknown |
| STATUSCHECK_UNKNOWN_UNSCHEDULABLE | 502 | Container is unschedulable due to unknown reasons |
| STATUSCHECK_CONTAINER_WAITING_UNKNOWN | 503 | Container is waiting due to unknown reason |
| STATUSCHECK_UNKNOWN_EVENT | 509 | Container event reason unknown |
| STATUSCHECK_INTERNAL_ERROR | 514 | Status Check internal error |
| DEPLOY_UNKNOWN | 504 | Deploy failed due to unknown reason |
| SYNC_UNKNOWN | 505 | SYNC failed due to known reason |
| BUILD_UNKNOWN | 506 | Build failed due to unknown reason |
| DEVINIT_UNKNOWN | 507 | Dev Init failed due to unknown reason |
| CLEANUP_UNKNOWN | 508 | Cleanup failed due to unknown reason |
| INIT_UNKNOWN | 510 | Initialization of the Skaffold session failed due to unknown reason(s) |
| BUILD_DOCKER_UNKNOWN | 511 | Build failed due to docker unknown error |
| TEST_UNKNOWN | 512 | Test failed due to unknown reason |
| BUILD_GCB_BUILD_UNKNOWN_STATUS | 513 | GCB error indicating build failed due to unknown status. |
| SYNC_INIT_ERROR | 601 | File Sync Initialize failure |
| DEVINIT_REGISTER_BUILD_DEPS | 701 | Failed to configure watcher for build dependencies in dev loop |
| DEVINIT_REGISTER_TEST_DEPS | 702 | Failed to configure watcher for test dependencies in dev loop |
| DEVINIT_REGISTER_DEPLOY_DEPS | 703 | Failed to configure watcher for deploy dependencies in dev loop |
| DEVINIT_REGISTER_CONFIG_DEP | 704 | Failed to configure watcher for Skaffold configuration file. |
| DEVINIT_UNSUPPORTED_V1_MANIFEST | 705 | Failed to configure watcher for build dependencies for a base image with v1 manifest. |
| DEVINIT_REGISTER_RENDER_DEPS | 706 | Failed to configure watcher for render dependencies in dev loop |
| STATUSCHECK_USER_CANCELLED | 800 | User cancelled the skaffold dev run |
| STATUSCHECK_DEADLINE_EXCEEDED | 801 | Deadline for status check exceeded |
| BUILD_CANCELLED | 802 | Build Cancelled |
| DEPLOY_CANCELLED | 803 | Deploy cancelled due to user cancellation or one or more deployers failed. |
| BUILD_DOCKER_CANCELLED | 804 | Docker build cancelled. |
| BUILD_DOCKER_DEADLINE | 805 | Build error due to docker deadline was reached before the docker action completed |
| BUILD_GCB_BUILD_CANCELLED | 806 | GCB Build cancelled. |
| INIT_CREATE_TAGGER_ERROR | 901 | Skaffold was unable to create the configured tagger |
| INIT_MINIKUBE_PAUSED_ERROR | 902 | Skaffold was unable to start as Minikube appears to be paused |
| INIT_MINIKUBE_NOT_RUNNING_ERROR | 903 | Skaffold was unable to start as Minikube appears to be stopped |
| INIT_CREATE_BUILDER_ERROR | 904 | Skaffold was unable to create a configured image builder |
| INIT_CREATE_DEPLOYER_ERROR | 905 | Skaffold was unable to create a configured deployer |
| INIT_CREATE_TEST_DEP_ERROR | 906 | Skaffold was unable to create a configured test |
| INIT_CACHE_ERROR | 907 | Skaffold encountered an error validating the artifact cache |
| INIT_CREATE_WATCH_TRIGGER_ERROR | 908 | Skaffold encountered an error when configuring file watching |
| INIT_CREATE_ARTIFACT_DEP_ERROR | 909 | Skaffold encountered an error when evaluating artifact dependencies |
| INIT_CLOUD_RUN_LOCATION_ERROR | 910 | No Location was specified for Cloud Run |
| DEPLOY_CLUSTER_CONNECTION_ERR | 1001 | Unable to connect to cluster |
| DEPLOY_DEBUG_HELPER_RETRIEVE_ERR | 1002 | Could not retrieve debug helpers. |
| DEPLOY_CLEANUP_ERR | 1003 | Deploy clean up error |
| DEPLOY_HELM_APPLY_LABELS | 1004 | Unable to apply helm labels. |
| DEPLOY_HELM_USER_ERR | 1005 | Deploy error due to user deploy config for helm deployer |
| DEPLOY_NO_MATCHING_BUILD | 1006 | An image was referenced with no matching build result |
| DEPLOY_HELM_VERSION_ERR | 1007 | Unable to get helm client version |
| DEPLOY_HELM_MIN_VERSION_ERR | 1008 | Helm version not supported. |
| DEPLOY_KUBECTL_VERSION_ERR | 1109 | Unable to retrieve kubectl version |
| DEPLOY_KUBECTL_OFFLINE_MODE_ERR | 1010 | User specified offline mode for rendering but remote manifests presents. |
| DEPLOY_ERR_WAITING_FOR_DELETION | 1011 | Error waiting for previous version deletion before next version is active. |
| DEPLOY_READ_MANIFEST_ERR | 1012 | Error reading manifests |
| DEPLOY_READ_REMOTE_MANIFEST_ERR | 1013 | Error reading remote manifests |
| DEPLOY_LIST_MANIFEST_ERR | 1014 | Errors listing manifests |
| DEPLOY_KUBECTL_USER_ERR | 1015 | Deploy error due to user deploy config for kubectl deployer |
| DEPLOY_KUSTOMIZE_USER_ERR | 1016 | Deploy error due to user deploy config for kustomize deployer |
| DEPLOY_REPLACE_IMAGE_ERR | 1017 | Error replacing a built artifact in the manifests |
| DEPLOY_TRANSFORM_MANIFEST_ERR | 1018 | Error transforming a manifest during skaffold debug |
| DEPLOY_SET_LABEL_ERR | 1019 | Error setting user specified additional labels. |
| DEPLOY_MANIFEST_WRITE_ERR | 1020 | Error writing hydrated kubernetes manifests. |
| DEPLOY_PARSE_MANIFEST_IMAGES_ERR | 1021 | Error getting images from a kubernetes manifest. |
| DEPLOY_HELM_CREATE_NS_NOT_AVAILABLE | 1022 | Helm config `createNamespace` not available |
| DEPLOY_CLUSTER_INTERNAL_SYSTEM_ERR | 1023 | Kubernetes cluster reported an internal system error |
| DEPLOY_KPTFILE_INIT_ERR | 1024 | The Kptfile cannot be created via `kpt live init`. |
| DEPLOY_KPT_SOURCE_ERR | 1025 | The `kpt fn source` cannot read the given manifests. |
| DEPLOY_KPTFILE_INVALID_YAML_ERR | 1026 | The Kptfile exists but cannot be opened or parsed. |
| DEPLOY_KPT_APPLY_ERR | 1027 | kpt fails to live apply the manifests to the cluster. |
| DEPLOY_GET_CLOUD_RUN_CLIENT_ERR | 1028 | The Cloud Run Client could not be created |
| DEPLOY_CLOUD_RUN_GET_SERVICE_ERR | 1029 | The Cloud Run Client could not get details about the service. |
| DEPLOY_CLOUD_RUN_UPDATE_SERVICE_ERR | 1030 | The Cloud Run Client was unable to update the service. |
| DEPLOY_CLOUD_RUN_DELETE_SERVICE_ERR | 1031 | The Cloud Run Client was unable to delete the service. |
| DEPLOY_CLOUD_RUN_GET_WORKER_POOL_ERR | 1032 | The Cloud Run Client could not get details about the workerpool. |
| DEPLOY_CLOUD_RUN_UPDATE_WORKER_POOL_ERR | 1033 | The Cloud Run Client was unable to update the workerpool. |
| DEPLOY_CLOUD_RUN_DELETE_WORKER_POOL_ERR | 1034 | The Cloud Run Client was unable to delete the workerpool. |
| TEST_USER_CONFIG_ERR | 1101 | Error expanding paths |
| TEST_CST_USER_ERR | 1102 | Error running container-structure-test |
| TEST_IMG_PULL_ERR | 1103 | Unable to docker pull image |
| TEST_CUSTOM_CMD_PARSE_ERR | 1104 | Unable to parse test command |
| TEST_CUSTOM_CMD_RUN_NON_ZERO_EXIT_ERR | 1105 | Command returned non-zero exit code |
| TEST_CUSTOM_CMD_RUN_TIMEDOUT_ERR | 1106 | command cancelled or timed out |
| TEST_CUSTOM_CMD_RUN_CANCELLED_ERR | 1107 | command cancelled or timed out |
| TEST_CUSTOM_CMD_RUN_EXECUTION_ERR | 1108 | command context error |
| TEST_CUSTOM_CMD_RUN_EXITED_ERR | 1110 | command exited |
| TEST_CUSTOM_CMD_RUN_ERR | 1111 | error running cmd |
| TEST_CUSTOM_DEPENDENCIES_CMD_ERR | 1112 | Error getting dependencies from command |
| TEST_CUSTOM_DEPENDENCIES_UNMARSHALL_ERR | 1113 | Unmarshalling dependency output error |
| TEST_CUSTOM_CMD_RETRIEVE_ERR | 1114 | Error retrieving the command |
| RENDER_KPTFILE_INIT_ERR | 1501 | Render errors The Kptfile cannot be created via `kpt pkg init`. |
| RENDER_KPTFILE_INVALID_YAML_ERR | 1401 | The Kptfile is not a valid yaml file |
| RENDER_KPTFILE_INVALID_SCHEMA_ERR | 1402 | The Kptfile is not a valid API schema |
| RENDER_SET_NAMESPACE_ERR | 1403 | Error setting namespace. |
| RENDER_NAMESPACE_ALREADY_SET_ERR | 1404 | Namespace is already set. |
| RENDER_REPLACE_IMAGE_ERR | 1405 | Error replacing a built artifact in the manifests |
| RENDER_TRANSFORM_MANIFEST_ERR | 1406 | Error transforming a manifest during skaffold debug |
| RENDER_SET_LABEL_ERR | 1407 | Error setting user specified additional labels. |
| RENDER_MANIFEST_WRITE_ERR | 1408 | Error writing hydrated kubernetes manifests. |
| RENDER_PARSE_MANIFEST_IMAGES_ERR | 1409 | Error getting images from a kubernetes manifest. |
| CONFIG_FILE_PARSING_ERR | 1201 | Catch-all configuration file parsing error |
| CONFIG_FILE_NOT_FOUND_ERR | 1202 | Main configuration file not found |
| CONFIG_DEPENDENCY_NOT_FOUND_ERR | 1203 | Dependency configuration file not found |
| CONFIG_DUPLICATE_NAMES_SAME_FILE_ERR | 1204 | Duplicate config names in the same configuration file |
| CONFIG_DUPLICATE_NAMES_ACROSS_FILES_ERR | 1205 | Duplicate config names in two configuration files |
| CONFIG_BAD_FILTER_ERR | 1206 | No configs matching configs filter |
| CONFIG_ZERO_FOUND_ERR | 1207 | No configs parsed from current file |
| CONFIG_APPLY_PROFILES_ERR | 1208 | Failed to apply profiles to config |
| CONFIG_DEFAULT_VALUES_ERR | 1209 | Failed to set default config values |
| CONFIG_FILE_PATHS_SUBSTITUTION_ERR | 1210 | Failed to substitute absolute file paths in config |
| CONFIG_MULTI_IMPORT_PROFILE_CONFLICT_ERR | 1211 | Same config imported at least twice with different set of profiles |
| CONFIG_PROFILES_NOT_FOUND_ERR | 1212 | Profile selection did not match known profile names |
| CONFIG_UNKNOWN_API_VERSION_ERR | 1213 | Config API version not found |
| CONFIG_UNKNOWN_VALIDATOR | 1214 | The validator is not allowed in skaffold-managed mode. |
| CONFIG_UNKNOWN_TRANSFORMER | 1215 | The transformer is not allowed in skaffold-managed mode. |
| CONFIG_MISSING_MANIFEST_FILE_ERR | 1216 | Manifest file not found |
| CONFIG_REMOTE_REPO_CACHE_NOT_FOUND_ERR | 1217 | Remote config repository cache not found and sync disabled |
| CONFIG_UPGRADE_ERR | 1218 | Skaffold config version mismatch |
| INSPECT_UNKNOWN_ERR | 1301 | Catch-all `skaffold inspect` command error |
| INSPECT_BUILD_ENV_ALREADY_EXISTS_ERR | 1302 | Trying to add new build environment that already exists |
| INSPECT_BUILD_ENV_INCORRECT_TYPE_ERR | 1303 | Trying to modify build environment that doesn't exist |
| INSPECT_PROFILE_NOT_FOUND_ERR | 1304 | Trying to modify a profile that doesn't exist |
| PORT_FORWARD_RUN_GCLOUD_NOT_FOUND | 1601 |  |
| PORT_FORWARD_RUN_PROXY_START_ERROR | 1602 |  |
| LOG_STREAM_RUN_GCLOUD_NOT_FOUND | 1603 | GCloud not found error |



<a name="proto.enums.SuggestionCode"></a>

### SuggestionCode
Enum for Suggestion codes

| Name | Number | Description |
| ---- |:------:| ----------- |
| NIL | 0 | default nil suggestion. This is usually set when no error happens. |
| ADD_DEFAULT_REPO | 100 | Add Default Repo |
| CHECK_DEFAULT_REPO | 101 | Verify Default Repo |
| CHECK_DEFAULT_REPO_GLOBAL_CONFIG | 102 | Verify default repo in the global config |
| GCLOUD_DOCKER_AUTH_CONFIGURE | 103 | run gcloud docker auth configure |
| DOCKER_AUTH_CONFIGURE | 104 | Run docker auth configure |
| CHECK_GCLOUD_PROJECT | 105 | Verify Gcloud Project |
| CHECK_DOCKER_RUNNING | 106 | Check if docker is running |
| FIX_USER_BUILD_ERR | 107 | Fix User Build Error |
| DOCKER_BUILD_RETRY | 108 | Docker build internal error, try again |
| FIX_CACHE_FROM_ARTIFACT_CONFIG | 109 | Fix `cacheFrom` config for given artifact and try again |
| FIX_SKAFFOLD_CONFIG_DOCKERFILE | 110 | Fix `dockerfile` config for a given artifact and try again. |
| FIX_JIB_PLUGIN_CONFIGURATION | 111 | Use a supported Jib plugin type |
| FIX_DOCKER_NETWORK_CONTAINER_NAME | 112 | Docker build network invalid docker container name (or id). |
| CHECK_DOCKER_NETWORK_CONTAINER_RUNNING | 113 | Docker build network container not existing in the current context. |
| FIX_DOCKER_NETWORK_MODE_WHEN_EXTRACTING_CONTAINER_NAME | 114 | Executing extractContainerNameFromNetworkMode with a non valid mode (only container mode allowed) |
| RUN_DOCKER_PRUNE | 115 | Prune Docker image |
| SET_CLEANUP_FLAG | 116 | Set Cleanup flag for skaffold command. |
| BUILD_FIX_UNKNOWN_PLATFORM_FLAG | 117 | Check value provided to the `--platform` flag |
| BUILD_INSTALL_PLATFORM_EMULATORS | 118 | Check if QEMU platform emulators are installed |
| SET_PUSH_AND_CONTAINER_REGISTRY | 119 | Set --push and container registry to run a multi-platform build |
| CHECK_CLUSTER_CONNECTION | 201 | Check cluster connection |
| CHECK_MINIKUBE_STATUS | 202 | Check minikube status |
| INSTALL_HELM | 203 | Install helm tool |
| UPGRADE_HELM | 204 | Upgrade helm tool |
| FIX_SKAFFOLD_CONFIG_HELM_ARTIFACT_OVERRIDES | 205 | Fix helm `releases.artifactOverrides` config to match with `build.artifacts` (no longer used in Skaffold v2) |
| UPGRADE_HELM32 | 206 | Upgrade helm version to v3.2.0 and higher. |
| FIX_SKAFFOLD_CONFIG_HELM_CREATE_NAMESPACE | 207 | Set `releases.createNamespace` to false. |
| INVALID_KPT_MANIFESTS | 208 | check the Kptfile validation. |
| ALIGN_KPT_INVENTORY | 209 | align the inventory info in kpt live apply. |
| INSTALL_KUBECTL | 220 | Install kubectl tool |
| SPECIFY_CLOUD_RUN_LOCATION | 230 | Specify Cloud Run Location |
| CHECK_CONTAINER_LOGS | 301 | Container run error |
| CHECK_READINESS_PROBE | 302 | Pod Health check error |
| CHECK_CONTAINER_IMAGE | 303 | Check Container image |
| ADDRESS_NODE_MEMORY_PRESSURE | 400 | Node pressure error |
| ADDRESS_NODE_DISK_PRESSURE | 401 | Node disk pressure error |
| ADDRESS_NODE_NETWORK_UNAVAILABLE | 402 | Node network unavailable error |
| ADDRESS_NODE_PID_PRESSURE | 403 | Node PID pressure error |
| ADDRESS_NODE_UNSCHEDULABLE | 404 | Node unschedulable error |
| ADDRESS_NODE_UNREACHABLE | 405 | Node unreachable error |
| ADDRESS_NODE_NOT_READY | 406 | Node not ready error |
| ADDRESS_FAILED_SCHEDULING | 407 | Scheduler failure error |
| CHECK_HOST_CONNECTION | 408 | Cluster Connectivity error |
| START_MINIKUBE | 501 | Minikube is stopped: use `minikube start` |
| UNPAUSE_MINIKUBE | 502 | Minikube is paused: use `minikube unpause` |
| RUN_DOCKER_PULL | 551 | Run Docker pull for the image with v1 manifest and try again. |
| SET_RENDER_FLAG_OFFLINE_FALSE | 600 | Rerun with correct offline flag value. |
| KPTFILE_MANUAL_INIT | 601 | Manually run `kpt pkg init` or `kpt live init` |
| KPTFILE_CHECK_YAML | 602 | Check if the Kptfile is correct. |
| REMOVE_NAMESPACE_FROM_MANIFESTS | 603 | Remove namespace from manifests |
| CONFIG_CHECK_FILE_PATH | 700 | Check configuration file path |
| CONFIG_CHECK_DEPENDENCY_DEFINITION | 701 | Check dependency config definition |
| CONFIG_CHANGE_NAMES | 702 | Change config name to avoid duplicates |
| CONFIG_CHECK_FILTER | 703 | Check config filter |
| CONFIG_CHECK_PROFILE_DEFINITION | 704 | Check profile definition in current config |
| CONFIG_CHECK_DEPENDENCY_PROFILES_SELECTION | 705 | Check active profile selection for dependency config |
| CONFIG_CHECK_PROFILE_SELECTION | 706 | Check profile selection flag |
| CONFIG_FIX_API_VERSION | 707 | Fix config API version or upgrade the skaffold binary |
| CONFIG_ALLOWLIST_VALIDATORS | 708 | Only the allow listed validators are acceptable in skaffold-managed mode. |
| CONFIG_ALLOWLIST_transformers | 709 | Only the allow listed transformers are acceptable in skaffold-managed mode. |
| CONFIG_FIX_MISSING_MANIFEST_FILE | 710 | Check mising manifest file section of config and fix as needed. |
| CONFIG_ENABLE_REMOTE_REPO_SYNC | 711 | Enable remote repo sync, or clone manually |
| CONFIG_FIX_SKAFFOLD_CONFIG_VERSION | 712 | Upgrade skaffold config version to latest |
| INSPECT_USE_MODIFY_OR_NEW_PROFILE | 800 | Create new build env in a profile instead, or use the 'modify' command |
| INSPECT_USE_ADD_BUILD_ENV | 801 | Check profile selection, or use the 'add' command instead |
| INSPECT_CHECK_INPUT_PROFILE | 802 | Check profile flag value |
| OPEN_ISSUE | 900 | Open an issue so this situation can be diagnosed |
| CHECK_CUSTOM_COMMAND | 1000 | Test error suggestion codes |
| FIX_CUSTOM_COMMAND_TIMEOUT | 1001 |  |
| CHECK_CUSTOM_COMMAND_DEPENDENCIES_CMD | 1002 |  |
| CHECK_CUSTOM_COMMAND_DEPENDENCIES_PATHS | 1003 |  |
| CHECK_TEST_COMMAND_AND_IMAGE_NAME | 1004 |  |



<a name="proto.enums.TesterType"></a>

### TesterType
Enum indicating test tools used

| Name | Number | Description |
| ---- |:------:| ----------- |
| UNKNOWN_TEST_TYPE | 0 | Could not determine Test Type |
| UNIT | 1 | Unit tests |
| CONTAINER_STRUCTURE_TEST | 2 | Container Structure tests |


 <!-- end enums -->
