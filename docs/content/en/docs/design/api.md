---
title: "Skaffold API"
linkTitle: "Skaffold API"
weight: 60


featureId: api

---
When running [`skaffold dev`]({{< relref "/docs/workflows/dev" >}}) or [`skaffold debug`]({{< relref "/docs/workflows/debug" >}}), 
Skaffold starts a server that exposes an API over the lifetime of the Skaffold process.
Besides the CLI, this API is the primary way tools like IDEs integrate with Skaffold for **retrieving information about the
pipeline** and for **controlling the phases in the pipeline**.

To retrieve information about the Skaffold pipeline, the Skaffold API provides two main functionalities:
  
  * A [streaming event log]({{< relref "#events-api">}}) created from the different phases in a pipeline run, and
  
  * A snapshot of the [overall state]({{< relref "#state-api" >}}) of the pipeline at any given time during the run.

To control the individual phases of the Skaffold, the Skaffold API provides [fine-grained control]({{< relref "#controlling-build-sync-deploy" >}})
over the individual phases of the pipeline (build, deploy, and sync).


## Connecting to the Skaffold API
The Skaffold API is `gRPC` based, and it is also exposed via the gRPC gateway as a JSON over HTTP service.
The server is hosted locally on the same host where the skaffold process is running, and will serve by default on ports 50051 and 50052.
These ports can be configured through the `--rpc-port` and `--rpc-http-port` flags.

For reference, we generate the server's [gRPC service definitions and message protos]({{< relref "/docs/references/api/grpc" >}}) as well as the [Swagger based HTTP API Spec]({{< relref "/docs/references/api/swagger" >}}).


### HTTP server
The HTTP API is exposed on port `50052` by default. The default HTTP port can be overridden with the `--rpc-http-port` flag. 
If the HTTP API port is taken, Skaffold will find the next available port.
The final port can be found from Skaffold's startup logs.

```code
$ skaffold dev
WARN[0000] port 50052 for gRPC HTTP server already in use: using 50055 instead
```

### gRPC Server

The gRPC API is exposed on port `50051` by default and can be overridden with the `--rpc-port` flag.
As with the HTTP API, if this port is taken, Skaffold will find the next available port.
You can find this port from Skaffold's logs on startup.

```code
$ skaffold dev
WARN[0000] port 50051 for gRPC server already in use: using 50053 instead
```

#### Creating a gRPC Client
To connect to the `gRPC` server at default port `50051`, create a client using the following code snippet.

{{< alert title="Note" >}}
The skaffold gRPC server is not compatible with HTTPS, so connections need to be marked as insecure with `grpc.WithInsecure()`
{{</alert>}}

```golang
import (
  "log"
  pb "github.com/GoogleContainerTools/skaffold/proto"
  "google.golang.org/grpc"
)

func main(){
  conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
  if err != nil {
    log.Fatalf("fail to dial: %v", err)
  }
  defer conn.Close()
  client := pb.NewSkaffoldServiceClient(conn)
}
```


## API Structure

Skaffold's API exposes the three main endpoints:

* Event API - continuous stream of lifecycle events
* State API - retrieve the current state
* Control API - control build/deploy/sync

### Event API

Skaffold provides a continuous development mode, [`skaffold dev`]({{< relref "/docs/workflows/dev" >}}), which rebuilds and redeploys
your application on changes. In a single development loop, one or more container images
may be built and deployed.

Skaffold exposes events for clients to be notified when phases within a development loop
start, succeed, or fail.
Tools that integrate with Skaffold can use these events to kick off parts of a development workflow depending on them.

Example scenarios:

* port-forwarding events are used by Cloud Code to automatically attach debuggers to running containers.     
* using an event indicating a frontend service has been deployed and port-forwarded successfully to
kick off a suite of Selenium tests against the newly deployed service.

**Event API Contract**

| protocol | endpoint | encoding |
| ---- | --- | --- |
| HTTP | `http://localhost:{HTTP_RPC_PORT}/v1/events` | newline separated JSON using chunk transfer encoding over HTTP|
| gRPC | `client.Events(ctx)` method on the [`SkaffoldService`]({{< relref "/docs/references/api#skaffoldservice">}}) | protobuf 3 over HTTP |


**Examples**

