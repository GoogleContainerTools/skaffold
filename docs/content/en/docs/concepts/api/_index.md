---
title: "Skaffold API"
linkTitle: "Skaffold API"
weight: 40
---


This page discusses the Skaffold API.

Skaffold exposes a API server over its lifetime. The API server is the primary way
to get notifications regarding the different phases in a pipeline run. The API server
also provides fine grain controls to Skaffold's individual components: build, deploy and sync, 
as opposed to relying on Skaffold’s built-in trigger mechanisms.


## Skaffold API 
Skaffold API is restful and `gRPC` based, so it works with any language that has an HTTP library, such as cURL and urllib.
The API server runs on localhost at predefined ports.
The protos used can be found here. (todo add link)

## gRPC Server

gRPC API server is exposed on port `50051` by default. If the port is busy, Skaffold will find the next available port. 
You can grab the port from Skaffold logs.

```code
$ skaffold dev
WARN[0000] port 50051 for gRPC server already in use: using 50053 instead 
``` 
You can also specify a port on the command line with flag `--rpc-port`.


## gRPC REST Server  
REST API server is exposed on port `50052` by default. If the port is busy, Skaffold will find the next available port. 
You can grab the port from Skaffold logs.

```code
$ skaffold dev
WARN[0000] port 50052 for gRPC HTTP server already in use: using 50055 instead 
``` 
You can also specify a port on the command line with flag `--rpc-http-port`.


## Skaffold API
Skaffold API Server exposes following endpoints.

### GET /v1/events

Skaffold provides a continuous development mode [`skaffold dev`](../modes/#skaffold_dev) which builds, deploys
your application on changes. In a single development loop, one or more container images
may be built and deployed. The time taken for the changes to deploy varies.

Skaffold exposes events for users to get notified when phases within a development loop
complete. 
You can use these events to automate next steps in your development workflow. 

e.g: when making a change to port-forwarded frontend service, reload the 
browser url after the service is deployed and running to test changes.

Here is way to get events for a `skaffold dev` [getting-started example](https://github.com/GoogleContainerTools/skaffold/tree/master/examples/getting-started)
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
