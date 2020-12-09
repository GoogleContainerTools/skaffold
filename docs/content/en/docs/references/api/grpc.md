---
title: "gRPC API"
linkTitle: "gRPC API"
weight: 30
---
<!--
******
WARNING!!!

The file docs/content/en/docs/references/api/grpc.md is generated based on proto/markdown.tmpl,
and generated with ./hack/generate_proto.sh!
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
| errCode | [StatusCode](#proto.StatusCode) |  | error code representing the error |
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
| errCode | [StatusCode](#proto.StatusCode) |  | Deprecated. Use actionableErr.errCode. status code representing success or failure |
| actionableErr | [ActionableErr](#proto.ActionableErr) |  | actionable error message |







<a name="proto.BuildMetadata"></a>
#### BuildMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| numberOfArtifacts | [int32](#int32) |  |  |
| builders | [BuildMetadata.ImageBuilder](#proto.BuildMetadata.ImageBuilder) | repeated |  |
| type | [BuildType](#proto.BuildType) |  |  |
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
| type | [BuilderType](#proto.BuilderType) |  |  |
| count | [int32](#int32) |  |  |







<a name="proto.BuildState"></a>
#### BuildState
`BuildState` maps Skaffold artifacts to their current build states


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| artifacts | [BuildState.ArtifactsEntry](#proto.BuildState.ArtifactsEntry) | repeated | A map of `artifact name -> build-state`. Artifact name is defined in the `skaffold.yaml`. The `build-state` can be: <br> - `"Not started"`: not yet started <br> - `"In progress"`: build started <br> - `"Complete"`: build succeeded <br> - `"Failed"`: build failed |
| autoTrigger | [bool](#bool) |  |  |
| statusCode | [StatusCode](#proto.StatusCode) |  |  |







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
| errCode | [StatusCode](#proto.StatusCode) |  | Deprecated. Use actionableErr.errCode. status code representing success or failure |
| actionableErr | [ActionableErr](#proto.ActionableErr) |  | actionable error message |







<a name="proto.DeployMetadata"></a>
#### DeployMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| deployers | [DeployMetadata.Deployer](#proto.DeployMetadata.Deployer) | repeated |  |
| cluster | [ClusterType](#proto.ClusterType) |  |  |







<a name="proto.DeployMetadata.Deployer"></a>
#### DeployMetadata.Deployer



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [DeployerType](#proto.DeployerType) |  |  |
| count | [int32](#int32) |  |  |







<a name="proto.DeployState"></a>
#### DeployState
`DeployState` describes the status of the current deploy


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  |  |
| autoTrigger | [bool](#bool) |  |  |
| statusCode | [StatusCode](#proto.StatusCode) |  |  |







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
| devLoopEvent | [DevLoopEvent](#proto.DevLoopEvent) |  | describes a start and end of a dev loop. |
| terminationEvent | [TerminationEvent](#proto.TerminationEvent) |  | describes a skaffold termination event |







<a name="proto.FileSyncEvent"></a>
#### FileSyncEvent
FileSyncEvent describes the sync status.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| fileCount | [int32](#int32) |  | number of files synced |
| image | [string](#string) |  | the container image to which files are sycned. |
| status | [string](#string) |  | status of file sync. one of: Not Started, In progress, Succeeded, Failed. |
| err | [string](#string) |  | Deprecated. Use actionableErr.message. error in case of status failed. |
| errCode | [StatusCode](#proto.StatusCode) |  | Deprecated. Use actionableErr.errCode. status code representing success or failure |
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
| statusCode | [StatusCode](#proto.StatusCode) |  |  |
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
| errCode | [StatusCode](#proto.StatusCode) |  | Deprecated. Use actionableErr.errCode. status code representing success or failure |
| actionableErr | [ActionableErr](#proto.ActionableErr) |  | actionable error message |







<a name="proto.StatusCheckState"></a>
#### StatusCheckState
`StatusCheckState` describes the state of status check of current deployed resources.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  |  |
| resources | [StatusCheckState.ResourcesEntry](#proto.StatusCheckState.ResourcesEntry) | repeated | A map of `resource name -> status-check-state`. Where `resource-name` is the kubernetes resource name. The `status-check-state` can be <br> - `"Not started"`: indicates that `status-check` has just started. <br> - `"In progress"`: InProgress is sent after every resource check is complete. <br> - `"Succeeded"`: - `"Failed"`: |
| statusCode | [StatusCode](#proto.StatusCode) |  | StatusCheck statusCode |







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
| suggestionCode | [SuggestionCode](#proto.SuggestionCode) |  | code representing a suggestion |
| action | [string](#string) |  | action represents the suggestion action |







<a name="proto.TerminationEvent"></a>
#### TerminationEvent
`TerminationEvent` marks the end of the skaffold session


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  | status oneof: Completed or Failed |
| err | [ActionableErr](#proto.ActionableErr) |  | actionable error message |







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


<a name="proto.BuildType"></a>

### BuildType
Enum indicating build type i.e. local, cluster vs GCB

| Name | Number | Description |
| ---- |:------:| ----------- |
| UNKNOWN_BUILD_TYPE | 0 | Could not determine Build Type |
| CLUSTER | 1 | Cluster Build |
| GCB | 2 | GCB Build |
| LOCAL | 3 | Local Build |



<a name="proto.BuilderType"></a>

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



<a name="proto.ClusterType"></a>

### ClusterType
Enum indicating cluster type the application is deployed to

| Name | Number | Description |
| ---- |:------:| ----------- |
| UNKNOWN_CLUSTER_TYPE | 0 | Could not determine Cluster Type |
| MINIKUBE | 1 | Minikube Cluster |
| GKE | 2 | GKE cluster |
| OTHER | 3 | All Cluster except Minikube and GKE |



<a name="proto.DeployerType"></a>

### DeployerType
Enum indicating deploy tools used

| Name | Number | Description |
| ---- |:------:| ----------- |
| UNKNOWN_DEPLOYER_TYPE | 0 | Could not determine Deployer Type |
| HELM | 1 | Helm Deployer |
| KUSTOMIZE | 2 | Kustomize Deployer |
| KUBECTL | 3 | Kubectl Deployer |



<a name="proto.StatusCode"></a>

### StatusCode
Enum for Status codes
These error codes are prepended by Phase Name e.g.
BUILD, DEPLOY, STATUSCHECK, DEVINIT
For Success Error codes, use range 200 to 250.
For Unknown error codes, use range 500 to 600.
For Cancelled Error code, use range 800 to 850.

| Name | Number | Description |
| ---- |:------:| ----------- |
| OK | 0 | A default status code for events that do not have an associated phase. Typically seen with the DevEndEvent event on success. |
| STATUSCHECK_SUCCESS | 200 | Status Check Success |
| BUILD_SUCCESS | 201 | Build Success |
| DEPLOY_SUCCESS | 202 | Deploy Success |
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
| BUILD_REGISTRY_GET_DIGEST_ERR | 118 | Build error due to digest for built artifact could not be retrieved from registry. |
| BUILD_UNKNOWN_JIB_PLUGIN_TYPE | 119 | Build error indicating unknown Jib plugin type. Should be one of [maven, gradle] |
| BUILD_JIB_GRADLE_DEP_ERR | 120 | Build error determining dependency for jib gradle project. |
| BUILD_JIB_MAVEN_DEP_ERR | 121 | Build error determining dependency for jib gradle project. |
| INIT_DOCKER_NETWORK_LISTING_CONTAINERS | 122 | Docker build error when listing containers. |
| INIT_DOCKER_NETWORK_INVALID_CONTAINER_NAME | 123 | Docker build error indicating an invalid container name (or id). |
| INIT_DOCKER_NETWORK_CONTAINER_DOES_NOT_EXIST | 124 | Docker build error indicating the container referenced does not exists in the docker context used. |
| STATUSCHECK_IMAGE_PULL_ERR | 300 | Container image pull error |
| STATUSCHECK_CONTAINER_CREATING | 301 | Container creating error |
| STATUSCHECK_RUN_CONTAINER_ERR | 302 | Container run error |
| STATUSCHECK_CONTAINER_TERMINATED | 303 | Container is already terminated |
| STATUSCHECK_DEPLOYMENT_ROLLOUT_PENDING | 304 | Deployment waiting for rollout |
| STATUSCHECK_CONTAINER_RESTARTING | 356 | Container restarting error |
| STATUSCHECK_UNHEALTHY | 357 | Readiness probe failed |
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
| STATUSCHECK_POD_INITIALIZING | 451 | Pod Initializing |
| UNKNOWN_ERROR | 500 | Could not determine error and phase |
| STATUSCHECK_UNKNOWN | 501 | Status Check error unknown |
| STATUSCHECK_UNKNOWN_UNSCHEDULABLE | 502 | Container is unschedulable due to unknown reasons |
| STATUSCHECK_CONTAINER_WAITING_UNKNOWN | 503 | Container is waiting due to unknown reason |
| STATUSCHECK_UNKNOWN_EVENT | 509 | Container event reason unknown |
| DEPLOY_UNKNOWN | 504 | Deploy failed due to unknown reason |
| SYNC_UNKNOWN | 505 | SYNC failed due to known reason |
| BUILD_UNKNOWN | 506 | Build failed due to unknown reason |
| DEVINIT_UNKNOWN | 507 | Dev Init failed due to unknown reason |
| CLEANUP_UNKNOWN | 508 | Cleanup failed due to unknown reason |
| INIT_UNKNOWN | 510 | Initialization of the Skaffold session failed due to unknown reason(s) |
| BUILD_DOCKER_UNKNOWN | 511 | Build failed due to docker unknown error |
| SYNC_INIT_ERROR | 601 | File Sync Initialize failure |
| DEVINIT_REGISTER_BUILD_DEPS | 701 | Failed to configure watcher for build dependencies in dev loop |
| DEVINIT_REGISTER_TEST_DEPS | 702 | Failed to configure watcher for test dependencies in dev loop |
| DEVINIT_REGISTER_DEPLOY_DEPS | 703 | Failed to configure watcher for deploy dependencies in dev loop |
| DEVINIT_REGISTER_CONFIG_DEP | 704 | Failed to configure watcher for Skaffold configuration file. |
| DEVINIT_UNSUPPORTED_V1_MANIFEST | 705 | Failed to configure watcher for build dependencies for a base image with v1 manifest. |
| STATUSCHECK_USER_CANCELLED | 800 | User cancelled the skaffold dev run |
| STATUSCHECK_DEADLINE_EXCEEDED | 801 | Deadline for status check exceeded |
| BUILD_CANCELLED | 802 | Build Cancelled |
| DEPLOY_CANCELLED | 803 | Deploy cancelled due to user cancellation or one or more deployers failed. |
| BUILD_DOCKER_CANCELLED | 804 | Docker build cancelled. |
| BUILD_DOCKER_DEADLINE | 805 | Build error due to docker deadline was reached before the docker action completed |
| INIT_CREATE_TAGGER_ERROR | 901 | Skaffold was unable to create the configured tagger |
| INIT_MINIKUBE_PAUSED_ERROR | 902 | Skaffold was unable to start as Minikube appears to be paused |
| INIT_MINIKUBE_NOT_RUNNING_ERROR | 903 | Skaffold was unable to start as Minikube appears to be stopped |
| INIT_CREATE_BUILDER_ERROR | 904 | Skaffold was unable to create a configured image builder |
| INIT_CREATE_DEPLOYER_ERROR | 905 | Skaffold was unable to create a configured deployer |
| INIT_CREATE_TEST_DEP_ERROR | 906 | Skaffold was unable to create a configured test |
| INIT_CACHE_ERROR | 907 | Skaffold encountered an error validating the artifact cache |
| INIT_CREATE_WATCH_TRIGGER_ERROR | 908 | Skaffold encountered an error when configuring file watching |
| INIT_CREATE_ARTIFACT_DEP_ERROR | 909 | Skaffold encountered an error when evaluating artifact dependencies |
| DEPLOY_CLUSTER_CONNECTION_ERR | 1001 | Unable to connect to cluster |
| DEPLOY_DEBUG_HELPER_RETRIEVE_ERR | 1002 | Could not retrieve debug helpers. |
| DEPLOY_CLEANUP_ERR | 1003 | Deploy clean up error |
| DEPLOY_HELM_APPLY_LABELS | 1004 | Unable to apply helm labels. |
| DEPLOY_HELM_USER_ERR | 1005 | Deploy error due to user deploy config for helm deployer |
| DEPLOY_NO_MATCHING_BUILD | 1006 | Helm error when no build result is found of value specified in helm `artifactOverrides` |
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



<a name="proto.SuggestionCode"></a>

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
| CHECK_CLUSTER_CONNECTION | 201 | Check cluster connection |
| CHECK_MINIKUBE_STATUS | 202 | Check minikube status |
| INSTALL_HELM | 203 | Install helm tool |
| UPGRADE_HELM | 204 | Upgrade helm tool |
| FIX_SKAFFOLD_CONFIG_HELM_ARTIFACT_OVERRIDES | 205 | Fix helm `releases.artifactOverrides` config to match with `build.artiofacts` |
| UPGRADE_HELM32 | 206 | Upgrade helm version to v3.2.0 and higher. |
| FIX_SKAFFOLD_CONFIG_HELM_CREATE_NAMESPACE | 207 | Set `releases.createNamespace` to false. |
| INSTALL_KUBECTL | 220 | Install kubectl tool |
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
| OPEN_ISSUE | 900 | Open an issue so this situation can be diagnosed |


 <!-- end enums -->

 <!-- end HasExtensions -->



