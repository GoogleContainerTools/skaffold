---
title: "Skaffold API"
linkTitle: "Skaffold API"
weight: 40
---

When running `skaffold dev` or `skaffold debug`, Skaffold starts a server that exposes an API over the lifetime of the run.
Besides the CLI, this API is the primary way tools like IDEs integrate with Skaffold for **retrieving information about the pipeline** and for **controlling the phases in the pipeline**. To  retrieve information of the Skaffold pipeline, the Skaffold API provides two main functionalities: 
- a streaming event log  
created from the different phases in a pipeline run
- and a snapshot of the overall state of the pipeline at any given time during the run.
The API also provides fine grained control over the individual phases of the Skaffold
pipeline (build, deploy and sync), as opposed to relying on Skaffold’s automation.


## Skaffold API 
The Skaffold API is `gRPC` based, and it is also exposed via the gRPC gateway as a JSON over HTTP service.
The server is hosted locally on the same host where the skaffold process is running, and will serve by default on ports 50051 and 50052. These ports can be configured through the `--rpc-port` and `--rpc-http-port` flags.
The server's gRPC service definitions and message protos can be found [here](/#). 
{{% todo 1703 %}}

### gRPC Server

The gRPC API is exposed on port `50051` by default. If this port is busy, Skaffold will find the next available port. 
You can find this port from Skaffold's logs on startup.

```code
$ skaffold dev
WARN[0000] port 50051 for gRPC server already in use: using 50053 instead 
``` 
You can also specify a port on the command line with the `--rpc-port` flag.


### HTTP (REST) API  
The HTTP API is exposed on port `50052` by default. As with the gRPC API, if this port is busy, Skaffold will find the next available port, and the final port can be found from Skaffold's startup logs.
You can grab the port from Skaffold logs.

```code
$ skaffold dev
WARN[0000] port 50052 for gRPC HTTP server already in use: using 50055 instead 
``` 
You can also specify a port on the command line with the `--rpc-http-port` flag.


## Skaffold API
Skaffold's API exposes the following endpoints:

### GET /v1/events

Skaffold provides a continuous development mode [`skaffold dev`](../modes/#skaffold_dev) which builds, deploys
your application on changes. In a single development loop, one or more container images
may be built and deployed. The time taken for the changes to deploy varies.

Skaffold exposes events for users to get notified when phases within a development loop
complete. 
You can use these events to automate next steps in your development workflow. 

For example, when making a change to port-forwarded frontend service, reload the 
browser url after the service is deployed and running to test changes.

An example output using curl to get events for a `skaffold dev` execution on our [getting-started example](https://github.com/GoogleContainerTools/skaffold/tree/master/examples/getting-started)
```code
 curl localhost:50052/v1/events
{"result":{"timestamp":"2019-10-16T18:26:11.385251549Z","event":{"metaEvent":{"entry":"Starting Skaffold: \u0026{Version:v0.39.0-16-g5bb7c9e0 ConfigVersion:skaffold/v1beta15 GitVersion: GitCommit:5bb7c9e078e4d522a5ffc42a2f1274fd17d75902 GitTreeState:dirty BuildDate:2019-10-03T15:01:29Z GoVersion:go1.13rc1 Compiler:gc Platform:linux/amd64}"}}}}
{"result":{"timestamp":"2019-10-16T18:26:11.436231589Z","event":{"buildEvent":{"artifact":"gcr.io/k8s-skaffold/skaffold-example","status":"In Progress"}},"entry":"Build started for artifact gcr.io/k8s-skaffold/skaffold-example"}}
{"result":{"timestamp":"2019-10-16T18:26:12.010124246Z","event":{"buildEvent":{"artifact":"gcr.io/k8s-skaffold/skaffold-example","status":"Complete"}},"entry":"Build completed for artifact gcr.io/k8s-skaffold/skaffold-example"}}
{"result":{"timestamp":"2019-10-16T18:26:12.391721823Z","event":{"deployEvent":{"status":"In Progress"}},"entry":"Deploy started"}}
{"result":{"timestamp":"2019-10-16T18:26:12.847239740Z","event":{"deployEvent":{"status":"Complete"}},"entry":"Deploy complete"}}
..
```
### Get /v1/state


### Get /v1/execute
