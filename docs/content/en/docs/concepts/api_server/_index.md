---
title: "Skaffold API Server"
linkTitle: "Skaffold API Server"
weight: 40
---


This page discusses the Skaffold API Server.

Skaffold exposes a API server over its lifetime. The API server is the primary way
to get notifications regarding the different phases in a pipeline run. 


## Skaffold API Server
Skaffold creates `gRPC HTTP` and `gRPC` server.


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


## Skaffold APIS Endpoints
Skaffold API Server exposes following endpoints.

### Events API


### State API


### Control API