{{% tabs %}}
{{% tab "HTTP API" %}}
Using `curl` and `HTTP_RPC_PORT=50052`, an example output of a `skaffold dev` execution on our [getting-started example](https://github.com/GoogleContainerTools/skaffold/tree/master/examples/getting-started)
```bash
 curl localhost:50052/v1/events
{"result":{"timestamp":"2019-10-16T18:26:11.385251549Z","event":{"metaEvent":{"entry":"Starting Skaffold: {Version:v0.39.0-16-g5bb7c9e0 ConfigVersion:skaffold/v1 GitVersion: GitCommit:5bb7c9e078e4d522a5ffc42a2f1274fd17d75902 GitTreeState:dirty BuildDate:2019-10-03T15:01:29Z GoVersion:go1.13rc1 Compiler:gc Platform:linux/amd64}"}}}}
{"result":{"timestamp":"2019-10-16T18:26:11.436231589Z","event":{"buildEvent":{"artifact":"gcr.io/k8s-skaffold/skaffold-example","status":"In Progress"}},"entry":"Build started for artifact gcr.io/k8s-skaffold/skaffold-example"}}
{"result":{"timestamp":"2019-10-16T18:26:12.010124246Z","event":{"buildEvent":{"artifact":"gcr.io/k8s-skaffold/skaffold-example","status":"Complete"}},"entry":"Build completed for artifact gcr.io/k8s-skaffold/skaffold-example"}}
{"result":{"timestamp":"2019-10-16T18:26:12.391721823Z","event":{"deployEvent":{"status":"In Progress"}},"entry":"Deploy started"}}
{"result":{"timestamp":"2019-10-16T18:26:12.847239740Z","event":{"deployEvent":{"status":"Complete"}},"entry":"Deploy complete"}}
..
```
{{% /tab %}}
{{% tab "gRPC API" %}}
To get events from the API using `gRPC`, first create a [`gRPC` client]({{< relref "#creating-a-grpc-client" >}}).
then, call the `client.Events()` method:

```golang
func main() {
  ctx, ctxCancel := context.WithCancel(context.Background())
  defer ctxCancel()
  // `client` is a gRPC client with connection to localhost:50051.
  logStream, err := client.Events(ctx, &empty.Empty{})
  if err != nil {
  	log.Fatalf("could not get events: %v", err)
  }
  for {
  	entry, err := logStream.Recv()
  	if err == io.EOF {
  		break
  	}
  	if err != nil {
  		log.Fatal(err)
  	}
  	log.Println(entry)
  }
}
```
{{% /tab %}}
{{% /tabs %}}

Each [Entry]({{<relref "/docs/references/api/grpc#proto.LogEntry" >}}) in the log contains an [Event]({{< relref "/docs/references/api/grpc#proto.Event" >}}) in the `LogEntry.Event` field and
a string description of the event in `LogEntry.entry` field.


### State API

The State API provides a snapshot of the current state of the following components:

- build state per artifacts 
- deploy state
- file sync state 
- status check state per resource 
- port-forwarded resources

**State API Contract**  

| protocol | endpoint | encoding |
| ---- | --- | --- |
| HTTP | `http://localhost:{HTTP_RPC_PORT}/v1/state` | newline separated JSON using chunk transfer encoding over HTTP|  
| gRPC | `client.GetState(ctx)` method on the [`SkaffoldService`]({{< relref "/docs/references/api/grpc#skaffoldservice">}}) | protobuf 3 over HTTP |


**Examples** 
{{% tabs %}}
{{% tab "HTTP API" %}}
Using `curl` and `HTTP_RPC_PORT=50052`, an example output of a `skaffold dev` execution on our [microservices example](https://github.com/GoogleContainerTools/skaffold/tree/master/examples/microservices)
```bash
 curl localhost:50052/v1/state | jq
 {
   "buildState": {
     "artifacts": {
       "gcr.io/k8s-skaffold/leeroy-app": "Complete",
       "gcr.io/k8s-skaffold/leeroy-web": "Complete"
     }
   },
   "deployState": {
     "status": "Complete"
   },
   "forwardedPorts": {
     "9000": {
       "localPort": 9000,
       "remotePort": 8080,
       "namespace": "default",
       "resourceType": "deployment",
       "resourceName": "leeroy-web"
     },
     "50055": {
       "localPort": 50055,
       "remotePort": 50051,
       "namespace": "default",
       "resourceType": "service",
       "resourceName": "leeroy-app"
     }
   },
   "statusCheckState": {
     "status": "Succeeded"
   },
   "fileSyncState": {
     "status": "Not Started"
   }
 }
```
{{% /tab %}}
{{% tab "gRPC API" %}}
To retrieve the state from the server using `gRPC`, first create [`gRPC` client]({{< relref "#creating-a-grpc-client" >}}).
Then, call the `client.GetState()` method:

```golang
func main() {
  // Create a gRPC client connection to localhost:50051.
  // See code above
  ctx, ctxCancel := context.WithCancel(context.Background())
  defer ctxCancel()
  grpcState, err = client.GetState(ctx, &empty.Empty{})
  ...
}
```
{{% /tab %}}
{{% /tabs %}}

### Control API

By default, [`skaffold dev`]({{< relref "/docs/workflows/dev" >}}) will automatically build artifacts, deploy manifests and sync files on every source code change.
However, this behavior can be paused and individual actions can be gated off by user input through the Control API.

With this API, users can tell Skaffold to wait for user input before performing any of these actions,
even if the requisite files were changed on the filesystem. By doing so, users can "queue up" changes while
they are iterating locally, and then have Skaffold rebuild and redeploy only when asked. This can be very
useful when builds are happening more frequently than desired, when builds or deploys take a long time or
are otherwise very costly, or when users want to integrate other tools with `skaffold dev`.

The automation can be turned off or on using the Control API, or with `auto-build` flag for building, `auto-deploy` flag for deploys, and the `auto-sync` flag for file sync.
If automation is turned off for a phase, Skaffold will wait for a request to the Control API before executing the associated action.

Each time a request is sent to the Control API by the user, the specified actions in the payload are executed immediately.
This means that _even if there are new file changes_, Skaffold will wait for another user request before executing any of the given actions again.

**Control API Contract**

| protocol | endpoint | 
| --- | --- |
| HTTP, method: POST | `http://localhost:{HTTP_RPC_PORT}/v1/execute`, the [Execution Service]({{<relref "/docs/references/api/swagger#/SkaffoldService/Execute">}}) |
| gRPC | `client.Execute(ctx)` method on the [`SkaffoldService`]({{< relref "/docs/references/api/grpc#skaffoldservice">}}) |
| HTTP, method: PUT | `http://localhost:{HTTP_RPC_PORT}/v1/build/auto_execute`, the [Auto Build Service]({{<relref "/docs/references/api/swagger#/SkaffoldService/AutoBuild">}}) |
| gRPC | `client.AutoBuild(ctx)` method on the [`SkaffoldService`]({{< relref "/docs/references/api/grpc#skaffoldservice">}}) |
| HTTP, method: PUT | `http://localhost:{HTTP_RPC_PORT}/v1/sync/auto_execute`, the [Auto Sync Service]({{<relref "/docs/references/api/swagger#/SkaffoldService/AutoSync">}}) |
| gRPC | `client.AutoSync(ctx)` method on the [`SkaffoldService`]({{< relref "/docs/references/api/grpc#skaffoldservice">}}) |
| HTTP, method: PUT | `http://localhost:{HTTP_RPC_PORT}/v1/deploy/auto_execute`, the [Auto Deploy Service]({{<relref "/docs/references/api/swagger#/SkaffoldService/AutoDeploy">}}) |
| gRPC | `client.AutoDeploy(ctx)` method on the [`SkaffoldService`]({{< relref "/docs/references/api/grpc#skaffoldservice">}}) |


**Examples**

{{% tabs %}}
{{% tab "HTTP API" %}}

Using our [Quickstart example]({{< relref "/docs/quickstart" >}}), we can start skaffold with `skaffold dev --auto-build=false`.
When we change `main.go`, Skaffold will notice file changes but will not rebuild the image until it receives a request to the Control API with `{"build": true}`:

```bash
curl -X POST http://localhost:50052/v1/execute -d '{"build": true}'
```       

At this point, Skaffold will wait to deploy the newly built image until we invoke the Control API with `{"deploy": true}`:
 
```bash
curl -X POST http://localhost:50052/v1/execute -d '{"deploy": true}'
```       

These steps can also be combined into a single request:

```bash
curl -X POST http://localhost:50052/v1/execute -d '{"build": true, "deploy": true}'
``` 

We can make Skaffold start noticing file changes automatically again by issuing the requests:

```bash
curl -X PUT http://localhost:50052/v1/build/auto_execute -d '{"enabled": true}'
curl -X PUT http://localhost:50052/v1/deploy/auto_execute -d '{"enabled": true}'
``` 

{{% /tab %}}
{{% tab "gRPC API" %}}
To access the Control API via the `gRPC`, create [`gRPC` client]({{< relref "#creating-a-grpc-client" >}}) as before.
Then, use the `client.Execute()` method with the desired payload to trigger it once:

```golang
func main() {
    ctx, ctxCancel := context.WithCancel(context.Background())
    defer ctxCancel()
    // `client` is the gRPC client with connection to localhost:50051.
    _, err = client.Execute(ctx, &pb.UserIntentRequest{
        Intent: &pb.Intent{
            Build:  true,
            Sync:   true,
            Deploy: true,
        },
    })
    if err != nil {
        log.Fatalf("error when trying to execute phases: %v", err)
    }
}
```
Use the `client.AutoBuild()`,`client.AutoSync()` and `client.AutoDeploy()` method to enable or disable auto build, auto sync and auto deploy:

```golang
func main() {
    ctx, ctxCancel := context.WithCancel(context.Background())
    defer ctxCancel()
    // `client` is the gRPC client with connection to localhost:50051.
    _, err = client.AutoBuild(ctx, &pb.TriggerRequest{
		State: &pb.TriggerState{
			Val: &pb.TriggerState_Enabled{
				Enabled: true,
			},
		},
	})    if err != nil {
        log.Fatalf("error when trying to auto trigger phases: %v", err)
    }
}
```
{{% /tab %}}
{{% /tabs %}}
